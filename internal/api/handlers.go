package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"playstore-api/internal/cache"
	"playstore-api/internal/config"
	"playstore-api/internal/models"
	"playstore-api/internal/scraper"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

const (
	// How long to allow a scrape, once it is no longer tied to the request.
	scrapeTimeout = 15 * time.Second
	// How long to allow a cache write, once it is no longer tied to the request.
	cacheWriteTimeout = 5 * time.Second
	// How many scrapes may run at once. The scrape is detached from the
	// caller, so without this bound one client spraying unique ids commits
	// unbounded upstream fetches, buffers and goroutines to the process.
	maxConcurrentScrapes = 8
	// How long a caller waits for a scrape slot before being asked back.
	upstreamWaitTimeout = 2 * time.Second
	// Failed scrapes are remembered briefly so a persistently failing key
	// does not re-buy a full upstream fetch on every request.
	negativeCacheTTL = time.Minute
	// Prefix for failure markers kept alongside cached successes.
	negativeCachePrefix = "neg:"
)

// Accepted shapes for the upstream selectors. Without these, attacker-chosen
// characters ('&', '#', '?') could reshape the outbound request to
// play.google.com. Anything else falls back to the configured default locale
// (gl) or a 400 (id).
var (
	packageIDPattern   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,200}$`)
	geoLocationPattern = regexp.MustCompile(`^[A-Za-z]{2}$`)
)

// cacheKey composes an unambiguous Redis key for a (packageID, gl) selector
// pair. Length-prefixing each component removes the delimiter ambiguity of a
// plain hyphen join, and the namespace keeps our entries separate from any
// other consumer of the shared Redis database.
func cacheKey(packageID, gl string) string {
	return fmt.Sprintf("playstore:v1:%d:%s:%d:%s", len(packageID), packageID, len(gl), gl)
}

type Handler struct {
	Scraper *scraper.PlaystoreScraper
	Cache   cache.Cache
	Config  *config.Config

	// Deduplicates in-flight scrapes by cache key.
	scrapes singleflight.Group

	// Caps concurrently running scrapes (see maxConcurrentScrapes).
	scrapeSem chan struct{}
}

// scrapeResult carries the status code alongside the payload, since
// singleflight only propagates a single value and an error.
type scrapeResult struct {
	data *models.PlaystoreData
	code int
}

func NewHandler(s *scraper.PlaystoreScraper, c cache.Cache, cfg *config.Config) *Handler {
	return &Handler{Scraper: s, Cache: c, Config: cfg, scrapeSem: make(chan struct{}, maxConcurrentScrapes)}
}

//go:embed static/landing.html
var readme string

func (h *Handler) GetREADME(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(readme))
}

//go:embed static/favicon.ico
var favicon string

func (h *Handler) GetFavicon(c *gin.Context) {
	c.Data(http.StatusOK, "image/x-icon", []byte(favicon))
}

func (h *Handler) GetAllData(c *gin.Context) {
	data, code, err := h.getData(c)
	if err != nil {
		log.Printf("failed to get all data: %s", err.Error())
		c.JSON(code, gin.H{"error": "failed to get data"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetDataByKey(c *gin.Context) {
	key := c.Params.ByName("key")
	data, code, err := h.getData(c)
	if err != nil {
		log.Printf("failed to get data by key: %s", err.Error())
		c.JSON(code, gin.H{"error": "failed to get data by key"})
		return
	}
	label, message := data.GetField(key)
	if label == "" && message == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("no data found for key '%s'", key)})
		return
	}

	// shields.io format
	c.JSON(http.StatusOK, gin.H{"schemaVersion": 1, "label": label, "message": message})
}

func (h *Handler) getData(c *gin.Context) (*models.PlaystoreData, int, error) {
	packageID := c.Query("id")
	if !packageIDPattern.MatchString(packageID) {
		return nil, http.StatusBadRequest, fmt.Errorf("'id' must be a non-empty package identifier ([A-Za-z0-9._-], max 200)")
	}

	gl := c.Query("gl")
	if !geoLocationPattern.MatchString(gl) {
		gl = h.Config.DefaultGeoLocation
	}

	// Length-prefixed and namespaced so the (packageID, gl) -> key mapping is
	// injective: with a plain "id-gl" join, (id="com.foo", gl="bar-US") and
	// (id="com.foo-bar", gl="US") collide on one key and can serve one
	// selector's cached entry to the other.
	cacheID := cacheKey(packageID, gl)
	if _, err := h.Cache.Get(c.Request.Context(), negativeCachePrefix+cacheID); err == nil {
		return nil, http.StatusBadGateway, fmt.Errorf("recent upstream failure for this package; retry shortly")
	}
	cachedData, err := h.Cache.Get(c.Request.Context(), cacheID)
	if err == nil {
		var data models.PlaystoreData
		unmarshalErr := json.Unmarshal([]byte(cachedData), &data)
		if unmarshalErr != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to unmarshal data from cache: %w", unmarshalErr)
		}
		return &data, http.StatusOK, nil
	}

	// Detached for the same reason as the cache write in scrape: work already
	// in flight should finish and populate the cache even if the caller has
	// gone away. Bounded so a slow upstream cannot pin the goroutine, and its
	// multi-megabyte buffers, indefinitely.
	scrapeCtx, cancelScrape := context.WithTimeout(context.WithoutCancel(c.Request.Context()), scrapeTimeout)
	defer cancelScrape()

	// Bound concurrent scrapes: the work below is detached from the caller,
	// so a saturated service sheds new fetch work instead of accumulating it.
	select {
	case h.scrapeSem <- struct{}{}:
		defer func() { <-h.scrapeSem }()
	case <-time.After(upstreamWaitTimeout):
		return nil, http.StatusTooManyRequests, fmt.Errorf("upstream busy, retry shortly")
	}

	// Collapse concurrent misses for the same key into one scrape. A popular
	// package requested by many clients at once, or any traffic arriving
	// against a cold cache, would otherwise fetch and parse the same page
	// once per caller.
	res, err, _ := h.scrapes.Do(cacheID, func() (interface{}, error) {
		data, code, err := h.scrape(scrapeCtx, packageID, gl, cacheID)
		return scrapeResult{data: data, code: code}, err
	})

	result := res.(scrapeResult)
	if err != nil {
		return nil, result.code, err
	}

	return result.data, http.StatusOK, nil
}

// markFailed records a brief negative-cache entry so a persistently failing
// key is not re-scraped on every request until the TTL expires.
func (h *Handler) markFailed(cacheID string) {
	ctx, cancel := context.WithTimeout(context.Background(), cacheWriteTimeout)
	defer cancel()
	_ = h.Cache.Set(ctx, negativeCachePrefix+cacheID, "1", negativeCacheTTL)
}

func (h *Handler) scrape(ctx context.Context, packageID, gl, cacheID string) (*models.PlaystoreData, int, error) {
	html, code, err := h.Scraper.FetchHTML(ctx, packageID, gl)
	if err != nil {
		h.markFailed(cacheID)
		return nil, code, fmt.Errorf("failed to fetch html: %w", err)
	}

	data, err := h.Scraper.Parse(packageID, html)
	if err != nil {
		h.markFailed(cacheID)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse html: %w", err)
	}

	b, err := json.Marshal(*data)
	if err != nil {
		h.markFailed(cacheID)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal data for cache: %w", err)
	}

	// Cache on a context detached from the request. The scrape is already paid
	// for, so a client that disconnected mid-scrape - or an upstream proxy that
	// hit its read timeout - must not cost us the result. Tying this to the
	// request context means the cache cannot warm under exactly the load that
	// causes those disconnects, which keeps every subsequent request a miss.
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheWriteTimeout)
	defer cancel()
	if err := h.Cache.Set(cacheCtx, cacheID, string(b), h.Config.CacheTTL); err != nil {
		// The response is still valid and worth returning without the cache write.
		log.Printf("failed to cache data for %q: %s", cacheID, err)
	}

	return data, http.StatusOK, nil
}

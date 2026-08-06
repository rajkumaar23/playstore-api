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
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// How long to allow a scrape, once it is no longer tied to the request.
	scrapeTimeout = 15 * time.Second
	// How long to allow a cache write, once it is no longer tied to the request.
	cacheWriteTimeout = 5 * time.Second
)

type Handler struct {
	Scraper *scraper.PlaystoreScraper
	Cache   cache.Cache
	Config  *config.Config
}

func NewHandler(s *scraper.PlaystoreScraper, c cache.Cache, cfg *config.Config) *Handler {
	return &Handler{Scraper: s, Cache: c, Config: cfg}
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
	if packageID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("'id' cannot be empty'")
	}

	gl := c.Query("gl")
	if gl == "" {
		gl = h.Config.DefaultGeoLocation
	}

	cacheID := fmt.Sprintf("%s-%s", packageID, gl)
	cachedData, err := h.Cache.Get(c.Request.Context(), cacheID)
	if err == nil {
		var data models.PlaystoreData
		unmarshalErr := json.Unmarshal([]byte(cachedData), &data)
		if unmarshalErr != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to unmarshal data from cache: %w", unmarshalErr)
		}
		return &data, http.StatusOK, nil
	}

	// Detached for the same reason as the cache write below: a scrape already
	// in flight should finish and populate the cache even if the caller has
	// gone away. Bounded so a slow upstream cannot pin the goroutine, and its
	// multi-megabyte buffers, indefinitely.
	scrapeCtx, cancelScrape := context.WithTimeout(context.WithoutCancel(c.Request.Context()), scrapeTimeout)
	defer cancelScrape()

	html, code, err := h.Scraper.FetchHTML(scrapeCtx, packageID, gl)
	if err != nil {
		return nil, code, fmt.Errorf("failed to fetch html: %w", err)
	}
	data, err := h.Scraper.Parse(packageID, html)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse html: %w", err)
	}

	b, err := json.Marshal(*data)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal data for cache: %w", err)
	}
	// Cache on a context detached from the request. The scrape is already paid
	// for, so a client that disconnected mid-scrape - or an upstream proxy that
	// hit its read timeout - must not cost us the result. Tying this to the
	// request context means the cache cannot warm under exactly the load that
	// causes those disconnects, which keeps every subsequent request a miss.
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), cacheWriteTimeout)
	defer cancel()
	if err := h.Cache.Set(cacheCtx, cacheID, string(b), h.Config.CacheTTL); err != nil {
		// The response is still valid and worth returning without the cache write.
		log.Printf("failed to cache data for %q: %s", cacheID, err)
	}

	return data, http.StatusOK, nil
}

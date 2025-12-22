package api

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"playstore-api/internal/cache"
	"playstore-api/internal/config"
	"playstore-api/internal/models"
	"playstore-api/internal/scraper"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetDataByKey(c *gin.Context) {
	key := c.Params.ByName("key")
	data, code, err := h.getData(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
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
		var data *models.PlaystoreData
		unmarshalErr := json.Unmarshal([]byte(cachedData), data)
		if unmarshalErr != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to unmarshal data from cache")
		}
		return data, http.StatusOK, nil
	}

	if err != redis.Nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from cache")
	}

	html, code, err := h.Scraper.FetchHTML(c.Request.Context(), packageID, gl)
	if err != nil {
		return nil, code, fmt.Errorf("failed to fetch html")
	}
	data, err := h.Scraper.Parse(packageID, html)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to parse html")
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal data for cache")
	}
	err = h.Cache.Set(c.Request.Context(), cacheID, string(b), time.Hour)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to set data in cache")
	}

	return data, http.StatusOK, nil
}

package api

import (
	"fmt"
	"log"
	"playstore-api/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(h *Handler) *gin.Engine {
	gin.SetMode(h.Config.GinMode)
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(GinMetricsMiddleware())

	metricsRouter := gin.New()
	metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
	log.Printf("metrics server starting at :%s", h.Config.MetricsPort)
	go metricsRouter.Run(fmt.Sprintf(":%s", h.Config.MetricsPort))

	router.GET("/", h.GetREADME)
	router.GET("/favicon.ico", h.GetFavicon)
	router.GET("/json", h.GetAllData)
	router.GET("/:key", h.GetDataByKey)

	return router
}

func Serve(cfg *config.Config, h *Handler) error {
	router := NewRouter(h)
	log.Printf("server starting at :%s", cfg.ServerPort)
	return router.Run(fmt.Sprintf(":%s", cfg.ServerPort))
}

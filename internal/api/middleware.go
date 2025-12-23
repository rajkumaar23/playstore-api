package api

import (
	"playstore-api/internal/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GinMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		path := c.Request.URL.Path
		method := c.Request.Method
		statusCode := c.Writer.Status()
		statusStr := strconv.Itoa(statusCode)

		metrics.ObserveRequest(method, path, statusStr, duration)
	}
}

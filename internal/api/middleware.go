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

		// Label with the matched route, not the raw path. The catch-all
		// GET /:key route means every probe for /wp-admin, /.env and the like
		// would otherwise mint its own label set - in a counter and a
		// twelve-bucket histogram - and the registry only grows.
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		method := c.Request.Method
		statusCode := c.Writer.Status()
		statusStr := strconv.Itoa(statusCode)

		metrics.ObserveRequest(method, path, statusStr, duration)
	}
}

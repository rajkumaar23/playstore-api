package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Total number of HTTP requests, partitioned by method, path template, status, and error(true/false)
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "playstore_api_http_requests_total",
			Help: "Total number of HTTP requests handled by the playstore API",
		},
		[]string{"method", "path", "status"},
	)

	// Duration of HTTP requests in seconds
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "playstore_api_http_request_duration_seconds",
			Help: "Histogram of HTTP request durations (seconds)",
			Buckets: []float64{
				0.00001, // 10μs
				0.00005, // 50μs
				0.0001,  // 100μs
				0.0005,  // 500μs
				0.001,   // 1ms
				0.005,   // 5ms
				0.01,    // 10ms
				0.05,    // 50ms
				0.1,     // 100ms
				0.5,     // 500ms
				1.0,     // 1s
				5.0,     // 5s
			},
		},
		[]string{"method", "path", "status"},
	)

	// Cache size (current number of items in cache)
	CacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "playstore_api_cache_size",
			Help: "Current number of items in cache",
		},
	)

	// Cache hits and misses
	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "playstore_api_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{},
	)

	CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "playstore_api_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{},
	)
)

func init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(CacheSize)
	prometheus.MustRegister(CacheHits)
	prometheus.MustRegister(CacheMisses)
}

// ObserveRequest records a single HTTP request.
func ObserveRequest(method, path, status string, duration time.Duration) {
	RequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	RequestsTotal.WithLabelValues(method, path, status).Inc()
}

// Cache helpers
func SetCacheSize(n float64) {
	CacheSize.Set(n)
}

func IncCacheHit() {
	CacheHits.WithLabelValues().Inc()
}

func IncCacheMiss() {
	CacheMisses.WithLabelValues().Inc()
}

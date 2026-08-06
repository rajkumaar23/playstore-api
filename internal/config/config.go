package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const defaultCacheTTL = 24 * time.Hour

type Config struct {
	RedisAddress       string
	ServerPort         string
	MetricsPort        string
	GinMode            string
	DefaultGeoLocation string
	CacheTTL           time.Duration
}

func LoadEnv() *Config {
	_ = godotenv.Load()

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "5003"
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "5004"
	}

	defaultGL := os.Getenv("DEFAULT_GL")
	if defaultGL == "" {
		defaultGL = "US"
	}

	cacheTTL := defaultCacheTTL
	if raw := os.Getenv("CACHE_TTL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Printf("invalid CACHE_TTL %q, falling back to %s: %v", raw, defaultCacheTTL, err)
		} else {
			cacheTTL = parsed
		}
	}

	return &Config{
		GinMode:            os.Getenv("GIN_MODE"),
		RedisAddress:       os.Getenv("REDIS_ADDRESS"),
		ServerPort:         serverPort,
		MetricsPort:        metricsPort,
		DefaultGeoLocation: defaultGL,
		CacheTTL:           cacheTTL,
	}
}

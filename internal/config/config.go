package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RedisAddress       string
	ServerPort         string
	MetricsPort        string
	GinMode            string
	DefaultGeoLocation string
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

	return &Config{
		GinMode:            os.Getenv("GIN_MODE"),
		RedisAddress:       os.Getenv("REDIS_ADDRESS"),
		ServerPort:         serverPort,
		MetricsPort:        metricsPort,
		DefaultGeoLocation: defaultGL,
	}
}

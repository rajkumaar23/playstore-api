package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RedisAddress       string
	ServerPort         string
	GinMode            string
	DefaultGeoLocation string
}

func LoadEnv() *Config {
	_ = godotenv.Load()

	var serverPort = os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "5003"
	}

	var defaultGL = os.Getenv("DEFAULT_GL")
	if defaultGL == "" {
		defaultGL = "US"
	}

	return &Config{
		GinMode:            os.Getenv("GIN_MODE"),
		RedisAddress:       os.Getenv("REDIS_ADDRESS"),
		ServerPort:         serverPort,
		DefaultGeoLocation: defaultGL,
	}
}

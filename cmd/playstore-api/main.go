package main

import (
	"context"
	"log"
	"playstore-api/internal/api"
	"playstore-api/internal/cache"
	"playstore-api/internal/config"
	"playstore-api/internal/scraper"
)

func main() {
	cfg := config.LoadEnv()

	// init cache
	redisCache, err := cache.NewRedisCache(context.Background(), cfg.RedisAddress)
	if err != nil {
		log.Fatalf("failed to create redis cache: %v", err)
	}
	defer redisCache.Close()

	s := scraper.NewPlaystoreScraper()
	h := api.NewHandler(s, redisCache, cfg)
	if err := api.Serve(cfg, h); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

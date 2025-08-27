package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var ctx context.Context
var rdb *redis.Client

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("error loading .env file")
	}

	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDRESS"),
	})
	defer rdb.Close()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		panic("redis connection failed")
	}

	gin.SetMode(os.Getenv("GIN_MODE"))
	
	router := gin.Default()
	router.Use(cors.Default())

	router.GET("/", getREADME)
	router.GET("/json", getAllData)
	router.GET("/:key", getDataByKey)
	
	router.Run(fmt.Sprintf(":%s", os.Getenv("SERVER_PORT")))
}

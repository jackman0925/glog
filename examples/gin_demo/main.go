package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackman0925/glog"
	"github.com/jackman0925/glog/middleware/ginmw"
)

func main() {
	logger, err := glog.New("../logger.yaml", "/gin-demo")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	r := gin.New()
	r.Use(
		ginmw.GinLoggerWithConfig(logger, ginmw.LoggerConfig{SkipPaths: []string{"/healthz"}}),
		ginmw.GinRecovery(logger, true),
	)

	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})
	r.GET("/panic", func(c *gin.Context) {
		panic("demo panic")
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("gin server failed: %v", err)
	}
}

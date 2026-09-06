package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func Start() *gin.Engine {
	app := gin.Default()

	// CORS Middleware for localhost:3000, Vercel frontend, and custom allowed origins
	app.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
		frontendURLEnv := os.Getenv("FRONTEND_URL")

		allowOrigin := "*"
		if origin != "" {
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "vercel.app") || allowedOriginsEnv == "*" {
				allowOrigin = origin
			} else if allowedOriginsEnv != "" {
				for _, allowed := range strings.Split(allowedOriginsEnv, ",") {
					if strings.TrimSpace(allowed) == origin {
						allowOrigin = origin
						break
					}
				}
			} else if frontendURLEnv != "" {
				for _, allowed := range strings.Split(frontendURLEnv, ",") {
					if strings.TrimSpace(allowed) == origin {
						allowOrigin = origin
						break
					}
				}
			} else {
				allowOrigin = origin
			}
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	return app
}

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("BACKEND_PORT")
	}
	if port == "" {
		port = os.Getenv("DEPLOY_RUN_PORT")
	}
	if port == "" {
		port = "3000"
	}

	log.Printf("========================================")
	log.Printf("  MINIMAL DEBUG SERVER STARTING")
	log.Printf("  Port: %s", port)
	log.Printf("========================================")

	// Check static directory
	entries, err := os.ReadDir("./static")
	if err != nil {
		log.Printf("[STATIC] ERROR: %v", err)
	} else {
		log.Printf("[STATIC] ./static has %d entries:", len(entries))
		for _, e := range entries {
			log.Printf("[STATIC]   %s (isDir=%v)", e.Name(), e.IsDir())
			if e.IsDir() {
				sub, _ := os.ReadDir("./static/" + e.Name())
				for _, s := range sub {
					log.Printf("[STATIC]     %s", s.Name())
				}
			}
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// ===== CORS =====
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	r.Use(gin.Logger())

	// ===== HEALTH CHECK FIRST (no dependencies) =====
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   "minimal-ok",
			"port":   port,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// ===== DEBUG: list static files =====
	r.GET("/api/debug-static", func(c *gin.Context) {
		entries, err := os.ReadDir("./static")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
			return
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"entry_count": len(entries),
			"entries":     names,
		})
	})

	// ===== DEBUG: panic test =====
	r.GET("/api/debug-panic", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"msg":    "this endpoint works, no panic",
		})
	})

	// ===== Static files + SPA fallback =====
	r.Static("/assets", "./static/assets")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "api not found"})
			return
		}
		c.File("./static/index.html")
	})

	log.Printf("[READY] Server listening on :%s", port)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

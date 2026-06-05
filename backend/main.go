package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"stock-app/handlers"
	"stock-app/services"
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
		port = "10000"
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(corsMiddleware())

	stockService := services.NewStockService()
	stockHandler := handlers.NewStockHandler(stockService)

	// 启动时异步填充全A缓存（含 panic recovery 防止崩溃）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] 全A缓存刷新goroutine崩溃: %v", r)
			}
		}()
		time.Sleep(5 * time.Second)
		log.Println("后台全A缓存刷新启动...")
		if _, err := stockService.ScanAllAShares(); err != nil {
			log.Printf("首次全A缓存刷新失败(非致命): %v", err)
		} else {
			log.Println("首次全A缓存刷新完成")
		}

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("定时全A缓存刷新...")
			if _, err := stockService.ScanAllAShares(); err != nil {
				log.Printf("定时全A缓存刷新失败: %v", err)
			} else {
				log.Println("定时全A缓存刷新完成")
			}
		}
	}()

	api := r.Group("/api")
	{
		stock := api.Group("/stock")
		{
			stock.GET("/market", stockHandler.GetMarketStatus)
			stock.GET("/quotes", stockHandler.GetQuotes)
			stock.POST("/diagnose", stockHandler.DiagnoseStock)
			stock.POST("/scan", stockHandler.ScanStocks)
			stock.POST("/scan-core-satellite", stockHandler.ScanCoreSatellite)
			stock.POST("/scan-dual-engine-fast", stockHandler.ScanDualEngineFast)
			stock.GET("/watchlist", stockHandler.GetWatchList)
			stock.POST("/watchlist/add", stockHandler.AddWatchList)
			stock.POST("/watchlist/remove", stockHandler.RemoveWatchList)
			stock.GET("/watchlist/scan", stockHandler.ScanWatchList)
			stock.GET("/detail/:code", stockHandler.GetStockDetail)
			stock.GET("/scores", stockHandler.GetStockScores)
			stock.GET("/unified-score", stockHandler.GetUnifiedScore)
			stock.GET("/position-health", stockHandler.GetPositionHealthScore)
			stock.GET("/search", stockHandler.SearchStocks)
			stock.POST("/scan-all", stockHandler.ScanAllAShares)
			stock.GET("/rank", stockHandler.GetAllStockRank)
			stock.POST("/sell-advice", stockHandler.GetSellAdvice)
		}
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"data":    "ok",
				"version": "5d95135",
				"build":   "2026-06-05",
			})
		})
	}

	// Serve static frontend files from disk
	r.Static("/assets", "./static/assets")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		c.File("./static/index.html")
	})

	log.Printf("[ENV] PORT=%s BACKEND_PORT=%s", os.Getenv("PORT"), os.Getenv("BACKEND_PORT"))
	log.Printf("Serving static frontend from disk at ./static")
	log.Printf("Server starting on :%s", port)

	srv := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

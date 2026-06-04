package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"stock-app/handlers"
	"stock-app/services"
)

//go:embed static/*
var staticEmbed embed.FS

func main() {
	// Must read PORT (Render standard). Fallback to BACKEND_PORT / DEPLOY_RUN_PORT for local dev.
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

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(corsMiddleware())

	stockService := services.NewStockService()
	stockHandler := handlers.NewStockHandler(stockService)

	// 启动时异步填充全A缓存（不阻塞服务启动），之后每10分钟自动刷新
	go func() {
		// 启动后等待2秒让服务先就绪，再跑首次全量扫描
		time.Sleep(2 * time.Second)
		log.Println("后台全A缓存刷新启动...")
		stockService.ScanAllAShares()
		log.Println("首次全A缓存刷新完成")

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("定时全A缓存刷新...")
			stockService.ScanAllAShares()
			log.Println("定时全A缓存刷新完成")
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
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": "ok"})
		})
	}

	// Serve static frontend files (embedded or from disk)
	staticFS, embedErr := fs.Sub(staticEmbed, "static")
	if embedErr == nil {
		// Use embedded static files (production container build)
		r.StaticFS("/assets", http.FS(staticFS))
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if len(path) >= 4 && path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
				return
			}
			// Try exact file
			f, err := staticFS.Open(path)
			if err == nil {
				f.Close()
				c.FileFromFS(path, http.FS(staticFS))
				return
			}
			// SPA fallback
			c.FileFromFS("index.html", http.FS(staticFS))
		})
		log.Println("Serving static frontend (embedded)")
	} else {
		log.Println("Static files not embedded, checking disk...")
	}

	// 使用自定义 http.Server，设置写超时为 90s
	// ScanDualEngineFast 并发后耗时约 5s，ScanAllAShares 耗时约 60s
	// 写超时必须 > 最慢接口，否则扫描中途连接被强制切断 → 前端收到空响应
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 180 * time.Second, // 全A扫描59页最坏~120s，180s留有余量
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("Server starting on :%s", port)
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

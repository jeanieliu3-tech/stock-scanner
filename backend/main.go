package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Log ALL env vars with full values for debugging
	for _, e := range os.Environ() {
		log.Printf("[ENV_FULL] %s", e)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Printf("[PORT] Using port: %s", port)

	// Serve: instant responses, minimal processing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[REQ] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Health check
		if r.URL.Path == "/api/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","port":"` + port + `"}`))
			log.Printf("[RESP] /api/health in %v", time.Since(start))
			return
		}

		// Serve static files from disk
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.StripPrefix("/", http.FileServer(http.Dir("./static"))).ServeHTTP(w, r)
			log.Printf("[RESP] %s in %v", r.URL.Path, time.Since(start))
			return
		}

		// Default: serve index.html
		http.ServeFile(w, r, "./static/index.html")
		log.Printf("[RESP] index.html in %v", time.Since(start))
	})

	addr := "0.0.0.0:" + port
	log.Printf("===== READY: listening on %s =====", addr)

	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

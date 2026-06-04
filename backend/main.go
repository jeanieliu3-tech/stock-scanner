package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// ---- DUMP ALL ENV VARS for debugging ----
	log.Printf("========================================")
	log.Printf("  ENVIRONMENT VARIABLES:")
	for _, e := range os.Environ() {
		// Only log non-sensitive vars
		key := strings.SplitN(e, "=", 2)[0]
		log.Printf("  ENV: %s", key)
	}
	log.Printf("========================================")

	port := os.Getenv("PORT")
	log.Printf("[PORT] os.Getenv(\"PORT\") = %q", port)
	if port == "" {
		port = "10000"
		log.Printf("[PORT] PORT empty, fallback to 10000")
	}

	log.Printf("[INFO] Will listen on 0.0.0.0:%s", port)

	// Check static directory
	entries, err := os.ReadDir("./static")
	if err != nil {
		log.Printf("[STATIC] ERROR: %v", err)
	} else {
		log.Printf("[STATIC] ./static has %d entries", len(entries))
		for _, e := range entries {
			log.Printf("[STATIC]   %s (dir=%v)", e.Name(), e.IsDir())
		}
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":"native-ok"}`))
	})

	// Debug: show all env vars
	mux.HandleFunc("/api/env", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"PORT":"` + os.Getenv("PORT") + `","shell_port":"` + os.Getenv("PORT") + `"}`))
	})

	// SPA fallback
	fs := http.FileServer(http.Dir("./static"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":404,"msg":"api not found"}`))
			return
		}
		filePath := "./static" + r.URL.Path
		if r.URL.Path == "/" {
			filePath = "./static/index.html"
		}
		if _, err := os.Stat(filePath); err == nil && r.URL.Path != "/" {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

	addr := "0.0.0.0:" + port
	log.Printf("[READY] Listening on %s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

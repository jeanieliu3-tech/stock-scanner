package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Printf("========================================")
	log.Printf("  ULTRA-MINIMAL SERVER (no Gin)")
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
		}
	}

	mux := http.NewServeMux()

	// API routes (registered first, take priority)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":"native-ok"}`))
	})

	mux.HandleFunc("/api/debug-static", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		entries, err := os.ReadDir("./static")
		if err != nil {
			w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}
		w.Write([]byte(`{"status":"ok","cnt":` + itoa(len(entries)) + `}`))
	})

	// SPA: serve static files, fallback to index.html
	fs := http.FileServer(http.Dir("./static"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Add CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// API paths not matched above → 404 JSON
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":404,"msg":"api not found"}`))
			return
		}

		// Check if file exists
		filePath := "./static" + r.URL.Path
		if r.URL.Path == "/" {
			filePath = "./static/index.html"
		}

		if _, err := os.Stat(filePath); err == nil {
			// File exists, serve it
			fs.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for any non-file path
		http.ServeFile(w, r, "./static/index.html")
	})

	addr := "0.0.0.0:" + port
	log.Printf("[READY] Server listening on %s", addr)
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

func itoa(n int) string {
	switch n {
	case 0:
		return "0"
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	case 5:
		return "5"
	case 6:
		return "6"
	case 7:
		return "7"
	case 8:
		return "8"
	case 9:
		return "9"
	}
	return "many"
}

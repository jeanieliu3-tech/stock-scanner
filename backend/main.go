package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Dump all env vars
	for _, e := range os.Environ() {
		log.Println(e)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Printf("READY on port=%s", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REQ %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK: path=" + r.URL.Path + " port=" + port))
	})

	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

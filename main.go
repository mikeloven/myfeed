package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "static/index.html")
			return
		}
		
		// Serve static files
		if r.URL.Path[:8] == "/static/" {
			http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))).ServeHTTP(w, r)
			return
		}
		
		http.NotFound(w, r)
	})

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "message": "MyFeed placeholder is running"}`)
	})

	http.HandleFunc("/api/feeds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"feeds": [], "message": "Coming soon - feed management"}`)
	})

	fmt.Printf("MyFeed placeholder server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
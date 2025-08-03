package main

import (
	"fmt"
	"log"
	"myfeed/database"
	"myfeed/handlers"
	"myfeed/services"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize services
	feedService := services.NewFeedService(db)
	articleService := services.NewArticleService(db)

	// Initialize handlers
	feedHandlers := handlers.NewFeedHandlers(feedService, articleService)
	articleHandlers := handlers.NewArticleHandlers(articleService)

	// Setup routes
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	
	// Health check
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "message": "MyFeed is running", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	}).Methods("GET")

	// Stats
	api.HandleFunc("/stats", feedHandlers.GetStats).Methods("GET")

	// Feed routes
	api.HandleFunc("/feeds", feedHandlers.GetFeeds).Methods("GET")
	api.HandleFunc("/feeds", feedHandlers.AddFeed).Methods("POST")
	api.HandleFunc("/feeds/{id:[0-9]+}", feedHandlers.GetFeed).Methods("GET")
	api.HandleFunc("/feeds/{id:[0-9]+}", feedHandlers.DeleteFeed).Methods("DELETE")
	api.HandleFunc("/feeds/{id:[0-9]+}/refresh", feedHandlers.RefreshFeed).Methods("POST")

	// Article routes
	api.HandleFunc("/articles", articleHandlers.GetArticles).Methods("GET")
	api.HandleFunc("/articles/{id:[0-9]+}", articleHandlers.GetArticle).Methods("GET")
	api.HandleFunc("/articles/{id:[0-9]+}/read", articleHandlers.MarkAsRead).Methods("PUT")
	api.HandleFunc("/articles/{id:[0-9]+}/save", articleHandlers.MarkAsSaved).Methods("PUT")
	api.HandleFunc("/articles/mark-all-read", articleHandlers.MarkAllAsRead).Methods("POST")
	api.HandleFunc("/articles/search", articleHandlers.SearchArticles).Methods("GET")

	// Static files and frontend
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	
	// Serve frontend for all other routes
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve API 404 for API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Serve index.html for all other routes (SPA routing)
		http.ServeFile(w, r, "static/index.html")
	})

	// Setup background jobs
	setupCronJobs(feedService, articleService)

	fmt.Printf("MyFeed server starting on port %s\n", port)
	fmt.Println("Database initialized and ready")
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func setupCronJobs(feedService *services.FeedService, articleService *services.ArticleService) {
	c := cron.New()

	// Refresh all feeds every 15 minutes
	c.AddFunc("*/15 * * * *", func() {
		log.Println("Starting scheduled feed refresh...")
		feeds, err := feedService.GetAllFeeds()
		if err != nil {
			log.Printf("Failed to get feeds for refresh: %v", err)
			return
		}

		for _, feed := range feeds {
			go feedService.RefreshFeed(feed.ID)
		}
		log.Printf("Started refresh for %d feeds", len(feeds))
	})

	// Cleanup old articles daily at 2 AM
	c.AddFunc("0 2 * * *", func() {
		log.Println("Starting article cleanup...")
		err := articleService.CleanupOldArticles(30)
		if err != nil {
			log.Printf("Failed to cleanup articles: %v", err)
		} else {
			log.Println("Article cleanup completed")
		}
	})

	c.Start()
	log.Println("Background jobs scheduled")
}
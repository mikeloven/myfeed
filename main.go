package main

import (
	"fmt"
	"log"
	"myfeed/database"
	"myfeed/handlers"
	"myfeed/middleware"
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
	authService := services.NewAuthService(db)
	folderService := services.NewFolderService(db)
	opmlService := services.NewOPMLService(db, feedService, folderService)

	// Ensure default admin user exists
	if err := authService.EnsureDefaultAdmin(); err != nil {
		log.Printf("Warning: Failed to ensure default admin: %v", err)
	}

	// Initialize middleware and handlers
	authMiddleware := middleware.NewAuthMiddleware(authService)
	feedHandlers := handlers.NewFeedHandlers(feedService, articleService)
	articleHandlers := handlers.NewArticleHandlers(articleService)
	folderHandlers := handlers.NewFolderHandlers(folderService, feedService)
	opmlHandlers := handlers.NewOPMLHandlers(opmlService)

	// Setup routes
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	
	// Public routes (no authentication required)
	public := api.PathPrefix("").Subrouter()
	
	// Health check
	public.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "message": "MyFeed is running", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	}).Methods("GET")

	// Authentication routes
	auth := public.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", authMiddleware.Login).Methods("POST")
	auth.HandleFunc("/logout", authMiddleware.Logout).Methods("POST")
	auth.HandleFunc("/user", authMiddleware.GetCurrentUser).Methods("GET")

	// Protected routes (authentication required)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware.RequireAuth)
	
	// Protected auth routes
	protectedAuth := protected.PathPrefix("/auth").Subrouter()
	protectedAuth.HandleFunc("/change-password", authMiddleware.ChangePassword).Methods("POST")

	// Stats
	protected.HandleFunc("/stats", feedHandlers.GetStats).Methods("GET")

	// Feed routes
	protected.HandleFunc("/feeds", feedHandlers.GetFeeds).Methods("GET")
	protected.HandleFunc("/feeds", feedHandlers.AddFeed).Methods("POST")
	protected.HandleFunc("/feeds/{id:[0-9]+}", feedHandlers.GetFeed).Methods("GET")
	protected.HandleFunc("/feeds/{id:[0-9]+}", feedHandlers.DeleteFeed).Methods("DELETE")
	protected.HandleFunc("/feeds/{id:[0-9]+}/refresh", feedHandlers.RefreshFeed).Methods("POST")

	// Article routes
	protected.HandleFunc("/articles", articleHandlers.GetArticles).Methods("GET")
	protected.HandleFunc("/articles/{id:[0-9]+}", articleHandlers.GetArticle).Methods("GET")
	protected.HandleFunc("/articles/{id:[0-9]+}/read", articleHandlers.MarkAsRead).Methods("PUT")
	protected.HandleFunc("/articles/{id:[0-9]+}/save", articleHandlers.MarkAsSaved).Methods("PUT")
	protected.HandleFunc("/articles/mark-all-read", articleHandlers.MarkAllAsRead).Methods("POST")
	protected.HandleFunc("/articles/search", articleHandlers.SearchArticles).Methods("GET")

	// Folder/Category routes
	protected.HandleFunc("/folders", folderHandlers.GetFolders).Methods("GET")
	protected.HandleFunc("/folders", folderHandlers.CreateFolder).Methods("POST")
	protected.HandleFunc("/folders/{id:[0-9]+}", folderHandlers.UpdateFolder).Methods("PUT")
	protected.HandleFunc("/folders/{id:[0-9]+}", folderHandlers.DeleteFolder).Methods("DELETE")
	protected.HandleFunc("/folders/move-feeds", folderHandlers.MoveFeedsToFolder).Methods("POST")

	// OPML Import/Export routes
	protected.HandleFunc("/opml/import", opmlHandlers.ImportOPML).Methods("POST")
	protected.HandleFunc("/opml/export", opmlHandlers.ExportOPML).Methods("GET")

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
	setupCronJobs(feedService, articleService, authService)

	fmt.Printf("MyFeed server starting on port %s\n", port)
	fmt.Println("Database initialized and ready")
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func setupCronJobs(feedService *services.FeedService, articleService *services.ArticleService, authService *services.AuthService) {
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

	// Cleanup expired sessions every hour
	c.AddFunc("0 * * * *", func() {
		err := authService.CleanupExpiredSessions()
		if err != nil {
			log.Printf("Failed to cleanup expired sessions: %v", err)
		}
	})

	c.Start()
	log.Println("Background jobs scheduled")
}
package handlers

import (
	"encoding/json"
	"myfeed/services"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type FeedHandlers struct {
	feedService    *services.FeedService
	articleService *services.ArticleService
}

func NewFeedHandlers(feedService *services.FeedService, articleService *services.ArticleService) *FeedHandlers {
	return &FeedHandlers{
		feedService:    feedService,
		articleService: articleService,
	}
}

type AddFeedRequest struct {
	URL      string `json:"url"`
	FolderID *int   `json:"folder_id,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (fh *FeedHandlers) GetFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := fh.feedService.GetAllFeeds()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    feeds,
	})
}

func (fh *FeedHandlers) AddFeed(w http.ResponseWriter, r *http.Request) {
	var req AddFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	feed, err := fh.feedService.AddFeed(req.URL, req.FolderID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    feed,
	})
}

func (fh *FeedHandlers) GetFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	feed, err := fh.feedService.GetFeedByID(feedID)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    feed,
	})
}

func (fh *FeedHandlers) RefreshFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	go fh.feedService.RefreshFeed(feedID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Feed refresh started"},
	})
}

func (fh *FeedHandlers) DeleteFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	feedID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	err = fh.feedService.DeleteFeed(feedID)
	if err != nil {
		http.Error(w, "Failed to delete feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Feed deleted successfully"},
	})
}

func (fh *FeedHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := fh.articleService.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    stats,
	})
}
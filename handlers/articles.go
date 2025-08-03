package handlers

import (
	"encoding/json"
	"myfeed/services"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ArticleHandlers struct {
	articleService *services.ArticleService
}

func NewArticleHandlers(articleService *services.ArticleService) *ArticleHandlers {
	return &ArticleHandlers{
		articleService: articleService,
	}
}

type MarkReadRequest struct {
	Read bool `json:"read"`
}

type MarkSavedRequest struct {
	Saved bool `json:"saved"`
}

func (ah *ArticleHandlers) GetArticles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	var feedID *int
	if feedIDStr := query.Get("feed_id"); feedIDStr != "" {
		if id, err := strconv.Atoi(feedIDStr); err == nil {
			feedID = &id
		}
	}
	
	var read *bool
	if readStr := query.Get("read"); readStr != "" {
		if readBool, err := strconv.ParseBool(readStr); err == nil {
			read = &readBool
		}
	}
	
	var saved *bool
	if savedStr := query.Get("saved"); savedStr != "" {
		if savedBool, err := strconv.ParseBool(savedStr); err == nil {
			saved = &savedBool
		}
	}
	
	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	
	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	articles, err := ah.articleService.GetArticles(feedID, read, saved, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    articles,
	})
}

func (ah *ArticleHandlers) GetArticle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	article, err := ah.articleService.GetArticleByID(articleID)
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    article,
	})
}

func (ah *ArticleHandlers) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	var req MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = ah.articleService.MarkAsRead(articleID, req.Read)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Article read status updated"},
	})
}

func (ah *ArticleHandlers) MarkAsSaved(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	var req MarkSavedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = ah.articleService.MarkAsSaved(articleID, req.Saved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Article saved status updated"},
	})
}

func (ah *ArticleHandlers) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	var feedID *int
	if feedIDStr := query.Get("feed_id"); feedIDStr != "" {
		if id, err := strconv.Atoi(feedIDStr); err == nil {
			feedID = &id
		}
	}

	err := ah.articleService.MarkAllAsRead(feedID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"message": "All articles marked as read"},
	})
}

func (ah *ArticleHandlers) SearchArticles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	searchQuery := query.Get("q")
	if searchQuery == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}
	
	limit := 50
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	
	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	articles, err := ah.articleService.SearchArticles(searchQuery, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    articles,
	})
}
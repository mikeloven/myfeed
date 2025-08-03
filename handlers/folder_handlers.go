package handlers

import (
	"encoding/json"
	"myfeed/services"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type FolderHandlers struct {
	folderService *services.FolderService
	feedService   *services.FeedService
}

func NewFolderHandlers(folderService *services.FolderService, feedService *services.FeedService) *FolderHandlers {
	return &FolderHandlers{
		folderService: folderService,
		feedService:   feedService,
	}
}

func (fh *FolderHandlers) GetFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := fh.folderService.GetAllFolders()
	if err != nil {
		http.Error(w, "Failed to get folders", http.StatusInternalServerError)
		return
	}

	// Build a hierarchical structure
	type FolderWithFeeds struct {
		ID        int                `json:"id"`
		Name      string             `json:"name"`
		ParentID  *int               `json:"parent_id"`
		Position  int                `json:"position"`
		CreatedAt string             `json:"created_at"`
		Feeds     []interface{}      `json:"feeds"`
		Children  []*FolderWithFeeds `json:"children"`
	}

	folderMap := make(map[int]*FolderWithFeeds)
	var rootFolders []*FolderWithFeeds

	// Create folder objects
	for _, folder := range folders {
		folderObj := &FolderWithFeeds{
			ID:        folder.ID,
			Name:      folder.Name,
			ParentID:  folder.ParentID,
			Position:  folder.Position,
			CreatedAt: folder.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Feeds:     []interface{}{},
			Children:  []*FolderWithFeeds{},
		}
		folderMap[folder.ID] = folderObj

		if folder.ParentID == nil {
			rootFolders = append(rootFolders, folderObj)
		}
	}

	// Build hierarchy
	for _, folder := range folders {
		if folder.ParentID != nil {
			if parent, exists := folderMap[*folder.ParentID]; exists {
				parent.Children = append(parent.Children, folderMap[folder.ID])
			}
		}
	}

	// Get feeds for each folder
	for folderID, folderObj := range folderMap {
		feeds, err := fh.folderService.GetFeedsInFolder(&folderID)
		if err == nil {
			for _, feed := range feeds {
				folderObj.Feeds = append(folderObj.Feeds, map[string]interface{}{
					"id":          feed.ID,
					"title":       feed.Title,
					"url":         feed.URL,
					"health":      feed.Health,
					"error_count": feed.ErrorCount,
				})
			}
		}
	}

	// Also get feeds without folders
	uncategorizedFeeds, err := fh.folderService.GetFeedsInFolder(nil)
	var uncategorizedFeedData []interface{}
	if err == nil {
		for _, feed := range uncategorizedFeeds {
			uncategorizedFeedData = append(uncategorizedFeedData, map[string]interface{}{
				"id":          feed.ID,
				"title":       feed.Title,
				"url":         feed.URL,
				"health":      feed.Health,
				"error_count": feed.ErrorCount,
			})
		}
	}

	response := map[string]interface{}{
		"success":             true,
		"data":                rootFolders,
		"uncategorized_feeds": uncategorizedFeedData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (fh *FolderHandlers) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		ParentID *int   `json:"parent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	folder, err := fh.folderService.CreateFolder(req.Name, req.ParentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    folder,
	})
}

func (fh *FolderHandlers) UpdateFolder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	folder, err := fh.folderService.UpdateFolder(id, req.Name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    folder,
	})
}

func (fh *FolderHandlers) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	err = fh.folderService.DeleteFolder(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Folder deleted successfully",
	})
}

func (fh *FolderHandlers) MoveFeedsToFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FeedIDs  []int `json:"feed_ids"`
		FolderID *int  `json:"folder_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := fh.folderService.MoveFeedsToFolder(req.FeedIDs, req.FolderID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Feeds moved successfully",
	})
}
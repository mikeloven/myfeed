package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"myfeed/services"
	"net/http"
	"strconv"
	"time"
)

type OPMLHandlers struct {
	opmlService *services.OPMLService
}

func NewOPMLHandlers(opmlService *services.OPMLService) *OPMLHandlers {
	return &OPMLHandlers{
		opmlService: opmlService,
	}
}

// ImportOPML handles OPML file import
func (oh *OPMLHandlers) ImportOPML(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Get the file from the form
	file, _, err := r.FormFile("opml_file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No file uploaded or invalid file",
		})
		return
	}
	defer file.Close()

	// Read file contents
	opmlData, err := io.ReadAll(file)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to read file",
		})
		return
	}

	// Import the OPML
	result, err := oh.opmlService.ImportOPML(opmlData)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to import OPML: %v", err),
		})
		return
	}

	// Return success response with import statistics
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Import completed: %d feeds imported, %d skipped", 
			result.ImportedFeeds, result.SkippedFeeds),
		"data": result,
	})
}

// ExportOPML handles OPML file export
func (oh *OPMLHandlers) ExportOPML(w http.ResponseWriter, r *http.Request) {
	// Generate OPML data
	opmlData, err := oh.opmlService.ExportOPML()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to export OPML: %v", err),
		})
		return
	}

	// Set headers for file download
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("myfeed_export_%s.opml", timestamp)
	
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(opmlData)))

	// Write OPML data
	w.Write(opmlData)
}
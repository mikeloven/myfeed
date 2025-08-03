package services

import (
	"encoding/xml"
	"fmt"
	"log"
	"myfeed/database"
	"myfeed/models"
	"time"

	"github.com/gilliek/go-opml/opml"
)

type OPMLService struct {
	db            *database.DB
	feedService   *FeedService
	folderService *FolderService
}

func NewOPMLService(db *database.DB, feedService *FeedService, folderService *FolderService) *OPMLService {
	return &OPMLService{
		db:            db,
		feedService:   feedService,
		folderService: folderService,
	}
}

// ImportResult holds the results of an OPML import operation
type ImportResult struct {
	TotalFeeds    int      `json:"total_feeds"`
	ImportedFeeds int      `json:"imported_feeds"`
	SkippedFeeds  int      `json:"skipped_feeds"`
	Errors        []string `json:"errors,omitempty"`
}

// ImportOPML imports feeds from OPML data
func (os *OPMLService) ImportOPML(opmlData []byte) (*ImportResult, error) {
	var doc opml.OPML
	if err := xml.Unmarshal(opmlData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %v", err)
	}

	result := &ImportResult{
		Errors: make([]string, 0),
	}

	// Process the outline structure
	for _, outline := range doc.Body.Outlines {
		os.processOutline(&outline, 0, result)
	}

	log.Printf("OPML import completed: %d total, %d imported, %d skipped", 
		result.TotalFeeds, result.ImportedFeeds, result.SkippedFeeds)

	return result, nil
}

// processOutline recursively processes OPML outline elements
func (os *OPMLService) processOutline(outline *opml.Outline, parentFolderID int, result *ImportResult) {
	// If this outline has an XML URL, it's a feed
	if outline.XMLURL != "" {
		result.TotalFeeds++
		
		// Check if feed already exists
		existingFeed, err := os.feedService.GetFeedByURL(outline.XMLURL)
		if err == nil && existingFeed != nil {
			result.SkippedFeeds++
			log.Printf("Skipping existing feed: %s", outline.XMLURL)
			return
		}

		// Add the feed using the feed service
		var folderID *int
		if parentFolderID > 0 {
			folderID = &parentFolderID
		}

		_, err = os.feedService.AddFeed(outline.XMLURL, folderID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to add feed %s: %v", outline.XMLURL, err))
			log.Printf("Failed to add feed %s: %v", outline.XMLURL, err)
		} else {
			result.ImportedFeeds++
			log.Printf("Imported feed: %s", outline.XMLURL)
		}
	} else if outline.Text != "" || outline.Title != "" {
		// This is a folder/category
		folderName := outline.Title
		if folderName == "" {
			folderName = outline.Text
		}

		// Create the folder
		var parentID *int
		if parentFolderID > 0 {
			parentID = &parentFolderID
		}

		folder, err := os.folderService.CreateFolder(folderName, parentID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create folder %s: %v", folderName, err))
			log.Printf("Failed to create folder %s: %v", folderName, err)
			// Continue with parent folder ID for child outlines
			folderID := parentFolderID
			// Process child outlines with parent folder ID
			for _, childOutline := range outline.Outlines {
				os.processOutline(&childOutline, folderID, result)
			}
		} else {
			log.Printf("Created folder: %s", folderName)
			// Process child outlines with new folder ID
			for _, childOutline := range outline.Outlines {
				os.processOutline(&childOutline, folder.ID, result)
			}
		}
	}
}

// ExportOPML exports all feeds to OPML format
func (os *OPMLService) ExportOPML() ([]byte, error) {
	// Get all folders and feeds
	folders, err := os.folderService.GetAllFolders()
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %v", err)
	}

	feeds, err := os.feedService.GetAllFeeds()
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds: %v", err)
	}

	// Create OPML document
	doc := opml.OPML{
		Version: "2.0",
		Head: opml.Head{
			Title:        "MyFeed Export",
			DateCreated:  time.Now().Format(time.RFC1123Z),
			DateModified: time.Now().Format(time.RFC1123Z),
			OwnerName:    "MyFeed",
		},
		Body: opml.Body{
			Outlines: make([]opml.Outline, 0),
		},
	}

	// Create a map for quick folder lookup
	folderMap := make(map[int]*models.Folder)
	for i := range folders {
		folderMap[folders[i].ID] = &folders[i]
	}

	// Group feeds by folder
	feedsByFolder := make(map[int][]*models.Feed)
	feedsWithoutFolder := make([]*models.Feed, 0)

	for i := range feeds {
		feed := &feeds[i]
		if feed.FolderID != nil && *feed.FolderID > 0 {
			feedsByFolder[*feed.FolderID] = append(feedsByFolder[*feed.FolderID], feed)
		} else {
			feedsWithoutFolder = append(feedsWithoutFolder, feed)
		}
	}

	// Add root-level folders and their feeds
	rootFolders := make([]*models.Folder, 0)
	for i := range folders {
		folder := &folders[i]
		if folder.ParentID == nil || *folder.ParentID == 0 {
			rootFolders = append(rootFolders, folder)
		}
	}

	// Process root folders
	for _, folder := range rootFolders {
		outline := os.createFolderOutline(folder, folderMap, feedsByFolder)
		doc.Body.Outlines = append(doc.Body.Outlines, outline)
	}

	// Add feeds without folders
	for _, feed := range feedsWithoutFolder {
		outline := opml.Outline{
			Type:        "rss",
			Title:       feed.Title,
			Text:        feed.Title,
			XMLURL:      feed.URL,
			Description: feed.Description,
		}
		doc.Body.Outlines = append(doc.Body.Outlines, outline)
	}

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OPML: %v", err)
	}

	// Add XML header
	result := []byte(xml.Header + string(xmlData))
	return result, nil
}

// createFolderOutline recursively creates OPML outline for a folder and its contents
func (os *OPMLService) createFolderOutline(folder *models.Folder, folderMap map[int]*models.Folder, feedsByFolder map[int][]*models.Feed) opml.Outline {
	outline := opml.Outline{
		Title:    folder.Name,
		Text:     folder.Name,
		Outlines: make([]opml.Outline, 0),
	}

	// Add feeds in this folder
	if feeds, exists := feedsByFolder[folder.ID]; exists {
		for _, feed := range feeds {
			feedOutline := opml.Outline{
				Type:        "rss",
				Title:       feed.Title,
				Text:        feed.Title,
				XMLURL:      feed.URL,
				Description: feed.Description,
			}
			outline.Outlines = append(outline.Outlines, feedOutline)
		}
	}

	// Add child folders
	for _, childFolder := range folderMap {
		if childFolder.ParentID != nil && *childFolder.ParentID == folder.ID {
			childOutline := os.createFolderOutline(childFolder, folderMap, feedsByFolder)
			outline.Outlines = append(outline.Outlines, childOutline)
		}
	}

	return outline
}
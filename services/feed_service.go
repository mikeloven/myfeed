package services

import (
	"database/sql"
	"fmt"
	"log"
	"myfeed/database"
	"myfeed/models"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedService struct {
	db     *database.DB
	parser *gofeed.Parser
}

func NewFeedService(db *database.DB) *FeedService {
	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout: 30 * time.Second,
	}
	
	return &FeedService{
		db:     db,
		parser: parser,
	}
}

func (fs *FeedService) AddFeed(url string, folderID *int) (*models.Feed, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("feed URL cannot be empty")
	}

	// Try to parse the feed first to validate it
	parsedFeed, err := fs.parser.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %v", err)
	}

	// Check if feed already exists
	existingFeed, err := fs.GetFeedByURL(url)
	if err == nil && existingFeed != nil {
		return nil, fmt.Errorf("feed already exists")
	}

	// Insert the feed
	query := `
		INSERT INTO feeds (url, title, description, folder_id, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	result, err := fs.db.Exec(query, url, parsedFeed.Title, parsedFeed.Description, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert feed: %v", err)
	}

	feedID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get feed ID: %v", err)
	}

	// Fetch initial articles
	go fs.RefreshFeed(int(feedID))

	return fs.GetFeedByID(int(feedID))
}

func (fs *FeedService) GetFeedByID(id int) (*models.Feed, error) {
	query := `
		SELECT id, url, title, description, folder_id, created_at, updated_at, 
		       last_fetch, health, error_count
		FROM feeds WHERE id = ?
	`
	
	feed := &models.Feed{}
	err := fs.db.QueryRow(query, id).Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.FolderID,
		&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.Health, &feed.ErrorCount,
	)
	
	if err != nil {
		return nil, err
	}
	
	return feed, nil
}

func (fs *FeedService) GetFeedByURL(url string) (*models.Feed, error) {
	query := `
		SELECT id, url, title, description, folder_id, created_at, updated_at, 
		       last_fetch, health, error_count
		FROM feeds WHERE url = ?
	`
	
	feed := &models.Feed{}
	err := fs.db.QueryRow(query, url).Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.FolderID,
		&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.Health, &feed.ErrorCount,
	)
	
	if err != nil {
		return nil, err
	}
	
	return feed, nil
}

func (fs *FeedService) GetAllFeeds() ([]models.Feed, error) {
	query := `
		SELECT id, url, title, description, folder_id, created_at, updated_at, 
		       last_fetch, health, error_count
		FROM feeds ORDER BY title
	`
	
	rows, err := fs.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		feed := models.Feed{}
		err := rows.Scan(
			&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.FolderID,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.Health, &feed.ErrorCount,
		)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	
	return feeds, nil
}

func (fs *FeedService) RefreshFeed(feedID int) error {
	feed, err := fs.GetFeedByID(feedID)
	if err != nil {
		return fmt.Errorf("failed to get feed: %v", err)
	}

	log.Printf("Refreshing feed: %s", feed.Title)

	parsedFeed, err := fs.parser.ParseURL(feed.URL)
	if err != nil {
		fs.updateFeedError(feedID, err)
		return fmt.Errorf("failed to parse feed: %v", err)
	}

	// Update feed metadata
	updateQuery := `
		UPDATE feeds 
		SET title = ?, description = ?, last_fetch = CURRENT_TIMESTAMP, 
		    health = 'healthy', error_count = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err = fs.db.Exec(updateQuery, parsedFeed.Title, parsedFeed.Description, feedID)
	if err != nil {
		return fmt.Errorf("failed to update feed: %v", err)
	}

	// Add new articles
	for _, item := range parsedFeed.Items {
		err := fs.addArticle(feedID, item)
		if err != nil {
			log.Printf("Failed to add article %s: %v", item.Title, err)
		}
	}

	log.Printf("Successfully refreshed feed: %s (%d articles)", feed.Title, len(parsedFeed.Items))
	return nil
}

func (fs *FeedService) addArticle(feedID int, item *gofeed.Item) error {
	// Check if article already exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM articles WHERE feed_id = ? AND url = ?`
	err := fs.db.QueryRow(checkQuery, feedID, item.Link).Scan(&count)
	if err != nil {
		return err
	}
	
	if count > 0 {
		return nil // Article already exists
	}

	publishedAt := time.Now()
	if item.PublishedParsed != nil {
		publishedAt = *item.PublishedParsed
	}

	content := item.Description
	if item.Content != "" {
		content = item.Content
	}

	author := ""
	if item.Author != nil {
		author = item.Author.Name
	}

	insertQuery := `
		INSERT INTO articles (feed_id, title, content, url, author, published_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err = fs.db.Exec(insertQuery, feedID, item.Title, content, item.Link, author, publishedAt)
	return err
}

func (fs *FeedService) updateFeedError(feedID int, feedError error) {
	updateQuery := `
		UPDATE feeds 
		SET health = CASE 
			WHEN error_count + 1 >= 3 THEN 'error'
			WHEN error_count + 1 >= 1 THEN 'warning'
			ELSE 'healthy'
		END,
		error_count = error_count + 1,
		last_fetch = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := fs.db.Exec(updateQuery, feedID)
	if err != nil {
		log.Printf("Failed to update feed error status: %v", err)
	}
	
	log.Printf("Feed %d error: %v", feedID, feedError)
}

func (fs *FeedService) DeleteFeed(feedID int) error {
	query := `DELETE FROM feeds WHERE id = ?`
	result, err := fs.db.Exec(query, feedID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}
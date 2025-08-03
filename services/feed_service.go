package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"myfeed/database"
	"myfeed/models"
	"net/http"
	"regexp"
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

	// Convert YouTube channel URL to RSS feed URL if needed
	rssURL, err := fs.convertToRSSURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to convert URL: %v", err)
	}

	// Try to parse the feed first to validate it
	parsedFeed, err := fs.parser.ParseURL(rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %v", err)
	}

	// Check if feed already exists (check both original URL and RSS URL)
	existingFeed, err := fs.GetFeedByURL(rssURL)
	if err == nil && existingFeed != nil {
		return nil, fmt.Errorf("feed already exists")
	}
	
	// Also check original URL if different
	if url != rssURL {
		existingFeed, err := fs.GetFeedByURL(url)
		if err == nil && existingFeed != nil {
			return nil, fmt.Errorf("feed already exists")
		}
	}

	// Insert the feed using the RSS URL
	query := `
		INSERT INTO feeds (url, title, description, folder_id, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	result, err := fs.db.Exec(query, rssURL, parsedFeed.Title, parsedFeed.Description, folderID)
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

// convertToRSSURL converts various URL formats to RSS feed URLs
func (fs *FeedService) convertToRSSURL(url string) (string, error) {
	// If it's already an RSS/Atom feed, return as-is
	if strings.Contains(strings.ToLower(url), "rss") || 
	   strings.Contains(strings.ToLower(url), "atom") || 
	   strings.Contains(strings.ToLower(url), "feed") {
		return url, nil
	}

	// Handle YouTube channel URLs
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return fs.convertYouTubeToRSS(url)
	}

	// For other URLs, assume they're already RSS feeds or return as-is
	return url, nil
}

// convertYouTubeToRSS converts YouTube channel URLs to RSS feed URLs
func (fs *FeedService) convertYouTubeToRSS(url string) (string, error) {
	// Pattern for different YouTube URL formats
	patterns := []struct {
		regex   *regexp.Regexp
		handler func([]string) (string, error)
	}{
		// Channel ID format: https://www.youtube.com/channel/UCxxx or /c/channelname
		{
			regexp.MustCompile(`youtube\.com/channel/([a-zA-Z0-9_-]+)`),
			func(matches []string) (string, error) {
				return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", matches[1]), nil
			},
		},
		// Custom channel name: https://www.youtube.com/c/channelname or @username
		{
			regexp.MustCompile(`youtube\.com/c/([a-zA-Z0-9_-]+)`),
			func(matches []string) (string, error) {
				channelID, err := fs.getYouTubeChannelID(fmt.Sprintf("https://www.youtube.com/c/%s", matches[1]))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID), nil
			},
		},
		// Username format: https://www.youtube.com/user/username
		{
			regexp.MustCompile(`youtube\.com/user/([a-zA-Z0-9_-]+)`),
			func(matches []string) (string, error) {
				channelID, err := fs.getYouTubeChannelID(fmt.Sprintf("https://www.youtube.com/user/%s", matches[1]))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID), nil
			},
		},
		// New handle format: https://www.youtube.com/@username
		{
			regexp.MustCompile(`youtube\.com/@([a-zA-Z0-9_-]+)`),
			func(matches []string) (string, error) {
				channelID, err := fs.getYouTubeChannelID(fmt.Sprintf("https://www.youtube.com/@%s", matches[1]))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID), nil
			},
		},
	}

	for _, pattern := range patterns {
		if matches := pattern.regex.FindStringSubmatch(url); matches != nil {
			return pattern.handler(matches)
		}
	}

	return "", fmt.Errorf("unsupported YouTube URL format: %s", url)
}

// getYouTubeChannelID extracts the channel ID from a YouTube channel page
func (fs *FeedService) getYouTubeChannelID(channelURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(channelURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch channel page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("channel page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read channel page: %v", err)
	}

	// Look for channel ID in various places in the HTML
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"channelId":"([a-zA-Z0-9_-]+)"`),
		regexp.MustCompile(`<meta property="og:url" content="https://www\.youtube\.com/channel/([a-zA-Z0-9_-]+)">`),
		regexp.MustCompile(`channel/([a-zA-Z0-9_-]+)`),
	}

	content := string(body)
	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); matches != nil {
			channelID := matches[1]
			// Validate that it looks like a YouTube channel ID (starts with UC and is 24 chars)
			if strings.HasPrefix(channelID, "UC") && len(channelID) == 24 {
				return channelID, nil
			}
		}
	}

	return "", fmt.Errorf("could not find channel ID for %s", channelURL)
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
package models

import (
	"time"
)

type Feed struct {
	ID          int       `json:"id" db:"id"`
	URL         string    `json:"url" db:"url"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	FolderID    *int      `json:"folder_id" db:"folder_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	LastFetch   *time.Time `json:"last_fetch" db:"last_fetch"`
	Health      string    `json:"health" db:"health"` // "healthy", "warning", "error"
	ErrorCount  int       `json:"error_count" db:"error_count"`
}

type Folder struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ParentID  *int      `json:"parent_id" db:"parent_id"`
	Position  int       `json:"position" db:"position"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Article struct {
	ID          int       `json:"id" db:"id"`
	FeedID      int       `json:"feed_id" db:"feed_id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"`
	URL         string    `json:"url" db:"url"`
	Author      string    `json:"author" db:"author"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
	Read        bool      `json:"read" db:"read"`
	Saved       bool      `json:"saved" db:"saved"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Setting struct {
	Key   string `json:"key" db:"key"`
	Value string `json:"value" db:"value"`
}

type FeedStats struct {
	TotalFeeds     int `json:"total_feeds"`
	TotalArticles  int `json:"total_articles"`
	UnreadArticles int `json:"unread_articles"`
	SavedArticles  int `json:"saved_articles"`
}

type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"` // Never return password in JSON
	IsAdmin   bool      `json:"is_admin" db:"is_admin"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	LastLogin *time.Time `json:"last_login" db:"last_login"`
}

type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}
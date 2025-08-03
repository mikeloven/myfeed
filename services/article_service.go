package services

import (
	"fmt"
	"myfeed/database"
	"myfeed/models"
	"strings"
)

type ArticleService struct {
	db *database.DB
}

func NewArticleService(db *database.DB) *ArticleService {
	return &ArticleService{db: db}
}

func (as *ArticleService) GetArticles(feedID *int, read *bool, saved *bool, limit, offset int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.content, a.url, a.author, 
		       a.published_at, a.read, a.saved, a.created_at
		FROM articles a
		WHERE 1=1
	`
	
	var args []interface{}
	
	if feedID != nil {
		query += " AND a.feed_id = ?"
		args = append(args, *feedID)
	}
	
	if read != nil {
		query += " AND a.read = ?"
		args = append(args, *read)
	}
	
	if saved != nil {
		query += " AND a.saved = ?"
		args = append(args, *saved)
	}
	
	query += " ORDER BY a.published_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := as.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		article := models.Article{}
		err := rows.Scan(
			&article.ID, &article.FeedID, &article.Title, &article.Content, &article.URL,
			&article.Author, &article.PublishedAt, &article.Read, &article.Saved, &article.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	
	return articles, nil
}

func (as *ArticleService) GetArticleByID(id int) (*models.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.content, a.url, a.author, 
		       a.published_at, a.read, a.saved, a.created_at
		FROM articles a
		WHERE a.id = ?
	`
	
	article := &models.Article{}
	err := as.db.QueryRow(query, id).Scan(
		&article.ID, &article.FeedID, &article.Title, &article.Content, &article.URL,
		&article.Author, &article.PublishedAt, &article.Read, &article.Saved, &article.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return article, nil
}

func (as *ArticleService) MarkAsRead(articleID int, read bool) error {
	query := `UPDATE articles SET read = ? WHERE id = ?`
	_, err := as.db.Exec(query, read, articleID)
	return err
}

func (as *ArticleService) MarkAsSaved(articleID int, saved bool) error {
	query := `UPDATE articles SET saved = ? WHERE id = ?`
	_, err := as.db.Exec(query, saved, articleID)
	return err
}

func (as *ArticleService) MarkAllAsRead(feedID *int) error {
	query := `UPDATE articles SET read = true WHERE 1=1`
	var args []interface{}
	
	if feedID != nil {
		query += " AND feed_id = ?"
		args = append(args, *feedID)
	}
	
	_, err := as.db.Exec(query, args...)
	return err
}

func (as *ArticleService) SearchArticles(searchQuery string, limit, offset int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.content, a.url, a.author, 
		       a.published_at, a.read, a.saved, a.created_at
		FROM articles a
		WHERE a.title LIKE ? OR a.content LIKE ? OR a.author LIKE ?
		ORDER BY a.published_at DESC 
		LIMIT ? OFFSET ?
	`
	
	searchPattern := "%" + strings.ToLower(searchQuery) + "%"
	rows, err := as.db.Query(query, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		article := models.Article{}
		err := rows.Scan(
			&article.ID, &article.FeedID, &article.Title, &article.Content, &article.URL,
			&article.Author, &article.PublishedAt, &article.Read, &article.Saved, &article.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	
	return articles, nil
}

func (as *ArticleService) GetStats() (*models.FeedStats, error) {
	stats := &models.FeedStats{}
	
	// Get total feeds
	err := as.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&stats.TotalFeeds)
	if err != nil {
		return nil, err
	}
	
	// Get total articles
	err = as.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&stats.TotalArticles)
	if err != nil {
		return nil, err
	}
	
	// Get unread articles
	err = as.db.QueryRow("SELECT COUNT(*) FROM articles WHERE read = false").Scan(&stats.UnreadArticles)
	if err != nil {
		return nil, err
	}
	
	// Get saved articles
	err = as.db.QueryRow("SELECT COUNT(*) FROM articles WHERE saved = true").Scan(&stats.SavedArticles)
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

func (as *ArticleService) CleanupOldArticles(daysOld int) error {
	query := `
		DELETE FROM articles 
		WHERE read = true 
		AND saved = false 
		AND created_at < datetime('now', '-' || ? || ' days')
	`
	
	result, err := as.db.Exec(query, daysOld)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d old articles\n", rowsAffected)
	}
	
	return nil
}
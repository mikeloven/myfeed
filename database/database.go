package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func NewDatabase() (*DB, error) {
	// Check if PostgreSQL connection string is provided
	if pgURL := os.Getenv("DATABASE_URL"); pgURL != "" {
		log.Println("INFO: DATABASE_URL found, attempting PostgreSQL connection...")
		return newPostgreSQLDatabase(pgURL)
	}
	
	// Fall back to SQLite for development
	log.Println("INFO: No DATABASE_URL found, using SQLite for development...")
	return newSQLiteDatabase()
}

func newPostgreSQLDatabase(databaseURL string) (*DB, error) {
	log.Println("Connecting to PostgreSQL database...")
	
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %v", err)
	}

	database := &DB{db}
	if err := database.createPostgreSQLTables(); err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL tables: %v", err)
	}

	log.Println("PostgreSQL database initialized successfully")
	return database, nil
}

func newSQLiteDatabase() (*DB, error) {
	log.Println("Using SQLite database for development...")
	
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "myfeed.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %v", err)
	}

	database := &DB{db}
	if err := database.createSQLiteTables(); err != nil {
		return nil, fmt.Errorf("failed to create SQLite tables: %v", err)
	}

	log.Println("SQLite database initialized successfully")
	return database, nil
}

func (db *DB) createSQLiteTables() error {
	schema := `
	-- Folders table
	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER,
		position INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE CASCADE
	);

	-- Feeds table
	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		folder_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_fetch DATETIME,
		health TEXT DEFAULT 'healthy' CHECK (health IN ('healthy', 'warning', 'error')),
		error_count INTEGER DEFAULT 0,
		FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE SET NULL
	);

	-- Articles table
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT,
		url TEXT,
		author TEXT,
		published_at DATETIME NOT NULL,
		read BOOLEAN DEFAULT FALSE,
		saved BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
	);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	-- Indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at);
	CREATE INDEX IF NOT EXISTS idx_articles_read ON articles(read);
	CREATE INDEX IF NOT EXISTS idx_articles_saved ON articles(saved);
	CREATE INDEX IF NOT EXISTS idx_feeds_folder_id ON feeds(folder_id);
	CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Indexes for users and sessions
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	-- Insert default settings
	INSERT OR IGNORE INTO settings (key, value) VALUES 
		('app_title', 'MyFeed'),
		('articles_per_page', '50'),
		('cleanup_after_days', '30'),
		('refresh_interval', '15m');
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) createPostgreSQLTables() error {
	schema := `
	-- Folders table
	CREATE TABLE IF NOT EXISTS folders (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
		position INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Feeds table
	CREATE TABLE IF NOT EXISTS feeds (
		id SERIAL PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		folder_id INTEGER REFERENCES folders(id) ON DELETE SET NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_fetch TIMESTAMP,
		health TEXT DEFAULT 'healthy' CHECK (health IN ('healthy', 'warning', 'error')),
		error_count INTEGER DEFAULT 0
	);

	-- Articles table
	CREATE TABLE IF NOT EXISTS articles (
		id SERIAL PRIMARY KEY,
		feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		content TEXT,
		url TEXT,
		author TEXT,
		published_at TIMESTAMP NOT NULL,
		read BOOLEAN DEFAULT FALSE,
		saved BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login TIMESTAMP
	);

	-- Sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at);
	CREATE INDEX IF NOT EXISTS idx_articles_read ON articles(read);
	CREATE INDEX IF NOT EXISTS idx_articles_saved ON articles(saved);
	CREATE INDEX IF NOT EXISTS idx_feeds_folder_id ON feeds(folder_id);
	CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

	-- Insert default settings
	INSERT INTO settings (key, value) VALUES 
		('app_title', 'MyFeed'),
		('articles_per_page', '50'),
		('cleanup_after_days', '30'),
		('refresh_interval', '15m')
	ON CONFLICT (key) DO NOTHING;
	`

	_, err := db.Exec(schema)
	return err
}
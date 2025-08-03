package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"myfeed/database"
	"myfeed/models"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db *database.DB
}

func NewAuthService(db *database.DB) *AuthService {
	return &AuthService{db: db}
}

func (as *AuthService) CreateUser(username, password string, isAdmin bool) (*models.User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Check if user already exists
	existingUser, err := as.GetUserByUsername(username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user already exists")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	// Insert the user
	query := `
		INSERT INTO users (username, password, is_admin)
		VALUES (?, ?, ?)
	`
	
	result, err := as.db.Exec(query, username, string(hashedPassword), isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %v", err)
	}

	return as.GetUserByID(int(userID))
}

func (as *AuthService) GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, username, password, is_admin, created_at, last_login
		FROM users WHERE id = ?
	`
	
	user := &models.User{}
	err := as.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Password, &user.IsAdmin,
		&user.CreatedAt, &user.LastLogin,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (as *AuthService) GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password, is_admin, created_at, last_login
		FROM users WHERE username = ?
	`
	
	user := &models.User{}
	err := as.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.IsAdmin,
		&user.CreatedAt, &user.LastLogin,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (as *AuthService) AuthenticateUser(username, password string) (*models.User, error) {
	user, err := as.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	_, err = as.db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", user.ID)
	if err != nil {
		log.Printf("Failed to update last login for user %d: %v", user.ID, err)
	}

	return user, nil
}

func (as *AuthService) CreateSession(userID int) (*models.Session, error) {
	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Session expires in 30 days
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	// Insert session
	query := `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`
	
	_, err = as.db.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return &models.Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}, nil
}

func (as *AuthService) GetSession(sessionID string) (*models.Session, error) {
	query := `
		SELECT id, user_id, created_at, expires_at
		FROM sessions WHERE id = ? AND expires_at > CURRENT_TIMESTAMP
	`
	
	session := &models.Session{}
	err := as.db.QueryRow(query, sessionID).Scan(
		&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return session, nil
}

func (as *AuthService) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := as.db.Exec(query, sessionID)
	return err
}

func (as *AuthService) CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`
	result, err := as.db.Exec(query)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d expired sessions", rowsAffected)
	}
	
	return nil
}

func (as *AuthService) GetUserCount() (int, error) {
	var count int
	err := as.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (as *AuthService) EnsureDefaultAdmin() error {
	// Check if any users exist
	count, err := as.GetUserCount()
	if err != nil {
		return err
	}

	// If no users exist, create default admin from environment
	if count == 0 {
		username := os.Getenv("ADMIN_USERNAME")
		password := os.Getenv("ADMIN_PASSWORD")
		
		if username == "" {
			username = "admin"
		}
		if password == "" {
			password = "admin123" // Default password - should be changed
			log.Println("WARNING: Using default admin password. Please change it!")
		}

		_, err := as.CreateUser(username, password, true)
		if err != nil {
			return fmt.Errorf("failed to create default admin: %v", err)
		}
		
		log.Printf("Created default admin user: %s", username)
	}

	return nil
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
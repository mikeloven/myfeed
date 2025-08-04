package middleware

import (
	"context"
	"encoding/json"
	"log"
	"myfeed/models"
	"myfeed/services"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	authService *services.AuthService
	store       *sessions.CookieStore
}

func NewAuthMiddleware(authService *services.AuthService) *AuthMiddleware {
	// Get session secret from environment
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "default-secret-change-in-production"
		log.Println("WARNING: Using default session secret. Set SESSION_SECRET environment variable!")
	}

	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	return &AuthMiddleware{
		authService: authService,
		store:       store,
	}
}

func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Temporary bypass for debugging - remove after fixing auth issue
		if os.Getenv("DISABLE_AUTH") == "true" {
			log.Println("WARNING: Authentication disabled for debugging")
			// Create a fake admin user for context
			fakeUser := &models.User{ID: 1, Username: "admin", IsAdmin: true}
			ctx := context.WithValue(r.Context(), UserContextKey, fakeUser)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		user := am.getCurrentUser(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthMiddleware) getCurrentUser(r *http.Request) *models.User {
	session, err := am.store.Get(r, "myfeed-session")
	if err != nil {
		return nil
	}

	sessionID, ok := session.Values["session_id"].(string)
	if !ok || sessionID == "" {
		return nil
	}

	// Verify session in database
	dbSession, err := am.authService.GetSession(sessionID)
	if err != nil {
		return nil
	}

	// Get user
	user, err := am.authService.GetUserByID(dbSession.UserID)
	if err != nil {
		return nil
	}

	return user
}

func (am *AuthMiddleware) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := am.authService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid credentials",
		})
		return
	}

	// Create session
	dbSession, err := am.authService.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	session, err := am.store.Get(r, "myfeed-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	session.Values["session_id"] = dbSession.ID
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
	})
}

func (am *AuthMiddleware) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := am.store.Get(r, "myfeed-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	sessionID, ok := session.Values["session_id"].(string)
	if ok && sessionID != "" {
		// Delete session from database
		am.authService.DeleteSession(sessionID)
	}

	// Clear session cookie
	session.Values["session_id"] = ""
	session.Options.MaxAge = -1
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (am *AuthMiddleware) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := am.getCurrentUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Not authenticated",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		},
	})
}

func (am *AuthMiddleware) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := am.getCurrentUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Not authenticated",
		})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := am.authService.ChangePassword(user.ID, req.CurrentPassword, req.NewPassword)
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
		"message": "Password changed successfully",
	})
}

func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
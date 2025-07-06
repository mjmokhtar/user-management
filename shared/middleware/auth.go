package middleware

import (
	"context"
	"net/http"
	"strings"
	"user-management/shared/interfaces"
	"user-management/shared/response"
)

// ContextKey represents context keys
type ContextKey string

const (
	// UserContextKey is the key for user in context
	UserContextKey ContextKey = "user"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authService interfaces.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService interfaces.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate middleware validates JWT token and sets user in context
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Authorization header required")
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(w, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			response.Unauthorized(w, "Token required")
			return
		}

		// Validate token and get user
		user, err := am.authService.GetUserFromToken(tokenString)
		if err != nil {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		// Set user in context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware checks if user has specific permission
func (am *AuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "User not found in context")
				return
			}

			// Check permission
			hasPermission, err := am.authService.HasPermission(user.ID, resource, action)
			if err != nil {
				response.InternalServerError(w, "Failed to check permission", err)
				return
			}

			if !hasPermission {
				response.Forbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware checks if user has specific role
func (am *AuthMiddleware) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "User not found in context")
				return
			}

			// Check role
			if !user.HasRole(roleName) {
				response.Forbidden(w, "Insufficient role")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin middleware checks if user is admin
func (am *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return am.RequireRole("admin")(next)
}

// OptionalAuth middleware validates token if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format, continue without user
			next.ServeHTTP(w, r)
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			// Empty token, continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate token and get user
		user, err := am.authService.GetUserFromToken(tokenString)
		if err != nil {
			// Invalid token, continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Set user in context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves user from request context
func GetUserFromContext(ctx context.Context) (*interfaces.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*interfaces.User)
	return user, ok
}

// CORS middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple request logging
		// In production, use proper logging library
		next.ServeHTTP(w, r)
	})
}

// ContentTypeJSON middleware sets JSON content type
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

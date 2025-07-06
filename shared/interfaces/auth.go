package interfaces

import "time"

// User represents a user entity for authentication
type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	Roles    []Role `json:"roles,omitempty"`
}

// Role represents a user role
type Role struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsActive    bool         `json:"is_active"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission represents a system permission
type Permission struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
}

// HasRole checks if user has specific role
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName && role.IsActive {
			return true
		}
	}
	return false
}

// HasPermission checks if user has specific permission
func (u *User) HasPermission(resource, action string) bool {
	for _, role := range u.Roles {
		if !role.IsActive {
			continue
		}
		for _, perm := range role.Permissions {
			if perm.Resource == resource && perm.Action == action {
				return true
			}
		}
	}
	return false
}

// IsAdmin checks if user is admin
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// AuthService interface for authentication operations
type AuthService interface {
	GetUserFromToken(tokenString string) (*User, error)
	HasPermission(userID int, resource, action string) (bool, error)
}

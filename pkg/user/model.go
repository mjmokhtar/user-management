package user

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user entity
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Hidden from JSON
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Roles        []Role    `json:"roles,omitempty"`
}

// Role represents a user role
type Role struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsActive    bool         `json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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

// UserRole represents user-role mapping
type UserRole struct {
	UserID     int       `json:"user_id"`
	RoleID     int       `json:"role_id"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy int       `json:"assigned_by"`
}

// CreateUserRequest represents request to create user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// UpdateUserRequest represents request to update user
type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

// AssignRoleRequest represents request to assign role to user
type AssignRoleRequest struct {
	UserID     int `json:"user_id"`
	RoleID     int `json:"role_id"`
	AssignedBy int `json:"assigned_by"`
}

// Domain validation errors
var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrPasswordTooWeak = errors.New("password must be at least 8 characters long")
	ErrNameRequired    = errors.New("name is required")
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailExists     = errors.New("email already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInactiveUser    = errors.New("user account is inactive")
	ErrUnauthorized    = errors.New("unauthorized access")
)

// Validate validates CreateUserRequest
func (req *CreateUserRequest) Validate() error {
	// Validate email
	if err := validateEmail(req.Email); err != nil {
		return err
	}

	// Validate password
	if err := validatePassword(req.Password); err != nil {
		return err
	}

	// Validate name
	if err := validateName(req.Name); err != nil {
		return err
	}

	return nil
}

// Validate validates LoginRequest
func (req *LoginRequest) Validate() error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}

	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}

	return nil
}

// Validate validates UpdateUserRequest
func (req *UpdateUserRequest) Validate() error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return ErrNameRequired
	}
	return nil
}

// HashPassword hashes a plain password
func (u *User) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies password against hash
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
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

// HasRole checks if user has specific role
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName && role.IsActive {
			return true
		}
	}
	return false
}

// IsAdmin checks if user is admin
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// GetPermissions returns all user permissions
func (u *User) GetPermissions() []Permission {
	permMap := make(map[string]Permission)

	for _, role := range u.Roles {
		if !role.IsActive {
			continue
		}
		for _, perm := range role.Permissions {
			permMap[perm.Name] = perm
		}
	}

	permissions := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		permissions = append(permissions, perm)
	}

	return permissions
}

// NewUser creates a new User with hashed password
func NewUser(email, password, name string) (*User, error) {
	req := &CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	user := &User{
		Email:    strings.ToLower(strings.TrimSpace(email)),
		Name:     strings.TrimSpace(name),
		IsActive: true,
	}

	if err := user.HashPassword(password); err != nil {
		return nil, err
	}

	return user, nil
}

// Helper validation functions
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email is required")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooWeak
	}
	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len(name) < 2 {
		return errors.New("name must be at least 2 characters long")
	}
	return nil
}

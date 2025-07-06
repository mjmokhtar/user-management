package user

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service defines user service interface
type Service interface {
	// Authentication
	Register(req *CreateUserRequest) (*User, error)
	Login(req *LoginRequest) (*LoginResponse, error)

	// User management
	GetProfile(userID int) (*User, error)
	UpdateProfile(userID int, req *UpdateUserRequest) (*User, error)
	GetUser(userID int) (*User, error)
	ListUsers(page, perPage int) ([]*User, int, error)
	DeactivateUser(userID int) error

	// Role management
	AssignUserRole(userID, roleID, assignedBy int) error
	RemoveUserRole(userID, roleID int) error
	GetUserRoles(userID int) ([]*Role, error)
	ListRoles() ([]*Role, error)

	// Permission checking
	HasPermission(userID int, resource, action string) (bool, error)
	GetUserPermissions(userID int) ([]*Permission, error)

	// JWT operations
	GenerateTokens(user *User) (accessToken, refreshToken string, err error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserFromToken(tokenString string) (*User, error)
}

// service implements Service interface
type service struct {
	repo      Repository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewService creates a new user service
func NewService(repo Repository, jwtSecret string, jwtExpiryHours int) Service {
	return &service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: time.Duration(jwtExpiryHours) * time.Hour,
	}
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// Register creates a new user account
func (s *service) Register(req *CreateUserRequest) (*User, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, err := s.repo.GetByEmail(req.Email)
	if err != nil && err != ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailExists
	}

	// Create new user
	user, err := NewUser(req.Email, req.Password, req.Name)
	if err != nil {
		return nil, err
	}

	// Save to database
	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default "user" role
	userRole, err := s.repo.GetRoleByName("user")
	if err != nil {
		log.Printf("Warning: failed to get default user role: %v", err)
	} else {
		if err := s.repo.AssignRole(user.ID, userRole.ID, user.ID); err != nil {
			log.Printf("Warning: failed to assign default role: %v", err)
		}
	}

	// Load user with roles for response
	userWithRoles, err := s.repo.GetUserWithRoles(user.ID)
	if err != nil {
		log.Printf("Warning: failed to load user roles: %v", err)
		return user, nil
	}

	return userWithRoles, nil
}

// Login authenticates user and returns tokens
func (s *service) Login(req *LoginRequest) (*LoginResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get user by email
	user, err := s.repo.GetByEmail(req.Email)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrInvalidPassword
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	// Verify password
	if err := user.CheckPassword(req.Password); err != nil {
		return nil, ErrInvalidPassword
	}

	// Load user with roles
	userWithRoles, err := s.repo.GetUserWithRoles(user.ID)
	if err != nil {
		log.Printf("Warning: failed to load user roles: %v", err)
		userWithRoles = user
	}

	// Generate tokens
	accessToken, refreshToken, err := s.GenerateTokens(userWithRoles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	response := &LoginResponse{
		User:         userWithRoles,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
	}

	return response, nil
}

// GetProfile returns user profile with roles and permissions
func (s *service) GetProfile(userID int) (*User, error) {
	user, err := s.repo.GetUserWithRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return user, nil
}

// UpdateProfile updates user profile
func (s *service) UpdateProfile(userID int, req *UpdateUserRequest) (*User, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Update user
	user, err := s.repo.Update(userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Load with roles
	userWithRoles, err := s.repo.GetUserWithRoles(user.ID)
	if err != nil {
		log.Printf("Warning: failed to load user roles: %v", err)
		return user, nil
	}

	return userWithRoles, nil
}

// GetUser returns user by ID (admin function)
func (s *service) GetUser(userID int) (*User, error) {
	user, err := s.repo.GetUserWithRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ListUsers returns paginated list of users
func (s *service) ListUsers(page, perPage int) ([]*User, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	users, total, err := s.repo.List(perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Load roles for each user (could be optimized with batch loading)
	for _, user := range users {
		roles, err := s.repo.GetUserRoles(user.ID)
		if err != nil {
			log.Printf("Warning: failed to load roles for user %d: %v", user.ID, err)
			continue
		}

		// Convert []*Role to []Role
		user.Roles = make([]Role, len(roles))
		for i, role := range roles {
			user.Roles[i] = *role
		}
	}

	return users, total, nil
}

// DeactivateUser deactivates a user account
func (s *service) DeactivateUser(userID int) error {
	if err := s.repo.Delete(userID); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

// AssignUserRole assigns a role to user
func (s *service) AssignUserRole(userID, roleID, assignedBy int) error {
	// Verify user exists
	_, err := s.repo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify role exists
	_, err = s.repo.GetRoleByID(roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Assign role
	if err := s.repo.AssignRole(userID, roleID, assignedBy); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveUserRole removes a role from user
func (s *service) RemoveUserRole(userID, roleID int) error {
	if err := s.repo.RemoveRole(userID, roleID); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	return nil
}

// GetUserRoles returns all roles for a user
func (s *service) GetUserRoles(userID int) ([]*Role, error) {
	roles, err := s.repo.GetUserRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return roles, nil
}

// ListRoles returns all available roles
func (s *service) ListRoles() ([]*Role, error) {
	roles, err := s.repo.ListRoles()
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return roles, nil
}

// HasPermission checks if user has specific permission
func (s *service) HasPermission(userID int, resource, action string) (bool, error) {
	hasPermission, err := s.repo.HasPermission(userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return hasPermission, nil
}

// GetUserPermissions returns all permissions for a user
func (s *service) GetUserPermissions(userID int) ([]*Permission, error) {
	permissions, err := s.repo.GetUserPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return permissions, nil
}

// GenerateTokens generates access and refresh tokens
func (s *service) GenerateTokens(user *User) (accessToken, refreshToken string, err error) {
	// Create access token claims
	accessClaims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "user-management-api",
			Subject:   fmt.Sprintf("user:%d", user.ID),
		},
	}

	// Generate access token
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create refresh token claims (longer expiry)
	refreshClaims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "user-management-api",
			Subject:   fmt.Sprintf("refresh:%d", user.ID),
		},
	}

	// Generate refresh token
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateToken validates JWT token and returns parsed token
func (s *service) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

// GetUserFromToken extracts user information from JWT token
func (s *service) GetUserFromToken(tokenString string) (*User, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Get user with current data from database
	user, err := s.repo.GetUserWithRoles(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from token: %w", err)
	}

	// Check if user is still active
	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	return user, nil
}

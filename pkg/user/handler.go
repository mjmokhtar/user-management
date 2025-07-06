package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"user-management/shared/middleware"
	"user-management/shared/response"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	service Service
	authMW  *middleware.AuthMiddleware
}

// NewHandler creates a new user handler
func NewHandler(service Service) *Handler {
	authService := NewAuthServiceAdapter(service)
	return &Handler{
		service: service,
		authMW:  middleware.NewAuthMiddleware(authService),
	}
}

// RegisterRoutes registers all user routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public routes (no authentication required)
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)

	// Protected routes (authentication required)
	mux.Handle("GET /api/auth/profile", h.authMW.Authenticate(http.HandlerFunc(h.GetProfile)))
	mux.Handle("PUT /api/auth/profile", h.authMW.Authenticate(http.HandlerFunc(h.UpdateProfile)))

	// Admin routes (admin role required)
	mux.Handle("GET /api/users", h.authMW.RequireAdmin(http.HandlerFunc(h.ListUsers)))
	mux.Handle("GET /api/users/{id}", h.authMW.RequireAdmin(http.HandlerFunc(h.GetUser)))
	mux.Handle("PUT /api/users/{id}", h.authMW.RequireAdmin(http.HandlerFunc(h.UpdateUser)))
	mux.Handle("DELETE /api/users/{id}", h.authMW.RequireAdmin(http.HandlerFunc(h.DeactivateUser)))

	// Role management (admin only)
	mux.Handle("GET /api/roles", h.authMW.RequireAdmin(http.HandlerFunc(h.ListRoles)))
	mux.Handle("POST /api/users/roles", h.authMW.RequireAdmin(http.HandlerFunc(h.AssignRole)))
	mux.Handle("DELETE /api/users/roles", h.authMW.RequireAdmin(http.HandlerFunc(h.RemoveRole)))
	mux.Handle("GET /api/users/{id}/roles", h.authMW.RequireAdmin(http.HandlerFunc(h.GetUserRoles)))

	// Permission checking (authenticated users)
	mux.Handle("GET /api/auth/permissions", h.authMW.Authenticate(http.HandlerFunc(h.GetMyPermissions)))
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	user, err := h.service.Register(&req)
	if err != nil {
		switch err {
		case ErrInvalidEmail, ErrPasswordTooWeak, ErrNameRequired:
			response.BadRequest(w, "Validation failed", err)
		case ErrEmailExists:
			response.Conflict(w, "Email already exists", err)
		default:
			response.InternalServerError(w, "Failed to register user", err)
		}
		return
	}

	// Remove sensitive data
	user.PasswordHash = ""

	response.Created(w, "User registered successfully", user)
}

// Login handles user authentication
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	loginResp, err := h.service.Login(&req)
	if err != nil {
		switch err {
		case ErrInvalidEmail:
			response.BadRequest(w, "Invalid email format", err)
		case ErrInvalidPassword, ErrUserNotFound:
			response.Unauthorized(w, "Invalid email or password")
		case ErrInactiveUser:
			response.Forbidden(w, "Account is inactive")
		default:
			response.InternalServerError(w, "Login failed", err)
		}
		return
	}

	// Remove sensitive data
	loginResp.User.PasswordHash = ""

	response.Success(w, "Login successful", loginResp)
}

// GetProfile returns current user profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	profile, err := h.service.GetProfile(user.ID)
	if err != nil {
		response.InternalServerError(w, "Failed to get profile", err)
		return
	}

	// Remove sensitive data
	profile.PasswordHash = ""

	response.Success(w, "Profile retrieved successfully", profile)
}

// UpdateProfile updates current user profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	updatedUser, err := h.service.UpdateProfile(user.ID, &req)
	if err != nil {
		switch err {
		case ErrNameRequired:
			response.BadRequest(w, "Validation failed", err)
		case ErrUserNotFound:
			response.NotFound(w, "User not found")
		default:
			response.InternalServerError(w, "Failed to update profile", err)
		}
		return
	}

	// Remove sensitive data
	updatedUser.PasswordHash = ""

	response.Success(w, "Profile updated successfully", updatedUser)
}

// ListUsers returns paginated list of users (admin only)
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	users, total, err := h.service.ListUsers(page, perPage)
	if err != nil {
		response.InternalServerError(w, "Failed to list users", err)
		return
	}

	// Remove sensitive data
	for _, user := range users {
		user.PasswordHash = ""
	}

	// Calculate pagination meta
	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.PaginatedSuccess(w, "Users retrieved successfully", users, meta)
}

// GetUser returns specific user by ID (admin only)
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID", err)
		return
	}

	user, err := h.service.GetUser(userID)
	if err != nil {
		switch err {
		case ErrUserNotFound:
			response.NotFound(w, "User not found")
		default:
			response.InternalServerError(w, "Failed to get user", err)
		}
		return
	}

	// Remove sensitive data
	user.PasswordHash = ""

	response.Success(w, "User retrieved successfully", user)
}

// UpdateUser updates specific user (admin only)
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID", err)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	updatedUser, err := h.service.UpdateProfile(userID, &req)
	if err != nil {
		switch err {
		case ErrNameRequired:
			response.BadRequest(w, "Validation failed", err)
		case ErrUserNotFound:
			response.NotFound(w, "User not found")
		default:
			response.InternalServerError(w, "Failed to update user", err)
		}
		return
	}

	// Remove sensitive data
	updatedUser.PasswordHash = ""

	response.Success(w, "User updated successfully", updatedUser)
}

// DeactivateUser deactivates specific user (admin only)
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID", err)
		return
	}

	// Prevent admin from deactivating themselves
	currentUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	if currentUser.ID == userID {
		response.BadRequest(w, "Cannot deactivate your own account", nil)
		return
	}

	if err := h.service.DeactivateUser(userID); err != nil {
		switch err {
		case ErrUserNotFound:
			response.NotFound(w, "User not found")
		default:
			response.InternalServerError(w, "Failed to deactivate user", err)
		}
		return
	}

	response.Success(w, "User deactivated successfully", nil)
}

// ListRoles returns all available roles (admin only)
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles()
	if err != nil {
		response.InternalServerError(w, "Failed to list roles", err)
		return
	}

	response.Success(w, "Roles retrieved successfully", roles)
}

// AssignRole assigns role to user (admin only)
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	req.AssignedBy = currentUser.ID

	if err := h.service.AssignUserRole(req.UserID, req.RoleID, req.AssignedBy); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, "User or role not found")
		} else {
			response.InternalServerError(w, "Failed to assign role", err)
		}
		return
	}

	response.Success(w, "Role assigned successfully", nil)
}

// RemoveRole removes role from user (admin only)
func (h *Handler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int `json:"user_id"`
		RoleID int `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	if err := h.service.RemoveUserRole(req.UserID, req.RoleID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, "User role not found")
		} else {
			response.InternalServerError(w, "Failed to remove role", err)
		}
		return
	}

	response.Success(w, "Role removed successfully", nil)
}

// GetUserRoles returns roles for specific user (admin only)
func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID", err)
		return
	}

	roles, err := h.service.GetUserRoles(userID)
	if err != nil {
		response.InternalServerError(w, "Failed to get user roles", err)
		return
	}

	response.Success(w, "User roles retrieved successfully", roles)
}

// GetMyPermissions returns current user's permissions
func (h *Handler) GetMyPermissions(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	permissions, err := h.service.GetUserPermissions(user.ID)
	if err != nil {
		response.InternalServerError(w, "Failed to get permissions", err)
		return
	}

	response.Success(w, "Permissions retrieved successfully", permissions)
}

// Helper function to extract ID from URL path
func extractIDFromPath(path, prefix string) (int, error) {
	if !strings.HasPrefix(path, prefix) {
		return 0, ErrUserNotFound
	}

	// Remove prefix and potential suffix (like /roles)
	idPart := strings.TrimPrefix(path, prefix)
	parts := strings.Split(idPart, "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, ErrUserNotFound
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	return id, nil
}

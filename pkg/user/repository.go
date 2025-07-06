package user

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository defines user repository interface
type Repository interface {
	// User CRUD operations
	Create(user *User) error
	GetByID(id int) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(id int, req *UpdateUserRequest) (*User, error)
	Delete(id int) error
	List(limit, offset int) ([]*User, int, error)

	// Role operations
	GetRoleByID(id int) (*Role, error)
	GetRoleByName(name string) (*Role, error)
	ListRoles() ([]*Role, error)

	// User-Role operations
	AssignRole(userID, roleID, assignedBy int) error
	RemoveRole(userID, roleID int) error
	GetUserRoles(userID int) ([]*Role, error)
	GetUserWithRoles(userID int) (*User, error)

	// Permission operations
	GetUserPermissions(userID int) ([]*Permission, error)
	HasPermission(userID int, resource, action string) (bool, error)
}

// repository implements Repository interface
type repository struct {
	db *sql.DB
}

// NewRepository creates a new user repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Schema name constant
const schema = "user_management"

// Create creates a new user
func (r *repository) Create(user *User) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.users (email, password_hash, name, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, schema)

	err := r.db.QueryRow(query, user.Email, user.PasswordHash, user.Name, user.IsActive).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrEmailExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves user by ID
func (r *repository) GetByID(id int) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, is_active, created_at, updated_at
		FROM %s.users
		WHERE id = $1
	`, schema)

	user := &User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves user by email
func (r *repository) GetByEmail(email string) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, is_active, created_at, updated_at
		FROM %s.users
		WHERE email = $1
	`, schema)

	user := &User{}
	err := r.db.QueryRow(query, strings.ToLower(email)).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates user information
func (r *repository) Update(id int, req *UpdateUserRequest) (*User, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(id) // No changes, return current user
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE %s.users 
		SET %s
		WHERE id = $%d
		RETURNING id, email, password_hash, name, is_active, created_at, updated_at
	`, schema, strings.Join(setParts, ", "), argIndex)

	user := &User{}
	err := r.db.QueryRow(query, args...).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// Delete soft deletes a user (sets is_active to false)
func (r *repository) Delete(id int) error {
	query := fmt.Sprintf(`
		UPDATE %s.users 
		SET is_active = false, updated_at = $1
		WHERE id = $2
	`, schema)

	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List retrieves paginated list of users
func (r *repository) List(limit, offset int) ([]*User, int, error) {
	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s.users WHERE is_active = true", schema)
	var total int
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, is_active, created_at, updated_at
		FROM %s.users
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, schema)

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name,
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

// GetRoleByID retrieves role by ID
func (r *repository) GetRoleByID(id int) (*Role, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM %s.roles
		WHERE id = $1
	`, schema)

	role := &Role{}
	err := r.db.QueryRow(query, id).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by ID: %w", err)
	}

	return role, nil
}

// GetRoleByName retrieves role by name
func (r *repository) GetRoleByName(name string) (*Role, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM %s.roles
		WHERE name = $1
	`, schema)

	role := &Role{}
	err := r.db.QueryRow(query, name).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsActive, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return role, nil
}

// ListRoles retrieves all active roles
func (r *repository) ListRoles() ([]*Role, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM %s.roles
		WHERE is_active = true
		ORDER BY name
	`, schema)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	roles := []*Role{}
	for rows.Next() {
		role := &Role{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.IsActive, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRole assigns a role to user
func (r *repository) AssignRole(userID, roleID, assignedBy int) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, schema)

	_, err := r.db.Exec(query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveRole removes a role from user
func (r *repository) RemoveRole(userID, roleID int) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.user_roles
		WHERE user_id = $1 AND role_id = $2
	`, schema)

	result, err := r.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user role not found")
	}

	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *repository) GetUserRoles(userID int) ([]*Role, error) {
	query := fmt.Sprintf(`
		SELECT r.id, r.name, r.description, r.is_active, r.created_at, r.updated_at
		FROM %s.roles r
		INNER JOIN %s.user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.is_active = true
		ORDER BY r.name
	`, schema, schema)

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	roles := []*Role{}
	for rows.Next() {
		role := &Role{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.IsActive, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetUserWithRoles retrieves user with their roles and permissions
func (r *repository) GetUserWithRoles(userID int) (*User, error) {
	// Get user
	user, err := r.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// Get user roles with permissions
	query := fmt.Sprintf(`
		SELECT DISTINCT r.id, r.name, r.description, r.is_active, r.created_at, r.updated_at,
		       p.id, p.name, p.description, p.resource, p.action, p.created_at
		FROM %s.roles r
		INNER JOIN %s.user_roles ur ON r.id = ur.role_id
		LEFT JOIN %s.role_permissions rp ON r.id = rp.role_id
		LEFT JOIN %s.permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1 AND r.is_active = true
		ORDER BY r.name, p.name
	`, schema, schema, schema, schema)

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with roles: %w", err)
	}
	defer rows.Close()

	roleMap := make(map[int]*Role)

	for rows.Next() {
		var roleID, permID sql.NullInt64
		var roleName, roleDesc sql.NullString
		var roleActive sql.NullBool
		var roleCreated, roleUpdated sql.NullTime
		var permName, permDesc, permResource, permAction sql.NullString
		var permCreated sql.NullTime

		err := rows.Scan(
			&roleID, &roleName, &roleDesc, &roleActive, &roleCreated, &roleUpdated,
			&permID, &permName, &permDesc, &permResource, &permAction, &permCreated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role with permissions: %w", err)
		}

		if roleID.Valid {
			role, exists := roleMap[int(roleID.Int64)]
			if !exists {
				role = &Role{
					ID:          int(roleID.Int64),
					Name:        roleName.String,
					Description: roleDesc.String,
					IsActive:    roleActive.Bool,
					CreatedAt:   roleCreated.Time,
					UpdatedAt:   roleUpdated.Time,
					Permissions: []Permission{},
				}
				roleMap[int(roleID.Int64)] = role
			}

			// Add permission if exists
			if permID.Valid {
				permission := Permission{
					ID:          int(permID.Int64),
					Name:        permName.String,
					Description: permDesc.String,
					Resource:    permResource.String,
					Action:      permAction.String,
					CreatedAt:   permCreated.Time,
				}
				role.Permissions = append(role.Permissions, permission)
			}
		}
	}

	// Convert map to slice
	user.Roles = make([]Role, 0, len(roleMap))
	for _, role := range roleMap {
		user.Roles = append(user.Roles, *role)
	}

	return user, nil
}

// GetUserPermissions retrieves all permissions for a user
func (r *repository) GetUserPermissions(userID int) ([]*Permission, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.name, p.description, p.resource, p.action, p.created_at
		FROM %s.permissions p
		INNER JOIN %s.role_permissions rp ON p.id = rp.permission_id
		INNER JOIN %s.roles r ON rp.role_id = r.id
		INNER JOIN %s.user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.is_active = true
		ORDER BY p.resource, p.action
	`, schema, schema, schema, schema)

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	permissions := []*Permission{}
	for rows.Next() {
		perm := &Permission{}
		err := rows.Scan(
			&perm.ID, &perm.Name, &perm.Description,
			&perm.Resource, &perm.Action, &perm.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// HasPermission checks if user has specific permission
func (r *repository) HasPermission(userID int, resource, action string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.permissions p
		INNER JOIN %s.role_permissions rp ON p.id = rp.permission_id
		INNER JOIN %s.roles r ON rp.role_id = r.id
		INNER JOIN %s.user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3 AND r.is_active = true
	`, schema, schema, schema, schema)

	var count int
	err := r.db.QueryRow(query, userID, resource, action).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return count > 0, nil
}

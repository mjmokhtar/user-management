package user

import (
	"user-management/shared/interfaces"
)

// AuthServiceAdapter adapts user.Service to interfaces.AuthService
type AuthServiceAdapter struct {
	userService Service
}

// NewAuthServiceAdapter creates a new auth service adapter
func NewAuthServiceAdapter(userService Service) interfaces.AuthService {
	return &AuthServiceAdapter{
		userService: userService,
	}
}

// GetUserFromToken adapts the method to return interfaces.User
func (a *AuthServiceAdapter) GetUserFromToken(tokenString string) (*interfaces.User, error) {
	// Get user from user service
	user, err := a.userService.GetUserFromToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Convert to interfaces.User
	interfaceUser := &interfaces.User{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		IsActive: user.IsActive,
		Roles:    make([]interfaces.Role, len(user.Roles)),
	}

	// Convert roles
	for i, role := range user.Roles {
		interfaceRole := interfaces.Role{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			IsActive:    role.IsActive,
			Permissions: make([]interfaces.Permission, len(role.Permissions)),
		}

		// Convert permissions
		for j, perm := range role.Permissions {
			interfaceRole.Permissions[j] = interfaces.Permission{
				ID:          perm.ID,
				Name:        perm.Name,
				Description: perm.Description,
				Resource:    perm.Resource,
				Action:      perm.Action,
				CreatedAt:   perm.CreatedAt,
			}
		}

		interfaceUser.Roles[i] = interfaceRole
	}

	return interfaceUser, nil
}

// HasPermission delegates to user service
func (a *AuthServiceAdapter) HasPermission(userID int, resource, action string) (bool, error) {
	return a.userService.HasPermission(userID, resource, action)
}

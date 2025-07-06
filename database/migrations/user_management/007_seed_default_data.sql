-- Migration: 007_seed_default_data.sql
-- Module: user_management
-- Description: Seed default user management data

-- UP
-- Insert default roles
INSERT INTO user_management.roles (name, description) 
VALUES 
    ('admin', 'System administrator with full access'),
    ('user', 'Regular user with limited access')
ON CONFLICT (name) DO NOTHING;

-- Insert default permissions
INSERT INTO user_management.permissions (name, description, resource, action) VALUES
    ('users:read', 'Read user data', 'users', 'read'),
    ('users:write', 'Create and update users', 'users', 'write'),
    ('users:delete', 'Delete users', 'users', 'delete'),
    ('roles:read', 'Read roles', 'roles', 'read'),
    ('roles:write', 'Create and update roles', 'roles', 'write'),
    ('roles:delete', 'Delete roles', 'roles', 'delete'),
    ('permissions:read', 'Read permissions', 'permissions', 'read'),
    ('dashboard:read', 'Access dashboard', 'dashboard', 'read')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO user_management.role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM user_management.roles r, user_management.permissions p 
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Assign basic permissions to user role
INSERT INTO user_management.role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM user_management.roles r, user_management.permissions p 
WHERE r.name = 'user' AND p.name IN ('dashboard:read', 'users:read')
ON CONFLICT DO NOTHING;

-- DOWN
DELETE FROM user_management.role_permissions;
DELETE FROM user_management.user_roles;
DELETE FROM user_management.permissions;
DELETE FROM user_management.roles;
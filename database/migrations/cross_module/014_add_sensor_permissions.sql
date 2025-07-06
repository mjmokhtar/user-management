-- Migration: 014_add_sensor_permissions.sql
-- Module: cross_module
-- Description: Add sensor permissions to user management

-- UP
-- Add sensor permissions
INSERT INTO user_management.permissions (name, description, resource, action) VALUES
    ('sensors:read', 'Read sensor data', 'sensors', 'read'),
    ('sensors:write', 'Create and update sensors', 'sensors', 'write'),
    ('sensors:delete', 'Delete sensors', 'sensors', 'delete'),
    ('sensor_readings:read', 'Read sensor readings', 'sensor_readings', 'read'),
    ('sensor_readings:write', 'Create sensor readings', 'sensor_readings', 'write'),
    ('analytics:read', 'Access analytics data', 'analytics', 'read'),
    ('locations:read', 'Read location data', 'locations', 'read'),
    ('locations:write', 'Create and update locations', 'locations', 'write')
ON CONFLICT (name) DO NOTHING;

-- Assign sensor permissions to admin role
INSERT INTO user_management.role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM user_management.roles r, user_management.permissions p 
WHERE r.name = 'admin' AND (
    p.name LIKE 'sensor%' OR 
    p.name LIKE 'analytics%' OR 
    p.name LIKE 'locations%'
)
ON CONFLICT DO NOTHING;

-- Assign basic sensor read permission to user role
INSERT INTO user_management.role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM user_management.roles r, user_management.permissions p 
WHERE r.name = 'user' AND p.name IN (
    'sensors:read', 
    'sensor_readings:read', 
    'locations:read'
)
ON CONFLICT DO NOTHING;

-- DOWN
DELETE FROM user_management.role_permissions WHERE permission_id IN (
    SELECT id FROM user_management.permissions 
    WHERE resource IN ('sensors', 'sensor_readings', 'analytics', 'locations')
);
DELETE FROM user_management.permissions 
WHERE resource IN ('sensors', 'sensor_readings', 'analytics', 'locations');
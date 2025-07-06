-- Migration: 006_create_role_permissions_table.sql
-- Module: user_management
-- Description: Create role_permissions mapping table

-- UP
CREATE TABLE IF NOT EXISTS user_management.role_permissions (
    role_id INTEGER REFERENCES user_management.roles(id) ON DELETE CASCADE,
    permission_id INTEGER REFERENCES user_management.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- DOWN
DROP TABLE IF EXISTS user_management.role_permissions CASCADE;
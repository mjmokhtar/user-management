-- Migration: 005_create_user_roles_table.sql
-- Module: user_management
-- Description: Create user_roles mapping table

-- UP
CREATE TABLE IF NOT EXISTS user_management.user_roles (
    user_id INTEGER REFERENCES user_management.users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES user_management.roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_by INTEGER REFERENCES user_management.users(id),
    PRIMARY KEY (user_id, role_id)
);

-- DOWN
DROP TABLE IF EXISTS user_management.user_roles CASCADE;
-- Migration: 004_create_permissions_table.sql
-- Module: user_management
-- Description: Create permissions table

-- UP
CREATE TABLE IF NOT EXISTS user_management.permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON user_management.permissions(resource, action);

-- DOWN
DROP TABLE IF EXISTS user_management.permissions CASCADE;
-- Migration: 003_create_roles_table.sql
-- Module: user_management
-- Description: Create roles table

-- UP
CREATE TABLE IF NOT EXISTS user_management.roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_roles_name ON user_management.roles(name);

-- DOWN
DROP TABLE IF EXISTS user_management.roles CASCADE;
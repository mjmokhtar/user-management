-- Migration: 002_create_users_table.sql
-- Module: user_management
-- Description: Create users table

-- UP
CREATE TABLE IF NOT EXISTS user_management.users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON user_management.users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON user_management.users(is_active);

-- DOWN
DROP TABLE IF EXISTS user_management.users CASCADE;
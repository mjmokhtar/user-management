-- Migration: 001_create_schema.sql
-- Module: user_management
-- Description: Create user_management schema

-- UP
CREATE SCHEMA IF NOT EXISTS user_management;

-- DOWN
DROP SCHEMA IF EXISTS user_management CASCADE;
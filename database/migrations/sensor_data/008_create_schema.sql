-- Migration: 008_create_schema.sql
-- Module: sensor_data
-- Description: Create sensor_data schema

-- UP
CREATE SCHEMA IF NOT EXISTS sensor_data;

-- DOWN
DROP SCHEMA IF EXISTS sensor_data CASCADE;
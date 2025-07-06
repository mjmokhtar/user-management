-- Migration: 010_create_locations_table.sql
-- Module: sensor_data
-- Description: Create locations table

-- UP
CREATE TABLE IF NOT EXISTS sensor_data.locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    address TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_locations_name ON sensor_data.locations(name);
CREATE INDEX IF NOT EXISTS idx_locations_coordinates ON sensor_data.locations(latitude, longitude);

-- Insert default locations
INSERT INTO sensor_data.locations (name, description, latitude, longitude, address) VALUES
    ('Building A - Floor 1', 'Main building first floor', -6.2088, 106.8456, 'Jakarta, Indonesia'),
    ('Building A - Floor 2', 'Main building second floor', -6.2088, 106.8456, 'Jakarta, Indonesia'),
    ('Building B - Lab', 'Laboratory building', -6.2090, 106.8458, 'Jakarta, Indonesia'),
    ('Outdoor - Parking', 'Outdoor parking area', -6.2085, 106.8460, 'Jakarta, Indonesia'),
    ('Warehouse', 'Storage warehouse', -6.2092, 106.8454, 'Jakarta, Indonesia')
ON CONFLICT DO NOTHING;

-- DOWN
DROP TABLE IF EXISTS sensor_data.locations CASCADE;
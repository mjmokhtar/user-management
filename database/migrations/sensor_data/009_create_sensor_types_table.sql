-- Migration: 009_create_sensor_types_table.sql
-- Module: sensor_data
-- Description: Create sensor types table

-- UP
CREATE TABLE IF NOT EXISTS sensor_data.sensor_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    unit VARCHAR(20) NOT NULL,
    min_value DECIMAL(10,2),
    max_value DECIMAL(10,2),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sensor_types_name ON sensor_data.sensor_types(name);
CREATE INDEX IF NOT EXISTS idx_sensor_types_active ON sensor_data.sensor_types(is_active);

-- Insert default sensor types
INSERT INTO sensor_data.sensor_types (name, description, unit, min_value, max_value) VALUES
    ('temperature', 'Temperature sensor', '°C', -50.00, 100.00),
    ('humidity', 'Humidity sensor', '%', 0.00, 100.00),
    ('pressure', 'Pressure sensor', 'hPa', 800.00, 1200.00),
    ('light', 'Light intensity sensor', 'lux', 0.00, 100000.00),
    ('motion', 'Motion detection sensor', 'boolean', 0.00, 1.00),
    ('co2', 'Carbon dioxide sensor', 'ppm', 0.00, 5000.00),
    ('voltage', 'Voltage sensor', 'V', 0.00, 50.00),
    ('current', 'Current sensor', 'A', 0.00, 100.00)
ON CONFLICT (name) DO NOTHING;

-- DOWN
DROP TABLE IF EXISTS sensor_data.sensor_types CASCADE;
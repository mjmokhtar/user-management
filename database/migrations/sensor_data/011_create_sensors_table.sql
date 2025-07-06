-- Migration: 011_create_sensors_table.sql
-- Module: sensor_data
-- Description: Create sensors table

-- UP
CREATE TABLE IF NOT EXISTS sensor_data.sensors (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sensor_type_id INTEGER REFERENCES sensor_data.sensor_types(id),
    location_id INTEGER REFERENCES sensor_data.locations(id),
    is_active BOOLEAN DEFAULT true,
    last_reading_at TIMESTAMP,
    battery_level INTEGER CHECK (battery_level >= 0 AND battery_level <= 100),
    firmware_version VARCHAR(50),
    created_by INTEGER REFERENCES user_management.users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sensors_device_id ON sensor_data.sensors(device_id);
CREATE INDEX IF NOT EXISTS idx_sensors_type ON sensor_data.sensors(sensor_type_id);
CREATE INDEX IF NOT EXISTS idx_sensors_location ON sensor_data.sensors(location_id);
CREATE INDEX IF NOT EXISTS idx_sensors_active ON sensor_data.sensors(is_active);
CREATE INDEX IF NOT EXISTS idx_sensors_last_reading ON sensor_data.sensors(last_reading_at);

-- DOWN
DROP TABLE IF EXISTS sensor_data.sensors CASCADE;
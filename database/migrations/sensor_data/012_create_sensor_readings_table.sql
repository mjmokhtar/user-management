-- Migration: 012_create_sensor_readings_table.sql
-- Module: sensor_data
-- Description: Create sensor_readings table

-- UP
CREATE TABLE IF NOT EXISTS sensor_data.sensor_readings (
    id BIGSERIAL PRIMARY KEY,
    sensor_id INTEGER REFERENCES sensor_data.sensors(id) ON DELETE CASCADE,
    value DECIMAL(15,4) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    quality INTEGER DEFAULT 100 CHECK (quality >= 0 AND quality <= 100),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for time-series queries
CREATE INDEX IF NOT EXISTS idx_sensor_readings_sensor_time ON sensor_data.sensor_readings(sensor_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_readings_timestamp ON sensor_data.sensor_readings(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_readings_quality ON sensor_data.sensor_readings(quality);

-- Partial index for recent readings (using static date for immutability)
-- Note: This index should be recreated periodically for optimal performance
CREATE INDEX IF NOT EXISTS idx_sensor_readings_recent ON sensor_data.sensor_readings(sensor_id, timestamp DESC) 
WHERE timestamp >= '2025-01-01'::timestamp;

-- DOWN
DROP TABLE IF EXISTS sensor_data.sensor_readings CASCADE;
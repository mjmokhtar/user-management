-- Migration: 013_seed_default_data.sql
-- Module: sensor_data
-- Description: Seed additional sensor data

-- UP
-- Additional sensor types if needed
INSERT INTO sensor_data.sensor_types (name, description, unit, min_value, max_value) VALUES
    ('ph', 'pH level sensor', 'pH', 0.00, 14.00),
    ('conductivity', 'Electrical conductivity sensor', 'µS/cm', 0.00, 10000.00),
    ('flow', 'Flow rate sensor', 'L/min', 0.00, 1000.00),
    ('level', 'Water level sensor', 'cm', 0.00, 500.00)
ON CONFLICT (name) DO NOTHING;

-- Note: Sample sensors and readings will be created after user system is set up
-- This can be done via API endpoints or separate data seeding script

-- DOWN
DELETE FROM sensor_data.sensor_types WHERE name IN ('ph', 'conductivity', 'flow', 'level');
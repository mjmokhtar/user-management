-- Migration: 015_seed_sample_data.sql
-- Module: cross_module
-- Description: Seed sample data for testing (requires both user and sensor schemas)

-- UP
-- Create sample admin user first
INSERT INTO user_management.users (email, password_hash, name, is_active) VALUES
    ('admin@iot.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeOLLU5UlEGsK.7J2', 'System Admin', true),
    ('sensor_operator@iot.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LeOLLU5UlEGsK.7J2', 'Sensor Operator', true)
ON CONFLICT (email) DO NOTHING;

-- Assign admin role to admin user
INSERT INTO user_management.user_roles (user_id, role_id, assigned_by)
SELECT u.id, r.id, u.id
FROM user_management.users u, user_management.roles r
WHERE u.email = 'admin@iot.com' AND r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Assign user role to operator
INSERT INTO user_management.user_roles (user_id, role_id, assigned_by)
SELECT u.id, r.id, (SELECT id FROM user_management.users WHERE email = 'admin@iot.com')
FROM user_management.users u, user_management.roles r
WHERE u.email = 'sensor_operator@iot.com' AND r.name = 'user'
ON CONFLICT DO NOTHING;

-- Now create sample sensors with proper created_by reference
INSERT INTO sensor_data.sensors (device_id, name, description, sensor_type_id, location_id, firmware_version, created_by) 
SELECT 
    'TEMP-001-A1', 
    'Office Temperature Sensor', 
    'Main office temperature monitoring',
    st.id,
    l.id,
    'v1.2.3',
    u.id
FROM sensor_data.sensor_types st, 
     sensor_data.locations l,
     user_management.users u
WHERE st.name = 'temperature' 
  AND l.name = 'Building A - Floor 1'
  AND u.email = 'admin@iot.com'
ON CONFLICT (device_id) DO NOTHING;

INSERT INTO sensor_data.sensors (device_id, name, description, sensor_type_id, location_id, firmware_version, created_by) 
SELECT 
    'HUM-001-A1', 
    'Office Humidity Sensor', 
    'Main office humidity monitoring',
    st.id,
    l.id,
    'v1.2.3',
    u.id
FROM sensor_data.sensor_types st, 
     sensor_data.locations l,
     user_management.users u
WHERE st.name = 'humidity' 
  AND l.name = 'Building A - Floor 1'
  AND u.email = 'admin@iot.com'
ON CONFLICT (device_id) DO NOTHING;

INSERT INTO sensor_data.sensors (device_id, name, description, sensor_type_id, location_id, firmware_version, created_by) 
SELECT 
    'TEMP-002-B1', 
    'Lab Temperature Sensor', 
    'Laboratory temperature monitoring',
    st.id,
    l.id,
    'v1.2.3',
    u.id
FROM sensor_data.sensor_types st, 
     sensor_data.locations l,
     user_management.users u
WHERE st.name = 'temperature' 
  AND l.name = 'Building B - Lab'
  AND u.email = 'admin@iot.com'
ON CONFLICT (device_id) DO NOTHING;

-- Sample sensor readings (for testing)
INSERT INTO sensor_data.sensor_readings (sensor_id, value, timestamp, quality)
SELECT 
    s.id,
    22.5 + (RANDOM() * 5), -- Random temperature between 22.5-27.5°C
    CURRENT_TIMESTAMP - (INTERVAL '1 hour' * generate_series),
    95 + (RANDOM() * 5)::INTEGER -- Quality between 95-100%
FROM sensor_data.sensors s,
     generate_series(1, 24) -- Last 24 hours of data
WHERE s.device_id = 'TEMP-001-A1';

INSERT INTO sensor_data.sensor_readings (sensor_id, value, timestamp, quality)
SELECT 
    s.id,
    45.0 + (RANDOM() * 20), -- Random humidity between 45-65%
    CURRENT_TIMESTAMP - (INTERVAL '1 hour' * generate_series),
    90 + (RANDOM() * 10)::INTEGER -- Quality between 90-100%
FROM sensor_data.sensors s,
     generate_series(1, 24) -- Last 24 hours of data
WHERE s.device_id = 'HUM-001-A1';

-- Update last_reading_at for sensors
UPDATE sensor_data.sensors 
SET last_reading_at = (
    SELECT MAX(timestamp) 
    FROM sensor_data.sensor_readings sr 
    WHERE sr.sensor_id = sensors.id
)
WHERE EXISTS (
    SELECT 1 FROM sensor_data.sensor_readings sr 
    WHERE sr.sensor_id = sensors.id
);

-- DOWN
DELETE FROM sensor_data.sensor_readings;
DELETE FROM sensor_data.sensors WHERE device_id IN ('TEMP-001-A1', 'HUM-001-A1', 'TEMP-002-B1');
DELETE FROM user_management.user_roles WHERE user_id IN (
    SELECT id FROM user_management.users WHERE email IN ('admin@iot.com', 'sensor_operator@iot.com')
);
DELETE FROM user_management.users WHERE email IN ('admin@iot.com', 'sensor_operator@iot.com');
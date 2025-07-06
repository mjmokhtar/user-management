package sensor

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository defines sensor repository interface
type Repository interface {
	// Sensor CRUD operations
	CreateSensor(sensor *Sensor) error
	GetSensorByID(id int) (*Sensor, error)
	GetSensorByDeviceID(deviceID string) (*Sensor, error)
	UpdateSensor(id int, req *UpdateSensorRequest) (*Sensor, error)
	DeleteSensor(id int) error
	ListSensors(limit, offset int) ([]*Sensor, int, error)
	ListSensorsByLocation(locationID int) ([]*Sensor, error)

	// Sensor Type operations
	GetSensorTypeByID(id int) (*SensorType, error)
	GetSensorTypeByName(name string) (*SensorType, error)
	ListSensorTypes() ([]*SensorType, error)

	// Location operations
	CreateLocation(location *Location) error
	GetLocationByID(id int) (*Location, error)
	UpdateLocation(id int, req *UpdateLocationRequest) (*Location, error)
	ListLocations() ([]*Location, error)

	// Sensor Reading operations
	CreateSensorReading(reading *SensorReading) error
	CreateBulkSensorReadings(readings []*SensorReading) error
	GetSensorReadings(query *SensorReadingQuery) ([]*SensorReading, int, error)
	GetLatestReading(sensorID int) (*SensorReading, error)
	GetSensorStatistics(sensorID int, startTime, endTime time.Time) (*SensorStatistics, error)

	// Update sensor last reading timestamp
	UpdateSensorLastReading(sensorID int, timestamp time.Time) error
}

// repository implements Repository interface
type repository struct {
	db *sql.DB
}

// NewRepository creates a new sensor repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Schema name constant
const schema = "sensor_data"

// CreateSensor creates a new sensor
func (r *repository) CreateSensor(sensor *Sensor) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.sensors (device_id, name, description, sensor_type_id, location_id, 
		                       is_active, firmware_version, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, schema)

	err := r.db.QueryRow(query,
		sensor.DeviceID, sensor.Name, sensor.Description, sensor.SensorTypeID,
		sensor.LocationID, sensor.IsActive, sensor.FirmwareVersion, sensor.CreatedBy).
		Scan(&sensor.ID, &sensor.CreatedAt, &sensor.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrDeviceIDExists
		}
		return fmt.Errorf("failed to create sensor: %w", err)
	}

	return nil
}

// GetSensorByID retrieves sensor by ID with related data
func (r *repository) GetSensorByID(id int) (*Sensor, error) {
	query := fmt.Sprintf(`
		SELECT s.id, s.device_id, s.name, s.description, s.sensor_type_id, s.location_id,
		       s.is_active, s.last_reading_at, s.battery_level, s.firmware_version,
		       s.created_by, s.created_at, s.updated_at,
		       st.id, st.name, st.description, st.unit, st.min_value, st.max_value,
		       st.is_active, st.created_at, st.updated_at,
		       l.id, l.name, l.description, l.latitude, l.longitude, l.address,
		       l.is_active, l.created_at, l.updated_at
		FROM %s.sensors s
		INNER JOIN %s.sensor_types st ON s.sensor_type_id = st.id
		LEFT JOIN %s.locations l ON s.location_id = l.id
		WHERE s.id = $1
	`, schema, schema, schema)

	sensor := &Sensor{}
	sensorType := &SensorType{}
	location := &Location{}

	var locationID sql.NullInt64
	var lastReadingAt sql.NullTime
	var batteryLevel sql.NullInt64
	var locID sql.NullInt64
	var locName, locDesc, locAddress sql.NullString
	var locLat, locLng sql.NullFloat64
	var locActive sql.NullBool
	var locCreated, locUpdated sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&sensor.ID, &sensor.DeviceID, &sensor.Name, &sensor.Description,
		&sensor.SensorTypeID, &locationID, &sensor.IsActive, &lastReadingAt,
		&batteryLevel, &sensor.FirmwareVersion, &sensor.CreatedBy,
		&sensor.CreatedAt, &sensor.UpdatedAt,
		&sensorType.ID, &sensorType.Name, &sensorType.Description, &sensorType.Unit,
		&sensorType.MinValue, &sensorType.MaxValue, &sensorType.IsActive,
		&sensorType.CreatedAt, &sensorType.UpdatedAt,
		&locID, &locName, &locDesc, &locLat, &locLng, &locAddress,
		&locActive, &locCreated, &locUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSensorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor by ID: %w", err)
	}

	// Set nullable fields
	if locationID.Valid {
		locationIDInt := int(locationID.Int64)
		sensor.LocationID = &locationIDInt
	}
	if lastReadingAt.Valid {
		sensor.LastReadingAt = &lastReadingAt.Time
	}
	if batteryLevel.Valid {
		batteryLevelInt := int(batteryLevel.Int64)
		sensor.BatteryLevel = &batteryLevelInt
	}

	// Set sensor type
	sensor.SensorType = sensorType

	// Set location if exists
	if locID.Valid {
		location.ID = int(locID.Int64)
		location.Name = locName.String
		location.Description = locDesc.String
		if locLat.Valid {
			location.Latitude = &locLat.Float64
		}
		if locLng.Valid {
			location.Longitude = &locLng.Float64
		}
		location.Address = locAddress.String
		location.IsActive = locActive.Bool
		location.CreatedAt = locCreated.Time
		location.UpdatedAt = locUpdated.Time
		sensor.Location = location
	}

	return sensor, nil
}

// GetSensorByDeviceID retrieves sensor by device ID
func (r *repository) GetSensorByDeviceID(deviceID string) (*Sensor, error) {
	query := fmt.Sprintf(`
		SELECT id FROM %s.sensors WHERE device_id = $1
	`, schema)

	var id int
	err := r.db.QueryRow(query, strings.ToUpper(deviceID)).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrSensorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor by device ID: %w", err)
	}

	return r.GetSensorByID(id)
}

// UpdateSensor updates sensor information
func (r *repository) UpdateSensor(id int, req *UpdateSensorRequest) (*Sensor, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.LocationID != nil {
		setParts = append(setParts, fmt.Sprintf("location_id = $%d", argIndex))
		args = append(args, *req.LocationID)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if req.BatteryLevel != nil {
		setParts = append(setParts, fmt.Sprintf("battery_level = $%d", argIndex))
		args = append(args, *req.BatteryLevel)
		argIndex++
	}

	if req.FirmwareVersion != nil {
		setParts = append(setParts, fmt.Sprintf("firmware_version = $%d", argIndex))
		args = append(args, *req.FirmwareVersion)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetSensorByID(id) // No changes, return current sensor
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE %s.sensors 
		SET %s
		WHERE id = $%d AND is_active = true
	`, schema, strings.Join(setParts, ", "), argIndex)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update sensor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrSensorNotFound
	}

	return r.GetSensorByID(id)
}

// DeleteSensor soft deletes a sensor (sets is_active to false)
func (r *repository) DeleteSensor(id int) error {
	query := fmt.Sprintf(`
		UPDATE %s.sensors 
		SET is_active = false, updated_at = $1
		WHERE id = $2
	`, schema)

	result, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete sensor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSensorNotFound
	}

	return nil
}

// ListSensors retrieves paginated list of sensors
func (r *repository) ListSensors(limit, offset int) ([]*Sensor, int, error) {
	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.sensors WHERE is_active = true
	`, schema)
	var total int
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sensors: %w", err)
	}

	// Get sensors with basic info (without joins for performance)
	query := fmt.Sprintf(`
		SELECT s.id, s.device_id, s.name, s.description, s.sensor_type_id, s.location_id,
		       s.is_active, s.last_reading_at, s.battery_level, s.firmware_version,
		       s.created_by, s.created_at, s.updated_at
		FROM %s.sensors s
		WHERE s.is_active = true
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2
	`, schema)

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sensors: %w", err)
	}
	defer rows.Close()

	sensors := []*Sensor{}
	for rows.Next() {
		sensor := &Sensor{}
		var locationID sql.NullInt64
		var lastReadingAt sql.NullTime
		var batteryLevel sql.NullInt64

		err := rows.Scan(
			&sensor.ID, &sensor.DeviceID, &sensor.Name, &sensor.Description,
			&sensor.SensorTypeID, &locationID, &sensor.IsActive, &lastReadingAt,
			&batteryLevel, &sensor.FirmwareVersion, &sensor.CreatedBy,
			&sensor.CreatedAt, &sensor.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan sensor: %w", err)
		}

		// Set nullable fields
		if locationID.Valid {
			locationIDInt := int(locationID.Int64)
			sensor.LocationID = &locationIDInt
		}
		if lastReadingAt.Valid {
			sensor.LastReadingAt = &lastReadingAt.Time
		}
		if batteryLevel.Valid {
			batteryLevelInt := int(batteryLevel.Int64)
			sensor.BatteryLevel = &batteryLevelInt
		}

		sensors = append(sensors, sensor)
	}

	return sensors, total, nil
}

// ListSensorsByLocation retrieves sensors by location
func (r *repository) ListSensorsByLocation(locationID int) ([]*Sensor, error) {
	query := fmt.Sprintf(`
		SELECT id FROM %s.sensors 
		WHERE location_id = $1 AND is_active = true
		ORDER BY name
	`, schema)

	rows, err := r.db.Query(query, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sensors by location: %w", err)
	}
	defer rows.Close()

	sensors := []*Sensor{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan sensor ID: %w", err)
		}

		sensor, err := r.GetSensorByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get sensor details: %w", err)
		}

		sensors = append(sensors, sensor)
	}

	return sensors, nil
}

// GetSensorTypeByID retrieves sensor type by ID
func (r *repository) GetSensorTypeByID(id int) (*SensorType, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, unit, min_value, max_value, is_active, created_at, updated_at
		FROM %s.sensor_types
		WHERE id = $1
	`, schema)

	sensorType := &SensorType{}
	err := r.db.QueryRow(query, id).Scan(
		&sensorType.ID, &sensorType.Name, &sensorType.Description, &sensorType.Unit,
		&sensorType.MinValue, &sensorType.MaxValue, &sensorType.IsActive,
		&sensorType.CreatedAt, &sensorType.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSensorTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor type by ID: %w", err)
	}

	return sensorType, nil
}

// GetSensorTypeByName retrieves sensor type by name
func (r *repository) GetSensorTypeByName(name string) (*SensorType, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, unit, min_value, max_value, is_active, created_at, updated_at
		FROM %s.sensor_types
		WHERE name = $1
	`, schema)

	sensorType := &SensorType{}
	err := r.db.QueryRow(query, name).Scan(
		&sensorType.ID, &sensorType.Name, &sensorType.Description, &sensorType.Unit,
		&sensorType.MinValue, &sensorType.MaxValue, &sensorType.IsActive,
		&sensorType.CreatedAt, &sensorType.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSensorTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor type by name: %w", err)
	}

	return sensorType, nil
}

// ListSensorTypes retrieves all active sensor types
func (r *repository) ListSensorTypes() ([]*SensorType, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, unit, min_value, max_value, is_active, created_at, updated_at
		FROM %s.sensor_types
		WHERE is_active = true
		ORDER BY name
	`, schema)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sensor types: %w", err)
	}
	defer rows.Close()

	sensorTypes := []*SensorType{}
	for rows.Next() {
		sensorType := &SensorType{}
		err := rows.Scan(
			&sensorType.ID, &sensorType.Name, &sensorType.Description, &sensorType.Unit,
			&sensorType.MinValue, &sensorType.MaxValue, &sensorType.IsActive,
			&sensorType.CreatedAt, &sensorType.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sensor type: %w", err)
		}
		sensorTypes = append(sensorTypes, sensorType)
	}

	return sensorTypes, nil
}

// CreateLocation creates a new location
func (r *repository) CreateLocation(location *Location) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.locations (name, description, latitude, longitude, address, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, schema)

	err := r.db.QueryRow(query,
		location.Name, location.Description, location.Latitude, location.Longitude,
		location.Address, location.IsActive).
		Scan(&location.ID, &location.CreatedAt, &location.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create location: %w", err)
	}

	return nil
}

// GetLocationByID retrieves location by ID
func (r *repository) GetLocationByID(id int) (*Location, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, latitude, longitude, address, is_active, created_at, updated_at
		FROM %s.locations
		WHERE id = $1
	`, schema)

	location := &Location{}
	err := r.db.QueryRow(query, id).Scan(
		&location.ID, &location.Name, &location.Description, &location.Latitude,
		&location.Longitude, &location.Address, &location.IsActive,
		&location.CreatedAt, &location.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get location by ID: %w", err)
	}

	return location, nil
}

// UpdateLocation updates location information
func (r *repository) UpdateLocation(id int, req *UpdateLocationRequest) (*Location, error) {
	// Build dynamic query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.Latitude != nil {
		setParts = append(setParts, fmt.Sprintf("latitude = $%d", argIndex))
		args = append(args, *req.Latitude)
		argIndex++
	}

	if req.Longitude != nil {
		setParts = append(setParts, fmt.Sprintf("longitude = $%d", argIndex))
		args = append(args, *req.Longitude)
		argIndex++
	}

	if req.Address != nil {
		setParts = append(setParts, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, *req.Address)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetLocationByID(id) // No changes, return current location
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE %s.locations 
		SET %s
		WHERE id = $%d
	`, schema, strings.Join(setParts, ", "), argIndex)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrLocationNotFound
	}

	return r.GetLocationByID(id)
}

// ListLocations retrieves all active locations
func (r *repository) ListLocations() ([]*Location, error) {
	query := fmt.Sprintf(`
		SELECT id, name, description, latitude, longitude, address, is_active, created_at, updated_at
		FROM %s.locations
		WHERE is_active = true
		ORDER BY name
	`, schema)

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}
	defer rows.Close()

	locations := []*Location{}
	for rows.Next() {
		location := &Location{}
		err := rows.Scan(
			&location.ID, &location.Name, &location.Description, &location.Latitude,
			&location.Longitude, &location.Address, &location.IsActive,
			&location.CreatedAt, &location.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		locations = append(locations, location)
	}

	return locations, nil
}

// CreateSensorReading creates a new sensor reading
func (r *repository) CreateSensorReading(reading *SensorReading) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.sensor_readings (sensor_id, value, timestamp, quality, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, schema)

	timestamp := reading.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	quality := reading.Quality
	if quality == 0 {
		quality = 100 // Default quality
	}

	err := r.db.QueryRow(query,
		reading.SensorID, reading.Value, timestamp, quality, reading.Metadata).
		Scan(&reading.ID, &reading.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create sensor reading: %w", err)
	}

	// Update sensor last reading timestamp
	if err := r.UpdateSensorLastReading(reading.SensorID, timestamp); err != nil {
		// Log warning but don't fail the reading creation
		fmt.Printf("Warning: failed to update sensor last reading: %v\n", err)
	}

	return nil
}

// CreateBulkSensorReadings creates multiple sensor readings in a transaction
func (r *repository) CreateBulkSensorReadings(readings []*SensorReading) error {
	if len(readings) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO %s.sensor_readings (sensor_id, value, timestamp, quality, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, schema)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	sensorLastReadings := make(map[int]time.Time)

	for _, reading := range readings {
		timestamp := reading.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		quality := reading.Quality
		if quality == 0 {
			quality = 100 // Default quality
		}

		err := stmt.QueryRow(
			reading.SensorID, reading.Value, timestamp, quality, reading.Metadata,
		).Scan(&reading.ID, &reading.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to create sensor reading: %w", err)
		}

		// Track latest timestamp per sensor
		if lastTime, exists := sensorLastReadings[reading.SensorID]; !exists || timestamp.After(lastTime) {
			sensorLastReadings[reading.SensorID] = timestamp
		}
	}

	// Update sensor last reading timestamps
	updateQuery := fmt.Sprintf(`
		UPDATE %s.sensors 
		SET last_reading_at = $1, updated_at = $2
		WHERE id = $3
	`, schema)

	updateStmt, err := tx.Prepare(updateQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	now := time.Now()
	for sensorID, lastReading := range sensorLastReadings {
		if _, err := updateStmt.Exec(lastReading, now, sensorID); err != nil {
			return fmt.Errorf("failed to update sensor last reading: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSensorReadings retrieves sensor readings based on query parameters
func (r *repository) GetSensorReadings(query *SensorReadingQuery) ([]*SensorReading, int, error) {
	// Build WHERE clause
	whereParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if query.SensorID != nil {
		whereParts = append(whereParts, fmt.Sprintf("sensor_id = $%d", argIndex))
		args = append(args, *query.SensorID)
		argIndex++
	}

	if query.StartTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *query.StartTime)
		argIndex++
	}

	if query.EndTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *query.EndTime)
		argIndex++
	}

	if query.MinQuality != nil {
		whereParts = append(whereParts, fmt.Sprintf("quality >= $%d", argIndex))
		args = append(args, *query.MinQuality)
		argIndex++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.sensor_readings %s
	`, schema, whereClause)

	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sensor readings: %w", err)
	}

	// Get readings
	limit := query.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	// Add limit and offset to args
	args = append(args, limit, offset)

	readingsQuery := fmt.Sprintf(`
		SELECT id, sensor_id, value, timestamp, quality, metadata, created_at
		FROM %s.sensor_readings
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, schema, whereClause, argIndex, argIndex+1)

	rows, err := r.db.Query(readingsQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get sensor readings: %w", err)
	}
	defer rows.Close()

	readings := []*SensorReading{}
	for rows.Next() {
		reading := &SensorReading{}
		err := rows.Scan(
			&reading.ID, &reading.SensorID, &reading.Value, &reading.Timestamp,
			&reading.Quality, &reading.Metadata, &reading.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan sensor reading: %w", err)
		}
		readings = append(readings, reading)
	}

	return readings, total, nil
}

// GetLatestReading retrieves the latest reading for a sensor
func (r *repository) GetLatestReading(sensorID int) (*SensorReading, error) {
	query := fmt.Sprintf(`
		SELECT id, sensor_id, value, timestamp, quality, metadata, created_at
		FROM %s.sensor_readings
		WHERE sensor_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`, schema)

	reading := &SensorReading{}
	err := r.db.QueryRow(query, sensorID).Scan(
		&reading.ID, &reading.SensorID, &reading.Value, &reading.Timestamp,
		&reading.Quality, &reading.Metadata, &reading.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No readings yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest reading: %w", err)
	}

	return reading, nil
}

// GetSensorStatistics calculates statistics for a sensor within time range
func (r *repository) GetSensorStatistics(sensorID int, startTime, endTime time.Time) (*SensorStatistics, error) {
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as count,
			MIN(value) as min_value,
			MAX(value) as max_value,
			AVG(value) as avg_value,
			(SELECT value FROM %s.sensor_readings WHERE sensor_id = $1 ORDER BY timestamp DESC LIMIT 1) as last_value,
			(SELECT timestamp FROM %s.sensor_readings WHERE sensor_id = $1 ORDER BY timestamp DESC LIMIT 1) as last_timestamp
		FROM %s.sensor_readings
		WHERE sensor_id = $1 AND timestamp >= $2 AND timestamp <= $3
	`, schema, schema, schema)

	stats := &SensorStatistics{
		SensorID: sensorID,
		Period:   fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")),
	}

	var lastTimestamp sql.NullTime

	err := r.db.QueryRow(query, sensorID, startTime, endTime).Scan(
		&stats.Count, &stats.MinValue, &stats.MaxValue, &stats.AvgValue,
		&stats.LastValue, &lastTimestamp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get sensor statistics: %w", err)
	}

	if lastTimestamp.Valid {
		stats.LastTimestamp = &lastTimestamp.Time
	}

	return stats, nil
}

// UpdateSensorLastReading updates sensor's last reading timestamp
func (r *repository) UpdateSensorLastReading(sensorID int, timestamp time.Time) error {
	query := fmt.Sprintf(`
		UPDATE %s.sensors 
		SET last_reading_at = $1, updated_at = $2
		WHERE id = $3
	`, schema)

	_, err := r.db.Exec(query, timestamp, time.Now(), sensorID)
	if err != nil {
		return fmt.Errorf("failed to update sensor last reading: %w", err)
	}

	return nil
}

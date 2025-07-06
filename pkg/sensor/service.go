package sensor

import (
	"fmt"
	"log"
	"time"
)

// Service defines sensor service interface
type Service interface {
	// Sensor management
	CreateSensor(req *CreateSensorRequest, createdBy int) (*Sensor, error)
	GetSensor(id int) (*Sensor, error)
	GetSensorByDeviceID(deviceID string) (*Sensor, error)
	UpdateSensor(id int, req *UpdateSensorRequest) (*Sensor, error)
	DeleteSensor(id int) error
	ListSensors(page, perPage int) ([]*Sensor, int, error)
	ListSensorsByLocation(locationID int) ([]*Sensor, error)

	// Sensor types
	GetSensorType(id int) (*SensorType, error)
	GetSensorTypeByName(name string) (*SensorType, error)
	ListSensorTypes() ([]*SensorType, error)

	// Location management
	CreateLocation(req *CreateLocationRequest) (*Location, error)
	GetLocation(id int) (*Location, error)
	UpdateLocation(id int, req *UpdateLocationRequest) (*Location, error)
	ListLocations() ([]*Location, error)

	// Sensor readings
	CreateSensorReading(req *CreateSensorReadingRequest) (*SensorReading, error)
	CreateBulkSensorReadings(req *BulkSensorReadingRequest) error
	GetSensorReadings(query *SensorReadingQuery) ([]*SensorReading, int, error)
	GetLatestReading(sensorID int) (*SensorReading, error)
	GetSensorStatistics(sensorID int, startTime, endTime time.Time) (*SensorStatistics, error)

	// Dashboard & Analytics
	GetSensorsDashboard() (*DashboardData, error)
	GetSensorHealth() ([]*SensorHealthStatus, error)
	GetLocationSummary(locationID int) (*LocationSummary, error)
}

// service implements Service interface
type service struct {
	repo Repository
}

// NewService creates a new sensor service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// DashboardData represents sensor dashboard data
type DashboardData struct {
	TotalSensors   int                   `json:"total_sensors"`
	ActiveSensors  int                   `json:"active_sensors"`
	OnlineSensors  int                   `json:"online_sensors"`
	OfflineSensors int                   `json:"offline_sensors"`
	SensorsByType  map[string]int        `json:"sensors_by_type"`
	RecentReadings []*SensorReading      `json:"recent_readings"`
	AlertSensors   []*SensorHealthStatus `json:"alert_sensors"`
}

// SensorHealthStatus represents sensor health information
type SensorHealthStatus struct {
	Sensor        *Sensor        `json:"sensor"`
	IsOnline      bool           `json:"is_online"`
	BatteryStatus string         `json:"battery_status"`
	LastReading   *SensorReading `json:"last_reading,omitempty"`
	HealthScore   int            `json:"health_score"` // 0-100
	Issues        []string       `json:"issues,omitempty"`
}

// LocationSummary represents location summary data
type LocationSummary struct {
	Location       *Location        `json:"location"`
	SensorCount    int              `json:"sensor_count"`
	ActiveSensors  int              `json:"active_sensors"`
	OnlineSensors  int              `json:"online_sensors"`
	Sensors        []*Sensor        `json:"sensors"`
	LatestReadings []*SensorReading `json:"latest_readings"`
}

// CreateSensor creates a new sensor with validation
func (s *service) CreateSensor(req *CreateSensorRequest, createdBy int) (*Sensor, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if device ID already exists
	existingSensor, err := s.repo.GetSensorByDeviceID(req.DeviceID)
	if err != nil && err != ErrSensorNotFound {
		return nil, fmt.Errorf("failed to check existing sensor: %w", err)
	}
	if existingSensor != nil {
		return nil, ErrDeviceIDExists
	}

	// Validate sensor type exists
	sensorType, err := s.repo.GetSensorTypeByID(req.SensorTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid sensor type: %w", err)
	}
	if !sensorType.IsActive {
		return nil, fmt.Errorf("sensor type is inactive")
	}

	// Validate location if provided
	if req.LocationID != nil {
		location, err := s.repo.GetLocationByID(*req.LocationID)
		if err != nil {
			return nil, fmt.Errorf("invalid location: %w", err)
		}
		if !location.IsActive {
			return nil, fmt.Errorf("location is inactive")
		}
	}

	// Create sensor
	sensor, err := NewSensor(req, createdBy)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateSensor(sensor); err != nil {
		return nil, fmt.Errorf("failed to create sensor: %w", err)
	}

	// Load with related data
	return s.repo.GetSensorByID(sensor.ID)
}

// GetSensor retrieves sensor by ID with related data
func (s *service) GetSensor(id int) (*Sensor, error) {
	sensor, err := s.repo.GetSensorByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor: %w", err)
	}

	// Load latest reading
	latestReading, err := s.repo.GetLatestReading(sensor.ID)
	if err != nil {
		log.Printf("Warning: failed to get latest reading for sensor %d: %v", sensor.ID, err)
	} else if latestReading != nil {
		sensor.LatestReading = latestReading
	}

	return sensor, nil
}

// GetSensorByDeviceID retrieves sensor by device ID
func (s *service) GetSensorByDeviceID(deviceID string) (*Sensor, error) {
	sensor, err := s.repo.GetSensorByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor by device ID: %w", err)
	}

	// Load latest reading
	latestReading, err := s.repo.GetLatestReading(sensor.ID)
	if err != nil {
		log.Printf("Warning: failed to get latest reading for sensor %d: %v", sensor.ID, err)
	} else if latestReading != nil {
		sensor.LatestReading = latestReading
	}

	return sensor, nil
}

// UpdateSensor updates sensor information
func (s *service) UpdateSensor(id int, req *UpdateSensorRequest) (*Sensor, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if sensor exists (we don't need the result, just check existence)
	_, err := s.repo.GetSensorByID(id)
	if err != nil {
		return nil, fmt.Errorf("sensor not found: %w", err)
	}

	// Validate location if being updated
	if req.LocationID != nil {
		location, err := s.repo.GetLocationByID(*req.LocationID)
		if err != nil {
			return nil, fmt.Errorf("invalid location: %w", err)
		}
		if !location.IsActive {
			return nil, fmt.Errorf("location is inactive")
		}
	}

	// Update sensor
	updatedSensor, err := s.repo.UpdateSensor(id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update sensor: %w", err)
	}

	return updatedSensor, nil
}

// DeleteSensor deactivates a sensor
func (s *service) DeleteSensor(id int) error {
	if err := s.repo.DeleteSensor(id); err != nil {
		return fmt.Errorf("failed to delete sensor: %w", err)
	}

	return nil
}

// ListSensors returns paginated list of sensors
func (s *service) ListSensors(page, perPage int) ([]*Sensor, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	sensors, total, err := s.repo.ListSensors(perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sensors: %w", err)
	}

	// Load sensor types and latest readings for each sensor
	for _, sensor := range sensors {
		// Load sensor type
		if sensorType, err := s.repo.GetSensorTypeByID(sensor.SensorTypeID); err == nil {
			sensor.SensorType = sensorType
		}

		// Load location if exists
		if sensor.LocationID != nil {
			if location, err := s.repo.GetLocationByID(*sensor.LocationID); err == nil {
				sensor.Location = location
			}
		}

		// Load latest reading
		if latestReading, err := s.repo.GetLatestReading(sensor.ID); err == nil && latestReading != nil {
			sensor.LatestReading = latestReading
		}
	}

	return sensors, total, nil
}

// ListSensorsByLocation returns sensors by location
func (s *service) ListSensorsByLocation(locationID int) ([]*Sensor, error) {
	// Validate location exists
	_, err := s.repo.GetLocationByID(locationID)
	if err != nil {
		return nil, fmt.Errorf("location not found: %w", err)
	}

	sensors, err := s.repo.ListSensorsByLocation(locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sensors by location: %w", err)
	}

	return sensors, nil
}

// GetSensorType retrieves sensor type by ID
func (s *service) GetSensorType(id int) (*SensorType, error) {
	sensorType, err := s.repo.GetSensorTypeByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor type: %w", err)
	}

	return sensorType, nil
}

// GetSensorTypeByName retrieves sensor type by name
func (s *service) GetSensorTypeByName(name string) (*SensorType, error) {
	sensorType, err := s.repo.GetSensorTypeByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor type by name: %w", err)
	}

	return sensorType, nil
}

// ListSensorTypes returns all active sensor types
func (s *service) ListSensorTypes() ([]*SensorType, error) {
	sensorTypes, err := s.repo.ListSensorTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to list sensor types: %w", err)
	}

	return sensorTypes, nil
}

// CreateLocation creates a new location
func (s *service) CreateLocation(req *CreateLocationRequest) (*Location, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create location
	location, err := NewLocation(req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateLocation(location); err != nil {
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	return location, nil
}

// GetLocation retrieves location by ID
func (s *service) GetLocation(id int) (*Location, error) {
	location, err := s.repo.GetLocationByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	return location, nil
}

// UpdateLocation updates location information
func (s *service) UpdateLocation(id int, req *UpdateLocationRequest) (*Location, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Update location
	updatedLocation, err := s.repo.UpdateLocation(id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	return updatedLocation, nil
}

// ListLocations returns all active locations
func (s *service) ListLocations() ([]*Location, error) {
	locations, err := s.repo.ListLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	return locations, nil
}

// CreateSensorReading creates a new sensor reading with validation
func (s *service) CreateSensorReading(req *CreateSensorReadingRequest) (*SensorReading, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get sensor and validate
	sensor, err := s.repo.GetSensorByID(req.SensorID)
	if err != nil {
		return nil, fmt.Errorf("sensor not found: %w", err)
	}

	if !sensor.IsActive {
		return nil, ErrSensorInactive
	}

	// Validate value against sensor type constraints
	if err := sensor.ValidateValue(req.Value); err != nil {
		return nil, err
	}

	// Create reading
	reading := &SensorReading{
		SensorID:  req.SensorID,
		Value:     req.Value,
		Timestamp: time.Now(),
		Quality:   100,
	}

	if req.Timestamp != nil {
		reading.Timestamp = *req.Timestamp
	}

	if req.Quality != nil {
		reading.Quality = *req.Quality
	}

	if req.Metadata != nil {
		reading.Metadata = req.Metadata
	}

	if err := s.repo.CreateSensorReading(reading); err != nil {
		return nil, fmt.Errorf("failed to create sensor reading: %w", err)
	}

	return reading, nil
}

// CreateBulkSensorReadings creates multiple sensor readings
func (s *service) CreateBulkSensorReadings(req *BulkSensorReadingRequest) error {
	if len(req.Readings) == 0 {
		return fmt.Errorf("no readings provided")
	}

	if len(req.Readings) > 1000 {
		return fmt.Errorf("too many readings, maximum 1000 per batch")
	}

	// Validate all readings and convert to SensorReading
	readings := make([]*SensorReading, len(req.Readings))
	sensorCache := make(map[int]*Sensor)

	for i, readingReq := range req.Readings {
		// Validate reading request
		if err := readingReq.Validate(); err != nil {
			return fmt.Errorf("reading %d: %w", i+1, err)
		}

		// Get sensor (with caching)
		sensor, exists := sensorCache[readingReq.SensorID]
		if !exists {
			var err error
			sensor, err = s.repo.GetSensorByID(readingReq.SensorID)
			if err != nil {
				return fmt.Errorf("reading %d: sensor not found: %w", i+1, err)
			}
			sensorCache[readingReq.SensorID] = sensor
		}

		if !sensor.IsActive {
			return fmt.Errorf("reading %d: sensor is inactive", i+1)
		}

		// Validate value
		if err := sensor.ValidateValue(readingReq.Value); err != nil {
			return fmt.Errorf("reading %d: %w", i+1, err)
		}

		// Create reading
		reading := &SensorReading{
			SensorID:  readingReq.SensorID,
			Value:     readingReq.Value,
			Timestamp: time.Now(),
			Quality:   100,
		}

		if readingReq.Timestamp != nil {
			reading.Timestamp = *readingReq.Timestamp
		}

		if readingReq.Quality != nil {
			reading.Quality = *readingReq.Quality
		}

		if readingReq.Metadata != nil {
			reading.Metadata = readingReq.Metadata
		}

		readings[i] = reading
	}

	// Create all readings in bulk
	if err := s.repo.CreateBulkSensorReadings(readings); err != nil {
		return fmt.Errorf("failed to create bulk sensor readings: %w", err)
	}

	return nil
}

// GetSensorReadings retrieves sensor readings with filters
func (s *service) GetSensorReadings(query *SensorReadingQuery) ([]*SensorReading, int, error) {
	// Set default limits
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.Limit > 1000 {
		query.Limit = 1000
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// Validate sensor if specified
	if query.SensorID != nil {
		_, err := s.repo.GetSensorByID(*query.SensorID)
		if err != nil {
			return nil, 0, fmt.Errorf("sensor not found: %w", err)
		}
	}

	readings, total, err := s.repo.GetSensorReadings(query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get sensor readings: %w", err)
	}

	return readings, total, nil
}

// GetLatestReading retrieves latest reading for a sensor
func (s *service) GetLatestReading(sensorID int) (*SensorReading, error) {
	// Validate sensor exists
	_, err := s.repo.GetSensorByID(sensorID)
	if err != nil {
		return nil, fmt.Errorf("sensor not found: %w", err)
	}

	reading, err := s.repo.GetLatestReading(sensorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest reading: %w", err)
	}

	return reading, nil
}

// GetSensorStatistics calculates statistics for a sensor
func (s *service) GetSensorStatistics(sensorID int, startTime, endTime time.Time) (*SensorStatistics, error) {
	// Validate sensor exists
	_, err := s.repo.GetSensorByID(sensorID)
	if err != nil {
		return nil, fmt.Errorf("sensor not found: %w", err)
	}

	// Validate time range
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	stats, err := s.repo.GetSensorStatistics(sensorID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor statistics: %w", err)
	}

	return stats, nil
}

// GetSensorsDashboard returns dashboard data with sensor overview
func (s *service) GetSensorsDashboard() (*DashboardData, error) {
	// Get all sensors for counting
	sensors, _, err := s.repo.ListSensors(1000, 0) // Get up to 1000 sensors for dashboard
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors for dashboard: %w", err)
	}

	dashboard := &DashboardData{
		TotalSensors:   len(sensors),
		SensorsByType:  make(map[string]int),
		RecentReadings: []*SensorReading{},
		AlertSensors:   []*SensorHealthStatus{},
	}

	onlineThreshold := 30 // 30 minutes

	// Process each sensor
	for _, sensor := range sensors {
		if sensor.IsActive {
			dashboard.ActiveSensors++
		}

		// Check if sensor is online
		if sensor.IsOnline(onlineThreshold) {
			dashboard.OnlineSensors++
		} else {
			dashboard.OfflineSensors++
		}

		// Count by sensor type
		if sensor.SensorType != nil {
			dashboard.SensorsByType[sensor.SensorType.Name]++
		}

		// Check for alerts
		healthStatus := s.calculateSensorHealth(sensor)
		if healthStatus.HealthScore < 80 || len(healthStatus.Issues) > 0 {
			dashboard.AlertSensors = append(dashboard.AlertSensors, healthStatus)
		}
	}

	// Get recent readings (last 50)
	recentQuery := &SensorReadingQuery{
		Limit:  50,
		Offset: 0,
	}
	recentReadings, _, err := s.repo.GetSensorReadings(recentQuery)
	if err != nil {
		log.Printf("Warning: failed to get recent readings for dashboard: %v", err)
	} else {
		dashboard.RecentReadings = recentReadings
	}

	return dashboard, nil
}

// GetSensorHealth returns health status for all sensors
func (s *service) GetSensorHealth() ([]*SensorHealthStatus, error) {
	sensors, _, err := s.repo.ListSensors(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors for health check: %w", err)
	}

	healthStatuses := make([]*SensorHealthStatus, len(sensors))

	for i, sensor := range sensors {
		healthStatuses[i] = s.calculateSensorHealth(sensor)
	}

	return healthStatuses, nil
}

// GetLocationSummary returns summary data for a location
func (s *service) GetLocationSummary(locationID int) (*LocationSummary, error) {
	// Get location
	location, err := s.repo.GetLocationByID(locationID)
	if err != nil {
		return nil, fmt.Errorf("location not found: %w", err)
	}

	// Get sensors in this location
	sensors, err := s.repo.ListSensorsByLocation(locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors for location: %w", err)
	}

	summary := &LocationSummary{
		Location:       location,
		SensorCount:    len(sensors),
		Sensors:        sensors,
		LatestReadings: []*SensorReading{},
	}

	onlineThreshold := 30 // 30 minutes

	// Process sensors
	for _, sensor := range sensors {
		if sensor.IsActive {
			summary.ActiveSensors++
		}

		if sensor.IsOnline(onlineThreshold) {
			summary.OnlineSensors++
		}

		// Get latest reading for each sensor
		if latestReading, err := s.repo.GetLatestReading(sensor.ID); err == nil && latestReading != nil {
			summary.LatestReadings = append(summary.LatestReadings, latestReading)
		}
	}

	return summary, nil
}

// calculateSensorHealth calculates health score and issues for a sensor
func (s *service) calculateSensorHealth(sensor *Sensor) *SensorHealthStatus {
	status := &SensorHealthStatus{
		Sensor:        sensor,
		IsOnline:      sensor.IsOnline(30), // 30 minutes threshold
		BatteryStatus: sensor.GetBatteryStatus(),
		HealthScore:   100,
		Issues:        []string{},
	}

	// Get latest reading
	if latestReading, err := s.repo.GetLatestReading(sensor.ID); err == nil && latestReading != nil {
		status.LastReading = latestReading
	}

	// Check various health factors

	// 1. Online status
	if !status.IsOnline {
		status.HealthScore -= 30
		status.Issues = append(status.Issues, "Sensor offline")
	}

	// 2. Battery level
	if sensor.BatteryLevel != nil {
		switch {
		case *sensor.BatteryLevel < 20:
			status.HealthScore -= 25
			status.Issues = append(status.Issues, "Critical battery level")
		case *sensor.BatteryLevel < 50:
			status.HealthScore -= 10
			status.Issues = append(status.Issues, "Low battery level")
		}
	}

	// 3. Reading quality
	if status.LastReading != nil {
		if status.LastReading.Quality < 80 {
			status.HealthScore -= 15
			status.Issues = append(status.Issues, "Poor reading quality")
		}
	}

	// 4. No recent readings
	if sensor.LastReadingAt == nil {
		status.HealthScore -= 20
		status.Issues = append(status.Issues, "No readings recorded")
	} else {
		// Check if reading is too old
		lastReadingAge := time.Since(*sensor.LastReadingAt)
		if lastReadingAge > 2*time.Hour {
			status.HealthScore -= 15
			status.Issues = append(status.Issues, "Readings too old")
		}
	}

	// 5. Sensor inactive
	if !sensor.IsActive {
		status.HealthScore = 0
		status.Issues = append(status.Issues, "Sensor inactive")
	}

	// Ensure health score doesn't go below 0
	if status.HealthScore < 0 {
		status.HealthScore = 0
	}

	return status
}

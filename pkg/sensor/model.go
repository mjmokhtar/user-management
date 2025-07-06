package sensor

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Sensor represents an IoT sensor device
type Sensor struct {
	ID              int            `json:"id"`
	DeviceID        string         `json:"device_id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	SensorTypeID    int            `json:"sensor_type_id"`
	LocationID      *int           `json:"location_id,omitempty"`
	IsActive        bool           `json:"is_active"`
	LastReadingAt   *time.Time     `json:"last_reading_at,omitempty"`
	BatteryLevel    *int           `json:"battery_level,omitempty"`
	FirmwareVersion string         `json:"firmware_version"`
	CreatedBy       int            `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	SensorType      *SensorType    `json:"sensor_type,omitempty"`
	Location        *Location      `json:"location,omitempty"`
	LatestReading   *SensorReading `json:"latest_reading,omitempty"`
}

// SensorType represents a type of sensor
type SensorType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Unit        string    `json:"unit"`
	MinValue    *float64  `json:"min_value,omitempty"`
	MaxValue    *float64  `json:"max_value,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Location represents a physical location
type Location struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Latitude    *float64  `json:"latitude,omitempty"`
	Longitude   *float64  `json:"longitude,omitempty"`
	Address     string    `json:"address"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SensorReading represents a sensor data reading
type SensorReading struct {
	ID        int64           `json:"id"`
	SensorID  int             `json:"sensor_id"`
	Value     float64         `json:"value"`
	Timestamp time.Time       `json:"timestamp"`
	Quality   int             `json:"quality"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// CreateSensorRequest represents request to create sensor
type CreateSensorRequest struct {
	DeviceID        string `json:"device_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	SensorTypeID    int    `json:"sensor_type_id"`
	LocationID      *int   `json:"location_id,omitempty"`
	FirmwareVersion string `json:"firmware_version"`
}

// UpdateSensorRequest represents request to update sensor
type UpdateSensorRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	LocationID      *int    `json:"location_id,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	BatteryLevel    *int    `json:"battery_level,omitempty"`
	FirmwareVersion *string `json:"firmware_version,omitempty"`
}

// CreateSensorReadingRequest represents request to create sensor reading
type CreateSensorReadingRequest struct {
	SensorID  int             `json:"sensor_id"`
	Value     float64         `json:"value"`
	Timestamp *time.Time      `json:"timestamp,omitempty"`
	Quality   *int            `json:"quality,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// BulkSensorReadingRequest represents bulk reading request
type BulkSensorReadingRequest struct {
	Readings []CreateSensorReadingRequest `json:"readings"`
}

// SensorReadingQuery represents query parameters for sensor readings
type SensorReadingQuery struct {
	SensorID   *int       `json:"sensor_id,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
	MinQuality *int       `json:"min_quality,omitempty"`
}

// SensorStatistics represents sensor data statistics
type SensorStatistics struct {
	SensorID      int        `json:"sensor_id"`
	Count         int64      `json:"count"`
	MinValue      *float64   `json:"min_value"`
	MaxValue      *float64   `json:"max_value"`
	AvgValue      *float64   `json:"avg_value"`
	LastValue     *float64   `json:"last_value"`
	LastTimestamp *time.Time `json:"last_timestamp"`
	Period        string     `json:"period"`
}

// CreateLocationRequest represents request to create location
type CreateLocationRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Address     string   `json:"address"`
}

// UpdateLocationRequest represents request to update location
type UpdateLocationRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Address     *string  `json:"address,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// Domain errors
var (
	ErrInvalidDeviceID    = errors.New("invalid device ID format")
	ErrDeviceIDExists     = errors.New("device ID already exists")
	ErrSensorNotFound     = errors.New("sensor not found")
	ErrSensorTypeNotFound = errors.New("sensor type not found")
	ErrLocationNotFound   = errors.New("location not found")
	ErrInvalidValue       = errors.New("sensor value out of range")
	ErrInvalidQuality     = errors.New("quality must be between 0 and 100")
	ErrInvalidBattery     = errors.New("battery level must be between 0 and 100")
	ErrSensorInactive     = errors.New("sensor is inactive")
)

// Validate validates CreateSensorRequest
func (req *CreateSensorRequest) Validate() error {
	// Validate device ID
	if err := validateDeviceID(req.DeviceID); err != nil {
		return err
	}

	// Validate name
	if err := validateName(req.Name); err != nil {
		return err
	}

	// Validate sensor type ID
	if req.SensorTypeID <= 0 {
		return errors.New("sensor type ID is required")
	}

	return nil
}

// Validate validates UpdateSensorRequest
func (req *UpdateSensorRequest) Validate() error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return errors.New("name cannot be empty")
	}

	if req.BatteryLevel != nil && (*req.BatteryLevel < 0 || *req.BatteryLevel > 100) {
		return ErrInvalidBattery
	}

	return nil
}

// Validate validates CreateSensorReadingRequest
func (req *CreateSensorReadingRequest) Validate() error {
	if req.SensorID <= 0 {
		return errors.New("sensor ID is required")
	}

	if req.Quality != nil && (*req.Quality < 0 || *req.Quality > 100) {
		return ErrInvalidQuality
	}

	return nil
}

// Validate validates CreateLocationRequest
func (req *CreateLocationRequest) Validate() error {
	if err := validateName(req.Name); err != nil {
		return err
	}

	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return errors.New("latitude must be between -90 and 90")
	}

	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return errors.New("longitude must be between -180 and 180")
	}

	return nil
}

// Validate validates UpdateLocationRequest
func (req *UpdateLocationRequest) Validate() error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return errors.New("name cannot be empty")
	}

	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return errors.New("latitude must be between -90 and 90")
	}

	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return errors.New("longitude must be between -180 and 180")
	}

	if req.Address != nil && len(strings.TrimSpace(*req.Address)) > 500 {
		return errors.New("address must be less than 500 characters")
	}

	return nil
}

// ValidateValue validates sensor reading value against sensor type constraints
func (s *Sensor) ValidateValue(value float64) error {
	if s.SensorType == nil {
		return nil // Cannot validate without sensor type info
	}

	if s.SensorType.MinValue != nil && value < *s.SensorType.MinValue {
		return ErrInvalidValue
	}

	if s.SensorType.MaxValue != nil && value > *s.SensorType.MaxValue {
		return ErrInvalidValue
	}

	return nil
}

// IsOnline checks if sensor is considered online (has recent readings)
func (s *Sensor) IsOnline(thresholdMinutes int) bool {
	if s.LastReadingAt == nil {
		return false
	}

	threshold := time.Now().Add(-time.Duration(thresholdMinutes) * time.Minute)
	return s.LastReadingAt.After(threshold)
}

// GetBatteryStatus returns battery status description
func (s *Sensor) GetBatteryStatus() string {
	if s.BatteryLevel == nil {
		return "unknown"
	}

	switch {
	case *s.BatteryLevel >= 80:
		return "good"
	case *s.BatteryLevel >= 50:
		return "medium"
	case *s.BatteryLevel >= 20:
		return "low"
	default:
		return "critical"
	}
}

// NewSensor creates a new sensor with validation
func NewSensor(req *CreateSensorRequest, createdBy int) (*Sensor, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	sensor := &Sensor{
		DeviceID:        strings.ToUpper(strings.TrimSpace(req.DeviceID)),
		Name:            strings.TrimSpace(req.Name),
		Description:     strings.TrimSpace(req.Description),
		SensorTypeID:    req.SensorTypeID,
		LocationID:      req.LocationID,
		IsActive:        true,
		FirmwareVersion: strings.TrimSpace(req.FirmwareVersion),
		CreatedBy:       createdBy,
	}

	return sensor, nil
}

// NewLocation creates a new location with validation
func NewLocation(req *CreateLocationRequest) (*Location, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	location := &Location{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Address:     strings.TrimSpace(req.Address),
		IsActive:    true,
	}

	return location, nil
}

// Helper validation functions
func validateDeviceID(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device ID is required")
	}

	// Device ID format: alphanumeric, hyphens, underscores (3-50 chars)
	deviceIDRegex := regexp.MustCompile(`^[A-Za-z0-9_-]{3,50}$`)
	if !deviceIDRegex.MatchString(deviceID) {
		return ErrInvalidDeviceID
	}

	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) < 2 {
		return errors.New("name must be at least 2 characters long")
	}
	if len(name) > 255 {
		return errors.New("name must be less than 255 characters")
	}
	return nil
}

// FormatValue formats sensor value with appropriate precision based on type
func (st *SensorType) FormatValue(value float64) string {
	switch st.Name {
	case "temperature":
		return fmt.Sprintf("%.1f %s", value, st.Unit)
	case "humidity":
		return fmt.Sprintf("%.0f %s", value, st.Unit)
	case "pressure":
		return fmt.Sprintf("%.1f %s", value, st.Unit)
	case "motion":
		if value > 0 {
			return "Motion detected"
		}
		return "No motion"
	default:
		return fmt.Sprintf("%.2f %s", value, st.Unit)
	}
}

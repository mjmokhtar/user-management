package sensor

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"user-management/shared/middleware"
	"user-management/shared/response"
)

// Handler handles HTTP requests for sensor operations
type Handler struct {
	service Service
	authMW  *middleware.AuthMiddleware
}

// NewHandler creates a new sensor handler
func NewHandler(service Service, authMW *middleware.AuthMiddleware) *Handler {
	return &Handler{
		service: service,
		authMW:  authMW,
	}
}

// RegisterRoutes registers all sensor routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public routes (for IoT devices to send data)
	mux.HandleFunc("POST /api/sensors/readings", h.CreateSensorReading)
	mux.HandleFunc("POST /api/sensors/readings/bulk", h.CreateBulkSensorReadings)

	// Protected routes (authentication required)
	mux.Handle("GET /api/sensors/dashboard", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetDashboard)))
	mux.Handle("GET /api/sensors", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.ListSensors)))
	mux.Handle("GET /api/sensors/{id}", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetSensor)))
	mux.Handle("GET /api/sensors/device/{device_id}", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetSensorByDeviceID)))
	mux.Handle("GET /api/sensors/readings", h.authMW.RequirePermission("sensor_readings", "read")(http.HandlerFunc(h.GetSensorReadings)))
	mux.Handle("GET /api/sensors/health", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetSensorHealth)))

	// Sensor management (write permissions)
	mux.Handle("POST /api/sensors", h.authMW.RequirePermission("sensors", "write")(http.HandlerFunc(h.CreateSensor)))
	mux.Handle("PUT /api/sensors/{id}", h.authMW.RequirePermission("sensors", "write")(http.HandlerFunc(h.UpdateSensor)))
	mux.Handle("DELETE /api/sensors/{id}", h.authMW.RequirePermission("sensors", "delete")(http.HandlerFunc(h.DeleteSensor)))

	// Sensor types (read-only for most users)
	mux.Handle("GET /api/sensor-types", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.ListSensorTypes)))
	mux.Handle("GET /api/sensor-types/{id}", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetSensorType)))

	// Location management
	mux.Handle("GET /api/locations", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.ListLocations)))
	mux.Handle("GET /api/locations/{id}", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetLocation)))
	mux.Handle("GET /api/locations/sensors", h.authMW.RequirePermission("sensors", "read")(http.HandlerFunc(h.GetLocationSummary)))
	mux.Handle("POST /api/locations", h.authMW.RequirePermission("sensors", "write")(http.HandlerFunc(h.CreateLocation)))
	mux.Handle("PUT /api/locations/{id}", h.authMW.RequirePermission("sensors", "write")(http.HandlerFunc(h.UpdateLocation)))

	// Analytics & Statistics
	mux.Handle("GET /api/sensors/statistics", h.authMW.RequirePermission("analytics", "read")(http.HandlerFunc(h.GetSensorStatistics)))
}

// CreateSensor handles sensor creation
func (h *Handler) CreateSensor(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	var req CreateSensorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	sensor, err := h.service.CreateSensor(&req, user.ID)
	if err != nil {
		switch err {
		case ErrInvalidDeviceID, ErrInvalidValue:
			response.BadRequest(w, "Validation failed", err)
		case ErrDeviceIDExists:
			response.Conflict(w, "Device ID already exists", err)
		case ErrSensorTypeNotFound, ErrLocationNotFound:
			response.NotFound(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to create sensor", err)
		}
		return
	}

	response.Created(w, "Sensor created successfully", sensor)
}

// GetSensor handles getting sensor by ID
func (h *Handler) GetSensor(w http.ResponseWriter, r *http.Request) {
	sensorID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid sensor ID", err)
		return
	}

	sensor, err := h.service.GetSensor(sensorID)
	if err != nil {
		switch err {
		case ErrSensorNotFound:
			response.NotFound(w, "Sensor not found")
		default:
			response.InternalServerError(w, "Failed to get sensor", err)
		}
		return
	}

	response.Success(w, "Sensor retrieved successfully", sensor)
}

// GetSensorByDeviceID handles getting sensor by device ID
func (h *Handler) GetSensorByDeviceID(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		response.BadRequest(w, "Invalid device ID", nil)
		return
	}

	sensor, err := h.service.GetSensorByDeviceID(deviceID)
	if err != nil {
		switch err {
		case ErrSensorNotFound:
			response.NotFound(w, "Sensor not found")
		default:
			response.InternalServerError(w, "Failed to get sensor", err)
		}
		return
	}

	response.Success(w, "Sensor retrieved successfully", sensor)
}

// UpdateSensor handles sensor updates
func (h *Handler) UpdateSensor(w http.ResponseWriter, r *http.Request) {
	sensorID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid sensor ID", err)
		return
	}

	var req UpdateSensorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	sensor, err := h.service.UpdateSensor(sensorID, &req)
	if err != nil {
		switch err {
		case ErrInvalidBattery:
			response.BadRequest(w, "Validation failed", err)
		case ErrSensorNotFound, ErrLocationNotFound:
			response.NotFound(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to update sensor", err)
		}
		return
	}

	response.Success(w, "Sensor updated successfully", sensor)
}

// DeleteSensor handles sensor deletion
func (h *Handler) DeleteSensor(w http.ResponseWriter, r *http.Request) {
	sensorID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid sensor ID", err)
		return
	}

	if err := h.service.DeleteSensor(sensorID); err != nil {
		switch err {
		case ErrSensorNotFound:
			response.NotFound(w, "Sensor not found")
		default:
			response.InternalServerError(w, "Failed to delete sensor", err)
		}
		return
	}

	response.Success(w, "Sensor deleted successfully", nil)
}

// ListSensors handles listing sensors with pagination
func (h *Handler) ListSensors(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	sensors, total, err := h.service.ListSensors(page, perPage)
	if err != nil {
		response.InternalServerError(w, "Failed to list sensors", err)
		return
	}

	// Calculate pagination meta
	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.PaginatedSuccess(w, "Sensors retrieved successfully", sensors, meta)
}

// CreateSensorReading handles single sensor reading creation
func (h *Handler) CreateSensorReading(w http.ResponseWriter, r *http.Request) {
	var req CreateSensorReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	reading, err := h.service.CreateSensorReading(&req)
	if err != nil {
		switch err {
		case ErrInvalidQuality, ErrInvalidValue:
			response.BadRequest(w, "Validation failed", err)
		case ErrSensorNotFound:
			response.NotFound(w, "Sensor not found")
		case ErrSensorInactive:
			response.Forbidden(w, "Sensor is inactive")
		default:
			response.InternalServerError(w, "Failed to create sensor reading", err)
		}
		return
	}

	response.Created(w, "Sensor reading created successfully", reading)
}

// CreateBulkSensorReadings handles bulk sensor readings creation
func (h *Handler) CreateBulkSensorReadings(w http.ResponseWriter, r *http.Request) {
	var req BulkSensorReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	if err := h.service.CreateBulkSensorReadings(&req); err != nil {
		if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "invalid") {
			response.BadRequest(w, "Validation failed", err)
		} else if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, err.Error())
		} else if strings.Contains(err.Error(), "inactive") {
			response.Forbidden(w, err.Error())
		} else {
			response.InternalServerError(w, "Failed to create bulk sensor readings", err)
		}
		return
	}

	response.Success(w, "Bulk sensor readings created successfully", map[string]int{
		"count": len(req.Readings),
	})
}

// GetSensorReadings handles getting sensor readings with filters
func (h *Handler) GetSensorReadings(w http.ResponseWriter, r *http.Request) {
	query := &SensorReadingQuery{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if sensorIDStr := r.URL.Query().Get("sensor_id"); sensorIDStr != "" {
		if sensorID, err := strconv.Atoi(sensorIDStr); err == nil {
			query.SensorID = &sensorID
		}
	}

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	if minQualityStr := r.URL.Query().Get("min_quality"); minQualityStr != "" {
		if minQuality, err := strconv.Atoi(minQualityStr); err == nil && minQuality >= 0 && minQuality <= 100 {
			query.MinQuality = &minQuality
		}
	}

	readings, total, err := h.service.GetSensorReadings(query)
	if err != nil {
		response.InternalServerError(w, "Failed to get sensor readings", err)
		return
	}

	// Calculate pagination meta
	totalPages := (total + query.Limit - 1) / query.Limit
	meta := &response.Meta{
		Page:       (query.Offset / query.Limit) + 1,
		PerPage:    query.Limit,
		Total:      total,
		TotalPages: totalPages,
	}

	response.PaginatedSuccess(w, "Sensor readings retrieved successfully", readings, meta)
}

// ListSensorTypes handles listing sensor types
func (h *Handler) ListSensorTypes(w http.ResponseWriter, r *http.Request) {
	sensorTypes, err := h.service.ListSensorTypes()
	if err != nil {
		response.InternalServerError(w, "Failed to list sensor types", err)
		return
	}

	response.Success(w, "Sensor types retrieved successfully", sensorTypes)
}

// CreateLocation handles location creation
func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	location, err := h.service.CreateLocation(&req)
	if err != nil {
		response.BadRequest(w, "Validation failed", err)
		return
	}

	response.Created(w, "Location created successfully", location)
}

// GetSensorType handles getting sensor type by ID
func (h *Handler) GetSensorType(w http.ResponseWriter, r *http.Request) {
	typeID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid sensor type ID", err)
		return
	}

	sensorType, err := h.service.GetSensorType(typeID)
	if err != nil {
		switch err {
		case ErrSensorTypeNotFound:
			response.NotFound(w, "Sensor type not found")
		default:
			response.InternalServerError(w, "Failed to get sensor type", err)
		}
		return
	}

	response.Success(w, "Sensor type retrieved successfully", sensorType)
}

// GetLocation handles getting location by ID
func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
	locationID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid location ID", err)
		return
	}

	location, err := h.service.GetLocation(locationID)
	if err != nil {
		switch err {
		case ErrLocationNotFound:
			response.NotFound(w, "Location not found")
		default:
			response.InternalServerError(w, "Failed to get location", err)
		}
		return
	}

	response.Success(w, "Location retrieved successfully", location)
}

// UpdateLocation handles location updates
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	locationID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.BadRequest(w, "Invalid location ID", err)
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body", err)
		return
	}

	location, err := h.service.UpdateLocation(locationID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			response.BadRequest(w, "Validation failed", err)
		} else if err == ErrLocationNotFound {
			response.NotFound(w, "Location not found")
		} else {
			response.InternalServerError(w, "Failed to update location", err)
		}
		return
	}

	response.Success(w, "Location updated successfully", location)
}

// ListLocations handles listing locations
func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.service.ListLocations()
	if err != nil {
		response.InternalServerError(w, "Failed to list locations", err)
		return
	}

	response.Success(w, "Locations retrieved successfully", locations)
}

// GetLocationSummary handles getting location summary with sensors
func (h *Handler) GetLocationSummary(w http.ResponseWriter, r *http.Request) {
	locationIDStr := r.URL.Query().Get("location_id")
	if locationIDStr == "" {
		response.BadRequest(w, "location_id parameter is required", nil)
		return
	}

	locationID, err := strconv.Atoi(locationIDStr)
	if err != nil {
		response.BadRequest(w, "Invalid location ID", err)
		return
	}

	summary, err := h.service.GetLocationSummary(locationID)
	if err != nil {
		if err == ErrLocationNotFound {
			response.NotFound(w, "Location not found")
		} else {
			response.InternalServerError(w, "Failed to get location summary", err)
		}
		return
	}

	response.Success(w, "Location summary retrieved successfully", summary)
}

// GetDashboard handles getting sensor dashboard data
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.GetSensorsDashboard()
	if err != nil {
		response.InternalServerError(w, "Failed to get dashboard data", err)
		return
	}

	response.Success(w, "Dashboard data retrieved successfully", dashboard)
}

// GetSensorHealth handles getting sensor health status
func (h *Handler) GetSensorHealth(w http.ResponseWriter, r *http.Request) {
	healthStatuses, err := h.service.GetSensorHealth()
	if err != nil {
		response.InternalServerError(w, "Failed to get sensor health data", err)
		return
	}

	response.Success(w, "Sensor health data retrieved successfully", healthStatuses)
}

// GetSensorStatistics handles getting sensor statistics
func (h *Handler) GetSensorStatistics(w http.ResponseWriter, r *http.Request) {
	sensorIDStr := r.URL.Query().Get("sensor_id")
	if sensorIDStr == "" {
		response.BadRequest(w, "sensor_id parameter is required", nil)
		return
	}

	sensorID, err := strconv.Atoi(sensorIDStr)
	if err != nil {
		response.BadRequest(w, "Invalid sensor ID", err)
		return
	}

	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		response.BadRequest(w, "start_time and end_time parameters are required", nil)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		response.BadRequest(w, "Invalid start_time format, use RFC3339", err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		response.BadRequest(w, "Invalid end_time format, use RFC3339", err)
		return
	}

	stats, err := h.service.GetSensorStatistics(sensorID, startTime, endTime)
	if err != nil {
		if err == ErrSensorNotFound {
			response.NotFound(w, "Sensor not found")
		} else {
			response.InternalServerError(w, "Failed to get sensor statistics", err)
		}
		return
	}

	response.Success(w, "Sensor statistics retrieved successfully", stats)
}

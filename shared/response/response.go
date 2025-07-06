package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta represents pagination and additional metadata
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents error response with details
type ErrorResponse struct {
	Success    bool              `json:"success"`
	Message    string            `json:"message"`
	Error      string            `json:"error"`
	Errors     []ValidationError `json:"errors,omitempty"`
	StatusCode int               `json:"status_code"`
}

// JSON sends JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Success sends success response
func Success(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	JSON(w, http.StatusOK, response)
}

// Created sends created response
func Created(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	JSON(w, http.StatusCreated, response)
}

// Error sends error response
func Error(w http.ResponseWriter, statusCode int, message string, err error) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	response := ErrorResponse{
		Success:    false,
		Message:    message,
		Error:      errorMsg,
		StatusCode: statusCode,
	}
	JSON(w, statusCode, response)
}

// BadRequest sends bad request error
func BadRequest(w http.ResponseWriter, message string, err error) {
	Error(w, http.StatusBadRequest, message, err)
}

// Unauthorized sends unauthorized error
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message, nil)
}

// Forbidden sends forbidden error
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message, nil)
}

// NotFound sends not found error
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message, nil)
}

// Conflict sends conflict error
func Conflict(w http.ResponseWriter, message string, err error) {
	Error(w, http.StatusConflict, message, err)
}

// InternalServerError sends internal server error
func InternalServerError(w http.ResponseWriter, message string, err error) {
	Error(w, http.StatusInternalServerError, message, err)
}

// ValidationErrors sends validation error response
func ValidationErrors(w http.ResponseWriter, message string, errors []ValidationError) {
	response := ErrorResponse{
		Success:    false,
		Message:    message,
		Errors:     errors,
		StatusCode: http.StatusBadRequest,
	}
	JSON(w, http.StatusBadRequest, response)
}

// PaginatedSuccess sends paginated success response
func PaginatedSuccess(w http.ResponseWriter, message string, data interface{}, meta *Meta) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	}
	JSON(w, http.StatusOK, response)
}

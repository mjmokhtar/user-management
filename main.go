package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user-management/config"
	"user-management/database"
	"user-management/pkg/mqtt"
	"user-management/pkg/sensor"
	"user-management/pkg/user"
	"user-management/shared/middleware"
)

func main() {
	// Load configuration
	cfg := config.MustLoad("app.toml")

	// Connect to database
	db := database.MustConnect(&cfg.Database)
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize services
	userRepo := user.NewRepository(db.DB)
	userService := user.NewService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)

	sensorRepo := sensor.NewRepository(db.DB)
	sensorService := sensor.NewService(sensorRepo)

	// Initialize MQTT broker
	mqttConfig := &mqtt.Config{
		Broker:   cfg.MQTT.Broker,
		Port:     cfg.MQTT.Port,
		Username: cfg.MQTT.Username,
		Password: cfg.MQTT.Password,
		ClientID: cfg.MQTT.ClientID,
		QoS:      cfg.MQTT.QoS,
	}

	mqttBroker := mqtt.NewMQTTBroker(mqttConfig, sensorService)

	// Start MQTT broker
	if err := mqttBroker.Start(); err != nil {
		log.Printf("Warning: Failed to start MQTT broker: %v", err)
		log.Println("Continuing without MQTT support...")
	} else {
		log.Println("MQTT broker started successfully")
		defer mqttBroker.Stop()
	}

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      setupRoutes(db, cfg, userService, sensorService),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// setupRoutes configures HTTP routes
func setupRoutes(db *database.DB, cfg *config.Config, userService user.Service, sensorService sensor.Service) http.Handler {
	mux := http.NewServeMux()

	// Create handlers with the services passed from main
	userHandler := user.NewHandler(userService)

	// Create auth service adapter for sensor handler
	authService := user.NewAuthServiceAdapter(userService)
	sensorHandler := sensor.NewHandler(sensorService, middleware.NewAuthMiddleware(authService))

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// API info endpoint
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"message": "IoT User Management API",
			"version": "1.0.0",
			"modules": ["user_management", "sensor_data"],
			"endpoints": {
				"auth": {
					"register": "POST /api/auth/register",
					"login": "POST /api/auth/login",
					"profile": "GET /api/auth/profile",
					"update_profile": "PUT /api/auth/profile",
					"permissions": "GET /api/auth/permissions"
				},
				"users": {
					"list": "GET /api/users",
					"get": "GET /api/users/{id}",
					"update": "PUT /api/users/{id}",
					"deactivate": "DELETE /api/users/{id}",
					"roles": "GET /api/users/{id}/roles"
				},
				"roles": {
					"list": "GET /api/roles",
					"assign": "POST /api/users/roles",
					"remove": "DELETE /api/users/roles"
				},
				"sensors": {
					"dashboard": "GET /api/sensors/dashboard",
					"list": "GET /api/sensors",
					"get": "GET /api/sensors/{id}",
					"get_by_device": "GET /api/sensors/device/{device_id}",
					"create": "POST /api/sensors",
					"update": "PUT /api/sensors/{id}",
					"delete": "DELETE /api/sensors/{id}",
					"health": "GET /api/sensors/health"
				},
				"sensor_data": {
					"create_reading": "POST /api/sensors/readings",
					"create_bulk": "POST /api/sensors/readings/bulk",
					"get_readings": "GET /api/sensors/readings",
					"statistics": "GET /api/sensors/statistics"
				},
				"locations": {
					"list": "GET /api/locations",
					"get": "GET /api/locations/{id}",
					"create": "POST /api/locations",
					"update": "PUT /api/locations/{id}",
					"summary": "GET /api/locations/sensors"
				},
				"sensor_types": {
					"list": "GET /api/sensor-types",
					"get": "GET /api/sensor-types/{id}"
				}
			}
		}`))
	})

	// Register domain routes
	userHandler.RegisterRoutes(mux)
	sensorHandler.RegisterRoutes(mux)

	// Apply middleware chain
	handler := middleware.CORS(mux)
	handler = middleware.Logging(handler)

	return handler
}

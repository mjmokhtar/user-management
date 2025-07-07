package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"user-management/pkg/sensor"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTBroker handles MQTT connections and message processing
type MQTTBroker struct {
	client        mqtt.Client
	sensorService sensor.Service
	config        *Config
}

// Config holds MQTT broker configuration
type Config struct {
	Broker   string `toml:"broker"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	ClientID string `toml:"client_id"`
	QoS      byte   `toml:"qos"`
}

// SensorDataMessage represents incoming sensor data via MQTT
type SensorDataMessage struct {
	DeviceID  string      `json:"device_id"`
	Timestamp *time.Time  `json:"timestamp,omitempty"`
	Value     float64     `json:"value"`
	Quality   *int        `json:"quality,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// BulkSensorDataMessage represents bulk sensor data
type BulkSensorDataMessage struct {
	DeviceID string              `json:"device_id"`
	Readings []SensorDataReading `json:"readings"`
}

// SensorDataReading represents individual reading in bulk message
type SensorDataReading struct {
	Timestamp *time.Time  `json:"timestamp,omitempty"`
	Value     float64     `json:"value"`
	Quality   *int        `json:"quality,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// DeviceStatusMessage represents device status updates
type DeviceStatusMessage struct {
	DeviceID        string `json:"device_id"`
	BatteryLevel    *int   `json:"battery_level,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	IsOnline        bool   `json:"is_online"`
}

// NewMQTTBroker creates a new MQTT broker instance
func NewMQTTBroker(config *Config, sensorService sensor.Service) *MQTTBroker {
	broker := &MQTTBroker{
		sensorService: sensorService,
		config:        config,
	}

	// Set up MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)

	// Set connection handlers
	opts.SetOnConnectHandler(broker.onConnect)
	opts.SetConnectionLostHandler(broker.onConnectionLost)

	broker.client = mqtt.NewClient(opts)

	return broker
}

// Start connects to MQTT broker and sets up subscriptions
func (mb *MQTTBroker) Start() error {
	log.Println("Connecting to MQTT broker...")

	if token := mb.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	log.Println("Successfully connected to MQTT broker")
	return nil
}

// Stop disconnects from MQTT broker
func (mb *MQTTBroker) Stop() {
	log.Println("Disconnecting from MQTT broker...")
	mb.client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
}

// onConnect is called when MQTT connection is established
func (mb *MQTTBroker) onConnect(client mqtt.Client) {
	log.Println("MQTT client connected, setting up subscriptions...")

	// Subscribe to different topic patterns
	subscriptions := map[string]mqtt.MessageHandler{
		"sensors/+/data":      mb.handleSensorData,
		"sensors/+/data/bulk": mb.handleBulkSensorData,
		"sensors/+/status":    mb.handleDeviceStatus,
		"sensors/+/heartbeat": mb.handleHeartbeat,
	}

	for topic, handler := range subscriptions {
		if token := client.Subscribe(topic, mb.config.QoS, handler); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe to topic %s: %v", topic, token.Error())
		} else {
			log.Printf("Successfully subscribed to topic: %s", topic)
		}
	}
}

// onConnectionLost is called when MQTT connection is lost
func (mb *MQTTBroker) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
	log.Println("Attempting to reconnect...")
}

// handleSensorData processes individual sensor readings
func (mb *MQTTBroker) handleSensorData(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received sensor data on topic: %s", msg.Topic())

	// Extract device ID from topic (sensors/{device_id}/data)
	deviceID := mb.extractDeviceIDFromTopic(msg.Topic())
	if deviceID == "" {
		log.Printf("Invalid topic format: %s", msg.Topic())
		return
	}

	// Parse message payload
	var sensorMsg SensorDataMessage
	if err := json.Unmarshal(msg.Payload(), &sensorMsg); err != nil {
		log.Printf("Failed to parse sensor data message: %v", err)
		return
	}

	// Use device ID from topic if not provided in message
	if sensorMsg.DeviceID == "" {
		sensorMsg.DeviceID = deviceID
	}

	// Process sensor reading
	if err := mb.processSensorReading(sensorMsg); err != nil {
		log.Printf("Failed to process sensor reading from %s: %v", deviceID, err)
		return
	}

	log.Printf("Successfully processed sensor reading from device: %s", deviceID)
}

// handleBulkSensorData processes bulk sensor readings
func (mb *MQTTBroker) handleBulkSensorData(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received bulk sensor data on topic: %s", msg.Topic())

	// Extract device ID from topic
	deviceID := mb.extractDeviceIDFromTopic(msg.Topic())
	if deviceID == "" {
		log.Printf("Invalid topic format: %s", msg.Topic())
		return
	}

	// Parse message payload
	var bulkMsg BulkSensorDataMessage
	if err := json.Unmarshal(msg.Payload(), &bulkMsg); err != nil {
		log.Printf("Failed to parse bulk sensor data message: %v", err)
		return
	}

	// Use device ID from topic if not provided in message
	if bulkMsg.DeviceID == "" {
		bulkMsg.DeviceID = deviceID
	}

	// Process bulk readings
	if err := mb.processBulkSensorReadings(bulkMsg); err != nil {
		log.Printf("Failed to process bulk sensor readings from %s: %v", deviceID, err)
		return
	}

	log.Printf("Successfully processed %d bulk readings from device: %s", len(bulkMsg.Readings), deviceID)
}

// handleDeviceStatus processes device status updates
func (mb *MQTTBroker) handleDeviceStatus(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received device status on topic: %s", msg.Topic())

	// Extract device ID from topic
	deviceID := mb.extractDeviceIDFromTopic(msg.Topic())
	if deviceID == "" {
		log.Printf("Invalid topic format: %s", msg.Topic())
		return
	}

	// Parse message payload
	var statusMsg DeviceStatusMessage
	if err := json.Unmarshal(msg.Payload(), &statusMsg); err != nil {
		log.Printf("Failed to parse device status message: %v", err)
		return
	}

	// Use device ID from topic if not provided in message
	if statusMsg.DeviceID == "" {
		statusMsg.DeviceID = deviceID
	}

	// Process device status update
	if err := mb.processDeviceStatus(statusMsg); err != nil {
		log.Printf("Failed to process device status from %s: %v", deviceID, err)
		return
	}

	log.Printf("Successfully processed device status from: %s", deviceID)
}

// handleHeartbeat processes device heartbeat messages
func (mb *MQTTBroker) handleHeartbeat(client mqtt.Client, msg mqtt.Message) {
	deviceID := mb.extractDeviceIDFromTopic(msg.Topic())
	if deviceID == "" {
		return
	}

	log.Printf("Received heartbeat from device: %s", deviceID)

	// Update device last seen timestamp
	statusMsg := DeviceStatusMessage{
		DeviceID: deviceID,
		IsOnline: true,
	}

	if err := mb.processDeviceStatus(statusMsg); err != nil {
		log.Printf("Failed to process heartbeat from %s: %v", deviceID, err)
	}
}

// processSensorReading converts MQTT message to sensor reading and saves it
func (mb *MQTTBroker) processSensorReading(msg SensorDataMessage) error {
	// Get sensor by device ID
	sensorData, err := mb.sensorService.GetSensorByDeviceID(msg.DeviceID)
	if err != nil {
		return fmt.Errorf("sensor not found for device %s: %w", msg.DeviceID, err)
	}

	// Convert metadata to JSON if provided
	var metadataJSON json.RawMessage
	if msg.Metadata != nil {
		metadataBytes, _ := json.Marshal(msg.Metadata)
		metadataJSON = json.RawMessage(metadataBytes)
	}

	// Create sensor reading request
	readingReq := &sensor.CreateSensorReadingRequest{
		SensorID:  sensorData.ID,
		Value:     msg.Value,
		Timestamp: msg.Timestamp,
		Quality:   msg.Quality,
		Metadata:  metadataJSON,
	}

	// Save sensor reading
	_, err = mb.sensorService.CreateSensorReading(readingReq)
	return err
}

// processBulkSensorReadings converts bulk MQTT message to sensor readings
func (mb *MQTTBroker) processBulkSensorReadings(msg BulkSensorDataMessage) error {
	// Get sensor by device ID
	sensorData, err := mb.sensorService.GetSensorByDeviceID(msg.DeviceID)
	if err != nil {
		return fmt.Errorf("sensor not found for device %s: %w", msg.DeviceID, err)
	}

	// Convert readings
	var readings []sensor.CreateSensorReadingRequest
	for _, reading := range msg.Readings {
		var metadataJSON json.RawMessage
		if reading.Metadata != nil {
			metadataBytes, _ := json.Marshal(reading.Metadata)
			metadataJSON = json.RawMessage(metadataBytes)
		}

		readingReq := sensor.CreateSensorReadingRequest{
			SensorID:  sensorData.ID,
			Value:     reading.Value,
			Timestamp: reading.Timestamp,
			Quality:   reading.Quality,
			Metadata:  metadataJSON,
		}
		readings = append(readings, readingReq)
	}

	// Save bulk readings
	bulkReq := &sensor.BulkSensorReadingRequest{
		Readings: readings,
	}

	return mb.sensorService.CreateBulkSensorReadings(bulkReq)
}

// processDeviceStatus updates device status information
func (mb *MQTTBroker) processDeviceStatus(msg DeviceStatusMessage) error {
	// Get sensor by device ID
	existingSensor, err := mb.sensorService.GetSensorByDeviceID(msg.DeviceID)
	if err != nil {
		return fmt.Errorf("sensor not found for device %s: %w", msg.DeviceID, err)
	}

	// Create update request
	updateReq := &sensor.UpdateSensorRequest{}

	if msg.BatteryLevel != nil {
		updateReq.BatteryLevel = msg.BatteryLevel
	}

	if msg.FirmwareVersion != "" {
		updateReq.FirmwareVersion = &msg.FirmwareVersion
	}

	// Update sensor
	_, err = mb.sensorService.UpdateSensor(existingSensor.ID, updateReq)
	return err
}

// extractDeviceIDFromTopic extracts device ID from MQTT topic
func (mb *MQTTBroker) extractDeviceIDFromTopic(topic string) string {
	// Expected format: sensors/{device_id}/data, sensors/{device_id}/status, etc.
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "sensors" {
		return parts[1]
	}
	return ""
}

// PublishCommand publishes command to specific device
func (mb *MQTTBroker) PublishCommand(deviceID string, command interface{}) error {
	topic := fmt.Sprintf("sensors/%s/commands", deviceID)

	payload, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	token := mb.client.Publish(topic, mb.config.QoS, false, payload)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to publish command: %w", token.Error())
	}

	log.Printf("Published command to device %s on topic %s", deviceID, topic)
	return nil
}

// GetConnectionStatus returns current MQTT connection status
func (mb *MQTTBroker) GetConnectionStatus() bool {
	return mb.client.IsConnected()
}

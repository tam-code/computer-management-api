package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// NotificationLevel represents the severity level of a notification
type NotificationLevel string

const (
	LevelInfo     NotificationLevel = "info"
	LevelWarning  NotificationLevel = "warning"
	LevelError    NotificationLevel = "error"
	LevelCritical NotificationLevel = "critical"
)

// Notifier is an interface for sending notifications with context support
type Notifier interface {
	SendNotification(notification Notification) error
	SendNotificationWithContext(ctx context.Context, notification Notification) error
	IsHealthy(ctx context.Context) bool
}

// NotificationConfig holds configuration for the notification client
type NotificationConfig struct {
	URL            string
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
	MaxPayloadSize int64
}

// DefaultConfig returns a default configuration for the notification client
func DefaultConfig(url string) NotificationConfig {
	return NotificationConfig{
		URL:            url,
		Timeout:        10 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
		MaxPayloadSize: 1024 * 1024, // 1MB
	}
}

// notificationClient is the concrete implementation of the Notifier interface
type notificationClient struct {
	config NotificationConfig
	client *http.Client
	logger *log.Logger
}

// NewNotifier creates a new Notifier with default configuration
func NewNotifier(url string) Notifier {
	return NewNotifierWithConfig(DefaultConfig(url))
}

// NewNotifierWithConfig creates a new Notifier with custom configuration
func NewNotifierWithConfig(config NotificationConfig) Notifier {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &notificationClient{
		config: config,
		client: client,
		logger: log.Default(),
	}
}

// SetLogger sets a custom logger for the notification client
func (c *notificationClient) SetLogger(logger *log.Logger) {
	if logger != nil {
		c.logger = logger
	}
}

// Notification represents the payload for the notification service
type Notification struct {
	Level                NotificationLevel `json:"level"`
	EmployeeAbbreviation string            `json:"employeeAbbreviation"`
	Message              string            `json:"message"`
	Timestamp            time.Time         `json:"timestamp,omitempty"`
	Source               string            `json:"source,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

// Validate checks if the notification is valid
func (n *Notification) Validate() error {
	if n.Level == "" {
		return fmt.Errorf("notification level is required")
	}
	if n.Message == "" {
		return fmt.Errorf("notification message is required")
	}
	if len(n.Message) > 1000 {
		return fmt.Errorf("notification message too long (max 1000 characters)")
	}
	if n.EmployeeAbbreviation != "" && len(n.EmployeeAbbreviation) > 10 {
		return fmt.Errorf("employee abbreviation too long (max 10 characters)")
	}

	// Validate level
	validLevels := []NotificationLevel{LevelInfo, LevelWarning, LevelError, LevelCritical}
	isValidLevel := false
	for _, level := range validLevels {
		if n.Level == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("invalid notification level: %s", n.Level)
	}

	return nil
}

// SendNotification sends a notification to the notification service
func (c *notificationClient) SendNotification(notification Notification) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	return c.SendNotificationWithContext(ctx, notification)
}

// SendNotificationWithContext sends a notification with context support
func (c *notificationClient) SendNotificationWithContext(ctx context.Context, notification Notification) error {
	// Validate notification
	if err := notification.Validate(); err != nil {
		return fmt.Errorf("invalid notification: %w", err)
	}

	// Set timestamp if not provided
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	// Set source if not provided
	if notification.Source == "" {
		notification.Source = "computer-management-api"
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}
			c.logger.Printf("Retrying notification send (attempt %d/%d)", attempt+1, c.config.RetryAttempts+1)
		}

		if err := c.sendNotificationAttempt(ctx, notification); err != nil {
			lastErr = err
			c.logger.Printf("Notification send attempt %d failed: %v", attempt+1, err)

			// Don't retry on validation, client errors, or payload size errors
			if strings.Contains(err.Error(), "400") ||
				strings.Contains(err.Error(), "invalid") ||
				strings.Contains(err.Error(), "payload too large") ||
				strings.Contains(err.Error(), "failed to marshal") {
				return err
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to send notification after %d attempts: %w", c.config.RetryAttempts+1, lastErr)
}

// sendNotificationAttempt performs a single notification send attempt
func (c *notificationClient) sendNotificationAttempt(ctx context.Context, notification Notification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Check payload size
	if int64(len(payload)) > c.config.MaxPayloadSize {
		return fmt.Errorf("notification payload too large: %d bytes (max %d)", len(payload), c.config.MaxPayloadSize)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "computer-management-api/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for better error reporting
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification service returned error status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		c.logger.Printf("Warning: unexpected status code %d from notification service", resp.StatusCode)
	}

	return nil
}

// IsHealthy checks if the notification service is healthy
func (c *notificationClient) IsHealthy(ctx context.Context) bool {
	// Create a simple health check request
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.URL+"/health", nil)
	if err != nil {
		// If health endpoint doesn't exist, try a HEAD request to main endpoint
		req, err = http.NewRequestWithContext(ctx, "HEAD", c.config.URL, nil)
		if err != nil {
			return false
		}
	}

	req.Header.Set("User-Agent", "computer-management-api/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

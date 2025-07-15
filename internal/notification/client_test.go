package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNotification_Validate(t *testing.T) {
	tests := []struct {
		name          string
		notification  Notification
		expectError   bool
		errorContains string
	}{
		{
			name: "valid notification",
			notification: Notification{
				Level:                LevelWarning,
				EmployeeAbbreviation: "ABC",
				Message:              "Test message",
			},
			expectError: false,
		},
		{
			name: "missing level",
			notification: Notification{
				EmployeeAbbreviation: "ABC",
				Message:              "Test message",
			},
			expectError:   true,
			errorContains: "level is required",
		},
		{
			name: "missing message",
			notification: Notification{
				Level:                LevelWarning,
				EmployeeAbbreviation: "ABC",
			},
			expectError:   true,
			errorContains: "message is required",
		},
		{
			name: "message too long",
			notification: Notification{
				Level:   LevelWarning,
				Message: strings.Repeat("a", 1001),
			},
			expectError:   true,
			errorContains: "message too long",
		},
		{
			name: "employee abbreviation too long",
			notification: Notification{
				Level:                LevelWarning,
				EmployeeAbbreviation: "VERYLONGABBREVIATION",
				Message:              "Test message",
			},
			expectError:   true,
			errorContains: "employee abbreviation too long",
		},
		{
			name: "invalid level",
			notification: Notification{
				Level:   "invalid",
				Message: "Test message",
			},
			expectError:   true,
			errorContains: "invalid notification level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.notification.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestNotificationClient_SendNotification_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("User-Agent") != "computer-management-api/1.0" {
			t.Errorf("Expected User-Agent computer-management-api/1.0, got %s", r.Header.Get("User-Agent"))
		}

		// Verify request body
		var notification Notification
		if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if notification.Level != LevelWarning {
			t.Errorf("Expected level warning, got %s", notification.Level)
		}
		if notification.Message != "Test message" {
			t.Errorf("Expected message 'Test message', got %s", notification.Message)
		}
		if notification.Source != "computer-management-api" {
			t.Errorf("Expected source 'computer-management-api', got %s", notification.Source)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewNotifier(server.URL)
	notification := Notification{
		Level:                LevelWarning,
		EmployeeAbbreviation: "ABC",
		Message:              "Test message",
	}

	err := client.SendNotification(notification)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestNotificationClient_SendNotification_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewNotifier(server.URL)
	notification := Notification{
		Level:   LevelWarning,
		Message: "Test message",
	}

	err := client.SendNotification(notification)
	if err == nil {
		t.Error("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain '500', got: %v", err)
	}
}

func TestNotificationClient_SendNotification_ValidationError(t *testing.T) {
	client := NewNotifier("http://localhost:8080")
	notification := Notification{
		// Missing required fields
		EmployeeAbbreviation: "ABC",
	}

	err := client.SendNotification(notification)
	if err == nil {
		t.Error("Expected validation error but got none")
	}
	if !strings.Contains(err.Error(), "invalid notification") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestNotificationClient_SendNotificationWithContext_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewNotifier(server.URL)
	notification := Notification{
		Level:   LevelWarning,
		Message: "Test message",
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.SendNotificationWithContext(ctx, notification)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

func TestNotificationClient_Retry_Mechanism(t *testing.T) {
	attempts := 0
	// Create a test server that fails first two attempts then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL)
	config.RetryAttempts = 3
	config.RetryDelay = 10 * time.Millisecond
	client := NewNotifierWithConfig(config)

	notification := Notification{
		Level:   LevelWarning,
		Message: "Test message",
	}

	err := client.SendNotification(notification)
	if err != nil {
		t.Errorf("Expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestNotificationClient_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		serverStatus   int
		expectedHealth bool
	}{
		{
			name:           "healthy service",
			serverStatus:   http.StatusOK,
			expectedHealth: true,
		},
		{
			name:           "client error still healthy",
			serverStatus:   http.StatusBadRequest,
			expectedHealth: true,
		},
		{
			name:           "server error unhealthy",
			serverStatus:   http.StatusInternalServerError,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := NewNotifier(server.URL)
			ctx := context.Background()
			healthy := client.IsHealthy(ctx)

			if healthy != tt.expectedHealth {
				t.Errorf("Expected health %v, got %v", tt.expectedHealth, healthy)
			}
		})
	}
}

func TestNotificationClient_PayloadSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL)
	config.MaxPayloadSize = 100 // Very small limit
	client := NewNotifierWithConfig(config)

	notification := Notification{
		Level:   LevelWarning,
		Message: strings.Repeat("a", 200), // Large message
	}

	err := client.SendNotification(notification)
	if err == nil {
		t.Error("Expected payload size error but got none")
	}
	if !strings.Contains(err.Error(), "payload too large") {
		t.Errorf("Expected payload size error, got: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	url := "http://example.com"
	config := DefaultConfig(url)

	if config.URL != url {
		t.Errorf("Expected URL %s, got %s", url, config.URL)
	}
	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", config.RetryAttempts)
	}
	if config.RetryDelay != time.Second {
		t.Errorf("Expected 1s retry delay, got %v", config.RetryDelay)
	}
	if config.MaxPayloadSize != 1024*1024 {
		t.Errorf("Expected 1MB max payload size, got %d", config.MaxPayloadSize)
	}
}

func TestNotificationLevels(t *testing.T) {
	levels := []NotificationLevel{LevelInfo, LevelWarning, LevelError, LevelCritical}
	expectedLevels := []string{"info", "warning", "error", "critical"}

	for i, level := range levels {
		if string(level) != expectedLevels[i] {
			t.Errorf("Expected level %s, got %s", expectedLevels[i], string(level))
		}
	}
}

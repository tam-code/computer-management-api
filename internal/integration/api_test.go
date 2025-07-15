package integration

import (
	"bytes"
	"computer-management-api/internal/config"
	"computer-management-api/internal/database"
	"computer-management-api/internal/handler"
	"computer-management-api/internal/model"
	"computer-management-api/internal/notification"
	"computer-management-api/internal/repository"
	"computer-management-api/internal/router"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// mockNotifier implements the Notifier interface for testing
type mockNotifier struct {
	notifications []notification.Notification
}

func (m *mockNotifier) SendNotification(n notification.Notification) error {
	m.notifications = append(m.notifications, n)
	return nil
}

func (m *mockNotifier) SendNotificationWithContext(ctx context.Context, n notification.Notification) error {
	return m.SendNotification(n)
}

func (m *mockNotifier) IsHealthy(ctx context.Context) bool {
	return true
}

// TestConfig represents configuration for integration tests
type TestConfig struct {
	DatabaseURL string
	TestTimeout time.Duration
}

// IntegrationTestSuite holds the test dependencies
type IntegrationTestSuite struct {
	DB     *sql.DB
	Router http.Handler
	Config *config.Config
}

// setupIntegrationTest initializes the test environment
func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	t.Helper()

	// Check if integration tests should run
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg := loadTestConfig(t)

	// Initialize test database
	db := initTestDatabase(t, cfg)

	// Clean database before test
	cleanDatabase(t, db)

	// Initialize dependencies
	repo := repository.NewComputerRepository(db)
	notifier := &mockNotifier{} // Use mock for tests
	computerHandler := handler.NewComputerHandler(repo, notifier, nil)

	// Create test config
	cfg = &config.Config{
		Security: config.SecurityConfig{
			RateLimitRPS:    100,
			RateLimitBurst:  200,
			RequestTimeout:  30 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			EnableCORS:      true,
			AllowedOrigins:  []string{"*"},
			TrustedProxies:  []string{},
		},
	}

	testRouter := router.NewRouter(computerHandler, cfg)

	return &IntegrationTestSuite{
		DB:     db,
		Router: testRouter,
		Config: cfg,
	}
}

// teardownIntegrationTest cleans up test resources
func teardownIntegrationTest(t *testing.T, suite *IntegrationTestSuite) {
	t.Helper()
	if suite.DB != nil {
		cleanDatabase(t, suite.DB)
		suite.DB.Close()
	}
}

// loadTestConfig loads configuration for testing
func loadTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// Try to load from environment or use defaults
	dbHost := getEnv("TEST_DB_HOST", "127.0.0.1")
	dbPort := getEnv("TEST_DB_PORT", "5452")
	dbUser := getEnv("TEST_DB_USER", "postgres")
	dbPassword := getEnv("TEST_DB_PASSWORD", "postgres")
	dbName := getEnv("TEST_DB_NAME", "postgres")

	cfg := &config.Config{
		Port:     8080,
		LogLevel: "info",
		Database: config.DatabaseConfig{
			Host:     dbHost,
			Port:     mustParseInt(dbPort),
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
			SSLMode:  "disable",
		},
	}

	return cfg
}

// initTestDatabase initializes the test database connection
func initTestDatabase(t *testing.T, cfg *config.Config) *sql.DB {
	t.Helper()

	db, err := database.InitDB(cfg)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v. Ensure test database is running.", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Failed to ping test database: %v", err)
	}

	return db
}

// cleanDatabase removes all test data
func cleanDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	// Use TRUNCATE for complete cleanup
	_, err := db.Exec("TRUNCATE TABLE computers RESTART IDENTITY CASCADE")
	if err != nil {
		// Fallback to DELETE if TRUNCATE fails
		_, err = db.Exec("DELETE FROM computers")
		if err != nil {
			t.Logf("Warning: Failed to clean database: %v", err)
		}
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustParseInt(s string) int {
	if s == "" {
		return 0
	}
	// Simple conversion for test purposes
	switch s {
	case "5452":
		return 5452
	case "5432":
		return 5432
	default:
		return 5432
	}
}

// Test helper to create HTTP request with JSON body
func createJSONRequest(method, url string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Test helper to parse JSON response
func parseJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, resp.Body.String())
	}
}

// Integration Tests

func TestIntegration_ComputerCRUD(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	// Test data
	testComputer := model.Computer{
		MACAddress:           "AA:BB:CC:DD:EE:FF",
		ComputerName:         "TEST-INTEGRATION-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "ABC",
		Description:          "Integration test computer",
	}

	var createdID uuid.UUID

	t.Run("Create Computer", func(t *testing.T) {
		req := createJSONRequest("POST", "/api/v1/computers", testComputer)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, resp.Code, resp.Body.String())
		}

		var response map[string]interface{}
		parseJSONResponse(t, resp, &response)

		if response["message"] != "Computer created successfully" {
			t.Errorf("Unexpected response message: %v", response["message"])
		}

		// Extract created ID for subsequent tests
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Failed to get data from response: %+v", response)
		}
		createdIDStr, ok := data["id"].(string)
		if !ok {
			t.Fatalf("Failed to get ID from data: %+v", data)
		}
		var err error
		createdID, err = uuid.Parse(createdIDStr)
		if err != nil {
			t.Fatalf("Failed to parse created ID: %v", err)
		}
		t.Logf("Created computer with ID: %s", createdID)
	})

	t.Run("Get All Computers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/computers", nil)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var response map[string]interface{}
		parseJSONResponse(t, resp, &response)

		computers := response["computers"].([]interface{})
		if len(computers) != 1 {
			t.Errorf("Expected 1 computer, got %d", len(computers))
		}
	})

	t.Run("Get Computer By ID", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/computers/%s", createdID.String())
		req := httptest.NewRequest("GET", url, nil)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var computer model.Computer
		parseJSONResponse(t, resp, &computer)

		if computer.MACAddress != testComputer.MACAddress {
			t.Errorf("Expected MAC %s, got %s", testComputer.MACAddress, computer.MACAddress)
		}
		if computer.ComputerName != testComputer.ComputerName {
			t.Errorf("Expected name %s, got %s", testComputer.ComputerName, computer.ComputerName)
		}
	})

	t.Run("Update Computer", func(t *testing.T) {
		updateData := model.Computer{
			MACAddress:           "AA:BB:CC:DD:EE:FF", // Same MAC
			ComputerName:         "UPDATED-INTEGRATION-001",
			IPAddress:            "192.168.1.101",
			EmployeeAbbreviation: "DEF",
			Description:          "Updated integration test computer",
		}

		url := fmt.Sprintf("/api/v1/computers/%s", createdID.String())
		req := createJSONRequest("PUT", url, updateData)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		// Verify the update
		getReq := httptest.NewRequest("GET", url, nil)
		getResp := httptest.NewRecorder()
		suite.Router.ServeHTTP(getResp, getReq)

		var updatedComputer model.Computer
		parseJSONResponse(t, getResp, &updatedComputer)

		if updatedComputer.ComputerName != updateData.ComputerName {
			t.Errorf("Expected updated name %s, got %s", updateData.ComputerName, updatedComputer.ComputerName)
		}
		if updatedComputer.EmployeeAbbreviation != updateData.EmployeeAbbreviation {
			t.Errorf("Expected updated employee %s, got %s", updateData.EmployeeAbbreviation, updatedComputer.EmployeeAbbreviation)
		}
	})

	t.Run("Get Employee Computers", func(t *testing.T) {
		url := "/api/v1/employees/DEF/computers"
		req := httptest.NewRequest("GET", url, nil)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var response map[string]interface{}
		parseJSONResponse(t, resp, &response)

		if response["employee_abbreviation"] != "DEF" {
			t.Errorf("Expected employee DEF, got %v", response["employee_abbreviation"])
		}

		computersRaw, ok := response["computers"]
		if !ok || computersRaw == nil {
			t.Errorf("No computers field in response: %+v", response)
			return
		}
		computers, ok := computersRaw.([]interface{})
		if !ok {
			t.Errorf("Computers field is not an array: %T %+v", computersRaw, computersRaw)
			return
		}
		if len(computers) != 1 {
			t.Errorf("Expected 1 computer for employee DEF, got %d", len(computers))
		}
	})

	t.Run("Delete Computer", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/computers/%s", createdID.String())
		req := httptest.NewRequest("DELETE", url, nil)
		resp := httptest.NewRecorder()

		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		// Verify deletion
		getReq := httptest.NewRequest("GET", url, nil)
		getResp := httptest.NewRecorder()
		suite.Router.ServeHTTP(getResp, getReq)

		if getResp.Code != http.StatusNotFound {
			t.Errorf("Expected status %d after deletion, got %d", http.StatusNotFound, getResp.Code)
		}
	})
}

func TestIntegration_ValidationErrors(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	tests := []struct {
		name           string
		computer       model.Computer
		expectedStatus int
	}{
		{
			name: "Invalid MAC Address",
			computer: model.Computer{
				MACAddress:   "invalid-mac",
				ComputerName: "TEST-001",
				IPAddress:    "192.168.1.1",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid IP Address",
			computer: model.Computer{
				MACAddress:   "AA:BB:CC:DD:EE:FF",
				ComputerName: "TEST-001",
				IPAddress:    "invalid-ip",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Employee Abbreviation",
			computer: model.Computer{
				MACAddress:           "AA:BB:CC:DD:EE:FF",
				ComputerName:         "TEST-001",
				IPAddress:            "192.168.1.1",
				EmployeeAbbreviation: "TOOLONG",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty Computer Name",
			computer: model.Computer{
				MACAddress: "AA:BB:CC:DD:EE:FF",
				IPAddress:  "192.168.1.1",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createJSONRequest("POST", "/api/v1/computers", tt.computer)
			resp := httptest.NewRecorder()

			suite.Router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestIntegration_DuplicateMAC(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	computer1 := model.Computer{
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		ComputerName: "TEST-001",
		IPAddress:    "192.168.1.1",
	}

	computer2 := model.Computer{
		MACAddress:   "AA:BB:CC:DD:EE:FF", // Same MAC
		ComputerName: "TEST-002",
		IPAddress:    "192.168.1.2",
	}

	// Create first computer
	req1 := createJSONRequest("POST", "/api/v1/computers", computer1)
	resp1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(resp1, req1)

	if resp1.Code != http.StatusCreated {
		t.Fatalf("Failed to create first computer: %d", resp1.Code)
	}

	// Try to create second computer with same MAC
	req2 := createJSONRequest("POST", "/api/v1/computers", computer2)
	resp2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusConflict {
		t.Errorf("Expected status %d for duplicate MAC, got %d. Body: %s", http.StatusConflict, resp2.Code, resp2.Body.String())
	}
}

func TestIntegration_Pagination(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	// Create multiple computers for pagination testing
	computers := []model.Computer{
		{MACAddress: "AA:BB:CC:DD:EE:01", ComputerName: "TEST-001", IPAddress: "192.168.1.1"},
		{MACAddress: "AA:BB:CC:DD:EE:02", ComputerName: "TEST-002", IPAddress: "192.168.1.2"},
		{MACAddress: "AA:BB:CC:DD:EE:03", ComputerName: "TEST-003", IPAddress: "192.168.1.3"},
		{MACAddress: "AA:BB:CC:DD:EE:04", ComputerName: "TEST-004", IPAddress: "192.168.1.4"},
		{MACAddress: "AA:BB:CC:DD:EE:05", ComputerName: "TEST-005", IPAddress: "192.168.1.5"},
	}

	// Create all computers
	for _, computer := range computers {
		req := createJSONRequest("POST", "/api/v1/computers", computer)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("Failed to create computer %s: %d", computer.ComputerName, resp.Code)
		}
	}

	t.Run("First Page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/computers?page_size=2&page=1", nil)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var response map[string]interface{}
		parseJSONResponse(t, resp, &response)

		computersData, ok := response["computers"]
		if !ok {
			t.Fatalf("Response missing 'computers' field. Response: %+v", response)
		}

		computers, ok := computersData.([]interface{})
		if !ok {
			t.Fatalf("'computers' field is not an array. Got: %T", computersData)
		}

		paginationData, ok := response["pagination"]
		if !ok {
			t.Fatalf("Response missing 'pagination' field. Response: %+v", response)
		}

		pagination, ok := paginationData.(map[string]interface{})
		if !ok {
			t.Fatalf("'pagination' field is not an object. Got: %T", paginationData)
		}

		if len(computers) > 2 {
			t.Errorf("Expected max 2 computers on first page (page_size=2), got %d", len(computers))
		}

		totalData, ok := pagination["total_items"]
		if !ok {
			t.Fatalf("Pagination missing 'total_items' field. Pagination: %+v", pagination)
		}

		total, ok := totalData.(float64)
		if !ok {
			t.Fatalf("'total' field is not a number. Got: %T, Value: %v", totalData, totalData)
		}

		if total < 5 { // Should be at least 5 (could be more from previous tests)
			t.Errorf("Expected total >= 5, got %v", total)
		}
	})

	t.Run("Second Page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/computers?page_size=2&page=2", nil)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var response map[string]interface{}
		parseJSONResponse(t, resp, &response)

		computersData, ok := response["computers"]
		if !ok {
			t.Fatalf("Response missing 'computers' field. Response: %+v", response)
		}

		computers, ok := computersData.([]interface{})
		if !ok {
			t.Fatalf("'computers' field is not an array. Got: %T", computersData)
		}

		if len(computers) > 2 {
			t.Errorf("Expected max 2 computers on second page (page_size=2), got %d", len(computers))
		}
	})
}

func TestIntegration_HealthCheck(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp := httptest.NewRecorder()

	suite.Router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var response map[string]interface{}
	parseJSONResponse(t, resp, &response)

	if response["message"] != "Service is healthy" {
		t.Errorf("Expected healthy service message, got: %v", response["message"])
	}
}

func TestIntegration_NotFoundEndpoints(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	testCases := []struct {
		method string
		url    string
	}{
		{"GET", "/api/v1/computers/nonexistent-id"},
		{"PUT", "/api/v1/computers/nonexistent-id"},
		{"DELETE", "/api/v1/computers/nonexistent-id"},
		{"GET", "/api/v1/nonexistent-endpoint"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.url), func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			resp := httptest.NewRecorder()

			suite.Router.ServeHTTP(resp, req)

			if !strings.Contains(tc.url, "nonexistent-id") {
				// For nonexistent endpoints, expect 404
				if resp.Code != http.StatusNotFound {
					t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.Code)
				}
			} else {
				// For operations on nonexistent IDs, might be 400 or 404 depending on validation
				if resp.Code != http.StatusNotFound && resp.Code != http.StatusBadRequest {
					t.Errorf("Expected status %d or %d, got %d", http.StatusNotFound, http.StatusBadRequest, resp.Code)
				}
			}
		})
	}
}

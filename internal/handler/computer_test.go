package handler

import (
	"bytes"
	"computer-management-api/internal/model"
	"computer-management-api/internal/notification"
	"computer-management-api/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Mock implementations for testing

// MockComputerRepository is a mock implementation of ComputerRepository
type MockComputerRepository struct {
	// Function fields to set expectations
	CreateComputerFunc                  func(ctx context.Context, computer model.Computer) error
	GetAllComputersFunc                 func(ctx context.Context) ([]model.Computer, error)
	GetAllComputersPaginatedFunc        func(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error)
	GetComputerByIDFunc                 func(ctx context.Context, id uuid.UUID) (*model.Computer, error)
	UpdateComputerFunc                  func(ctx context.Context, id uuid.UUID, computer model.Computer) error
	DeleteComputerFunc                  func(ctx context.Context, id uuid.UUID) error
	GetComputersByEmployeeFunc          func(ctx context.Context, employeeAbbreviation string) ([]model.Computer, error)
	GetComputersByEmployeePaginatedFunc func(ctx context.Context, employeeAbbreviation string, params repository.PaginationParams) (*repository.PaginatedResult, error)
	ComputerExistsFunc                  func(ctx context.Context, macAddress string) (bool, error)
	AssignComputerToEmployeeFunc        func(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error
	RemoveComputerFromEmployeeFunc      func(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error
}

func (m *MockComputerRepository) CreateComputer(ctx context.Context, computer model.Computer) error {
	if m.CreateComputerFunc != nil {
		return m.CreateComputerFunc(ctx, computer)
	}
	return nil
}

func (m *MockComputerRepository) GetAllComputers(ctx context.Context) ([]model.Computer, error) {
	if m.GetAllComputersFunc != nil {
		return m.GetAllComputersFunc(ctx)
	}
	return []model.Computer{}, nil
}

func (m *MockComputerRepository) GetAllComputersPaginated(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error) {
	if m.GetAllComputersPaginatedFunc != nil {
		return m.GetAllComputersPaginatedFunc(ctx, params)
	}
	return &repository.PaginatedResult{Items: []model.Computer{}, TotalCount: 0}, nil
}

func (m *MockComputerRepository) GetComputerByMAC(ctx context.Context, macAddress string) (*model.Computer, error) {
	// This method is still needed for repository interface compliance
	// even though the handler endpoint was removed
	return nil, repository.ErrComputerNotFound
}

func (m *MockComputerRepository) GetComputerByID(ctx context.Context, id uuid.UUID) (*model.Computer, error) {
	if m.GetComputerByIDFunc != nil {
		return m.GetComputerByIDFunc(ctx, id)
	}
	return nil, repository.ErrComputerNotFound
}

func (m *MockComputerRepository) UpdateComputer(ctx context.Context, id uuid.UUID, computer model.Computer) error {
	if m.UpdateComputerFunc != nil {
		return m.UpdateComputerFunc(ctx, id, computer)
	}
	return nil
}

func (m *MockComputerRepository) DeleteComputer(ctx context.Context, id uuid.UUID) error {
	if m.DeleteComputerFunc != nil {
		return m.DeleteComputerFunc(ctx, id)
	}
	return nil
}

func (m *MockComputerRepository) GetComputersByEmployee(ctx context.Context, employeeAbbreviation string) ([]model.Computer, error) {
	if m.GetComputersByEmployeeFunc != nil {
		return m.GetComputersByEmployeeFunc(ctx, employeeAbbreviation)
	}
	return []model.Computer{}, nil
}

func (m *MockComputerRepository) GetComputersByEmployeePaginated(ctx context.Context, employeeAbbreviation string, params repository.PaginationParams) (*repository.PaginatedResult, error) {
	if m.GetComputersByEmployeePaginatedFunc != nil {
		return m.GetComputersByEmployeePaginatedFunc(ctx, employeeAbbreviation, params)
	}
	return &repository.PaginatedResult{Items: []model.Computer{}, TotalCount: 0}, nil
}

func (m *MockComputerRepository) AssignComputerToEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error {
	if m.AssignComputerToEmployeeFunc != nil {
		return m.AssignComputerToEmployeeFunc(ctx, computerID, employeeAbbreviation)
	}
	return nil
}

func (m *MockComputerRepository) RemoveComputerFromEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error {
	if m.RemoveComputerFromEmployeeFunc != nil {
		return m.RemoveComputerFromEmployeeFunc(ctx, computerID, employeeAbbreviation)
	}
	return nil
}

func (m *MockComputerRepository) ComputerExists(ctx context.Context, macAddress string) (bool, error) {
	if m.ComputerExistsFunc != nil {
		return m.ComputerExistsFunc(ctx, macAddress)
	}
	return false, nil
}

// MockNotifier is a mock implementation of Notifier
type MockNotifier struct {
	SendNotificationFunc            func(notification notification.Notification) error
	SendNotificationWithContextFunc func(ctx context.Context, notification notification.Notification) error
	IsHealthyFunc                   func(ctx context.Context) bool
	// Track calls for verification
	NotificationsSent []notification.Notification
}

func (m *MockNotifier) SendNotification(notification notification.Notification) error {
	m.NotificationsSent = append(m.NotificationsSent, notification)
	if m.SendNotificationFunc != nil {
		return m.SendNotificationFunc(notification)
	}
	return nil
}

func (m *MockNotifier) SendNotificationWithContext(ctx context.Context, notification notification.Notification) error {
	m.NotificationsSent = append(m.NotificationsSent, notification)
	if m.SendNotificationWithContextFunc != nil {
		return m.SendNotificationWithContextFunc(ctx, notification)
	}
	return nil
}

func (m *MockNotifier) IsHealthy(ctx context.Context) bool {
	if m.IsHealthyFunc != nil {
		return m.IsHealthyFunc(ctx)
	}
	return true
}

// Helper functions for tests

func createTestComputer() model.Computer {
	return model.Computer{
		ID:                   uuid.New(),
		MACAddress:           "00:1B:44:11:3A:B7",
		ComputerName:         "TEST-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "ABC",
		Description:          "Test computer",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func createTestHandler() (*ComputerHandler, *MockComputerRepository, *MockNotifier) {
	mockRepo := &MockComputerRepository{}
	mockNotifier := &MockNotifier{
		NotificationsSent: make([]notification.Notification, 0),
	}
	logger := log.New(bytes.NewBuffer([]byte{}), "", 0) // Silent logger for tests

	handler := NewComputerHandler(mockRepo, mockNotifier, logger)
	return handler, mockRepo, mockNotifier
}

func createJSONRequest(method, url string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Test CreateComputerHandler

func TestCreateComputerHandler_Success(t *testing.T) {
	handler, mockRepo, mockNotifier := createTestHandler()

	computer := createTestComputer()
	computer.ID = uuid.Nil // ID should be auto-generated

	// Set up mock expectations
	mockRepo.CreateComputerFunc = func(ctx context.Context, c model.Computer) error {
		if c.ComputerName != computer.ComputerName || c.MACAddress != computer.MACAddress {
			t.Errorf("Unexpected computer data: got %+v", c)
		}
		return nil
	}

	// Mock the async notification check
	mockRepo.GetComputersByEmployeeFunc = func(ctx context.Context, emp string) ([]model.Computer, error) {
		if emp != "ABC" {
			t.Errorf("Expected employee ABC, got %s", emp)
		}
		return []model.Computer{computer}, nil
	}

	// Ensure mockNotifier is used
	_ = mockNotifier

	req := createJSONRequest("POST", "/computers", computer)
	rr := httptest.NewRecorder()

	handler.CreateComputerHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.Message != "Computer created successfully" {
		t.Errorf("Expected success message, got %s", response.Message)
	}
	if response.Data == nil {
		t.Error("Expected response data to be present")
	}
}

func TestCreateComputerHandler_InvalidJSON(t *testing.T) {
	handler, _, _ := createTestHandler()

	req, _ := http.NewRequest("POST", "/computers", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateComputerHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !strings.Contains(response.Error, "Invalid JSON") {
		t.Errorf("Expected JSON error message, got %s", response.Error)
	}
}

func TestCreateComputerHandler_ValidationError(t *testing.T) {
	handler, _, _ := createTestHandler()

	computer := model.Computer{
		// Missing required fields
		EmployeeAbbreviation: "ABC",
	}

	req := createJSONRequest("POST", "/computers", computer)
	rr := httptest.NewRecorder()

	handler.CreateComputerHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.Error != "Validation failed" {
		t.Errorf("Expected validation error, got %s", response.Error)
	}
	if response.Details == nil {
		t.Error("Expected validation details to be present")
	}
}

func TestCreateComputerHandler_RepositoryError(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computer := createTestComputer()
	computer.ID = uuid.Nil

	mockRepo.CreateComputerFunc = func(ctx context.Context, c model.Computer) error {
		return errors.New("database error")
	}

	req := createJSONRequest("POST", "/computers", computer)
	rr := httptest.NewRecorder()

	handler.CreateComputerHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !strings.Contains(response.Error, "Failed to create computer") {
		t.Errorf("Expected repository error message, got %s", response.Error)
	}
}

// Test GetAllComputersHandler

func TestGetAllComputersHandler_Success(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computers := []model.Computer{createTestComputer(), createTestComputer()}
	expectedResult := &repository.PaginatedResult{
		Items:      computers,
		TotalCount: 2,
	}

	mockRepo.GetAllComputersPaginatedFunc = func(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error) {
		if params.Offset != 0 || params.Limit != 10 {
			t.Errorf("Expected default pagination params (offset: 0, limit: 10), got offset: %d, limit: %d", params.Offset, params.Limit)
		}
		return expectedResult, nil
	}

	req, _ := http.NewRequest("GET", "/computers", nil)
	rr := httptest.NewRecorder()

	handler.GetAllComputersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if _, exists := response["computers"]; !exists {
		t.Error("Expected computers field in response")
	}
	if _, exists := response["pagination"]; !exists {
		t.Error("Expected pagination field in response")
	}
}

func TestGetAllComputersHandler_WithPagination(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computers := []model.Computer{createTestComputer()}
	expectedResult := &repository.PaginatedResult{
		Items:      computers,
		TotalCount: 25,
	}

	mockRepo.GetAllComputersPaginatedFunc = func(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error) {
		if params.Offset != 10 || params.Limit != 5 {
			t.Errorf("Expected pagination params (offset: 10, limit: 5), got offset: %d, limit: %d", params.Offset, params.Limit)
		}
		return expectedResult, nil
	}

	req, _ := http.NewRequest("GET", "/computers?page=3&page_size=5", nil)
	rr := httptest.NewRecorder()

	handler.GetAllComputersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	pagination := response["pagination"].(map[string]interface{})
	if pagination["page"] != float64(3) {
		t.Errorf("Expected page 3, got %v", pagination["page"])
	}
	if pagination["page_size"] != float64(5) {
		t.Errorf("Expected page_size 5, got %v", pagination["page_size"])
	}
	if pagination["total_items"] != float64(25) {
		t.Errorf("Expected total_items 25, got %v", pagination["total_items"])
	}
}

func TestGetAllComputersHandler_RepositoryError(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	mockRepo.GetAllComputersPaginatedFunc = func(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error) {
		return nil, errors.New("database error")
	}

	req, _ := http.NewRequest("GET", "/computers", nil)
	rr := httptest.NewRecorder()

	handler.GetAllComputersHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// Test GetComputerHandler

func TestGetComputerHandler_Success(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computer := createTestComputer()
	computerID := computer.ID

	mockRepo.GetComputerByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Computer, error) {
		if id != computerID {
			t.Errorf("Expected computer ID %s, got %s", computerID, id)
		}
		return &computer, nil
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/computers/%s", computerID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": computerID.String()})
	rr := httptest.NewRecorder()

	handler.GetComputerHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response model.Computer
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.ID != computer.ID {
		t.Errorf("Expected computer ID %s, got %s", computer.ID, response.ID)
	}
	if response.ComputerName != computer.ComputerName {
		t.Errorf("Expected computer name %s, got %s", computer.ComputerName, response.ComputerName)
	}
}

func TestGetComputerHandler_InvalidUUID(t *testing.T) {
	handler, _, _ := createTestHandler()

	req, _ := http.NewRequest("GET", "/computers/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	rr := httptest.NewRecorder()

	handler.GetComputerHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !strings.Contains(response.Error, "Invalid UUID") {
		t.Errorf("Expected UUID error message, got %s", response.Error)
	}
}

func TestGetComputerHandler_NotFound(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computerID := uuid.New()
	mockRepo.GetComputerByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Computer, error) {
		return nil, repository.ErrComputerNotFound
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/computers/%s", computerID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": computerID.String()})
	rr := httptest.NewRecorder()

	handler.GetComputerHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rr.Code)
	}
}

// Test UpdateComputerHandler

func TestUpdateComputerHandler_Success(t *testing.T) {
	handler, mockRepo, mockNotifier := createTestHandler()

	computerID := uuid.New()
	computer := createTestComputer()
	computer.ComputerName = "UPDATED-001"

	mockRepo.UpdateComputerFunc = func(ctx context.Context, id uuid.UUID, c model.Computer) error {
		if id != computerID {
			t.Errorf("Expected computer ID %s, got %s", computerID, id)
		}
		if c.ComputerName != "UPDATED-001" {
			t.Errorf("Expected computer name UPDATED-001, got %s", c.ComputerName)
		}
		return nil
	}

	// Mock the async notification check
	mockRepo.GetComputersByEmployeeFunc = func(ctx context.Context, emp string) ([]model.Computer, error) {
		return []model.Computer{computer}, nil
	}

	// Ensure mockNotifier is used
	_ = mockNotifier

	req := createJSONRequest("PUT", fmt.Sprintf("/computers/%s", computerID), computer)
	req = mux.SetURLVars(req, map[string]string{"id": computerID.String()})
	rr := httptest.NewRecorder()

	handler.UpdateComputerHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.Message != "Computer updated successfully" {
		t.Errorf("Expected update success message, got %s", response.Message)
	}
}

// Test DeleteComputerHandler

func TestDeleteComputerHandler_Success(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computerID := uuid.New()

	mockRepo.DeleteComputerFunc = func(ctx context.Context, id uuid.UUID) error {
		if id != computerID {
			t.Errorf("Expected computer ID %s, got %s", computerID, id)
		}
		return nil
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/computers/%s", computerID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": computerID.String()})
	rr := httptest.NewRecorder()

	handler.DeleteComputerHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.Message != "Computer deleted successfully" {
		t.Errorf("Expected delete success message, got %s", response.Message)
	}
}

// Test GetEmployeeComputersHandler

func TestGetEmployeeComputersHandler_Success(t *testing.T) {
	handler, mockRepo, _ := createTestHandler()

	computers := []model.Computer{createTestComputer()}
	expectedResult := &repository.PaginatedResult{
		Items:      computers,
		TotalCount: 1,
	}

	mockRepo.GetComputersByEmployeePaginatedFunc = func(ctx context.Context, emp string, params repository.PaginationParams) (*repository.PaginatedResult, error) {
		if emp != "ABC" {
			t.Errorf("Expected employee ABC, got %s", emp)
		}
		if params.Offset != 0 || params.Limit != 10 {
			t.Errorf("Expected default pagination params, got offset: %d, limit: %d", params.Offset, params.Limit)
		}
		return expectedResult, nil
	}

	req, _ := http.NewRequest("GET", "/computers/employee/ABC", nil)
	req = mux.SetURLVars(req, map[string]string{"employee_abbreviation": "ABC"})
	rr := httptest.NewRecorder()

	handler.GetEmployeeComputersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if _, exists := response["computers"]; !exists {
		t.Error("Expected computers field in response")
	}
	if _, exists := response["pagination"]; !exists {
		t.Error("Expected pagination field in response")
	}
	if response["employee_abbreviation"] != "ABC" {
		t.Errorf("Expected employee_abbreviation ABC, got %v", response["employee_abbreviation"])
	}
}

func TestGetEmployeeComputersHandler_InvalidEmployee(t *testing.T) {
	handler, _, _ := createTestHandler()

	req, _ := http.NewRequest("GET", "/computers/employee/AB", nil) // Too short
	req = mux.SetURLVars(req, map[string]string{"employee_abbreviation": "AB"})
	rr := httptest.NewRecorder()

	handler.GetEmployeeComputersHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !strings.Contains(response.Error, "employee abbreviation must be exactly 3 characters long") {
		t.Errorf("Expected employee abbreviation error, got %s", response.Error)
	}
}

// Test HealthHandler

func TestHealthHandler_Success(t *testing.T) {
	handler, _, _ := createTestHandler()

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if response.Message != "Service is healthy" {
		t.Errorf("Expected health success message, got %s", response.Message)
	}
	if response.Data == nil {
		t.Error("Expected response data to be present")
	}
}

// Test checkAndNotify function (indirectly through async calls)

func TestCheckAndNotify_ThresholdExceeded(t *testing.T) {
	handler, mockRepo, mockNotifier := createTestHandler()

	// Create enough computers to exceed threshold
	computers := []model.Computer{
		createTestComputer(),
		createTestComputer(),
		createTestComputer(),
		createTestComputer(), // 4 computers > 3 threshold
	}

	mockRepo.GetComputersByEmployeeFunc = func(ctx context.Context, emp string) ([]model.Computer, error) {
		if emp != "ABC" {
			t.Errorf("Expected employee ABC, got %s", emp)
		}
		return computers, nil
	}

	// Call checkAndNotify directly
	handler.checkAndNotify("ABC")

	// Give some time for the async operation
	time.Sleep(100 * time.Millisecond)

	// Check that notification was sent
	if len(mockNotifier.NotificationsSent) == 0 {
		t.Error("Expected notification to be sent")
	} else {
		notification := mockNotifier.NotificationsSent[0]
		if notification.Level != "warning" {
			t.Errorf("Expected warning level, got %s", notification.Level)
		}
		if notification.EmployeeAbbreviation != "ABC" {
			t.Errorf("Expected employee ABC, got %s", notification.EmployeeAbbreviation)
		}
		if !strings.Contains(notification.Message, "4 computers") {
			t.Errorf("Expected message to contain '4 computers', got %s", notification.Message)
		}
	}
}

func TestCheckAndNotify_BelowThreshold(t *testing.T) {
	handler, mockRepo, mockNotifier := createTestHandler()

	computers := []model.Computer{createTestComputer()} // Only 1 computer < 3 threshold

	mockRepo.GetComputersByEmployeeFunc = func(ctx context.Context, emp string) ([]model.Computer, error) {
		return computers, nil
	}

	handler.checkAndNotify("ABC")

	time.Sleep(100 * time.Millisecond)

	// Check that no notification was sent
	if len(mockNotifier.NotificationsSent) > 0 {
		t.Error("Expected no notification to be sent for computers below threshold")
	}
}

func TestCheckAndNotify_EmptyEmployee(t *testing.T) {
	handler, mockRepo, mockNotifier := createTestHandler()

	// Track if repository method was called
	repoCalled := false
	mockRepo.GetComputersByEmployeeFunc = func(ctx context.Context, emp string) ([]model.Computer, error) {
		repoCalled = true
		return []model.Computer{}, nil
	}

	// Should return early without calling any methods
	handler.checkAndNotify("")

	time.Sleep(100 * time.Millisecond)

	if repoCalled {
		t.Error("Repository method should not be called for empty employee")
	}
	if len(mockNotifier.NotificationsSent) > 0 {
		t.Error("No notification should be sent for empty employee")
	}
}

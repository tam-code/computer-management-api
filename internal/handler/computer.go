package handler

import (
	"computer-management-api/internal/model"
	"computer-management-api/internal/notification"
	"computer-management-api/internal/repository"
	"computer-management-api/pkg/validation"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Constants for timeouts and validation
const (
	DefaultTimeout        = 10 * time.Second
	LongRunningTimeout    = 15 * time.Second
	NotificationTimeout   = 5 * time.Second
	EmployeeAbbrevLength  = 3
	MaxComputersThreshold = 3
)

// Error response structure for consistent JSON error responses
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Success response structure for consistent JSON success responses
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ComputerHandler handles the HTTP requests for computers.
type ComputerHandler struct {
	Repo     repository.ComputerRepository
	Notifier notification.Notifier
	Logger   *log.Logger

	// Helper components for cleaner code organization
	ErrorHandler   *ErrorHandler
	ResponseHelper *ResponseHelper
}

// NewComputerHandler creates a new ComputerHandler with dependencies and helpers
func NewComputerHandler(repo repository.ComputerRepository, notifier notification.Notifier, logger *log.Logger) *ComputerHandler {
	if logger == nil {
		logger = log.Default()
	}

	return &ComputerHandler{
		Repo:           repo,
		Notifier:       notifier,
		Logger:         logger,
		ErrorHandler:   NewErrorHandler(logger),
		ResponseHelper: NewResponseHelper(),
	}
}

// CreateComputerHandler handles the creation of a new computer.
func (h *ComputerHandler) CreateComputerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	var computer model.Computer
	if err := json.NewDecoder(r.Body).Decode(&computer); err != nil {
		h.ErrorHandler.HandleJSONDecodeError(w, err)
		return
	}

	// Validate input using validation package
	if validationErrors := validation.ValidateComputerInput(&computer); len(validationErrors) > 0 {
		// Convert []string to map[string]string for ErrorHandler
		errorMap := make(map[string]string)
		for i, err := range validationErrors {
			errorMap[fmt.Sprintf("error_%d", i)] = err
		}
		h.ErrorHandler.HandleValidationErrors(w, errorMap)
		return
	}

	// Generate ID if not provided
	if computer.ID == uuid.Nil {
		computer.ID = uuid.New()
	}

	// Create computer
	if err := h.Repo.CreateComputer(ctx, computer); err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "create")
		return
	}

	// Async notification (non-blocking)
	go h.checkAndNotify(computer.EmployeeAbbreviation)

	// Send success response with helper
	successData := h.ResponseHelper.CreateComputerSuccessData(computer.ID.String(), computer.MACAddress)
	h.ErrorHandler.SendSuccessResponse(w, http.StatusCreated, "Computer created successfully", successData)
}

// GetAllComputersHandler handles the retrieval of all computers with pagination.
func (h *ComputerHandler) GetAllComputersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, LongRunningTimeout)
	defer cancel()

	// Parse pagination parameters
	paginationParams := h.ResponseHelper.ParsePaginationParams(r)

	// Always use paginated endpoint for list operations
	result, err := h.Repo.GetAllComputersPaginated(ctx, repository.PaginationParams{
		Offset: paginationParams.Offset,
		Limit:  paginationParams.Limit,
	})
	if err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "retrieve")
		return
	}

	// Calculate pagination metadata
	paginationMeta := h.ResponseHelper.CalculatePaginationMeta(paginationParams, result.TotalCount)

	// Create paginated response
	responseData := h.ResponseHelper.CreatePaginatedListResponseData(result.Items, paginationMeta, map[string]interface{}{
		"computers": result.Items,
	})
	delete(responseData, "items") // Remove generic "items" key since we have "computers"

	h.ErrorHandler.SendJSONResponse(w, http.StatusOK, responseData)
}

// GetComputerHandler handles the retrieval of a single computer by ID.
func (h *ComputerHandler) GetComputerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	id, valid := h.ErrorHandler.ParseAndValidateUUID(w, vars["id"])
	if !valid {
		return
	}

	computer, err := h.Repo.GetComputerByID(ctx, id)
	if err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "retrieve")
		return
	}

	h.ErrorHandler.SendJSONResponse(w, http.StatusOK, computer)
}

// UpdateComputerHandler handles the update of a computer.
func (h *ComputerHandler) UpdateComputerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	id, valid := h.ErrorHandler.ParseAndValidateUUID(w, vars["id"])
	if !valid {
		return
	}

	var computer model.Computer
	if err := json.NewDecoder(r.Body).Decode(&computer); err != nil {
		h.ErrorHandler.HandleJSONDecodeError(w, err)
		return
	}

	// Validate input using validation package
	if validationErrors := validation.ValidateComputerInputForUpdate(&computer); len(validationErrors) > 0 {
		// Convert []string to map[string]string for ErrorHandler
		errorMap := make(map[string]string)
		for i, err := range validationErrors {
			errorMap[fmt.Sprintf("error_%d", i)] = err
		}
		h.ErrorHandler.HandleValidationErrors(w, errorMap)
		return
	}

	if err := h.Repo.UpdateComputer(ctx, id, computer); err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "update")
		return
	}

	// Async notification (non-blocking)
	go h.checkAndNotify(computer.EmployeeAbbreviation)

	// Send success response
	successData := h.ResponseHelper.CreateComputerSuccessData(id.String(), "")
	h.ErrorHandler.SendSuccessResponse(w, http.StatusOK, "Computer updated successfully", successData)
}

// DeleteComputerHandler handles the deletion of a computer.
func (h *ComputerHandler) DeleteComputerHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	id, valid := h.ErrorHandler.ParseAndValidateUUID(w, vars["id"])
	if !valid {
		return
	}

	if err := h.Repo.DeleteComputer(ctx, id); err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "delete")
		return
	}

	// Send success response instead of just 204
	successData := h.ResponseHelper.CreateComputerSuccessData(id.String(), "")
	h.ErrorHandler.SendSuccessResponse(w, http.StatusOK, "Computer deleted successfully", successData)
}

// GetEmployeeComputersHandler handles the retrieval of all computers for an employee with pagination.
func (h *ComputerHandler) GetEmployeeComputersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	employeeAbbreviation := vars["employee_abbreviation"]

	if err := validation.ValidateEmployeeAbbreviation(employeeAbbreviation); err != nil {
		h.ErrorHandler.HandleEmployeeAbbreviationError(w, err)
		return
	}

	// Parse pagination parameters
	paginationParams := h.ResponseHelper.ParsePaginationParams(r)

	// Always use paginated endpoint for list operations
	result, err := h.Repo.GetComputersByEmployeePaginated(ctx, employeeAbbreviation, repository.PaginationParams{
		Offset: paginationParams.Offset,
		Limit:  paginationParams.Limit,
	})
	if err != nil {
		h.ErrorHandler.HandleRepositoryError(w, err, "retrieve")
		return
	}

	// Calculate pagination metadata
	paginationMeta := h.ResponseHelper.CalculatePaginationMeta(paginationParams, result.TotalCount)

	// Create paginated response
	responseData := h.ResponseHelper.CreatePaginatedListResponseData(result.Items, paginationMeta, map[string]interface{}{
		"employee_abbreviation": employeeAbbreviation,
		"computers":             result.Items,
	})
	delete(responseData, "items") // Remove generic "items" key

	h.ErrorHandler.SendJSONResponse(w, http.StatusOK, responseData)
}

// checkAndNotify performs asynchronous notification checking
func (h *ComputerHandler) checkAndNotify(employeeAbbreviation string) {
	if employeeAbbreviation == "" {
		return
	}

	ctx, cancel := h.ResponseHelper.CreateRequestContext(&http.Request{}, NotificationTimeout)
	defer cancel()

	computers, err := h.Repo.GetComputersByEmployee(ctx, employeeAbbreviation)
	if err != nil {
		h.Logger.Printf("Failed to check employee computers for notification: %v", err)
		return
	}

	if len(computers) >= MaxComputersThreshold {
		notification := notification.Notification{
			Level:                notification.LevelWarning,
			EmployeeAbbreviation: employeeAbbreviation,
			Message:              fmt.Sprintf("Employee %s has %d computers assigned (threshold: %d)", employeeAbbreviation, len(computers), MaxComputersThreshold),
			Metadata: map[string]string{
				"computer_count": fmt.Sprintf("%d", len(computers)),
				"threshold":      fmt.Sprintf("%d", MaxComputersThreshold),
			},
		}

		if err := h.Notifier.SendNotification(notification); err != nil {
			h.Logger.Printf("Failed to send notification for employee %s: %v", employeeAbbreviation, err)
		} else {
			h.Logger.Printf("Notification sent for employee %s (%d computers)", employeeAbbreviation, len(computers))
		}
	}
}

// HealthHandler provides a health check endpoint
func (h *ComputerHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	healthData := h.ResponseHelper.CreateHealthCheckData()
	h.ErrorHandler.SendSuccessResponse(w, http.StatusOK, "Service is healthy", healthData)
}

// RemoveComputerFromEmployeeHandler handles removing a computer from an employee.
// This endpoint unassigns a computer from a specific employee by setting the employee_abbreviation to empty.
func (h *ComputerHandler) RemoveComputerFromEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	employeeAbbreviation := vars["employee_abbreviation"]
	computerIDStr := vars["computer_id"]

	// Validate employee abbreviation
	if err := validation.ValidateEmployeeAbbreviation(employeeAbbreviation); err != nil {
		h.ErrorHandler.HandleEmployeeAbbreviationError(w, err)
		return
	}

	// Parse and validate computer ID
	computerID, valid := h.ErrorHandler.ParseAndValidateUUID(w, computerIDStr)
	if !valid {
		return
	}

	// Remove computer from employee
	if err := h.Repo.RemoveComputerFromEmployee(ctx, computerID, employeeAbbreviation); err != nil {
		if err.Error() == fmt.Sprintf("computer not found or not assigned to employee %s", employeeAbbreviation) {
			h.ErrorHandler.SendErrorResponse(w, http.StatusNotFound, "Computer not found or not assigned to this employee", "COMPUTER_NOT_FOUND", nil)
			return
		}
		h.ErrorHandler.HandleRepositoryError(w, err, "remove computer from employee")
		return
	}

	// Send success response
	successData := h.ResponseHelper.CreateComputerSuccessData(computerID.String(), employeeAbbreviation)
	h.ErrorHandler.SendSuccessResponse(w, http.StatusOK, "Computer successfully removed from employee", successData)
}

// AssignComputerToEmployeeHandler handles assigning a computer to an employee.
// This endpoint assigns a computer to a specific employee by setting the employee_abbreviation.
func (h *ComputerHandler) AssignComputerToEmployeeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ResponseHelper.CreateRequestContext(r, DefaultTimeout)
	defer cancel()

	vars := mux.Vars(r)
	employeeAbbreviation := vars["employee_abbreviation"]
	computerIDStr := vars["computer_id"]

	// Validate employee abbreviation
	if err := validation.ValidateEmployeeAbbreviation(employeeAbbreviation); err != nil {
		h.ErrorHandler.HandleEmployeeAbbreviationError(w, err)
		return
	}

	// Parse and validate computer ID
	computerID, valid := h.ErrorHandler.ParseAndValidateUUID(w, computerIDStr)
	if !valid {
		return
	}

	// Assign computer to employee
	if err := h.Repo.AssignComputerToEmployee(ctx, computerID, employeeAbbreviation); err != nil {
		if err.Error() == fmt.Sprintf("computer with ID %s not found", computerID) {
			h.ErrorHandler.SendErrorResponse(w, http.StatusNotFound, "Computer not found", "COMPUTER_NOT_FOUND", nil)
			return
		}
		h.ErrorHandler.HandleRepositoryError(w, err, "assign computer to employee")
		return
	}

	// Async notification check (non-blocking)
	go h.checkAndNotify(employeeAbbreviation)

	// Send success response
	successData := h.ResponseHelper.CreateComputerSuccessData(computerID.String(), employeeAbbreviation)
	h.ErrorHandler.SendSuccessResponse(w, http.StatusOK, "Computer successfully assigned to employee", successData)
}

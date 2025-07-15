package handler

import (
	"net/http"
)

// ComputerHandlerInterface defines the contract for computer HTTP handlers.
// This interface enables easy testing, mocking, and dependency injection.
type ComputerHandlerInterface interface {
	// Computer CRUD operations
	CreateComputerHandler(w http.ResponseWriter, r *http.Request)
	GetAllComputersHandler(w http.ResponseWriter, r *http.Request)
	GetComputerHandler(w http.ResponseWriter, r *http.Request)
	UpdateComputerHandler(w http.ResponseWriter, r *http.Request)
	DeleteComputerHandler(w http.ResponseWriter, r *http.Request)

	// Employee-specific operations
	GetEmployeeComputersHandler(w http.ResponseWriter, r *http.Request)
	RemoveComputerFromEmployeeHandler(w http.ResponseWriter, r *http.Request)
	AssignComputerToEmployeeHandler(w http.ResponseWriter, r *http.Request)

	// Health and monitoring
	HealthHandler(w http.ResponseWriter, r *http.Request)
}

// Ensure ComputerHandler implements ComputerHandlerInterface at compile time
var _ ComputerHandlerInterface = (*ComputerHandler)(nil)

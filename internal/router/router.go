package router

import (
	"computer-management-api/internal/config"
	"computer-management-api/internal/handler"
	"computer-management-api/internal/middleware"

	"github.com/gorilla/mux"
)

// NewRouter creates a new router and sets up the routes with security middleware.
func NewRouter(h handler.ComputerHandlerInterface, cfg *config.Config) *mux.Router {
	r := mux.NewRouter()

	// Initialize security middleware
	securityMW := middleware.NewSecurityMiddleware(&cfg.Security)

	// Apply global middleware in order
	r.Use(securityMW.SecurityHeaders)
	r.Use(securityMW.CORS)
	r.Use(securityMW.TrustedProxy)
	r.Use(securityMW.RateLimit)
	r.Use(securityMW.RequestTimeout)

	api := r.PathPrefix("/api/v1").Subrouter()

	// Computer CRUD operations
	api.HandleFunc("/computers", h.CreateComputerHandler).Methods("POST")
	api.HandleFunc("/computers", h.GetAllComputersHandler).Methods("GET")
	api.HandleFunc("/computers/{id}", h.GetComputerHandler).Methods("GET")
	api.HandleFunc("/computers/{id}", h.UpdateComputerHandler).Methods("PUT")
	api.HandleFunc("/computers/{id}", h.DeleteComputerHandler).Methods("DELETE")

	// Employee-specific operations
	api.HandleFunc("/employees/{employee_abbreviation}/computers", h.GetEmployeeComputersHandler).Methods("GET")
	api.HandleFunc("/employees/{employee_abbreviation}/computers/{computer_id}", h.RemoveComputerFromEmployeeHandler).Methods("DELETE")
	api.HandleFunc("/employees/{employee_abbreviation}/computers/{computer_id}", h.AssignComputerToEmployeeHandler).Methods("PUT")

	// Health check
	api.HandleFunc("/health", h.HealthHandler).Methods("GET")

	return r
}

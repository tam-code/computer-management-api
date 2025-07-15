package repository

import (
	"computer-management-api/internal/model"
	"computer-management-api/pkg/validation"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Custom errors for better error handling
var (
	ErrComputerNotFound = errors.New("computer not found")
	ErrDuplicateMAC     = errors.New("computer with this MAC address already exists")
	ErrInvalidMACFormat = errors.New("invalid MAC address format")
)

// PaginationParams holds pagination parameters for repository queries
type PaginationParams struct {
	Offset int
	Limit  int
}

// PaginatedResult holds paginated query results
type PaginatedResult struct {
	Items      []model.Computer
	TotalCount int
}

// ComputerRepository is an interface for interacting with computer data.
type ComputerRepository interface {
	CreateComputer(ctx context.Context, computer model.Computer) error
	GetAllComputers(ctx context.Context) ([]model.Computer, error)
	GetAllComputersPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult, error)
	GetComputerByMAC(ctx context.Context, macAddress string) (*model.Computer, error)
	GetComputerByID(ctx context.Context, id uuid.UUID) (*model.Computer, error)
	UpdateComputer(ctx context.Context, id uuid.UUID, computer model.Computer) error
	DeleteComputer(ctx context.Context, id uuid.UUID) error
	GetComputersByEmployee(ctx context.Context, employeeAbbreviation string) ([]model.Computer, error)
	GetComputersByEmployeePaginated(ctx context.Context, employeeAbbreviation string, params PaginationParams) (*PaginatedResult, error)
	ComputerExists(ctx context.Context, macAddress string) (bool, error)
	RemoveComputerFromEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error
	AssignComputerToEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error
}

// computerRepository is the concrete implementation of the ComputerRepository interface.

type computerRepository struct {
	DB *sql.DB
}

// NewComputerRepository creates a new ComputerRepository.
func NewComputerRepository(db *sql.DB) ComputerRepository {
	return &computerRepository{DB: db}
}

// CreateComputer adds a new computer to the database.
func (r *computerRepository) CreateComputer(ctx context.Context, computer model.Computer) error {
	// Set timeout for the operation
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Validate and normalize MAC address
	normalizedMAC, err := validation.ValidateMAC(computer.MACAddress)
	if err != nil {
		return ErrInvalidMACFormat
	}
	computer.MACAddress = normalizedMAC

	// Validate IP address
	if err := validation.ValidateIP(computer.IPAddress); err != nil {
		return fmt.Errorf("invalid IP address: %w", err)
	}

	query := `
		INSERT INTO computers (id, mac_address, computer_name, ip_address, employee_abbreviation, description)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = r.DB.ExecContext(ctx, query,
		computer.ID,
		computer.MACAddress,
		computer.ComputerName,
		computer.IPAddress,
		computer.EmployeeAbbreviation,
		computer.Description,
	)

	if err != nil {
		// Check for unique constraint violations (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			if strings.Contains(err.Error(), "computers_mac_address_key") || strings.Contains(err.Error(), "computers_pkey") {
				return fmt.Errorf("%w: %s", ErrDuplicateMAC, computer.MACAddress)
			}
		}
		return fmt.Errorf("failed to create computer: %w", err)
	}

	return nil
}

// GetAllComputers retrieves all computers from the database.
func (r *computerRepository) GetAllComputers(ctx context.Context) ([]model.Computer, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		ORDER BY computer_name`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query computers: %w", err)
	}
	defer rows.Close()

	var computers []model.Computer
	for rows.Next() {
		var c model.Computer
		if err := rows.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan computer: %w", err)
		}
		computers = append(computers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return computers, nil
}

// GetAllComputersPaginated retrieves all computers with pagination support.
func (r *computerRepository) GetAllComputersPaginated(ctx context.Context, params PaginationParams) (*PaginatedResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		ORDER BY computer_name
		OFFSET $1 LIMIT $2`

	rows, err := r.DB.QueryContext(ctx, query, params.Offset, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query computers: %w", err)
	}
	defer rows.Close()

	var computers []model.Computer
	for rows.Next() {
		var c model.Computer
		if err := rows.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan computer: %w", err)
		}
		computers = append(computers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Get total count of computers for pagination
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM computers`
	err = r.DB.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count of computers: %w", err)
	}

	return &PaginatedResult{
		Items:      computers,
		TotalCount: totalCount,
	}, nil
}

// ComputerExists checks if a computer with the given MAC address already exists
func (r *computerRepository) ComputerExists(ctx context.Context, macAddress string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT EXISTS(SELECT 1 FROM computers WHERE mac_address = $1)`

	var exists bool
	err := r.DB.QueryRowContext(ctx, query, macAddress).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check computer existence: %w", err)
	}

	return exists, nil
}

// GetComputerByMAC retrieves a computer by its MAC address (primary key).
func (r *computerRepository) GetComputerByMAC(ctx context.Context, macAddress string) (*model.Computer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		WHERE mac_address = $1`

	row := r.DB.QueryRowContext(ctx, query, macAddress)

	var c model.Computer
	if err := row.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrComputerNotFound
		}
		return nil, fmt.Errorf("failed to get computer by MAC: %w", err)
	}
	return &c, nil
}

// GetComputerByID retrieves a single computer by its ID.
func (r *computerRepository) GetComputerByID(ctx context.Context, id uuid.UUID) (*model.Computer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		WHERE id = $1`

	row := r.DB.QueryRowContext(ctx, query, id)

	var c model.Computer
	if err := row.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrComputerNotFound
		}
		return nil, fmt.Errorf("failed to get computer by ID: %w", err)
	}
	return &c, nil
}

// UpdateComputer updates a computer in the database.
func (r *computerRepository) UpdateComputer(ctx context.Context, id uuid.UUID, computer model.Computer) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE computers
		SET mac_address = $1, computer_name = $2, ip_address = $3, employee_abbreviation = $4, description = $5
		WHERE id = $6`

	result, err := r.DB.ExecContext(ctx, query,
		computer.MACAddress,
		computer.ComputerName,
		computer.IPAddress,
		computer.EmployeeAbbreviation,
		computer.Description,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update computer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrComputerNotFound
	}

	return nil
}

// DeleteComputer deletes a computer from the database.
func (r *computerRepository) DeleteComputer(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `DELETE FROM computers WHERE id = $1`

	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete computer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrComputerNotFound
	}

	return nil
}

// GetComputersByEmployee retrieves all computers for a specific employee.
func (r *computerRepository) GetComputersByEmployee(ctx context.Context, employeeAbbreviation string) ([]model.Computer, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Leverage the index on employee_abbreviation for fast lookup
	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		WHERE employee_abbreviation = $1 
		ORDER BY computer_name`

	rows, err := r.DB.QueryContext(ctx, query, employeeAbbreviation)
	if err != nil {
		return nil, fmt.Errorf("failed to query computers by employee: %w", err)
	}
	defer rows.Close()

	var computers []model.Computer
	for rows.Next() {
		var c model.Computer
		if err := rows.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan computer: %w", err)
		}
		computers = append(computers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return computers, nil
}

// GetComputersByEmployeePaginated retrieves all computers for a specific employee with pagination support.
func (r *computerRepository) GetComputersByEmployeePaginated(ctx context.Context, employeeAbbreviation string, params PaginationParams) (*PaginatedResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at 
		FROM computers 
		WHERE employee_abbreviation = $1 
		ORDER BY computer_name
		OFFSET $2 LIMIT $3`

	rows, err := r.DB.QueryContext(ctx, query, employeeAbbreviation, params.Offset, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query computers by employee: %w", err)
	}
	defer rows.Close()

	var computers []model.Computer
	for rows.Next() {
		var c model.Computer
		if err := rows.Scan(&c.ID, &c.MACAddress, &c.ComputerName, &c.IPAddress, &c.EmployeeAbbreviation, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan computer: %w", err)
		}
		computers = append(computers, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Get total count of computers for pagination
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM computers WHERE employee_abbreviation = $1`
	err = r.DB.QueryRowContext(ctx, countQuery, employeeAbbreviation).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count of computers: %w", err)
	}

	return &PaginatedResult{
		Items:      computers,
		TotalCount: totalCount,
	}, nil
}

// RemoveComputerFromEmployee removes a computer from an employee by setting employee_abbreviation to empty.
// This method verifies that the computer is currently assigned to the specified employee before removing it.
func (r *computerRepository) RemoveComputerFromEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// First, verify that the computer exists and is assigned to the specified employee
	query := `
		UPDATE computers 
		SET employee_abbreviation = '', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND employee_abbreviation = $2`

	result, err := r.DB.ExecContext(ctx, query, computerID, employeeAbbreviation)
	if err != nil {
		return fmt.Errorf("failed to remove computer from employee: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Computer either doesn't exist or is not assigned to this employee
		return fmt.Errorf("computer not found or not assigned to employee %s", employeeAbbreviation)
	}

	return nil
}

// AssignComputerToEmployee assigns a computer to a specific employee by updating the employee_abbreviation field.
// This method verifies that the computer exists and optionally handles reassignment scenarios.
func (r *computerRepository) AssignComputerToEmployee(ctx context.Context, computerID uuid.UUID, employeeAbbreviation string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// First, verify that the computer exists
	query := `
		UPDATE computers 
		SET employee_abbreviation = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	result, err := r.DB.ExecContext(ctx, query, employeeAbbreviation, computerID)
	if err != nil {
		return fmt.Errorf("failed to assign computer to employee: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("computer with ID %s not found", computerID)
	}

	return nil
}

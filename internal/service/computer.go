package service

import (
	"computer-management-api/internal/model"
	"computer-management-api/internal/repository"
	"computer-management-api/pkg/errors"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// ComputerService handles business logic for computer operations
type ComputerService struct {
	repo     repository.ComputerRepository
	notifier NotificationService
	logger   *log.Logger
}

// NotificationService interface for sending notifications
type NotificationService interface {
	SendComputerNotification(ctx context.Context, notification ComputerNotification) error
}

// ComputerNotification represents a notification about computer operations
type ComputerNotification struct {
	Type                 NotificationType
	EmployeeAbbreviation string
	ComputerCount        int
	ComputerName         string
	Message              string
	Metadata             map[string]string
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeThresholdExceeded NotificationType = "threshold_exceeded"
	NotificationTypeComputerCreated   NotificationType = "computer_created"
	NotificationTypeComputerUpdated   NotificationType = "computer_updated"
	NotificationTypeComputerDeleted   NotificationType = "computer_deleted"
)

// Business rules constants
const (
	MaxComputersPerEmployee = 3
	MinEmployeeAbbrevLength = 2
	MaxEmployeeAbbrevLength = 10
)

// NewComputerService creates a new computer service
func NewComputerService(repo repository.ComputerRepository, notifier NotificationService, logger *log.Logger) *ComputerService {
	if logger == nil {
		logger = log.Default()
	}
	return &ComputerService{
		repo:     repo,
		notifier: notifier,
		logger:   logger,
	}
}

// CreateComputer creates a new computer with business logic validation
func (s *ComputerService) CreateComputer(ctx context.Context, computer model.Computer) (*model.Computer, error) {
	// Validate business rules
	if err := s.validateComputerForCreation(ctx, computer); err != nil {
		return nil, err
	}

	// Generate ID if not provided
	if computer.ID == uuid.Nil {
		computer.ID = uuid.New()
	}

	// Normalize MAC address
	computer.MACAddress = strings.ToUpper(strings.ReplaceAll(computer.MACAddress, "-", ":"))

	// Create the computer
	if err := s.repo.CreateComputer(ctx, computer); err != nil {
		return nil, errors.DatabaseError("failed to create computer", err)
	}

	// Check if we need to send a notification
	go s.checkAndNotifyThreshold(computer.EmployeeAbbreviation)

	// Send creation notification
	if computer.EmployeeAbbreviation != "" {
		go s.sendCreationNotification(computer)
	}

	s.logger.Printf("Computer created successfully: ID=%s, MAC=%s, Employee=%s",
		computer.ID, computer.MACAddress, computer.EmployeeAbbreviation)

	return &computer, nil
}

// GetAllComputers retrieves computers with pagination
func (s *ComputerService) GetAllComputers(ctx context.Context, params repository.PaginationParams) (*repository.PaginatedResult, error) {
	result, err := s.repo.GetAllComputersPaginated(ctx, params)
	if err != nil {
		return nil, errors.DatabaseError("failed to retrieve computers", err)
	}

	s.logger.Printf("Retrieved %d computers (offset %d, limit %d)",
		len(result.Items), params.Offset, params.Limit)

	return result, nil
}

// GetComputerByID retrieves a computer by its ID
func (s *ComputerService) GetComputerByID(ctx context.Context, id uuid.UUID) (*model.Computer, error) {
	computer, err := s.repo.GetComputerByID(ctx, id)
	if err != nil {
		if err == repository.ErrComputerNotFound {
			return nil, errors.NotFoundError("computer")
		}
		return nil, errors.DatabaseError("failed to retrieve computer", err)
	}

	return computer, nil
}

// UpdateComputer updates an existing computer
func (s *ComputerService) UpdateComputer(ctx context.Context, id uuid.UUID, updates model.Computer) (*model.Computer, error) {
	// Check if computer exists
	existing, err := s.repo.GetComputerByID(ctx, id)
	if err != nil {
		if err == repository.ErrComputerNotFound {
			return nil, errors.NotFoundError("computer")
		}
		return nil, errors.DatabaseError("failed to retrieve computer for update", err)
	}

	// Validate business rules for update
	if err := s.validateComputerForUpdate(ctx, id, updates); err != nil {
		return nil, err
	}

	// Preserve ID and timestamps
	updates.ID = id
	updates.CreatedAt = existing.CreatedAt

	// Normalize MAC address if provided
	if updates.MACAddress != "" {
		updates.MACAddress = strings.ToUpper(strings.ReplaceAll(updates.MACAddress, "-", ":"))
	}

	// Update the computer
	if err := s.repo.UpdateComputer(ctx, id, updates); err != nil {
		return nil, errors.DatabaseError("failed to update computer", err)
	}

	// Get updated computer
	updated, err := s.repo.GetComputerByID(ctx, id)
	if err != nil {
		return nil, errors.DatabaseError("failed to retrieve updated computer", err)
	}

	// Send update notification if employee changed
	if existing.EmployeeAbbreviation != updates.EmployeeAbbreviation {
		go s.sendUpdateNotification(*existing, *updated)
		if updates.EmployeeAbbreviation != "" {
			go s.checkAndNotifyThreshold(updates.EmployeeAbbreviation)
		}
	}

	s.logger.Printf("Computer updated successfully: ID=%s", id)

	return updated, nil
}

// DeleteComputer deletes a computer
func (s *ComputerService) DeleteComputer(ctx context.Context, id uuid.UUID) error {
	// Check if computer exists
	computer, err := s.repo.GetComputerByID(ctx, id)
	if err != nil {
		if err == repository.ErrComputerNotFound {
			return errors.NotFoundError("computer")
		}
		return errors.DatabaseError("failed to retrieve computer for deletion", err)
	}

	// Delete the computer
	if err := s.repo.DeleteComputer(ctx, id); err != nil {
		return errors.DatabaseError("failed to delete computer", err)
	}

	// Send deletion notification
	if computer.EmployeeAbbreviation != "" {
		go s.sendDeletionNotification(*computer)
	}

	s.logger.Printf("Computer deleted successfully: ID=%s", id)

	return nil
}

// GetComputersByEmployee retrieves computers for a specific employee with pagination
func (s *ComputerService) GetComputersByEmployee(ctx context.Context, employeeAbbrev string, params repository.PaginationParams) (*repository.PaginatedResult, error) {
	if err := s.validateEmployeeAbbreviation(employeeAbbrev); err != nil {
		return nil, err
	}

	result, err := s.repo.GetComputersByEmployeePaginated(ctx, employeeAbbrev, params)
	if err != nil {
		return nil, errors.DatabaseError("failed to retrieve employee computers", err)
	}

	s.logger.Printf("Retrieved %d computers for employee %s (offset %d, limit %d)",
		len(result.Items), employeeAbbrev, params.Offset, params.Limit)

	return result, nil
}

// Business logic validation methods

func (s *ComputerService) validateComputerForCreation(ctx context.Context, computer model.Computer) error {
	// Validate required fields
	if computer.MACAddress == "" {
		return errors.ValidationError("MAC address is required")
	}
	if computer.ComputerName == "" {
		return errors.ValidationError("computer name is required")
	}
	if computer.IPAddress == "" {
		return errors.ValidationError("IP address is required")
	}

	// Validate MAC address format
	if !isValidMACAddress(computer.MACAddress) {
		return errors.ValidationError("invalid MAC address format")
	}

	// Validate IP address format
	if !isValidIPAddress(computer.IPAddress) {
		return errors.ValidationError("invalid IP address format")
	}

	// Validate employee abbreviation
	if computer.EmployeeAbbreviation != "" {
		if err := s.validateEmployeeAbbreviation(computer.EmployeeAbbreviation); err != nil {
			return err
		}
	}

	// Check if MAC address already exists
	exists, err := s.repo.ComputerExists(ctx, computer.MACAddress)
	if err != nil {
		return errors.DatabaseError("failed to check computer existence", err)
	}
	if exists {
		return errors.AlreadyExistsError("computer with this MAC address")
	}

	// Check employee computer count if employee is specified
	if computer.EmployeeAbbreviation != "" {
		computers, err := s.repo.GetComputersByEmployee(ctx, computer.EmployeeAbbreviation)
		if err != nil {
			return errors.DatabaseError("failed to check employee computer count", err)
		}
		if len(computers) >= MaxComputersPerEmployee {
			return errors.ValidationError(fmt.Sprintf("employee already has the maximum number of computers (%d)", MaxComputersPerEmployee))
		}
	}

	return nil
}

func (s *ComputerService) validateComputerForUpdate(ctx context.Context, id uuid.UUID, computer model.Computer) error {
	// Validate fields that are being updated
	if computer.MACAddress != "" {
		if !isValidMACAddress(computer.MACAddress) {
			return errors.ValidationError("invalid MAC address format")
		}

		// Check if MAC address is already used by another computer
		existing, err := s.repo.GetComputerByMAC(ctx, computer.MACAddress)
		if err != nil && err != repository.ErrComputerNotFound {
			return errors.DatabaseError("failed to check MAC address uniqueness", err)
		}
		if existing != nil && existing.ID != id {
			return errors.AlreadyExistsError("computer with this MAC address")
		}
	}

	if computer.IPAddress != "" && !isValidIPAddress(computer.IPAddress) {
		return errors.ValidationError("invalid IP address format")
	}

	if computer.EmployeeAbbreviation != "" {
		if err := s.validateEmployeeAbbreviation(computer.EmployeeAbbreviation); err != nil {
			return err
		}
	}

	return nil
}

func (s *ComputerService) validateEmployeeAbbreviation(abbrev string) error {
	if len(abbrev) < MinEmployeeAbbrevLength {
		return errors.ValidationError(fmt.Sprintf("employee abbreviation must be at least %d characters", MinEmployeeAbbrevLength))
	}
	if len(abbrev) > MaxEmployeeAbbrevLength {
		return errors.ValidationError(fmt.Sprintf("employee abbreviation must be at most %d characters", MaxEmployeeAbbrevLength))
	}
	return nil
}

// Notification methods

func (s *ComputerService) checkAndNotifyThreshold(employeeAbbrev string) {
	if employeeAbbrev == "" {
		return
	}

	ctx := context.Background()
	computers, err := s.repo.GetComputersByEmployee(ctx, employeeAbbrev)
	if err != nil {
		s.logger.Printf("Failed to check employee computers for notification: %v", err)
		return
	}

	if len(computers) >= MaxComputersPerEmployee {
		notification := ComputerNotification{
			Type:                 NotificationTypeThresholdExceeded,
			EmployeeAbbreviation: employeeAbbrev,
			ComputerCount:        len(computers),
			Message:              fmt.Sprintf("Employee %s has %d computers assigned (threshold: %d)", employeeAbbrev, len(computers), MaxComputersPerEmployee),
			Metadata: map[string]string{
				"threshold": fmt.Sprintf("%d", MaxComputersPerEmployee),
				"count":     fmt.Sprintf("%d", len(computers)),
			},
		}

		if err := s.notifier.SendComputerNotification(ctx, notification); err != nil {
			s.logger.Printf("Failed to send threshold notification for employee %s: %v", employeeAbbrev, err)
		} else {
			s.logger.Printf("Threshold notification sent for employee %s (%d computers)", employeeAbbrev, len(computers))
		}
	}
}

func (s *ComputerService) sendCreationNotification(computer model.Computer) {
	ctx := context.Background()
	notification := ComputerNotification{
		Type:                 NotificationTypeComputerCreated,
		EmployeeAbbreviation: computer.EmployeeAbbreviation,
		ComputerName:         computer.ComputerName,
		Message:              fmt.Sprintf("Computer %s created for employee %s", computer.ComputerName, computer.EmployeeAbbreviation),
		Metadata: map[string]string{
			"computer_id":   computer.ID.String(),
			"computer_name": computer.ComputerName,
			"mac_address":   computer.MACAddress,
		},
	}

	if err := s.notifier.SendComputerNotification(ctx, notification); err != nil {
		s.logger.Printf("Failed to send creation notification: %v", err)
	}
}

func (s *ComputerService) sendUpdateNotification(old, new model.Computer) {
	ctx := context.Background()
	notification := ComputerNotification{
		Type:                 NotificationTypeComputerUpdated,
		EmployeeAbbreviation: new.EmployeeAbbreviation,
		ComputerName:         new.ComputerName,
		Message:              fmt.Sprintf("Computer %s updated (reassigned from %s to %s)", new.ComputerName, old.EmployeeAbbreviation, new.EmployeeAbbreviation),
		Metadata: map[string]string{
			"computer_id":   new.ID.String(),
			"computer_name": new.ComputerName,
			"old_employee":  old.EmployeeAbbreviation,
			"new_employee":  new.EmployeeAbbreviation,
		},
	}

	if err := s.notifier.SendComputerNotification(ctx, notification); err != nil {
		s.logger.Printf("Failed to send update notification: %v", err)
	}
}

func (s *ComputerService) sendDeletionNotification(computer model.Computer) {
	ctx := context.Background()
	notification := ComputerNotification{
		Type:                 NotificationTypeComputerDeleted,
		EmployeeAbbreviation: computer.EmployeeAbbreviation,
		ComputerName:         computer.ComputerName,
		Message:              fmt.Sprintf("Computer %s deleted (was assigned to %s)", computer.ComputerName, computer.EmployeeAbbreviation),
		Metadata: map[string]string{
			"computer_id":   computer.ID.String(),
			"computer_name": computer.ComputerName,
			"mac_address":   computer.MACAddress,
		},
	}

	if err := s.notifier.SendComputerNotification(ctx, notification); err != nil {
		s.logger.Printf("Failed to send deletion notification: %v", err)
	}
}

// Validation helper functions (these should ideally be moved to a validation package)

func isValidMACAddress(mac string) bool {
	// Implement MAC address validation logic
	// For now, just check basic format
	mac = strings.ReplaceAll(mac, "-", ":")
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
	}
	return true
}

func isValidIPAddress(ip string) bool {
	// Implement IP address validation logic
	// For now, just check basic format
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	return true
}

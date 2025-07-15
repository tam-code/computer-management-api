package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"computer-management-api/internal/model"
)

// MAC address validation constants
const (
	MACAddressLength = 17 // XX:XX:XX:XX:XX:XX format
)

// Employee validation constants
const (
	EmployeeAbbrevExactLength = 3 // Employee abbreviation must be exactly 3 characters
)

// ValidateMAC validates a MAC address format and returns normalized version
func ValidateMAC(mac string) (string, error) {
	// Remove any spaces and convert to uppercase
	normalized := strings.ToUpper(strings.ReplaceAll(mac, " ", ""))

	// Convert hyphens to colons for consistency
	normalized = strings.ReplaceAll(normalized, "-", ":")

	// Check if it's a valid MAC address format
	macRegex := regexp.MustCompile(`^([0-9A-F]{2}:){5}([0-9A-F]{2})$`)
	if !macRegex.MatchString(normalized) {
		return "", fmt.Errorf("invalid MAC address format: %s", mac)
	}

	return normalized, nil
}

// ValidateIP validates an IP address format (IPv4 or IPv6)
func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	return nil
}

// ValidateEmployeeAbbreviation validates employee abbreviation
func ValidateEmployeeAbbreviation(abbrev string) error {
	if abbrev == "" {
		return nil // Optional field
	}

	if len(abbrev) != EmployeeAbbrevExactLength {
		return fmt.Errorf("employee abbreviation must be exactly %d characters long", EmployeeAbbrevExactLength)
	}

	// Only allow alphanumeric characters
	alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumeric.MatchString(abbrev) {
		return fmt.Errorf("employee abbreviation can only contain alphanumeric characters")
	}

	return nil
}

// ValidateComputerName validates computer name
func ValidateComputerName(name string) error {
	if name == "" {
		return fmt.Errorf("computer name is required")
	}

	if len(name) > 255 {
		return fmt.Errorf("computer name cannot exceed 255 characters")
	}

	return nil
}

// ValidateRequired checks if a string field is not empty
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateComputerInput validates all required fields for creating a new computer
func ValidateComputerInput(computer *model.Computer) []string {
	var errors []string

	// Validate computer name
	if err := ValidateComputerName(computer.ComputerName); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate MAC address and get normalized version
	normalizedMAC, err := ValidateMAC(computer.MACAddress)
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		computer.MACAddress = normalizedMAC // Update with normalized version
	}

	// Validate IP address
	if err := ValidateIP(computer.IPAddress); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate employee abbreviation (optional field)
	if err := ValidateEmployeeAbbreviation(computer.EmployeeAbbreviation); err != nil {
		errors = append(errors, err.Error())
	}

	return errors
}

// ValidateComputerInputForUpdate validates fields for updating a computer (similar to create but may have different rules)
func ValidateComputerInputForUpdate(computer *model.Computer) []string {
	// For now, use the same validation as create
	// This can be modified later if update has different requirements
	return ValidateComputerInput(computer)
}

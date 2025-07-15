package model

import (
	"time"

	"github.com/google/uuid"
)

// Computer represents a computer in the system.
type Computer struct {
	ID                   uuid.UUID `json:"id"`
	MACAddress           string    `json:"mac_address"`
	ComputerName         string    `json:"computer_name"`
	IPAddress            string    `json:"ip_address"`
	EmployeeAbbreviation string    `json:"employee_abbreviation,omitempty"`
	Description          string    `json:"description,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

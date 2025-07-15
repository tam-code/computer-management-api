package validation

import (
	"computer-management-api/internal/model"
	"testing"
)

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		name        string
		mac         string
		expectError bool
		expected    string
	}{
		{
			name:        "Valid MAC with colons",
			mac:         "AA:BB:CC:DD:EE:FF",
			expectError: false,
			expected:    "AA:BB:CC:DD:EE:FF",
		},
		{
			name:        "Valid MAC with hyphens",
			mac:         "AA-BB-CC-DD-EE-FF",
			expectError: false,
			expected:    "AA:BB:CC:DD:EE:FF",
		},
		{
			name:        "Valid MAC lowercase",
			mac:         "aa:bb:cc:dd:ee:ff",
			expectError: false,
			expected:    "AA:BB:CC:DD:EE:FF",
		},
		{
			name:        "Invalid MAC too short",
			mac:         "AA:BB:CC:DD:EE",
			expectError: true,
		},
		{
			name:        "Invalid MAC too long",
			mac:         "AA:BB:CC:DD:EE:FF:GG",
			expectError: true,
		},
		{
			name:        "Invalid MAC characters",
			mac:         "ZZ:BB:CC:DD:EE:FF",
			expectError: true,
		},
		{
			name:        "Invalid MAC format",
			mac:         "invalid-mac",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateMAC(tt.mac)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for MAC %s, but got none", tt.mac)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for MAC %s: %v", tt.mac, err)
				}
				if result != tt.expected {
					t.Errorf("Expected normalized MAC %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		expectError bool
	}{
		{
			name:        "Valid IPv4",
			ip:          "192.168.1.1",
			expectError: false,
		},
		{
			name:        "Valid IPv6",
			ip:          "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expectError: false,
		},
		{
			name:        "Invalid IP",
			ip:          "256.256.256.256",
			expectError: true,
		},
		{
			name:        "Invalid IP format",
			ip:          "not-an-ip",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for IP %s, but got none", tt.ip)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for IP %s: %v", tt.ip, err)
				}
			}
		})
	}
}

func TestValidateEmployeeAbbreviation(t *testing.T) {
	tests := []struct {
		name        string
		abbrev      string
		expectError bool
	}{
		{
			name:        "Valid 3 character abbreviation",
			abbrev:      "ABC",
			expectError: false,
		},
		{
			name:        "Valid 3 character with numbers",
			abbrev:      "AB1",
			expectError: false,
		},
		{
			name:        "Empty abbreviation (allowed)",
			abbrev:      "",
			expectError: false,
		},
		{
			name:        "Too short (2 characters)",
			abbrev:      "AB",
			expectError: true,
		},
		{
			name:        "Too long (4 characters)",
			abbrev:      "ABCD",
			expectError: true,
		},
		{
			name:        "Invalid characters",
			abbrev:      "AB-",
			expectError: true,
		},
		{
			name:        "With space",
			abbrev:      "A B",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmployeeAbbreviation(tt.abbrev)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for abbreviation '%s', but got none", tt.abbrev)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for abbreviation '%s': %v", tt.abbrev, err)
				}
			}
		})
	}
}

func TestValidateComputerName(t *testing.T) {
	tests := []struct {
		name         string
		computerName string
		expectError  bool
	}{
		{
			name:         "Valid computer name",
			computerName: "TEST-001",
			expectError:  false,
		},
		{
			name:         "Empty computer name",
			computerName: "",
			expectError:  true,
		},
		{
			name:         "Very long computer name",
			computerName: string(make([]byte, 256)), // 256 characters
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComputerName(tt.computerName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for computer name '%s', but got none", tt.computerName)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for computer name '%s': %v", tt.computerName, err)
				}
			}
		})
	}
}

func TestValidateComputerInput(t *testing.T) {
	tests := []struct {
		name           string
		computer       model.Computer
		expectedErrors int
	}{
		{
			name: "Valid computer",
			computer: model.Computer{
				ComputerName:         "TEST-001",
				MACAddress:           "AA:BB:CC:DD:EE:FF",
				IPAddress:            "192.168.1.1",
				EmployeeAbbreviation: "ABC",
			},
			expectedErrors: 0,
		},
		{
			name: "Invalid MAC address",
			computer: model.Computer{
				ComputerName:         "TEST-001",
				MACAddress:           "invalid-mac",
				IPAddress:            "192.168.1.1",
				EmployeeAbbreviation: "ABC",
			},
			expectedErrors: 1,
		},
		{
			name: "Multiple validation errors",
			computer: model.Computer{
				ComputerName:         "",
				MACAddress:           "invalid-mac",
				IPAddress:            "invalid-ip",
				EmployeeAbbreviation: "ABCD",
			},
			expectedErrors: 4,
		},
		{
			name: "Valid computer with empty employee abbreviation",
			computer: model.Computer{
				ComputerName: "TEST-001",
				MACAddress:   "AA:BB:CC:DD:EE:FF",
				IPAddress:    "192.168.1.1",
			},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateComputerInput(&tt.computer)

			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}
		})
	}
}

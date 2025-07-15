package integration

import (
	"computer-management-api/internal/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AssignComputerToEmployee tests the new endpoint for assigning computers to employees
func TestIntegration_AssignComputerToEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := setupIntegrationTest(t)
	defer teardownIntegrationTest(t, suite)

	t.Run("Assign_Computer_To_Employee_Success", func(t *testing.T) {
		// Step 1: Create a computer without an employee assignment
		computer := model.Computer{
			MACAddress:           "AA:BB:CC:DD:EE:33",
			ComputerName:         "Test-Computer-Assign",
			IPAddress:            "192.168.1.200",
			EmployeeAbbreviation: "", // No initial assignment
			Description:          "Computer for assignment test",
		}

		// Create the computer
		req := createJSONRequest("POST", "/api/v1/computers", computer)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse map[string]interface{}
		parseJSONResponse(t, resp, &createResponse)

		// Extract the created computer ID
		data, ok := createResponse["data"].(map[string]interface{})
		require.True(t, ok)
		createdIDStr, ok := data["id"].(string)
		require.True(t, ok)
		createdID, err := uuid.Parse(createdIDStr)
		require.NoError(t, err)

		// Step 2: Assign the computer to an employee
		targetEmployee := "ABC"
		req = createJSONRequest("PUT", fmt.Sprintf("/api/v1/employees/%s/computers/%s", targetEmployee, createdID), nil)
		resp = httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var assignResponse map[string]interface{}
		parseJSONResponse(t, resp, &assignResponse)

		assert.Equal(t, "Computer successfully assigned to employee", assignResponse["message"])
		assert.NotNil(t, assignResponse["data"])

		// Step 3: Verify the computer is now assigned to the employee
		req = createJSONRequest("GET", fmt.Sprintf("/api/v1/employees/%s/computers", targetEmployee), nil)
		resp = httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var getResponse map[string]interface{}
		parseJSONResponse(t, resp, &getResponse)

		computers, ok := getResponse["computers"].([]interface{})
		require.True(t, ok)
		assert.Len(t, computers, 1, "Employee should have exactly 1 computer after assignment")

		// Verify it's the correct computer
		assignedComputer := computers[0].(map[string]interface{})
		assert.Equal(t, createdIDStr, assignedComputer["id"])
		assert.Equal(t, computer.ComputerName, assignedComputer["computer_name"])
		assert.Equal(t, targetEmployee, assignedComputer["employee_abbreviation"])
	})

	t.Run("Reassign_Computer_To_Different_Employee", func(t *testing.T) {
		// Step 1: Create a computer assigned to employee "XYZ"
		computer := model.Computer{
			MACAddress:           "AA:BB:CC:DD:EE:44",
			ComputerName:         "Test-Computer-Reassign",
			IPAddress:            "192.168.1.201",
			EmployeeAbbreviation: "XYZ",
			Description:          "Computer for reassignment test",
		}

		// Create the computer
		req := createJSONRequest("POST", "/api/v1/computers", computer)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusCreated, resp.Code)

		var createResponse map[string]interface{}
		parseJSONResponse(t, resp, &createResponse)

		// Extract the created computer ID
		data, ok := createResponse["data"].(map[string]interface{})
		require.True(t, ok)
		createdIDStr, ok := data["id"].(string)
		require.True(t, ok)
		createdID, err := uuid.Parse(createdIDStr)
		require.NoError(t, err)

		// Step 2: Reassign the computer to a different employee
		newEmployee := "DEF"
		req = createJSONRequest("PUT", fmt.Sprintf("/api/v1/employees/%s/computers/%s", newEmployee, createdID), nil)
		resp = httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var assignResponse map[string]interface{}
		parseJSONResponse(t, resp, &assignResponse)

		assert.Equal(t, "Computer successfully assigned to employee", assignResponse["message"])
	})

	t.Run("Assign_Computer_Not_Found", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := createJSONRequest("PUT", fmt.Sprintf("/api/v1/employees/TST/computers/%s", nonExistentID), nil)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)

		var errorResponse map[string]interface{}
		parseJSONResponse(t, resp, &errorResponse)

		assert.Contains(t, errorResponse["error"].(string), "Computer not found")
	})

	t.Run("Assign_Computer_Invalid_UUID", func(t *testing.T) {
		req := createJSONRequest("PUT", "/api/v1/employees/TST/computers/invalid-uuid", nil)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var errorResponse map[string]interface{}
		parseJSONResponse(t, resp, &errorResponse)

		assert.Contains(t, errorResponse["error"].(string), "Invalid UUID format")
	})

	t.Run("Assign_Computer_Invalid_Employee_Abbreviation", func(t *testing.T) {
		computerID := uuid.New()

		req := createJSONRequest("PUT", fmt.Sprintf("/api/v1/employees/INVALID/computers/%s", computerID), nil)
		resp := httptest.NewRecorder()
		suite.Router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var errorResponse map[string]interface{}
		parseJSONResponse(t, resp, &errorResponse)

		assert.Contains(t, errorResponse["error"].(string), "employee abbreviation must be exactly 3 characters")
	})
}

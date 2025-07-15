package integration

import (
	"computer-management-api/internal/config"
	"computer-management-api/internal/database"
	"computer-management-api/internal/model"
	"computer-management-api/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestIntegration_DatabaseOperations tests database operations end-to-end
func TestIntegration_DatabaseOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Setup
	cfg := loadTestConfig(t)
	db := initTestDatabase(t, cfg)
	defer func() {
		cleanDatabase(t, db)
		db.Close()
	}()

	repo := repository.NewComputerRepository(db)
	ctx := context.Background()

	// Test data
	testComputer := model.Computer{
		ID:                   uuid.New(),
		MACAddress:           "AA:BB:CC:DD:EE:FF",
		ComputerName:         "DB-TEST-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "TDB",
		Description:          "Database integration test",
	}

	t.Run("Create and Retrieve Computer", func(t *testing.T) {
		// Create
		err := repo.CreateComputer(ctx, testComputer)
		if err != nil {
			t.Fatalf("Failed to create computer: %v", err)
		}

		// Retrieve by ID
		retrieved, err := repo.GetComputerByID(ctx, testComputer.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve computer by ID: %v", err)
		}

		// Verify data
		if retrieved.MACAddress != testComputer.MACAddress {
			t.Errorf("Expected MAC %s, got %s", testComputer.MACAddress, retrieved.MACAddress)
		}
		if retrieved.ComputerName != testComputer.ComputerName {
			t.Errorf("Expected name %s, got %s", testComputer.ComputerName, retrieved.ComputerName)
		}
		if retrieved.EmployeeAbbreviation != testComputer.EmployeeAbbreviation {
			t.Errorf("Expected employee %s, got %s", testComputer.EmployeeAbbreviation, retrieved.EmployeeAbbreviation)
		}

		// Retrieve by MAC
		retrievedByMAC, err := repo.GetComputerByMAC(ctx, testComputer.MACAddress)
		if err != nil {
			t.Fatalf("Failed to retrieve computer by MAC: %v", err)
		}
		if retrievedByMAC.ID != testComputer.ID {
			t.Errorf("Expected ID %s, got %s", testComputer.ID, retrievedByMAC.ID)
		}
	})

	t.Run("Update Computer", func(t *testing.T) {
		// Update data
		testComputer.ComputerName = "DB-TEST-UPDATED"
		testComputer.IPAddress = "192.168.1.101"
		testComputer.Description = "Updated database test"

		err := repo.UpdateComputer(ctx, testComputer.ID, testComputer)
		if err != nil {
			t.Fatalf("Failed to update computer: %v", err)
		}

		// Verify update
		updated, err := repo.GetComputerByID(ctx, testComputer.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated computer: %v", err)
		}

		if updated.ComputerName != "DB-TEST-UPDATED" {
			t.Errorf("Expected updated name 'DB-TEST-UPDATED', got %s", updated.ComputerName)
		}
		if updated.IPAddress != "192.168.1.101" {
			t.Errorf("Expected updated IP '192.168.1.101', got %s", updated.IPAddress)
		}
	})

	t.Run("Get Computers by Employee", func(t *testing.T) {
		// Create additional computer for same employee
		testComputer2 := model.Computer{
			ID:                   uuid.New(),
			MACAddress:           "BB:CC:DD:EE:FF:AA",
			ComputerName:         "DB-TEST-002",
			IPAddress:            "192.168.1.102",
			EmployeeAbbreviation: "TDB", // Same employee
			Description:          "Second database test computer",
		}

		err := repo.CreateComputer(ctx, testComputer2)
		if err != nil {
			t.Fatalf("Failed to create second computer: %v", err)
		}

		// Get computers by employee
		computers, err := repo.GetComputersByEmployee(ctx, "TDB")
		if err != nil {
			t.Fatalf("Failed to get computers by employee: %v", err)
		}

		if len(computers) != 2 {
			t.Errorf("Expected 2 computers for employee TDB, got %d", len(computers))
		}

		// Test paginated version
		paginatedResult, err := repo.GetComputersByEmployeePaginated(ctx, "TDB", repository.PaginationParams{
			Offset: 0,
			Limit:  1,
		})
		if err != nil {
			t.Fatalf("Failed to get paginated computers by employee: %v", err)
		}

		if len(paginatedResult.Items) != 1 {
			t.Errorf("Expected 1 computer in paginated result, got %d", len(paginatedResult.Items))
		}
		if paginatedResult.TotalCount != 2 {
			t.Errorf("Expected total count 2, got %d", paginatedResult.TotalCount)
		}
	})

	t.Run("Get All Computers Paginated", func(t *testing.T) {
		result, err := repo.GetAllComputersPaginated(ctx, repository.PaginationParams{
			Offset: 0,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("Failed to get all computers paginated: %v", err)
		}

		if len(result.Items) != 2 {
			t.Errorf("Expected 2 computers total, got %d", len(result.Items))
		}
		if result.TotalCount != 2 {
			t.Errorf("Expected total count 2, got %d", result.TotalCount)
		}
	})

	t.Run("Delete Computer", func(t *testing.T) {
		err := repo.DeleteComputer(ctx, testComputer.ID)
		if err != nil {
			t.Fatalf("Failed to delete computer: %v", err)
		}

		// Verify deletion
		_, err = repo.GetComputerByID(ctx, testComputer.ID)
		if err == nil {
			t.Error("Expected error when retrieving deleted computer")
		}
		if err != repository.ErrComputerNotFound {
			t.Errorf("Expected ErrComputerNotFound, got %v", err)
		}
	})
}

func TestIntegration_DatabaseConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database constraint test in short mode")
	}

	// Setup
	cfg := loadTestConfig(t)
	db := initTestDatabase(t, cfg)
	defer func() {
		cleanDatabase(t, db)
		db.Close()
	}()

	repo := repository.NewComputerRepository(db)
	ctx := context.Background()

	t.Run("Duplicate MAC Address Constraint", func(t *testing.T) {
		computer1 := model.Computer{
			ID:           uuid.New(),
			MACAddress:   "AA:BB:CC:DD:EE:FF",
			ComputerName: "CONSTRAINT-TEST-1",
			IPAddress:    "192.168.1.1",
		}

		computer2 := model.Computer{
			ID:           uuid.New(),
			MACAddress:   "AA:BB:CC:DD:EE:FF", // Same MAC
			ComputerName: "CONSTRAINT-TEST-2",
			IPAddress:    "192.168.1.2",
		}

		// Create first computer
		err := repo.CreateComputer(ctx, computer1)
		if err != nil {
			t.Fatalf("Failed to create first computer: %v", err)
		}

		// Try to create second computer with same MAC
		err = repo.CreateComputer(ctx, computer2)
		if err == nil {
			t.Error("Expected error when creating computer with duplicate MAC")
		}
		if !errors.Is(err, repository.ErrDuplicateMAC) {
			t.Errorf("Expected ErrDuplicateMAC, got %v", err)
		}
	})

	t.Run("Database Connection Timeout", func(t *testing.T) {
		// Test with a very short timeout
		shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		computer := model.Computer{
			ID:           uuid.New(),
			MACAddress:   "CC:DD:EE:FF:AA:BB",
			ComputerName: "TIMEOUT-TEST",
			IPAddress:    "192.168.1.3",
		}

		err := repo.CreateComputer(shortCtx, computer)
		if err == nil {
			t.Error("Expected timeout error")
		}
		// Check if it's a context deadline exceeded error
		if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Logf("Got error: %v (type: %T)", err, err)
			// Don't fail the test as timeout behavior can vary
		}
	})
}

func TestIntegration_DatabaseTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database transaction test in short mode")
	}

	// Setup
	cfg := loadTestConfig(t)
	db := initTestDatabase(t, cfg)
	defer func() {
		cleanDatabase(t, db)
		db.Close()
	}()

	ctx := context.Background()

	t.Run("Manual Transaction Test", func(t *testing.T) {
		// Start transaction
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Insert computer within transaction
		computer := model.Computer{
			ID:           uuid.New(),
			MACAddress:   "TX:AA:BB:CC:DD:EE",
			ComputerName: "TRANSACTION-TEST",
			IPAddress:    "192.168.1.200",
		}

		query := `
			INSERT INTO computers (id, mac_address, computer_name, ip_address, employee_abbreviation, description)
			VALUES ($1, $2, $3, $4, $5, $6)`

		_, err = tx.ExecContext(ctx, query,
			computer.ID,
			computer.MACAddress,
			computer.ComputerName,
			computer.IPAddress,
			computer.EmployeeAbbreviation,
			computer.Description,
		)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		// Rollback transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify that the computer was not actually inserted
		repo := repository.NewComputerRepository(db)
		_, err = repo.GetComputerByID(ctx, computer.ID)
		if err == nil {
			t.Error("Computer should not exist after rollback")
		}
		if err != repository.ErrComputerNotFound {
			t.Errorf("Expected ErrComputerNotFound after rollback, got %v", err)
		}
	})
}

func TestIntegration_DatabasePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database performance test in short mode")
	}

	// Setup
	cfg := loadTestConfig(t)
	db := initTestDatabase(t, cfg)
	defer func() {
		cleanDatabase(t, db)
		db.Close()
	}()

	repo := repository.NewComputerRepository(db)
	ctx := context.Background()

	t.Run("Bulk Insert Performance", func(t *testing.T) {
		start := time.Now()

		// Create 50 computers to test performance
		for i := 0; i < 50; i++ {
			computer := model.Computer{
				ID:           uuid.New(),
				MACAddress:   fmt.Sprintf("AA:BB:CC:DD:EE:%02X", i),
				ComputerName: fmt.Sprintf("PERF-TEST-%03d", i),
				IPAddress:    fmt.Sprintf("192.168.1.%d", i+10),
			}

			err := repo.CreateComputer(ctx, computer)
			if err != nil {
				t.Fatalf("Failed to create computer %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		t.Logf("Created 50 computers in %v (%.2f computers/sec)", duration, float64(50)/duration.Seconds())

		// Test retrieval performance
		start = time.Now()
		result, err := repo.GetAllComputersPaginated(ctx, repository.PaginationParams{
			Offset: 0,
			Limit:  100,
		})
		if err != nil {
			t.Fatalf("Failed to retrieve computers: %v", err)
		}

		retrievalDuration := time.Since(start)
		t.Logf("Retrieved %d computers in %v", len(result.Items), retrievalDuration)

		if len(result.Items) != 50 {
			t.Errorf("Expected 50 computers, got %d", len(result.Items))
		}
	})
}

// Helper function to check if database is available
func isDatabaseAvailable(cfg *config.Config) bool {
	db, err := database.InitDB(cfg)
	if err != nil {
		return false
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return db.PingContext(ctx) == nil
}

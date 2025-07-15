package repository

import (
	"computer-management-api/internal/model"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t testing.TB) (*sql.DB, sqlmock.Sqlmock, ComputerRepository) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewComputerRepository(db)
	return db, mock, repo
}

func TestNewComputerRepository(t *testing.T) {
	db, _, _ := setupTestDB(t)
	defer db.Close()

	repo := NewComputerRepository(db)
	assert.NotNil(t, repo)
}

func TestCreateComputer_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computer := model.Computer{
		ID:                   uuid.New(),
		MACAddress:           "AA:BB:CC:DD:EE:FF",
		ComputerName:         "TEST-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "JDO",
		Description:          "Test computer",
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO computers (id, mac_address, computer_name, ip_address, employee_abbreviation, description) VALUES ($1, $2, $3, $4, $5, $6)`)).
		WithArgs(computer.ID, computer.MACAddress, computer.ComputerName, computer.IPAddress, computer.EmployeeAbbreviation, computer.Description).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err := repo.CreateComputer(ctx, computer)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateComputer_InvalidMAC(t *testing.T) {
	db, _, repo := setupTestDB(t)
	defer db.Close()

	computer := model.Computer{
		MACAddress:   "invalid-mac",
		ComputerName: "TEST-001",
		IPAddress:    "192.168.1.100",
	}

	ctx := context.Background()
	err := repo.CreateComputer(ctx, computer)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidMACFormat))
}

func TestCreateComputer_InvalidIP(t *testing.T) {
	db, _, repo := setupTestDB(t)
	defer db.Close()

	computer := model.Computer{
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		ComputerName: "TEST-001",
		IPAddress:    "invalid-ip",
	}

	ctx := context.Background()
	err := repo.CreateComputer(ctx, computer)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IP address")
}

func TestCreateComputer_DuplicateMAC(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computer := model.Computer{
		ID:           uuid.New(),
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		ComputerName: "TEST-001",
		IPAddress:    "192.168.1.100",
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO computers`)).
		WillReturnError(errors.New(`pq: duplicate key value violates unique constraint "computers_pkey"`))

	ctx := context.Background()
	err := repo.CreateComputer(ctx, computer)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateMAC))
}

func TestGetAllComputers_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	expectedComputers := []model.Computer{
		{
			ID:                   uuid.New(),
			MACAddress:           "AA:BB:CC:DD:EE:FF",
			ComputerName:         "TEST-001",
			IPAddress:            "192.168.1.100",
			EmployeeAbbreviation: "JDO",
			Description:          "Test computer 1",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   uuid.New(),
			MACAddress:           "AA:BB:CC:DD:EE:FE",
			ComputerName:         "TEST-002",
			IPAddress:            "192.168.1.101",
			EmployeeAbbreviation: "JAN",
			Description:          "Test computer 2",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "mac_address", "computer_name", "ip_address", "employee_abbreviation", "description", "created_at", "updated_at"})
	for _, computer := range expectedComputers {
		rows.AddRow(computer.ID, computer.MACAddress, computer.ComputerName, computer.IPAddress, computer.EmployeeAbbreviation, computer.Description, computer.CreatedAt, computer.UpdatedAt)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers ORDER BY computer_name`)).
		WillReturnRows(rows)

	ctx := context.Background()
	computers, err := repo.GetAllComputers(ctx)

	assert.NoError(t, err)
	assert.Len(t, computers, 2)
	assert.Equal(t, expectedComputers[0].MACAddress, computers[0].MACAddress)
	assert.Equal(t, expectedComputers[1].MACAddress, computers[1].MACAddress)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllComputers_QueryError(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers ORDER BY computer_name`)).
		WillReturnError(errors.New("database error"))

	ctx := context.Background()
	computers, err := repo.GetAllComputers(ctx)

	assert.Error(t, err)
	assert.Nil(t, computers)
	assert.Contains(t, err.Error(), "failed to query computers")
}

func TestGetComputerByID_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()
	now := time.Now()
	expectedComputer := model.Computer{
		ID:                   computerID,
		MACAddress:           "AA:BB:CC:DD:EE:FF",
		ComputerName:         "TEST-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "JDO",
		Description:          "Test computer",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	rows := sqlmock.NewRows([]string{"id", "mac_address", "computer_name", "ip_address", "employee_abbreviation", "description", "created_at", "updated_at"}).
		AddRow(expectedComputer.ID, expectedComputer.MACAddress, expectedComputer.ComputerName, expectedComputer.IPAddress, expectedComputer.EmployeeAbbreviation, expectedComputer.Description, expectedComputer.CreatedAt, expectedComputer.UpdatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE id = $1`)).
		WithArgs(computerID).
		WillReturnRows(rows)

	ctx := context.Background()
	computer, err := repo.GetComputerByID(ctx, computerID)

	assert.NoError(t, err)
	assert.NotNil(t, computer)
	assert.Equal(t, expectedComputer.ID, computer.ID)
	assert.Equal(t, expectedComputer.MACAddress, computer.MACAddress)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetComputerByID_NotFound(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE id = $1`)).
		WithArgs(computerID).
		WillReturnError(sql.ErrNoRows)

	ctx := context.Background()
	computer, err := repo.GetComputerByID(ctx, computerID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrComputerNotFound))
	assert.Nil(t, computer)
}

func TestGetComputerByMAC_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	macAddress := "AA:BB:CC:DD:EE:FF"
	now := time.Now()
	expectedComputer := model.Computer{
		ID:                   uuid.New(),
		MACAddress:           macAddress,
		ComputerName:         "TEST-001",
		IPAddress:            "192.168.1.100",
		EmployeeAbbreviation: "JDO",
		Description:          "Test computer",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	rows := sqlmock.NewRows([]string{"id", "mac_address", "computer_name", "ip_address", "employee_abbreviation", "description", "created_at", "updated_at"}).
		AddRow(expectedComputer.ID, expectedComputer.MACAddress, expectedComputer.ComputerName, expectedComputer.IPAddress, expectedComputer.EmployeeAbbreviation, expectedComputer.Description, expectedComputer.CreatedAt, expectedComputer.UpdatedAt)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE mac_address = $1`)).
		WithArgs(macAddress).
		WillReturnRows(rows)

	ctx := context.Background()
	computer, err := repo.GetComputerByMAC(ctx, macAddress)

	assert.NoError(t, err)
	assert.NotNil(t, computer)
	assert.Equal(t, expectedComputer.MACAddress, computer.MACAddress)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetComputerByMAC_NotFound(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	macAddress := "AA:BB:CC:DD:EE:FF"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE mac_address = $1`)).
		WithArgs(macAddress).
		WillReturnError(sql.ErrNoRows)

	ctx := context.Background()
	computer, err := repo.GetComputerByMAC(ctx, macAddress)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrComputerNotFound))
	assert.Nil(t, computer)
}

func TestUpdateComputer_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()
	computer := model.Computer{
		MACAddress:           "AA:BB:CC:DD:EE:FF",
		ComputerName:         "UPDATED-001",
		IPAddress:            "192.168.1.200",
		EmployeeAbbreviation: "UPD",
		Description:          "Updated computer",
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE computers SET mac_address = $1, computer_name = $2, ip_address = $3, employee_abbreviation = $4, description = $5 WHERE id = $6`)).
		WithArgs(computer.MACAddress, computer.ComputerName, computer.IPAddress, computer.EmployeeAbbreviation, computer.Description, computerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err := repo.UpdateComputer(ctx, computerID, computer)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateComputer_NotFound(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()
	computer := model.Computer{
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		ComputerName: "UPDATED-001",
		IPAddress:    "192.168.1.200",
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE computers SET mac_address = $1, computer_name = $2, ip_address = $3, employee_abbreviation = $4, description = $5 WHERE id = $6`)).
		WithArgs(computer.MACAddress, computer.ComputerName, computer.IPAddress, computer.EmployeeAbbreviation, computer.Description, computerID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	err := repo.UpdateComputer(ctx, computerID, computer)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrComputerNotFound))
}

func TestDeleteComputer_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM computers WHERE id = $1`)).
		WithArgs(computerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err := repo.DeleteComputer(ctx, computerID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComputer_NotFound(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	computerID := uuid.New()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM computers WHERE id = $1`)).
		WithArgs(computerID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	err := repo.DeleteComputer(ctx, computerID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrComputerNotFound))
}

func TestGetComputersByEmployee_Success(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	employeeAbbr := "JDO"
	now := time.Now()
	expectedComputers := []model.Computer{
		{
			ID:                   uuid.New(),
			MACAddress:           "AA:BB:CC:DD:EE:FF",
			ComputerName:         "JDO-001",
			IPAddress:            "192.168.1.100",
			EmployeeAbbreviation: employeeAbbr,
			Description:          "John's laptop",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   uuid.New(),
			MACAddress:           "AA:BB:CC:DD:EE:FE",
			ComputerName:         "JDO-002",
			IPAddress:            "192.168.1.101",
			EmployeeAbbreviation: employeeAbbr,
			Description:          "John's desktop",
			CreatedAt:            now,
			UpdatedAt:            now,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "mac_address", "computer_name", "ip_address", "employee_abbreviation", "description", "created_at", "updated_at"})
	for _, computer := range expectedComputers {
		rows.AddRow(computer.ID, computer.MACAddress, computer.ComputerName, computer.IPAddress, computer.EmployeeAbbreviation, computer.Description, computer.CreatedAt, computer.UpdatedAt)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE employee_abbreviation = $1 ORDER BY computer_name`)).
		WithArgs(employeeAbbr).
		WillReturnRows(rows)

	ctx := context.Background()
	computers, err := repo.GetComputersByEmployee(ctx, employeeAbbr)

	assert.NoError(t, err)
	assert.Len(t, computers, 2)
	assert.Equal(t, employeeAbbr, computers[0].EmployeeAbbreviation)
	assert.Equal(t, employeeAbbr, computers[1].EmployeeAbbreviation)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetComputersByEmployee_Empty(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	employeeAbbr := "XXX"

	rows := sqlmock.NewRows([]string{"id", "mac_address", "computer_name", "ip_address", "employee_abbreviation", "description", "created_at", "updated_at"})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers WHERE employee_abbreviation = $1 ORDER BY computer_name`)).
		WithArgs(employeeAbbr).
		WillReturnRows(rows)

	ctx := context.Background()
	computers, err := repo.GetComputersByEmployee(ctx, employeeAbbr)

	assert.NoError(t, err)
	assert.Len(t, computers, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestComputerExists_True(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	macAddress := "AA:BB:CC:DD:EE:FF"

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM computers WHERE mac_address = $1)`)).
		WithArgs(macAddress).
		WillReturnRows(rows)

	ctx := context.Background()
	exists, err := repo.ComputerExists(ctx, macAddress)

	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestComputerExists_False(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	macAddress := "AA:BB:CC:DD:EE:FF"

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM computers WHERE mac_address = $1)`)).
		WithArgs(macAddress).
		WillReturnRows(rows)

	ctx := context.Background()
	exists, err := repo.ComputerExists(ctx, macAddress)

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContextTimeout(t *testing.T) {
	db, mock, repo := setupTestDB(t)
	defer db.Close()

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure context times out
	time.Sleep(1 * time.Millisecond)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mac_address, computer_name, ip_address, employee_abbreviation, description, created_at, updated_at FROM computers ORDER BY computer_name`)).
		WillDelayFor(100 * time.Millisecond).
		WillReturnError(context.DeadlineExceeded)

	computers, err := repo.GetAllComputers(ctx)

	assert.Error(t, err)
	assert.Nil(t, computers)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// Benchmark tests
func BenchmarkCreateComputer(b *testing.B) {
	db, mock, repo := setupTestDB(b)
	defer db.Close()

	computer := model.Computer{
		ID:           uuid.New(),
		MACAddress:   "AA:BB:CC:DD:EE:FF",
		ComputerName: "BENCH-001",
		IPAddress:    "192.168.1.100",
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO computers`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = repo.CreateComputer(ctx, computer)
	}
}

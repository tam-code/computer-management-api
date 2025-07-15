# Integration Tests

This directory contains comprehensive integration tests for the Computer Management API. These tests validate the entire application stack from HTTP endpoints to database operations.

## Overview

The integration tests are organized into several test files:

- `api_test.go` - End-to-end HTTP API testing
- `database_test.go` - Database operations and constraints testing

## Test Categories

### API Integration Tests (`api_test.go`)

Tests the complete HTTP API endpoints including:

- **CRUD Operations**: Create, Read, Update, Delete computers
- **Validation**: Input validation and error handling
- **Pagination**: List endpoints with pagination
- **Employee Operations**: Employee-specific computer queries
- **Error Scenarios**: 404s, validation errors, duplicate data
- **Health Checks**: Service health endpoint

### Database Integration Tests (`database_test.go`)

Tests database layer including:

- **Repository Operations**: Direct database CRUD operations
- **Database Constraints**: Unique constraints, foreign keys
- **Transactions**: Transaction rollback behavior
- **Performance**: Bulk operations and query performance
- **Connection Handling**: Timeouts and connection management

## Prerequisites

### Required Services

1. **PostgreSQL Database** (runs on port 5452)
   - Used for persistent data storage
   - Automatically configured via docker-compose

2. **Notification Service** (runs on port 8081)
   - External notification service for testing integrations
   - Mock service is used in tests

### Environment Setup

The tests require the following environment variables (or use defaults):

```bash
# Test Database Configuration
TEST_DB_HOST=localhost
TEST_DB_PORT=5452
TEST_DB_USER=postgres
TEST_DB_PASSWORD=postgres
TEST_DB_NAME=postgres
```

## Running Integration Tests

### Quick Start

1. **Setup test environment**:
   ```bash
   ./scripts/setup-integration-tests.sh
   ```

2. **Run all integration tests**:
   ```bash
   make test-integration
   ```

3. **Cleanup**:
   ```bash
   make test-integration-teardown
   ```

### Specific Test Categories

```bash
# Run only API integration tests
make test-integration-api

# Run only database integration tests
make test-integration-db

# Run with verbose output
go test -v ./internal/integration/

# Run specific test
go test -run TestIntegration_ComputerCRUD ./internal/integration/
```

### Manual Setup

If you prefer manual setup:

1. **Start services**:
   ```bash
   docker-compose up -d db notification
   ```

2. **Wait for readiness**:
   ```bash
   # Check database
   docker-compose exec db pg_isready -U postgres
   
   # Check notification service
   curl http://localhost:8081/health
   ```

3. **Run tests**:
   ```bash
   go test ./internal/integration/
   ```

4. **Cleanup**:
   ```bash
   docker-compose down
   ```

## Test Features

### Automatic Database Cleanup

- Each test suite automatically cleans the database before and after execution
- No test data pollution between test runs
- Safe to run tests multiple times

### Mock Services

- Notification service is mocked for predictable test behavior
- Database transactions are used for isolation
- No external dependencies beyond the test database

### Comprehensive Coverage

The integration tests cover:

- ✅ **Happy Path**: All successful operations
- ✅ **Error Handling**: Various error scenarios
- ✅ **Edge Cases**: Boundary conditions and limits
- ✅ **Performance**: Basic performance characteristics
- ✅ **Data Integrity**: Database constraints and validation

## Test Data

### Test Computer Records

Tests use standardized test data:

```go
testComputer := model.Computer{
    MACAddress:           "AA:BB:CC:DD:EE:FF",
    ComputerName:         "TEST-INTEGRATION-001",
    IPAddress:            "192.168.1.100",
    EmployeeAbbreviation: "ABC",
    Description:          "Integration test computer",
}
```

### Naming Conventions

- Computer names: `TEST-*`, `DB-TEST-*`, `PERF-TEST-*`
- MAC addresses: Test-specific patterns
- Employee abbreviations: `ABC`, `DEF`, `TDB` (Test Database)

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```
   Failed to connect to test database
   ```
   **Solution**: Ensure PostgreSQL is running on port 5452
   ```bash
   docker-compose up -d db
   ```

2. **Tests Skip in Short Mode**
   ```
   Skipping integration test in short mode
   ```
   **Solution**: Don't use `-short` flag for integration tests
   ```bash
   go test ./internal/integration/  # Not: go test -short
   ```

3. **Port Already in Use**
   ```
   Port 5452 already in use
   ```
   **Solution**: Stop existing services or change ports in docker-compose.yml

4. **Permission Denied**
   ```
   Permission denied when accessing scripts
   ```
   **Solution**: Make scripts executable
   ```bash
   chmod +x scripts/*.sh
   ```

### Debugging

Enable verbose output for detailed test information:

```bash
go test -v ./internal/integration/
```

Check service logs:

```bash
# Database logs
docker-compose logs db

# Notification service logs
docker-compose logs notification
```

## CI/CD Integration

### GitHub Actions / CI Systems

```yaml
# Example CI configuration
- name: Setup Integration Tests
  run: ./scripts/setup-integration-tests.sh

- name: Run Integration Tests
  run: make test-integration

- name: Cleanup
  run: make test-integration-teardown
  if: always()
```

### Local Development

Add to your development workflow:

```bash
# Full development cycle with integration tests
make ci                    # Runs: deps, lint, test-coverage, build
make test-integration     # Additional integration testing
```

## Performance Benchmarks

The integration tests include basic performance benchmarks:

- **Bulk Insert**: Tests creating multiple records
- **Query Performance**: Tests retrieval operations
- **Pagination**: Tests large dataset handling

Run performance tests specifically:

```bash
go test -run TestIntegration_DatabasePerformance ./internal/integration/
```

## Contributing

When adding new integration tests:

1. **Follow naming convention**: `TestIntegration_FeatureName`
2. **Use test helpers**: Leverage existing setup/teardown functions
3. **Clean up data**: Ensure tests don't affect each other
4. **Document test purpose**: Add clear comments for complex tests
5. **Test both success and failure**: Cover happy path and error cases

### Example Test Structure

```go
func TestIntegration_NewFeature(t *testing.T) {
    suite := setupIntegrationTest(t)
    defer teardownIntegrationTest(t, suite)

    t.Run("Success Case", func(t *testing.T) {
        // Test implementation
    })

    t.Run("Error Case", func(t *testing.T) {
        // Test error scenarios
    })
}
```

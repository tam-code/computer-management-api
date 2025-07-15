# Computer Management API

A RESTful API service for managing computers and their assignments to employees, built with Go and PostgreSQL.

## 🚀 Features

- **Computer CRUD Operations**: Create, read, update, and delete computers
- **Employee-Computer Management**: Assign and remove computers from employees
- **Data Validation**: Comprehensive input validation for all endpoints
- **Pagination Support**: Efficient data retrieval with pagination
- **Health Monitoring**: Health check endpoint for service monitoring
- **Notification System**: Integrated notification system for threshold monitoring
- **Security Middleware**: Rate limiting, CORS, and security headers
- **Comprehensive Testing**: Full test suite with integration tests

## 📋 Prerequisites

- **Go 1.23+**: [Download and install Go](https://golang.org/dl/)
- **PostgreSQL 13+**: [Download and install PostgreSQL](https://www.postgresql.org/download/)
- **Docker & Docker Compose** (optional): For containerized deployment

## 🛠️ Installation & Setup

#### Using Docker Compose, start the project containers

```bash
docker-compose up -d
```

## 🧪 Running Tests

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Integration Tests

```bash
# Run integration tests
go test ./internal/integration/... -v

# Run specific integration test
go test ./internal/integration/... -v -run="TestIntegration_ComputerCRUD"
```

### Performance Tests

```bash
# Run performance tests
go test ./internal/integration/... -v -run="TestIntegration_DatabasePerformance"
```

## 📚 API Documentation

### Base URL
```
http://localhost:8089/api/v1
```

### Authentication
Currently, the API doesn't require authentication, but security middleware is applied.

### Endpoints
All endpoints under api/v1

#### Health Check
```http
GET /health
```

#### Computer Management

**Get All Computers**
```http
GET /computers?page=1&limit=10
```

**Get Computer by ID**
```http
GET /computers/{id}
```

**Create Computer**
```http
POST /computers
Content-Type: application/json

{
  "name": "Dell-Laptop-001",
  "computer_name": "Test Name",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "ip_address": "192.168.1.100",
  "employee_abbreviation": "ABC",
  "description": "Dell Latitude 5520"
}
```

**Update Computer**
```http
PUT /computers/{id}
Content-Type: application/json

{
  "name": "Updated-Dell-Laptop-001",
  "computer_name": "Test Name",
  "mac_address": "AA:BB:CC:DD:EE:FF",
  "ip_address": "192.168.1.101",
  "employee_abbreviation": "XYZ",
  "description": "Updated Dell Latitude 5520"
}
```

**Delete Computer**
```http
DELETE /computers/{id}
```

#### Employee-Computer Management

**Get Employee's Computers**
```http
GET /employees/{employee_abbreviation}/computers
```

**Assign Computer to Employee**
```http
PUT /employees/{employee_abbreviation}/computers/{computer_id}
```

**Remove Computer from Employee**
```http
DELETE /employees/{employee_abbreviation}/computers/{computer_id}
```

### Example Usage with cURL

```bash
# Create a computer
curl -X POST http://localhost:8089/api/v1/computers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MacBook-Pro-001",
    "mac_address": "AA:BB:CC:DD:EE:FF",
    "computer_name": "Test Name",
    "ip_address": "192.168.1.100",
    "employee_abbreviation": "JDO",
    "description": "MacBook Pro 16-inch"
  }'

# Get all computers
curl http://localhost:8089/api/v1/computers

# Assign computer to employee
curl -X PUT http://localhost:8089/api/v1/employees/ABC/computers/550e8400-e29b-41d4-a716-446655440000

# Remove computer from employee
curl -X DELETE http://localhost:8089/api/v1/employees/ABC/computers/550e8400-e29b-41d4-a716-446655440000
```

## 🔧 Configuration

The application supports configuration through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database username | `postgres` |
| `DB_PASSWORD` | Database password | `password` |
| `DB_NAME` | Database name | `computer_management` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `PORT` | Server port | `8089` |
| `NOTIFICATION_ENDPOINT` | Notification service URL | (optional) |

## 🏗️ Project Structure

```
computer-management-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   └── database.go          # Database connection
│   ├── handler/
│   │   ├── computer.go          # HTTP handlers
│   │   └── interface.go         # Handler interfaces
│   ├── model/
│   │   └── computer.go          # Data models
│   ├── notification/
│   │   └── client.go            # Notification client
│   ├── repository/
│   │   └── computer.go          # Data access layer
│   ├── router/
│   │   └── router.go            # HTTP routing
│   └── integration/
│       └── *_test.go            # Integration tests
├── docker-compose.yml           # Docker services
├── Dockerfile                   # Container definition
├── go.mod                       # Go modules
├── go.sum                       # Dependency checksums
└── README.md                    # Project documentation
```

## 🚨 Error Handling

The API returns appropriate HTTP status codes:

- `200 OK`: Successful operation
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid input data
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate MAC address)
- `500 Internal Server Error`: Server error

Error responses include descriptive messages:

```json
{
  "error": "computer with this MAC address already exists: AA:BB:CC:DD:EE:FF"
}
```

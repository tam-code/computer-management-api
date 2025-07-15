# Makefile for Computer Management API

.PHONY: help test test-coverage test-verbose test-unit test-integration clean build run dev deps lint fmt vet security docker-build docker-run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Testing
test: ## Run all tests
	go test ./...

test-bench: ## Run benchmarks
	go test -bench=. ./...

test-race: ## Run tests with race detection
	go test -race ./...

# Code Quality
lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	gofumpt -l -w .

vet: ## Run go vet
	go vet ./...

security: ## Run security checks
	gosec ./...

# Cleanup
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache

clean-deps: ## Clean dependency cache
	go clean -modcache

# Integration test setup
test-integration-setup: ## Setup integration test environment
	@echo "Starting test database..."
	docker-compose up -d db notification
	@echo "Waiting for database to be ready..."
	@sleep 5

test-integration-teardown: ## Teardown integration test environment
	@echo "Stopping test services..."
	docker-compose down

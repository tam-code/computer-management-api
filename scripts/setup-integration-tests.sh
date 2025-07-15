#!/bin/bash

# Integration Test Setup Script
# This script sets up the test environment for running integration tests

set -e

echo "🔧 Setting up integration test environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start test database and notification service
echo "🐘 Starting test database..."
docker-compose up -d db notification

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if docker-compose exec -T db pg_isready -U postgres > /dev/null 2>&1; then
        echo "✅ Database is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "   Attempt $attempt/$max_attempts..."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ Database failed to start within expected time"
    exit 1
fi

# Check notification service
echo "📧 Checking notification service..."
max_attempts=15
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -f http://localhost:8081/health > /dev/null 2>&1; then
        echo "✅ Notification service is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "   Attempt $attempt/$max_attempts..."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "⚠️  Notification service may not be ready, but proceeding with tests..."
fi

echo "🚀 Environment setup complete!"
echo ""
echo "You can now run integration tests with:"
echo "  make test-integration"
echo "  make test-integration-api"
echo "  make test-integration-db"
echo ""
echo "To teardown the environment:"
echo "  make test-integration-teardown"

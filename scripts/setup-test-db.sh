#!/bin/bash

# Setup Test Database for Integration Tests
echo "🗄️ Setting up test database for Task 7 integration tests"

# Create test database with Docker
docker run --name postgres-test -e POSTGRES_PASSWORD=testpass -e POSTGRES_DB=newsapp_test -p 5433:5432 -d postgres:13

# Wait for database to be ready
echo "Waiting for database to be ready..."
sleep 10

# Set environment variables for tests
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=postgres
export DB_PASSWORD=testpass
export DB_NAME=newsapp_test
export INTEGRATION_TEST=1

echo "✅ Test database ready. Run integration tests with:"
echo "INTEGRATION_TEST=1 go test ./internal/repositories -v"
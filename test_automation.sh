#!/bin/bash

echo "🚀 Starting QCAT Automation System Test..."

# Check if server is running
echo "📡 Checking if QCAT server is running..."
if curl -s http://localhost:8082/health > /dev/null; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running. Please start the server first:"
    echo "   go run cmd/qcat/main.go"
    exit 1
fi

# Wait a moment for system to initialize
echo "⏳ Waiting for system initialization..."
sleep 3

# Run the test
echo "🧪 Running automation system tests..."
go run test_automation.go

echo "✅ Test completed!"
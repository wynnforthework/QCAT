#!/bin/bash

# Build script for QCAT services
set -e

echo "Building QCAT services..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build main QCAT application
echo "Building main QCAT application..."
go build -o bin/qcat ./cmd/qcat

# Build optimizer service
echo "Building optimizer service..."
go build -o bin/optimizer ./cmd/optimizer

# Build other services (placeholders for now)
echo "Creating placeholder services..."

# Create placeholder ingestor service
cat > bin/ingestor << 'EOF'
#!/bin/bash
echo "Ingestor service starting on port 8082..."
while true; do
    sleep 30
    echo "Ingestor service heartbeat: $(date)"
done
EOF
chmod +x bin/ingestor

# Create placeholder trader service
cat > bin/trader << 'EOF'
#!/bin/bash
echo "Trader service starting on port 8083..."
while true; do
    sleep 30
    echo "Trader service heartbeat: $(date)"
done
EOF
chmod +x bin/trader

echo "All services built successfully!"
echo "Services available in bin/ directory:"
ls -la bin/

echo ""
echo "To run the main application:"
echo "  ./bin/qcat"
echo ""
echo "Services will be managed automatically by the orchestrator."
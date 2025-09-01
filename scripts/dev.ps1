# QCAT Development Helper Script for PowerShell
# Provides convenient shortcuts for common development tasks

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

# Color functions
function Write-Info($message) {
    Write-Host "[INFO] $message" -ForegroundColor Blue
}

function Write-Success($message) {
    Write-Host "[SUCCESS] $message" -ForegroundColor Green
}

function Write-Warning($message) {
    Write-Host "[WARNING] $message" -ForegroundColor Yellow
}

function Write-Error($message) {
    Write-Host "[ERROR] $message" -ForegroundColor Red
}

# Make script executable
try {
    if (Get-Command bash -ErrorAction SilentlyContinue) {
        bash -c "chmod +x scripts/start_local_improved.sh" 2>$null
    }
} catch {
    # Ignore chmod errors on Windows
}

switch ($Command.ToLower()) {
    "dev" {
        Write-Info "Starting development mode..."
        bash scripts/start_local_improved.sh --dev
    }
    
    "dev-api" {
        Write-Info "Starting API service only (dev mode)..."
        bash scripts/start_local_improved.sh --services api --dev
    }
    
    "dev-frontend" {
        Write-Info "Starting frontend service only (dev mode)..."
        bash scripts/start_local_improved.sh --services frontend --dev
    }
    
    "dev-optimizer" {
        Write-Info "Starting optimizer service only (dev mode)..."
        bash scripts/start_local_improved.sh --services optimizer --dev
    }
    
    "prod" {
        Write-Info "Starting production mode..."
        bash scripts/start_local_improved.sh --production
    }
    
    "quick-start" {
        Write-Info "Quick start (skip deps and build)..."
        bash scripts/start_local_improved.sh --skip-deps --skip-build
    }
    
    "debug" {
        Write-Info "Starting with debug output..."
        bash scripts/start_local_improved.sh --debug
    }
    
    "start-local" {
        Write-Info "Starting local development environment..."
        bash scripts/start_local_improved.sh
    }
    
    "build" {
        Write-Info "Building Go applications..."
        go build -o qcat.exe ./cmd/qcat/main.go
        go build -o optimizer.exe ./cmd/optimizer/main.go
        Write-Success "Build complete."
    }
    
    "test" {
        Write-Info "Running tests..."
        go test -v ./internal/...
    }
    
    "clean" {
        Write-Info "Cleaning build artifacts..."
        Remove-Item -Path "qcat.exe", "optimizer.exe" -ErrorAction SilentlyContinue
        Write-Success "Clean complete."
    }
    
    "deps" {
        Write-Info "Installing dependencies..."
        go mod download
        go mod tidy
        if (Test-Path "frontend") {
            Set-Location frontend
            npm install
            Set-Location ..
        }
        Write-Success "Dependencies installed."
    }
    
    "status" {
        Write-Info "Checking service status..."
        $ports = @(8082, 8081, 3001)
        foreach ($port in $ports) {
            $connection = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue
            if ($connection.TcpTestSucceeded) {
                Write-Success "Port $port is active"
            } else {
                Write-Warning "Port $port is not active"
            }
        }
    }
    
    "help" {
        Write-Host @"
QCAT Development Helper v1.0 (PowerShell)

Usage: .\scripts\dev.ps1 [command]

Available commands:

  Development Environment:
    start-local     - Start local development environment (all services)
    dev             - Start development mode (hot reload, debug)
    dev-api         - Start API service only (dev mode)
    dev-frontend    - Start frontend service only (dev mode)
    dev-optimizer   - Start optimizer service only (dev mode)
    prod            - Start production mode
    quick-start     - Quick start (skip deps and build)
    debug           - Start with debug output

  Build & Test:
    build           - Build Go applications
    test            - Run tests
    clean           - Clean build artifacts
    deps            - Install dependencies

  Utilities:
    status          - Check service status
    help            - Show this help

Examples:
    .\scripts\dev.ps1 dev              # Start development mode
    .\scripts\dev.ps1 dev-frontend     # Start only frontend in dev mode
    .\scripts\dev.ps1 quick-start      # Quick start without deps/build
    .\scripts\dev.ps1 status           # Check service status

"@ -ForegroundColor Cyan
    }
    
    default {
        Write-Error "Unknown command: $Command"
        Write-Host ""
        & $MyInvocation.MyCommand.Path help
    }
}

# QCAT - Quantitative Contract Automated Trading System

QCAT is a comprehensive automated trading system for cryptocurrency contracts, featuring advanced quantitative strategies, risk management, and portfolio optimization.

## Features

- Fully automated trading with 10 key automation capabilities
- Advanced risk management and position sizing
- Strategy optimization and backtesting
- Real-time market data processing
- Portfolio management and optimization
- Hot market detection and analysis
- Modern web interface built with React and shadcn/ui

## Tech Stack

- Backend: Go
- Frontend: React + Next.js + shadcn/ui
- Database: PostgreSQL
- Cache: Redis
- CI/CD: GitHub Actions

## Project Structure

```
.
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── config/           # Configuration handling
│   ├── market/           # Market data processing
│   ├── exchange/         # Exchange connectivity
│   ├── strategy/         # Trading strategies
│   ├── risk/             # Risk management
│   ├── portfolio/        # Portfolio management
│   ├── optimizer/        # Strategy optimization
│   ├── backtest/         # Backtesting engine
│   ├── hotlist/          # Hot market analysis
│   └── monitor/          # System monitoring
├── pkg/                   # Public libraries
├── api/                   # API definitions
├── frontend/             # React frontend application
├── configs/              # Configuration files
├── scripts/              # Utility scripts
├── docs/                 # Documentation
└── test/                 # Additional test files
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Node.js 20 or later
- PostgreSQL 15 or later
- Redis 7 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/qcat.git
   cd qcat
   ```

2. Install backend dependencies:
   ```bash
   go mod download
   ```

3. Install frontend dependencies:
   ```bash
   cd frontend
   npm install
   ```

4. Configure the application:
   - Copy `configs/config.yaml.example` to `configs/config.yaml`
   - Update the configuration with your settings

5. Start the development servers:
   ```bash
   # Backend
   go run cmd/qcat/main.go

   # Frontend
   cd frontend
   npm run dev
   ```

## Development

- Follow Go best practices and project layout conventions
- Use conventional commits for version control
- Write tests for new features
- Update documentation as needed

## Testing

```bash
# Run backend tests
go test ./...

# Run frontend tests
cd frontend
npm test
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
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
- RESTful API and WebSocket support

## Tech Stack

- Backend: Go + Gin + WebSocket
- Frontend: React + Next.js + shadcn/ui
- Database: PostgreSQL
- Cache: Redis
- CI/CD: GitHub Actions

## Project Structure

```
.
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── api/              # API server and handlers
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
   git clone <repository-url>
   cd QCAT
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Install Node.js dependencies:
   ```bash
   cd frontend
   npm install
   ```

4. Configure the application:
   ```bash
   cp configs/config.yaml.example configs/config.yaml
   # Edit configs/config.yaml with your settings
   ```

5. Start the backend server:
   ```bash
   go run cmd/qcat/main.go
   ```

6. Start the frontend development server:
   ```bash
   cd frontend
   npm run dev
   ```

## API Documentation

### REST API Endpoints

#### Optimizer
- `POST /api/v1/optimizer/run` - Start optimization task
- `GET /api/v1/optimizer/tasks` - List optimization tasks
- `GET /api/v1/optimizer/tasks/:id` - Get optimization task details
- `GET /api/v1/optimizer/results/:id` - Get optimization results

#### Strategy
- `GET /api/v1/strategy/` - List strategies
- `GET /api/v1/strategy/:id` - Get strategy details
- `POST /api/v1/strategy/` - Create new strategy
- `PUT /api/v1/strategy/:id` - Update strategy
- `DELETE /api/v1/strategy/:id` - Delete strategy
- `POST /api/v1/strategy/:id/promote` - Promote strategy version
- `POST /api/v1/strategy/:id/start` - Start strategy
- `POST /api/v1/strategy/:id/stop` - Stop strategy
- `POST /api/v1/strategy/:id/backtest` - Run backtest

#### Portfolio
- `GET /api/v1/portfolio/overview` - Get portfolio overview
- `GET /api/v1/portfolio/allocations` - Get portfolio allocations
- `POST /api/v1/portfolio/rebalance` - Trigger rebalancing
- `GET /api/v1/portfolio/history` - Get portfolio history

#### Risk
- `GET /api/v1/risk/overview` - Get risk overview
- `GET /api/v1/risk/limits` - Get risk limits
- `POST /api/v1/risk/limits` - Set risk limits
- `GET /api/v1/risk/circuit-breakers` - Get circuit breakers
- `POST /api/v1/risk/circuit-breakers` - Set circuit breakers
- `GET /api/v1/risk/violations` - Get risk violations

#### Hotlist
- `GET /api/v1/hotlist/symbols` - Get hot symbols
- `POST /api/v1/hotlist/approve` - Approve symbol for trading
- `GET /api/v1/hotlist/whitelist` - Get whitelist
- `POST /api/v1/hotlist/whitelist` - Add to whitelist
- `DELETE /api/v1/hotlist/whitelist/:symbol` - Remove from whitelist

#### Metrics
- `GET /api/v1/metrics/strategy/:id` - Get strategy metrics
- `GET /api/v1/metrics/system` - Get system metrics
- `GET /api/v1/metrics/performance` - Get performance metrics

#### Audit
- `GET /api/v1/audit/logs` - Get audit logs
- `GET /api/v1/audit/decisions` - Get decision chains
- `GET /api/v1/audit/performance` - Get performance metrics
- `POST /api/v1/audit/export` - Export audit report

### WebSocket Endpoints

- `ws://localhost:8082/ws/market/:symbol` - Real-time market data
- `ws://localhost:8082/ws/strategy/:id` - Strategy status updates
- `ws://localhost:8082/ws/alerts` - Alert notifications

### Health Check

- `GET /health` - Server health status

## Configuration

The application configuration is stored in `configs/config.yaml`:

```yaml
app:
  name: qcat
  version: 1.0.0
  env: development

server:
  host: localhost
  port: 8082

database:
  driver: postgres
  host: localhost
  port: 5432
  name: qcat
  user: postgres
  password: ""
  sslmode: disable

redis:
  enabled: true
  host: localhost
  port: 6379
  password: ""
  db: 0

exchange:
  binance:
    api_key: ""
    api_secret: ""
    testnet: true
    rate_limit: 1200

risk:
  max_leverage: 10
  max_position_size: 100000
  max_drawdown: 0.1
  circuit_breaker_threshold: 0.05
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/qcat cmd/qcat/main.go
```

### Docker

```bash
docker build -t qcat .
docker run -p 8080:8080 qcat
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions, please open an issue on GitHub or contact the development team.
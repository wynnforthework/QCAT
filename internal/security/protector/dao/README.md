# Fund Protector Data Access Layer (DAO)

This package provides a comprehensive data access layer for the Fund Protector system, implementing the repository pattern with PostgreSQL as the backend database.

## Overview

The DAO layer provides:
- Type-safe database operations
- Transaction management
- Connection pooling
- Comprehensive error handling
- Support for complex data types (JSON, arrays)
- Database migration support

## Architecture

### Core Components

1. **Interfaces** (`interfaces.go`) - Define contracts for all DAO operations
2. **Models** (`models.go`) - Data structures with database mapping
3. **PostgreSQL Implementation** (`postgres.go`) - Main DAO manager implementation
4. **Individual DAOs** - Specific implementations for each entity type
5. **Connection Management** (`connection.go`) - Database connection utilities
6. **Tests** (`dao_test.go`) - Comprehensive unit tests

### Supported Entities

- **HistoricalReturns** - Daily portfolio return data
- **HistoricalEquity** - Portfolio equity value over time
- **RiskSnapshots** - Calculated risk metrics snapshots
- **TransferRecords** - Fund transfer operations
- **EmergencyEvents** - Emergency situations and responses
- **PositionSnapshots** - Trading position snapshots
- **FundStatusSnapshots** - Overall fund health snapshots
- **CircuitBreakerEvents** - Circuit breaker activations
- **ProtectionMetrics** - System performance metrics

## Usage

### Basic Setup

```go
import "qcat/internal/security/protector/dao"

// Configure database connection
config := &dao.DatabaseConfig{
    Host:            "localhost",
    Port:            5432,
    User:            "qcat_user",
    Password:        "password",
    DBName:          "qcat",
    SSLMode:         "require",
    MaxOpen:         25,
    MaxIdle:         5,
    ConnMaxLifetime: 1 * time.Hour,
    ConnMaxIdleTime: 15 * time.Minute,
}

// Create DAO manager
manager, err := dao.NewDAOManagerFromConfig(config)
if err != nil {
    log.Fatal("Failed to create DAO manager:", err)
}
defer manager.Close()
```

### Working with Historical Returns

```go
ctx := context.Background()

// Insert a new return record
returnRecord := &dao.HistoricalReturn{
    Date:           time.Now().Truncate(24 * time.Hour),
    ReturnValue:    0.025, // 2.5% return
    PortfolioValue: &portfolioValue,
    Volatility:     &volatility,
}

err := manager.HistoricalReturns().Insert(ctx, returnRecord)
if err != nil {
    log.Printf("Failed to insert return: %v", err)
}

// Get last 30 days of returns
returns, err := manager.HistoricalReturns().GetLastNDays(ctx, 30)
if err != nil {
    log.Printf("Failed to get returns: %v", err)
}

// Get latest return
latest, err := manager.HistoricalReturns().GetLatest(ctx)
if err != nil {
    log.Printf("Failed to get latest return: %v", err)
}
```

### Working with Transfer Records

```go
// Create a transfer record
transfer := &dao.TransferRecord{
    ID:                    "TRF_" + time.Now().Format("20060102150405"),
    Type:                  "PROFIT_TRANSFER",
    Amount:                1000.0,
    Currency:              "USDT",
    FromAddress:           "trading_account",
    ToAddress:             "cold_wallet_address",
    Status:                "PENDING",
    TriggerReason:         stringPtr("profit_threshold_reached"),
    Priority:              1,
    RequiredConfirmations: 6,
    Metadata: dao.JSONMap{
        "auto_transfer": true,
        "profit_ratio":  0.15,
    },
}

err := manager.TransferRecords().Insert(ctx, transfer)
if err != nil {
    log.Printf("Failed to insert transfer: %v", err)
}

// Update transfer status
err = manager.TransferRecords().UpdateStatus(ctx, transfer.ID, "COMPLETED")
if err != nil {
    log.Printf("Failed to update transfer status: %v", err)
}

// Get pending transfers
pending, err := manager.TransferRecords().GetByStatus(ctx, "PENDING", 100)
if err != nil {
    log.Printf("Failed to get pending transfers: %v", err)
}
```

### Transaction Management

```go
// Start a transaction
tx, err := manager.BeginTx(ctx)
if err != nil {
    log.Printf("Failed to begin transaction: %v", err)
    return
}

// Perform multiple operations
err = tx.HistoricalReturns().Insert(ctx, returnRecord)
if err != nil {
    tx.Rollback()
    return
}

err = tx.RiskSnapshots().Insert(ctx, riskSnapshot)
if err != nil {
    tx.Rollback()
    return
}

// Commit transaction
err = tx.Commit()
if err != nil {
    log.Printf("Failed to commit transaction: %v", err)
}
```

### Working with Risk Snapshots

```go
// Create a risk snapshot
snapshot := &dao.RiskSnapshot{
    Timestamp:         time.Now(),
    RiskLevel:         "MEDIUM",
    RiskScore:         0.35,
    VaR95:             -0.05,
    ExpectedShortfall: -0.08,
    MaxDrawdown:       0.12,
    VolatilityIndex:   0.25,
    Leverage:          2.5,
    Concentration:     0.4,
    SharpeRatio:       &sharpeRatio,
}

err := manager.RiskSnapshots().Insert(ctx, snapshot)
if err != nil {
    log.Printf("Failed to insert risk snapshot: %v", err)
}

// Get high-risk snapshots
highRisk, err := manager.RiskSnapshots().GetByRiskLevel(ctx, "HIGH", 50)
if err != nil {
    log.Printf("Failed to get high-risk snapshots: %v", err)
}
```

### Working with Emergency Events

```go
// Create an emergency event
event := &dao.EmergencyEvent{
    ID:          "EMG_" + time.Now().Format("20060102150405"),
    Type:        "DAILY_LOSS_EXCEEDED",
    Severity:    "HIGH",
    Description: "Daily loss limit exceeded",
    TriggerData: dao.JSONMap{
        "daily_loss_ratio": 0.06,
        "max_daily_loss":   0.05,
        "actual_loss":      -5000.0,
    },
    Status:            "ACTIVE",
    NotificationsSent: 0,
}

err := manager.EmergencyEvents().Insert(ctx, event)
if err != nil {
    log.Printf("Failed to insert emergency event: %v", err)
}

// Get active emergencies
active, err := manager.EmergencyEvents().GetActive(ctx)
if err != nil {
    log.Printf("Failed to get active emergencies: %v", err)
}

// Resolve emergency
err = manager.EmergencyEvents().UpdateStatus(ctx, event.ID, "RESOLVED")
if err != nil {
    log.Printf("Failed to resolve emergency: %v", err)
}
```

## Database Migration

Run the migration SQL file to set up the database schema:

```bash
psql -U qcat_user -d qcat -f migrations/001_create_fund_protector_tables.sql
```

Or use the migration function:

```go
migrationSQL, err := os.ReadFile("migrations/001_create_fund_protector_tables.sql")
if err != nil {
    log.Fatal("Failed to read migration file:", err)
}

err = dao.MigrateDatabase(db, string(migrationSQL))
if err != nil {
    log.Fatal("Failed to run migration:", err)
}
```

## Testing

Run the tests with a test database:

```bash
# Set up test database environment variables
export TEST_DB_HOST=localhost
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=password
export TEST_DB_NAME=qcat_test

# Run tests
go test ./internal/security/protector/dao/...
```

## Custom Data Types

### JSONMap

For storing JSON data in PostgreSQL JSONB columns:

```go
metadata := dao.JSONMap{
    "key1": "value1",
    "key2": 123,
    "nested": map[string]interface{}{
        "subkey": "subvalue",
    },
}
```

### StringList

For storing string arrays in PostgreSQL:

```go
actions := dao.StringList{"action1", "action2", "action3"}
```

## Error Handling

All DAO methods return detailed errors that can be checked:

```go
transfer, err := manager.TransferRecords().GetByID(ctx, "nonexistent")
if err != nil {
    log.Printf("Error: %v", err)
}
if transfer == nil {
    log.Println("Transfer not found")
}
```

## Performance Considerations

1. **Connection Pooling**: Configured automatically with sensible defaults
2. **Indexes**: Created on frequently queried columns
3. **Batch Operations**: Use transactions for multiple related operations
4. **Data Cleanup**: Use the cleanup functions to manage data retention
5. **Query Optimization**: Indexes are created for time-series and status queries

## Security

1. **SQL Injection Protection**: All queries use parameterized statements
2. **Connection Security**: Supports SSL/TLS connections
3. **Data Encryption**: Sensitive data can be encrypted at the application level
4. **Access Control**: Database-level permissions should be configured appropriately

## Monitoring

The DAO layer provides built-in monitoring capabilities:

- Connection pool statistics
- Query performance metrics
- Error tracking
- Data quality validation

Use the protection metrics DAO to track system performance and health.
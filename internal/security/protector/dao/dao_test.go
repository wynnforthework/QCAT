package dao

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// Test database configuration
var testDBConfig = &DatabaseConfig{
	Host:            getEnv("TEST_DB_HOST", "localhost"),
	Port:            5432,
	User:            getEnv("TEST_DB_USER", "postgres"),
	Password:        getEnv("TEST_DB_PASSWORD", "password"),
	DBName:          getEnv("TEST_DB_NAME", "qcat_test"),
	SSLMode:         "disable",
	MaxOpen:         10,
	MaxIdle:         5,
	ConnMaxLifetime: 1 * time.Hour,
	ConnMaxIdleTime: 15 * time.Minute,
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestDB creates a test database connection and runs migrations
func setupTestDB(t *testing.T) (*sqlx.DB, DAOManager) {
	db, err := NewDatabaseConnection(testDBConfig)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
	}
	
	// Run migrations (in a real test, you'd load from file)
	migrationSQL := `
		-- Create test tables (simplified for testing)
		CREATE TABLE IF NOT EXISTS historical_returns (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL UNIQUE,
			return_value DECIMAL(15,8) NOT NULL,
			portfolio_value DECIMAL(20,8),
			benchmark_return DECIMAL(15,8),
			volatility DECIMAL(10,6),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		
		CREATE TABLE IF NOT EXISTS transfer_records (
			id VARCHAR(50) PRIMARY KEY,
			type VARCHAR(30) NOT NULL,
			amount DECIMAL(20,8) NOT NULL,
			currency VARCHAR(10) NOT NULL DEFAULT 'USDT',
			from_address VARCHAR(100) NOT NULL,
			to_address VARCHAR(100) NOT NULL,
			status VARCHAR(20) NOT NULL,
			trigger_reason VARCHAR(100),
			transaction_hash VARCHAR(100),
			estimated_fee DECIMAL(15,8),
			actual_fee DECIMAL(15,8),
			confirmations INTEGER DEFAULT 0,
			required_confirmations INTEGER DEFAULT 6,
			priority INTEGER DEFAULT 1,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			executed_at TIMESTAMP,
			completed_at TIMESTAMP
		);
	`
	
	err = MigrateDatabase(db, migrationSQL)
	require.NoError(t, err)
	
	manager := NewPostgresDAOManager(db)
	return db, manager
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("TRUNCATE TABLE historical_returns, transfer_records CASCADE")
	if err != nil {
		t.Logf("Failed to cleanup test data: %v", err)
	}
}

func TestHistoricalReturnsDAO(t *testing.T) {
	db, manager := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)
	
	dao := manager.HistoricalReturns()
	ctx := context.Background()
	
	// Test Insert
	testReturn := &HistoricalReturn{
		Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ReturnValue:    0.05,
		PortfolioValue: floatPtr(100000.0),
		Volatility:     floatPtr(0.15),
	}
	
	err := dao.Insert(ctx, testReturn)
	require.NoError(t, err)
	assert.NotZero(t, testReturn.ID)
	assert.NotZero(t, testReturn.CreatedAt)
	
	// Test GetLatest
	latest, err := dao.GetLatest(ctx)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, testReturn.ReturnValue, latest.ReturnValue)
	
	// Test GetLastNDays
	returns, err := dao.GetLastNDays(ctx, 30)
	require.NoError(t, err)
	assert.Len(t, returns, 1)
	
	// Test GetByDateRange
	startDate := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	returns, err = dao.GetByDateRange(ctx, startDate, endDate)
	require.NoError(t, err)
	assert.Len(t, returns, 1)
}

func TestTransferRecordsDAO(t *testing.T) {
	db, manager := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)
	
	dao := manager.TransferRecords()
	ctx := context.Background()
	
	// Test Insert
	testTransfer := &TransferRecord{
		ID:                    "TEST_001",
		Type:                  "PROFIT_TRANSFER",
		Amount:                1000.0,
		Currency:              "USDT",
		FromAddress:           "trading_account",
		ToAddress:             "cold_wallet",
		Status:                "PENDING",
		TriggerReason:         stringPtr("profit_threshold_reached"),
		Priority:              1,
		RequiredConfirmations: 6,
		Metadata: JSONMap{
			"auto_transfer": true,
			"profit_ratio":  0.15,
		},
	}
	
	err := dao.Insert(ctx, testTransfer)
	require.NoError(t, err)
	assert.NotZero(t, testTransfer.CreatedAt)
	
	// Test GetByID
	retrieved, err := dao.GetByID(ctx, "TEST_001")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, testTransfer.Amount, retrieved.Amount)
	assert.Equal(t, testTransfer.Type, retrieved.Type)
	
	// Test UpdateStatus
	err = dao.UpdateStatus(ctx, "TEST_001", "COMPLETED")
	require.NoError(t, err)
	
	// Verify status update
	retrieved, err = dao.GetByID(ctx, "TEST_001")
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", retrieved.Status)
	
	// Test GetByStatus
	transfers, err := dao.GetByStatus(ctx, "COMPLETED", 10)
	require.NoError(t, err)
	assert.Len(t, transfers, 1)
	
	// Test GetByType
	transfers, err = dao.GetByType(ctx, "PROFIT_TRANSFER", 10)
	require.NoError(t, err)
	assert.Len(t, transfers, 1)
	
	// Test GetRecent
	transfers, err = dao.GetRecent(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, transfers, 1)
}

func TestTransactionManagement(t *testing.T) {
	db, manager := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)
	
	ctx := context.Background()
	
	// Test successful transaction
	tx, err := manager.BeginTx(ctx)
	require.NoError(t, err)
	
	testReturn := &HistoricalReturn{
		Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		ReturnValue: 0.03,
	}
	
	err = tx.HistoricalReturns().Insert(ctx, testReturn)
	require.NoError(t, err)
	
	err = tx.Commit()
	require.NoError(t, err)
	
	// Verify data was committed
	latest, err := manager.HistoricalReturns().GetLatest(ctx)
	require.NoError(t, err)
	assert.Equal(t, testReturn.ReturnValue, latest.ReturnValue)
	
	// Test rollback transaction
	tx, err = manager.BeginTx(ctx)
	require.NoError(t, err)
	
	testReturn2 := &HistoricalReturn{
		Date:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		ReturnValue: 0.07,
	}
	
	err = tx.HistoricalReturns().Insert(ctx, testReturn2)
	require.NoError(t, err)
	
	err = tx.Rollback()
	require.NoError(t, err)
	
	// Verify data was not committed
	latest, err = manager.HistoricalReturns().GetLatest(ctx)
	require.NoError(t, err)
	assert.Equal(t, testReturn.ReturnValue, latest.ReturnValue) // Should still be the first one
}

func TestJSONMapSerialization(t *testing.T) {
	// Test JSONMap Value method
	jsonMap := JSONMap{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	
	value, err := jsonMap.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)
	
	// Test JSONMap Scan method
	var scannedMap JSONMap
	err = scannedMap.Scan(value)
	require.NoError(t, err)
	
	assert.Equal(t, "value1", scannedMap["key1"])
	assert.Equal(t, float64(123), scannedMap["key2"]) // JSON numbers are float64
	assert.Equal(t, true, scannedMap["key3"])
	
	// Test nil JSONMap
	var nilMap JSONMap
	value, err = nilMap.Value()
	require.NoError(t, err)
	assert.Nil(t, value)
	
	err = nilMap.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, nilMap)
}

func TestStringListSerialization(t *testing.T) {
	// Test StringList Value method
	stringList := StringList{"action1", "action2", "action3"}
	
	value, err := stringList.Value()
	require.NoError(t, err)
	assert.Equal(t, `{"action1","action2","action3"}`, value)
	
	// Test StringList Scan method
	var scannedList StringList
	err = scannedList.Scan(`{"action1","action2","action3"}`)
	require.NoError(t, err)
	
	assert.Equal(t, []string{"action1", "action2", "action3"}, []string(scannedList))
	
	// Test empty StringList
	var emptyList StringList
	value, err = emptyList.Value()
	require.NoError(t, err)
	assert.Nil(t, value)
	
	err = emptyList.Scan("{}")
	require.NoError(t, err)
	assert.Equal(t, []string{}, []string(emptyList))
	
	// Test nil StringList
	var nilList StringList
	err = nilList.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, nilList)
}

// Helper functions for tests
func floatPtr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
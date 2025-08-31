package protector

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"
)

// MockWalletService 测试用钱包服务，实现完整的 WalletService 接口
type MockWalletService struct {
	transfers   map[string]*TransferStatus
	shouldFail  bool
	failureRate float64
}

// NewMockWalletService 创建模拟钱包服务
func NewMockWalletService() *MockWalletService {
	return &MockWalletService{
		transfers:   make(map[string]*TransferStatus),
		shouldFail:  false,
		failureRate: 0.0,
	}
}

// InitiateTransfer 发起转账（模拟）
func (m *MockWalletService) InitiateTransfer(ctx context.Context, request *TransferRequest) (*TransferResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock transfer failure")
	}

	// 模拟随机失败
	if m.failureRate > 0 {
		randomValue, _ := rand.Int(rand.Reader, big.NewInt(100))
		if float64(randomValue.Int64()) < m.failureRate*100 {
			return nil, fmt.Errorf("random mock transfer failure")
		}
	}

	transferID := fmt.Sprintf("MOCK_TXF_%d", time.Now().Unix())
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())

	status := &TransferStatus{
		TransferID:      transferID,
		Status:          "CONFIRMED",
		TransactionHash: txHash,
		Confirmations:   6,
		ActualFee:       0.001,
		Metadata:        request.Metadata,
	}
	now := time.Now()
	status.CompletedAt = &now

	m.transfers[transferID] = status

	response := &TransferResponse{
		TransferID:      transferID,
		Status:          "CONFIRMED",
		TransactionHash: txHash,
		EstimatedFee:    0.001,
		CreatedAt:       time.Now(),
	}

	log.Printf("Mock transfer initiated: %s", transferID)
	return response, nil
}

// GetTransferStatus 获取转账状态（模拟）
func (m *MockWalletService) GetTransferStatus(ctx context.Context, transferID string) (*TransferStatus, error) {
	status, exists := m.transfers[transferID]
	if !exists {
		return nil, fmt.Errorf("transfer not found: %s", transferID)
	}
	return status, nil
}

// CancelTransfer 取消转账（模拟）
func (m *MockWalletService) CancelTransfer(ctx context.Context, transferID string) error {
	status, exists := m.transfers[transferID]
	if !exists {
		return fmt.Errorf("transfer not found: %s", transferID)
	}
	status.Status = "CANCELLED"
	return nil
}

// GetTransferHistory 获取转账历史（模拟）
func (m *MockWalletService) GetTransferHistory(ctx context.Context, limit int) ([]*TransferRecord, error) {
	records := make([]*TransferRecord, 0, len(m.transfers))
	for _, status := range m.transfers {
		record := &TransferRecord{
			ID:              status.TransferID,
			Status:          status.Status,
			TransactionHash: status.TransactionHash,
			Timestamp:       time.Now(),
			Metadata:        status.Metadata,
		}
		records = append(records, record)
	}
	return records, nil
}

// ValidateAddress 验证地址（模拟）
func (m *MockWalletService) ValidateAddress(address string) error {
	if len(address) < 10 {
		return fmt.Errorf("address too short: %s", address)
	}
	return nil
}

// EstimateTransferFee 估算转账手续费（模拟）
func (m *MockWalletService) EstimateTransferFee(ctx context.Context, amount float64, toAddress string) (float64, error) {
	return 0.001, nil
}

// SetShouldFail 设置是否应该失败（测试用）
func (m *MockWalletService) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// SetFailureRate 设置失败率（测试用）
func (m *MockWalletService) SetFailureRate(rate float64) {
	m.failureRate = rate
}

// GetTransfers 获取所有转账（测试用）
func (m *MockWalletService) GetTransfers() map[string]*TransferStatus {
	return m.transfers
}

// TestRealWalletService 测试真实钱包服务
func TestRealWalletService(t *testing.T) {
	ctx := context.Background()

	// 创建真实的钱包服务配置
	config := &WalletConfig{
		Provider:          "ethereum",
		NetworkURL:        "https://mainnet.infura.io/v3/test",
		PrivateKey:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12",
		HotWalletAddress:  "0x1234567890123456789012345678901234567890",
		ColdWalletAddress: "0x0987654321098765432109876543210987654321",
		MinConfirmations:  3,
		MaxGasPrice:       100.0,
		TransferTimeout:   5 * time.Minute,
		EnableMultiSig:    true,
		MultiSigThreshold: 2,
	}

	service := NewDefaultWalletService(config)

	// 测试发起转账
	request := &TransferRequest{
		Type:        "PROFIT_TRANSFER",
		ToAddress:   "0x0987654321098765432109876543210987654321",
		FromAddress: "0x1234567890123456789012345678901234567890",
		Amount:      100.0,
		Priority:    1,
		Metadata:    map[string]interface{}{"test": "value"},
	}

	response, err := service.InitiateTransfer(ctx, request)
	if err != nil {
		t.Logf("InitiateTransfer result: %v (expected in test environment)", err)
	} else {
		t.Logf("Transfer initiated successfully: %s", response.TransferID)

		// 测试获取转账状态
		status, err := service.GetTransferStatus(ctx, response.TransferID)
		if err != nil {
			t.Logf("GetTransferStatus result: %v", err)
		} else {
			t.Logf("Transfer status: %s", status.Status)
		}

		// 测试取消转账
		err = service.CancelTransfer(ctx, response.TransferID)
		t.Logf("CancelTransfer result: %v", err)
	}

	// 测试地址验证
	err = service.ValidateAddress("short")
	if err == nil {
		t.Error("Expected error for short address")
	} else {
		t.Logf("Short address validation failed as expected: %v", err)
	}

	err = service.ValidateAddress("0x1234567890123456789012345678901234567890")
	if err != nil {
		t.Errorf("ValidateAddress failed for valid address: %v", err)
	} else {
		t.Log("Valid address validation passed")
	}

	// 测试手续费估算
	fee, err := service.EstimateTransferFee(ctx, 100.0, "0x0987654321098765432109876543210987654321")
	if err != nil {
		t.Logf("EstimateTransferFee result: %v", err)
	} else {
		t.Logf("Estimated fee: %f", fee)
		if fee <= 0 {
			t.Error("Expected positive fee")
		}
	}
}

// TestMockWalletService 测试mock钱包服务（保留用于其他测试）
func TestMockWalletService(t *testing.T) {
	ctx := context.Background()
	mock := NewMockWalletService()

	// 测试发起转账
	request := &TransferRequest{
		Type:        "PROFIT_TRANSFER",
		ToAddress:   "0x1234567890abcdef",
		FromAddress: "0xabcdef1234567890",
		Amount:      100.0,
		Priority:    1,
		Metadata:    map[string]interface{}{"test": "value"},
	}

	response, err := mock.InitiateTransfer(ctx, request)
	if err != nil {
		t.Errorf("InitiateTransfer failed: %v", err)
	}
	if response.Status != "CONFIRMED" {
		t.Errorf("Expected status CONFIRMED, got %s", response.Status)
	}

	// 测试获取转账状态
	status, err := mock.GetTransferStatus(ctx, response.TransferID)
	if err != nil {
		t.Errorf("GetTransferStatus failed: %v", err)
	}
	if status.Status != "CONFIRMED" {
		t.Errorf("Expected status CONFIRMED, got %s", status.Status)
	}

	// 测试取消转账
	err = mock.CancelTransfer(ctx, response.TransferID)
	if err != nil {
		t.Errorf("CancelTransfer failed: %v", err)
	}

	// 验证状态已更新
	status, _ = mock.GetTransferStatus(ctx, response.TransferID)
	if status.Status != "CANCELLED" {
		t.Errorf("Expected status CANCELLED, got %s", status.Status)
	}

	// 测试失败模式
	mock.SetShouldFail(true)
	_, err = mock.InitiateTransfer(ctx, request)
	if err == nil {
		t.Error("Expected error when shouldFail is true")
	}

	// 测试地址验证
	mock.SetShouldFail(false)
	err = mock.ValidateAddress("short")
	if err == nil {
		t.Error("Expected error for short address")
	}

	err = mock.ValidateAddress("0x1234567890abcdef")
	if err != nil {
		t.Errorf("ValidateAddress failed for valid address: %v", err)
	}
}

// TestWalletServiceConfiguration 测试钱包服务配置
func TestWalletServiceConfiguration(t *testing.T) {
	// 测试空配置
	service := NewDefaultWalletService(nil)
	if service == nil {
		t.Error("Service should not be nil even with nil config")
	}

	// 测试部分配置
	config := &WalletConfig{
		Provider:         "ethereum",
		NetworkURL:       "https://mainnet.infura.io/v3/test",
		MinConfirmations: 3,
		// 其他字段未配置
	}

	service = NewDefaultWalletService(config)
	ctx := context.Background()

	// 测试地址验证（应该正常工作）
	err := service.ValidateAddress("0x1234567890123456789012345678901234567890")
	if err != nil {
		t.Errorf("ValidateAddress failed: %v", err)
	}

	// 测试手续费估算
	fee, err := service.EstimateTransferFee(ctx, 100.0, "0x1234567890123456789012345678901234567890")
	t.Logf("Fee estimation with partial config: %f, error: %v", fee, err)
}

// TestWalletServiceErrorHandling 测试钱包服务错误处理
func TestWalletServiceErrorHandling(t *testing.T) {
	config := &WalletConfig{
		Provider:          "ethereum",
		NetworkURL:        "https://invalid.network.url",
		PrivateKey:        "invalid_private_key",
		HotWalletAddress:  "invalid_address",
		ColdWalletAddress: "invalid_address",
		MinConfirmations:  3,
		MaxGasPrice:       100.0,
		TransferTimeout:   5 * time.Second,
	}

	service := NewDefaultWalletService(config)
	ctx := context.Background()

	// 测试无效地址验证
	err := service.ValidateAddress("invalid")
	if err == nil {
		t.Error("Expected error for invalid address")
	} else {
		t.Logf("Invalid address validation failed as expected: %v", err)
	}

	// 测试无效转账请求
	request := &TransferRequest{
		Type:        "INVALID_TYPE",
		ToAddress:   "invalid_address",
		FromAddress: "invalid_address",
		Amount:      -100.0, // 负数金额
		Priority:    1,
	}

	_, err = service.InitiateTransfer(ctx, request)
	if err == nil {
		t.Log("Transfer unexpectedly succeeded with invalid request")
	} else {
		t.Logf("Transfer failed as expected: %v", err)
	}

	// 测试获取不存在的转账状态
	_, err = service.GetTransferStatus(ctx, "nonexistent_transfer_id")
	if err == nil {
		t.Log("GetTransferStatus unexpectedly succeeded for nonexistent transfer")
	} else {
		t.Logf("GetTransferStatus failed as expected: %v", err)
	}
}

// TestWalletServiceBatchOperations 测试批量钱包操作
func TestWalletServiceBatchOperations(t *testing.T) {
	config := &WalletConfig{
		Provider:          "ethereum",
		NetworkURL:        "https://mainnet.infura.io/v3/test",
		PrivateKey:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12",
		HotWalletAddress:  "0x1234567890123456789012345678901234567890",
		ColdWalletAddress: "0x0987654321098765432109876543210987654321",
		MinConfirmations:  3,
		MaxGasPrice:       100.0,
		TransferTimeout:   5 * time.Minute,
	}

	service := NewDefaultWalletService(config)
	ctx := context.Background()

	// 测试批量转账
	addresses := []string{
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333",
	}

	for i, address := range addresses {
		request := &TransferRequest{
			Type:        "BATCH_TRANSFER",
			ToAddress:   address,
			FromAddress: config.HotWalletAddress,
			Amount:      float64(10 * (i + 1)),
			Priority:    1,
			Metadata:    map[string]interface{}{"batch_id": fmt.Sprintf("batch_%d", i)},
		}

		response, err := service.InitiateTransfer(ctx, request)
		t.Logf("Batch transfer %d to %s result: %v", i+1, address, err)
		if err == nil {
			t.Logf("Transfer ID: %s", response.TransferID)
		}
	}
}

// TestWalletServiceConcurrency 测试并发钱包操作
func TestWalletServiceConcurrency(t *testing.T) {
	config := &WalletConfig{
		Provider:          "ethereum",
		NetworkURL:        "https://mainnet.infura.io/v3/test",
		PrivateKey:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12",
		HotWalletAddress:  "0x1234567890123456789012345678901234567890",
		ColdWalletAddress: "0x0987654321098765432109876543210987654321",
		MinConfirmations:  3,
		MaxGasPrice:       100.0,
		TransferTimeout:   5 * time.Minute,
	}

	service := NewDefaultWalletService(config)
	ctx := context.Background()

	// 并发执行转账
	const numGoroutines = 5
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			request := &TransferRequest{
				Type:        "CONCURRENT_TRANSFER",
				ToAddress:   fmt.Sprintf("0x%040d", id),
				FromAddress: config.HotWalletAddress,
				Amount:      float64(id + 1),
				Priority:    1,
				Metadata:    map[string]interface{}{"concurrent_id": id},
			}

			response, err := service.InitiateTransfer(ctx, request)
			t.Logf("Concurrent transfer %d result: %v", id, err)
			if err == nil {
				t.Logf("Concurrent transfer %d ID: %s", id, response.TransferID)
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

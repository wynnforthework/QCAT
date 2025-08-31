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

// MockWalletService 模拟钱包服务（用于测试）
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

// TestMockWalletService 测试mock钱包服务
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

package orderbook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/cache"
)

// 新增：数据库接口定义，避免循环导入
type DatabaseInterface interface {
	GetOrderBookHistory(ctx context.Context, symbol string, start, end time.Time) ([]interface{}, error)
	SaveOrderBookSnapshot(ctx context.Context, symbol string, book *OrderBook) error
	GetLatestOrderBookSnapshot(ctx context.Context, symbol string) (interface{}, error)
}

// Manager manages multiple order books
type Manager struct {
	books     map[string]*OrderBook
	cache     cache.Cacher
	snapshots map[string]time.Time
	mu        sync.RWMutex

	// 新增：数据库连接（可选）
	db DatabaseInterface // 新增：数据库接口，用于查询历史数据
}

// NewManager creates a new order book manager
func NewManager(cache cache.Cacher) *Manager {
	return &Manager{
		books:     make(map[string]*OrderBook),
		cache:     cache,
		snapshots: make(map[string]time.Time),
	}
}

// 新增：SetDatabase sets the database connection for historical data queries
func (m *Manager) SetDatabase(db DatabaseInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.db = db
}

// GetOrderBook returns an order book for a symbol
func (m *Manager) GetOrderBook(symbol string) *OrderBook {
	m.mu.Lock()
	defer m.mu.Unlock()

	book, exists := m.books[symbol]
	if !exists {
		book = NewOrderBook(symbol)
		m.books[symbol] = book
	}
	return book
}

// UpdateOrderBook updates an order book with new data
func (m *Manager) UpdateOrderBook(symbol string, bids, asks []Level, timestamp time.Time) error {
	book := m.GetOrderBook(symbol)
	book.Update(bids, asks, timestamp)

	// Cache the snapshot
	if err := m.cacheSnapshot(symbol); err != nil {
		return fmt.Errorf("failed to cache snapshot: %w", err)
	}

	return nil
}

// cacheSnapshot caches the current state of an order book
func (m *Manager) cacheSnapshot(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	book, exists := m.books[symbol]
	if !exists {
		return fmt.Errorf("order book not found: %s", symbol)
	}

	// Only cache if enough time has passed since last snapshot
	lastSnapshot, exists := m.snapshots[symbol]
	if exists && time.Since(lastSnapshot) < time.Second {
		return nil
	}

	snapshot := book.GetSnapshot(20) // Cache top 20 levels
	if err := m.cache.SetOrderBook(context.Background(), symbol, snapshot, 5*time.Second); err != nil {
		return fmt.Errorf("failed to cache order book: %w", err)
	}

	m.snapshots[symbol] = time.Now()
	return nil
}

// GetMidPrice returns the mid price for a symbol
func (m *Manager) GetMidPrice(symbol string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}
	return book.GetMidPrice()
}

// GetSpread returns the bid-ask spread for a symbol
func (m *Manager) GetSpread(symbol string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}
	return book.GetSpread()
}

// GetVWAP returns the VWAP for a given quantity
func (m *Manager) GetVWAP(symbol string, quantity float64, side string) (float64, bool) {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0, false
	}

	if side == "buy" {
		return book.Asks.GetVWAP(quantity)
	}
	return book.Bids.GetVWAP(quantity)
}

// GetDepth returns the total depth up to a price
func (m *Manager) GetDepth(symbol string, price float64, side string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}

	if side == "buy" {
		return book.Asks.GetDepth(price)
	}
	return book.Bids.GetDepth(price)
}

// GetSnapshot returns a snapshot of an order book
func (m *Manager) GetSnapshot(symbol string, depth int) map[string]interface{} {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return nil
	}
	return book.GetSnapshot(depth)
}

// GetHistory returns historical order book snapshots for a symbol within a time range
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*OrderBook, error) {
	// 新增：实现历史订单簿数据查询
	var historicalBooks []*OrderBook

	// 首先尝试从缓存获取历史数据
	cacheBooks, err := m.getHistoryFromCache(ctx, symbol, start, end)
	if err != nil {
		// 如果缓存失败，记录错误但继续尝试其他方法
		fmt.Printf("Failed to get history from cache: %v\n", err)
	}

	if len(cacheBooks) > 0 {
		historicalBooks = append(historicalBooks, cacheBooks...)
	}

	// 如果缓存中没有足够的数据，尝试从数据库获取
	if len(historicalBooks) == 0 && m.db != nil {
		dbBooks, err := m.getHistoryFromDatabase(ctx, symbol, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to get history from database: %w", err)
		}
		historicalBooks = append(historicalBooks, dbBooks...)
	}

	// 如果仍然没有数据，返回空切片
	if len(historicalBooks) == 0 {
		return []*OrderBook{}, nil
	}

	// 新增：按时间戳排序
	m.sortOrderBooksByTimestamp(historicalBooks)

	return historicalBooks, nil
}

// 新增：从缓存获取历史数据
func (m *Manager) getHistoryFromCache(ctx context.Context, symbol string, start, end time.Time) ([]*OrderBook, error) {
	var books []*OrderBook

	// 尝试获取缓存中的历史快照
	// 注意：这里假设缓存中有按时间戳存储的快照
	// 实际实现可能需要根据具体的缓存策略调整

	// 尝试获取最近的几个快照
	for i := 0; i < 10; i++ { // 限制查询数量
		cacheKey := fmt.Sprintf("orderbook:%s:%d", symbol, time.Now().Add(-time.Duration(i)*time.Minute).Unix())

		var snapshot map[string]interface{}
		err := m.cache.Get(ctx, cacheKey, &snapshot)
		if err != nil {
			// 如果获取失败，继续尝试下一个
			continue
		}

		// 解析快照数据
		book, err := m.parseSnapshot(symbol, snapshot)
		if err != nil {
			continue
		}

		// 检查时间范围
		if book.Timestamp.After(start) && book.Timestamp.Before(end) {
			books = append(books, book)
		}
	}

	return books, nil
}

// 新增：从数据库获取历史数据
func (m *Manager) getHistoryFromDatabase(ctx context.Context, symbol string, start, end time.Time) ([]*OrderBook, error) {
	// 新增：实现完整的数据库查询逻辑
	if m.db == nil {
		return []*OrderBook{}, fmt.Errorf("database connection not available")
	}

	// 查询数据库中的订单簿快照
	snapshots, err := m.db.GetOrderBookHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query order book history from database: %w", err)
	}

	// 转换为OrderBook对象
	var books []*OrderBook
	for _, snapshot := range snapshots {
		// 新增：类型断言和转换逻辑
		if snapshotMap, ok := snapshot.(map[string]interface{}); ok {
			book := NewOrderBook(symbol)

			// 解析时间戳
			if timestampStr, ok := snapshotMap["timestamp"].(string); ok {
				if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
					book.Timestamp = timestamp
				}
			}

			// 解析买单数据
			if bidsData, ok := snapshotMap["bids"].([][]float64); ok {
				for _, bidData := range bidsData {
					if len(bidData) >= 2 {
						book.Bids.Update(bidData[0], bidData[1])
					}
				}
			}

			// 解析卖单数据
			if asksData, ok := snapshotMap["asks"].([][]float64); ok {
				for _, askData := range asksData {
					if len(askData) >= 2 {
						book.Asks.Update(askData[0], askData[1])
					}
				}
			}

			books = append(books, book)
		}
	}

	return books, nil
}

// 新增：保存订单簿快照到数据库
func (m *Manager) SaveSnapshotToDatabase(ctx context.Context, symbol string) error {
	if m.db == nil {
		return fmt.Errorf("database connection not available")
	}

	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	// 保存到数据库
	if err := m.db.SaveOrderBookSnapshot(ctx, symbol, book); err != nil {
		return fmt.Errorf("failed to save order book snapshot to database: %w", err)
	}

	return nil
}

// 新增：从数据库获取最新订单簿快照
func (m *Manager) GetLatestSnapshotFromDatabase(ctx context.Context, symbol string) (*OrderBook, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	snapshot, err := m.db.GetLatestOrderBookSnapshot(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest order book snapshot from database: %w", err)
	}

	if snapshot == nil {
		return nil, nil // 没有找到数据
	}

	// 新增：类型断言和转换逻辑
	if snapshotMap, ok := snapshot.(map[string]interface{}); ok {
		book := NewOrderBook(symbol)

		// 解析时间戳
		if timestampStr, ok := snapshotMap["timestamp"].(string); ok {
			if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				book.Timestamp = timestamp
			}
		}

		// 解析买单数据
		if bidsData, ok := snapshotMap["bids"].([][]float64); ok {
			for _, bidData := range bidsData {
				if len(bidData) >= 2 {
					book.Bids.Update(bidData[0], bidData[1])
				}
			}
		}

		// 解析卖单数据
		if asksData, ok := snapshotMap["asks"].([][]float64); ok {
			for _, askData := range asksData {
				if len(askData) >= 2 {
					book.Asks.Update(askData[0], askData[1])
				}
			}
		}

		return book, nil
	}

	return nil, fmt.Errorf("invalid snapshot data format")
}

// 新增：解析快照数据
func (m *Manager) parseSnapshot(symbol string, snapshot map[string]interface{}) (*OrderBook, error) {
	book := NewOrderBook(symbol)

	// 解析时间戳
	if timestampStr, ok := snapshot["timestamp"].(string); ok {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			book.Timestamp = timestamp
		}
	}

	// 解析买单数据
	if bidsData, ok := snapshot["bids"].([]interface{}); ok {
		for _, bidData := range bidsData {
			if bidMap, ok := bidData.(map[string]interface{}); ok {
				price, _ := bidMap["price"].(float64)
				quantity, _ := bidMap["quantity"].(float64)
				book.Bids.Update(price, quantity)
			}
		}
	}

	// 解析卖单数据
	if asksData, ok := snapshot["asks"].([]interface{}); ok {
		for _, askData := range asksData {
			if askMap, ok := askData.(map[string]interface{}); ok {
				price, _ := askMap["price"].(float64)
				quantity, _ := askMap["quantity"].(float64)
				book.Asks.Update(price, quantity)
			}
		}
	}

	return book, nil
}

// 新增：按时间戳排序订单簿
func (m *Manager) sortOrderBooksByTimestamp(books []*OrderBook) {
	// 使用简单的冒泡排序（对于小数据集足够）
	for i := 0; i < len(books)-1; i++ {
		for j := i + 1; j < len(books); j++ {
			if books[i].Timestamp.After(books[j].Timestamp) {
				books[i], books[j] = books[j], books[i]
			}
		}
	}
}

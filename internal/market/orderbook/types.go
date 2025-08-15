package orderbook

import (
	"sort"
	"sync"
	"time"
)

// Level represents a price level in the order book
type Level struct {
	Price    float64
	Quantity float64
}

// Side represents a side of the order book (bids or asks)
type Side struct {
	levels    []Level
	priceMap  map[float64]int // price -> index in levels
	dirtyFlag bool
	mu        sync.RWMutex
}

// OrderBook represents a full order book for a symbol
type OrderBook struct {
	Symbol    string
	Bids      *Side
	Asks      *Side
	Timestamp time.Time
	mu        sync.RWMutex
}

// NewOrderBook creates a new order book instance
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol: symbol,
		Bids: &Side{
			levels:   make([]Level, 0, 100),
			priceMap: make(map[float64]int),
		},
		Asks: &Side{
			levels:   make([]Level, 0, 100),
			priceMap: make(map[float64]int),
		},
	}
}

// Update updates a price level in the order book
func (s *Side) Update(price, quantity float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, exists := s.priceMap[price]; exists {
		if quantity <= 0 {
			// Remove level
			s.levels = append(s.levels[:idx], s.levels[idx+1:]...)
			delete(s.priceMap, price)
			// Update indices
			for i := idx; i < len(s.levels); i++ {
				s.priceMap[s.levels[i].Price] = i
			}
		} else {
			// Update quantity
			s.levels[idx].Quantity = quantity
		}
	} else if quantity > 0 {
		// Add new level
		s.levels = append(s.levels, Level{Price: price, Quantity: quantity})
		s.priceMap[price] = len(s.levels) - 1
	}
	s.dirtyFlag = true
}

// Sort sorts the levels (bids descending, asks ascending)
func (s *Side) Sort(descending bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dirtyFlag {
		return
	}

	if descending {
		sort.Slice(s.levels, func(i, j int) bool {
			return s.levels[i].Price > s.levels[j].Price
		})
	} else {
		sort.Slice(s.levels, func(i, j int) bool {
			return s.levels[i].Price < s.levels[j].Price
		})
	}

	// Update price map
	for i := range s.levels {
		s.priceMap[s.levels[i].Price] = i
	}
	s.dirtyFlag = false
}

// GetLevels returns sorted price levels
func (s *Side) GetLevels(limit int) []Level {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.levels) {
		limit = len(s.levels)
	}
	result := make([]Level, limit)
	copy(result, s.levels[:limit])
	return result
}

// GetDepth returns the total quantity up to a given price
func (s *Side) GetDepth(price float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var depth float64
	for _, level := range s.levels {
		if level.Price <= price { // for asks
			depth += level.Quantity
		}
	}
	return depth
}

// GetVWAP calculates volume-weighted average price for a given quantity
func (s *Side) GetVWAP(quantity float64) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cumQty, cumValue float64
	for _, level := range s.levels {
		available := level.Quantity
		if cumQty+available >= quantity {
			needed := quantity - cumQty
			cumValue += needed * level.Price
			cumQty += needed
			return cumValue / cumQty, true
		}
		cumValue += available * level.Price
		cumQty += available
	}
	return 0, false
}

// GetMidPrice calculates the mid price from the best bid and ask
func (ob *OrderBook) GetMidPrice() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	ob.Bids.Sort(true)  // descending
	ob.Asks.Sort(false) // ascending

	if len(ob.Bids.levels) == 0 || len(ob.Asks.levels) == 0 {
		return 0
	}

	bestBid := ob.Bids.levels[0].Price
	bestAsk := ob.Asks.levels[0].Price
	return (bestBid + bestAsk) / 2
}

// GetSpread calculates the bid-ask spread
func (ob *OrderBook) GetSpread() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	ob.Bids.Sort(true)  // descending
	ob.Asks.Sort(false) // ascending

	if len(ob.Bids.levels) == 0 || len(ob.Asks.levels) == 0 {
		return 0
	}

	bestBid := ob.Bids.levels[0].Price
	bestAsk := ob.Asks.levels[0].Price
	return bestAsk - bestBid
}

// Update updates the order book with new data
func (ob *OrderBook) Update(bids, asks []Level, timestamp time.Time) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.Timestamp = timestamp

	// Update bids
	for _, bid := range bids {
		ob.Bids.Update(bid.Price, bid.Quantity)
	}
	ob.Bids.Sort(true) // descending

	// Update asks
	for _, ask := range asks {
		ob.Asks.Update(ask.Price, ask.Quantity)
	}
	ob.Asks.Sort(false) // ascending
}

// GetSnapshot returns a snapshot of the order book
func (ob *OrderBook) GetSnapshot(depth int) map[string]interface{} {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return map[string]interface{}{
		"symbol":    ob.Symbol,
		"timestamp": ob.Timestamp,
		"bids":      ob.Bids.GetLevels(depth),
		"asks":      ob.Asks.GetLevels(depth),
	}
}

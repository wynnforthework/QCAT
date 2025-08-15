package hotlist

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/market"
)

// Scanner manages symbol scanning and scoring
type Scanner struct {
	db          *sql.DB
	market      *market.Ingestor
	scores      map[string]*Score
	subscribers []chan *Score
	mu          sync.RWMutex
}

// Score represents a symbol's score and metrics
type Score struct {
	Symbol      string    `json:"symbol"`
	Score       float64   `json:"score"`
	Metrics     Metrics   `json:"metrics"`
	LastScanned time.Time `json:"last_scanned"`
	LastUpdated time.Time `json:"last_updated"`
	IsEnabled   bool      `json:"is_enabled"`
}

// Metrics represents various metrics used for scoring
type Metrics struct {
	// Volume metrics
	Volume24h       float64 `json:"volume_24h"`
	VolumeMA7d      float64 `json:"volume_ma_7d"`
	VolumeStdDev    float64 `json:"volume_std_dev"`
	VolumeChange24h float64 `json:"volume_change_24h"`

	// Price metrics
	PriceChange24h  float64 `json:"price_change_24h"`
	PriceVolatility float64 `json:"price_volatility"`
	PriceTrend      float64 `json:"price_trend"`

	// Funding metrics
	FundingRate      float64 `json:"funding_rate"`
	FundingDeviation float64 `json:"funding_deviation"`
	FundingPredicted float64 `json:"funding_predicted"`

	// Open Interest metrics
	OIChange24h float64 `json:"oi_change_24h"`
	OITrend     float64 `json:"oi_trend"`
	OIRatio     float64 `json:"oi_ratio"`

	// Market metrics
	Liquidity   float64 `json:"liquidity"`
	Spread      float64 `json:"spread"`
	MarketDepth float64 `json:"market_depth"`
}

// NewScanner creates a new scanner
func NewScanner(db *sql.DB, market *market.Ingestor) *Scanner {
	return &Scanner{
		db:          db,
		market:      market,
		scores:      make(map[string]*Score),
		subscribers: make([]chan *Score, 0),
	}
}

// Start starts the scanner
func (s *Scanner) Start(ctx context.Context) error {
	// Load existing scores
	if err := s.loadScores(ctx); err != nil {
		return fmt.Errorf("failed to load scores: %w", err)
	}

	// Start scanning
	go s.scan(ctx)

	return nil
}

// Subscribe subscribes to score updates
func (s *Scanner) Subscribe() chan *Score {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *Score, 100)
	s.subscribers = append(s.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscription
func (s *Scanner) Unsubscribe(ch chan *Score) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// GetScore returns a symbol's score
func (s *Scanner) GetScore(symbol string) (*Score, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	score, exists := s.scores[symbol]
	return score, exists
}

// ListScores returns all scores
func (s *Scanner) ListScores() []*Score {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scores := make([]*Score, 0, len(s.scores))
	for _, score := range s.scores {
		scores = append(scores, score)
	}
	return scores
}

// EnableSymbol enables a symbol for scanning
func (s *Scanner) EnableSymbol(ctx context.Context, symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	score, exists := s.scores[symbol]
	if !exists {
		score = &Score{
			Symbol:      symbol,
			LastScanned: time.Now(),
			LastUpdated: time.Now(),
		}
		s.scores[symbol] = score
	}

	score.IsEnabled = true
	return s.saveScore(ctx, score)
}

// DisableSymbol disables a symbol for scanning
func (s *Scanner) DisableSymbol(ctx context.Context, symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	score, exists := s.scores[symbol]
	if !exists {
		return fmt.Errorf("symbol not found: %s", symbol)
	}

	score.IsEnabled = false
	return s.saveScore(ctx, score)
}

// loadScores loads scores from the database
func (s *Scanner) loadScores(ctx context.Context) error {
	query := `
		SELECT symbol, score, metrics, last_scanned, last_updated, is_enabled
		FROM hotlist
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query scores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var score Score
		var metrics []byte

		if err := rows.Scan(
			&score.Symbol,
			&score.Score,
			&metrics,
			&score.LastScanned,
			&score.LastUpdated,
			&score.IsEnabled,
		); err != nil {
			return fmt.Errorf("failed to scan score: %w", err)
		}

		s.scores[score.Symbol] = &score
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating scores: %w", err)
	}

	return nil
}

// saveScore saves a score to the database
func (s *Scanner) saveScore(ctx context.Context, score *Score) error {
	query := `
		INSERT INTO hotlist (
			symbol, score, metrics, last_scanned, last_updated, is_enabled
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) ON CONFLICT (symbol) DO UPDATE SET
			score = EXCLUDED.score,
			metrics = EXCLUDED.metrics,
			last_scanned = EXCLUDED.last_scanned,
			last_updated = EXCLUDED.last_updated,
			is_enabled = EXCLUDED.is_enabled
	`

	_, err := s.db.ExecContext(ctx, query,
		score.Symbol,
		score.Score,
		score.Metrics,
		score.LastScanned,
		score.LastUpdated,
		score.IsEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to save score: %w", err)
	}

	return nil
}

// scan periodically scans and scores symbols
func (s *Scanner) scan(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scanSymbols(ctx)
		}
	}
}

// scanSymbols scans and scores all enabled symbols
func (s *Scanner) scanSymbols(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, score := range s.scores {
		if !score.IsEnabled {
			continue
		}

		// Calculate metrics
		metrics, err := s.calculateMetrics(ctx, score.Symbol)
		if err != nil {
			log.Printf("Failed to calculate metrics for %s: %v", score.Symbol, err)
			continue
		}

		// Calculate score
		totalScore := s.calculateScore(metrics)

		// Update score
		score.Score = totalScore
		score.Metrics = *metrics
		score.LastScanned = time.Now()
		score.LastUpdated = time.Now()

		// Save score
		if err := s.saveScore(ctx, score); err != nil {
			log.Printf("Failed to save score for %s: %v", score.Symbol, err)
			continue
		}

		// Notify subscribers
		for _, ch := range s.subscribers {
			select {
			case ch <- score:
			default:
				// Channel is full, skip
			}
		}
	}
}

// calculateMetrics calculates metrics for a symbol
func (s *Scanner) calculateMetrics(ctx context.Context, symbol string) (*Metrics, error) {
	metrics := &Metrics{}

	// Calculate volume metrics
	trades, err := s.market.GetTradeHistory(ctx, symbol, time.Now().Add(-7*24*time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}

	var volume24h, volumeTotal float64
	var volumes []float64

	for _, trade := range trades {
		volume := trade.Price * trade.Quantity
		volumes = append(volumes, volume)
		volumeTotal += volume

		if trade.Timestamp.After(time.Now().Add(-24 * time.Hour)) {
			volume24h += volume
		}
	}

	metrics.Volume24h = volume24h
	metrics.VolumeMA7d = volumeTotal / 7
	metrics.VolumeStdDev = calculateStdDev(volumes)
	metrics.VolumeChange24h = (volume24h - metrics.VolumeMA7d) / metrics.VolumeMA7d * 100

	// Calculate price metrics
	klines, err := s.market.GetKlineHistory(ctx, symbol, "1h", time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get kline history: %w", err)
	}

	if len(klines) > 0 {
		first := klines[0]
		last := klines[len(klines)-1]
		metrics.PriceChange24h = (last.Close - first.Open) / first.Open * 100

		var returns []float64
		for i := 1; i < len(klines); i++ {
			ret := (klines[i].Close - klines[i-1].Close) / klines[i-1].Close
			returns = append(returns, ret)
		}
		metrics.PriceVolatility = calculateStdDev(returns) * math.Sqrt(24) * 100
		metrics.PriceTrend = calculateTrend(returns)
	}

	// Calculate funding metrics
	funding, err := s.market.GetFundingRates(ctx, symbol, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get funding rates: %w", err)
	}

	var rates []float64
	for _, rate := range funding {
		rates = append(rates, rate.Rate)
		if rate.NextTime.After(time.Now()) {
			metrics.FundingPredicted = rate.NextRate
		}
	}

	if len(rates) > 0 {
		metrics.FundingRate = rates[len(rates)-1]
		metrics.FundingDeviation = calculateStdDev(rates)
	}

	// Calculate OI metrics
	oi, err := s.market.GetOpenInterest(ctx, symbol, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}

	if len(oi) > 0 {
		first := oi[0]
		last := oi[len(oi)-1]
		metrics.OIChange24h = (last.Value - first.Value) / first.Value * 100
		metrics.OIRatio = last.Value / metrics.Volume24h

		var values []float64
		for _, o := range oi {
			values = append(values, o.Value)
		}
		metrics.OITrend = calculateTrend(values)
	}

	// Calculate market metrics
	book, err := s.market.GetOrderBook(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	if len(book.Bids) > 0 && len(book.Asks) > 0 {
		bestBid := book.Bids[0].Price
		bestAsk := book.Asks[0].Price
		metrics.Spread = (bestAsk - bestBid) / bestBid * 100

		var bidDepth, askDepth float64
		for _, bid := range book.Bids[:10] {
			bidDepth += bid.Price * bid.Quantity
		}
		for _, ask := range book.Asks[:10] {
			askDepth += ask.Price * ask.Quantity
		}
		metrics.MarketDepth = (bidDepth + askDepth) / 2
		metrics.Liquidity = metrics.MarketDepth / metrics.Volume24h
	}

	return metrics, nil
}

// calculateScore calculates the total score from metrics
func (s *Scanner) calculateScore(metrics *Metrics) float64 {
	// Volume score (0-30 points)
	volumeScore := math.Min(30, math.Max(0,
		10*math.Log10(1+metrics.Volume24h/1000000)+ // Base volume score
			10*math.Max(-1, math.Min(1, metrics.VolumeChange24h/100))+ // Volume change score
			10*(1-math.Min(1, metrics.VolumeStdDev/metrics.VolumeMA7d)), // Volume stability score
	))

	// Price score (0-20 points)
	priceScore := math.Min(20, math.Max(0,
		5*math.Abs(metrics.PriceChange24h/10)+ // Price movement score
			10*(1-math.Min(1, metrics.PriceVolatility/100))+ // Volatility score
			5*math.Max(-1, math.Min(1, metrics.PriceTrend)), // Trend score
	))

	// Funding score (0-20 points)
	fundingScore := math.Min(20, math.Max(0,
		10*math.Abs(metrics.FundingRate*100)+ // Current funding score
			5*(1-math.Min(1, metrics.FundingDeviation*10))+ // Funding stability score
			5*math.Abs(metrics.FundingPredicted*100), // Predicted funding score
	))

	// Open Interest score (0-20 points)
	oiScore := math.Min(20, math.Max(0,
		10*math.Max(-1, math.Min(1, metrics.OIChange24h/100))+ // OI change score
			5*math.Max(-1, math.Min(1, metrics.OITrend))+ // OI trend score
			5*(1-math.Min(1, metrics.OIRatio)), // OI/Volume ratio score
	))

	// Market quality score (0-10 points)
	marketScore := math.Min(10, math.Max(0,
		4*(1-math.Min(1, metrics.Spread/0.1))+ // Spread score
			3*math.Min(1, metrics.MarketDepth/1000000)+ // Depth score
			3*math.Min(1, metrics.Liquidity), // Liquidity score
	))

	// Total score (0-100 points)
	return volumeScore + priceScore + fundingScore + oiScore + marketScore
}

// calculateStdDev calculates the standard deviation of a slice of numbers
func calculateStdDev(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0
	}

	// Calculate mean
	var sum float64
	for _, n := range numbers {
		sum += n
	}
	mean := sum / float64(len(numbers))

	// Calculate variance
	var variance float64
	for _, n := range numbers {
		diff := n - mean
		variance += diff * diff
	}
	variance /= float64(len(numbers))

	return math.Sqrt(variance)
}

// calculateTrend calculates the trend of a slice of numbers (-1 to 1)
func calculateTrend(numbers []float64) float64 {
	if len(numbers) < 2 {
		return 0
	}

	// Calculate linear regression slope
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(numbers))

	for i, y := range numbers {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	return math.Max(-1, math.Min(1, slope*float64(len(numbers))))
}

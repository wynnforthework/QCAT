package portfolio

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sync"
	"time"
)

// MABScheduler implements Multi-Armed Bandit scheduling
type MABScheduler struct {
	arms        map[string]*Arm
	totalTrials int
	mu          sync.RWMutex
}

// Arm represents a MAB arm (strategy)
type Arm struct {
	ID           string
	Rewards      []float64 // 收益序列
	TotalReward  float64   // 总收益
	TotalTrials  int       // 总尝试次数
	UCB          float64   // UCB值
	Thompson     float64   // Thompson采样值
	Alpha        float64   // Beta分布参数α
	Beta         float64   // Beta分布参数β
	LastSelected time.Time // 最后选择时间
}

// NewMABScheduler creates a new MAB scheduler
func NewMABScheduler() *MABScheduler {
	return &MABScheduler{
		arms: make(map[string]*Arm),
	}
}

// AddArm adds a new arm
func (s *MABScheduler) AddArm(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.arms[id]; exists {
		return fmt.Errorf("arm already exists: %s", id)
	}

	s.arms[id] = &Arm{
		ID:      id,
		Alpha:   1, // 初始Beta分布参数
		Beta:    1,
		Rewards: make([]float64, 0),
	}

	return nil
}

// UpdateReward updates reward for an arm
func (s *MABScheduler) UpdateReward(id string, reward float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	arm, exists := s.arms[id]
	if !exists {
		return fmt.Errorf("arm not found: %s", id)
	}

	// Update arm statistics
	arm.Rewards = append(arm.Rewards, reward)
	arm.TotalReward += reward
	arm.TotalTrials++
	s.totalTrials++

	// Update Beta distribution parameters
	if reward > 0 {
		arm.Alpha += reward
	} else {
		arm.Beta -= reward
	}

	// Update UCB value
	if s.totalTrials > 0 {
		exploration := math.Sqrt(2 * math.Log(float64(s.totalTrials)) / float64(arm.TotalTrials))
		exploitation := arm.TotalReward / float64(arm.TotalTrials)
		arm.UCB = exploitation + exploration
	}

	// Update Thompson sampling value
	arm.Thompson = Beta(arm.Alpha, arm.Beta)

	return nil
}

// SelectArmUCB selects an arm using UCB1 algorithm
func (s *MABScheduler) SelectArmUCB(ctx context.Context) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.arms) == 0 {
		return "", fmt.Errorf("no arms available")
	}

	// Find arm with highest UCB value
	var bestArm *Arm
	maxUCB := -math.MaxFloat64

	for _, arm := range s.arms {
		// If arm has never been tried, select it
		if arm.TotalTrials == 0 {
			return arm.ID, nil
		}

		if arm.UCB > maxUCB {
			maxUCB = arm.UCB
			bestArm = arm
		}
	}

	return bestArm.ID, nil
}

// SelectArmThompson selects an arm using Thompson sampling
func (s *MABScheduler) SelectArmThompson(ctx context.Context) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.arms) == 0 {
		return "", fmt.Errorf("no arms available")
	}

	// Find arm with highest Thompson sampling value
	var bestArm *Arm
	maxThompson := -math.MaxFloat64

	for _, arm := range s.arms {
		// Sample from Beta distribution
		sample := Beta(arm.Alpha, arm.Beta)
		if sample > maxThompson {
			maxThompson = sample
			bestArm = arm
		}
	}

	return bestArm.ID, nil
}

// GetArmStats returns statistics for an arm
func (s *MABScheduler) GetArmStats(id string) (*Arm, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arm, exists := s.arms[id]
	if !exists {
		return nil, fmt.Errorf("arm not found: %s", id)
	}
	return arm, nil
}

// RemoveArm removes an arm
func (s *MABScheduler) RemoveArm(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.arms[id]; !exists {
		return fmt.Errorf("arm not found: %s", id)
	}

	delete(s.arms, id)
	return nil
}

// GetAllArms returns all arms sorted by UCB value
func (s *MABScheduler) GetAllArms() []*Arm {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arms := make([]*Arm, 0, len(s.arms))
	for _, arm := range s.arms {
		arms = append(arms, arm)
	}

	// Sort by UCB value in descending order (highest UCB first)
	slices.SortFunc(arms, func(a, b *Arm) int {
		if a.UCB > b.UCB {
			return -1
		}
		if a.UCB < b.UCB {
			return 1
		}
		return 0
	})

	return arms
}

// Beta generates a random number from Beta distribution
func Beta(alpha, beta float64) float64 {
	x := Gamma(alpha, 1)
	y := Gamma(beta, 1)
	return x / (x + y)
}

// Gamma generates a random number from Gamma distribution
func Gamma(alpha, beta float64) float64 {
	if alpha < 1 {
		return Gamma(1+alpha, beta) * math.Pow(rand.Float64(), 1.0/alpha)
	}

	d := alpha - 1/3
	c := 1 / math.Sqrt(9*d)

	for {
		x := rand.NormFloat64()
		v := 1 + c*x
		v = v * v * v
		u := rand.Float64()

		if u < 1-0.0331*x*x*x*x ||
			math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v / beta
		}
	}
}

package operations

import "time"

// Domain identifies a logical operational area.
type Domain string

const (
	DomainDoubleLoop   Domain = "double_loop"
	DomainStrategy     Domain = "strategy"
	DomainTrading      Domain = "trading"
	DomainExchangeKeys Domain = "exchange_keys"
	DomainSystem       Domain = "system"
)

// Status represents the health state of a domain or check.
type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
	StatusUnknown  Status = "unknown"
)

// CheckMode specifies when a check should run.
type CheckMode int

const (
	CheckModeStartup CheckMode = iota + 1
	CheckModeRuntime
	CheckModeAll
)

// Result captures the outcome of a single operational check.
type Result struct {
	ID         string                 `json:"id"`
	Domain     Domain                 `json:"domain"`
	Name       string                 `json:"name"`
	Status     Status                 `json:"status"`
	Summary    string                 `json:"summary"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StartedAt  time.Time              `json:"started_at"`
	FinishedAt time.Time              `json:"finished_at"`
	Duration   time.Duration          `json:"duration"`
}

// Check defines the interface every operational check must implement.
type Check interface {
	ID() string
	Domain() Domain
	Mode() CheckMode
	Interval() time.Duration
	Run() Result
}

// DomainSummary aggregates the results for a domain.
type DomainSummary struct {
	Status Status   `json:"status"`
	Checks []Result `json:"checks"`
}

// Snapshot represents the operational state at a point in time.
type Snapshot struct {
	Mode        string                   `json:"mode"`
	GeneratedAt time.Time                `json:"generated_at"`
	Domains     map[Domain]DomainSummary `json:"domains"`
}

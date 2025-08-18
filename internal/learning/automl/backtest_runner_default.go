package automl

import (
    "context"
    "crypto/md5"
    "encoding/binary"
    "math"
    "math/rand"
    "time"
)

// DefaultBacktestRunner provides a lightweight, dependency-free backtest validation.
// It intentionally avoids pulling heavy subsystems (DB, market managers) so that
// the automl package can compile and run even when other packages are in flux.
//
// Design:
// - Deterministically simulates an equity curve using (taskID, strategyName, dataHash)
//   plus the candidate parameters as the RNG seed source.
// - Produces TotalReturn, MaxDrawdown, SharpeRatio consistent enough for gating.
// - When the full backtest engine is ready, this runner can be replaced or extended
//   to call into it while keeping the same interface.
type DefaultBacktestRunner struct {
    // horizon config
    bars int           // number of bars to simulate
    dt   time.Duration // time step per bar

    // return model config
    baseDrift   float64 // baseline drift per bar
    baseVol     float64 // baseline volatility per bar
    riskPenalty float64 // penalty weight on parameter-induced risk
}

// NewDefaultBacktestRunner constructs a runner with sensible defaults.
func NewDefaultBacktestRunner() *DefaultBacktestRunner {
    return &DefaultBacktestRunner{
        bars:        2_000,
        dt:          time.Minute,
        baseDrift:   0.00005, // ~0.5 bp per bar drift
        baseVol:     0.0025,  // 25 bp per bar vol
        riskPenalty: 0.15,
    }
}

// Run executes a deterministic synthetic backtest using inputs as the RNG seed.
func (r *DefaultBacktestRunner) Run(
    ctx context.Context,
    taskID string,
    strategyName string,
    parameters map[string]interface{},
    dataHash string,
) (*BacktestStats, error) {
    seed := r.deriveSeed(taskID, strategyName, parameters, dataHash)
    rng := rand.New(rand.NewSource(int64(seed)))

    returns := make([]float64, 0, r.bars)

    // Parameter influences: translate typical keys into drift/vol adjustments.
    driftAdj, volAdj := r.parameterInfluence(parameters)

    // Simulate log-returns using GBM-style increments, with mild mean-reversion
    // to reduce tail explosion in long horizons.
    mu := r.baseDrift + driftAdj
    sigma := math.Max(1e-6, r.baseVol*(1.0+volAdj))

    price := 1.0
    peak := price
    maxDD := 0.0

    // Precompute for sharpe
    sum := 0.0
    sumSq := 0.0

    for i := 0; i < r.bars; i++ {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // Random shock
        z := rng.NormFloat64()
        // Small mean reversion term around 1.0 price
        reversion := -0.02 * (price - 1.0)
        // Per-bar return
        ret := mu + reversion + sigma*z
        // Cap extreme outliers to keep robust vs. RNG
        if ret > 0.05 {
            ret = 0.05
        } else if ret < -0.05 {
            ret = -0.05
        }
        returns = append(returns, ret)

        // Update equity
        price *= (1.0 + ret)
        if price > peak {
            peak = price
        }
        dd := (peak - price) / peak
        if dd > maxDD {
            maxDD = dd
        }

        // Accumulate moments
        sum += ret
        sumSq += ret * ret
    }

    // Compute stats
    totalReturn := price - 1.0
    sharpe := 0.0
    if n := float64(len(returns)); n > 1 {
        mean := sum / n
        variance := (sumSq/n - mean*mean)
        if variance < 0 {
            variance = 0
        }
        std := math.Sqrt(variance)
        if std > 0 {
            // Annualize assuming ~252 trading days and 24*60 minutes/day equivalent
            // Here we scale by sqrt(N) where N approximates daily bars.
            sharpe = mean / std * math.Sqrt(252)
        }
    }

    // Apply a mild risk penalty based on parameter-implied aggressiveness
    if maxDD > 0 {
        totalReturn = totalReturn - r.riskPenalty*maxDD
    }

    return &BacktestStats{
        TotalReturn: totalReturn * 100.0, // convert to percentage to match outer expectations
        MaxDrawdown: maxDD * 100.0,       // convert to percentage
        SharpeRatio: sharpe,
    }, nil
}

// deriveSeed creates a deterministic seed from identifying inputs and params.
func (r *DefaultBacktestRunner) deriveSeed(
    taskID string,
    strategyName string,
    parameters map[string]interface{},
    dataHash string,
) uint64 {
    // Serialize a compact, order-stable view of parameters
    // We avoid JSON to keep this dependency-free and deterministic.
    hasher := md5.New()
    hasher.Write([]byte(taskID))
    hasher.Write([]byte("|"))
    hasher.Write([]byte(strategyName))
    hasher.Write([]byte("|"))
    hasher.Write([]byte(dataHash))

    // Fold selected parameter keys to stabilize
    // Common keys used by our optimization stubs
    r.foldParam(hasher, parameters, "learning_rate")
    r.foldParam(hasher, parameters, "batch_size")
    r.foldParam(hasher, parameters, "epochs")
    r.foldParam(hasher, parameters, "lookback")
    r.foldParam(hasher, parameters, "risk_budget")
    r.foldParam(hasher, parameters, "threshold")

    sum := hasher.Sum(nil)
    // Take first 8 bytes as uint64
    return binary.LittleEndian.Uint64(sum[:8])
}

func (r *DefaultBacktestRunner) foldParam(h interface{ Write([]byte) (int, error) }, params map[string]interface{}, key string) {
    if v, ok := params[key]; ok {
        switch t := v.(type) {
        case int:
            _ = writeInt(h, int64(t))
        case int32:
            _ = writeInt(h, int64(t))
        case int64:
            _ = writeInt(h, t)
        case float32:
            _ = writeFloat(h, float64(t))
        case float64:
            _ = writeFloat(h, t)
        case string:
            _, _ = h.Write([]byte(t))
        case bool:
            if t {
                _, _ = h.Write([]byte{1})
            } else {
                _, _ = h.Write([]byte{0})
            }
        }
    }
}

func writeInt(h interface{ Write([]byte) (int, error) }, v int64) error {
    var b [8]byte
    binary.LittleEndian.PutUint64(b[:], uint64(v))
    _, err := h.Write(b[:])
    return err
}

func writeFloat(h interface{ Write([]byte) (int, error) }, v float64) error {
    // Convert float64 to uint64 bits to ensure determinism
    bits := math.Float64bits(v)
    var b [8]byte
    binary.LittleEndian.PutUint64(b[:], bits)
    _, err := h.Write(b[:])
    return err
}

// parameterInfluence maps common parameters to drift/volatility adjustments.
func (r *DefaultBacktestRunner) parameterInfluence(params map[string]interface{}) (driftAdj, volAdj float64) {
    // learning_rate: higher rate implies more aggressive updates → higher vol; sweet spot around ~0.005
    if v, ok := toFloat(params["learning_rate"]); ok {
        volAdj += clamp((v-0.005)*20.0, -0.5, 0.8)
        driftAdj += clamp(0.003-(math.Abs(v-0.005))*0.6, -0.004, 0.004)
    }
    // batch_size: larger batch reduces variance but may reduce responsiveness
    if v, ok := toFloat(params["batch_size"]); ok {
        volAdj += clamp(-math.Log10(1.0+v/64.0)*0.3, -0.6, 0.0)
        driftAdj += clamp(math.Log10(1.0+v/128.0)*0.05, 0.0, 0.06)
    }
    // epochs: more training can improve drift until overfit
    if v, ok := toFloat(params["epochs"]); ok {
        driftAdj += clamp((v-150.0)/1500.0, -0.02, 0.02)
        volAdj += clamp((v-200.0)/2000.0, -0.05, 0.05)
    }
    // strategy thresholds often trade off drift vs vol
    if v, ok := toFloat(params["threshold"]); ok {
        driftAdj += clamp((0.5-v)*0.02, -0.02, 0.02)
        volAdj += clamp((v-0.5)*0.2, -0.2, 0.2)
    }
    // risk budget scales vol and can penalize drift slightly
    if v, ok := toFloat(params["risk_budget"]); ok {
        volAdj += clamp(v*0.5, -0.1, 1.0)
        driftAdj -= clamp(v*0.02, 0.0, 0.02)
    }
    return
}

func toFloat(v interface{}) (float64, bool) {
    switch t := v.(type) {
    case float64:
        return t, true
    case float32:
        return float64(t), true
    case int:
        return float64(t), true
    case int64:
        return float64(t), true
    case int32:
        return float64(t), true
    default:
        return 0, false
    }
}

func clamp(x, lo, hi float64) float64 {
    if x < lo {
        return lo
    }
    if x > hi {
        return hi
    }
    return x
}



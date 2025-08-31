package factors

import (
	"fmt"
	"math"
	"sort"
)

// SMAIndicator 简单移动平均指标
type SMAIndicator struct{}

func (sma *SMAIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20 // 默认周期
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for SMA calculation: need %d, got %d", period, len(data))
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算SMA
	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		result[i] = sum / float64(period)
	}

	return result, nil
}

func (sma *SMAIndicator) GetName() string {
	return "SMA"
}

func (sma *SMAIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// EMAIndicator 指数移动平均指标
type EMAIndicator struct{}

func (ema *EMAIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided for EMA calculation")
	}

	result := make([]float64, len(data))
	alpha := 2.0 / (float64(period) + 1.0)

	// 第一个值使用收盘价
	result[0] = data[0].Close

	// 计算EMA
	for i := 1; i < len(data); i++ {
		result[i] = alpha*data[i].Close + (1-alpha)*result[i-1]
	}

	return result, nil
}

func (ema *EMAIndicator) GetName() string {
	return "EMA"
}

func (ema *EMAIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// RSIIndicator 相对强弱指数
type RSIIndicator struct{}

func (rsi *RSIIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 14
	}

	if len(data) < period+1 {
		return nil, fmt.Errorf("insufficient data for RSI calculation: need %d, got %d", period+1, len(data))
	}

	result := make([]float64, len(data))

	// 前period个值设为NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	// 计算价格变化
	gains := make([]float64, len(data)-1)
	losses := make([]float64, len(data)-1)

	for i := 1; i < len(data); i++ {
		change := data[i].Close - data[i-1].Close
		if change > 0 {
			gains[i-1] = change
			losses[i-1] = 0
		} else {
			gains[i-1] = 0
			losses[i-1] = -change
		}
	}

	// 计算初始平均收益和损失
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// 计算RSI
	for i := period; i < len(data); i++ {
		if i > period {
			// 使用Wilder's smoothing
			avgGain = (avgGain*float64(period-1) + gains[i-1]) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + losses[i-1]) / float64(period)
		}

		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	return result, nil
}

func (rsi *RSIIndicator) GetName() string {
	return "RSI"
}

func (rsi *RSIIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// STDDEVIndicator 标准差指标
type STDDEVIndicator struct{}

func (std *STDDEVIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for STDDEV calculation: need %d, got %d", period, len(data))
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算标准差
	for i := period - 1; i < len(data); i++ {
		// 计算均值
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		mean := sum / float64(period)

		// 计算方差
		variance := 0.0
		for j := i - period + 1; j <= i; j++ {
			diff := data[j].Close - mean
			variance += diff * diff
		}
		variance /= float64(period)

		result[i] = math.Sqrt(variance)
	}

	return result, nil
}

func (std *STDDEVIndicator) GetName() string {
	return "STDDEV"
}

func (std *STDDEVIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// MAXIndicator 最大值指标
type MAXIndicator struct{}

func (max *MAXIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for MAX calculation: need %d, got %d", period, len(data))
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动最大值
	for i := period - 1; i < len(data); i++ {
		maxVal := data[i-period+1].High
		for j := i - period + 2; j <= i; j++ {
			if data[j].High > maxVal {
				maxVal = data[j].High
			}
		}
		result[i] = maxVal
	}

	return result, nil
}

func (max *MAXIndicator) GetName() string {
	return "MAX"
}

func (max *MAXIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// MINIndicator 最小值指标
type MINIndicator struct{}

func (min *MINIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for MIN calculation: need %d, got %d", period, len(data))
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动最小值
	for i := period - 1; i < len(data); i++ {
		minVal := data[i-period+1].Low
		for j := i - period + 2; j <= i; j++ {
			if data[j].Low < minVal {
				minVal = data[j].Low
			}
		}
		result[i] = minVal
	}

	return result, nil
}

func (min *MINIndicator) GetName() string {
	return "MIN"
}

func (min *MINIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// MACDIndicator MACD指标
type MACDIndicator struct{}

func (macd *MACDIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	fastPeriod := int(params["fast_period"])
	slowPeriod := int(params["slow_period"])
	signalPeriod := int(params["signal_period"])

	if fastPeriod <= 0 {
		fastPeriod = 12
	}
	if slowPeriod <= 0 {
		slowPeriod = 26
	}
	if signalPeriod <= 0 {
		signalPeriod = 9
	}

	if len(data) < slowPeriod {
		return nil, fmt.Errorf("insufficient data for MACD calculation")
	}

	// 计算快速EMA
	fastEMA := make([]float64, len(data))
	fastAlpha := 2.0 / (float64(fastPeriod) + 1.0)
	fastEMA[0] = data[0].Close
	for i := 1; i < len(data); i++ {
		fastEMA[i] = fastAlpha*data[i].Close + (1-fastAlpha)*fastEMA[i-1]
	}

	// 计算慢速EMA
	slowEMA := make([]float64, len(data))
	slowAlpha := 2.0 / (float64(slowPeriod) + 1.0)
	slowEMA[0] = data[0].Close
	for i := 1; i < len(data); i++ {
		slowEMA[i] = slowAlpha*data[i].Close + (1-slowAlpha)*slowEMA[i-1]
	}

	// 计算MACD线
	macdLine := make([]float64, len(data))
	for i := range macdLine {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	return macdLine, nil
}

func (macd *MACDIndicator) GetName() string {
	return "MACD"
}

func (macd *MACDIndicator) GetRequiredParams() []string {
	return []string{"fast_period", "slow_period", "signal_period"}
}

// RANKIndicator 排名指标
type RANKIndicator struct{}

func (rank *RANKIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for RANK calculation")
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动排名
	for i := period - 1; i < len(data); i++ {
		// 获取当前窗口的值
		window := make([]float64, period)
		for j := 0; j < period; j++ {
			window[j] = data[i-period+1+j].Close
		}

		// 排序并找到当前值的排名
		sorted := make([]float64, period)
		copy(sorted, window)
		sort.Float64s(sorted)

		currentValue := data[i].Close
		rank := 1
		for _, val := range sorted {
			if val < currentValue {
				rank++
			}
		}

		result[i] = float64(rank) / float64(period)
	}

	return result, nil
}

func (rank *RANKIndicator) GetName() string {
	return "RANK"
}

func (rank *RANKIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// DELAYIndicator 滞后指标
type DELAYIndicator struct{}

func (delay *DELAYIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 1
	}

	if len(data) <= period {
		return nil, fmt.Errorf("insufficient data for DELAY calculation")
	}

	result := make([]float64, len(data))

	// 前period个值设为NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	// 计算滞后值
	for i := period; i < len(data); i++ {
		result[i] = data[i-period].Close
	}

	return result, nil
}

func (delay *DELAYIndicator) GetName() string {
	return "DELAY"
}

func (delay *DELAYIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// DELTAIndicator 差分指标
type DELTAIndicator struct{}

func (delta *DELTAIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 1
	}

	if len(data) <= period {
		return nil, fmt.Errorf("insufficient data for DELTA calculation")
	}

	result := make([]float64, len(data))

	// 前period个值设为NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	// 计算差分
	for i := period; i < len(data); i++ {
		result[i] = data[i].Close - data[i-period].Close
	}

	return result, nil
}

func (delta *DELTAIndicator) GetName() string {
	return "DELTA"
}

func (delta *DELTAIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// TSSUMIndicator 时序求和指标
type TSSUMIndicator struct{}

func (tssum *TSSUMIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for TS_SUM calculation")
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动求和
	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		result[i] = sum
	}

	return result, nil
}

func (tssum *TSSUMIndicator) GetName() string {
	return "TS_SUM"
}

func (tssum *TSSUMIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// TSMAXIndicator 时序最大值指标
type TSMAXIndicator struct{}

func (tsmax *TSMAXIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for TS_MAX calculation")
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动最大值
	for i := period - 1; i < len(data); i++ {
		maxVal := data[i-period+1].Close
		for j := i - period + 2; j <= i; j++ {
			if data[j].Close > maxVal {
				maxVal = data[j].Close
			}
		}
		result[i] = maxVal
	}

	return result, nil
}

func (tsmax *TSMAXIndicator) GetName() string {
	return "TS_MAX"
}

func (tsmax *TSMAXIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// TSMINIndicator 时序最小值指标
type TSMINIndicator struct{}

func (tsmin *TSMINIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	if period <= 0 {
		period = 20
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for TS_MIN calculation")
	}

	result := make([]float64, len(data))

	// 前period-1个值设为NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	// 计算滚动最小值
	for i := period - 1; i < len(data); i++ {
		minVal := data[i-period+1].Close
		for j := i - period + 2; j <= i; j++ {
			if data[j].Close < minVal {
				minVal = data[j].Close
			}
		}
		result[i] = minVal
	}

	return result, nil
}

func (tsmin *TSMINIndicator) GetName() string {
	return "TS_MIN"
}

func (tsmin *TSMINIndicator) GetRequiredParams() []string {
	return []string{"period"}
}

// BBUpperIndicator 布林带上轨指标
type BBUpperIndicator struct{}

func (bb *BBUpperIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	stdMultiplier := params["std"]

	if period <= 0 {
		period = 20
	}
	if stdMultiplier <= 0 {
		stdMultiplier = 2.0
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for BB_UPPER calculation")
	}

	// 计算SMA
	smaIndicator := &SMAIndicator{}
	sma, err := smaIndicator.Calculate(data, map[string]float64{"period": float64(period)})
	if err != nil {
		return nil, err
	}

	// 计算标准差
	stdIndicator := &STDDEVIndicator{}
	std, err := stdIndicator.Calculate(data, map[string]float64{"period": float64(period)})
	if err != nil {
		return nil, err
	}

	// 计算上轨
	result := make([]float64, len(data))
	for i := range result {
		if math.IsNaN(sma[i]) || math.IsNaN(std[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = sma[i] + stdMultiplier*std[i]
		}
	}

	return result, nil
}

func (bb *BBUpperIndicator) GetName() string {
	return "BB_UPPER"
}

func (bb *BBUpperIndicator) GetRequiredParams() []string {
	return []string{"period", "std"}
}

// BBLowerIndicator 布林带下轨指标
type BBLowerIndicator struct{}

func (bb *BBLowerIndicator) Calculate(data []MarketData, params map[string]float64) ([]float64, error) {
	period := int(params["period"])
	stdMultiplier := params["std"]

	if period <= 0 {
		period = 20
	}
	if stdMultiplier <= 0 {
		stdMultiplier = 2.0
	}

	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for BB_LOWER calculation")
	}

	// 计算SMA
	smaIndicator := &SMAIndicator{}
	sma, err := smaIndicator.Calculate(data, map[string]float64{"period": float64(period)})
	if err != nil {
		return nil, err
	}

	// 计算标准差
	stdIndicator := &STDDEVIndicator{}
	std, err := stdIndicator.Calculate(data, map[string]float64{"period": float64(period)})
	if err != nil {
		return nil, err
	}

	// 计算下轨
	result := make([]float64, len(data))
	for i := range result {
		if math.IsNaN(sma[i]) || math.IsNaN(std[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = sma[i] - stdMultiplier*std[i]
		}
	}

	return result, nil
}

func (bb *BBLowerIndicator) GetName() string {
	return "BB_LOWER"
}

func (bb *BBLowerIndicator) GetRequiredParams() []string {
	return []string{"period", "std"}
}

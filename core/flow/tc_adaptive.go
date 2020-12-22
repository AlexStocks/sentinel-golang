package flow

import "github.com/alibaba/sentinel-golang/core/system_metric"

// MemoryAdaptiveTrafficShapingCalculator is a memory adaptive traffic shaping calculator
//
// adaptive flow control algorithm
// If the watermark is less than Rule.LowWaterMark, the threshold is Rule.SafeThreshold.
// If the watermark is greater than Rule.HighWaterMark, the threshold is Rule.RiskThreshold.
// Otherwise, the threshold is ((watermark - LowWaterMark)/(HighWaterMark - LowWaterMark)) *
//	(SafeThreshold - RiskThreshold) + RiskThreshold.
type MemoryAdaptiveTrafficShapingCalculator struct {
	owner         *TrafficShapingController
	safeThreshold int64
	riskThreshold int64
	lowWaterMark  int64
	highWaterMark int64
}

func NewMemoryAdaptiveTrafficShapingCalculator(owner *TrafficShapingController, r *Rule) *MemoryAdaptiveTrafficShapingCalculator {
	return &MemoryAdaptiveTrafficShapingCalculator{
		owner:         owner,
		safeThreshold: r.SafeThreshold,
		riskThreshold: r.RiskThreshold,
		lowWaterMark:  r.LowWaterMark,
		highWaterMark: r.HighWaterMark,
	}
}

func (m *MemoryAdaptiveTrafficShapingCalculator) BoundOwner() *TrafficShapingController {
	return m.owner
}

func (m *MemoryAdaptiveTrafficShapingCalculator) CalculateAllowedTokens(_ uint32, _ int32) float64 {
	var threshold float64
	mem := system_metric.CurrentMemoryUsage()
	if mem <= m.lowWaterMark {
		threshold = float64(m.safeThreshold)
	} else if mem >= m.highWaterMark {
		threshold = float64(m.riskThreshold)
	} else {
		threshold = float64(mem-m.lowWaterMark)/float64(m.highWaterMark-m.lowWaterMark)*
			float64(m.safeThreshold-m.riskThreshold) + float64(m.riskThreshold)
	}
	return threshold
}

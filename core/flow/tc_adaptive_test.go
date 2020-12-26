package flow

import (
	"testing"

	"github.com/alibaba/sentinel-golang/core/system_metric"
	"github.com/stretchr/testify/assert"
)

func TestMemoryAdaptiveTrafficShapingCalculator_CalculateAllowedTokens(t *testing.T) {
	tc1 := &MemoryAdaptiveTrafficShapingCalculator{
		owner:         nil,
		safeThreshold: 1000,
		riskThreshold: 100,
		lowWaterMark:  1024,
		highWaterMark: 2048,
	}
	system_metric.SetSystemMemoryUsage(100)
	assert.True(t, int64(tc1.CalculateAllowedTokens(0, 0)) == tc1.safeThreshold)
	system_metric.SetSystemMemoryUsage(1024)
	assert.True(t, int64(tc1.CalculateAllowedTokens(0, 0)) == tc1.safeThreshold)
	system_metric.SetSystemMemoryUsage(1536)
	assert.True(t, int64(tc1.CalculateAllowedTokens(0, 0)) == 550)
	system_metric.SetSystemMemoryUsage(2048)
	assert.True(t, int64(tc1.CalculateAllowedTokens(0, 0)) == 100)
	system_metric.SetSystemMemoryUsage(3072)
	assert.True(t, int64(tc1.CalculateAllowedTokens(0, 0)) == 100)
}

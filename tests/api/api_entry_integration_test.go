package api

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/core/system_metric"
	"github.com/stretchr/testify/assert"
)

func TestAdaptiveFlowControl(t *testing.T) {
	if err := api.InitDefault(); err != nil {
		t.Error(err)
	}
	rs := "hello0"
	rule := flow.Rule{
		Resource:               rs,
		TokenCalculateStrategy: flow.Direct,
		ControlBehavior:        flow.Reject,
		Threshold:              3,
		RelationStrategy:       flow.CurrentResource,
		StatIntervalInMs:       0,
		SafeThreshold:          2,
		RiskThreshold:          1,
		LowWaterMark:           1 * 1024,
		HighWaterMark:          20 * 1024,
	}

	t.Log("start to test memory adaptive flow control")
	rule1 := rule
	num := 3
	rule1.Threshold = float64(num)
	ok, err := flow.LoadRules([]*flow.Rule{&rule1})
	assert.True(t, ok)
	assert.Nil(t, err)

	for i := 0; i < num; i++ {
		entry, blockError := api.Entry(rs, api.WithTrafficType(base.Inbound))
		assert.Nil(t, blockError)
		if blockError != nil {
			t.Errorf("entry error:%+v", blockError)
		}
		entry.Exit()
	}

	_, blockError := api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)
	if blockError != nil {
		t.Logf("entry error:%+v", blockError)
	}

	time.Sleep(1.5e9)
	memSize, err := system_metric.GetProcessMemoryStat()
	assert.Nil(t, err)

	t.Log("\nstart to test memory based adaptive flow control")
	rule2 := rule
	rule2.TokenCalculateStrategy = flow.AdaptiveMemory
	rule2.SafeThreshold = 10
	rule2.RiskThreshold = 3
	rule2.LowWaterMark = memSize + 300*1024
	rule2.HighWaterMark = memSize + 800*1024
	ok, err = flow.LoadRules([]*flow.Rule{&rule2})
	assert.True(t, ok)
	assert.Nil(t, err)
	entry, blockError := api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.Nil(t, blockError)
	entry.Exit()

	// + 80k
	system_metric.SetSystemMemoryUsage(memSize + 80*1024)
	entry, blockError = api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.Nil(t, blockError)
	entry.Exit()

	// + 400k
	time.Sleep(1.2e9)
	system_metric.SetSystemMemoryUsage(memSize + 400*1024)
	for i := 0; i < int(rule2.SafeThreshold); i++ {
		entry, blockError = api.Entry(rs, api.WithTrafficType(base.Inbound))
		if blockError == nil {
			entry.Exit()
		}
	}
	_, blockError = api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)

	// + 1MB
	time.Sleep(1.2e9)
	system_metric.SetSystemMemoryUsage(memSize + 1024*1024)
	for i := 0; i < int(rule2.RiskThreshold); i++ {
		entry, blockError = api.Entry(rs, api.WithTrafficType(base.Inbound))
		assert.Nil(t, blockError)
		entry.Exit()
	}
	_, blockError = api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)
}

func TestAdaptiveFlowControl2(t *testing.T) {
	debug.SetGCPercent(-1)

	if err := api.InitDefault(); err != nil {
		t.Error(err)
	}

	rs := "hello0"
	rule := flow.Rule{
		ID:            "",
		Resource:      rs,
		Threshold:     2000,
		SafeThreshold: 150,
		RiskThreshold: 10,
		LowWaterMark:  100998840320,
		HighWaterMark: 268435456000,
	}

	t.Log("start to test flow control")
	rule1 := rule
	rule1.DebugMode = true
	rule1.TokenCalculateStrategy = flow.AdaptiveMemory
	ok, err := flow.LoadRules([]*flow.Rule{&rule1})
	assert.True(t, ok)
	assert.Nil(t, err)
	system_metric.SetSystemMemoryUsage(136794800128)
	_, blockError := api.Entry(rs, api.WithTrafficType(base.Inbound))
	assert.Nil(t, blockError)
}

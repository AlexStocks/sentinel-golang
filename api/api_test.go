// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"runtime/debug"
	"testing"
	"time"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/core/system_metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type prepareSlotMock struct {
	mock.Mock
}

func (m *prepareSlotMock) Name() string {
	return "mock-sentinel-prepare-slot"
}

func (m *prepareSlotMock) Order() uint32 {
	return 0
}

func (m *prepareSlotMock) Prepare(ctx *base.EntryContext) {
	m.Called(ctx)
	return
}

type mockRuleCheckSlot1 struct {
	mock.Mock
}

func (m *mockRuleCheckSlot1) Name() string {
	return "mock-sentinel-rule-check-slot1"
}

func (m *mockRuleCheckSlot1) Order() uint32 {
	return 0
}

func (m *mockRuleCheckSlot1) Check(ctx *base.EntryContext) *base.TokenResult {
	arg := m.Called(ctx)
	return arg.Get(0).(*base.TokenResult)
}

type mockRuleCheckSlot2 struct {
	mock.Mock
}

func (m *mockRuleCheckSlot2) Name() string {
	return "mock-sentinel-rule-check-slot2"
}

func (m *mockRuleCheckSlot2) Order() uint32 {
	return 0
}

func (m *mockRuleCheckSlot2) Check(ctx *base.EntryContext) *base.TokenResult {
	arg := m.Called(ctx)
	return arg.Get(0).(*base.TokenResult)
}

type statisticSlotMock struct {
	mock.Mock
}

func (m *statisticSlotMock) Name() string {
	return "mock-sentinel-stat-check-slot"
}

func (m *statisticSlotMock) Order() uint32 {
	return 0
}

func (m *statisticSlotMock) OnEntryPassed(ctx *base.EntryContext) {
	m.Called(ctx)
	return
}
func (m *statisticSlotMock) OnEntryBlocked(ctx *base.EntryContext, blockError *base.BlockError) {
	m.Called(ctx, blockError)
	return
}
func (m *statisticSlotMock) OnCompleted(ctx *base.EntryContext) {
	m.Called(ctx)
	return
}

func Test_entryWithArgsAndChainPass(t *testing.T) {
	sc := base.NewSlotChain()
	ps1 := &prepareSlotMock{}
	rcs1 := &mockRuleCheckSlot1{}
	rcs2 := &mockRuleCheckSlot2{}
	ssm := &statisticSlotMock{}
	sc.AddStatPrepareSlot(ps1)
	sc.AddRuleCheckSlot(rcs1)
	sc.AddRuleCheckSlot(rcs2)
	sc.AddStatSlot(ssm)

	ps1.On("Prepare", mock.Anything).Return()
	rcs1.On("Check", mock.Anything).Return(base.NewTokenResultPass())
	rcs2.On("Check", mock.Anything).Return(base.NewTokenResultPass())
	ssm.On("OnEntryPassed", mock.Anything).Return()
	ssm.On("OnCompleted", mock.Anything).Return()

	entry, b := entry("abc", &EntryOptions{
		resourceType: base.ResTypeCommon,
		entryType:    base.Inbound,
		batchCount:   1,
		flag:         0,
		slotChain:    sc,
	})
	assert.Nil(t, b, "the entry should not be blocked")
	assert.Equal(t, "abc", entry.Resource().Name())

	entry.Exit()

	ps1.AssertNumberOfCalls(t, "Prepare", 1)
	rcs1.AssertNumberOfCalls(t, "Check", 1)
	rcs2.AssertNumberOfCalls(t, "Check", 1)
	ssm.AssertNumberOfCalls(t, "OnEntryPassed", 1)
	ssm.AssertNumberOfCalls(t, "OnEntryBlocked", 0)
	ssm.AssertNumberOfCalls(t, "OnCompleted", 1)
}

func Test_entryWithArgsAndChainBlock(t *testing.T) {
	sc := base.NewSlotChain()
	ps1 := &prepareSlotMock{}
	rcs1 := &mockRuleCheckSlot1{}
	rcs2 := &mockRuleCheckSlot2{}
	ssm := &statisticSlotMock{}
	sc.AddStatPrepareSlot(ps1)
	sc.AddRuleCheckSlot(rcs1)
	sc.AddRuleCheckSlot(rcs2)
	sc.AddStatSlot(ssm)

	blockType := base.BlockTypeFlow

	ps1.On("Prepare", mock.Anything).Return()
	rcs1.On("Check", mock.Anything).Return(base.NewTokenResultBlocked(blockType))
	rcs2.On("Check", mock.Anything).Return(base.NewTokenResultPass())
	ssm.On("OnEntryPassed", mock.Anything).Return()
	ssm.On("OnEntryBlocked", mock.Anything, mock.Anything).Return()
	ssm.On("OnCompleted", mock.Anything).Return()

	entry, b := entry("abc", &EntryOptions{
		resourceType: base.ResTypeCommon,
		entryType:    base.Inbound,
		batchCount:   1,
		flag:         0,
		slotChain:    sc,
	})
	assert.Nil(t, entry)
	assert.NotNil(t, b)
	assert.Equal(t, blockType, b.BlockType())

	ps1.AssertNumberOfCalls(t, "Prepare", 1)
	rcs1.AssertNumberOfCalls(t, "Check", 1)
	rcs2.AssertNumberOfCalls(t, "Check", 0)
	ssm.AssertNumberOfCalls(t, "OnEntryPassed", 0)
	ssm.AssertNumberOfCalls(t, "OnEntryBlocked", 1)
	ssm.AssertNumberOfCalls(t, "OnCompleted", 0)
}

func TestAdaptiveFlowControl(t *testing.T) {
	debug.SetGCPercent(-1)
	InitDefault()

	rs := "hello0"
	rule := flow.Rule{
		ID:                     "",
		Resource:               rs,
		TokenCalculateStrategy: 0,
		ControlBehavior:        0,
		Threshold:              3,
		RelationStrategy:       0,
		RefResource:            "",
		MaxQueueingTimeMs:      0,
		WarmUpPeriodSec:        0,
		WarmUpColdFactor:       0,
		StatIntervalInMs:       0,
		AdaptiveEnable:         false,
		AdaptiveType:           flow.Memory,
		SafeThreshold:          2,
		RiskThreshold:          1,
		LowWaterMark:           1 * 1024,
		HighWaterMark:          20 * 1024,
	}

	t.Log("start to test flow control")
	rule1 := rule
	num := 3
	rule1.Threshold = float64(num)
	ok, err := flow.LoadRules([]*flow.Rule{&rule1})
	assert.True(t, ok)
	assert.Nil(t, err)

	for i := 0; i < num; i++ {
		fmt.Printf("loop index %d\n", i)
		entry, blockError := Entry(rs, WithTrafficType(base.Inbound))
		assert.Nil(t, blockError)
		if blockError != nil {
			t.Errorf("entry error:%+v", blockError)
		}
		entry.Exit()
	}
	_, blockError := Entry(rs, WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)
	if blockError != nil {
		t.Logf("entry error:%+v", blockError)
	}

	time.Sleep(1.5e9)
	memSize, err := system_metric.GetProcessMemoryStat()
	assert.Nil(t, err)

	t.Log("\nstart to test memory based adaptive flow control")
	rule2 := rule
	rule2.AdaptiveEnable = true
	rule2.AdaptiveType = flow.Memory
	rule2.SafeThreshold = 10
	rule2.RiskThreshold = 1
	rule2.LowWaterMark = uint64(memSize) + 300*1024
	rule2.HighWaterMark = uint64(memSize) + 800*1024
	ok, err = flow.LoadRules([]*flow.Rule{&rule2})
	assert.True(t, ok)
	assert.Nil(t, err)
	entry, blockError := Entry(rs, WithTrafficType(base.Inbound))
	assert.Nil(t, blockError)
	entry.Exit()

	// + 80k
	num = 10 * 1024
	arr := make([]int32, num)
	for i := 0; i < num; i++ {
		arr[i] = int32(i)
	}
	time.Sleep(time.Duration((config.DefaultMemoryStatCollectIntervalMs + 10) * 1e6))
	entry, blockError = Entry(rs, WithTrafficType(base.Inbound))
	assert.Nil(t, blockError)
	entry.Exit()

	// + 400k
	num = 100 * 1024
	arr = make([]int32, num)
	for i := 0; i < num; i++ {
		arr[i] = int32(i)
	}
	time.Sleep(time.Duration((config.DefaultMemoryStatCollectIntervalMs + 10) * 1e6))
	//for i := 0; i < 2; i++ {
	//	entry, blockError = Entry(rs, WithTrafficType(base.Inbound))
	//	assert.Nil(t, blockError)
	//	entry.Exit()
	//}
	for i := 0; i < int(rule2.SafeThreshold); i++ {
		entry, blockError = Entry(rs, WithTrafficType(base.Inbound))
		if blockError == nil {
			entry.Exit()
		}
	}
	_, blockError = Entry(rs, WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)

	// + 1MB
	num = 256 * 1024
	arr = make([]int32, num)
	for i := 0; i < num; i++ {
		arr[i] = int32(i)
	}
	time.Sleep(1.4e9)
	for i := 0; i < int(rule2.RiskThreshold); i++ {
		entry, blockError = Entry(rs, WithTrafficType(base.Inbound))
		assert.Nil(t, blockError)
		entry.Exit()
	}
	_, blockError = Entry(rs, WithTrafficType(base.Inbound))
	assert.NotNil(t, blockError)
}

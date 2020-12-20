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

package system_metric

import (
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/metrics"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

const (
	NotRetrievedLoadValue     float64 = -1.0
	NotRetrievedCpuUsageValue float64 = -1.0
	NotRetrievedMemoryValue   int64   = -1
)

var (
	currentLoad        atomic.Value
	currentCpuUsage    atomic.Value
	currentMemoryUsage atomic.Value

	prevCpuStat    *cpu.TimesStat
	initOnce       sync.Once
	memoryStatOnce sync.Once

	CurrentPID               = os.Getpid()
	TotalMemorySize, _, _, _ = getMemoryStat()

	ssStopChan = make(chan struct{})
)

func init() {
	currentLoad.Store(NotRetrievedLoadValue)
	currentCpuUsage.Store(NotRetrievedCpuUsageValue)
	currentMemoryUsage.Store(NotRetrievedMemoryValue)
}

func getMemoryStat() (total, used, free uint64, usedPercent float64) {
	stat, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0, 0
	}

	return stat.Total, stat.Used, stat.Free, stat.UsedPercent
}

// GetProcessMemoryStat gets current process's memory usage in Bytes
func GetProcessMemoryStat() (int64, error) {
	p, err := process.NewProcess(int32(CurrentPID))
	if err != nil {
		return 0, err
	}

	memInfo, err := p.MemoryInfo()
	var rss int64
	if memInfo != nil {
		rss = int64(memInfo.RSS)
	}

	return rss, err
}

func InitCollector(intervalMs uint32) {
	if intervalMs == 0 {
		return
	}
	initOnce.Do(func() {
		// Initial retrieval.
		retrieveAndUpdateSystemStat()

		ticker := util.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		go util.RunWithRecover(func() {
			for {
				select {
				case <-ticker.C():
					retrieveAndUpdateSystemStat()
				case <-ssStopChan:
					ticker.Stop()
					return
				}
			}
		})
	})
}

func retrieveAndUpdateSystemStat() {
	cpuStats, err := cpu.Times(false)
	if err != nil {
		logging.Warn("[retrieveAndUpdateSystemStat] Failed to retrieve current CPU usage", "err", err.Error())
	}
	loadStat, err := load.Avg()
	if err != nil {
		logging.Warn("[retrieveAndUpdateSystemStat] Failed to retrieve current system load", "err", err.Error())
	}
	if len(cpuStats) > 0 {
		curCpuStat := &cpuStats[0]
		recordCpuUsage(prevCpuStat, curCpuStat)
		// Cache the latest CPU stat info.
		prevCpuStat = curCpuStat
	}
	if loadStat != nil {
		currentLoad.Store(loadStat.Load1)
	}
}

func InitMemoryCollector(intervalMs uint32) {
	if intervalMs == 0 {
		return
	}
	memoryStatOnce.Do(func() {
		// Initial memory retrieval.
		retrieveAndUpdateMemoryStat()

		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		go util.RunWithRecover(func() {
			for {
				select {
				case <-ticker.C:
					retrieveAndUpdateMemoryStat()
				case <-ssStopChan:
					ticker.Stop()
					return
				}
			}
		})
	})
}

func retrieveAndUpdateMemoryStat() {
	memoryUsedBytes, err := GetProcessMemoryStat()
	if err == nil {
		metrics.SetProcessMemorySize(memoryUsedBytes)
		currentMemoryUsage.Store(memoryUsedBytes)
	}
}

func recordCpuUsage(prev, curCpuStat *cpu.TimesStat) {
	if prev != nil && curCpuStat != nil {
		prevTotal := calculateTotalCpuTick(prev)
		curTotal := calculateTotalCpuTick(curCpuStat)

		tDiff := curTotal - prevTotal
		var cpuUsage float64
		if tDiff == 0 {
			cpuUsage = 0
		} else {
			prevUsed := calculateUserCpuTick(prev) + calculateKernelCpuTick(prev)
			curUsed := calculateUserCpuTick(curCpuStat) + calculateKernelCpuTick(curCpuStat)
			cpuUsage = (curUsed - prevUsed) / tDiff
			cpuUsage = math.Max(0.0, cpuUsage)
			cpuUsage = math.Min(1.0, cpuUsage)
		}
		metrics.SetCPURatio(cpuUsage)
		currentCpuUsage.Store(cpuUsage)
	}
}

func calculateTotalCpuTick(stat *cpu.TimesStat) float64 {
	return stat.User + stat.Nice + stat.System + stat.Idle +
		stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
}

func calculateUserCpuTick(stat *cpu.TimesStat) float64 {
	return stat.User + stat.Nice
}

func calculateKernelCpuTick(stat *cpu.TimesStat) float64 {
	return stat.System + stat.Irq + stat.Softirq
}

func CurrentLoad() float64 {
	r, ok := currentLoad.Load().(float64)
	if !ok {
		return NotRetrievedLoadValue
	}
	return r
}

// Note: SetSystemLoad is used for unit test, the user shouldn't call this function.
func SetSystemLoad(load float64) {
	currentLoad.Store(load)
}

func CurrentCpuUsage() float64 {
	r, ok := currentCpuUsage.Load().(float64)
	if !ok {
		return NotRetrievedCpuUsageValue
	}
	return r
}

// Note: SetSystemCpuUsage is used for unit test, the user shouldn't call this function.
func SetSystemCpuUsage(cpuUsage float64) {
	currentCpuUsage.Store(cpuUsage)
}

func CurrentMemoryUsage() int64 {
	bytes, ok := currentMemoryUsage.Load().(int64)
	if !ok {
		return NotRetrievedMemoryValue
	}

	return bytes
}

// Note: SetSystemCpuUsage is used for unit test, the user shouldn't call this function.
func SetSystemMemoryUsage(memoryUsage int64) {
	currentMemoryUsage.Store(memoryUsage)
}

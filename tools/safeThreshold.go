/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2021-01-25 17:27
# FILE    : a.go
******************************************************/

package main

import (
	"flag"
	"fmt"
)

func main() {
	var (
		totalMemory int64

		sentinelThreshold int64
		safeThreshold     int64
		riskThreshold     int64
		riskParam         float64
		safeParam         float64

		mem           int64
		lowWaterMark  float64
		highWaterMark float64

		k float64
	)

	flag.Float64Var(&riskParam, "risk", 0.667, "危险系数")
	flag.Float64Var(&safeParam, "safe", 0.333, "安全系数")

	flag.Int64Var(&totalMemory, "total", 100, "总内存")
	highWaterMark = riskParam * totalMemory
	lowWaterMark = safeParam * totalMemory
	mem = 0.5 * totalMemory

	flag.Int64Var(&sentinelThreshold, "sentinel", 10, "sentinel限流值")
	// safeThreshold = 2
	riskThreshold = 2

	k = (float64(mem) - float64(lowWaterMark)) / float64(highWaterMark-lowWaterMark)
	safeThreshold = (sentinelThreshold - k*float64(riskThreshold)) / (1 - k)

	fmt.Println("safeThreshold:", safeThreshold)
}

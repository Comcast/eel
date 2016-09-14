/**
 * Copyright 2015 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"sync/atomic"
	"time"
)

type ServiceStats struct {
	InCount           uint64
	OutCount          uint64
	ErrorCount        uint64
	TotalTimeInternal uint64
	TotalTimeExternal uint64
	TotalBytesIn      uint64
	TotalBytesOut     uint64
}

func AddLatencyLog(ctx Context, stats *ServiceStats, key string) {
	duration := int64(0)
	startTime := int64(0)
	if ctx.Value("start_ts") != nil {
		startTime = (ctx.Value("start_ts")).(int64)
	}
	if startTime > 0 {
		duration = time.Now().UnixNano() - startTime
	}
	ctx.AddLogValue("key", duration/1e6)
	stats.IncTimeInternal(duration)

}

func (stats *ServiceStats) Clone() *ServiceStats {
	clone := ServiceStats{}
	clone.InCount = atomic.LoadUint64(&stats.InCount)
	clone.OutCount = atomic.LoadUint64(&stats.OutCount)
	clone.ErrorCount = atomic.LoadUint64(&stats.ErrorCount)
	clone.TotalTimeInternal = atomic.LoadUint64(&stats.TotalTimeInternal)
	clone.TotalTimeExternal = atomic.LoadUint64(&stats.TotalTimeExternal)
	clone.TotalBytesIn = atomic.LoadUint64(&stats.TotalBytesIn)
	clone.TotalBytesOut = atomic.LoadUint64(&stats.TotalBytesOut)
	return &clone
}

func (stats *ServiceStats) Add(src *ServiceStats) *ServiceStats {
	atomic.AddUint64(&stats.InCount, src.InCount)
	atomic.AddUint64(&stats.OutCount, src.OutCount)
	atomic.AddUint64(&stats.ErrorCount, src.ErrorCount)
	atomic.AddUint64(&stats.TotalTimeInternal, src.TotalTimeInternal)
	atomic.AddUint64(&stats.TotalTimeExternal, src.TotalTimeExternal)
	atomic.AddUint64(&stats.TotalBytesIn, src.TotalBytesIn)
	atomic.AddUint64(&stats.TotalBytesOut, src.TotalBytesOut)
	return stats
}

func (stats *ServiceStats) Subtract(src *ServiceStats) *ServiceStats {
	atomic.AddUint64(&stats.InCount, -src.InCount)
	atomic.AddUint64(&stats.OutCount, -src.OutCount)
	atomic.AddUint64(&stats.ErrorCount, -src.ErrorCount)
	atomic.AddUint64(&stats.TotalTimeInternal, -src.TotalTimeInternal)
	atomic.AddUint64(&stats.TotalTimeExternal, -src.TotalTimeExternal)
	atomic.AddUint64(&stats.TotalBytesIn, -src.TotalBytesIn)
	atomic.AddUint64(&stats.TotalBytesOut, -src.TotalBytesOut)
	return stats
}

func (stats *ServiceStats) Reset() {
	atomic.StoreUint64(&stats.InCount, 0)
	atomic.StoreUint64(&stats.OutCount, 0)
	atomic.StoreUint64(&stats.ErrorCount, 0)
	atomic.StoreUint64(&stats.TotalTimeInternal, 0)
	atomic.StoreUint64(&stats.TotalTimeExternal, 0)
	atomic.StoreUint64(&stats.TotalBytesIn, 0)
	atomic.StoreUint64(&stats.TotalBytesOut, 0)
}

func (stats *ServiceStats) IncErrors() {
	atomic.AddUint64(&stats.ErrorCount, 1)
}

func (stats *ServiceStats) IncInCount() {
	atomic.AddUint64(&stats.InCount, 1)
}

func (stats *ServiceStats) IncOutCount() {
	atomic.AddUint64(&stats.OutCount, 1)
}

func (stats *ServiceStats) IncTimeInternal(nanos int64) {
	atomic.AddUint64(&stats.TotalTimeInternal, uint64(nanos))
}

func (stats *ServiceStats) IncTimeExternal(nanos int64) {
	atomic.AddUint64(&stats.TotalTimeExternal, uint64(nanos))
}

func (stats *ServiceStats) IncBytesIn(size int) {
	atomic.AddUint64(&stats.TotalBytesIn, uint64(size))
}

func (stats *ServiceStats) IncBytesOut(size int) {
	atomic.AddUint64(&stats.TotalBytesOut, uint64(size))
}

type propFunc func() int

// StatsLoop logs some basic stats at pre-defined interval.
// If iterations is negative, the loop is endless.  Otherwise the loop
// terminates after the specified number of iterations.
func (stats *ServiceStats) StatsLoop(ctx Context, interval time.Duration, iterations int, label string, getWorkQueueFillLevel propFunc, getNumWorkersIdle propFunc) {
	backup := new(ServiceStats)
	for i := 0; iterations < 0 || i < iterations; i++ {
		clone := stats.Clone()
		clone2 := clone.Clone()
		ctx.AddValue(label, clone.Subtract(backup))
		ctx.Log().Metric("eel_service_stats", "label", label,
			"InCount", clone.InCount,
			"OutCount", clone.OutCount,
			"ErrorCount", clone.ErrorCount,
			"TotalTimeInternal", clone.TotalTimeInternal,
			"TotalTimeExternal", clone.TotalTimeExternal,
			"TotalBytesIn", clone.TotalBytesIn,
			"TotalBytesOut", clone.TotalBytesOut,
			"MessageQueueFillLevel", getWorkQueueFillLevel(),
			"WorkersIdle", getNumWorkersIdle())
		backup = clone2
		time.Sleep(interval)
	}
}

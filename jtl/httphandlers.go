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

package jtl

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	. "github.com/Comcast/eel/util"
)

// StatusHandler http handler for health and status checks. Writes JSON containing config.json, handler configs and basic stats to w.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := Gctx.SubContext()
	w.Header().Set("Content-Type", "application/json")
	state := make(map[string]interface{}, 0)
	state["Version"] = GetConfig(ctx).Version
	state["Config"] = GetConfig(ctx)
	callstats := make(map[string]interface{}, 0)
	if ctx.Value(EelDispatcher) != nil {
		callstats["WorkQueueFillLevel"] = len(GetWorkDispatcher(ctx, "").WorkQueue)
		callstats["WorkersIdle"] = len(GetWorkDispatcher(ctx, "").WorkerQueue)
	}
	if ctx.Value(EelTotalStats) != nil {
		callstats["TotalStats"] = ctx.Value(EelTotalStats)
	}
	if ctx.Value(Eel1MinStats) != nil {
		callstats[Eel1MinStats] = ctx.Value(Eel1MinStats)
	}
	if ctx.Value(Eel5MinStats) != nil {
		callstats[Eel5MinStats] = ctx.Value(Eel5MinStats)
	}
	if ctx.Value(Eel1hrStats) != nil {
		callstats[Eel1hrStats] = ctx.Value(Eel1hrStats)
	}
	if ctx.Value(Eel24hrStats) != nil {
		callstats[Eel24hrStats] = ctx.Value(Eel24hrStats)
	}
	callstats["StartTime"] = ctx.Value(EelStartTime)
	host, _ := os.Hostname()
	if host != "" {
		callstats["Hostname"] = host
	}
	elapsed1 := time.Since(start)
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					callstats["IpAddress"] = ipnet.IP.String()
				}
			}
		}
	}
	elapsed2 := time.Since(start)
	state["Stats"] = callstats
	state["CustomHandlers"] = GetHandlerFactory(ctx).CustomHandlerMap
	state["TopicHandlers"] = GetHandlerFactory(ctx).TopicHandlerMap
	buf, err := json.MarshalIndent(state, "", "\t")
	elapsed3 := time.Since(start)
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
	} else {
		fmt.Fprintf(w, string(buf))
	}
	elapsed4 := time.Since(start)
	ctx.Log().Info("action", "health", "d1", int64(elapsed1/1e6), "d2", int64(elapsed2/1e6), "d3", int64(elapsed3/1e6), "d4", int64(elapsed4/1e6))
}

// VetHandler http handler for vetting all handler configurations. Writes JSON with list of warnings (if any) to w.
func VetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, warnings := NewHandlerFactory(Gctx, HandlerPaths)
	if len(warnings) == 0 {
		fmt.Fprintf(w, `{"status":"ok"}`)
	} else {
		buf, err := json.MarshalIndent(warnings, "", "\t")
		if err != nil {
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		} else {
			fmt.Fprintf(w, string(buf))
		}
	}
}

// NilHandler http handler to to do almost nothing.
func NilHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// ReloadConfigHandler http handler to relaod all configs from disk. Response is similar to StatusHandler.
func ReloadConfigHandler(w http.ResponseWriter, r *http.Request) {
	if Gctx.Value(EelDispatcher) != nil {
		dp := Gctx.Value(EelDispatcher).(*WorkDispatcher)
		dp.Stop(Gctx)
	}
	ReloadConfig()
	InitHttpTransport(Gctx)
	dp := NewWorkDispatcher(GetConfig(Gctx).WorkerPoolSize, GetConfig(Gctx).MessageQueueDepth, "xh")
	dp.Start(Gctx)
	Gctx.AddValue(EelDispatcher, dp)
	dc := NewLocalInMemoryDupChecker(GetConfig(Gctx).DuplicateTimeout, 10000)
	Gctx.AddValue(EelDuplicateChecker, dc)
	StatusHandler(w, r)
}

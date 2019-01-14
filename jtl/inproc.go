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
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/Comcast/eel/util"
)

// EventHandler processes incoming events (arbitrary JSON payloads) and places them on the worker pool queue.
// If certain headers are set (X-Debug, X-Sync) a response will be returned immediately bypassing the worker pool queue.
func EventHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	debug := false
	sync := false
	if r.Header.Get("X-Debug") == "true" {
		debug = true
	} else if r.Header.Get("X-Sync") == "true" || r.URL.Path == Gctx.ConfigValue(EelSyncPath).(string) {
		sync = true
	}
	ctx.AddValue("start_ts", time.Now().UnixNano())
	ctx.AddValue(EelRequestHeader, r.Header)
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	stats.IncInCount()
	// adopt trace header if present
	traceHeaderKey := GetConfig(ctx).HttpTransactionHeader
	if r.Header.Get(traceHeaderKey) != "" {
		ctx.AddLogValue("tx.traceId", r.Header.Get(traceHeaderKey))
		ctx.AddValue("tx.traceId", r.Header.Get(traceHeaderKey))
		ctx.AddValue(traceHeaderKey, r.Header.Get(traceHeaderKey))
	} else {
		ctx.AddLogValue("tx.traceId", ctx.Id())
		ctx.AddValue("tx.traceId", ctx.Id())
		ctx.AddValue(traceHeaderKey, ctx.Id())
	}
	// adopt tenant id if present
	tenantHeaderKey := GetConfig(ctx).HttpTenantHeader
	if r.Header.Get(tenantHeaderKey) != "" {
		ctx.AddValue(EelTenantId, r.Header.Get(tenantHeaderKey))
		ctx.AddValue(tenantHeaderKey, r.Header.Get(tenantHeaderKey))
		ctx.AddLogValue(LogTenantId, r.Header.Get(tenantHeaderKey))
	}
	w.Header().Set("Content-Type", "application/json")
	if r.ContentLength > GetConfig(ctx).MaxMessageSize {
		ctx.Log().Error("status", "413", "action", "rejected", "error_type", "rejected", "cause", "message_too_large", "msg_length", r.ContentLength, "error", "message too large")
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write(GetResponse(ctx, StatusRequestTooLarge))
		stats.IncErrors()
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, GetConfig(ctx).MaxMessageSize)
	defer r.Body.Close()
	if r.Method != "POST" {
		ctx.Log().Error("status", "400", "action", "rejected", "error_type", "rejected", "cause", "http_post_required", "method", r.Method, "error", "post required")
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusHttpPostRequired))
		stats.IncErrors()
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_reading_message", "error", err.Error())
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		w.Write(GetResponse(ctx, StatusHttpPostRequired))
		stats.IncErrors()
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("status", "400", "action", "rejected", "error_type", "rejected", "cause", "blank_message", "error", "blank message")
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusEmptyBody))
		stats.IncErrors()
		return
	}
	dc := ctx.Value(EelDuplicateChecker).(DuplicateChecker)
	if dc.GetTtl() > 0 && dc.IsDuplicate(ctx, body) {
		ctx.Log().Info("status", "200", "action", "dropping_duplicate")
		ctx.Log().Metric("dropping_duplicate", M_Namespace, "xrs", M_Metric, "dropping_duplicate", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusOK)
		w.Write(GetResponse(ctx, StatusDuplicateEliminated))
		stats.IncErrors()
		return
	}
	// json validation maybe only in debug mode?
	evt, err := NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("status", "400", "action", "rejected", "error_type", "rejected", "cause", "invalid_json", "error", err.Error(), "trace.in.data", string(body))
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusInvalidJson))
		stats.IncErrors()
		return
	}
	stats.IncBytesIn(len(body))
	if GetConfig(ctx).LogParams != nil {
		for k, v := range GetConfig(ctx).LogParams {
			ev := evt.ParseExpression(ctx, v)
			ctx.AddLogValue(k, ev)
		}
	}
	if debug || sync {
		ctx.Log().Info("status", "200", "action", "accepted")
		ctx.Log().Metric("accepted", M_Namespace, "xrs", M_Metric, "accepted", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		var events interface{}
		events = handleEvent(ctx, stats, evt, string(body), debug, sync)
		if sync {
			switch events.(type) {
			case []interface{}:
				if len(events.([]interface{})) == 1 {
					events = events.([]interface{})[0]
				} else if len(events.([]interface{})) == 0 {
					events = ""
				}
			}
		}
		buf, err := json.MarshalIndent(events, "", "\t")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(GetResponse(ctx, map[string]interface{}{"error": err.Error()}))
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, string(buf))
		}
		AddLatencyLog(ctx, stats, "stat.eel.time")
	} else {
		tenantId := ""
		if ctx.Value(EelTenantId) != nil {
			tenantId = ctx.Value(EelTenantId).(string)
		}
		if ctx.Value(EelDispatcher+"_"+tenantId) != nil {
			dp := GetWorkDispatcher(ctx, tenantId)
			work := WorkRequest{Raw: string(body), Event: evt, Ctx: ctx}
			select {
			case dp.WorkQueue <- &work:
				ctx.Log().Info("status", "202", "action", "accepted")
				ctx.Log().Metric("accepted", M_Namespace, "xrs", M_Metric, "accepted", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
				w.WriteHeader(http.StatusAccepted)
				w.Write(GetResponse(ctx, StatusProcessed))
			case <-time.After(time.Millisecond * time.Duration(GetConfig(ctx).MessageQueueTimeout)):
				// consider spilling over to SQS here
				ctx.Log().Error("status", "429", "action", "rejected", "error_type", "work_queue", "cause", "queue_full")
				ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
				// 408
				//w.WriteHeader(http.StatusRequestTimeout)
				// 429
				w.WriteHeader(HttpStatusTooManyRequests)
				w.Write(GetResponse(ctx, StatusQueueFull))
			}
		}
	}
}

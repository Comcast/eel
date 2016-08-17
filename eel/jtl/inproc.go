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

	. "github.com/Comcast/eel/eel/util"
)

// EventHandler processes incoming events (arbitrary JSON payloads) and places them on the worker pool queue.
// If certain headers are set (X-Debug, X-Sync) a response will be returned immediately bypassing the worker pool queue.
func EventHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	debug := false
	sync := false
	if r.Header.Get("X-Debug") == "true" {
		debug = true
	} else if r.Header.Get("X-Sync") == "true" || r.URL.Path == GetConfig(ctx).EventProcPath {
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
	}
	w.Header().Set("Content-Type", "application/json")
	if r.ContentLength > GetConfig(ctx).MaxMessageSize {
		ctx.Log().Error("status", "413", "event", "rejected", "reason", "message_too_large", "msg.length", r.ContentLength, "msg.max.length", GetConfig(ctx).MaxMessageSize, "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
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
		ctx.Log().Error("status", "400", "event", "rejected", "reason", "http_post_required", "method", r.Method, "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusHttpPostRequired))
		stats.IncErrors()
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("status", "500", "event", "rejected", "reason", "error_reading_message", "error", err.Error(), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		w.Write(GetResponse(ctx, StatusHttpPostRequired))
		stats.IncErrors()
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("status", "400", "event", "rejected", "reason", "blank_message", "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusEmptyBody))
		stats.IncErrors()
		return
	}
	dc := ctx.Value(EelDuplicateChecker).(DuplicateChecker)
	if dc.GetTtl() > 0 && dc.IsDuplicate(ctx, body) {
		ctx.Log().Error("status", "200", "event", "dropping_duplicate", "trace.in.data", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("dropping_duplicate", M_Namespace, "xrs", M_Metric, "dropping_duplicate", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusOK)
		w.Write(GetResponse(ctx, StatusDuplicateEliminated))
		stats.IncErrors()
		return
	}
	// json validation maybe only in debug mode?
	msg, err := NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("status", "400", "event", "rejected", "reason", "invalid_json", "error", err.Error(), "trace.in.data", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusInvalidJson))
		stats.IncErrors()
		return
	}
	stats.IncBytesIn(len(body))
	if debug || sync {
		ctx.Log().Info("status", "200", "event", "accepted", "trace.in.data", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		ctx.Log().Metric("accepted", M_Namespace, "xrs", M_Metric, "accepted", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
		var events interface{}
		events = handleEvent(ctx, stats, msg, string(body), debug, sync)
		if sync {
			switch events.(type) {
			case []interface{}:
				if len(events.([]interface{})) == 1 {
					events = events.([]interface{})[0]
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
	} else {
		if ctx.Value(EelDispatcher) != nil {
			dp := GetWorkDispatcher(ctx)
			work := WorkRequest{Message: string(body), Ctx: ctx}
			select {
			case dp.WorkQueue <- &work:
				ctx.Log().Info("status", "202", "event", "accepted", "trace.in.data", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
				ctx.Log().Metric("accepted", M_Namespace, "xrs", M_Metric, "accepted", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
				w.WriteHeader(http.StatusAccepted)
				w.Write(GetResponse(ctx, StatusProcessed))
			case <-time.After(time.Millisecond * time.Duration(GetConfig(ctx).MessageQueueTimeout)):
				// consider spilling over to SQS here
				ctx.Log().Error("status", "429", "event", "rejected", "reason", "queue_full", "trace.in.data", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
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

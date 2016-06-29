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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	StatusQueueFull           = map[string]interface{}{"error": "queue full"}
	StatusInvalidJson         = map[string]interface{}{"error": "invalid json"}
	StatusEmptyBody           = map[string]interface{}{"error": "empty body"}
	StatusProcessed           = map[string]interface{}{"status": "processed"}
	StatusProcessedDummy      = map[string]interface{}{"status": "processed", "dummy": true}
	StatusDuplicateEliminated = map[string]interface{}{"status": "duplicate eliminated"}
	StatusRequestTooLarge     = map[string]interface{}{"error": "request too large"}
	StatusHttpPostRequired    = map[string]interface{}{"error": "http post required"}
	StatusUnknownTopic        = map[string]interface{}{"error": "unknown topic"}
	StatusAlreadySubscribed   = map[string]interface{}{"error": "already subscribed"}
	StatusNotEvenSubscribed   = map[string]interface{}{"error": "not even subscribed"}

	HttpStatusTooManyRequests = 429
)

func GetResponse(ctx Context, kv map[string]interface{}) []byte {
	if kv == nil {
		kv = make(map[string]interface{}, 0)
	}
	kv["tx.traceId"] = ctx.Value("tx.traceId")
	buf, err := json.Marshal(kv)
	if err != nil {
		return []byte(`{ "error" : "` + err.Error() + `", "tx.traceId" : ` + ctx.Value("tx.traceId").(string) + `}`)
	}
	return buf
}

func GetHttpClient(ctx Context) *http.Client {
	if ctx.Value(EelHttpClient) != nil {
		return ctx.Value(EelHttpClient).(*http.Client)
	}
	return nil
}

// InitHttpTransport initializes http transport with some parameters from config.json.
func InitHttpTransport(ctx Context) {
	tr := &http.Transport{
		MaxIdleConnsPerHost:   GetConfig(ctx).MaxIdleConnsPerHost,
		ResponseHeaderTimeout: GetConfig(ctx).ResponseHeaderTimeout * time.Millisecond,
	}
	if GetConfig(ctx).CloseIdleConnectionIntervalSec > 0 && !GetConfig(ctx).CloseIdleConnectionsStarted {
		go func() {
			GetConfig(ctx).CloseIdleConnectionsStarted = true
			for {
				time.Sleep(time.Duration(GetConfig(ctx).CloseIdleConnectionIntervalSec) * time.Second)
				ctx.Log().Info("event", "closing_idle_connections")
				tr.CloseIdleConnections()
			}
		}()
	}
	ctx.AddValue(EelHttpTransport, tr)
	client := &http.Client{Transport: tr}
	client.Timeout = GetConfig(ctx).HttpTimeout * time.Millisecond
	ctx.AddValue(EelHttpClient, client)
}

// HitEndpoint helper method for posting payloads to endpoints. Supports other verbs, http headers and basic auth.
func HitEndpoint(ctx Context, url string, payload string, verb string, headers map[string]string, auth map[string]string) (string, int, error) {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	stats.IncBytesOut(len(payload))
	req, err := http.NewRequest(verb, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		ctx.Log().Error("event", "error_new_request", "url", url, "verb", verb, "error", err.Error())
		stats.IncErrors()
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "EEL")
	// add trace header to outbound call
	traceHeader := GetConfig(ctx).HttpTransactionHeader
	if "" != traceHeader {
		traceId := ctx.Value("tx.traceId")
		if nil == traceId {
			traceId = ctx.Id()
		}
		req.Header.Set(traceHeader, traceId.(string))
	}
	// add supplied headers
	if headers != nil {
		for hk, hv := range headers {
			req.Header.Set(hk, hv)
		}
	}
	if auth != nil {
		if auth["type"] == "basic" {
			req.SetBasicAuth(auth["username"], auth["password"])
		}
	}
	duration := int64(0)
	startTime := int64(0)
	if ctx.Value("start_ts") != nil {
		startTime = (ctx.Value("start_ts")).(int64)
	}
	if startTime > 0 {
		duration = time.Now().UnixNano() - startTime
	}
	ctx.AddLogValue("stat.eel.time", duration)
	stats.IncTimeInternal(duration)
	// send request
	resp, err := GetHttpClient(ctx).Do(req)
	if err != nil {
		ctx.Log().Error("event", "error_reaching_service", "trace.out.url", url, "trace.out.verb", verb, "trace.out.headers", headers, "error", err.Error())
		stats.IncErrors()
		if ctx.LogValue("destination") != nil {
			ctx.Log().Metric("drops", M_Namespace, "xrs", M_Metric, "drops", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName+"&destination="+ctx.LogValue("destination").(string), M_Val, 1.0)
		}
		return "", 0, err
	}
	if startTime > 0 {
		duration = (time.Now().UnixNano() - startTime) - duration
	}
	ctx.AddLogValue("stat.external.time", duration)
	stats.IncTimeExternal(duration)
	// read response
	var body []byte
	if resp != nil && resp.Body != nil {
		var readErr error
		body, readErr = ioutil.ReadAll(resp.Body)
		if readErr != nil {
			ctx.Log().Error("event", "error_reaching_service", "reason", "error_reading_response", "trace.out.url", url, "trace.out.verb", verb, "trace.out.headers", headers, "status", strconv.Itoa(resp.StatusCode), "error", readErr.Error())
			stats.IncErrors()
			return "", resp.StatusCode, readErr
		}
		closeErr := resp.Body.Close()
		if closeErr != nil {
			ctx.Log().Error("event", "error_reaching_service", "reason", "error_closing_response", "trace.out.url", url, "trace.out.verb", verb, "trace.out.headers", headers, "status", strconv.Itoa(resp.StatusCode), "error", closeErr.Error())
			stats.IncErrors()
		}
		if body == nil {
			return "", resp.StatusCode, nil
		}
	}
	// only log short responses from outgoing http requests
	if len(body) <= 512 {
		ctx.Log().Info("event", "reached_service", "trace.out.url", url, "trace.out.verb", verb, "trace.out.headers", headers, "status", strconv.Itoa(resp.StatusCode), "length", len(body), "response", string(body))
	} else {
		ctx.Log().Info("event", "reached_service", "trace.out.url", url, "trace.out.verb", verb, "trace.out.headers", headers, "status", strconv.Itoa(resp.StatusCode), "length", len(body))
	}
	if ctx.LogValue("destination") != nil {
		ctx.Log().Metric("hits", M_Namespace, "xrs", M_Metric, "hits", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName+"&destination="+ctx.LogValue("destination").(string), M_Val, 1.0)
	}
	return string(body), resp.StatusCode, nil
}

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
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/Comcast/eel/util"
)

type (
	WebhookPlugin struct {
		Settings     *PluginSettings
		ShuttingDown bool
	}

	responseWriter struct {
		http.ResponseWriter

		statusCode int
	}
)

var apiBasePath = ""

func NewWebhookPlugin(settings *PluginSettings) InboundPlugin {
	p := new(WebhookPlugin)
	p.Settings = settings
	return p
}

func SetApiBasePath(basePath string) {
	apiBasePath = basePath
}

func GetApiBasePath() string {
	return apiBasePath
}

func (p *WebhookPlugin) GetSettings() *PluginSettings {
	return p.Settings
}

func (p *WebhookPlugin) StartPlugin(ctx Context) {
	p.StartWebhookConsumer(ctx)
}

func (p *WebhookPlugin) StartWebhookConsumer(ctx Context) {
	ctx.Log().Info("action", "starting_plugin", "op", "webhook")
	go p.startWebhookServices(ctx)
}

func (p *WebhookPlugin) startWebhookServices(ctx Context) {
	defer ctx.HandlePanic()
	p.ShuttingDown = false
	p.Settings.Active = true
	eventProxyPort := int(p.GetSettings().Parameters["EventPort"].(float64))
	if eventProxyPort == 0 {
		eventProxyPort = 8080
	}
	eventProxyPath := apiBasePath + p.GetSettings().Parameters["EventProxyPath"].(string)
	eventProcPath := apiBasePath + p.GetSettings().Parameters["EventProcPath"].(string)
	http.HandleFunc(eventProxyPath, wrap(ctx, EventHandler))
	http.HandleFunc(eventProcPath, wrap(ctx, EventHandler))
	http.HandleFunc(apiBasePath+"/elementsevent", wrap(ctx, EventHandler)) // hard coded during transition period
	http.HandleFunc(apiBasePath+"/notify", wrap(ctx, EventHandler))        // hard coded during transition period

	ctx.Log().Info("action", "listening_for_events", "port", eventProxyPort, "proxy_path", eventProxyPath, "proc_path", eventProcPath, "op", "webhook")
	err := http.ListenAndServe(":"+strconv.Itoa(eventProxyPort), nil)
	if err != nil {
		ctx.Log().Error("error_type", "eel_service", "error", err.Error())
	}
	p.Settings.Active = false
	ctx.Log().Info("action", "stopping_plugin", "op", "webhook")
	if p.Settings.ExitOnErr {
		os.Exit(1)
	}
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func wrap(ctx Context, f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx = Start(ctx, HTTPHandle, nil)
		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
		}
		defer func() {
			attrs := map[string]string{
				HTTPRouteKey:      r.URL.Redacted(),
				HTTPMethodKey:     r.Method,
				HTTPStatusCodeKey: strconv.Itoa(rw.statusCode),
			}

			Record(ctx, HTTPHandleDuration, attrs, int(time.Since(start).Milliseconds()))
			End(ctx, attrs, nil)
		}()

		f(rw, r)
	}
}

func (p *WebhookPlugin) StopPlugin(ctx Context) {
	ctx.Log().Info("action", "shutdown_plugin", "op", "stdin", "details", "cannot_shutdonw")
}

func (p *WebhookPlugin) IsActive() bool {
	return p.Settings.Active
}

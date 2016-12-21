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
	"strconv"

	. "github.com/Comcast/eel/eel/util"
)

type WebhookPlugin struct {
	Settings     *PluginSettings
	ShuttingDown bool
}

func NewWebhookPlugin(settings *PluginSettings) InboundPlugin {
	p := new(WebhookPlugin)
	p.Settings = settings
	return p
}

func (p *WebhookPlugin) GetSettings() *PluginSettings {
	return p.Settings
}

func (p *WebhookPlugin) StartPlugin(ctx Context) {
	p.StartWebhookConsumer(ctx)
}

func (p *WebhookPlugin) StartWebhookConsumer(ctx Context) {
	ctx.Log().Info("action", "starting_plugin", "op", "webhook")
	p.startWebhookServices(ctx)
	ctx.Log().Info("action", "stopping_plugin", "op", "webhook")
}

func (p *WebhookPlugin) startWebhookServices(ctx Context) {
	p.ShuttingDown = false
	p.Settings.Active = true
	eventProxyPort := int(p.GetSettings().Parameters["EventPort"].(float64))
	if eventProxyPort == 0 {
		eventProxyPort = 8080
	}
	eventProxyPath := p.GetSettings().Parameters["EventProxyPath"].(string)
	eventProcPath := p.GetSettings().Parameters["EventProcPath"].(string)
	http.HandleFunc(eventProxyPath, EventHandler)
	http.HandleFunc(eventProcPath, EventHandler)
	ctx.Log().Info("action", "listening_for_events", "port", eventProxyPort, "proxy_path", eventProxyPath, "proc_path", eventProcPath, "op", "webhook")
	err := http.ListenAndServe(":"+strconv.Itoa(eventProxyPort), nil)
	if err != nil {
		ctx.Log().Error("error_type", "eel_service", "error", err.Error())
	}
}

func (p *WebhookPlugin) StopPlugin(ctx Context) {
	ctx.Log().Info("action", "shutdown_plugin", "op", "stdin", "details", "cannot_shutdonw")
}

func (p *WebhookPlugin) CanShutdown() bool {
	return false
}

func (p *WebhookPlugin) IsActive() bool {
	return p.Settings.Active
}

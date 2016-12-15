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
	Settings *PluginSettings
}

func NewWebhookPlugin(settings *PluginSettings) InboundPlugin {
	p := new(WebhookPlugin)
	p.Settings = settings
	return p
}

func (p *WebhookPlugin) GetSettings() *PluginSettings {
	return p.Settings
}

func (p *WebhookPlugin) StartPlugin(ctx Context, c chan int) {
	p.StartWebhookConsumer(ctx, c)
}

func (p *WebhookPlugin) StartWebhookConsumer(ctx Context, c chan int) {
	ctx.Log().Info("action", "starting_plugin", "op", "webhook")
	startWebhookServices(ctx)
	ctx.Log().Info("action", "stopping_plugin", "op", "webhook")
	c <- 0
}

func startWebhookServices(ctx Context) {
	eventProxyPort := GetConfig(ctx).EventPort
	if eventProxyPort == 0 {
		eventProxyPort = 8080
	}
	eventProxyPath := GetConfig(ctx).EventProxyPath
	eventProcPath := GetConfig(ctx).EventProcPath
	http.HandleFunc(eventProxyPath, EventHandler)
	http.HandleFunc(eventProcPath, EventHandler)
	ctx.Log().Info("action", "listening_for_events", "port", eventProxyPort, "proxy_path", eventProxyPath, "proc_path", eventProcPath)
	err := http.ListenAndServe(":"+strconv.Itoa(eventProxyPort), nil)
	if err != nil {
		ctx.Log().Error("error_type", "eel_service", "error", err.Error())
	}
}

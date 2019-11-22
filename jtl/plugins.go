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

// inbound plugins, currently only supported plugins are webhook and stdin,
// other plugins could be provided for websocket, kafka, sqs etc.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/Comcast/eel/util"
)

type InboundPlugin interface {
	StartPlugin(Context)
	GetSettings() *PluginSettings
	StopPlugin(Context)
	IsActive() bool
}

type PluginTemplateParams struct {
	BasePath string
	PluginSettings
}

type PluginSettings struct {
	Type       string
	Name       string
	Active     bool
	RestartOk  bool
	AutoStart  bool
	ExitOnErr  bool
	Parameters map[string]interface{}
	Stats      PluginStats
}

type PluginStats struct {
	MessageCount      uint64
	MessagesPerSecond uint64
	MessagesPerMinute uint64
	Misc              map[string]interface{}
	sync.RWMutex
}

type NewInboundPlugin func(*PluginSettings) InboundPlugin

type PluginConfigList []*PluginSettings

// plugins by name
var inboundPluginMap = make(map[string]InboundPlugin, 0)

// plugins by type
var inboundPluginTypeMap = make(map[string]NewInboundPlugin, 0)

var pluginConfigList PluginConfigList

// RegisterInboundPlugin registers an (external) plugin implementation by plugin type
func RegisterInboundPluginType(newPlugin NewInboundPlugin, pluginType string) {
	inboundPluginTypeMap[pluginType] = newPlugin
}

func GetInboundPluginByType(pluginType string) InboundPlugin {
	// currently only one active plugin per type allowed!!!
	for _, v := range inboundPluginMap {
		if v.GetSettings().Type == pluginType {
			return v
		}
	}
	return nil
}

func GetInboundPluginByName(name string) InboundPlugin {
	return inboundPluginMap[name]
}

func PluginConfigHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	w.Header().Set("Content-Type", "application/json")
	state := make(map[string]interface{}, 0)
	state["Version"] = GetConfig(ctx).Version
	state["PluginConfigs"] = pluginConfigList
	buf, err := json.MarshalIndent(state, "", "\t")
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
	} else {
		fmt.Fprintf(w, string(buf))
	}
}

func ManagePluginsUIHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	pluginsTemplate, err := template.ParseFiles("web/plugins.html")
	if err != nil {
		ctx.Log().Error("error_type", "manage_plugins", "cause", "template_parse_error", "error", err.Error())
	}
	operation := r.FormValue("operation")
	name := r.FormValue("name")
	if operation == "Start" && name != "" {
		p := GetInboundPluginByName(name)
		if p != nil && !p.IsActive() {
			p.StartPlugin(ctx)
			time.Sleep(1 * time.Second)
		}
	} else if operation == "Stop" && name != "" {
		p := GetInboundPluginByName(name)
		if p != nil && p.IsActive() {
			p.StopPlugin(ctx)
		}
	}
	psl := make([]*PluginTemplateParams, 0)
	for _, p := range inboundPluginMap {
		params := &PluginTemplateParams{
			BasePath:       GetApiBasePath(),
			PluginSettings: *p.GetSettings(),
		}
		psl = append(psl, params)
	}
	err = pluginsTemplate.Execute(w, psl)
	if err != nil {
		ctx.Log().Error("error_type", "manage_plugins", "cause", "template_exec_error", "error", err.Error())
	}
}

func ManagePluginsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	path := strings.Split(r.URL.Path, "/")
	pName := path[len(path)-2]
	p := GetInboundPluginByName(pName)
	if p == nil {
		ctx.Log().Error("error_type", "manage_plugins", "cause", "unknown_plugin", "error", "unknown plugin "+pName)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, "unknown plugin "+pName)
		return
	}
	op := path[len(path)-1]
	switch op {
	case "start":
		if p.GetSettings().RestartOk == false {
			ctx.Log().Error("error_type", "manage_plugins", "cause", "plugin_cannot_restart", "error", "plugin cannot restart "+pName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, "plugin cannot restart "+pName)
			return
		}
		if p.GetSettings().Active == true {
			ctx.Log().Error("error_type", "manage_plugins", "cause", "plugin_already_running", "error", "plugin already running "+pName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, "plugin already running "+pName)
			return
		}
		p.StartPlugin(ctx)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"plugin":"%s", "action":"started"}`, pName)
	case "stop":
		if p.GetSettings().RestartOk == false {
			ctx.Log().Error("error_type", "manage_plugins", "cause", "plugin_cannot_restart", "error", "plugin cannot restart "+pName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, "plugin cannot restart "+pName)
			return
		}
		if p.GetSettings().Active == false {
			ctx.Log().Error("error_type", "manage_plugins", "cause", "plugin_stopped", "error", "plugin already stopped "+pName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, "plugin already stopped "+pName)
			return
		}
		p.StopPlugin(ctx)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"plugin":"%s", "action":"stopped"}`, pName)
	case "state":
		state := make(map[string]interface{}, 0)
		state["Version"] = GetConfig(ctx).Version
		state["PluginConfigs"] = p.GetSettings()
		buf, err := json.MarshalIndent(state, "", "\t")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, string(buf))
		}
	case "config":
		state := make(map[string]interface{}, 0)
		state["Version"] = GetConfig(ctx).Version
		state["PluginConfigs"] = p.GetSettings()
		buf, err := json.MarshalIndent(state, "", "\t")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, string(buf))
		}
	default:
		ctx.Log().Error("error_type", "manage_plugins", "cause", "unknown_op", "error", "unknown op "+op)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, "unknown plugin "+pName)
		return
	}
}

func GetPluginConfigList(ctx Context) PluginConfigList {
	configFile, err := os.Open(filepath.Join(filepath.Dir(ConfigPath), "plugins.json"))
	if err != nil {
		// csv-context-go may not be ready yet for logging
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_plugin_config", "cause", "open_config", "error", err.Error())
		os.Exit(1)
	}
	defer configFile.Close()
	configData, err := ioutil.ReadAll(configFile)
	if err != nil {
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_plugin_config", "cause", "read_config", "error", err.Error())
		os.Exit(1)
	}
	var config PluginConfigList
	err = json.Unmarshal(configData, &config)
	if err != nil {
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_plugin_config", "cause", "parse_config", "error", err.Error())
		os.Exit(1)
	}
	return config
}

func LoadInboundPlugins(ctx Context, enablePlugins bool) {
	// load plugin configs
	pluginConfigList = GetPluginConfigList(ctx)
	for _, e := range pluginConfigList {
		// dependency injection
		np := inboundPluginTypeMap[e.Type]
		if np == nil {
			ctx.Log().Error("error_type", "bad_plugin_config", "cause", "unknown_plugin_type", "plugin_type", e.Type)
		} else {
			inboundPluginMap[e.Name] = np(e)
		}
	}
	// launch plugins
	for k, v := range inboundPluginMap {
		if !enablePlugins && v.GetSettings().Type != "WEBHOOK" {
			v.GetSettings().AutoStart = false
		}

		if v.GetSettings().AutoStart {
			ctx.Log().Info("action", "launching_inbound_plugin", "plugin_name", k, "pugin_type", v.GetSettings().Type)
			v.StartPlugin(ctx)
		} else {
			ctx.Log().Info("action", "skipping_inactive_plugin", "plugin_name", k, "pugin_type", v.GetSettings().Type)
		}
	}
	// need sync path in inproc.go
	if GetInboundPluginByType("WEBHOOK") != nil {
		syncPath := apiBasePath + GetInboundPluginByType("WEBHOOK").GetSettings().Parameters["EventProcPath"].(string)
		Gctx.AddConfigValue(EelSyncPath, syncPath)
	} else {
		Gctx.AddConfigValue(EelSyncPath, "")
	}
}

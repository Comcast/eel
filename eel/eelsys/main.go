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

// A simple proxy service to forward JSON events and transform or filter them along the way.
package main

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	. "github.com/Comcast/eel/eel/handlers"
	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

// build hint: go build -ldflags "-X main.Version 2.0"

var (
	Version = "1.0"
)

var (
	env         = flag.String("env", "default", "environment name such as qa, prod for logging")
	basePath    = flag.String("path", "", "base path for config.json and handlers (optional)")
	configPath  = flag.String("config", "", "path to config.json (optional)")
	handlerPath = flag.String("handlers", "", "path to handlers (optional)")
	logLevel    = flag.String("loglevel", L_InfoLevel, "log level (optional)")
)

// useCores if GOMAXPROCS not set use all cores you got.
func useCores(ctx Context) {
	cores := os.Getenv("GOMAXPROCS")
	if cores == "" {
		n := runtime.NumCPU()
		ctx.Log().Info("event", "use_cores", "cores", n)
		runtime.GOMAXPROCS(n)
		cores = strconv.Itoa(n)
	} else {
		ctx.Log().Info("event", "use_cores_from_env", "cores", cores)
	}
}

// startProxyServices starts service and registers all http handlers.
func startProxyServices(ctx Context) {
	eventProxyPort := GetConfig(ctx).EventPort
	if eventProxyPort == 0 {
		eventProxyPort = 8080
	}
	eventProxyPath := GetConfig(ctx).EventProxyPath
	eventProcPath := GetConfig(ctx).EventProcPath
	ctx.Log().Info("event", "registering_event_proxy", "path", "http://localhost:"+strconv.Itoa(eventProxyPort)+eventProxyPath)
	http.HandleFunc(eventProxyPath, EventHandler)
	http.HandleFunc(eventProcPath, EventHandler)
	http.HandleFunc("/health/shallow", NilHandler)
	http.HandleFunc("/health/deep", StatusHandler)
	http.HandleFunc("/health", StatusHandler)
	http.HandleFunc("/status", StatusHandler)
	http.HandleFunc("/reload", ReloadConfigHandler)
	http.HandleFunc("/vet", VetHandler)
	http.HandleFunc("/test", TopicTestHandler)
	http.HandleFunc("/test/process/", ProcessExpressionHandler)
	http.HandleFunc("/test/ast", ParserDebugHandler)
	http.HandleFunc("/test/astjson/", GetASTJsonHandler)
	http.HandleFunc("/test/asttree/", ParserDebugVizHandler)
	http.HandleFunc("/event/dummy", DummyEventHandler)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(filepath.Join(BasePath, "mascot")))))
	ctx.Log().Info("event", "listening_for_events", "port", eventProxyPort, "proxy_path", eventProxyPath, "proc_path", eventProcPath)
	err := http.ListenAndServe(":"+strconv.Itoa(eventProxyPort), nil)
	if err != nil {
		ctx.Log().Error("event", "http_error", "error", err.Error())
	}
}

// initLogging sets up context and stats loop.
func initLogging() {
	if *basePath != "" {
		BasePath = *basePath
	}
	if *configPath != "" {
		ConfigPath = filepath.Join(BasePath, *configPath)
	} else {
		ConfigPath = filepath.Join(BasePath, EelConfigFile)
	}
	Gctx = NewDefaultContext(*logLevel)
	config := GetConfigFromFile(Gctx)
	if *handlerPath != "" {
		HandlerPath = *handlerPath
	} else if config.HandlerConfigPath != "" {
		HandlerPath = config.HandlerConfigPath
	}
	AppId = config.AppName
	Gctx.AddLogValue("app.id", AppId)
	InstanceName, _ = os.Hostname()
	Gctx.AddLogValue("instance.id", InstanceName)
	if *env != "" {
		EnvName = *env
		Gctx.AddLogValue("env.name", EnvName)
	}
	Gctx.AddValue(EelStartTime, time.Now().Local().Format("2006-01-02 15:04:05 +0800"))
	stats := new(ServiceStats)
	Gctx.AddValue(EelTotalStats, stats)
	Gctx.AddValue(Eel1MinStats, new(ServiceStats))
	Gctx.AddValue(Eel5MinStats, new(ServiceStats))
	Gctx.AddValue(Eel1hrStats, new(ServiceStats))
	Gctx.AddValue(Eel24hrStats, new(ServiceStats))
	if config.LogStats {
		go Gctx.Log().RuntimeLogLoop(time.Duration(60)*time.Second, -1)
		go stats.StatsLoop(Gctx, 300*time.Second, -1, Eel5MinStats)
		go stats.StatsLoop(Gctx, 60*time.Second, -1, Eel1MinStats)
		go stats.StatsLoop(Gctx, 60*time.Minute, -1, Eel1hrStats)
		go stats.StatsLoop(Gctx, 24*time.Hour, -1, Eel24hrStats)
	}
}

func main() {
	flag.Parse()
	initLogging()
	ReloadConfig()
	GetConfig(Gctx).Version = Version
	InitHttpTransport(Gctx)
	ctx := Gctx.SubContext()
	ctx.Log().Info("event", "starting", "version", Version)
	useCores(ctx)
	dc := NewLocalInMemoryDupChecker(GetConfig(ctx).DuplicateTimeout, 10000)
	Gctx.AddValue(EelDuplicateChecker, dc)
	dp := NewWorkDispatcher(GetConfig(ctx).WorkerPoolSize, GetConfig(ctx).MessageQueueDepth)
	dp.Start(ctx)
	Gctx.AddValue(EelDispatcher, dp)
	startProxyServices(ctx)
}

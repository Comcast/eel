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

	. "github.com/Comcast/eel/jtl"
	. "github.com/Comcast/eel/util"

	_ "net/http/pprof"
)

// build hint: go build -ldflags "-X main.Version 2.0"

var (
	Version = "1.0"
)

var (
	// proxy params
	env         = flag.String("env", "default", "environment name such as qa, prod for logging")
	basePath    = flag.String("path", "", "base path for config.json and handlers (optional)")
	configPath  = flag.String("config", "", "path to config.json (optional)")
	handlerPath = flag.String("handlers", "", "path to handlers (optional)")
	logLevel    = flag.String("loglevel", L_InfoLevel, "log level (optional)")
	// cmd params
	in    = flag.String("in", "", "incoming event string or @file")
	tf    = flag.String("tf", "", "transformation string or @file")
	istbe = flag.Bool("istbe", true, "is template by example flag")
)

// useCores if GOMAXPROCS not set use all cores you got.
func useCores(ctx Context) {
	cores := os.Getenv("GOMAXPROCS")
	if cores == "" {
		n := runtime.NumCPU()
		ctx.Log().Info("action", "use_cores", "cores", n)
		runtime.GOMAXPROCS(n)
		cores = strconv.Itoa(n)
	} else {
		ctx.Log().Info("action", "use_cores_from_env", "cores", cores)
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

	Gctx.AddConfigValue(EelTraceLogger, NewTraceLogger(Gctx, config))

	getWorkQueueFillLevel := func() int {
		wd := GetWorkDispatcher(Gctx)
		if wd != nil {
			return len(wd.WorkQueue)
		}
		return -1
	}

	getNumWorkersIdle := func() int {
		wd := GetWorkDispatcher(Gctx)
		if wd != nil {
			return len(wd.WorkerQueue)
		}
		return -1
	}

	if config.LogStats {
		go Gctx.Log().RuntimeLogLoop(time.Duration(60)*time.Second, -1)
		go stats.StatsLoop(Gctx, 300*time.Second, -1, Eel5MinStats, getWorkQueueFillLevel, getNumWorkersIdle)
		go stats.StatsLoop(Gctx, 60*time.Second, -1, Eel1MinStats, getWorkQueueFillLevel, getNumWorkersIdle)
		go stats.StatsLoop(Gctx, 60*time.Minute, -1, Eel1hrStats, getWorkQueueFillLevel, getNumWorkersIdle)
		go stats.StatsLoop(Gctx, 24*time.Hour, -1, Eel24hrStats, getWorkQueueFillLevel, getNumWorkersIdle)
	}
}

func registerAdminServices() {
	c := Gctx.SubContext()
	// old handlers
	http.HandleFunc("/health/shallow", c.WrapPanicHttpHandler(NilHandler))
	http.HandleFunc("/health/deep", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/health", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/status", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/pluginconfigs", c.WrapPanicHttpHandler(PluginConfigHandler))
	http.HandleFunc("/plugins", c.WrapPanicHttpHandler(ManagePluginsUIHandler))
	http.HandleFunc("/plugins/", c.WrapPanicHttpHandler(ManagePluginsHandler))
	http.HandleFunc("/reload", c.WrapPanicHttpHandler(ReloadConfigHandler))
	http.HandleFunc("/toggletracelogger", c.WrapPanicHttpHandler(TraceLogConfigHandler))
	http.HandleFunc("/vet", c.WrapPanicHttpHandler(VetHandler))
	http.HandleFunc("/test", c.WrapPanicHttpHandler(TopicTestHandler))
	http.HandleFunc("/test/handlers", c.WrapPanicHttpHandler(HandlersTestHandler))
	http.HandleFunc("/test/process/",c.WrapPanicHttpHandler( ProcessExpressionHandler))
	http.HandleFunc("/test/ast", c.WrapPanicHttpHandler(ParserDebugHandler))
	http.HandleFunc("/test/astjson/", c.WrapPanicHttpHandler(GetASTJsonHandler))
	http.HandleFunc("/test/asttree/", c.WrapPanicHttpHandler(ParserDebugVizHandler))
	http.HandleFunc("/event/dummy", c.WrapPanicHttpHandler(DummyEventHandler))
	// v1 handlers
	http.HandleFunc("/v1/health/shallow",c.WrapPanicHttpHandler( NilHandler))
	http.HandleFunc("/v1/health/deep", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/v1/health", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/v1/status", c.WrapPanicHttpHandler(StatusHandler))
	http.HandleFunc("/v1/pluginconfigs", c.WrapPanicHttpHandler(PluginConfigHandler))
	http.HandleFunc("/v1/plugins", c.WrapPanicHttpHandler(ManagePluginsUIHandler))
	http.HandleFunc("/v1/plugins/", c.WrapPanicHttpHandler(ManagePluginsHandler))
	http.HandleFunc("/v1/reload", c.WrapPanicHttpHandler(ReloadConfigHandler))
	http.HandleFunc("/v1/toggletracelogger", c.WrapPanicHttpHandler(TraceLogConfigHandler))
	http.HandleFunc("/v1/vet", c.WrapPanicHttpHandler(VetHandler))
	http.HandleFunc("/v1/test", c.WrapPanicHttpHandler(TopicTestHandler))
	http.HandleFunc("/v1/test/handlers", c.WrapPanicHttpHandler(HandlersTestHandler))
	http.HandleFunc("/v1/test/process/", c.WrapPanicHttpHandler(ProcessExpressionHandler))
	http.HandleFunc("/v1/test/ast", c.WrapPanicHttpHandler(ParserDebugHandler))
	http.HandleFunc("/v1/test/astjson/", c.WrapPanicHttpHandler(GetASTJsonHandler))
	http.HandleFunc("/v1/test/asttree/", c.WrapPanicHttpHandler(ParserDebugVizHandler))
	http.HandleFunc("/v1/event/dummy", c.WrapPanicHttpHandler(DummyEventHandler))
	http.HandleFunc("/v1/event/panic",c.WrapPanicHttpHandler(PanicEventHandler))
	//
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(filepath.Join(BasePath, "mascot")))))
}

func main() {
	flag.Parse()
	if *tf != "" {
		eelCmd(*in, *tf, *istbe)
	} else {
		initLogging()
		ReloadConfig()
		GetConfig(Gctx).Version = Version
		InitHttpTransport(Gctx)
		ctx := Gctx.SubContext()
		ctx.Log().Info("action", "starting", "version", Version)
		useCores(ctx)
		dc := NewLocalInMemoryDupChecker(GetConfig(ctx).DuplicateTimeout, 10000)
		Gctx.AddValue(EelDuplicateChecker, dc)
		dp := NewWorkDispatcher(GetConfig(ctx).WorkerPoolSize, GetConfig(ctx).MessageQueueDepth)
		dp.Start(ctx)
		Gctx.AddValue(EelDispatcher, dp)
		registerAdminServices()
		// register inbound plugins
		RegisterInboundPluginType(NewStdinPlugin, "STDIN")
		RegisterInboundPluginType(NewWebhookPlugin, "WEBHOOK")
		LoadInboundPlugins(Gctx)
		// hang on channel forever
		<-make(chan int)
	}
}


//used to test  HttpHandlerFunc panic can be removed later
func PanicEventHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	ctx.Log().Info("op","PanicEventNoHandler")
	panic("I am panic now and Nobody care!")
}


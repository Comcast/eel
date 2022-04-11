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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

// EelSettings struct representing config.json master config file.
type EelSettings struct {
	Name                           string
	AppName                        string
	ElementsPublishEndpoint        string
	ElementsWebhookEndpoint        string
	ElementsAuth                   string
	EelWebhook                     string
	Brokers                        string
	FunctionalMonitoringPort       int
	Endpoint                       interface{}
	MaxAttempts                    int
	InitialDelay                   time.Duration
	InitialBackoff                 time.Duration
	Pad                            time.Duration
	BackoffMethod                  string
	Organization                   string
	EventTopics                    []string
	ActionTopics                   []string
	MaxMessageSize                 int64
	HttpTransactionHeader          string
	HttpDebugHeader                string
	HttpTenantHeader               string
	HttpPartnerHeader              string
	HttpTimeout                    time.Duration
	ResponseHeaderTimeout          time.Duration
	MaxIdleConnsPerHost            int
	CustomProperties               map[string]interface{}
	Misc                           map[string]interface{}
	LogParams                      map[string]string
	DebugLogParams                 *EelDebugLogParams
	TraceLogParams                 *EelTraceLogParams
	WorkerPoolSize                 map[string]int
	MessageQueueTimeout            int
	MessageQueueDepth              int
	TopicPath                      string
	LogStats                       bool
	SendCloudWatchMetrics          bool
	DuplicateTimeout               int
	CloseIdleConnectionIntervalSec int
	CloseIdleConnectionsStarted    bool
	RetryQueues                    []string
	RetryServiceAvailable          bool
	UseRetryQueue                  bool
	Version                        string
	HandlerConfigPath              string
	AllowPartner                   bool
	DefaultPartner                 string
}

// EelDebugLogParams struct is an optional debug white list and log param config in eel settings
type EelDebugLogParams struct {
	Lock           sync.RWMutex
	RefreshSec     int
	WhiteListTable string
	IdWhiteList    map[string]string
	IdPath         string
	LogParams      map[string]string
}

// EelTraceLogParams struct is an optional trace log config in eel settings to be activated for brief periods of time to capture a full trace of incoming or outgoing events
type EelTraceLogParams struct {
	Active      bool
	FileName    string
	LogIncoming bool
	LogOutgoing bool
	LogParams   map[string]string
}

const (
	EelFile                 = "mascot/eel.txt"
	EelConfigFile           = "config-eel/config.json"
	DefaultConfigFolder     = "config-handlers"
	Eel1MinStats            = "Eel.Stats.1Min"
	Eel5MinStats            = "Eel.Stats.5Min"
	Eel1hrStats             = "Eel.Stats.1hr"
	Eel24hrStats            = "Eel.Stats.24hr"
	EelTotalStats           = "Eel.TotalStats"
	EelPathWhiteList        = "Eel.PathWhiteList"
	EelDispatcher           = "Eel.Dispatcher"
	EelDuplicateChecker     = "Eel.DuplicateChecker"
	EelStartTime            = "StartTime"
	EelConfig               = "Eel.Settings"
	EelHandlerFactory       = "HandlerFactory"
	EelObserver             = "Observer"
	EelHttpClient           = "Eel.HttpClient"
	EelHttpTransport        = "Eel.HttpTransport"
	EelRequestHeader        = "Eel.Header"
	EelRequestQuery         = "Eel.Query"
	EelNamedTransformations = "Eel.NamedTransformations"
	EelHandlerConfig        = "Eel.HandlerConfig"
	EelTenantId             = "Eel.TenantId"
	EelPartnerId            = "Eel.PartnerId"
	EelCustomProperties     = "Eel.CustomProperties"
	EelRetryService         = "Eel.RetryService"
	EelErrors               = "Eel.Errors"
	EelSyncPath             = "Eel.SyncPath"
	EelTraceLogger          = "Eel.TraceLogger"
	EelCache                = "Eel.Cache"
	EelTenantIds            = "Eel.TenantIds"
	LogTenantId             = "gears.app.id"
	LogPartnerId            = "gears.partner.id"
)

const (
	M_Namespace = "Namespace"
	M_Metric    = "Metric"
	M_Unit      = "Unit"
	M_Dims      = "Dims"
	M_Val       = "Val"
)

var (
	LogLevel     = L_InfoLevel
	Gctx         Context
	Mutex        = &sync.RWMutex{}
	Once         sync.Once
	BasePath     = ""
	ConfigPath   = ""
	HandlerPath  = ""
	HandlerPaths = []string{""}
	InstanceName = "localhost"
	EnvName      = "default"
	AppId        = "eel"
)

func InitGctx(logLevel string) {
	Mutex.Lock()
	Gctx = NewDefaultContext(logLevel)
	Mutex.Unlock()
}

// GetConfig is a helper function to obtain the global config from the context.
func GetConfig(ctx Context) *EelSettings {
	if ctx.ConfigValue(EelConfig) != nil {
		return ctx.ConfigValue(EelConfig).(*EelSettings)
	}
	return nil
}

// GetDebugLogParams is a helper function to obtain the global debug white list and log information.
func GetDebugLogParams(ctx Context) *EelDebugLogParams {
	if ctx.ConfigValue(EelConfig) != nil {
		return ctx.ConfigValue(EelConfig).(*EelSettings).DebugLogParams
	}
	return nil
}

// GetTenant gets tenant id from context if one was passed in as http header.
func GetTenantId(ctx Context) string {
	if ctx.Value(EelTenantId) != nil {
		return ctx.Value(EelTenantId).(string)
	}
	return ""
}

// GetTenant gets tenant id from context if one was passed in as http header.
func GetCustomProperties(ctx Context) map[string]interface{} {
	if ctx.Value(EelCustomProperties) != nil {
		return ctx.Value(EelCustomProperties).(map[string]interface{})
	}
	return make(map[string]interface{}, 0)
}

// GetConfigFromFile loads config.json from disk and returns a pointer to a EelSettings struct.
func GetConfigFromFile(ctx Context) *EelSettings {
	configFile, err := os.Open(ConfigPath)
	if err != nil {
		// csv-context-go may not be ready yet for logging
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_config", "cause", "open_config", "error", err.Error())
		os.Exit(1)
	}
	defer configFile.Close()
	configData, err := ioutil.ReadAll(configFile)
	if err != nil {
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_config", "cause", "read_config", "error", err.Error())
		os.Exit(1)
	}
	var config EelSettings
	err = json.Unmarshal(configData, &config)
	if err != nil {
		fmt.Printf("{ \"error\" : \"%s\" }", err.Error())
		ctx.Log().Error("error_type", "get_config", "cause", "parse_config", "error", err.Error())
		os.Exit(1)
	}
	return &config
}

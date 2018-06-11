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

package test

import (
	"os"
	"time"

	. "github.com/Comcast/eel/jtl"
	. "github.com/Comcast/eel/util"
)

var TenantIds = []string{"", "tenant1", "tenant2"}

func initTests(handlers string) {
	LogsOn := os.Getenv("LOGS_ON")
	BasePath = ""
	ConfigPath = "../" + EelConfigFile
	HandlerPath = handlers
	InitGctx(L_InfoLevel)
	Gctx.AddLogValue("app.id", "eel")
	Gctx.AddValue("Eel.StartTime", time.Now().Local().Format("2006-01-02 15:04:05 +0800"))
	if LogsOn != "on" && LogsOn != "true" {
		Gctx.DisableLogging()
	}
	Gctx.AddValue(EelTotalStats, new(ServiceStats))
	Gctx.AddValue(Eel5MinStats, new(ServiceStats))
	ReloadConfig()
	InitHttpTransport(Gctx)

	Gctx.AddValue(EelTenantIds, TenantIds)
	for _, tenantId := range TenantIds {
		dp := NewWorkDispatcher(GetConfig(Gctx).WorkerPoolSize[""], GetConfig(Gctx).MessageQueueDepth, tenantId)
		dp.Start(Gctx)
		Gctx.AddValue(EelDispatcher+"_"+tenantId, dp)
	}

	//The majority of test cases are under tenant1 only
	Gctx.AddValue(EelTenantId, "tenant1")

	dc := NewLocalInMemoryDupChecker(GetConfig(Gctx).DuplicateTimeout, 10000)
	Gctx.AddValue(EelDuplicateChecker, dc)
	Gctx.AddConfigValue(EelSyncPath, "/proc")
}

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
	"bufio"
	"io"
	"os"
	"sync/atomic"
	"time"

	. "github.com/Comcast/eel/util"
)

type StdinPlugin struct {
	Settings     *PluginSettings
	ShuttingDown bool
}

func NewStdinPlugin(settings *PluginSettings) InboundPlugin {
	p := new(StdinPlugin)
	p.Settings = settings
	return p
}

func (p *StdinPlugin) GetSettings() *PluginSettings {
	return p.Settings
}

func (p *StdinPlugin) StartPlugin(ctx Context) {
	go p.StartStdInConsumer(ctx, os.Stdin)
}

func (p *StdinPlugin) StartStdInConsumer(ctx Context, r io.Reader) {
	defer ctx.HandlePanic()
	p.ShuttingDown = false
	p.Settings.Active = true
	//scanner := bufio.NewScanner(r)
	stdinreader := bufio.NewReader(r)
	ctx.Log().Info("action", "starting_plugin", "op", "stdin")
	tenantId := ""
	if ctx.Value(EelTenantId) != nil {
		tenantId = ctx.Value(EelTenantId).(string)
	}
	dp := GetWorkDispatcher(ctx, tenantId)
	if dp == nil {
		ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "no_work_dispatcher", "op", "stdin", "tenant_id", tenantId)
		return
	}
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	traceHeaderKey := GetConfig(ctx).HttpTransactionHeader
	timeOutMS := time.Duration(GetConfig(ctx).MessageQueueTimeout)
	//for scanner.Scan() {
	for {
		//body := scanner.Text()
		line, isPrefix, err := stdinreader.ReadLine()
		if err != nil {
			ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "read_line", "error", err.Error(), "action", "exiting_with_error_code", "expect", "restart_by_supervisord")
			break
		}
		body := string(line)
		for isPrefix {
			line, isPrefix, err = stdinreader.ReadLine()
			if err != nil {
				ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "read_line_continuation", "error", err.Error(), "action", "exiting_with_error_code", "expect", "restart_by_supervisord")
				break
			}
			ctx.Log().Info("action", "read_continuation", "op", "stdin")
			body += string(line)
		}
		if body == "" {
			ctx.Log().Info("action", "blank_message", "op", "stdin")
			continue
		}
		sctx := ctx.SubContext()
		sctx.AddLogValue("tx.traceId", sctx.Id())
		sctx.AddValue("tx.traceId", sctx.Id())
		sctx.AddValue(traceHeaderKey, sctx.Id())
		evt, err := NewJDocFromString(body)
		if err != nil {
			ctx.Log().Error("error_type", "rejected", "cause", "invalid_json", "error", err.Error(), "trace.in.data", body, "op", "stdin")
			ctx.Log().Metric("rejected", M_Namespace, "xrs", M_Metric, "rejected", M_Unit, "Count", M_Dims, "app="+AppId+"&env="+EnvName+"&instance="+InstanceName, M_Val, 1.0)
			stats.IncErrors()
			continue
		}
		if GetConfig(ctx).LogParams != nil {
			for k, v := range GetConfig(ctx).LogParams {
				ev := evt.ParseExpression(ctx, v)
				sctx.AddLogValue(k, ev)
			}
		}
		stats.IncBytesIn(len(body))
		work := WorkRequest{Raw: body, Event: evt, Ctx: sctx}
		select {
		case dp.WorkQueue <- &work:
			sctx.Log().Info("action", "accepted", "op", "stdin")
			atomic.AddUint64(&p.Settings.Stats.MessageCount, 1)
		case <-time.After(time.Millisecond * timeOutMS):
			sctx.Log().Error("error_type", "rejected", "action", "rejected", "op", "stdin", "cause", "queue_full")
		}
	}
	p.Settings.Active = false
	ctx.Log().Info("action", "stopping_plugin", "op", "stdin")
	if p.Settings.ExitOnErr {
		os.Exit(1)
	}
}

func (p *StdinPlugin) StopPlugin(ctx Context) {
	ctx.Log().Info("action", "shutdown_plugin", "op", "stdin", "details", "cannot_shutdonw")
}

func (p *StdinPlugin) IsActive() bool {
	return p.Settings.Active
}

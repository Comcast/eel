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
	"time"

	. "github.com/Comcast/eel/eel/util"
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
	p.StartStdInConsumer(ctx, os.Stdin)
}

func (p *StdinPlugin) StartStdInConsumer(ctx Context, r io.Reader) {
	p.ShuttingDown = false
	p.Settings.Active = true
	//scanner := bufio.NewScanner(r)
	stdinreader := bufio.NewReader(r)
	ctx.Log().Info("action", "starting_plugin", "op", "stdin")
	dp := GetWorkDispatcher(ctx)
	if dp == nil {
		ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "no_work_dispatcher", "op", "stdin")
		return
	}
	traceHeaderKey := GetConfig(ctx).HttpTransactionHeader
	timeOutMS := time.Duration(GetConfig(ctx).MessageQueueTimeout)
	//for scanner.Scan() {
	for {
		//msgBody := scanner.Text()
		line, isPrefix, err := stdinreader.ReadLine()
		if err != nil {
			ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "read_line", "error", err.Error(), "action", "exiting_with_error_code", "expect", "restart_by_supervisord")
			// exit here banking on supervisord to restart a healthy system
			os.Exit(1)
		}
		msgBody := string(line)
		for isPrefix {
			line, isPrefix, err = stdinreader.ReadLine()
			if err != nil {
				ctx.Log().Error("error_type", "stdin_consumer_error", "cause", "read_line_continuation", "error", err.Error(), "action", "exiting_with_error_code", "expect", "restart_by_supervisord")
				// exit here banking on supervisord to restart a healthy system
				os.Exit(1)
			}
			ctx.Log().Info("action", "read_continuation", "op", "stdin")
			msgBody += string(line)
		}
		if msgBody == "" {
			ctx.Log().Info("action", "blank_message", "op", "stdin")
			continue
		}
		sctx := ctx.SubContext()
		sctx.AddLogValue("tx.traceId", sctx.Id())
		sctx.AddValue("tx.traceId", sctx.Id())
		sctx.AddValue(traceHeaderKey, sctx.Id())
		work := WorkRequest{Message: msgBody, Ctx: sctx}
		select {
		case dp.WorkQueue <- &work:
			sctx.Log().Info("action", "accepted", "op", "stdin")
		case <-time.After(time.Millisecond * timeOutMS):
			sctx.Log().Error("error_type", "rejected", "action", "rejected", "op", "stdin", "cause", "queue_full")
		}
	}
	ctx.Log().Info("action", "stopping_plugin", "op", "stdin")
}

func (p *StdinPlugin) StopPlugin(ctx Context) {
	ctx.Log().Info("action", "shutdown_plugin", "op", "stdin", "details", "cannot_shutdonw")
}

func (p *StdinPlugin) CanShutdown() bool {
	return false
}

func (p *StdinPlugin) IsActive() bool {
	return p.Settings.Active
}

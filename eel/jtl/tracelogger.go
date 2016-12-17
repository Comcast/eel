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
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	. "github.com/Comcast/eel/eel/util"
)

type TraceLogger struct {
	File         *os.File
	Writer       *bufio.Writer
	Settings     *EelTraceLogParams
	EventChannel chan *JDoc
}

func NewTraceLogger(ctx Context, config *EelSettings) *TraceLogger {
	tl := new(TraceLogger)
	if config.TraceLogParams != nil && config.TraceLogParams.FileName != "" {
		tl.Settings = config.TraceLogParams
		tl.EventChannel = make(chan *JDoc)
		ctx.Log().Info("SETTINGS", tl.Settings)
		var err error
		tl.File, err = os.Create(config.TraceLogParams.FileName)
		if err != nil {
			ctx.Log().Error("error_type", "io_error", "op", "trace_logger", "cause", "unable_to_create_file", "error", err.Error())
		} else {
			tl.Writer = bufio.NewWriter(tl.File)
		}
	}
	tl.processTraceLogLoop(ctx)
	return tl
}

func TraceLogConfigHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	w.Header().Set("Content-Type", "application/json")
	state := make(map[string]interface{}, 0)
	state["Version"] = GetConfig(ctx).Version
	if ctx.ConfigValue(EelTraceLogger) != nil {
		tl := ctx.ConfigValue(EelTraceLogger).(*TraceLogger)
		tl.Settings.Active = !tl.Settings.Active
		GetConfig(ctx).TraceLogParams.Active = !GetConfig(ctx).TraceLogParams.Active
		state["TraceLogConfigs"] = tl.Settings
	}
	buf, err := json.MarshalIndent(state, "", "\t")
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
	} else {
		fmt.Fprintf(w, string(buf))
	}
}

func (t *TraceLogger) processTraceLogLoop(ctx Context) {
	go func() {
		for {
			event := <-t.EventChannel
			if t.Writer != nil && t.Settings.LogParams != nil {
				entry := make(map[string]interface{}, 0)
				for k, v := range t.Settings.LogParams {
					ev := event.ParseExpression(ctx, v)
					entry[k] = ev
				}
				buf, err := json.Marshal(entry)
				if err != nil {
					ctx.Log().Error("error_type", "json_error", "op", "trace_logger", "cause", "unable_to_marshal_entry", "error", err.Error())
				} else {
					t.Writer.WriteString(string(buf) + "\n")
				}
			}
		}
	}()
}

func (t *TraceLogger) TraceLog(ctx Context, event *JDoc, incoming bool) {
	if !t.Settings.Active {
		return
	}
	if incoming && !t.Settings.LogIncoming {
		return
	}
	if !incoming && !t.Settings.LogOutgoing {
		return
	}
	t.EventChannel <- event
}

func (t *TraceLogger) CloseTraceLog(ctx Context) {
	if t.File != nil {
		err := t.File.Close()
		if err != nil {
			ctx.Log().Error("error_type", "io_error", "op", "trace_logger", "cause", "unable_to_close_file", "error", err.Error())
		}
	}
}

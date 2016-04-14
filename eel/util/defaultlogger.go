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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	L_NilLevel    = "nil" // log nothing, equivalent to /dev/null
	L_MetricLevel = "metric"
	L_ErrorLevel  = "error"
	L_WarnLevel   = "warn"
	L_InfoLevel   = "info"
	L_DebugLevel  = "debug"
)

// DefaultLogWriter wraps the default stdout writer as a singelton (shared by all loggers)
type DefaultLogWriter struct {
	writer  *bufio.Writer
	enabled bool
	sync.RWMutex
}

var (
	defaultLogWriter = NewDefaultLogWriter()
)

func NewDefaultLogWriter() *DefaultLogWriter {
	dlw := new(DefaultLogWriter)
	dlw.writer = bufio.NewWriter(os.Stdout)
	dlw.enabled = true
	return dlw
}

// DefaultLogger simple implementation of the Logger interface.
type DefaultLogger struct {
	ctx                    *DefaultContext
	dfw                    *DefaultLogWriter
	level                  string
	err, warn, info, debug bool
}

// NewDefaultLogger creates a new default logger. There is one logger instance per context instance.
func NewDefaultLogger(ctx *DefaultContext, level string) *DefaultLogger {
	logger := new(DefaultLogger)
	logger.dfw = defaultLogWriter
	logger.ctx = ctx
	logger.level = level
	if level == L_DebugLevel {
		logger.debug = true
		logger.info = true
		logger.warn = true
		logger.err = true
	} else if level == L_InfoLevel {
		logger.info = true
		logger.warn = true
		logger.err = true
	} else if level == L_WarnLevel {
		logger.warn = true
		logger.err = true
	} else if level == L_ErrorLevel {
		logger.err = true
	}
	return logger
}

func (l *DefaultLogWriter) log(level string, id string, vals map[string]interface{}, args ...interface{}) error {
	if !l.enabled {
		return nil
	}
	l.Lock()
	defer l.Unlock()
	d := make(map[string]interface{}, 0)
	for k, v := range vals {
		d[k] = v
	}
	for i := 0; i < len(args)/2; i++ {
		d[args[2*i].(string)] = args[i*2+1]
	}
	d["log.id"] = id
	d["log.level"] = level
	d["log.timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	//d["log.type"] = "bw"
	buf, err := json.Marshal(d)
	if err != nil {
		fmt.Printf("{ \"log_error\" : \"%s\"}\n", err.Error())
		return err
	}
	l.writer.Write(buf)
	l.writer.WriteString("\n")
	l.writer.Flush()
	return nil
}

func (l *DefaultLogger) Debug(args ...interface{}) {
	if l.debug {
		l.dfw.log(L_DebugLevel, l.ctx.Id(), l.ctx.lvals, args...)
	}
}

func (l *DefaultLogger) Info(args ...interface{}) {
	if l.info {
		l.dfw.log(L_InfoLevel, l.ctx.Id(), l.ctx.lvals, args...)
	}
}

func (l *DefaultLogger) Warn(args ...interface{}) {
	if l.warn {
		l.dfw.log(L_WarnLevel, l.ctx.Id(), l.ctx.lvals, args...)
	}
}

func (l *DefaultLogger) Error(args ...interface{}) {
	if l.err {
		l.dfw.log(L_ErrorLevel, l.ctx.Id(), l.ctx.lvals, args...)
	}
}

func (l *DefaultLogger) Metric(statKey interface{}, args ...interface{}) {
	//if l.info || l.debug {
	//l.dfw.log(L_MetricLevel, l.ctx.Id(), l.ctx.lvals, args...)
	//}
}

func (l *DefaultLogger) RuntimeLogLoop(interval time.Duration, iterations int) {
}

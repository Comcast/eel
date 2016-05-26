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
	"fmt"
	"time"
)

// Context is the interface for a request context and logging.
type Context interface {
	SetId(id string)
	Id() string
	SubContext() Context
	AddValue(key interface{}, value interface{})
	AddLogValue(key interface{}, value interface{})
	AddConfigValue(key interface{}, value interface{})
	Value(key interface{}) interface{}
	LogValue(key interface{}) interface{}
	ConfigValue(key interface{}) interface{}
	Log() Logger
	DisableLogging()
	EnableLogging()
}

// Logger is the interface for logging.
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Warn(args ...interface{})
	Metric(statKey interface{}, args ...interface{})
	RuntimeLogLoop(interval time.Duration, iterations int)
}

type NetworkError struct {
	Endpoint string
	Message  string
	Status   int
}

func (e NetworkError) Error() string {
	return fmt.Sprintf("error reaching endpoint: %s: status: %d message: %s", e.Endpoint, e.Status, e.Message)
}

type SyntaxError struct {
	Message  string
	Function string
}

func (e SyntaxError) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

type RuntimeError struct {
	Message  string
	Function string
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

type ParseError struct {
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

// ClearErrors clears any stale errors from current transacction in case lib user recycles contexts
func ClearErrors(ctx Context) {
	if ctx == nil {
		return
	}
	ctx.AddValue(EelErrors, make([]error, 0))
}

// AddError adds error to list of errors in current transaction in current context for lib use
func AddError(ctx Context, err error) {
	if ctx == nil || err == nil {
		return
	}
	if ctx.Value(EelErrors) == nil {
		ctx.AddValue(EelErrors, make([]error, 0))
	}
	e := ctx.Value(EelErrors).([]error)
	e = append(e, err)
	ctx.AddValue(EelErrors, e)
}

// GetErrors gets list of errors in current transaction from current context for lib use
func GetErrors(ctx Context) []error {
	if ctx == nil || ctx.Value(EelErrors) == nil {
		return nil
	}
	errs := ctx.Value(EelErrors).([]error)
	if len(errs) == 0 {
		return nil
	}
	return errs
}

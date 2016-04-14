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

import "time"

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

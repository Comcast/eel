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

type (
	// Observer is the common interface for metrics and tracing
	Observer interface {
		Start(ctx Context, trace string, attrs map[string]string) Context
		End(ctx Context, attrs map[string]string, err error)
		Record(ctx Context, metric string, attrs map[string]string, val int)
	}
)

// RegisterObserver registers an observer implementation
func RegisterObserver(ctx Context, o Observer) {
	ctx.AddValue(EelObserver, o)
}

// Start starts a new span
func Start(ctx Context, trace string, attrs map[string]string) Context {
	if o, ok := ctx.Value(EelObserver).(Observer); ok {
		return o.Start(ctx, trace, attrs)
	}

	return ctx.SubContext()
}

// End ends the existing span
func End(ctx Context, attrs map[string]string, err error) {
	if o, ok := ctx.Value(EelObserver).(Observer); ok {
		o.End(ctx, attrs, err)
	}
}

// Record records a metric if given
func Record(ctx Context, metric string, attrs map[string]string, val int) {
	if o, ok := ctx.Value(EelObserver).(Observer); ok {
		o.Record(ctx, metric, attrs, val)
	}
}

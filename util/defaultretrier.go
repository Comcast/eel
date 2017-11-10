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
	"time"
)

type DefaultRetrier struct {
}

// RetryEndpoint same as HitEndpoint but with local trivial implementation exponential backoff.
func (d *DefaultRetrier) RetryEndpoint(ctx Context, url string, payload string, verb string, headers map[string]string, auth map[string]string) (string, int, error) {
	return d.Retry(ctx, url, payload, verb, headers, auth, HitEndpoint)
}

// Retry implements retry logic with injected request function.
func (*DefaultRetrier) Retry(ctx Context, url string, payload string, verb string, headers map[string]string, auth map[string]string, f func(ctx Context, url string, payload string, verb string, headers map[string]string, auth map[string]string) (string, int, error)) (string, int, error) {
	attempt := 1
	backoffMs := GetConfig(ctx).InitialBackoff * time.Millisecond
	initialDelayMs := GetConfig(ctx).InitialDelay * time.Millisecond
	padMs := GetConfig(ctx).Pad * time.Millisecond
start:
	ctx.AddLogValue("attempt", attempt)
	resp, status, err := f(ctx, url, payload, verb, headers, auth)
	if err != nil || status < 200 || status > 499 {
		if attempt < GetConfig(ctx).MaxAttempts {
			if attempt == 1 {
				time.Sleep(initialDelayMs)
			} else {
				time.Sleep(backoffMs + padMs)
				backoffMs *= 2
			}
			attempt++
			goto start
		}
	}
	return resp, status, err
}

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

// Retrier is the interface for retrying failed http requests.
type Retrier interface {
	RetryEndpoint(Context, string, string, string, map[string]string, map[string]string) (string, int, error)
	Retry(Context, string, string, string, map[string]string, map[string]string, func(Context, string, string, string, map[string]string, map[string]string) (string, int, error)) (string, int, error)
}

var defaultRetrier Retrier

func init() {
	defaultRetrier = new(DefaultRetrier)
}

// SetDefaultRetrier sets the default retrier to an external implementation
func SetDefaultRetrier(retrier Retrier) {
	defaultRetrier = retrier
}

// GetRetrier gets the Retrier to be used
func GetRetrier(ctx Context) Retrier {
	return defaultRetrier
}

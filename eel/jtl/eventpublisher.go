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
	. "github.com/Comcast/eel/eel/util"
)

type (
	// EventPublisher is the common interface for all event publishers
	EventPublisher interface {
		// Publish sends the transformed event payload to the desired endpoint using the appropriate protocol (for example, HTTP POST).
		Publish() (string, error)
		SetEndpoint(endpoint string)
		GetEndpoint() string
		SetPayload(payload string)
		GetPayload() string
		SetAuthInfo(auth map[string]string)
		GetProtocol() string
		GetApi() string
		SetHeaders(headers map[string]string)
		GetHeaders() map[string]string
		SetVerb(verb string)
		GetVerb() string
		SetPath(path string)
		GetPath() string
		GetUrl() string
		SetPayloadParsed(event *JDoc)
		GetPayloadParsed() *JDoc
	}
)

var publisherMap = make(map[string]NewPublisher, 0)

type NewPublisher func(ctx Context) EventPublisher

func init() {
	publisherMap["http"] = NewHttpPublisher
}

// RegisterEventPublisher registers an external event publisher implementation for a new protocol
func RegisterEventPublisher(newPublisher NewPublisher, protocol string) {
	// maybe return an error if a publisher is already registered
	publisherMap[protocol] = newPublisher
}

// UnregisterEventPublisher removes a publisher implementation
func UnregisterEventPublisher(protocol string) {
	delete(publisherMap, protocol)
}

// NewEventPublisher factory method to return matching publisher for a given protocol
func NewEventPublisher(ctx Context, protocol string) EventPublisher {
	if newPublisher, ok := publisherMap[protocol]; ok {
		return newPublisher(ctx)
	}
	return nil
}

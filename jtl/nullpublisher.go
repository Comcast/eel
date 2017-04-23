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
	. "github.com/Comcast/eel/util"
)

type (
	NullPublisher struct {
		endpoint string
		path     string
		payload  string
		protocol string
		api      string
		verb     string
		debug    bool
		auth     map[string]string
		headers  map[string]string
		handler  *HandlerConfiguration
		event    *JDoc
		ctx      Context
	}
)

// NewNullPublisher creates a new HTTP publisher.
func NewNullPublisher(ctx Context) EventPublisher {
	hp := new(NullPublisher)
	hp.ctx = ctx
	hp.protocol = "null"
	hp.api = "null"
	ctx.AddLogValue("destination", "null")
	return hp
}

func (p *NullPublisher) Publish() (string, error) {
	return "", nil
}

func (p *NullPublisher) GetUrl() string {
	return ""
}

func (p *NullPublisher) GetErrors() []error {
	return GetErrors(p.ctx)
}

func (p *NullPublisher) SetDebug(debug bool) {
	p.debug = debug
}

func (p *NullPublisher) GetDebug() bool {
	return p.debug
}

func (p *NullPublisher) SetPath(path string) {
	p.path = path
}

func (p *NullPublisher) GetPath() string {
	return p.path
}

func (p *NullPublisher) SetEndpoint(endpoint string) {
	p.endpoint = endpoint
}

func (p *NullPublisher) GetEndpoint() string {
	return p.endpoint
}

func (p *NullPublisher) SetPayload(payload string) {
	p.payload = payload
}

func (p *NullPublisher) GetPayload() string {
	return p.payload
}

func (p *NullPublisher) SetVerb(verb string) {
	p.verb = verb
}

func (p *NullPublisher) GetVerb() string {
	return p.verb
}

func (p *NullPublisher) GetProtocol() string {
	return p.protocol
}

func (p *NullPublisher) GetApi() string {
	return p.api
}

func (p *NullPublisher) SetAuthInfo(auth map[string]string) {
	p.auth = auth
}

func (p *NullPublisher) SetHeaders(headers map[string]string) {
	p.headers = headers
}

func (p *NullPublisher) GetHeaders() map[string]string {
	return p.headers
}

func (p *NullPublisher) SetPayloadParsed(event *JDoc) {
	p.event = event
}

func (p *NullPublisher) GetPayloadParsed() *JDoc {
	return p.event
}

func (p *NullPublisher) SetHandler(handler *HandlerConfiguration) {
	p.handler = handler
}

func (p *NullPublisher) GetHandler() *HandlerConfiguration {
	return p.handler
}

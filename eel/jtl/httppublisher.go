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
	"errors"
	"strings"

	. "github.com/Comcast/eel/eel/util"
)

type (
	HttpPublisher struct {
		endpoint string
		path     string
		payload  string
		protocol string
		api      string
		verb     string
		auth     map[string]string
		headers  map[string]string
		event    *JDoc
		ctx      Context
	}
)

// NewHttpPublisher creates a new HTTP publisher.
func NewHttpPublisher(ctx Context) EventPublisher {
	hp := new(HttpPublisher)
	hp.ctx = ctx
	hp.protocol = "http"
	hp.api = "http"
	ctx.AddLogValue("destination", "main_http")
	return hp
}

func (p *HttpPublisher) Publish() (string, error) {
	if p.endpoint == "" {
		return "", errors.New("missing endpoint")
	}
	if p.verb == "" {
		return "", errors.New("missing verb")
	}
	resp, status, err := GetRetrier(p.ctx).RetryEndpoint(p.ctx, p.GetUrl(), p.payload, p.verb, p.headers, nil)
	if err != nil {
		return resp, err
	}
	if status < 200 || status >= 300 {
		return resp, NetworkError{p.endpoint, "endpoint returned error", status}
	}
	return resp, nil
}

func (p *HttpPublisher) GetUrl() string {
	if p.endpoint == "" {
		return p.path
	}
	if p.path == "" {
		return p.endpoint
	}
	if strings.HasSuffix(p.endpoint, "/") && strings.HasPrefix(p.path, "/") {
		return p.endpoint + p.path[1:]
	} else if strings.HasSuffix(p.endpoint, "/") || strings.HasPrefix(p.path, "/") {
		return p.endpoint + p.path
	}
	return p.endpoint + "/" + p.path
}

func (p *HttpPublisher) GetErrors() []error {
	return GetErrors(p.ctx)
}

func (p *HttpPublisher) SetPath(path string) {
	p.path = path
}

func (p *HttpPublisher) GetPath() string {
	return p.path
}

func (p *HttpPublisher) SetEndpoint(endpoint string) {
	p.endpoint = endpoint
}

func (p *HttpPublisher) GetEndpoint() string {
	return p.endpoint
}

func (p *HttpPublisher) SetPayload(payload string) {
	p.payload = payload
}

func (p *HttpPublisher) GetPayload() string {
	return p.payload
}

func (p *HttpPublisher) SetVerb(verb string) {
	p.verb = verb
}

func (p *HttpPublisher) GetVerb() string {
	return p.verb
}

func (p *HttpPublisher) GetProtocol() string {
	return p.protocol
}

func (p *HttpPublisher) GetApi() string {
	return p.api
}

func (p *HttpPublisher) SetAuthInfo(auth map[string]string) {
	p.auth = auth
}

func (p *HttpPublisher) SetHeaders(headers map[string]string) {
	p.headers = headers
}

func (p *HttpPublisher) GetHeaders() map[string]string {
	return p.headers
}

func (p *HttpPublisher) SetPayloadParsed(event *JDoc) {
	p.event = event
}

func (p *HttpPublisher) GetPayloadParsed() *JDoc {
	return p.event
}

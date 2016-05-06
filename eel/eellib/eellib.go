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

package eellib

import (
	"encoding/json"
	"errors"

	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

// EELInit initalize environment for EEL API use
func EELInit(ctx Context) {
	Gctx = ctx
	eelSettings := new(EelSettings)
	ctx.AddConfigValue(EelConfig, eelSettings)
	eelServiceStats := new(ServiceStats)
	ctx.AddValue(EelTotalStats, eelServiceStats)
	InitHttpTransport(ctx)
}

// EELNewHandlerFactory creates handler factory for given folder with handler files.
func EELNewHandlerFactory(ctx Context, configFolder string) (*HandlerFactory, []string) {
	if Gctx == nil {
		return nil, []string{"must call EELInit first"}
	}
	if ctx == nil {
		return nil, []string{"ctx cannot be nil"}
	}
	eelHandlerFactory, warnings := NewHandlerFactory(ctx, []string{configFolder})
	return eelHandlerFactory, warnings
}

// EELGetHandlersForEvent gets all handlers for a given event.
func EELGetHandlersForEvent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]*HandlerConfiguration, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
	}
	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, err
	}
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	return eelMatchingHandlers, nil
}

// EELGetPublishers is similar to EELTransformEvent but return slice of publishers instead of slice of events.
func EELGetPublishers(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]EventPublisher, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
	}
	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, err
	}
	publishers := make([]EventPublisher, 0)
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	for _, h := range eelMatchingHandlers {
		p, err := h.ProcessEvent(ctx, doc)
		if err != nil {
			return nil, err
		}
		publishers = append(publishers, p...)
	}
	return publishers, nil
}

// EELTransformEvent transforms single event based on set of configuration handlers. Can yield multiple results.
func EELTransformEvent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]interface{}, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
	}
	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}
	publishers, err := EELGetPublishers(ctx, event, eelHandlerFactory)
	if err != nil {
		return nil, err
	}
	events := make([]interface{}, 0)
	for _, p := range publishers {
		events = append(events, p.GetPayloadParsed().GetOriginalObject())
	}
	return events, nil
}

// EELSingleTransform can work with raw JSON transformation or a transformation wrapped in a config handler.
// Transformation must yield single result or no result (if filtered).
func EELSimpleTransform(ctx Context, event string, transformation string, isTransformationByExample bool) (string, error) {
	if Gctx == nil {
		return "", errors.New("must call EELInit first")
	}
	if ctx == nil {
		return "", errors.New("ctx cannot be nil")
	}
	doc, err := NewJDocFromString(event)
	if err != nil {
		return "", err
	}
	tf, err := NewJDocFromString(transformation)
	if err != nil {
		return "", err
	}
	h := new(HandlerConfiguration)
	err = json.Unmarshal([]byte(transformation), h)
	if err != nil || h.Transformation == nil {
		h.Transformation = tf.GetOriginalObject()
		h.IsTransformationByExample = isTransformationByExample
	}
	if h.Protocol == "" {
		h.Protocol = "http"
	}
	if h.Endpoint == nil {
		h.Endpoint = "http://localhost"
	}
	hf, _ := NewHandlerFactory(ctx, nil)
	h, _ = hf.GetHandlerConfigurationFromJson(ctx, "", *h)
	p, err := h.ProcessEvent(ctx, doc)
	if err != nil {
		return "", err
	}
	if len(p) > 1 {
		return "", errors.New("transformation must yield single result")
	} else if len(p) == 0 {
		return "", nil
	} else {
		return p[0].GetPayload(), nil
	}
}

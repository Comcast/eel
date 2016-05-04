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
	"errors"

	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

func EELInit(ctx Context) {
	Gctx = ctx
	eelSettings := new(EelSettings)
	ctx.AddConfigValue(EelConfig, eelSettings)
	eelServiceStats := new(ServiceStats)
	ctx.AddValue(EelTotalStats, eelServiceStats)
	InitHttpTransport(ctx)
}

func EELNewHandlerFactory(ctx Context, configFolder string) (*HandlerFactory, []string) {
	if Gctx == nil {
		return nil, []string{"must call EELInit first"}
	}
	eelHandlerFactory, warnings := NewHandlerFactory(ctx, []string{configFolder})
	return eelHandlerFactory, warnings
}

func EELGetHandlersForEvent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]*HandlerConfiguration, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
	}
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, err
	}
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	return eelMatchingHandlers, nil
}

func EELGetPublishers(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]EventPublisher, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
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

func EELTransformEvent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]interface{}, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
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

func EELSingleTransform(ctx Context, event string, transformation string, isTransformationByExample bool) (string, error) {
	if Gctx == nil {
		return "", errors.New("must call EELInit first")
	}
	tf, err := NewJDocFromString(transformation)
	if err != nil {
		return "", err
	}
	doc, err := NewJDocFromString(event)
	if err != nil {
		return "", err
	}
	var tfd *JDoc
	if isTransformationByExample {
		tfd = doc.ApplyTransformationByExample(ctx, tf)
	} else {
		tfd = doc.ApplyTransformation(ctx, tf)
	}
	if tfd == nil {
		return "", errors.New("bad transformation")
	}
	return tfd.StringPretty(), nil
}

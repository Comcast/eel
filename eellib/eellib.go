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
	"sync"

	. "github.com/Comcast/eel/jtl"
	. "github.com/Comcast/eel/util"
)

// EELInit initalize environment for EEL API use
func EELInit(ctx Context) {
	Gctx = ctx
	eelSettings := new(EelSettings)
	eelSettings.MaxAttempts = 3
	eelSettings.InitialDelay = 125
	eelSettings.InitialBackoff = 500
	eelSettings.BackoffMethod = "Exponential"
	eelSettings.HttpTimeout = 3000
	eelSettings.ResponseHeaderTimeout = 3000
	eelSettings.MaxMessageSize = 512000
	eelSettings.HttpTransactionHeader = "X-B3-TraceId"
	eelSettings.HttpTenantHeader = "Xrs-Tenant-Id"
	eelSettings.AppName = "eellib"
	eelSettings.Name = "eellib"
	eelSettings.Version = "1.0"
	ctx.AddConfigValue(EelConfig, eelSettings)
	eelServiceStats := new(ServiceStats)
	ctx.AddValue(EelTotalStats, eelServiceStats)
	InitHttpTransport(ctx)
}

// EELGetSettings get current settings for read / write
func EELGetSettings(ctx Context) (*EelSettings, error) {
	if Gctx == nil {
		return nil, errors.New("must call EELInit first")
	}
	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}
	if ctx.ConfigValue(EelConfig) == nil {
		return nil, errors.New("no settings")
	}
	return ctx.ConfigValue(EelConfig).(*EelSettings), nil
}

// EELGetSettings get current settings for read / write
func EELUpdateSettings(ctx Context, eelSettings *EelSettings) error {
	if Gctx == nil {
		return errors.New("must call EELInit first")
	}
	if ctx == nil {
		return errors.New("ctx cannot be nil")
	}
	if eelSettings == nil {
		return errors.New("no settings")
	}
	ctx.AddValue(EelConfig, eelSettings)
	InitHttpTransport(ctx)
	return nil
}

// EELNewHandlerFactory creates handler factory for given folder with handler files.
func EELNewHandlerFactory(ctx Context, configFolder string) (*HandlerFactory, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
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
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, err
	}
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	return eelMatchingHandlers, nil
}

// EELGetPublishers is similar to EELTransformEvent but return slice of publishers instead of slice of events.
func EELGetPublishers(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]EventPublisher, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, []error{err}
	}
	publishers := make([]EventPublisher, 0)
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	for _, h := range eelMatchingHandlers {
		p, err := h.ProcessEvent(ctx, doc)
		if err != nil {
			return nil, []error{err}
		}
		publishers = append(publishers, p...)
	}
	return publishers, GetErrors(ctx)
}

// EELGetPublishersConcurrent is the concurrent version of EELGetPublishers. Useful when processing multiple
// expensive transformations at the same time (e.g. making heavy use of curl() function calls).
func EELGetPublishersConcurrent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]EventPublisher, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	doc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, []error{err}
	}
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, doc)
	var mtx sync.Mutex
	var wg sync.WaitGroup
	publishers := make([]EventPublisher, 0)
	errs := make([]error, 0)
	for _, h := range eelMatchingHandlers {
		wg.Add(1)
		go func(h *HandlerConfiguration) {
			sctx := ctx.SubContext()
			d, _ := NewJDocFromInterface(event)
			pp, err := h.ProcessEvent(sctx, d)
			if err != nil {
				mtx.Lock()
				errs = append(errs, err)
				mtx.Unlock()
				wg.Done()
				return
			}
			ee := GetErrors(sctx)
			if ee != nil && len(ee) > 0 {
				mtx.Lock()
				errs = append(errs, ee...)
				mtx.Unlock()
			}
			if pp != nil && len(pp) > 0 {
				mtx.Lock()
				publishers = append(publishers, pp...)
				mtx.Unlock()
			}
			wg.Done()
		}(h)
	}
	wg.Wait()
	if len(errs) > 0 {
		return publishers, errs
	} else {
		return publishers, nil
	}
}

// EELTransformEvent transforms single event based on set of configuration handlers. Can yield multiple results.
func EELTransformEventByHandlerName(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory, tenant string, name string) ([]interface{}, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)

	handler := eelHandlerFactory.GetHandlerByName(ctx, tenant, name)
	if handler == nil {
		return nil, []error{errors.New("no handler " + tenant + " / " + name)}
	}
	eventDoc, err := NewJDocFromInterface(event)
	if err != nil {
		return nil, []error{errors.New("bad event")}
	}
	publishers, err := handler.ProcessEvent(ctx, eventDoc)
	if err != nil && publishers == nil {
		return nil, []error{err}
	}
	events := make([]interface{}, 0)
	for _, p := range publishers {
		events = append(events, p.GetPayloadParsed().GetOriginalObject())
	}
	return events, GetErrors(ctx)
}

// EELTransformEvent transforms single event based on set of configuration handlers. Can yield multiple results.
func EELTransformEvent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]interface{}, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	publishers, errs := EELGetPublishers(ctx, event, eelHandlerFactory)
	if errs != nil && publishers == nil {
		return nil, errs
	}
	events := make([]interface{}, 0)
	for _, p := range publishers {
		events = append(events, p.GetPayloadParsed().GetOriginalObject())
	}
	if errs != nil {
		errs = append(errs, GetErrors(ctx)...)
	} else {
		errs = GetErrors(ctx)
	}
	return events, errs
}

// EELTransformEventConcurrent is the concurrent version of EELTransformEvent. Useful when processing multiple
// expensive transformations at the same time (e.g. making heavy use of curl() function calls).
func EELTransformEventConcurrent(ctx Context, event interface{}, eelHandlerFactory *HandlerFactory) ([]interface{}, []error) {
	if Gctx == nil {
		return nil, []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return nil, []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	publishers, errs := EELGetPublishersConcurrent(ctx, event, eelHandlerFactory)
	if errs != nil && publishers == nil {
		return nil, errs
	}
	events := make([]interface{}, 0)
	for _, p := range publishers {
		events = append(events, p.GetPayloadParsed().GetOriginalObject())
	}
	if errs != nil {
		errs = append(errs, GetErrors(ctx)...)
	} else {
		errs = GetErrors(ctx)
	}
	return events, errs
}

// EELSingleTransform can work with raw JSON transformation or a transformation wrapped in a config handler.
// Transformation must yield single result or no result (if filtered).
func EELSimpleTransform(ctx Context, event string, transformation string, isTransformationByExample bool) (string, []error) {
	if Gctx == nil {
		return "", []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return "", []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	doc, err := NewJDocFromString(event)
	if err != nil {
		return "", []error{err}
	}
	tf, err := NewJDocFromString(transformation)
	if err != nil {
		return "", []error{err}
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
		return "", []error{err}
	}
	if len(p) > 1 {
		return "", []error{errors.New("transformation must yield single result")}
	} else if len(p) == 0 {
		return "", nil
	} else {
		return p[0].GetPayload(), GetErrors(ctx)
	}
}

// EELSingleTransform can work with raw JSON transformation or a transformation wrapped in a config handler.
// Transformation must yield single result or no result (if filtered).
func EELSimpleEvalExpression(ctx Context, event string, expr string) (string, []error) {
	if Gctx == nil {
		return "", []error{errors.New("must call EELInit first")}
	}
	if ctx == nil {
		return "", []error{errors.New("ctx cannot be nil")}
	}
	ctx = ctx.SubContext()
	ClearErrors(ctx)
	doc, err := NewJDocFromString(event)
	if err != nil {
		return "", []error{err}
	}
	result := doc.ParseExpression(ctx, expr)
	if result == nil {
		return "", GetErrors(ctx)
	}
	rdoc, err := NewJDocFromInterface(result)
	if err != nil {
		return "", []error{err}
	}
	return rdoc.StringPretty(), GetErrors(ctx)
}

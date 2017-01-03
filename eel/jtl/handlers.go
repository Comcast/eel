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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	. "github.com/Comcast/eel/eel/util"
)

type (
	// HandlerConfiguration is the central configuration structure for topic handlers as well as custom match handlers.
	HandlerConfiguration struct {
		// core handler settings
		Version  string // arbitrary version
		Name     string // short name (should be unique within a tenant)
		File     string // file location
		Info     string // description
		TenantId string // tenant id (the tenant id in the match section, if present, is just a meaningless custom match value!)
		Active   bool   // only used if set to true
		// matching events to handlers
		TerminateOnMatch bool                   // terminate looking for further topic handlers if set to true and this handler matches
		Topic            string                 // for matching input events to handler configs using hierarchical topic pattern, e.g. "/a/b/c" or "a/*/c" or "" (only Topic or Match can be used!)
		Match            map[string]interface{} // for matching input events to handler configs, based on matching key-value pairs (by path or by example)
		IsMatchByExample bool                   // choose syntax style by path or by example for handler matching
		// payload generation
		Transformation            interface{}                // main transformation (by path or by example)
		IsTransformationByExample bool                       // choose syntax style by path or by example for event transformation
		Transformations           map[string]*Transformation // optional - named transformations, used by transform() function
		// custom properties
		CustomProperties map[string]interface{} // optional - overrides custom properties in config.json, in addition, map values can be jpath expessions
		// filtering by pattern
		Filter                    map[string]interface{} // optional - only forward event if event matches this pattern (by path or by example)
		IsFilterByExample         bool                   // optional - choose syntax style by path or by example for event filtering
		IsFilterInverted          bool                   // optional - true: filter if event matches pattern, false: filter if event does not match pattern
		FilterAfterTransformation bool                   // optional - true: apply filters after transformation, false (default): apply filter before transformation on raw event
		// filtering by boolean expression
		FilterIfTrue  string // optional - filter event if this expression resolves to true
		FilterIfFalse string // optional - filter event if this expression resolves to false
		// several filters if desired
		Filters []*Filter
		// outgoing HTTP config
		Path        interface{}       // relative path added to endpoint URL, if array of multiple paths, event will be fanned out to endpoint/path1, endpoint/path2 etc.
		Verb        string            // otpional - HTTP verb like PUT, POST
		AuthInfo    map[string]string // optional - to overwrite default auth info, example: {"type":"basic","username":"foo","password":"bar"}
		Protocol    string            // optional - if omitted defaults to http, other valid values: x1, emo, email, sms (the protocol in the match section, if present, is just a meaningless custom match value!)
		Endpoint    interface{}       // optional - overwrite default endpoint from config.json, if array of multiple endpoints, event will be fanned out to endpoint1, endpoint2 etc.
		HttpHeaders map[string]string // optional - http headers
		// internal pre-compiled configs
		t *JDoc // transformation
		f *JDoc // filter
		m *JDoc // match
	}
	handlerMatchInstance struct {
		handler  *HandlerConfiguration
		strength int
	}
	handlerMatchInstanceList []*handlerMatchInstance
	// HandlerFactory factory providing correct handler(s) for a given document
	HandlerFactory struct {
		CustomHandlerMap map[string]map[string]*HandlerConfiguration   // tenant_id -> handler_name ->  handler config
		TopicHandlerMap  map[string]map[string][]*HandlerConfiguration // tenant_id -> topic_name -> list of topic handlers
	}
)

func (hmil handlerMatchInstanceList) Len() int           { return len(hmil) }
func (hmil handlerMatchInstanceList) Swap(i, j int)      { hmil[i], hmil[j] = hmil[j], hmil[i] }
func (hmil handlerMatchInstanceList) Less(i, j int) bool { return hmil[i].strength > hmil[j].strength }

func newHandlerMatchInstance(handler *HandlerConfiguration, strength int) *handlerMatchInstance {
	hmi := new(handlerMatchInstance)
	hmi.handler = handler
	hmi.strength = strength
	return hmi
}

// ReloadConfig reloads config.json as well as all handler configs from disk.
func ReloadConfig() {
	config := GetConfigFromFile(Gctx)
	Gctx.Log().Info("action", "load_config", "config", *config)
	Gctx.AddConfigValue(EelConfig, config)
	HandlerPaths = make([]string, 0)
	if HandlerPath != "" {
		HandlerPaths = append(HandlerPaths, filepath.Join(BasePath, HandlerPath))
	}
	if len(HandlerPaths) == 0 {
		HandlerPaths = append(HandlerPaths, filepath.Join(BasePath, DefaultConfigFolder))
	}
	hf, _ := NewHandlerFactory(Gctx, HandlerPaths)
	Gctx.AddConfigValue(EelHandlerFactory, hf)
}

// GetHandlerFactory get current instance of handler factory from context.
func GetHandlerFactory(ctx Context) *HandlerFactory {
	if ctx.ConfigValue(EelHandlerFactory) != nil {
		return ctx.ConfigValue(EelHandlerFactory).(*HandlerFactory)
	}
	return nil
}

// GetConfig is a helper function to obtain the global config from the context.
func GetCurrentHandlerConfig(ctx Context) *HandlerConfiguration {
	if ctx.Value(EelHandlerConfig) != nil {
		return ctx.Value(EelHandlerConfig).(*HandlerConfiguration)
	}
	return nil
}

// NewHandlerFactory creates new handler factory and loads handler files from all config folders.
func NewHandlerFactory(ctx Context, configFolders []string) (*HandlerFactory, []error) {
	if configFolders == nil {
		configFolders = []string{ConfigPath}
	}
	warnings := make([]error, 0)
	hf := new(HandlerFactory)
	hf.TopicHandlerMap = make(map[string]map[string][]*HandlerConfiguration, 0)
	hf.CustomHandlerMap = make(map[string]map[string]*HandlerConfiguration, 0)
	tenantMap := make(map[string]bool, 0)
	for _, folder := range configFolders {
		configFiles := hf.getAllConfigurationFiles(ctx, folder)
		for _, configFile := range configFiles {
			handler, w := hf.GetHandlerConfigurationFromFile(ctx, configFile)
			warnings = append(warnings, w...)
			if handler != nil && handler.Active {
				if handler.Topic != "" {
					// if is topic handler
					if _, ok := hf.TopicHandlerMap[handler.TenantId]; !ok {
						hf.TopicHandlerMap[handler.TenantId] = make(map[string][]*HandlerConfiguration)
					}
					if _, ok := hf.TopicHandlerMap[handler.TenantId][handler.Topic]; !ok {
						hf.TopicHandlerMap[handler.TenantId][handler.Topic] = make([]*HandlerConfiguration, 0)
					}
					ctx.Log().Info("action", "registering_topic_handler", "tenant", handler.TenantId, "name", handler.Name, "topic", handler.Topic)
					hf.TopicHandlerMap[handler.TenantId][handler.Topic] = append(hf.TopicHandlerMap[handler.TenantId][handler.Topic], handler)
				} else {
					// custom match handler or default handler (formerly known as notification handler)
					if _, ok := hf.CustomHandlerMap[handler.TenantId]; !ok {
						hf.CustomHandlerMap[handler.TenantId] = make(map[string]*HandlerConfiguration, 0)
					}
					hf.CustomHandlerMap[handler.TenantId][handler.Name] = handler
					ctx.Log().Info("action", "registering_handler", "tenant", handler.TenantId, "name", handler.Name, "match", handler.Match)
				}
				tenantMap[handler.TenantId] = true
			}
		}
	}
	return hf, warnings
}

func (h *HandlerConfiguration) GetNamedTransformation() map[string]*Transformation {
	return h.Transformations
}

func (h *HandlerConfiguration) GetTransformation() *JDoc {
	return h.t
}

func (h *HandlerConfiguration) GetFilter() *JDoc {
	return h.f
}

func (h *HandlerConfiguration) Save() error {
	buf, err := json.MarshalIndent(h, "", "\t")
	if err != nil {
		return err
	}
	Gctx.Log().Info("file", h.File, "buf", string(buf))
	return ioutil.WriteFile(h.File, buf, 0644)
}

func (hf *HandlerFactory) getPartialTopicHandlersForTenant(tenantId string, currentTopic string) ([]*HandlerConfiguration, bool) {
	hls := make([]*HandlerConfiguration, 0)
	segments := strings.Split(currentTopic, "/")
	candidates := make([]string, 0)
	// exact match
	candidates = append(candidates, currentTopic)
	// single wild card
	for i := 1; i < len(segments)-1; i++ {
		t := ""
		for k := 1; k < len(segments); k++ {
			if k == i {
				t += "/*"
			} else {
				t += "/" + segments[k]
			}
		}
		candidates = append(candidates, t)
	}
	// double wild card
	for i := 1; i < len(segments)-1; i++ {
		for j := i + 1; j < len(segments)-1; j++ {
			t := ""
			for k := 1; k < len(segments); k++ {
				if k == i || k == j {
					t += "/*"
				} else {
					t += "/" + segments[k]
				}
			}
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return hls, false
	}
	tenantTopicHandlers := hf.TopicHandlerMap[tenantId]
	for _, c := range candidates {
		if handlers, ok := tenantTopicHandlers[c]; ok {
			hls = append(hls, handlers...)
			for _, h := range handlers {
				if h.TerminateOnMatch {
					return hls, true
				}
			}
		}
	}
	return hls, false
}

// GetTopicHandlersForTenant retrieves topic handlers by tenant id and topic
func (hf *HandlerFactory) GetTopicHandlersForTenant(tenantId string, topic string) []*HandlerConfiguration {
	hls := make([]*HandlerConfiguration, 0)
	if _, ok := hf.TopicHandlerMap[tenantId]; !ok {
		return hls
	}
	handlers, terminate := hf.getPartialTopicHandlersForTenant(tenantId, topic)
	hls = append(hls, handlers...)
	if terminate {
		return hls
	}
	segments := strings.Split(topic, "/")
	for i := len(segments) - 1; i > 0; i-- {
		parentTopic := ""
		for k := 0; k < i-1; k++ {
			parentTopic += segments[k] + "/"
		}
		parentTopic += segments[i-1]
		th, terminate := hf.getPartialTopicHandlersForTenant(tenantId, parentTopic)
		hls = append(hls, th...)
		if terminate {
			return hls
		}
	}
	// if nothing else, try if there is a default topic handler for this tenant
	if len(hls) == 0 {
		tenantTopicHandlers := hf.TopicHandlerMap[tenantId]
		if handlers, ok := tenantTopicHandlers[""]; ok {
			hls = append(hls, handlers...)
		}
	}
	return hls
}

func (h *HandlerConfiguration) matchesChoiceOfValues(ctx Context, event *JDoc, matchMap map[string]interface{}) (bool, int) {
	numMatches := 0
	for path, expectedValue := range matchMap {
		actualVal := event.ParseExpression(ctx, path)
		expectedVal := event.ParseExpression(ctx, expectedValue)
		switch actualVal.(type) {
		// support a choice of values (think of an array as a set): if any of the actual array elements matches any of the expected array elements we have a match
		// also a single flat element can match any elment of an array (both ways, in actual as well as expected value) - we may revisit this behavior in the future
		case []interface{}:
			switch expectedVal.(type) {
			case []interface{}:
				priorMatches := numMatches
				for _, e := range expectedVal.([]interface{}) {
					for _, a := range actualVal.([]interface{}) {
						if DeepEquals(e, a) {
							numMatches++
						}
					}
				}
				// must at least match one element of the choices
				if priorMatches == numMatches {
					return false, numMatches
				}
			default:
				priorMatches := numMatches
				for _, a := range actualVal.([]interface{}) {
					if DeepEquals(expectedVal, a) {
						numMatches++
					}
				}
				// must at least match one element of the choices
				if priorMatches == numMatches {
					return false, numMatches
				}
			}
		default:
			switch expectedVal.(type) {
			case []interface{}:
				priorMatches := numMatches
				for _, e := range expectedVal.([]interface{}) {
					if DeepEquals(actualVal, e) {
						numMatches++
					}
				}
				// must at least match one element of the choices
				if priorMatches == numMatches {
					return false, numMatches
				}
			default:
				if DeepEquals(actualVal, expectedVal) {
					numMatches++
				} else {
					return false, numMatches
				}
			}
		}
	}
	return true, numMatches
}

// matchesExpectedValues returns if matches plus match strength (number of matches)
func (h *HandlerConfiguration) matchesExpectedValues(ctx Context, event *JDoc) (bool, int) {
	if h.Match != nil {
		if h.IsMatchByExample {
			return event.MatchesPattern(h.m)
		} else {
			return h.matchesChoiceOfValues(ctx, event, h.Match)
		}
	}
	return true, 0
}

// matchesExpectedValues returns true if event matches filter
func (h *HandlerConfiguration) matchesFilter(ctx Context, event *JDoc) bool {
	if h.f != nil {
		if h.IsFilterByExample {
			matches, _ := event.MatchesPattern(h.f)
			if h.IsFilterInverted {
				return !matches
			} else {
				return matches
			}
		} else {
			matches, _ := h.matchesChoiceOfValues(ctx, event, h.Filter)
			if h.IsFilterInverted {
				return !matches
			} else {
				return matches
			}
		}
	}
	return true
}

// GetHandlersForEvent obtains matching list of handlers for a given JSON document.
func (hf *HandlerFactory) GetHandlersForEvent(ctx Context, event *JDoc) []*HandlerConfiguration {
	hls := make([]*HandlerConfiguration, 0)
	// check for topic handlers
	topic := event.GetStringValueForExpression(ctx, GetConfig(ctx).TopicPath)
	preferredTenantId := GetTenantId(ctx)
	if preferredTenantId != "" {
		handlers := hf.GetTopicHandlersForTenant(preferredTenantId, topic)
		hls = append(hls, handlers...)
	} else {
		for tenantId, _ := range hf.TopicHandlerMap {
			handlers := hf.GetTopicHandlersForTenant(tenantId, topic)
			hls = append(hls, handlers...)
		}
	}
	// check custom match handlers
	if preferredTenantId != "" {
		hmil := make([]*handlerMatchInstance, 0)
		for _, handler := range hf.CustomHandlerMap[preferredTenantId] {
			if matches, strength := handler.matchesExpectedValues(ctx, event); matches {
				hmil = append(hmil, newHandlerMatchInstance(handler, strength))
			}
		}
		// only pick top matches until we encounter terminate on match
		sort.Sort(handlerMatchInstanceList(hmil))
		for _, hmi := range hmil {
			hls = append(hls, hmi.handler)
			if hmi.handler.TerminateOnMatch {
				break
			}
		}
	} else {
		for tenantId, _ := range hf.CustomHandlerMap {
			hmil := make([]*handlerMatchInstance, 0)
			for _, handler := range hf.CustomHandlerMap[tenantId] {
				if matches, strength := handler.matchesExpectedValues(ctx, event); matches {
					hmil = append(hmil, newHandlerMatchInstance(handler, strength))
				}
			}
			// only pick top matches until we encounter terminate on match
			sort.Sort(handlerMatchInstanceList(hmil))
			for _, hmi := range hmil {
				hls = append(hls, hmi.handler)
				if hmi.handler.TerminateOnMatch {
					break
				}
			}
		}
	}
	return hls
}

// GetHandlersByName obtains handler by name
func (hf *HandlerFactory) GetHandlerByName(ctx Context, tenant string, name string) *HandlerConfiguration {
	for _, m1 := range hf.TopicHandlerMap {
		for _, m2 := range m1 {
			for _, handler := range m2 {
				if handler.Name == name && handler.TenantId == tenant {
					return handler
				}
			}
		}
	}
	for tenantId, _ := range hf.CustomHandlerMap {
		if tenantId == tenant {
			for _, handler := range hf.CustomHandlerMap[tenantId] {
				if handler.Name == name {
					return handler
				}
			}
		}
	}
	return nil
}

// GetAllHandlers obtains list of all handlers.
func (hf *HandlerFactory) GetAllHandlers(ctx Context) []*HandlerConfiguration {
	hls := make([]*HandlerConfiguration, 0)
	for _, m1 := range hf.TopicHandlerMap {
		for _, m2 := range m1 {
			for _, handler := range m2 {
				hls = append(hls, handler)
			}
		}
	}
	for tenantId, _ := range hf.CustomHandlerMap {
		for _, handler := range hf.CustomHandlerMap[tenantId] {
			hls = append(hls, handler)
		}
	}
	return hls
}

// IsValidTransformation helper function to check if a transformation given as parameter is valid.
func (h *HandlerConfiguration) IsValidTransformation(ctx Context, t *JDoc, istbe bool) (bool, string) {
	if t == nil {
		return false, "missing transformation"
	}
	if !istbe {
		o := t.GetOriginalObject()
		switch o.(type) {
		case map[string]interface{}:
			for k, v := range o.(map[string]interface{}) {
				if reflect.TypeOf(v).Kind() == reflect.String {
					if !t.IsValidPathExpression(ctx, v.(string)) {
						return false, v.(string)
					}
				}
				if !t.IsValidPathExpression(ctx, k) {
					return false, k
				}
			}
		default:
			return false, "transformation not a map"
		}
	} else {
		//TODO: validate transformation by example
	}
	return true, ""
}

func (hf *HandlerFactory) getAllConfigurationFiles(ctx Context, configFolder string) []string {
	if configFolder == "" {
		configFolder = DefaultConfigFolder
	}
	fileList := []string{}
	err := filepath.Walk(configFolder, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".json") {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		ctx.Log().Error("error_type", "load_handler", "cause", "error_exploring_topic_handler_config_files", "folder", configFolder)
	}
	return fileList
}

// GetHandlerConfigurationFromFile loads a single handler config from disk and returns a handler and a (hopefully empty) list of warning strings.
func (hf *HandlerFactory) GetHandlerConfigurationFromFile(ctx Context, filepath string) (*HandlerConfiguration, []error) {
	warnings := make([]error, 0)
	file, err := os.Open(filepath)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		ctx.Log().Error("error_type", "load_handler", "cause", "error_loading_config_file", "file", filepath)
		warnings = append(warnings, errors.New("error opening config file "+filepath))
		return nil, warnings
	}
	config, err := ioutil.ReadAll(file)
	if err != nil {
		ctx.Log().Error("error_type", "load_handler", "cause", "error_reading_config", "file", filepath)
		warnings = append(warnings, errors.New("error reading config file "+filepath))
		return nil, warnings
	}
	var handler HandlerConfiguration
	err = json.Unmarshal(config, &handler)
	if err != nil {
		ctx.Log().Error("error_type", "load_handler", "cause", "error_parsing_config", "file", filepath)
		warnings = append(warnings, ParseError{"invalid json in config file " + filepath})
		return nil, warnings
	}
	handler.File = filepath
	return hf.GetHandlerConfigurationFromJson(ctx, filepath, handler)
}

// GetHandlerConfigurationFromJson parses and vets handler configuration from a partially populated HandlerConfiguration struct.
func (hf *HandlerFactory) GetHandlerConfigurationFromJson(ctx Context, filepath string, handler HandlerConfiguration) (*HandlerConfiguration, []error) {
	var err error
	warnings := make([]error, 0)
	ctx.Log().Info("action", "loading_handler", "file", filepath)
	segments := strings.Split(filepath, "/")
	if len(segments) >= 2 {
		handler.TenantId = segments[len(segments)-2]
	} else {
		handler.TenantId = "unknown"
		//ctx.Log().Error("action", "missing_tenant_id", "file", filepath, "name", handler.Name)
		//warnings = append(warnings, ParseError{"missing tenant id in config file " + filepath})
	}
	if filepath != "" && !strings.Contains(filepath, "/"+handler.TenantId+"/") {
		ctx.Log().Error("error_type", "load_handler", "cause", "handler_has_bad_location", "file", filepath, "name", handler.Name, "tenant", handler.TenantId)
		warnings = append(warnings, ParseError{"bad location of config file " + filepath})
	}
	if handler.Name == "" {
		ctx.Log().Error("error_type", "load_handler", "cause", "blank_name", "file", filepath, "name", handler.Name, "tenant", handler.TenantId)
		warnings = append(warnings, ParseError{"blank name in config file " + filepath})
	}
	if handler.Version == "" {
		ctx.Log().Error("error_type", "load_handler", "cause", "missing_version", "file", filepath, "name", handler.Name, "tenant", handler.TenantId)
		warnings = append(warnings, ParseError{"missing version config file " + filepath})
	}
	if handler.Transformation != nil {
		handler.t, err = NewJDocFromInterface(handler.Transformation)
		if err != nil {
			ctx.Log().Error("error_type", "load_handler", "cause", "invalid_transformation", "file", filepath, "name", handler.Name, "tenant", handler.TenantId, "error", err.Error())
			warnings = append(warnings, ParseError{"non json transformation in config file " + filepath})
		}
		valid, reason := handler.IsValidTransformation(ctx, handler.t, handler.IsTransformationByExample)
		if !valid {
			ctx.Log().Error("error_type", "load_handler", "cause", "invalid_transformation", "file", filepath, "reason", reason, "name", handler.Name, "tenant", handler.TenantId)
			warnings = append(warnings, ParseError{"invalid transformation in config file " + filepath + ": " + reason})
		}
	}
	if handler.Transformations != nil {
		for k, v := range handler.Transformations {
			tf, err := NewJDocFromInterface(v.Transformation)
			if err != nil {
				ctx.Log().Error("error_type", "load_handler", "cause", "invalid_transformation", "file", filepath, "name", handler.Name, "tenant", handler.TenantId, "transformation_name", k, "error", err.Error())
				warnings = append(warnings, ParseError{"non json transformation " + k + " in config file " + filepath})
			}
			v.SetTransformation(tf)
			valid, reason := handler.IsValidTransformation(ctx, v.GetTransformation(), v.IsTransformationByExample)
			if !valid {
				ctx.Log().Error("error_type", "load_handler", "cause", "invalid_transformation", "file", filepath, "reason", reason, "name", handler.Name, "tenant", handler.TenantId)
				warnings = append(warnings, ParseError{"invalid transformation " + k + " in config file " + filepath + ": " + reason})
			}
		}
	}
	if handler.Match != nil {
		handler.m, err = NewJDocFromMap(handler.Match)
		if err != nil {
			ctx.Log().Error("error_type", "load_handler", "cause", "invalid_match", "file", filepath, "name", handler.Name, "tenant", handler.TenantId)
			warnings = append(warnings, ParseError{"non json match in config file " + filepath})
		}
		valid, invalidPath := handler.IsValidTransformation(ctx, handler.m, handler.IsMatchByExample)
		if !valid {
			ctx.Log().Error("error_type", "load_handler", "cause", "invalid_match", "file", filepath, "path", invalidPath, "name", handler.Name, "tenant", handler.TenantId)
			warnings = append(warnings, ParseError{"invalid match in config file " + filepath})
		}
	}
	if handler.Filter != nil {
		handler.f, err = NewJDocFromMap(handler.Filter)
		if err != nil {
			ctx.Log().Error("error_type", "load_handler", "cause", "invalid_filter", "file", filepath, "name", handler.Name, "tenant", handler.TenantId)
			warnings = append(warnings, ParseError{"non json filter in config file " + filepath})
		}
	}
	if handler.Filters != nil {
		for _, f := range handler.Filters {
			f.f, err = NewJDocFromMap(f.Filter)
			if err != nil {
				ctx.Log().Error("error_type", "load_handler", "cause", "invalid_filter", "file", filepath, "name", handler.Name, "tenant", handler.TenantId, "filer", f.Filter)
				warnings = append(warnings, ParseError{"non json filter in config file " + filepath})
			}
		}
	}
	// default to http protocol if none other specified
	if handler.Protocol == "" {
		handler.Protocol = "http"
	}
	// default to POST verb if none other specified
	if handler.Verb == "" {
		handler.Verb = "POST"
	}
	// default endpoint is localhost
	/*if handler.Endpoint == "" {
		handler.Endpoint = "http://localhost/"
	}*/
	return &handler, warnings
}

// for main filter and the two boolean filters (deprecated but still in use)
func (h *HandlerConfiguration) filterEvent(ctx Context, event *JDoc) bool {
	if h.FilterIfTrue != "" {
		f := event.ParseExpression(ctx, h.FilterIfTrue)
		switch f.(type) {
		case bool:
			if f.(bool) == true {
				return true
			}
		}
	}
	if h.FilterIfFalse != "" {
		f := event.ParseExpression(ctx, h.FilterIfFalse)
		switch f.(type) {
		case bool:
			if f.(bool) == false {
				return true
			}
		}
	}
	if h.Filter != nil && h.matchesFilter(ctx, event) {
		return true
	}
	return false
}

func (h *HandlerConfiguration) logFilter(ctx Context, event *JDoc, f *Filter) {
	ctx = ctx.SubContext()
	if f.LogParams != nil {
		for k, v := range f.LogParams {
			ev := event.ParseExpression(ctx, v)
			ctx.AddLogValue(k, ev)
		}
	}
	ctx.Log().Info("action", "filtered_event", "tenant", h.TenantId, "handler", h.Name)
}

func (h *HandlerConfiguration) applyDebugLogsIfWhiteListed(ctx Context, event *JDoc, wl *JDoc) bool {
	debug := false
	dlp := GetDebugLogParams(ctx)
	if dlp == nil {
		//ctx.Log().Info("action", "debug_no_white_list")
		return false
	}
	if dlp.IdWhiteList == nil || dlp.LogParams == nil {
		//ctx.Log().Info("action", "debug_no_white_list")
		return false
	}
	wlistId := wl.ParseExpression(ctx, dlp.IdPath)
	if wlistId == nil {
		//ctx.Log().Info("action", "debug_no_location")
		return false
	}
	switch wlistId.(type) {
	case string:
	default:
		return false
	}
	c := ctx.SubContext()
	dlp.Lock.RLock()
	defer dlp.Lock.RUnlock()
	if _, ok := dlp.IdWhiteList[wlistId.(string)]; ok {
		debug = true
		for k, v := range dlp.LogParams {
			ev := event.ParseExpression(ctx, v)
			c.AddLogValue(k, ev)
		}
		c.Log().Info("action", "debug_event", "handler", h.Name)
	} else {
		//ctx.Log().Info("action", "location_not_in_white_list", "location", wlistId)
	}
	return debug
}

// new version allowing a set of filters presented as array
func (h *HandlerConfiguration) applyFilters(ctx Context, event *JDoc, after bool) bool {
	if h.Filters != nil {
		for _, f := range h.Filters {
			if f.FilterAfterTransformation == after {
				if f.IsFilterByExample {
					matches, _ := event.MatchesPattern(f.f)
					if !f.IsFilterInverted && matches {
						h.logFilter(ctx, event, f)
						return true
					} else if f.IsFilterInverted && !matches {
						h.logFilter(ctx, event, f)
						return true
					}
				} else {
					matches, _ := h.matchesChoiceOfValues(ctx, event, f.Filter)
					if !f.IsFilterInverted && matches {
						h.logFilter(ctx, event, f)
						return true
					} else if f.IsFilterInverted && !matches {
						h.logFilter(ctx, event, f)
						return true
					}
				}
			}
		}
	}
	return false
}

// ProcessEvent lets a single handler process an event and returns a list of prepopulates publishers (typically one publisher).
func (h *HandlerConfiguration) ProcessEvent(ctx Context, event *JDoc) ([]EventPublisher, error) {
	if event == nil {
		return make([]EventPublisher, 0), errors.New("no event")
	}
	/*if h.t == nil {
		return make([]EventPublisher, 0), errors.New("no transformation")
	}*/
	if h.Protocol == "" {
		return make([]EventPublisher, 0), errors.New("no protocol")
	}
	ctx.AddValue(EelHandlerConfig, h)
	// filtering
	// must apply filters BEFORE evaluating custom properties in case custom properties include recursive curl!!!
	if !h.FilterAfterTransformation {
		if h.filterEvent(ctx, event) {
			return make([]EventPublisher, 0), nil
		}
	}
	if h.applyFilters(ctx, event, false) {
		return make([]EventPublisher, 0), nil
	}
	// custom properties
	if h.CustomProperties != nil {
		cp := make(map[string]interface{}, 0)
		for k, v := range h.CustomProperties {
			cp[k] = event.ParseExpression(ctx, v)
		}
		ctx.AddValue(EelCustomProperties, cp)
	}
	// apply debug logs
	debug := h.applyDebugLogsIfWhiteListed(ctx, event, event)
	if ctx.ConfigValue(EelTraceLogger) != nil {
		ctx.ConfigValue(EelTraceLogger).(*TraceLogger).TraceLog(ctx, event, true)
	}
	// prepare headers
	headers := make(map[string]string, 0)
	if h.HttpHeaders != nil {
		for hk, jexpr := range h.HttpHeaders {
			headers[hk] = ToFlatString(event.ParseExpression(ctx, jexpr))
		}
	}
	debugHeaderKey := GetConfig(ctx).HttpDebugHeader
	if debug && debugHeaderKey != "" {
		headers[debugHeaderKey] = "true"
	}
	// prepare relative paths
	relativePaths := make([]string, 0)
	if h.Path != nil {
		switch h.Path.(type) {
		case []interface{}:
			for _, exp := range h.Path.([]interface{}) {
				switch exp.(type) {
				case string:
					relativePaths = append(relativePaths, event.GetStringValueForExpression(ctx, exp.(string)))
				}
			}
		case string:
			relativePaths = append(relativePaths, event.GetStringValueForExpression(ctx, h.Path.(string)))
		}
	}
	if len(relativePaths) == 0 {
		relativePaths = append(relativePaths, "")
	}
	// prepare endpoint
	endpoints := make([]string, 0)
	if h.Endpoint != nil {
		switch h.Endpoint.(type) {
		case []interface{}:
			for _, exp := range h.Endpoint.([]interface{}) {
				switch exp.(type) {
				case string:
					ep := event.GetStringValueForExpression(ctx, exp.(string))
					if ep != "" {
						endpoints = append(endpoints, ep)
					}
				}
			}
		case string:
			ep := event.GetStringValueForExpression(ctx, h.Endpoint.(string))
			if ep != "" {
				endpoints = append(endpoints, ep)
			}
		}
	}
	if len(endpoints) == 0 && GetConfig(ctx).Endpoint != nil {
		switch GetConfig(ctx).Endpoint.(type) {
		case []interface{}:
			for _, exp := range GetConfig(ctx).Endpoint.([]interface{}) {
				switch exp.(type) {
				case string:
					ep := event.GetStringValueForExpression(ctx, exp.(string))
					if ep != "" {
						endpoints = append(endpoints, ep)
					}
				}
			}
		case string:
			ep := event.GetStringValueForExpression(ctx, GetConfig(ctx).Endpoint.(string))
			if ep != "" {
				endpoints = append(endpoints, ep)
			}
		}
	}
	/*if len(endpoints) == 0 {
		return make([]EventPublisher, 0), errors.New("no endpoints")
	}*/
	// prepare payload
	payload := ""
	var tfd *JDoc
	if h.Transformation != nil {
		if h.IsTransformationByExample {
			tfd = event.ApplyTransformationByExample(ctx, h.t)
		} else {
			tfd = event.ApplyTransformation(ctx, h.t)
		}
		if tfd == nil {
			ctx.Log().Error("error_type", "process_event", "cause", "bad_transformation", "event", event.String(), "transformation", h.t.String(), "handler", h.Name)
			return make([]EventPublisher, 0), errors.New("bad transformation")
		}
		payload = tfd.StringPretty()
		// apply debug logs
		h.applyDebugLogsIfWhiteListed(ctx, tfd, event)
		if ctx.ConfigValue(EelTraceLogger) != nil {
			ctx.ConfigValue(EelTraceLogger).(*TraceLogger).TraceLog(ctx, event, false)
		}
	}
	// filtering
	if h.FilterAfterTransformation {
		if h.filterEvent(ctx, tfd) {
			return make([]EventPublisher, 0), nil
		}
	}
	if h.applyFilters(ctx, tfd, true) {
		return make([]EventPublisher, 0), nil
	}
	// prepare publisher(s)
	publishers := make([]EventPublisher, 0)
	for _, ep := range endpoints {
		for _, rp := range relativePaths {
			publisher := NewEventPublisher(ctx.SubContext(), h.Protocol)
			if publisher == nil {
				ctx.Log().Error("error_type", "process_event", "cause", "unsupported_protocol", "protocol", h.Protocol, "event", event.String(), "handler", h.Name)
				continue
			}
			publisher.SetEndpoint(ep)
			if h.AuthInfo != nil {
				publisher.SetAuthInfo(h.AuthInfo)
			}
			publisher.SetHeaders(headers)
			if h.Verb != "" {
				publisher.SetVerb(h.Verb)
			}
			publisher.SetPayload(payload)
			publisher.SetPath(rp)
			publisher.SetPayloadParsed(tfd)
			publisher.SetDebug(debug)
			publishers = append(publishers, publisher)
		}
	}
	return publishers, nil
}

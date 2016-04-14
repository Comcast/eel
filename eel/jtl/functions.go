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
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robertkrimen/otto"

	. "code.comcast.com/XfinityRulesService/csv-rules-eel/eel/util"
)

type (
	// JFunction represents built-in functions that can be used in jpath expressions.
	JFunction struct {
		fn           func(ctx Context, doc *JDoc, params []string) interface{}
		minNumParams int
		maxNumParams int
	}
)

// NewFunction gets function implementation by name.
func NewFunction(fn string) *JFunction {
	//stats := gctx.Value(EelTotalStats).(*ServiceStats)
	switch fn {
	case "curl":
		// hit external web service
		// jpath expressions of the form {{/...}} will be evaluated prior to function execution
		// selector - optional jpath to be applied to (json) response
		// payload - to be sent to external mapping service
		// url - url of external mapping service
		// method - POST, GET etc.
		// curl('<method>','<url>','<payload>','<selector>',['<header-map>'])
		// curl('POST', 'http://foo.com/bar/json', 'foo-{{/content/bar}}', '/content/comcastId')
		// example: {{curl('POST', 'http://foo.com/bar/json', '{{/content/bar}}', '/content/comcastId')}}
		return &JFunction{fnCurl, 2, 5}
	case "uuid":
		// returns UUID string
		// uuid()
		return &JFunction{fnUuid, 0, 0}
	case "header":
		// returns a value given the http request header key
		// header('mykey')
		return &JFunction{fnHeader, 1, 1}
	case "ident":
		// returns input parameter unchanged, for debugging only
		// ident('foo')
		return &JFunction{fnIdent, 1, 1}
	case "upper":
		// upper case input string, example upper('foo')
		return &JFunction{fnUpper, 1, 1}
	case "lower":
		// lower case input string, example lower('foo')
		return &JFunction{fnLower, 1, 1}
	case "substr":
		// substring by start and end index, example substr('foo', 0, 1)
		return &JFunction{fnSubstr, 3, 3}
	case "eval":
		// evaluates simple path expression on current document and returns result
		return &JFunction{fnEval, 1, 2}
	case "prop":
		// return property from CustomProperties section in config.json
		return &JFunction{fnProp, 1, 1}
	case "js":
		// execute arbitrary javascript and return result
		return &JFunction{fnJs, 1, 100}
	case "alt":
		// return first non blank parameter (alternative)
		return &JFunction{fnAlt, 2, 100}
	case "case":
		// simplification of nested ifte(equals(),'foo', ifte(equals(...),...)) cascade
		// case('<path_1>','<comparison_value_1>','<return_value_1>', '<path_2>','<comparison_value_2>','<return_value_2>,...,'<default>')
		return &JFunction{fnCase, 3, 100}
	case "regex":
		// apply regex to string value: regex('<string>', '<regex>')
		return &JFunction{fnRegex, 2, 2}
	case "and":
		// boolean and: and('<bool>', '<bool>', ...)
		return &JFunction{fnAnd, 1, 100}
	case "or":
		// boolean or: or('<bool>', '<bool>', ...)
		return &JFunction{fnOr, 1, 100}
	case "not":
		// boolean not: not('<bool>')
		return &JFunction{fnNot, 1, 1}
	case "contains":
		// checks if document contains another document: contains('<doc1>', ['<doc2>'])
		return &JFunction{fnContains, 1, 2}
	case "equals":
		// checks if document is equal to another json document or if two strings are equal: equals('<doc1>',['<doc2>'])
		return &JFunction{fnEquals, 1, 2}
	case "join":
		// merges two json documents into one, key conflicts are resolved at random
		return &JFunction{fnJoin, 2, 2}
	case "format":
		// formats time string: format('<ms>',['<layout>'],['<timezone>']), example: format('1439962298000','Mon Jan 2 15:04:05 2006','PST')
		return &JFunction{fnFormat, 1, 3}
	case "ifte":
		// if condition then this else that: ifte('<condition>','<then>',['<else>']), example: ifte('{{equals('{{/data/name}}','')}}','','by {{/data/name}}')
		return &JFunction{fnIfte, 1, 3}
	case "transform":
		// apply transformation: transform('<name_of_transformation>', '<doc>', ['<pattern>'], ['<join>']), example: transform('my_transformation', '{{/content}}')
		// - the transformation is selected by name from an optional transformation map in the handler config
		// - if the document is an array, the transformation will be iteratively applied to all array elements
		// - if a pattern is provided will only be applied if document is matching the pattern
		// - if a join is provided it will be joined with the document before applying the transformation
		return &JFunction{fnTransform, 1, 4}
	case "itransform":
		// apply transformation iteratively: transform('<name_of_transformation>', '<doc>', ['<pattern>'], ['<join>']), example: transform('my_transformation', '{{/content}}')
		// - the transformation is selected by name from an optional transformation map in the handler config
		// - if the document is an array, the transformation will be iteratively applied to all array elements
		// - if a pattern is provided will only be applied if document is matching the pattern
		// - if a join is provided it will be joined with the document before applying the transformation
		return &JFunction{fnITransform, 1, 4}
	case "true":
		// returns always true, shorthand for equals('1', '1')
		return &JFunction{fnTrue, 0, 0}
	case "false":
		// returns always false, shorthand for equals('1', '2')
		return &JFunction{fnFalse, 0, 0}
	case "time":
		// returns current time as timestamp
		return &JFunction{fnTime, 0, 0}
	case "tenant":
		// returns tenant of current handler
		return &JFunction{fnTenant, 0, 0}
	case "traceid":
		// returns current trace id used for logging
		return &JFunction{fnTraceId, 0, 0}
	case "choose":
		// chooses elements for list or array based on pattern
		return &JFunction{fnChoose, 2, 2}
	case "crush":
		// experimental function for collapsing a JSON document into a flat array
		return &JFunction{fnCrush, 1, 1}
	default:
		//gctx.Log.Error("event", "execute_function", "function", fn, "error", "not_implemented")
		//stats.IncErrors()
		return nil
	}
}

// fnRegex regular expression function.
func fnRegex(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "regex", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	reg, err := regexp.Compile(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "regex", "error", "invalid_regex", "params", params)
		stats.IncErrors()
		return nil

	}
	return reg.FindString(extractStringParam(params[0]))
}

// fnAlt alternative function.
func fnAlt(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 {
		ctx.Log().Error("event", "execute_function", "function", "alt", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	for _, p := range params {
		if sp := extractStringParam(p); sp != "" {
			return sp
		}
	}
	return ""
}

// fnAnd boolean and function.
func fnAnd(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "and", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	result := true
	for i := 0; i < len(params); i++ {
		b, err := strconv.ParseBool(extractStringParam(params[i]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "and", "error", "non_boolean_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
		result = result && b
	}
	return result
}

// fnOr boolean or function.
func fnOr(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "or", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	result := false
	for i := 0; i < len(params); i++ {
		b, err := strconv.ParseBool(extractStringParam(params[i]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "or", "error", "non_boolean_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
		result = result || b
	}
	return result
}

// fnNot boolean not function.
func fnNot(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "not", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	result, err := strconv.ParseBool(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "not", "error", "non_boolean_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	return !result
}

// fnContains contains function.
func fnContains(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "contains", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	if len(params) == 2 {
		var err error
		doc, err = NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "contains", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
	}
	containee, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "contains", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	matches, _ := doc.MatchesPattern(containee)
	return matches
}

// fnChoose choose function.
func fnChoose(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	mydoc, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	pattern, err := NewJDocFromString(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	choice := doc.choose(mydoc.GetOriginalObject(), pattern.GetOriginalObject())
	return choice
}

// fnCrush crush function.
func fnCrush(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "crush", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	mydoc, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "crush", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	crush := doc.crush(mydoc.GetOriginalObject(), nil)
	return crush
}

// fnEquals function. Performs deep equals on two JSON documents, otherwise a simple string comparison will be done.
func fnEquals(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "equals", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	pattern, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil && len(params) == 1 {
		ctx.Log().Error("event", "execute_function", "function", "equals", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	if len(params) == 2 {
		doc, err = NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			// if not json, just do string comparison (only makes sense for the 2-param version, otherwise must be json)
			return extractStringParam(params[0]) == extractStringParam(params[1])
		}
	}
	return doc.Equals(pattern)
}

// fnIfte is an if-then-else function. The first parameter must evaluate to a boolean, the third parameter is optional.
func fnIfte(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 || len(params) > 3 {
		ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	conditionMet, err := strconv.ParseBool(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "non_boolean_parameter", "params", params)
		stats.IncErrors()
		return nil
	}
	//TODO: preserve parameter type (bool, string, json, int, float), also applies to alt() etc.
	res := ""
	if conditionMet {
		res = extractStringParam(params[1])
	} else if len(params) == 3 {
		res = extractStringParam(params[2])
	}
	// hack to resurrect json
	if strings.Contains(res, "{") && strings.Contains(res, "}") {
		doc, err := NewJDocFromString(res)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "invalid_json", "params", params)
			stats.IncErrors()
			return nil

		}
		return doc.GetOriginalObject()
	}
	return res
}

// fnCase is a simplification of a nested ifte(equals(),'foo', ifte(equals(...),...)) cascade.
// Example: case('<path_1>','<comparison_value_1>','<return_value_1>', '<path_2>','<comparison_value_2>','<return_value_2>,...,'<default>')
func fnCase(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 3 || len(params)%3 != 1 {
		ctx.Log().Error("event", "execute_function", "function", "case", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	for i := 0; i < len(params)/3; i++ {
		if extractStringParam(params[i*3]) == extractStringParam(params[i*3+1]) {
			return extractStringParam(params[i*3+2])
		}
	}
	return extractStringParam(params[len(params)-1])
}

// fnJs JavaScript function. Kind of useful for everything that does not have a built-in function.
func fnJs(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "js", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	vm := otto.New()
	for i := 2; i < len(params)-1; i += 2 {
		vm.Set(extractStringParam(params[i]), extractStringParam(params[i+1]))
	}
	//ctx.Log.Info("run", extractStringParam(params[0]))
	value, err := vm.Run(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "js", "error", "vm_error", "params", params, "detail", err.Error())
		stats.IncErrors()
		return nil
	}
	if len(params) > 1 {
		//ctx.Log.Info("get", extractStringParam(params[1]))
		value, err = vm.Get(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "js", "error", "vm_val_error", "params", params, "detail", err.Error())
			stats.IncErrors()
			return nil
		}
	}
	var ret interface{}
	if value.IsString() {
		ret, err = value.ToString()
	} else if value.IsNumber() {
		var i64ret int64
		i64ret, err = value.ToInteger()
		ret = int(i64ret)
	} else if value.IsBoolean() {
		ret, err = value.ToBoolean()
	} else {
		ret = value.String()
	}
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "js", "error", "vm_val_conv_error", "params", params, "detail", err.Error())
		stats.IncErrors()
		return nil
	}
	return ret
}

// fnCurl provides curl-like functionality to reach out to helper web services. This function usually has grave performance consequences.
func fnCurl(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 || len(params) > 5 {
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return nil
	}
	url := extractStringParam(params[1])
	if ctx.ConfigValue("debug.url") != nil {
		url = ctx.ConfigValue("debug.url").(string)
	}
	// compose http headers: at a minimum use trace header (if available), then add extra headers (if given in param #5)
	hmap := make(map[string]interface{})
	if len(params) == 5 {
		hdoc, err := NewJDocFromString(extractStringParam(params[4]))
		if err != nil {
			stats.IncErrors()
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "invalid_headers", "detail", err.Error(), "params", params)
		} else {
			hmap = hdoc.GetMapValue("/")
		}
	}
	headers := make(map[string]string, 0)
	if hmap != nil {
		for k, v := range hmap {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}
	body := ""
	if len(params) >= 3 {
		body = extractStringParam(params[2])
	}
	ctx.AddLogValue("destination", "external_service")
	resp, status, err := HitEndpoint(ctx, url, body, extractStringParam(params[0]), headers, nil)
	if err != nil {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "unexpected_response", "status", strconv.Itoa(status), "detail", err.Error(), "response", resp, "params", params)
		return nil
	}
	if status < 200 || status >= 300 {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "unexpected_response", "status", strconv.Itoa(status), "response", resp, "params", params)
		return nil
	}
	selector := ""
	if len(params) >= 4 {
		selector = extractStringParam(params[3])
	}
	if selector != "" {
		j, _ := NewJDocFromString("")
		if !j.IsValidPath(selector) {
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "invalid_path_selector", "params", params, "selector", selector, "params", params)
			stats.IncErrors()
			return nil
		}
		jresp, err := NewJDocFromString(resp)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "no_json_response", "params", params, "detail", err.Error(), "response", resp)
			stats.IncErrors()
			return nil
		}
		result := jresp.EvalPath(ctx, selector)
		if result == nil {
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "selector_yields_blank_response", "params", params, "response", resp, "selector", selector)
			stats.IncErrors()
			return nil
		}
		return result
	} else {
		var res interface{}
		err := json.Unmarshal([]byte(resp), &res)
		if err != nil {
			return resp
		} else {
			return res
		}
	}
}

// fnmHeader function to obtain http header value from incoming event by key.
func fnHeader(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "header", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	key := extractStringParam(params[0])
	header := ctx.Value(EelRequestHeader)
	if header == nil {
		ctx.Log().Info("event", "execute_function", "function", "header", "message", "header_object_not_found")
		return ""
	}
	h, ok := header.(http.Header)
	if !ok {
		ctx.Log().Info("event", "execute_function", "function", "header", "message", "header_object_not_valid")
		return ""
	}
	return h.Get(key)
}

// fnUuid return a new uuid.
func fnUuid(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "uuid", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		return ""
	}
	uuid, err := NewUUID()
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "uuid", "error", "uuid_generator", "params", params)
		stats.IncErrors()
		return ""
	}
	return uuid
}

// fnTraceId returns current tarce id used for logging.
func fnTraceId(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "traceid", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		return ""
	}
	return ctx.LogValue("tx.traceId")
}

// fnTime return current time in nano-seconds.
func fnTime(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "time", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		return ""
	}
	return time.Now().UnixNano()
}

// fnIdent is a function that does nothing. Somtimes interesting for debugging.
func fnIdent(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "ident", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	return extractStringParam(params[0])
}

// fnUpper function to uppercase a string.
func fnUpper(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "upper", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	return strings.ToUpper(extractStringParam(params[0]))
}

// fnLower function to lowercase a string.
func fnLower(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "lower", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	return strings.ToLower(extractStringParam(params[0]))
}

// fnSubstr function to lowercase a string.
func fnSubstr(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 3 {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	i, err := strconv.Atoi(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "param_1_not_int", "params", params)
		stats.IncErrors()
		return ""
	}
	j, err := strconv.Atoi(extractStringParam(params[2]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "param_2_not_int", "params", params)
		stats.IncErrors()
		return ""
	}
	return extractStringParam(params[0])[i:j]
}

// fnEval function to exaluate a jpath expression agains current document or against a document passed in as parameter. Often useful in combination with fnCurl.
func fnEval(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil {
		ctx.Log().Error("event", "execute_function", "function", "eval", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	} else if len(params) == 1 {
		return doc.EvalPath(ctx, extractStringParam(params[0]))
	} else if len(params) == 2 {
		ldoc, err := NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "eval", "error", "json_expected", "params", params)
			stats.IncErrors()
			return ""
		}
		return ldoc.EvalPath(ctx, extractStringParam(params[0]))
	} else {
		ctx.Log().Error("event", "execute_function", "function", "eval", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
}

// fnJoin functions joins two JSON documents given as parameters and returns results.
func fnJoin(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	docA, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "json_expected_p0", "params", params)
		stats.IncErrors()
		return ""
	}
	docB, err := NewJDocFromString(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "json_expected_p1", "params", params)
		stats.IncErrors()
		return ""
	}
	var section interface{}
	section = docA.GetOriginalObject()
	switch section.(type) {
	// not sure any more if we really want an iterative join
	// apply sub-transformation iteratively to all array elements
	//case []interface{}:
	//	for i, a := range section.([]interface{}) {
	//		littleDoc, err := NewJDocFromInterface(a)
	//		if err != nil {
	//			ctx.Log().Error("event", "execute_function", "function", "join", "error", err.Error(), "params", params)
	//			stats.IncErrors()
	//			return ""
	//		}
	//		m := docA.merge(littleDoc.GetOriginalObject(), docB.GetOriginalObject())
	//		if m == nil {
	//			ctx.Log().Error("event", "execute_function", "function", "join", "error", "merge_failed", "params", params)
	//			stats.IncErrors()
	//			return ""
	//		}
	//		section.([]interface{})[i] = m
	//	}
	//	return section
	// apply sub-transformation to single sub-section of document
	default:
		m := docA.merge(docA.GetOriginalObject(), docB.GetOriginalObject())
		if m == nil {
			ctx.Log().Error("event", "execute_function", "function", "join", "error", "merge_failed", "params", params)
			stats.IncErrors()
			return ""
		}
		docC, err := NewJDocFromInterface(m)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "join", "error", "invalid_merge_json", "params", params)
			stats.IncErrors()
			return ""
		}
		return docC.GetOriginalObject()
	}
}

// fnProp function pulls property from custom properties section in config.json.
func fnProp(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "prop", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	cp := GetCustomProperties(ctx)
	if cp != nil {
		if val, ok := cp[extractStringParam(params[0])]; ok {
			return val
		}
	}
	props := GetConfig(ctx).CustomProperties
	if props == nil || props[extractStringParam(params[0])] == nil {
		ctx.Log().Error("event", "execute_function", "function", "prop", "error", "property_not_found", "params", params)
		stats.IncErrors()
		return ""
	}
	return doc.ParseExpression(ctx, props[extractStringParam(params[0])])
}

// fnTenant return current tenant.
func fnTenant(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "tenant", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		return ""
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "tenant", "error", "no_handler", "params", params)
		stats.IncErrors()
		return ""
	}
	return h.TenantId
}

// fnTransform function applies transformation (given by name) to current document.
func fnTransform(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "no_handler", "params", params)
		stats.IncErrors()
		return ""
	}
	if h.Transformations == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "no_named_transformations", "params", params)
		stats.IncErrors()
		return ""
	}
	t := h.Transformations[extractStringParam(params[0])]
	if t == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "unknown_transformation", "params", params)
		stats.IncErrors()
		return ""
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		err := json.Unmarshal([]byte(extractStringParam(params[1])), &section)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "transform", "error", "invalid_json", "params", params)
			stats.IncErrors()
			return ""
		}
	}
	var pattern *JDoc
	if len(params) >= 3 && extractStringParam(params[2]) != "" {
		var err error
		pattern, err = NewJDocFromString(extractStringParam(params[2]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "transform", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
	}
	var join *JDoc
	if len(params) == 4 && extractStringParam(params[3]) != "" {
		var err error
		join, err = NewJDocFromString(extractStringParam(params[3]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "transform", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
	}
	if pattern != nil {
		c, _ := doc.contains(section, pattern.GetOriginalObject(), 0)
		if !c {
			return section
		}
	}
	if join != nil {
		section = doc.merge(join.GetOriginalObject(), section)
	}
	littleDoc, err := NewJDocFromInterface(section)
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", err.Error(), "params", params)
		stats.IncErrors()
		return ""
	}
	var littleRes *JDoc
	if t.IsTransformationByExample {
		littleRes = littleDoc.ApplyTransformationByExample(ctx, t.t)
	} else {
		littleRes = littleDoc.ApplyTransformation(ctx, t.t)
	}
	return littleRes.GetOriginalObject()
}

// fnTransform function applies transformation (given by name) to current document.
func fnITransform(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "no_handler", "params", params)
		stats.IncErrors()
		return ""
	}
	if h.Transformations == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "no_named_transformations", "params", params)
		stats.IncErrors()
		return ""
	}
	t := h.Transformations[extractStringParam(params[0])]
	if t == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "unknown_transformation", "params", params)
		stats.IncErrors()
		return ""
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		err := json.Unmarshal([]byte(extractStringParam(params[1])), &section)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "invalid_json", "params", params)
			stats.IncErrors()
			return ""
		}
	}
	var pattern *JDoc
	if len(params) >= 3 && extractStringParam(params[2]) != "" {
		var err error
		pattern, err = NewJDocFromString(extractStringParam(params[2]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
	}
	var join *JDoc
	if len(params) == 4 && extractStringParam(params[3]) != "" {
		var err error
		join, err = NewJDocFromString(extractStringParam(params[3]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			return nil
		}
	}
	switch section.(type) {
	// apply sub-transformation iteratively to all array elements
	case []interface{}:
		for i, a := range section.([]interface{}) {
			if pattern != nil {
				c, _ := doc.contains(a, pattern.GetOriginalObject(), 0)
				if !c {
					continue
				}
			}
			if join != nil {
				a = doc.merge(join.GetOriginalObject(), a)
			}
			//ctx.Log().Info("A", a, "MERGED", amerged, "JOIN", join.GetOriginalObject())
			littleDoc, err := NewJDocFromInterface(a)
			if err != nil {
				ctx.Log().Error("event", "execute_function", "function", "itransform", "error", err.Error(), "params", params)
				stats.IncErrors()
				return ""
			}
			var littleRes *JDoc
			if t.IsTransformationByExample {
				littleRes = littleDoc.ApplyTransformationByExample(ctx, t.t)
			} else {
				littleRes = littleDoc.ApplyTransformation(ctx, t.t)
			}
			//ctx.Log().Info("item_in", a, "item_out", littleRes.StringPretty(), "path", extractStringParam(params[2]), "idx", i)
			section.([]interface{})[i] = littleRes.GetOriginalObject()
		}
		return section
	/*case map[string]interface{}:
	for k, v := range section.(map[string]interface{}) {
		if pattern != nil {
			c, _ := doc.contains(v, pattern.GetOriginalObject(), 0)
			if !c {
				continue
			}
		}
		if join != nil {
			v = doc.merge(join.GetOriginalObject(), v)
		}
		//ctx.Log().Info("A", a, "MERGED", amerged, "JOIN", join.GetOriginalObject())
		littleDoc, err := NewJDocFromInterface(v)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "itransform", "error", err.Error(), "params", params)
			stats.IncErrors()
			return ""
		}
		var littleRes *JDoc
		if t.IsTransformationByExample {
			littleRes = littleDoc.ApplyTransformationByExample(ctx, t.t)
		} else {
			littleRes = littleDoc.ApplyTransformation(ctx, t.t)
		}
		//ctx.Log().Info("item_in", a, "item_out", littleRes.StringPretty(), "path", extractStringParam(params[2]), "idx", i)
		section.(map[string]interface{})[k] = littleRes.GetOriginalObject()
	}
	return section*/
	// apply sub-transformation to single sub-section of document
	default:
		if pattern != nil {
			c, _ := doc.contains(section, pattern.GetOriginalObject(), 0)
			if !c {
				return section
			}
		}
		if join != nil {
			section = doc.merge(join.GetOriginalObject(), section)
		}
		littleDoc, err := NewJDocFromInterface(section)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "itransform", "error", err.Error(), "params", params)
			stats.IncErrors()
			return ""
		}
		var littleRes *JDoc
		if t.IsTransformationByExample {
			littleRes = littleDoc.ApplyTransformationByExample(ctx, t.t)
		} else {
			littleRes = littleDoc.ApplyTransformation(ctx, t.t)
		}
		return littleRes.GetOriginalObject()
	}
}

// fnFormat function provides human readable string for a timestamp relative to a given time zone. Follows go conventions.
func fnFormat(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 3 {
		ctx.Log().Error("event", "execute_function", "function", "format", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	ts := time.Now()
	if len(params) >= 1 {
		ms, err := strconv.Atoi(extractStringParam(params[0]))
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "format", "error", "time_stamp_expected", "params", params)
			stats.IncErrors()
			return ""
		}
		ts = time.Unix(int64(ms/1000), 0)
	}
	layout := "3:04pm"
	if len(params) >= 2 {
		layout = extractStringParam(params[1])
	}
	if len(params) == 3 {
		tz, err := time.LoadLocation(extractStringParam(params[2]))
		if err == nil {
			ts = ts.In(tz)
		} else {
			ctx.Log().Error("event", "failed_loading_location", "location", extractStringParam(params[2]), "error", err.Error())
		}
	}
	return ts.Format(layout)
}

// fnTrue convenience function which always retrurns true.
func fnTrue(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "true", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	return true
}

// fnFalse convenience function which always retrurns false.
func fnFalse(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("event", "execute_function", "function", "false", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		return ""
	}
	return false
}

func extractStringParam(param string) string {
	param = strings.TrimSpace(param)
	return param[1 : len(param)-1]
}

// ExecuteFunction executes a function on a given JSON document with given parameters.
func (f *JFunction) ExecuteFunction(ctx Context, doc *JDoc, params []string) interface{} {
	return f.fn(ctx, doc, params)
}

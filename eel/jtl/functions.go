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
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robertkrimen/otto"

	. "github.com/Comcast/eel/eel/util"
)

type (
	// JFunction represents built-in functions that can be used in jpath expressions.
	JFunction struct {
		fn           func(ctx Context, doc *JDoc, params []*JParam) interface{}
		minNumParams int
		maxNumParams int
	}
)

type (
	JParam struct {
		val interface{}
		typ string
		err error
	}
)

func NewJParam(param string) *JParam {
	//TODO: add ctx and expected type and errors
	param = strings.TrimSpace(param)
	if param == "" || param == "''" {
		return &JParam{"", TYPE_NULL, nil}
	}
	if len(param) >= 2 && strings.HasPrefix(param, "'") && strings.HasSuffix(param, "'") {
		return &JParam{param[1 : len(param)-1], TYPE_STRING, nil}
	}
	if strings.Contains(param, "{{") && strings.Contains(param, "}}") {
		return &JParam{param, TYPE_EXPR, nil}
	}
	if strings.HasPrefix(param, "{") && strings.HasSuffix(param, "}") {
		var m map[string]interface{}
		err := json.Unmarshal([]byte(param), &m)
		if err != nil {
			return &JParam{param, TYPE_MAP, err}
		} else {
			return &JParam{m, TYPE_MAP, nil}
		}
	}
	if strings.HasPrefix(param, "[") && strings.HasSuffix(param, "]") {
		var a []interface{}
		err := json.Unmarshal([]byte(param), &a)
		if err != nil {
			return &JParam{param, TYPE_ARRAY, err}
		} else {
			return &JParam{a, TYPE_ARRAY, nil}
		}
	}
	if param == "true" {
		return &JParam{true, TYPE_BOOL, nil}
	}
	if param == "false" {
		return &JParam{false, TYPE_BOOL, nil}
	}
	i, err := strconv.ParseInt(param, 10, 64)
	if err == nil {
		return &JParam{i, TYPE_INT, nil}
	}
	f, err := strconv.ParseFloat(param, 64)
	if err == nil {
		return &JParam{f, TYPE_FLOAT, nil}
	}
	return &JParam{param, TYPE_STRING, errors.New("parameter unparsable")}
}

func (p *JParam) Log() {
	Gctx.Log().Info("PARAM", *p, "type", p.GetType(), "val", p.GetVal(), "error", p.GetErr())
}

func (p *JParam) GetType() string {
	return p.typ
}

func (p *JParam) GetErr() error {
	return p.err
}

func (p *JParam) GetVal() interface{} {
	return p.val
}

func (p *JParam) GetStringVal() string {
	switch p.val.(type) {
	case string:
		return p.val.(string)
	}
	return ""
}

func (p *JParam) GetBoolVal() bool {
	switch p.val.(type) {
	case bool:
		return p.val.(bool)
	}
	return false
}

func (p *JParam) GetIntVal() int {
	switch p.val.(type) {
	case int:
		return p.val.(int)
	case int32:
		return int(p.val.(int32))
	case int64:
		return int(p.val.(int64))
	}
	return 0
}

func (p *JParam) GetFloatVal() float64 {
	switch p.val.(type) {
	case float32:
		return float64(p.val.(float32))
	case float64:
		return p.val.(float64)
	}
	return 0.0
}

func (p *JParam) GetJDocVal() *JDoc {
	m, _ := NewJDocFromInterface(p.val)
	return m
}

func (p *JParam) GetMapVal() map[string]interface{} {
	switch p.val.(type) {
	case map[string]interface{}:
		return p.val.(map[string]interface{})
	}
	return nil
}

func (p *JParam) GetArrayVal() []interface{} {
	switch p.val.(type) {
	case []interface{}:
		return p.val.([]interface{})
	}
	return nil
}

const (
	TYPE_STRING = "string"
	TYPE_INT    = "int"
	TYPE_FLOAT  = "float"
	TYPE_ARRAY  = "array"
	TYPE_MAP    = "map"
	TYPE_BOOL   = "bool"
	TYPE_NULL   = "null"
	TYPE_EXPR   = "expression"
)

// NewFunction gets function implementation by name.
func NewFunction(fn string) *JFunction {
	//stats := gctx.Value(EelTotalStats).(*ServiceStats)
	switch fn {
	case "curl":
		// hit external web service
		// method - POST, GET etc.
		// url - url of external service
		// payload - payload to be sent to external service
		// headers - headers to be sent to external service
		// retries - if true, applies retry policy as specified in config.json in case of failure, no retries if false
		// curl('<method>','<url>',['<payload>'],['<header-map>'],['<retries>'])
		// example curl('POST', 'http://foo.com/bar/json', 'foo-{{/content/bar}}')
		return &JFunction{fnCurl, 2, 5}
	case "uuid":
		// returns UUID string
		// uuid()
		return &JFunction{fnUuid, 0, 0}
	case "header":
		// returns a value given the http request header key, or all headers if no key is given
		// header('mykey')
		return &JFunction{fnHeader, 0, 1}
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
		// apply regex to string value and return (first) result: regex('<string>', '<regex>')
		return &JFunction{fnRegex, 2, 3}
	case "match":
		// apply regex to string value and return true if matches: match('<string>', '<regex>')
		return &JFunction{fnMatch, 2, 2}
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
		// collapse a JSON document into a flat array
		return &JFunction{fnCrush, 1, 1}
	case "len":
		// returns length of object (string, array, map)
		return &JFunction{fnLen, 1, 1}
	case "exists":
		// returns true if path exists in document
		return &JFunction{fnExists, 1, 2}
	default:
		//gctx.Log.Error("event", "execute_function", "function", fn, "error", "not_implemented")
		//stats.IncErrors()
		return nil
	}
}

// fnRegex regular expression function returns first matching value.
func fnRegex(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) > 3 {
		ctx.Log().Error("event", "execute_function", "function", "regex", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to regex function"), "regex"})
		return nil
	}
	reg, err := regexp.Compile(params[1].GetStringVal())
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "regex", "error", "invalid_regex", "params", params, "message", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid regex in call to regex function: %s", err.Error()), "regex"})
		return nil

	}
	all := false
	if len(params) == 3 {
		all = params[2].GetBoolVal()
		if params[2].GetType() != TYPE_BOOL {
			ctx.Log().Error("event", "execute_function", "function", "regex", "error", "non_boolean_parameter", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to regex function"), "regex"})
			return nil
		}
	}
	if all {
		items := reg.FindAllString(params[0].GetStringVal(), -1)
		res := ""
		for _, it := range items {
			res += it
		}
		return res
	} else {
		return reg.FindString(params[0].GetStringVal())
	}
}

// fnMatch regular expression function returns true if regex matches.
func fnMatch(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "match", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to match function"), "match"})
		return nil
	}
	reg, err := regexp.Compile(params[1].GetStringVal())
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "match", "error", "invalid_regex", "params", params, "message", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid regex in call to match function: %s", err.Error()), "match"})
		return nil
	}
	return reg.MatchString(params[0].GetStringVal())
}

// fnAlt alternative function.
func fnAlt(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 {
		ctx.Log().Error("event", "execute_function", "function", "alt", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to alt function"), "alt"})
		return nil
	}
	for _, p := range params {
		if sp := p.GetStringVal(); sp != "" {
			return sp
		}
	}
	return ""
}

// fnAnd boolean and function.
func fnAnd(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "and", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to and function"), "and"})
		return nil
	}
	result := true
	for i := 0; i < len(params); i++ {
		b := params[i].GetBoolVal()
		if params[i].GetType() != TYPE_BOOL {
			ctx.Log().Error("event", "execute_function", "function", "and", "error", "non_boolean_parameter", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to and function"), "and"})
			return nil
		}
		result = result && b
	}
	return result
}

// fnOr boolean or function.
func fnOr(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "or", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to or function"), "or"})
		return nil
	}
	result := false
	for i := 0; i < len(params); i++ {
		b := params[i].GetBoolVal()
		if params[i].GetType() != TYPE_BOOL {
			ctx.Log().Error("event", "execute_function", "function", "or", "error", "non_boolean_parameter", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to or function"), "or"})
			return nil
		}
		result = result || b
	}
	return result
}

// fnNot boolean not function.
func fnNot(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "not", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to not function"), "not"})
		return nil
	}
	result := params[0].GetBoolVal()
	if params[0].GetType() != TYPE_BOOL {
		ctx.Log().Error("event", "execute_function", "function", "not", "error", "non_boolean_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to not function"), "not"})
		return nil
	}
	return !result
}

// fnContains contains function.
func fnContains(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "contains", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to contains function"), "contains"})
		return nil
	}
	if len(params) == 2 {
		//var err error
		doc = params[1].GetJDocVal()
		/*if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "contains", "error", "non_json_parameter", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to contains function"), "contains", params})
			return nil
		}*/
	}
	containee := params[0].GetJDocVal()
	/*if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "contains", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to contains function"), "contains", params})
		return nil
	}*/
	matches, _ := doc.MatchesPattern(containee)
	return matches
}

// fnChoose choose function.
func fnChoose(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to choose function"), "choose"})
		return nil
	}
	mydoc := params[0].GetVal()
	/*if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to choose function"), "choose", params})
		return nil
	}*/
	pattern := params[1].GetVal()
	/*if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "choose", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to choose function"), "choose", params})
		return nil
	}*/
	choice := doc.choose(mydoc, pattern)
	return choice
}

// fnCrush crush function.
func fnCrush(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "crush", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to choose function"), "crush"})
		return nil
	}
	mydoc := params[0].GetVal()
	/*if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "crush", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to crush function"), "crush", params})
		return nil
	}*/
	crush := doc.crush(mydoc, nil)
	return crush
}

// fnEquals function. Performs deep equals on two JSON documents, otherwise a simple string comparison will be done.
func fnEquals(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "equals", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to equals function"), "equals"})
		return nil
	}
	pattern := params[0].GetJDocVal()
	//TODO
	/*if err != nil && len(params) == 1 {
		ctx.Log().Error("event", "execute_function", "function", "equals", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to equals function"), "equals", params})
		return nil
	}*/
	if len(params) == 2 {
		if params[0].GetType() != params[1].GetType() {
			return false
		}
		if params[1].GetType() != TYPE_ARRAY && params[1].GetType() != TYPE_MAP {
			return params[0].GetVal() == params[1].GetVal()
		}
		doc = params[1].GetJDocVal()
		//TODO
		/*if err != nil {
			// if not json, just do string comparison (only makes sense for the 2-param version, otherwise must be json)
			return extractStringParam(params[0]) == extractStringParam(params[1])
		}*/
	}
	return doc.Equals(pattern)
}

// fnIfte is an if-then-else function. The first parameter must evaluate to a boolean, the third parameter is optional.
func fnIfte(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 || len(params) > 3 {
		ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to ifte function"), "ifte"})
		return nil
	}
	conditionMet := params[0].GetBoolVal()
	if params[0].GetType() != TYPE_BOOL {
		ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "non_boolean_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to ifte function"), "ifte"})
		return nil
	}
	//TODO: preserve parameter type (bool, string, json, int, float), also applies to alt() etc.
	var res interface{}
	if conditionMet {
		res = params[1].GetVal()
	} else if len(params) == 3 {
		res = params[2].GetVal()
	}
	// hack to resurrect json
	//TODO: check this
	/*if strings.Contains(res, "{") && strings.Contains(res, "}") {
		doc, err := NewJDocFromString(res)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "ifte", "error", "invalid_json", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to ifte function"), "ifte", params})
			return nil

		}
		return doc.GetOriginalObject()
	}*/
	return res
}

// fnCase is a simplification of a nested ifte(equals(),'foo', ifte(equals(...),...)) cascade.
// Example: case('<path_1>','<comparison_value_1>','<return_value_1>', '<path_2>','<comparison_value_2>','<return_value_2>,...,'<default>')
func fnCase(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 3 || len(params)%3 != 1 {
		ctx.Log().Error("event", "execute_function", "function", "case", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to case function"), "case"})
		return nil
	}
	for i := 0; i < len(params)/3; i++ {
		if params[i*3].GetVal() == params[i*3+1].GetVal() {
			return params[i*3+2].GetVal()
		}
	}
	return params[len(params)-1].GetVal()
}

// fnJs JavaScript function. Kind of useful for everything that does not have a built-in function.
func fnJs(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("event", "execute_function", "function", "js", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to js function"), "js"})
		return nil
	}
	vm := otto.New()
	for i := 2; i < len(params)-1; i += 2 {
		vm.Set(params[i].GetStringVal(), params[i+1].GetStringVal())
	}
	//ctx.Log.Info("run", extractStringParam(params[0]))
	value, err := vm.Run(params[0].GetStringVal())
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "js", "error", "vm_error", "params", params, "detail", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("js vm error: %s", err.Error), "js"})
		return nil
	}
	if len(params) > 1 {
		//ctx.Log.Info("get", extractStringParam(params[1]))
		value, err = vm.Get(params[1].GetStringVal())
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "js", "error", "vm_val_error", "params", params, "detail", err.Error())
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("js vm value error: %s", err.Error), "js"})
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
		AddError(ctx, RuntimeError{fmt.Sprintf("js vm value conversion error: %s", err.Error), "js"})
		return nil
	}
	return ret
}

// fnCurl provides curl-like functionality to reach out to helper web services. This function usually has grave performance consequences.
func fnCurl(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 || len(params) > 5 {
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to curl function"), "curl"})
		return nil
	}
	var err error
	retry := false
	if len(params) >= 5 {
		retry = params[4].GetBoolVal()
		if params[4].GetType() != TYPE_BOOL {
			stats.IncErrors()
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "non_boolean_parameter", "params", params)
			AddError(ctx, SyntaxError{"non boolean parameter in call to curl function", "curl"})
			return nil
		}
	}
	url := params[1].GetStringVal()
	if ctx.ConfigValue("debug.url") != nil {
		url = ctx.ConfigValue("debug.url").(string)
	}
	// compose http headers: at a minimum use trace header (if available), then add extra headers (if given in param #5)
	hmap := make(map[string]interface{})
	if len(params) >= 4 {
		hmap = params[3].GetMapVal()
		/*if err != nil {
			stats.IncErrors()
			ctx.Log().Error("event", "execute_function", "function", "curl", "error", "invalid_headers", "detail", err.Error(), "params", params)
			AddError(ctx, SyntaxError{fmt.Sprintf("invalid headers parameters in call to curl function"), "curl", params})
		} else {
			hmap = hdoc.GetMapValue("/")
		}*/
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
		body = params[2].GetStringVal()
	}
	ctx.AddLogValue("destination", "external_service")
	var resp string
	var status int
	if retry {
		resp, status, err = GetRetrier(ctx).RetryEndpoint(ctx, url, body, params[0].GetStringVal(), headers, nil)
	} else {
		resp, status, err = HitEndpoint(ctx, url, body, params[0].GetStringVal(), headers, nil)
	}
	if err != nil {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "unexpected_response", "status", strconv.Itoa(status), "detail", err.Error(), "response", resp, "params", params)
		AddError(ctx, NetworkError{url, err.Error(), status})
		return nil
	}
	if status < 200 || status >= 300 {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("event", "execute_function", "function", "curl", "error", "unexpected_response", "status", strconv.Itoa(status), "response", resp, "params", params)
		AddError(ctx, NetworkError{url, "could not reach endpoint", status})
		return nil
	}
	var res interface{}
	err = json.Unmarshal([]byte(resp), &res)
	if err != nil {
		return resp
	} else {
		return res
	}
}

// fnmHeader function to obtain http header value from incoming event by key.
func fnHeader(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) > 1 {
		ctx.Log().Error("event", "execute_function", "function", "header", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{"wrong number of parameters in call to header function", "header"})
		return ""
	}
	header := ctx.Value(EelRequestHeader)
	if header == nil {
		ctx.Log().Info("event", "execute_function", "function", "header", "message", "header_object_not_found")
		stats.IncErrors()
		AddError(ctx, RuntimeError{"header object not found in call to header function", "header"})
		return ""
	}
	h, ok := header.(http.Header)
	if !ok {
		ctx.Log().Info("event", "execute_function", "function", "header", "message", "header_object_not_valid")
		AddError(ctx, RuntimeError{"header object not valid in call to header function", "header"})
		stats.IncErrors()
		return ""
	}
	if len(params) == 1 && len(params[0].GetStringVal()) > 2 {
		key := params[0].GetStringVal()
		return h.Get(key)
	} else {
		return h
	}
}

// fnUuid return a new uuid.
func fnUuid(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "uuid", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "uuid"})
		return ""
	}
	uuid, err := NewUUID()
	if err != nil {
		ctx.Log().Error("event", "execute_function", "function", "uuid", "error", "uuid_generator", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("uuid generator error in call to uuid function"), "uuid"})
		return ""
	}
	return uuid
}

// fnTraceId returns current tarce id used for logging.
func fnTraceId(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "traceid", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "traceid"})
		return ""
	}
	return ctx.LogValue("tx.traceId")
}

// fnTime return current time in nano-seconds.
func fnTime(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "time", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "time"})
		return ""
	}
	return time.Now().UnixNano()
}

// fnIdent is a function that does nothing. Somtimes interesting for debugging.
func fnIdent(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "ident", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to ident function"), "ident"})
		return ""
	}
	/*p := NewJParam(params[0])
	if p.GetErr() != nil {
		ctx.Log().Error("event", "execute_function", "function", "ident", "error", "bad_parameter", "param", p.GetVal(), "type", p.GetType(), "err", p.GetErr().Error(), "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("bad parameter in call to ident function: %s", p.GetErr()), "ident"})
		return ""
	}*/
	return params[0].GetVal()
}

// fnUpper function to uppercase a string.
func fnUpper(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "upper", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to upper function"), "upper"})
		return ""
	}
	return strings.ToUpper(params[0].GetStringVal())
}

// fnLower function to lowercase a string.
func fnLower(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "lower", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to lower function"), "lower"})
		return ""
	}
	return strings.ToLower(params[0].GetStringVal())
}

// fnSubstr function to lowercase a string.
func fnSubstr(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 3 {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to substr function"), "substr"})
		return ""
	}
	i := params[1].GetIntVal()
	if params[1].GetType() != TYPE_INT {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "param_not_int", "params")
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non int parameters in call to substr function"), "substr"})
		return ""
	}
	j := params[2].GetIntVal()
	if params[2].GetType() != TYPE_INT {
		ctx.Log().Error("event", "execute_function", "function", "substr", "error", "param_not_int", "params")
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non int parameters in call to substr function"), "substr"})
		return ""
	}
	//TODO: check string length
	return params[0].GetStringVal()[i:j]
}

// fnEval function to exaluate a jpath expression agains current document or against a document passed in as parameter. Often useful in combination with fnCurl.
func fnEval(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "eval", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to eval function"), "eval"})
		return ""
	} else if len(params) == 1 {
		return doc.EvalPath(ctx, params[0].GetStringVal())
	} else {
		ldoc := params[1].GetJDocVal()
		if params[1].GetType() == TYPE_NULL {
			ctx.Log().Error("event", "execute_function", "function", "eval", "error", "json_expected", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to eval function"), "eval"})
			return ""
		}
		return ldoc.EvalPath(ctx, params[0].GetStringVal())
	}
}

// fnLen functions returns length of parameter.
func fnLen(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "len", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to len function"), "len"})
		return nil
	}
	obj := params[0].GetVal()
	switch obj.(type) {
	case int:
		return 1
	case int32:
		return 1
	case int64:
		return 1
	case float32:
		return 1
	case float64:
		return 1
	case string:
		return len(obj.(string))
	case []interface{}:
		return len(obj.([]interface{}))
	case map[string]interface{}:
		return len(obj.(map[string]interface{}))
	}
	return 0
}

// fnJoin functions joins two JSON documents given as parameters and returns results.
func fnJoin(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to join function"), "join"})
		return nil
	}
	docA := params[0].GetJDocVal()
	if params[0].GetType() == TYPE_NULL {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to join function"), "join"})
		return nil
	}
	docB := params[1].GetJDocVal()
	if params[1].GetType() == TYPE_NULL {
		ctx.Log().Error("event", "execute_function", "function", "join", "error", "non_json_parameter", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to join function"), "join"})
		return nil
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
			AddError(ctx, RuntimeError{fmt.Sprintf("merge failed in call to join function"), "join"})
			return nil
		}
		docC, err := NewJDocFromInterface(m)
		if err != nil {
			ctx.Log().Error("event", "execute_function", "function", "join", "error", "invalid_merge_json", "params", params)
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("non json merge result in call to join function"), "join"})
			return nil
		}
		return docC.GetOriginalObject()
	}
}

// fnProp function pulls property from custom properties section in config.json.
func fnProp(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("event", "execute_function", "function", "prop", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to prop function"), "prop"})
		return ""
	}
	cp := GetCustomProperties(ctx)
	if cp != nil {
		if val, ok := cp[params[0].GetStringVal()]; ok {
			return val
		}
	}
	props := GetConfig(ctx).CustomProperties
	if props == nil || props[params[0].GetStringVal()] == nil {
		ctx.Log().Error("event", "execute_function", "function", "prop", "error", "property_not_found", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("property %s not found in call to prop function", params[0].GetStringVal()), "prop"})
		return ""
	}
	return doc.ParseExpression(ctx, props[params[0].GetStringVal()])
}

// fnTenant return current tenant.
func fnTenant(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "tenant", "error", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to tenant function"), "tenant"})
		return ""
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "tenant", "error", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to tenant function"), "tenant"})
		return ""
	}
	return h.TenantId
}

// fnTransform function applies transformation (given by name) to current document.
func fnTransform(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to transform function"), "transform"})
		return nil
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to transform function"), "transform"})
		return nil
	}
	if h.Transformations == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "no_named_transformations", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformations found in call to transform function"), "transform"})
		return nil
	}
	t := h.Transformations[params[0].GetStringVal()]
	if t == nil {
		ctx.Log().Error("event", "execute_function", "function", "transform", "error", "unknown_transformation", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformation %s found in call to transform function", params[0].GetStringVal()), "transform"})
		return nil
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		section = params[1].GetVal()
	}
	var pattern *JDoc
	if len(params) >= 3 && params[2].GetType() != TYPE_NULL {
		pattern = params[2].GetJDocVal()
	}
	var join *JDoc
	if len(params) == 4 && params[3].GetType() != TYPE_NULL {
		//var err error
		join = params[3].GetJDocVal()
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
		AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to transform function"), "transform"})
		return nil
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
func fnITransform(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to itransform function"), "itransform"})
		return nil
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to itransform function"), "itransform"})
		return nil
	}
	if h.Transformations == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "no_named_transformations", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformations found in call to itransform function"), "itransform"})
		return nil
	}
	t := h.Transformations[params[0].GetStringVal()]
	if t == nil {
		ctx.Log().Error("event", "execute_function", "function", "itransform", "error", "unknown_transformation", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformation %s found in call to itransform function", params[0].GetStringVal()), "itransform"})
		return nil
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		section = params[1].GetVal()
	}
	var pattern *JDoc
	if len(params) >= 3 && params[2].GetType() != TYPE_NULL {
		pattern = params[2].GetJDocVal()
	}
	var join *JDoc
	if len(params) == 4 && params[3].GetType() != TYPE_NULL {
		//var err error
		join = params[3].GetJDocVal()
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
				AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to itransform function"), "itransform"})
				return nil
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
			AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to itransform function: %s", err.Error()), "itransform"})
			return nil
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
func fnFormat(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 3 {
		ctx.Log().Error("event", "execute_function", "function", "format", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to format function"), "format"})
		return ""
	}
	ts := time.Now()
	if len(params) >= 1 {
		if params[0].GetType() != TYPE_INT {
			ctx.Log().Error("event", "execute_function", "function", "format", "error", "time_stamp_expected", "params", params)
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("time stamp parameter expected in call to format function"), "format"})
			return ""
		}
		ms := params[0].GetIntVal()
		ts = time.Unix(int64(ms/1000), 0)
	}
	layout := "3:04pm"
	if len(params) >= 2 {
		layout = params[1].GetStringVal()
	}
	if len(params) == 3 {
		tz, err := time.LoadLocation(params[2].GetStringVal())
		if err == nil {
			ts = ts.In(tz)
		} else {
			ctx.Log().Error("event", "failed_loading_location", "location", params[2].GetStringVal(), "error", err.Error())
			AddError(ctx, RuntimeError{fmt.Sprintf("failed loading location %s", params[2].GetStringVal()), "format"})
		}
	}
	return ts.Format(layout)
}

// fnTrue convenience function which always retrurns true.
func fnTrue(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "true", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to true function"), "true"})
		return ""
	}
	return true
}

// fnFalse convenience function which always retrurns false.
func fnFalse(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0].GetStringVal() != "" {
		ctx.Log().Error("event", "execute_function", "function", "false", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to false function"), "false"})
		return ""
	}
	return false
}

// fnExists returns true if a particular path exists in a json document.
func fnExists(ctx Context, doc *JDoc, params []*JParam) interface{} {
	//OK
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 2 {
		ctx.Log().Error("event", "execute_function", "function", "exists", "error", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to exists function"), "exists"})
		return false
	}
	if len(params) == 2 {
		//var err error
		doc = params[1].GetJDocVal()
	}
	return doc.HasPath(params[0].GetStringVal())
}

/*func extractStringParam(param string) string {
	param = strings.TrimSpace(param)
	return param[1 : len(param)-1]
}*/

// ExecuteFunction executes a function on a given JSON document with given parameters.
func (f *JFunction) ExecuteFunction(ctx Context, doc *JDoc, params []*JParam) interface{} {
	return f.fn(ctx, doc, params)
}

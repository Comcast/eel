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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/robertkrimen/otto"

	. "github.com/Comcast/eel/util"
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
		// method - POST, GET etc.
		// url - url of external service
		// payload - payload to be sent to external service
		// headers - headers to be sent to external service
		// retries - if true, applies retry policy as specified in config.json in case of failure, no retries if false
		// curl('<method>','<url>',['<payload>'],['<header-map>'],['<retries>'])
		// example curl('POST', 'http://foo.com/bar/json', 'foo-{{/content/bar}}')
		return &JFunction{fnCurl, 2, 5}
	case "curlOAuth1":
		// curlOAuth1('<method>','<url>',['<payload>'],['<header-map>'],['<retries>'],['<oauth1-provider>'])
		return &JFunction{fnCurlOAuth1, 2, 6}
	case "hmac":
		// hmac("<hashFunc>", '<input>', '<key>')
		return &JFunction{fnHmac, 3, 3}
	case "loadfile":
		// loadfile("<filename>')
		return &JFunction{fnLoadFile, 1, 1}
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
	case "base64decode":
		// base64 decode input string, example base64decode('foo')
		return &JFunction{fnBase64Decode, 1, 1}
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
	case "etransform":
		// apply external transformation and return single result (efficient shortcut for and equivalent to curl http://localhost:8080/proc)
		return &JFunction{fnETransform, 1, 1}
	case "ptransform":
		// apply external transformation and execute publisher(s) (efficient shortcut for and equivalent to curl http://localhost:8080/proxy)
		return &JFunction{fnPTransform, 1, 1}
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
	case "string":
		// returns length of object (string, array, map)
		return &JFunction{fnString, 2, 2}
	case "exists":
		// returns true if path exists in document
		return &JFunction{fnExists, 1, 2}
	case "calc":
		// evalutaes simple arithmetic expressions in native go and returns result
		return &JFunction{fnCalc, 1, 1}
	case "hashmod":
		// hash a given string and then mod it by the given divider
		return &JFunction{fnHashMod, 2, 2}
	default:
		//gctx.Log.Error("error_type", "func_", "op", fn, "cause", "not_implemented")
		//stats.IncErrors()
		return nil
	}
}

// fnRegex regular expression function returns first matching value.
func fnRegex(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) > 3 {
		ctx.Log().Error("error_type", "func_regex", "op", "regex", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to regex function"), "regex", params})
		return nil
	}
	reg, err := regexp.Compile(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_regex", "op", "regex", "cause", "invalid_regex", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid regex in call to regex function: %s", err.Error()), "regex", params})
		return nil

	}
	all := false
	if len(params) == 3 {
		all, err = strconv.ParseBool(extractStringParam(params[2]))
		if err != nil {
			ctx.Log().Error("error_type", "func_regex", "op", "regex", "cause", "non_boolean_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to regex function"), "regex", params})
			return nil
		}
	}
	if all {
		items := reg.FindAllString(extractStringParam(params[0]), -1)
		res := ""
		for _, it := range items {
			res += it
		}
		return res
	} else {
		return reg.FindString(extractStringParam(params[0]))
	}
}

// fnCalc regular expression function returns true if regex matches.
func fnCalc(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_calc", "op", "calc", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to calc function"), "calc", params})
		return nil
	}
	functions := map[string]govaluate.ExpressionFunction{
		"now": func(args ...interface{}) (interface{}, error) {
			return float64(time.Now().UnixNano() / 1e6), nil
		},
	}
	expr, err := govaluate.NewEvaluableExpressionWithFunctions(extractStringParam(params[0]), functions)
	if err != nil {
		ctx.Log().Error("error_type", "func_calc", "op", "calc", "cause", "invalid_expression", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid expression in call to calc function: %s", err.Error()), "calc", params})
		return nil
	}
	res, err := expr.Evaluate(nil)
	if err != nil {
		ctx.Log().Error("error_type", "func_calc", "op", "calc", "cause", "invalid_evaluation", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid evaluation in call to calc function: %s", err.Error()), "calc", params})
		return nil
	}
	return res
}

// fnMatch regular expression function returns true if regex matches.
func fnMatch(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("error_type", "func_match", "op", "match", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to match function"), "match", params})
		return nil
	}
	reg, err := regexp.Compile(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_match", "op", "match", "cause", "invalid_regex", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("invalid regex in call to match function: %s", err.Error()), "match", params})
		return nil
	}
	return reg.MatchString(extractStringParam(params[0]))
}

// fnAlt alternative function.
func fnAlt(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	ctx.Log().Error("error_type", "func_alt", "op", "alt", "cause", "now_implemented_in_parser", "params", params)
	stats.IncErrors()
	AddError(ctx, SyntaxError{fmt.Sprintf("alt function now implemented in parser"), "alt", params})
	return nil
}

// fnAnd boolean and function.
func fnAnd(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("error_type", "func_and", "op", "and", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to and function"), "and", params})
		return nil
	}
	result := true
	for i := 0; i < len(params); i++ {
		b, err := strconv.ParseBool(extractStringParam(params[i]))
		if err != nil {
			ctx.Log().Error("error_type", "func_and", "op", "and", "cause", "non_boolean_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to and function"), "and", params})
			return nil
		}
		result = result && b
		/*if result == false {
			return result
		}*/
	}
	return result
}

// fnOr boolean or function.
func fnOr(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("error_type", "func_or", "op", "or", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to or function"), "or", params})
		return nil
	}
	result := false
	for i := 0; i < len(params); i++ {
		b, err := strconv.ParseBool(extractStringParam(params[i]))
		if err != nil {
			ctx.Log().Error("error_type", "func_or", "op", "or", "cause", "non_boolean_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to or function"), "or", params})
			return nil
		}
		result = result || b
		/*if result == true {
			return result
		}*/
	}
	return result
}

// fnNot boolean not function.
func fnNot(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_not", "op", "not", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to not function"), "not", params})
		return nil
	}
	result, err := strconv.ParseBool(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_not", "op", "not", "cause", "non_boolean_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non boolean parameters in call to not function"), "not", params})
		return nil
	}
	return !result
}

// fnContains contains function.
func fnContains(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("error_type", "func_contains", "op", "contains", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to contains function"), "contains", params})
		return nil
	}
	if len(params) == 2 {
		var err error
		doc, err = NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("error_type", "func_contains", "op", "contains", "cause", "non_json_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to contains function"), "contains", params})
			return nil
		}
	}
	containee, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_contains", "op", "contains", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to contains function"), "contains", params})
		return nil
	}
	matches, _ := doc.MatchesPattern(containee)
	return matches
}

// fnChoose choose function.
func fnChoose(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("error_type", "func_choose", "op", "choose", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to choose function"), "choose", params})
		return nil
	}
	mydoc, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_choose", "op", "choose", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to choose function"), "choose", params})
		return nil
	}
	pattern, err := NewJDocFromString(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_choose", "op", "choose", "error", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to choose function"), "choose", params})
		return nil
	}
	choice := doc.choose(mydoc.GetOriginalObject(), pattern.GetOriginalObject())
	return choice
}

// fnCrush crush function.
func fnCrush(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_crush", "op", "crush", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to choose function"), "crush", params})
		return nil
	}
	mydoc, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_crush", "op", "crush", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to crush function"), "crush", params})
		return nil
	}
	crush := doc.crush(mydoc.GetOriginalObject(), nil)
	return crush
}

// fnEquals function. Performs deep equals on two JSON documents, otherwise a simple string comparison will be done.
func fnEquals(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 || len(params) > 2 {
		ctx.Log().Error("error_type", "func_equals", "op", "equals", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to equals function"), "equals", params})
		return nil
	}
	pattern, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil && len(params) == 1 {
		ctx.Log().Error("error_type", "func_equals", "op", "equals", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to equals function"), "equals", params})
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
	ctx.Log().Error("error_type", "func_ifte", "op", "ifte", "cause", "now_implemented_in_parser", "params", params)
	stats.IncErrors()
	AddError(ctx, SyntaxError{fmt.Sprintf("ifte function now implemented in parser"), "ifte", params})
	return nil
}

// fnCase is a simplification of a nested ifte(equals(),'foo', ifte(equals(...),...)) cascade.
// Example: case('<path_1>','<comparison_value_1>','<return_value_1>', '<path_2>','<comparison_value_2>','<return_value_2>,...,'<default>')
func fnCase(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	ctx.Log().Error("error_type", "func_case", "op", "case", "cause", "now_implemented_in_parser", "params", params)
	stats.IncErrors()
	AddError(ctx, SyntaxError{fmt.Sprintf("case function now implemented in parser"), "case", params})
	return nil
}

// fnJs JavaScript function. Kind of useful for everything that does not have a built-in function.
func fnJs(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 1 {
		ctx.Log().Error("error_type", "func_js", "op", "js", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to js function"), "js", params})
		return nil
	}
	vm := otto.New()
	for i := 2; i < len(params)-1; i += 2 {
		vm.Set(extractStringParam(params[i]), extractStringParam(params[i+1]))
	}
	//ctx.Log.Info("run", extractStringParam(params[0]))
	value, err := vm.Run(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_js", "op", "js", "cause", "vm_error", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("js vm error: %s", err.Error), "js", params})
		return nil
	}
	if len(params) > 1 {
		//ctx.Log.Info("get", extractStringParam(params[1]))
		value, err = vm.Get(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("error_type", "func_js", "op", "js", "cause", "vm_val_error", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("js vm value error: %s", err.Error), "js", params})
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
		ctx.Log().Error("error_type", "func_js", "op", "js", "cause", "vm_val_conv_error", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("js vm value conversion error: %s", err.Error), "js", params})
		return nil
	}
	return ret
}

// fnCurl provides curl-like functionality to reach out to helper web services. This function usually has grave performance consequences.
func fnCurl(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 2 || len(params) > 5 {
		ctx.Log().Error("error_type", "func_curl", "op", "curl", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to curl function"), "curl", params})
		return nil
	}
	var err error
	retry := false
	if len(params) >= 5 {
		retry, err = strconv.ParseBool(extractStringParam(params[4]))
		if err != nil {
			stats.IncErrors()
			ctx.Log().Error("error_type", "func_curl", "op", "curl", "cause", "non_boolean_parameter", "params", params, "error", err.Error())
			AddError(ctx, SyntaxError{"non boolean parameter in call to curl function", "curl", params})
			return nil
		}
	}
	endpoint := extractStringParam(params[1])
	// url encode query string
	parsed, _ := url.Parse(endpoint)
	parsed.RawQuery = parsed.Query().Encode()
	endpoint = parsed.String()
	if ctx.ConfigValue("debug.url") != nil {
		endpoint = ctx.ConfigValue("debug.url").(string)
	}
	// compose http headers: at a minimum use trace header (if available), then add extra headers (if given in param #5)
	hmap := make(map[string]interface{})
	if len(params) >= 4 {
		hdoc, err := NewJDocFromString(extractStringParam(params[3]))
		if err != nil {
			stats.IncErrors()
			ctx.Log().Error("error_type", "func_curl", "op", "curl", "cause", "invalid_headers", "error", err.Error(), "params", params)
			AddError(ctx, SyntaxError{fmt.Sprintf("invalid headers parameters in call to curl function"), "curl", params})
		} else {
			hmap = hdoc.GetMapValue("/")
		}
	}
	headers := make(map[string]string, 0)
	traceHeaderKey := GetConfig(ctx).HttpTransactionHeader
	if traceHeaderKey != "" && ctx.Value(traceHeaderKey) != nil {
		if _, ok := ctx.Value(traceHeaderKey).(string); ok {
			headers[traceHeaderKey] = ctx.Value(traceHeaderKey).(string)
		}
	}
	tenantHeaderKey := GetConfig(ctx).HttpTenantHeader
	if tenantHeaderKey != "" && ctx.Value(tenantHeaderKey) != nil {
		if _, ok := ctx.Value(tenantHeaderKey).(string); ok {
			headers[tenantHeaderKey] = ctx.Value(tenantHeaderKey).(string)
		}
	}
	debugHeaderKey := GetConfig(ctx).HttpDebugHeader
	if debugHeaderKey != "" {
		dlp := GetDebugLogParams(ctx)
		if dlp != nil && dlp.IdWhiteList != nil && dlp.LogParams != nil {
			wlistId := doc.ParseExpression(ctx, dlp.IdPath)
			if wlistId != nil {
				switch wlistId.(type) {
				case string:
					dlp.Lock.RLock()
					defer dlp.Lock.RUnlock()
					if _, ok := dlp.IdWhiteList[wlistId.(string)]; ok {
						headers[GetConfig(ctx).HttpTransactionHeader] = "true"
					}
				}
			}
		}
	}
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
	var resp string
	var status int
	if retry {
		resp, status, err = GetRetrier(ctx).RetryEndpoint(ctx, endpoint, body, extractStringParam(params[0]), headers, nil)
	} else {
		resp, status, err = HitEndpoint(ctx, endpoint, body, extractStringParam(params[0]), headers, nil)
	}
	if err != nil {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("error_type", "func_curl", "op", "curl", "cause", "curl_error", "status", strconv.Itoa(status), "error", err.Error(), "response", resp, "params", params)
		AddError(ctx, NetworkError{endpoint, err.Error(), status})
		return nil
	}
	if status < 200 || status >= 300 {
		// this error will already be counted by hitEndpoint
		ctx.Log().Error("error_type", "func_curl", "op", "curl", "cause", "curl_status", "status", strconv.Itoa(status), "response", resp, "params", params)
		AddError(ctx, NetworkError{endpoint, "endpoint returned error", status})
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

// fnCurlOAuth1 provides curl-like functionality to reach out to helper web services, with oauth 1.0 authenication
func fnCurlOAuth1(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) < 6 {
		ctx.Log().Error("error_type", "func_curlOAuth1", "op", "curlOAuth1", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to curlOAuth1 function"), "curl", params})
		return nil
	} else {
		method := extractStringParam(params[0])
		endpoint := extractStringParam(params[1])
		headers := extractStringParam(params[3])
		oauthProvider := extractStringParam(params[5])

		//add Authorization header by oauthProvider
		res := make(map[string]string)
		err := json.Unmarshal([]byte(headers), &res)
		if nil != err {
			ctx.Log().Error("error_type", "func_curlOAuth1", "op", "curlOAuth1", "cause", "invalid_header_passed_in", "params", params)
			return nil
		}
		oauthConsumer := NewOAuthConsumer(oauthProvider)
		res["Authorization"], err = oauthConsumer.GetOAuth1Header(ctx, method, endpoint)
		if nil != err {
			ctx.Log().Error("error_type", "func_curlOAuth1", "op", "curlOAuth1", "cause", "no_valid_oauth_header", "params", params)
			return nil
		}
		newHeaders, err := json.Marshal(res)
		if nil != err {
			ctx.Log().Error("error_type", "func_curlOAuth1", "op", "curlOAuth1", "cause", "error_when_create_new_header", "params", params)
			return nil
		}
		params[3] = "'" + string(newHeaders) + "'"

		//unset oauthProvider for fnCurl
		params = params[:5]
	}
	return fnCurl(ctx, doc, params)
}

// fnHeader function to obtain http header value from incoming event by key.
func fnHeader(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) > 1 {
		ctx.Log().Error("error_type", "func_header", "op", "header", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{"wrong number of parameters in call to header function", "header", params})
		return ""
	}
	header := ctx.Value(EelRequestHeader)
	if header == nil {
		ctx.Log().Info("error_type", "func_header", "op", "header", "cause", "header_object_not_found")
		stats.IncErrors()
		AddError(ctx, RuntimeError{"header object not found in call to header function", "header", params})
		return ""
	}
	h, ok := header.(http.Header)
	if !ok {
		ctx.Log().Info("error_type", "func_header", "op", "header", "cause", "header_object_not_valid")
		AddError(ctx, RuntimeError{"header object not valid in call to header function", "header", params})
		stats.IncErrors()
		return ""
	}
	if len(params) == 1 && len(params[0]) > 2 {
		key := extractStringParam(params[0])
		return h.Get(key)
	} else {
		return h
	}
}

// fnUuid return a new uuid.
func fnUuid(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_uuid", "op", "uuid", "cause", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "uuid", params})
		return ""
	}
	uuid, err := NewUUID()
	if err != nil {
		ctx.Log().Error("error_type", "func_uuid", "op", "uuid", "cause", "uuid_init", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("uuid generator error in call to uuid function"), "uuid", params})
		return ""
	}
	return uuid
}

// fnTraceId returns current tarce id used for logging.
func fnTraceId(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_traceid", "op", "traceid", "cause", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "traceid", params})
		return ""
	}
	return ctx.LogValue("tx.traceId")
}

// fnTime return current time in milli-seconds.
func fnTime(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_time", "op", "time", "cause", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to uuid function"), "time", params})
		return ""
	}
	return time.Now().UnixNano() / 1e6
}

// fnIdent is a function that does nothing. Sometimes interesting for debugging.
func fnIdent(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_ident", "op", "ident", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to ident function"), "ident", params})
		return ""
	}
	return extractStringParam(params[0])
}

// fnUpper function to uppercase a string.
func fnUpper(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_upper", "op", "upper", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to upper function"), "upper", params})
		return ""
	}
	return strings.ToUpper(extractStringParam(params[0]))
}

// fnLower function to lowercase a string.
func fnLower(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_lower", "op", "lower", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to lower function"), "lower", params})
		return ""
	}
	return strings.ToLower(extractStringParam(params[0]))
}

// fnBase64Decode function to decode a base64 string.
func fnBase64Decode(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_base64decode", "op", "base64decode", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to base64decode function"), "base64decode", params})
		return ""
	}

	bs, err := base64.StdEncoding.DecodeString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_base64decode", "op", "base64decode", "cause", "error_decode", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, RuntimeError{err.Error(), "base64decode", params})
		return ""
	}

	return string(bs)
}

// fnSubstr function to lowercase a string.
func fnSubstr(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 3 {
		ctx.Log().Error("error_type", "func_substr", "op", "substr", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to substr function"), "substr", params})
		return ""
	}
	i, err := strconv.Atoi(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_substr", "op", "substr", "cause", "param_not_int", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non int parameters in call to substr function"), "substr", params})
		return ""
	}
	j, err := strconv.Atoi(extractStringParam(params[2]))
	if err != nil {
		ctx.Log().Error("error_type", "func_substr", "op", "substr", "cause", "param_not_int", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non int parameters in call to substr function"), "substr", params})
		return ""
	}
	return extractStringParam(params[0])[i:j]
}

// fnEval function to evaluate a jpath expression against the current document or against a document passed in as parameter. Often useful in combination with fnCurl.
func fnEval(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 2 {
		ctx.Log().Error("error_type", "func_eval", "op", "eval", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to eval function"), "eval", params})
		return ""
	} else if len(params) == 1 {
		return doc.EvalPath(ctx, extractStringParam(params[0]))
	} else {
		ldoc, err := NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("error_type", "func_eval", "op", "eval", "cause", "json_expected", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to eval function"), "eval", params})
			return ""
		}
		return ldoc.EvalPath(ctx, extractStringParam(params[0]))
	}
}

// fnLen functions returns length of parameter.
func fnLen(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_len", "op", "len", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to len function"), "len", params})
		return nil
	}
	var obj interface{}
	err := json.Unmarshal([]byte(extractStringParam(params[0])), &obj)
	if err != nil {
		return len(extractStringParam(params[0]))
	}
	switch obj.(type) {
	case []interface{}:
		return len(obj.([]interface{}))
	case map[string]interface{}:
		return len(obj.(map[string]interface{}))
	}
	return 0
}

// fnString function converts object to string. Currently, we support []string. ie. convert ["d1", "d2"] to "d1, d2"
func fnString(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("error_type", "func_string", "op", "string", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to string function"), "string", params})
		return nil
	}
	var obj []string
	err := json.Unmarshal([]byte(extractStringParam(params[0])), &obj)
	if err != nil {
		return extractStringParam(params[0])
	}

	return strings.Join(obj, extractStringParam(params[1])+" ")

}

// fnJoin functions joins two JSON documents given as parameters and returns results.
func fnJoin(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to join function"), "join", params})
		return nil
	}
	docA, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to join function"), "join", params})
		return nil
	}
	docB, err := NewJDocFromString(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "non_json_parameter", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to join function"), "join", params})
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
	//			ctx.Log().Error("error_type", "func_join", "op", "join", "error", err.Error(), "params", params)
	//			stats.IncErrors()
	//			return ""
	//		}
	//		m := docA.merge(littleDoc.GetOriginalObject(), docB.GetOriginalObject())
	//		if m == nil {
	//			ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "merge_failed", "params", params)
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
			ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "merge_failed", "params", params)
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("merge failed in call to join function"), "join", params})
			return nil
		}
		docC, err := NewJDocFromInterface(m)
		if err != nil {
			ctx.Log().Error("error_type", "func_join", "op", "join", "cause", "invalid_merge_json", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("non json merge result in call to join function"), "join", params})
			return nil
		}
		return docC.GetOriginalObject()
	}
}

// fnProp function pulls property from custom properties section in config.json.
func fnProp(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 {
		ctx.Log().Error("error_type", "func_prop", "op", "prop", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to prop function"), "prop", params})
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
		ctx.Log().Error("error_type", "func_prop", "op", "prop", "cause", "property_not_found", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("property %s not found in call to prop function", extractStringParam(params[0])), "prop", params})
		return ""
	}
	return doc.ParseExpression(ctx, props[extractStringParam(params[0])])
}

// fnTenant return current tenant.
func fnTenant(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_tenant", "op", "tenant", "cause", "no_parameters_expected", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to tenant function"), "tenant", params})
		return ""
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("error_type", "func_tenant", "op", "tenant", "cause", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to tenant function"), "tenant", params})
		return ""
	}
	return h.TenantId
}

// fnTransform function applies transformation (given by name) to current document.
func fnTransform(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to transform function"), "transform", params})
		return nil
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to transform function"), "transform", params})
		return nil
	}
	if h.Transformations == nil {
		ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "no_named_transformations", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformations found in call to transform function"), "transform", params})
		return nil
	}
	t := h.Transformations[extractStringParam(params[0])]
	if t == nil {
		ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "unknown_transformation", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformation %s found in call to transform function", extractStringParam(params[0])), "transform", params})
		return nil
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		err := json.Unmarshal([]byte(extractStringParam(params[1])), &section)
		if err != nil {
			ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "invalid_json", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to transform function"), "transform", params})
			return nil
		}
	}
	var pattern *JDoc
	if len(params) >= 3 && extractStringParam(params[2]) != "" {
		var err error
		pattern, err = NewJDocFromString(extractStringParam(params[2]))
		if err != nil {
			ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "non_json_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to transform function"), "transform", params})
			return nil
		}
	}
	var join *JDoc
	if len(params) == 4 && extractStringParam(params[3]) != "" {
		var err error
		join, err = NewJDocFromString(extractStringParam(params[3]))
		if err != nil {
			ctx.Log().Error("error_type", "func_transform", "op", "transform", "cause", "non_json_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to transform function"), "transform", params})
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
		ctx.Log().Error("error_type", "func_transform", "cause", "json_parse_error", "op", "transform", "error", err.Error(), "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to transform function"), "transform", params})
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
func fnITransform(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 4 {
		ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to itransform function"), "itransform", params})
		return nil
	}
	h := GetCurrentHandlerConfig(ctx)
	if h == nil {
		ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "no_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("current handler not found in call to itransform function"), "itransform", params})
		return nil
	}
	if h.Transformations == nil {
		ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "no_named_transformations", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformations found in call to itransform function"), "itransform", params})
		return nil
	}
	t := h.Transformations[extractStringParam(params[0])]
	if t == nil {
		ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "unknown_transformation", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no named transformation %s found in call to itransform function", extractStringParam(params[0])), "itransform", params})
		return nil
	}
	var section interface{}
	section = doc.GetOriginalObject()
	if len(params) >= 2 {
		err := json.Unmarshal([]byte(extractStringParam(params[1])), &section)
		if err != nil {
			ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "invalid_json", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to itransform function"), "itransform", params})
			return nil
		}
	}
	var pattern *JDoc
	if len(params) >= 3 && extractStringParam(params[2]) != "" {
		var err error
		pattern, err = NewJDocFromString(extractStringParam(params[2]))
		if err != nil {
			ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "non_json_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to itransform function"), "itransform", params})
			return nil
		}
	}
	var join *JDoc
	if len(params) == 4 && extractStringParam(params[3]) != "" {
		var err error
		join, err = NewJDocFromString(extractStringParam(params[3]))
		if err != nil {
			ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "non_json_parameter", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to itransform function"), "itransform", params})
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
				ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "json_parse_error", "error", err.Error(), "params", params)
				stats.IncErrors()
				AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to itransform function"), "itransform", params})
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
			ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "error", err.Error(), "params", params)
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
			ctx.Log().Error("error_type", "func_itransform", "op", "itransform", "cause", "json_parse_error", "error", err.Error(), "params", params)
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("transformation error in call to itransform function: %s", err.Error()), "itransform", params})
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
func fnFormat(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 3 {
		ctx.Log().Error("error_type", "func_format", "op", "format", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to format function"), "format", params})
		return ""
	}
	ts := time.Now()
	if len(params) >= 1 {
		ms, err := strconv.Atoi(extractStringParam(params[0]))
		if err != nil {
			ctx.Log().Error("error_type", "func_format", "op", "format", "cause", "time_stamp_expected", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("time stamp parameter expected in call to format function"), "format", params})
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
			ctx.Log().Error("error_type", "func_format", "op", "format", "cause", "failed_loading_location", "location", extractStringParam(params[2]), "error", err.Error())
			AddError(ctx, RuntimeError{fmt.Sprintf("failed loading location %s in call to format function", extractStringParam(params[2])), "format", params})
		}
	}
	return ts.Format(layout)
}

// fnTrue convenience function which always retrurns true.
func fnTrue(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_true", "op", "true", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to true function"), "true", params})
		return ""
	}
	return true
}

// fnFalse convenience function which always retrurns false.
func fnFalse(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || params[0] != "" {
		ctx.Log().Error("error_type", "func_false", "op", "false", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("no parameters expected in call to false function"), "false", params})
		return ""
	}
	return false
}

// fnExists returns true if a particular path exists in a json document.
func fnExists(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 2 {
		ctx.Log().Error("error_type", "func_exists", "op", "exists", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to exists function"), "exists", params})
		return false
	}
	if len(params) == 2 {
		var err error
		doc, err = NewJDocFromString(extractStringParam(params[1]))
		if err != nil {
			ctx.Log().Error("error_type", "func_exists", "op", "exists", "cause", "json_expected", "params", params, "error", err.Error())
			stats.IncErrors()
			AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to exists function"), "exists", params})
			return false
		}
	}
	return doc.HasPath(extractStringParam(params[0]))
}

// fnETransform function applies matching transformation to document passed in as parameter (equivalent to curl http://localhost:8080/proc).
// A single result is returned by this function.
func fnETransform(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 1 {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to etransform function"), "etransform", params})
		return nil
	}
	// prepare event
	event, err := NewJDocFromString(extractStringParam(params[0]))
	if err != nil {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "invalid_json", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to etransform function"), "etransform", params})
		return nil
	}
	// pick handler
	handlers := GetHandlerFactory(ctx).GetHandlersForEvent(ctx, event)
	if len(handlers) == 0 {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "no_matching_handler", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no matching handler found in call to etransform function"), "etransform", params})
		return nil
	}
	if len(handlers) > 1 {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "too_many_matching_handlers", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("too many matching handlers found in call to etransform function"), "etransform", params})
		return nil
	}
	// apply debug logs
	logParams := GetConfig(ctx).LogParams
	if logParams != nil {
		for k, v := range logParams {
			ev := event.ParseExpression(ctx, v)
			ctx.AddLogValue(k, ev)
		}
	}
	// apply handler / transformation
	eps, err := handlers[0].ProcessEvent(Gctx.SubContext(), event)
	if err != nil {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "bad_transformation", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("failed to process external transformation in call to etransform function"), "etransform", params})
		return nil
	}
	if len(eps) == 0 {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "no_results", "params", params)
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("no results found in call to etransform function"), "etransform", params})
		return nil
	}
	// if this check is present some unit tests will fail
	/*if len(eps) > 1 {
		ctx.Log().Error("error_type", "func_etransform", "op", "etransform", "cause", "too_many_results", "params", params, "count", len(eps))
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("too many results found in call to etransform function"), "etransform", params})
		return nil
	}*/
	result := eps[0].GetPayloadParsed().GetOriginalObject()
	return result
}

// fnPTransform function applies matching transformation to document passed in as parameter. Any resulting publisher(s) will be
// executed (equivalent to curl http://localhost:8080/proxy).
func fnPTransform(ctx Context, doc *JDoc, params []string) interface{} {
	// note: calling ptransform in sync or debug mode does not make sense - should we raise an error in such a scenario?
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) == 0 || len(params) > 1 {
		ctx.Log().Error("error_type", "func_ptransform", "op", "etransform", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to ptransform function"), "etransform", params})
		return nil
	}
	// prepare event
	rawEvent := extractStringParam(params[0])
	event, err := NewJDocFromString(rawEvent)
	if err != nil {
		ctx.Log().Error("error_type", "func_ptransform", "op", "etransform", "cause", "invalid_json", "params", params, "error", err.Error())
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("non json parameters in call to ptransform function"), "etransform", params})
		return nil
	}
	// apply debug logs
	logParams := GetConfig(ctx).LogParams
	if logParams != nil {
		for k, v := range logParams {
			ev := event.ParseExpression(ctx, v)
			ctx.AddLogValue(k, ev)
		}
	}
	// handle event and execute publisher(s)
	// both sync=true or debug=true would not make sense here
	handleEvent(ctx, stats, event, rawEvent, false, false)
	return nil
}

func extractStringParam(param string) string {
	param = strings.TrimSpace(param)
	return param[1 : len(param)-1]
}

// ExecuteFunction executes a function on a given JSON document with given parameters.
func (f *JFunction) ExecuteFunction(ctx Context, doc *JDoc, params []string) interface{} {
	return f.fn(ctx, doc, params)
}

func fnHashMod(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 2 {
		ctx.Log().Error("error_type", "func_hashmod", "op", "hashmod", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to hashmod function"), "hashmod", params})
		return ""
	}

	ctx.Log().Debug("op", "hashmod", "params", params)

	str := extractStringParam(params[0])
	d, err := strconv.Atoi(extractStringParam(params[1]))
	if err != nil {
		ctx.Log().Error("error_type", "func_hashmod", "op", "hashmod", "cause", "invalid divisor", "params", params, "error", err)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("Invalid divisor parameter"), "hashmod", params})
		return ""
	}

	h := fnv.New32a()
	h.Write([]byte(str))
	partition := h.Sum32() % uint32(d)

	return fmt.Sprintf("%d", partition)
}

// fnHmac uses specified hash function to hash input with key
func fnHmac(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 3 {
		ctx.Log().Error("error_type", "func_hmac", "op", "hmac", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to hmac function"), "curl", params})
		return nil
	} else {
		hashFunc := extractStringParam(params[0])
		input := extractStringParam(params[1])
		key := extractStringParam(params[2])

		if hashFunc == "SHA1" {
			key_for_sign := []byte(key)
			h := hmac.New(sha1.New, key_for_sign)
			h.Write([]byte(input))
			return base64.StdEncoding.EncodeToString(h.Sum(nil))
		} else {
			ctx.Log().Error("error_type", "func_hmac", "op", "hmac", "cause", "hash_func_not_yet_support", "params", params)
			return nil
		}
	}

	return nil
}

// fnLoadFile reads file, and returns the content as string
func fnLoadFile(ctx Context, doc *JDoc, params []string) interface{} {
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if params == nil || len(params) != 1 || params[0] == "" {
		ctx.Log().Error("error_type", "func_lloadFile", "op", "loadFile", "cause", "wrong_number_of_parameters", "params", params)
		stats.IncErrors()
		AddError(ctx, SyntaxError{fmt.Sprintf("wrong number of parameters in call to loadFile function"), "curl", params})
		return nil
	} else {
		filename := extractStringParam(params[0])

		if _, err := os.Stat(filename); nil != err {
			ctx.Log().Error("error_type", "func_loadJson", "op", "loadFile", "cause", "error_access_file", "errror", err, "params", params)
			return nil
		}

		bs, err := ioutil.ReadFile(filename)
		if nil != err {
			ctx.Log().Error("error_type", "func_loadJson", "op", "loadFile", "cause", "error_read_file", "errror", err, "params", params)
			return nil
		}
		return string(bs)
	}

	return nil
}

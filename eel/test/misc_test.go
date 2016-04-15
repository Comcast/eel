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

package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/Comcast/eel/eel/handlers"
	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

var (
	event1 = `{
    "content": {
        "alert_end_time": "2015-11-10 00:03",
        "alert_start_time": "2015-11-10 00:00",
        "mac_address": "28cfda08c555",
        "message": "High WiFi",
        "phonenumber": "16501112222",
        "service_account": "1234567890",
        "severity": "P1",
		"device": "Mark's IPhone"
    },
    "expires": 0,
    "sequence": 1449629344335,
    "timestamp": 1449629344335,
	"sync": true
}`
	event2 = `{
    "content": {
        "alert_end_time": "2015-11-10 00:03",
        "alert_start_time": "2015-11-10 00:00",
        "mac_address": "28cfda08c555",
        "message": "High WiFi",
        "phonenumber": "16501112222",
        "service_account": "1234567890",
        "severity": "P1"
    },
    "expires": 0,
    "sequence": 1449629344335,
    "timestamp": 1449629344335,
	"sync": true
}`
)

func TestJDocEquals(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	e2, err := NewJDocFromString(event2)
	if err != nil {
		t.Fatal("could not get event2")
	}
	if !e1.Equals(e1) {
		t.Fatal("events not equal")
	}
	if e1.Equals(e2) {
		t.Fatal("events equal")
	}
	if !DeepEquals(e1.GetOriginalObject(), e1.GetOriginalObject()) {
		t.Fatal("events not deep equal")
	}
	if DeepEquals(e1.GetOriginalObject(), e2.GetOriginalObject()) {
		t.Fatal("events deep equal")
	}
}

func TestSimpleJPathExpressions(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	examples := [][]string{
		{"{{/content/service_account}}-{{/timestamp}}", "1234567890-1449629344335"},
		{"{{/content/service_account}}{{/timestamp}}", "12345678901449629344335"},
		{"{{/content/service_account}}", "1234567890"},
		{"foo", "foo"},
		{"{{/foo}}", ""},
		{"a{b", "a{b"},
		{"a}b", "a}b"},
		{"", ""},
		{"{{/sequence}}{{/sequence}}{{/sequence}}", "144962934433514496293443351449629344335"},
	}
	for i, e := range examples {
		if e1.ParseExpression(Gctx, e[0]) != e[1] {
			t.Fatalf("failed expression %d:\n%s\n%s\n%s\n", i, e[0], e[1], e1.ParseExpression(Gctx, e[0]))
		}
	}
}

var (
	badTransformation1 = `{
		"Version" : "1.0",
	  	"Name": "Bad Transformation 1",
		"Path" : "{{/content/accountId}}",
		"Active" : true,
		"TerminateOnMatch" : true,
		"Transformation" : {
			"{{/event}}":"{{/content",
			"{{/sync}}":true
		}
	}`
	badTransformation2 = `{
		"Version" : "1.0",
	  	"Name": "Bad Transformation 2",
		"Path" : "{{/content/accountId}}",
		"Active" : true,
		"TerminateOnMatch" : true,
		"Transformation" : {
			"{{/event}}":"{/content}}",
			"{{/sync}}":true
		}
	}`
)

func TestBadTransformations(t *testing.T) {
	initTests("../../config-handlers")
	thf := GetHandlerFactory(Gctx)
	var h HandlerConfiguration
	err := json.Unmarshal([]byte(badTransformation1), &h)
	if err != nil {
		t.Fatalf("could not parse json 1: %s\n", err.Error())
	}
	_, warnings := thf.GetHandlerConfigurationFromJson(Gctx, "", h)
	if len(warnings) == 0 {
		t.Fatal("invalid transformation 1 not detected")
	} else if len(warnings) > 3 {
		t.Fatalf("t1 has more warnings than expected: %v\n", warnings)
	}
	err = json.Unmarshal([]byte(badTransformation2), &h)
	if err != nil {
		t.Fatalf("could not parse json 2: %s\n", err.Error())
	}
	_, warnings = thf.GetHandlerConfigurationFromJson(Gctx, "", h)
	if len(warnings) == 0 {
		t.Fatal("invalid transformation 2 not detected")
	} else if len(warnings) > 3 {
		t.Fatalf("t2 has more warnings than expected: %v\n", warnings)
	}
}

func TestHttpPublisher(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	handlers := GetHandlerFactory(Gctx).GetHandlersForEvent(Gctx, e1)
	if handlers == nil || len(handlers) != 1 {
		t.Fatal("could not get handler")
	}
	publishers, err := handlers[0].ProcessEvent(Gctx, e1)
	if err != nil {
		t.Fatal("could not transform event")
	}
	if len(publishers) != 1 {
		t.Fatal("wrong number of publishers")
	}
	if publishers[0].GetHeaders()["X-Tenant-Id"] != "tenant1" {
		t.Fatal("bad tenant id %s expected %s\n", publishers[0].GetHeaders()["X-Tenant-Id"], "tenant1")
	}
	if publishers[0].GetUrl() != "http://localhost:8088" {
		t.Fatalf("bad url %s expected %s\n", publishers[0].GetUrl(), "http://localhost:8088")
	}
	if publishers[0].GetVerb() != "POST" {
		t.Fatalf("bad verb %s expected %s\n", publishers[0].GetVerb(), "POST")
	}
	if !publishers[0].GetPayloadParsed().Equals(e1) {
		t.Fatalf("bad payload %s expected %s\n", publishers[0].GetPayload(), event1)
	}
}

func TestParserFormat(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	expected := "Tue Aug 18 17:35:56 2015"
	test := "{{format('1439937356000','Mon Jan 2 15:04:05 2006','EST')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	if result.(string) != expected {
		t.Fatalf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserJoin(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	expected, err := NewJDocFromString("{\"a\":\"1\", \"b\":\"2\"}")
	if err != nil {
		t.Errorf("could not load expected response: %s\n", err.Error())
	}
	test := "{{join('{\"a\":\"1\"}','{\"b\":\"2\"}')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	res, err := NewJDocFromMap(result.(map[string]interface{}))
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	if !res.Equals(expected) {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserArrayPathAsEval(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{eval('/content/message')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "High WiFi"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEscape(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "foo${{bar.baz$}}boo"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo{{bar.baz}}boo"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEscapeTwo(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "${{foo$}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "{{foo}}"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEqualsTrue(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{equals('" + e1.StringPretty() + "')}}"
	ast, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := ast.Execute(Gctx, e1)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEqualsFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{equals('" + event2 + "')}}"
	ast, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := ast.Execute(Gctx, e1)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEqualsTrueWithSecondParamString(t *testing.T) {
	initTests("../../config-handlers")
	test := "{{equals('carthaginem esse delendam', 'carthaginem esse delendam')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, nil)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserEqualsFalseWithSecondParamString(t *testing.T) {
	initTests("../../config-handlers")
	test := "{{equals('carthaginem esse delendam', 'carthaginem non esse delendam')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, nil)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserTrue(t *testing.T) {
	initTests("../../config-handlers")
	test := "{{true()}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, nil)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserFalse(t *testing.T) {
	initTests("../../config-handlers")
	test := "{{false()}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, nil)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserIfteFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{ifte('{{not('{{equals('{{/content/message}}','High WiFi')}}')}}','status is {{/content/message}}','donno nothing')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "donno nothing"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserIfteTrue(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{ifte('{{equals('{{/content/message}}','High WiFi')}}','status is {{/content/message}}','donno nothing')}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "status is High WiFi"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserContainsTrue(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{and('{{contains('{"content":{"message":"High WiFi" } }','{{/}}')}}','{{contains('{"content":{"phonenumber": "16501112222" } }','{{/}}')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserContainsFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{and('{{contains('{"content":{"message":"High WiFi" } }','{{/}}')}}','{{contains('{"content":{"phonenumber": "16501112223" } }','{{/}}')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserOrTrue(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{or('false', '{{ident('true')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserOrFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{or('false', '{{ident('false')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserNotFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{not('{{ident('true')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserRegex2(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{regex('barfoobar', 'f.o')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserAndTrue(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{and('true', '{{ident('true')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserAndFalse(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{and('true', '{{ident('false')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := false
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

var (
	nest = `{
		"event" : {
			"value" : "away",
			"timestamp" : 1426630898428
		}
	}`
)

func TestNotification2(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(nest)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{
		"topic" : "foo/bar",
		"payload" : "Hey Everybody! The Nest was just set to {{/event/value}} at {{js('result = new Date({{/event/timestamp}}).toUTCString()', 'result')}}. I hope that's cool."
	}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := `{
		"topic" : "foo/bar",
		"payload" : "Hey Everybody! The Nest was just set to away at Tue, 17 Mar 2015 22:21:38 UTC. I hope that's cool."
	}`
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

var (
	iot = `{
	    "content": {
	        "_links": {
	            "iot:account": {
	                "href": "https://hub.int.iot.comcast.net/client/account/ABCDEFG123"
	            }
	        }
		}
	}`
)

func TestParserJS3(t *testing.T) {
	// note that this test is functionally identical to TestParserRegex
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(iot)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{js('patt = new RegExp("[A-Z0-9]+"); result = patt.exec(input);', 'result', 'input', '{{/content/_links/iot:account/href}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "ABCDEFG123"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserRegex(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(iot)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{regex('{{/content/_links/iot:account/href}}','[A-Z0-9]+')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "ABCDEFG123"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserSelectBool(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{/sync}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := true
	if result.(bool) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserJS(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{js('result = 40+2; result += 2;', 'result')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := 44
	if result.(int) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserJS2(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{js('result = input', 'result', 'input', '42')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "42"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestEvalSpaceIncluded(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{eval('/content/fo: o','{"content": {"fo: o":"bar"} }')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "bar"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserIllegalPath(t *testing.T) {
	initTests("../../config-handlers")
	test := `{{event}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal path %s\n", test)
	}
}

func TestParserIllegalPath2(t *testing.T) {
	initTests("../../config-handlers")
	test := `{/event}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal path %s\n", test)
	}
}

func TestParserIllegalFunction(t *testing.T) {
	initTests("../../config-handlers")
	test := `{{foo()}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal function %s\n", test)
	}
}

func TestParserInvalidFunction(t *testing.T) {
	initTests("../../config-handlers")
	test := `{{alt('')}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal function %s\n", test)
	}
}

func TestParserInvalidFunction2(t *testing.T) {
	initTests("../../config-handlers")
	test := `{{alt('', ''}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal function %s\n", test)
	}
}

func TestParserInvalidFunction3(t *testing.T) {
	initTests("../../config-handlers")
	test := `{{alt('', ''))}}`
	_, err := NewJExpr(test)
	if err == nil {
		t.Errorf("missing error for illegal function %s\n", test)
	}
}

func TestParserSelectHeader(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{header('mykey')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	// error case 1
	result := jexpr.Execute(Gctx, e1)
	expected := ""
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
	// error case 2
	test2 := `{{header('mykey')}}`
	jexpr, err = NewJExpr(test2)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	Gctx.AddValue("header", "not_a_header_type")
	expected = ""
	result = jexpr.Execute(Gctx, e1)
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
	// normal case
	test3 := `{{header('mykey')}}`
	jexpr, err = NewJExpr(test3)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	header := make(http.Header)
	header.Add("mykey", "myvalue")
	Gctx.AddValue(EelRequestHeader, header)
	expected = "myvalue"
	result = jexpr.Execute(Gctx, e1)
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
	// error case 3
	test4 := `{{header('unknownKey')}}`
	jexpr, err = NewJExpr(test4)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result = jexpr.Execute(Gctx, e1)
	expected = ""
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserSelectNumber(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{/timestamp}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := 1449629344335
	if result.(int) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserConstant(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `hello world`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	if result.(string) != test {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, test)
	}
}

func TestParserConstantLB(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `hello
	world`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	if result.(string) != test {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, test)
	}
}

func TestParserSelectJson(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{/}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result, err := NewJDocFromMap(jexpr.Execute(Gctx, e1).(map[string]interface{}))
	if err != nil {
		t.Errorf("error parsing: %s\n", err.Error())
	}
	if !e1.Equals(result) {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, event1)
	}
}

func TestParserSelectJsonAlt(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{eval('/')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result, err := NewJDocFromMap(jexpr.Execute(Gctx, e1).(map[string]interface{}))
	if err != nil {
		t.Errorf("error parsing: %s\n", err.Error())
	}
	if !e1.Equals(result) {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, event1)
	}
}

func TestParserSelectString(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{/content/message}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "High WiFi"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

func TestParserSelectStringAlt(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{eval('/content/message')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "High WiFi"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

func TestParserWhiteSpace(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := ""
	ts := httptest.NewServer(http.HandlerFunc(FortyTwoJsonHandler))
	defer ts.Close()
	props := GetConfig(Gctx).CustomProperties
	props["ServiceUrl"] = ts.URL
	test = `foo-
					{{eval(
						'/accountId',
						'{{curl(
							'POST',
							'{{prop(
								'ServiceUrl'
							)}}',
							'',
							'',
							''
						)}}'
					)}}
					-
					{{ident(
						'bar'
					)}}
					-bar`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo-42-bar-bar"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

func TestParserDeepNesting(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := `{{ident('{{ident('{{ident('bar')}}')}}')}}`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("parsing error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "bar"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

func TestParserCurl(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := ""
	ts := httptest.NewServer(http.HandlerFunc(FortyTwoJsonHandler))
	defer ts.Close()
	props := GetConfig(Gctx).CustomProperties
	props["ServiceUrl"] = ts.URL
	test = `foo-{{curl('POST', '{{prop('ServiceUrl')}}', '{ "query" : "{{/content/accountId}}-foo"}', '/accountId')}}-{{ident('bar')}}-bar`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo-42-bar-bar"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

func TestParserCurl2(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(event1)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := ""
	ts := httptest.NewServer(http.HandlerFunc(FortyTwoJsonHandler))
	defer ts.Close()
	props := GetConfig(Gctx).CustomProperties
	props["ServiceUrl"] = ts.URL
	test = `foo-{{eval('/accountId','{{curl('POST', '{{prop('ServiceUrl')}}', '', '', '')}}')}}-{{ident('bar')}}-bar`
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo-42-bar-bar"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %s\n", result, expected)
	}
}

var (
	nest2 = `{
		"content": {
			"_embedded": {
				"iot:states": {
					"states": [
						{ "name": "foo", "key": "123"},
						{ "name": "bar", "key": "456"}
					]
				}
			}
		}
	}
	`
)

func TestParserArrayPath(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(nest2)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{/content/_embedded/iot:states/states[0]/name}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "foo"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserArrayPathByKey(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(nest2)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{/content/_embedded/iot:states/states[name=bar]/key}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := "456"
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserArrayPathIdxOutOfBounds(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(nest2)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{/content/_embedded/iot:states/states[3]/name}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := ""
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

func TestParserArrayPathMissingChild(t *testing.T) {
	initTests("../../config-handlers")
	e1, err := NewJDocFromString(nest2)
	if err != nil {
		t.Fatal("could not get event1")
	}
	test := "{{/content/_embedded/iot:states/states[0]/missing}}"
	jexpr, err := NewJExpr(test)
	if err != nil {
		t.Errorf("error: %s\n", err.Error())
	}
	result := jexpr.Execute(Gctx, e1)
	expected := ""
	if result.(string) != expected {
		t.Errorf("wrong parsing result: %v expected: %v\n", result, expected)
	}
}

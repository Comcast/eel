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
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/Comcast/eel/eel/eellib"
	. "github.com/Comcast/eel/eel/handlers"
	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

func TestEELLibrary(t *testing.T) {
	initTests("../../config-handlers")
	in, err := NewJDocFromString(`{ "message" : "hello world!!!" }`)
	if err != nil {
		t.Fatalf("could not parse in event: %s\n", err.Error())
	}
	out, err := NewJDocFromString(`{ "event" : { "message" : "hello world!!!" }}`)
	if err != nil {
		t.Fatalf("could not parse out event: %s\n", err.Error())
	}
	tf, err := NewJDocFromString(`{ "{{/event}}" : "{{/}}" }`)
	if err != nil {
		t.Fatalf("could not parse tranformation: %s\n", err.Error())
	}
	transformed := in.ApplyTransformation(Gctx, tf)
	if transformed == nil {
		t.Fatalf("transformationfailed")
	}
	if !transformed.Equals(out) {
		t.Fatalf("actual and expected event differ:\nactual:\n%s\nexpected:\n%s\nin:\n%s\nt:\n%s\n", transformed.StringPretty(), out.StringPretty(), in.StringPretty(), tf.StringPretty())
	}
}

func TestEELLibrary2(t *testing.T) {
	// initialize context
	ctx := NewDefaultContext(L_InfoLevel)
	Gctx = ctx
	ctx.AddLogValue("app.id", "myapp")
	eelSettings := new(EelSettings)
	ctx.AddConfigValue(EelConfig, eelSettings)
	eelServiceStats := new(ServiceStats)
	ctx.AddValue(EelTotalStats, eelServiceStats)
	InitHttpTransport(ctx)
	// load handlers from folder: note parameter is an array of one or more folders
	eelHandlerFactory, warnings := NewHandlerFactory(ctx, []string{"../../config-handlers"})
	// check if parsing handlers caused warnings
	for _, w := range warnings {
		t.Logf("warning loading handlers: %s\n", w)
	}
	// prepare incoming test event
	in, err := NewJDocFromString(`{ "message" : "hello world!!!" }`)
	if err != nil {
		t.Fatalf("could not parse in event: %s\n", err.Error())
	}
	// find matching handlers
	eelMatchingHandlers := eelHandlerFactory.GetHandlersForEvent(ctx, in)
	if len(eelMatchingHandlers) == 0 {
		t.Fatalf("no matching handlers")
	}
	if len(eelMatchingHandlers) > 1 {
		t.Errorf("too many matching handlers")
	}
	// process event and get publisher objects in return - typically we expect exactly one publisher (unless event was filtered)
	eelPublishers, err := eelMatchingHandlers[0].ProcessEvent(ctx, in)
	if err != nil {
		t.Fatalf("error transforming event: %s\n", err.Error())
	}
	if len(eelPublishers) == 0 {
		t.Fatalf("no publishers (event filtered?)")
	}
	if len(eelPublishers) > 1 {
		t.Errorf("too many publihsers (fanout intended?)")
	}
	// we don't want to actually publish the event, we are just interested in the payload
	//outStr := eelPublishers[0].GetPayload()
	//t.Logf("transformed payload as string: %s\n", outStr)
	outObj := eelPublishers[0].GetPayloadParsed().GetOriginalObject()
	//t.Logf("transformed payload as object: %v\n", outObj)
	// we are using the default identity transformation, so in and out event should be identical
	if !DeepEquals(outObj, in.GetOriginalObject()) {
		t.Fatalf("unexpected transformation result")
	}
	//resp, err := eelPublishers[0].Publish()
	//if err != nil {
	//	t.Fatalf("could not publish event: %s\n", err.Error())
	//}
	//t.Logf("response: %s\n", resp)
}

func TestEELLibrary3Sequential(t *testing.T) {
	// initialize context
	ctx := NewDefaultContext(L_InfoLevel)
	EELInit(ctx)
	// load handlers from folder: note parameter is an array of one or more folders
	eelHandlerFactory, warnings := EELNewHandlerFactory(ctx, "../../config-handlers")
	// check if parsing handlers caused warnings
	for _, w := range warnings {
		t.Logf("warning loading handlers: %s\n", w)
	}
	// prepare incoming test event
	in := map[string]interface{}{
		"message": "hello world!!!",
	}
	// process event and get publisher objects in return - typically we expect exactly one publisher (unless event was filtered)
	outs, errs := EELTransformEvent(ctx, in, eelHandlerFactory)
	if errs != nil {
		t.Fatalf("could not transform event: %v\n", errs)
	}
	if len(outs) != 1 {
		t.Fatalf("unexpected number of results: %d\n", len(outs))
	}
	if !DeepEquals(outs[0], in) {
		t.Fatalf("unexpected transformation result\n")
	}
}

func TestEELLibrary3Concurrent(t *testing.T) {
	// initialize context
	ctx := NewDefaultContext(L_InfoLevel)
	EELInit(ctx)
	// load handlers from folder: note parameter is an array of one or more folders
	eelHandlerFactory, warnings := EELNewHandlerFactory(ctx, "../../config-handlers")
	// check if parsing handlers caused warnings
	for _, w := range warnings {
		t.Logf("warning loading handlers: %s\n", w)
	}
	// prepare incoming test event
	in := map[string]interface{}{
		"message": "hello world!!!",
	}
	// process event and get publisher objects in return - typically we expect exactly one publisher (unless event was filtered)
	outs, errs := EELTransformEventConcurrent(ctx, in, eelHandlerFactory)
	if errs != nil {
		t.Fatalf("could not transform event: %v\n", errs)
	}
	if len(outs) != 1 {
		t.Fatalf("unexpected number of results: %d\n", len(outs))
	}
	if !DeepEquals(outs[0], in) {
		t.Fatalf("unexpected transformation result\n")
	}
}

func TestEELLibrary4(t *testing.T) {
	ctx := NewDefaultContext(L_InfoLevel)
	in := `{ "message" : "hello world!!!" }`
	expected, err := NewJDocFromString(`{ "event" : { "message" : "hello world!!!" }}`)
	if err != nil {
		t.Fatalf("could not parse expected event: %s\n", err.Error())
	}
	transformation := `{ "{{/event}}" : "{{/}}" }`
	EELInit(ctx)
	out, errs := EELSimpleTransform(ctx, in, transformation, false)
	if errs != nil {
		t.Fatalf("bad tranformation: %v\n", errs)
	}
	outDoc, err := NewJDocFromString(out)
	if err != nil {
		t.Fatalf("could not parse out event: %s\n", err.Error())
	}
	if !DeepEquals(expected.GetOriginalObject(), outDoc.GetOriginalObject()) {
		t.Fatalf("unexpected transformation result: %s\n", out)
	}
}

func TestEELLibrary5(t *testing.T) {
	ctx := NewDefaultContext(L_InfoLevel)
	//in := `{ "message" : "hello world!!!" }`
	in := ``
	// note: results of type string are surrounded by double quotes
	expected := `"xyz"`
	expr := `{{ident('xyz')}}`
	EELInit(ctx)
	out, errs := EELSimpleEvalExpression(ctx, in, expr)
	if errs != nil {
		t.Fatalf("bad tranformation: %v\n", errs)
	}
	if out != expected {
		t.Fatalf("unexpected eval result: %s expected: %s\n", out, expected)
	}
}

func TestEELLibraryError(t *testing.T) {
	ctx := NewDefaultContext(L_InfoLevel)
	in := `{}`
	transformation := `{ "{{/event}}" : "{{curl('GET','http://x.y.z')}}" }`
	EELInit(ctx)
	eelSettings, err := EELGetSettings(ctx)
	if err != nil {
		t.Fatalf("error getting settings: %s\n", err.Error())
	}
	eelSettings.MaxAttempts = 3
	eelSettings.InitialDelay = 125
	eelSettings.InitialBackoff = 1000
	eelSettings.BackoffMethod = "Exponential"
	eelSettings.HttpTimeout = 1000
	eelSettings.ResponseHeaderTimeout = 1000
	EELUpdateSettings(ctx, eelSettings)
	_, errs := EELSimpleTransform(ctx, in, transformation, false)
	if errs == nil {
		t.Fatalf("no errors!\n")
	}
	if len(errs) != 1 {
		t.Fatalf("unexpected number of errors: %d\n", len(errs))
	}
	//expectedError := "error reaching endpoint: http://x.y.z: status: 0 message: Get http://x.y.z: dial tcp: lookup x.y.z: no such host"
	if !strings.Contains(errs[0].Error(), "error reaching endpoint") {
		t.Fatalf("unexpecte error: %s\n", errs[0].Error())
	}
}

func TestDontTouchEvent(t *testing.T) {
	initTests("data/test00/handlers")
	transformEvent(t, "data/test00/", nil)
}

func TestCanonicalizeEvent(t *testing.T) {
	initTests("data/test01/handlers")
	transformEvent(t, "data/test01/", nil)
}

/*func TestInjectExternalServiceResponse(t *testing.T) {
	initTests("data/test02/handlers")
	transformEvent(t, "data/test02/", nil)
}*/

func TestInjectExternalServiceResponseSim(t *testing.T) {
	initTests("data/test26/handlers")
	ts0 := httptest.NewServer(http.HandlerFunc(FortyTwoJsonHandler))
	defer ts0.Close()
	GetHandlerFactory(Gctx).GetAllHandlers(Gctx)[0].CustomProperties["42aas_api"] = ts0.URL
	transformEvent(t, "data/test26/", nil)
}

func TestTransformationByExample(t *testing.T) {
	initTests("data/test03/handlers")
	transformEvent(t, "data/test03/", nil)
}

func TestNamedTransformations(t *testing.T) {
	initTests("data/test04/handlers")
	transformEvent(t, "data/test04/", nil)
}

func TestMessageGeneration(t *testing.T) {
	initTests("data/test05/handlers")
	transformEvent(t, "data/test05/", nil)
}

func TestTerminateOnMatchTrue(t *testing.T) {
	initTests("data/test06/handlers")
	transformEvent(t, "data/test06/", nil)
}

func TestTerminateOnMatchFalse(t *testing.T) {
	initTests("data/test07/handlers")
	fanoutEvent(t, "data/test07/", 3, false, nil)
}

func TestMultiTenancy(t *testing.T) {
	initTests("data/test08/handlers")
	fanoutEvent(t, "data/test08/", 2, false, nil)
}

func TestSequentialHandlerCascade(t *testing.T) {
	initTests("data/test09/handlers")
	fanoutEvent(t, "data/test09/", 3, true, nil)
}

func TestJavaScript(t *testing.T) {
	initTests("data/test10/handlers")
	transformEvent(t, "data/test10/", nil)
}

func TestMatchByExample(t *testing.T) {
	initTests("data/test11/handlers")
	transformEvent(t, "data/test11/", nil)
}

func TestHeaders(t *testing.T) {
	initTests("data/test12/handlers")
	transformEvent(t, "data/test12/", map[string]string{"X-MyKey": "xyz"})
}

func TestCustomProperties(t *testing.T) {
	initTests("data/test13/handlers")
	transformEvent(t, "data/test13/", nil)
}

func TestFanOut(t *testing.T) {
	initTests("data/test14/handlers")
	fanoutEvent(t, "data/test14/", 4, false, nil)
}

func TestStringOps(t *testing.T) {
	initTests("data/test15/handlers")
	transformEvent(t, "data/test15/", nil)
}

func TestNamedTransformations2(t *testing.T) {
	initTests("data/test16/handlers")
	transformEvent(t, "data/test16/", nil)
}

func TestContains(t *testing.T) {
	initTests("data/test17/handlers")
	transformEvent(t, "data/test17/", nil)
}

func TestMultiTenency2(t *testing.T) {
	initTests("data/test18/handlers")
	fanoutEvent(t, "data/test18/", 1, false, map[string]string{"X-TenantId": "tenant1"})
}

func TestMessageGeneration2(t *testing.T) {
	initTests("data/test19/handlers")
	transformEvent(t, "data/test19/", nil)
}

func TestRegex(t *testing.T) {
	initTests("data/test20/handlers")
	transformEvent(t, "data/test20/", nil)
}

func TestExternalLookupFailure(t *testing.T) {
	initTests("data/test21/handlers")
	transformEvent(t, "data/test21/", nil)
}

func TestFilterByPath(t *testing.T) {
	initTests("data/test22/handlers")
	fanoutEvent(t, "data/test22/", 0, false, nil)
}

func TestFilterByExample(t *testing.T) {
	initTests("data/test23/handlers")
	fanoutEvent(t, "data/test23/", 0, false, nil)
}

func TestChoiceOfValuesMatch(t *testing.T) {
	initTests("data/test24/handlers")
	transformEvent(t, "data/test24/", nil)
}

func TestArrayPathSelector(t *testing.T) {
	initTests("data/test25/handlers")
	transformEvent(t, "data/test25/", nil)
}

func TestNamedTransformationsAndArrays(t *testing.T) {
	initTests("data/test27/handlers")
	transformEvent(t, "data/test27/", nil)
}

func TestNamedTransformationsAndArrays2(t *testing.T) {
	initTests("data/test28/handlers")
	transformEvent(t, "data/test28/", nil)
}

func TestNamedTransformationsAndArrays3(t *testing.T) {
	initTests("data/test29/handlers")
	transformEvent(t, "data/test29/", nil)
}

func TestNamedTransformationsAndArrays4(t *testing.T) {
	initTests("data/test30/handlers")
	transformEvent(t, "data/test30/", nil)
}

func TestNamedTransformationsAndArrays5(t *testing.T) {
	initTests("data/test31/handlers")
	transformEvent(t, "data/test31/", nil)
}

func TestSimpleTypes(t *testing.T) {
	initTests("data/test32/handlers")
	transformEvent(t, "data/test32/", nil)
}

func TestSimpleTypes2(t *testing.T) {
	initTests("data/test33/handlers")
	transformEvent(t, "data/test33/", nil)
}

func TestJoin(t *testing.T) {
	initTests("data/test34/handlers")
	transformEvent(t, "data/test34/", nil)
}

func TestJoin2(t *testing.T) {
	initTests("data/test35/handlers")
	transformEvent(t, "data/test35/", nil)
}

func TestJoin3(t *testing.T) {
	initTests("data/test36/handlers")
	transformEvent(t, "data/test36/", nil)
}

/*func TestJoin4(t *testing.T) {
	initTests("data/test37/handlers")
	transformEvent(t, "data/test37/", nil)
}*/

func TestMatchArrays(t *testing.T) {
	initTests("data/test38/handlers")
	transformEvent(t, "data/test38/", nil)
}

func TestMatchArrays2(t *testing.T) {
	initTests("data/test39/handlers")
	fanoutEvent(t, "data/test39/", 0, false, nil)
}

/*func TestRulesDeleteLocation(t *testing.T) {
	initTests("data/test40/handlers")
	transformEvent(t, "data/test40/", nil)
}*/

func TestMatchArrays3(t *testing.T) {
	initTests("data/test41/handlers")
	transformEvent(t, "data/test41/", nil)
}

func TestTerminateOnMatchTrueByExample(t *testing.T) {
	initTests("data/test42/handlers")
	transformEvent(t, "data/test42/", nil)
}

func TestMatchChoiceOfValues2(t *testing.T) {
	initTests("data/test43/handlers")
	transformEvent(t, "data/test43/", nil)
}

func TestMatchChoiceOfValues3(t *testing.T) {
	initTests("data/test44/handlers")
	transformEvent(t, "data/test44/", nil)
}

func TestMatchChoiceOfValues4(t *testing.T) {
	initTests("data/test45/handlers")
	fanoutEvent(t, "data/test45/", 0, false, nil)
}

func TestFilterCascade(t *testing.T) {
	initTests("data/test46/handlers")
	fanoutEvent(t, "data/test46/", 0, false, nil)
}

func TestChooseArrayElements(t *testing.T) {
	initTests("data/test47/handlers")
	transformEvent(t, "data/test47/", nil)
}

func TestNamedTransformationWithArraysAndPattern(t *testing.T) {
	initTests("data/test48/handlers")
	transformEvent(t, "data/test48/", nil)
}

func TestNamedTransformationWithArraysAndJoin(t *testing.T) {
	initTests("data/test49/handlers")
	transformEvent(t, "data/test49/", nil)
}

/*func TestAnOddTransformation(t *testing.T) {
	initTests("data/test50/handlers")
	transformEvent(t, "data/test50/", nil)
}*/

func TestML(t *testing.T) {
	initTests("data/test51/handlers")
	transformEvent(t, "data/test51/", nil)
}

func TestML2(t *testing.T) {
	initTests("data/test52/handlers")
	transformEvent(t, "data/test52/", nil)
}

func TestCrush(t *testing.T) {
	initTests("data/test53/handlers")
	transformEvent(t, "data/test53/", nil)
}

func TestTrueConditional(t *testing.T) {
	initTests("data/test54/handlers")
	transformEvent(t, "data/test54/", nil)
}

func TestTrueConditionalCase(t *testing.T) {
	initTests("data/test55/handlers")
	transformEvent(t, "data/test55/", nil)
}

func TestTopicHandlerParent(t *testing.T) {
	initTests("data/test97/handlers")
	fanoutEvent(t, "data/test97/", 1, false, nil)
}

func TestTopicHandlerWildCard(t *testing.T) {
	initTests("data/test98/handlers")
	fanoutEvent(t, "data/test98/", 1, false, nil)
}

func TestTopicHandlerDoubleWildCard(t *testing.T) {
	initTests("data/test99/handlers")
	fanoutEvent(t, "data/test99/", 1, false, nil)
}

func generateEndpointList(originalEndpoints interface{}, mockEndpoint string) []interface{} {
	res := make([]interface{}, 0)
	length := 0
	switch originalEndpoints.(type) {
	case []interface{}:
		res = append(res, originalEndpoints.([]interface{})...)
		length += len(originalEndpoints.([]interface{}))
	case string:
		res = append(res, originalEndpoints)
		length++
	}
	for i := 0; i < length; i++ {
		res = append(res, mockEndpoint)
	}
	Gctx.Log().Info("length", length, "endpoints", res)
	return res
}

func transformEvent(t *testing.T, folder string, headers map[string]string) {
	// mock event
	event, err := NewJDocFromFile(filepath.Join(folder, "in.json"))
	if err != nil {
		t.Fatalf("could not read in event: %s\n", err.Error())
	}
	// rebind event service url
	ts0 := httptest.NewServer(http.HandlerFunc(PostHandler))
	defer ts0.Close()
	GetConfig(Gctx).Endpoint = generateEndpointList(GetConfig(Gctx).Endpoint, ts0.URL)
	hf := GetHandlerFactory(Gctx)
	for _, h := range hf.GetAllHandlers(Gctx) {
		if h.Endpoint != "" {
			h.Endpoint = generateEndpointList(h.Endpoint, ts0.URL)
		}
	}
	// register event service handler
	ts1 := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts1.Close()
	// simulate event
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts1.URL, bytes.NewBufferString(event.String()))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event.String())))
	if headers != nil {
		for k, v := range headers {
			r.Header.Add(k, v)
		}
	}
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting elements event: %s\n", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	_, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read event response: %s\n", err.Error())
	}
	// give post hanlder a chance to store event
	delayMS := 100
	iter := 0
	for {
		time.Sleep(time.Duration(delayMS) * time.Millisecond)
		iter++
		if Gctx.Value("debug_post_body") != nil {
			break
		}
		if iter > 50 {
			break
		}
	}
	// read and compare received event
	eel := Gctx.Value("debug_post_body")
	if eel == nil {
		t.Fatalf("no response")
	}
	receivedEvent, err := NewJDocFromString(string(eel.([]byte)))
	if err != nil {
		t.Fatalf("bad json in response")
	}
	buf, err := ioutil.ReadFile(filepath.Join(folder, "out.json"))
	if err != nil {
		t.Fatalf("could not read out event")
	}
	expectedEvent, err := NewJDocFromString(string(buf))
	if err != nil {
		t.Fatalf("bad json in expected response")
	}
	if !expectedEvent.Equals(receivedEvent) {
		t.Fatalf("actual and expected event differ:\n%s\n%s\n", receivedEvent.StringPretty(), expectedEvent.StringPretty())
	}
}

func fanoutEvent(t *testing.T, folder string, numExpectedResults int, recursive bool, headers map[string]string) {
	// mock event
	event, err := NewJDocFromFile(filepath.Join(folder, "in.json"))
	if err != nil {
		t.Fatalf("could not read in event: %s\n", err.Error())
	}
	// rebind event service url
	ts0 := httptest.NewServer(http.HandlerFunc(PostHandler))
	defer ts0.Close()
	// register event service handler
	ts1 := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts1.Close()
	// bend urls for testing
	GetConfig(Gctx).Endpoint = generateEndpointList(GetConfig(Gctx).Endpoint, ts0.URL)
	if recursive {
		GetConfig(Gctx).Endpoint = append(GetConfig(Gctx).Endpoint.([]interface{}), ts1.URL)
	}
	hf := GetHandlerFactory(Gctx)
	for _, h := range hf.GetAllHandlers(Gctx) {
		if h.Endpoint != nil {
			h.Endpoint = generateEndpointList(h.Endpoint, ts0.URL)
			if recursive {
				h.Endpoint = append(h.Endpoint.([]interface{}), ts1.URL)
			}
		}
		Gctx.Log().Info("handler", h)
	}
	// simulate event
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts1.URL, bytes.NewBufferString(event.String()))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event.String())))
	if headers != nil {
		for k, v := range headers {
			r.Header.Add(k, v)
		}
	}
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting elements event: %s\n", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	_, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read event response: %s\n", err.Error())
	}
	// give post hanlder a chance to store event
	delayMS := 100
	iter := 0
	for {
		time.Sleep(time.Duration(delayMS) * time.Millisecond)
		iter++
		if Gctx.Value("debug_post_body_array") != nil && len(Gctx.Value("debug_post_body_array").([][]byte)) >= numExpectedResults {
			break
		}
		if iter > 50 {
			break
		}
	}
	// read and compare received event
	eel := Gctx.Value("debug_post_body_array")
	if eel == nil && numExpectedResults > 0 {
		t.Fatalf("no response")
	}
	numEvents := 0
	if eel != nil {
		numEvents = len(eel.([][]byte))
	}
	if numExpectedResults != numEvents {
		t.Fatalf("actual and expected number of event differ:\n%d - %d\n", numEvents, numExpectedResults)
	}
	// load expected events into map
	expectedEvents := make(map[int]*JDoc, 0)
	for i := 0; i < numExpectedResults; i++ {
		buf, err := ioutil.ReadFile(filepath.Join(folder, "out"+strconv.Itoa(i)+".json"))
		if err != nil {
			t.Fatalf("could not read out%d event", i)
		}
		expectedEvent, err := NewJDocFromString(string(buf))
		if err != nil {
			t.Fatalf("bad json in expected response %d", i)
		}
		expectedEvents[i] = expectedEvent
	}
	// check if all events are present
	for i := 0; i < numExpectedResults; i++ {
		receivedEvent, err := NewJDocFromString(string(eel.([][]byte)[i]))
		if err != nil {
			t.Fatalf("bad json in response")
		}
		for k, v := range expectedEvents {
			if v.Equals(receivedEvent) {
				delete(expectedEvents, k)
				break
			}
		}
		if len(expectedEvents) != numExpectedResults-(i+1) {
			t.Fatalf("could not find response out %d\n", i)
		}
	}
}

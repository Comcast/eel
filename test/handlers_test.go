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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	. "github.com/Comcast/eel/jtl"
	. "github.com/Comcast/eel/util"
)

func TestHealthCheckHandler(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(StatusHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("GET", ts.URL, nil)
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "200 OK" {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	status, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(status) == 0 {
		t.Fatalf("wrong length of status response: %d\n", len(status))
	}
	statusJSON, err := NewJDocFromString(string(status))
	if err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	if statusJSON.ParseExpression(Gctx, "{{/Version}}") != "1.0" {
		t.Fatalf("wrong version: expected: %s received: %s\n", "1.0", statusJSON.ParseExpression(Gctx, "{{/Version}}"))
	}
}

func TestReloadHandler(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(ReloadConfigHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("GET", ts.URL, nil)
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	status, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(status) == 0 {
		t.Fatalf("wrong length of status response: %d\n", len(status))
	}
	statusJSON, err := NewJDocFromString(string(status))
	if err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	if statusJSON.ParseExpression(Gctx, "{{/Version}}") != "1.0" {
		t.Fatalf("wrong version: expected: %s received: %s\n", "1.0", statusJSON.ParseExpression(Gctx, "{{/Version}}"))
	}
}

func TestVetHandler(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(VetHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("GET", ts.URL, nil)
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	status, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(status) == 0 {
		t.Fatalf("wrong length of status response: %d\n", len(status))
	}
	statusJSON, err := NewJDocFromString(string(status))
	if err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	if statusJSON.ParseExpression(Gctx, "{{/status}}") != "ok" {
		t.Fatalf("bad vetting: %s\n", string(status))
	}
}

func TestNopHandler(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(NilHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("GET", ts.URL, nil)
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "200 OK" {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	status, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(status) == 0 {
		t.Fatalf("wrong length of status response: %d\n", len(status))
	}
	statusJSON, err := NewJDocFromString(string(status))
	if err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	if statusJSON.ParseExpression(Gctx, "{{/status}}") != "ok" {
		t.Fatalf("bad vetting: %s\n", string(status))
	}
}

var (
	testEvent = `{
	    "foo": "bar"
	}`
	testDebugResponse = `{
		"api": "http",
		"handler": "Default",
		"tenant.id": "tenant1",
		"trace.in.data": {
			"foo": "bar"
		},
		"trace.out.data": {
			"foo": "bar"
		},
		"trace.out.endpoint": "http://localhost:8088",
		"trace.out.headers": {
			"X-B3-TraceId": "276ba4a9-c425-4a96-9353-63705042c734",
			"X-Tenant-Id": "tenant1"
		},
		"trace.out.partition": 0,
		"trace.out.topic": "",
		"trace.out.path": "",
		"trace.out.protocol": "http",
		"trace.out.url": "http://localhost:8088",
		"trace.out.verb": "POST",
		"tx.id": "276ba4a9-c425-4a96-9353-63705042c734",
		"tx.traceId": "276ba4a9-c425-4a96-9353-63705042c734"
	}`
)

func TestEventHandlerDebug(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(testEvent))
	r.Header.Add("X-Debug", "true")
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(testEvent)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(bs) < len(testDebugResponse)/2 {
		t.Fatalf("wrong length of eel response: expected %d received %d\n response:%s\n", len(testDebugResponse), len(bs), string(bs))
	}
	responseStr := string(bs)
	var events []interface{}
	if err := json.Unmarshal([]byte(responseStr), &events); err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	event := events[0].(map[string]interface{})
	var expected map[string]interface{}
	if err := json.Unmarshal([]byte(testDebugResponse), &expected); err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	// transfer randomly generated tx.id to expected response
	expected["tx.id"] = event["tx.id"]
	expected["tx.traceId"] = event["tx.traceId"]
	expected["trace.out.headers"] = event["trace.out.headers"]
	delete(event, "topic")
	em, _ := json.MarshalIndent(expected, "", "\t")
	expectedStr := string(em)
	em, _ = json.MarshalIndent(event, "", "\t")
	eventStr := string(em)
	if !reflect.DeepEqual(event, expected) {
		t.Fatalf("wrong debug message from eel: expected:\n %s received:\n %s\n", expectedStr, eventStr)
	}
}

func TestEventHandlerSync(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(testEvent))
	r.Header.Add("X-Sync", "true")
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(testEvent)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if len(bs) < len(testEvent)/2 {
		t.Fatalf("wrong length of eel response: expected %d received %d\n response:%s\n", len(testDebugResponse), len(bs), string(bs))
	}
	eventStr := string(bs)
	var event interface{}
	if err := json.Unmarshal([]byte(eventStr), &event); err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	var expected map[string]interface{}
	if err := json.Unmarshal([]byte(testEvent), &expected); err != nil {
		t.Fatalf("json error: %s\n", err.Error())
	}
	if !reflect.DeepEqual(event, expected) {
		t.Fatalf("wrong debug message from eel: expected:\n %s received:\n %s\n", testDebugResponse, eventStr)
	}
}

func TestEventHandler(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(testEvent))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(testEvent)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.StatusCode != 202 {
		t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
	}
	event, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	eventStr := string(event)
	expected := `"status":"processed"`
	if !strings.Contains(eventStr, expected) {
		t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expected, eventStr)
	}
}

/*func TestEventHandlerMulti(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	client := &http.Client{}
	for i := 0; i < 10; i++ {
		go func() {
			r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(testEvent))
			r.Header.Add("Content-Type", "application/json")
			r.Header.Add("Content-Length", strconv.Itoa(len(testEvent)))
			resp, err := client.Do(r)
			if err != nil {
				t.Fatalf("error posting notification: %s\n", err.Error())
			}
			if resp.StatusCode != 202 {
				t.Fatalf("eel returned unhappy status: %s\n", resp.Status)
			}
			event, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
			}
			eventStr := string(event)
			expected := `"status":"processed"`
			if !strings.Contains(eventStr, expected) {
				t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expected, eventStr)
			}
		}()
	}
}*/

func TestEventHandlerInvalidJSON(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	event := `foo bar`
	expectedResponse := `"error":"invalid json"`
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "400 Bad Request" {
		t.Fatalf("eel did not return 400 Bad Request: %s\n", resp.Status)
	}
	response, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if !strings.Contains(string(response), expectedResponse) {
		t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expectedResponse, response)
	}
}

/*func TestEventHandlerInvalidJSONMulti(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	event := `foo bar`
	expectedResponse := `"error":"invalid json"`
	client := &http.Client{}
	for i := 0; i < 10; i++ {
		go func() {
			r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
			r.Header.Add("Content-Type", "application/json")
			r.Header.Add("Content-Length", strconv.Itoa(len(event)))
			resp, err := client.Do(r)
			if err != nil {
				t.Fatalf("error posting notification: %s\n", err.Error())
			}
			if resp.Status != "400 Bad Request" {
				t.Fatalf("eel did not return 400 Bad Request: %s\n", resp.Status)
			}
			response, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
			}
			if !strings.Contains(string(response), expectedResponse) {
				t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expectedResponse, response)
			}
		}()
	}
}*/

func TestEventHandlerBlankMessage(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	event := ``
	expectedResponse := `"error":"empty body"`
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "400 Bad Request" {
		t.Fatalf("eel did not return 400 Bad Request: %s\n", resp.Status)
	}
	response, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if !strings.Contains(string(response), expectedResponse) {
		t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expectedResponse, response)
	}
}

func TestEventHandlerLargeMessage(t *testing.T) {
	initTests("../config-handlers")
	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	event := string(make([]byte, 1000000))
	expectedResponse := `"error":"request too large"`
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event)))
	resp, err := client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "413 Request Entity Too Large" {
		t.Errorf("eel did not return 413 Request Entity Too Large: %s\n", resp.Status)
	}
	response, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if !strings.Contains(string(response), expectedResponse) {
		t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expectedResponse, response)
	}
}

func TestDropDuplicateEvent(t *testing.T) {
	initTests("../config-handlers")
	// turn on duplicate checker
	dc := NewLocalInMemoryDupChecker(1000, 10000)
	Gctx.AddValue(EelDuplicateChecker, dc)

	ts := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts.Close()
	event := `{"foo":"bar"}`
	expectedResponse := `"status":"duplicate eliminated"`
	client := &http.Client{}
	r, _ := http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event)))
	resp, err := client.Do(r)
	r, _ = http.NewRequest("POST", ts.URL, bytes.NewBufferString(event))
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(event)))
	resp, err = client.Do(r)
	if err != nil {
		t.Fatalf("error posting notification: %s\n", err.Error())
	}
	if resp.Status != "200 OK" {
		t.Fatalf("eel did not return 200: %s\n", resp.Status)
	}
	response, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf("couldn't read eel debug response: %s\n", err.Error())
	}
	if !strings.Contains(string(response), expectedResponse) {
		t.Fatalf("wrong response from eel: expected:\n %s received:\n %s\n", expectedResponse, response)
	}
}

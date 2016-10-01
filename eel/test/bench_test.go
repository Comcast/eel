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
	"testing"
	"time"

	. "github.com/Comcast/eel/eel/handlers"
	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

func benchmarkSingleEvent(b *testing.B, folder string, headers map[string]string) {
	// mock event
	event, err := NewJDocFromFile(filepath.Join(folder, "in.json"))
	if err != nil {
		b.Fatalf("could not read in event: %s\n", err.Error())
	}
	buf, err := ioutil.ReadFile(filepath.Join(folder, "out.json"))
	if err != nil {
		b.Fatalf("could not read out event")
	}
	expectedEvent, err := NewJDocFromString(string(buf))
	if err != nil {
		b.Fatalf("bad json in expected response")
	}
	// rebind event service url
	ts0 := httptest.NewServer(http.HandlerFunc(PostHandler))
	defer ts0.Close()
	GetConfig(Gctx).Endpoint = ts0.URL
	hf := GetHandlerFactory(Gctx)
	for _, h := range hf.GetAllHandlers(Gctx) {
		if h.Endpoint != "" {
			h.Endpoint = ts0.URL
		}
	}
	// register event service handler
	ts1 := httptest.NewServer(http.HandlerFunc(EventHandler))
	defer ts1.Close()
	// simulate event
	client := &http.Client{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
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
			b.Fatalf("error posting elements event: %s\n", err.Error())
		}
		if resp.StatusCode != 200 && resp.StatusCode != 202 {
			b.Fatalf("eel returned unhappy status: %s\n", resp.Status)
		}
		_, err = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			b.Fatalf("couldn't read event response: %s\n", err.Error())
		}
		var eel interface{}
		iter := 0
		for {
			// give post hanlder a chance to store event
			time.Sleep(time.Duration(1) * time.Millisecond)
			// read and compare received event
			eel = Gctx.Value("debug_post_body")
			if eel == nil && iter >= 1000 {
				b.Fatalf("no response")
			} else {
				break
			}
			iter++
		}
		receivedEvent, err := NewJDocFromString(string(eel.([]byte)))
		if err != nil {
			b.Fatalf("bad json in response")
		}
		if !expectedEvent.Equals(receivedEvent) {
			b.Fatalf("actual and expected event differ:\n%s\n%s\n", receivedEvent.StringPretty(), expectedEvent.StringPretty())
		}
	}
}

func benchmarkRawTransformation(b *testing.B, folder string, isTransformationByExample bool) {
	in, err := NewJDocFromFile(filepath.Join(folder, "in.json"))
	if err != nil {
		b.Fatalf("could not read in event: %s\n", err.Error())
	}
	out, err := NewJDocFromFile(filepath.Join(folder, "out.json"))
	if err != nil {
		b.Fatalf("could not read out event: %s\n", err.Error())
	}
	h, err := NewJDocFromFile(filepath.Join(folder, "handlers/tenant1/handler.json"))
	if err != nil {
		b.Fatalf("could not read handler: %s\n", err.Error())
	}
	t, err := NewJDocFromMap(h.EvalPath(Gctx, "/Transformation").(map[string]interface{}))
	if err != nil {
		b.Fatalf("could not parse transformation: %s\n", err.Error())
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if n == 0 {
			//b.Logf("n: %d\n", b.N)
		}
		if isTransformationByExample {
			transformed := in.ApplyTransformationByExample(Gctx, t)
			if transformed == nil {
				b.Fatalf("transformationfailed")
			}
			if !transformed.Equals(out) {
				b.Fatalf("actual and expected event differ (transf by example):\nactual:\n%s\nexpected:\n%s\nin:\n%s\nt:\n%s\n", transformed.StringPretty(), out.StringPretty(), in.StringPretty(), t.StringPretty())
			}
		} else {
			transformed := in.ApplyTransformation(Gctx, t)
			if transformed == nil {
				b.Fatalf("transformationfailed")
			}
			if !transformed.Equals(out) {
				b.Fatalf("actual and expected event differ (transf by path):\nactual:\n%s\nexpected:\n%s\nin:\n%s\nt:\n%s\n", transformed.StringPretty(), out.StringPretty(), in.StringPretty(), t.StringPretty())
			}
		}
	}
}

func BenchmarkRawTransformationCanonicalizeEvent(b *testing.B) {
	initTests("data/test01/handlers")
	benchmarkRawTransformation(b, "data/test01/", false)
}

/*func BenchmarkRawTransformationArithmetic(b *testing.B) {
	initTests("data/test57/handlers")
	benchmarkRawTransformation(b, "data/test57/", false)
}*/

func BenchmarkRawTransformationByExample(b *testing.B) {
	initTests("data/test03/handlers")
	benchmarkRawTransformation(b, "data/test03/", true)
}

func BenchmarkRawTransformationMessageGeneration(b *testing.B) {
	initTests("data/test05/handlers")
	benchmarkRawTransformation(b, "data/test05/", false)
}

func BenchmarkRawTransformationJavaScript(b *testing.B) {
	initTests("data/test10/handlers")
	benchmarkRawTransformation(b, "data/test10/", false)
}

func BenchmarkRawTransformationMatchByExample(b *testing.B) {
	initTests("data/test11/handlers")
	benchmarkRawTransformation(b, "data/test11/", false)
}

func BenchmarkRawTransformationStringOps(b *testing.B) {
	initTests("data/test15/handlers")
	benchmarkRawTransformation(b, "data/test15/", false)
}

/*func BenchmarkRawTransformationContains(b *testing.B) {
	initTests("data/test17/handlers")
	benchmarkRawTransformation(b, "data/test17/", false)
}*/

func BenchmarkRawTransformationCase(b *testing.B) {
	initTests("data/test19/handlers")
	benchmarkRawTransformation(b, "data/test19/", false)
}

func BenchmarkRawTransformationRegex(b *testing.B) {
	initTests("data/test20/handlers")
	benchmarkRawTransformation(b, "data/test20/", false)
}

func BenchmarkCanonicalizeEvent(b *testing.B) {
	initTests("data/test01/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test01/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkTransformationByExample(b *testing.B) {
	initTests("data/test03/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test03/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkNamedTransformation(b *testing.B) {
	initTests("data/test04/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test04/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkMessageGeneration(b *testing.B) {
	initTests("data/test05/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test05/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkJavaScript(b *testing.B) {
	initTests("data/test10/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test10/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkMatchByExample(b *testing.B) {
	initTests("data/test11/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test11/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkCustomProperties(b *testing.B) {
	initTests("data/test13/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test13/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkStringOps(b *testing.B) {
	initTests("data/test15/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test15/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkContains(b *testing.B) {
	initTests("data/test17/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test17/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkCase(b *testing.B) {
	initTests("data/test19/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test19/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkRegex(b *testing.B) {
	initTests("data/test20/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test20/"
	benchmarkSingleEvent(b, folder, headers)
}

func BenchmarkNamedTransformationsAndArrays(b *testing.B) {
	initTests("data/test27/handlers")
	headers := make(map[string]string, 0)
	folder := "data/test27/"
	benchmarkSingleEvent(b, folder, headers)
}

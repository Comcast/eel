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

package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	. "github.com/Comcast/eel/eel/jtl"
	. "github.com/Comcast/eel/eel/util"
)

type (
	TopicHandlerTest struct {
		Response         string
		Message          string
		Transformation   string
		Transformations  string
		CustomProperties string
		Filter           string
		ErrorMessage     string
		Path             interface{}
		PathExpression   string
		IsTBE            bool
		IsFBE            bool
		IsFInvt          bool
	}
	AstTest struct {
		Message          string
		Expression       string
		Result           string
		ErrorMessage     string
		Transformations  string
		CustomProperties string
		Lists            [][][]string
	}
)

// DummyEventHandler http handler to accept any JSON payload. Performs some basic validations and otherwise does nothing.
func DummyEventHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	delta := int64(0)
	if ctx.Value("dummy.last.ts") != nil {
		delta = time.Now().UnixNano() - ctx.Value("dummy.last.ts").(int64)
	}
	Gctx.AddValue("dummy.last.ts", time.Now().UnixNano())
	ctx.AddLogValue("delta", delta/1e6)
	w.Header().Set("Content-Type", "application/json")
	if r.ContentLength > GetConfig(ctx).MaxMessageSize {
		ctx.Log().Error("dummy", true, "status", "413", "event", "rejected", "reason", "message_too_large", "msg.length", r.ContentLength, "msg.max.length", GetConfig(ctx).MaxMessageSize, "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		http.Error(w, string(StatusRequestTooLarge), http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, GetConfig(ctx).MaxMessageSize)
	defer r.Body.Close()
	if r.Method != "POST" {
		ctx.Log().Error("dummy", true, "status", "400", "event", "rejected", "reason", "http_post_required", "method", r.Method, "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		w.WriteHeader(http.StatusBadRequest)
		w.Write(StatusHttpPostRequired)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("dummy", true, "status", "500", "event", "rejected", "reason", "error_reading_message", "error", err.Error(), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("dummy", true, "status", "400", "event", "rejected", "reason", "blank_message", "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		w.WriteHeader(http.StatusBadRequest)
		w.Write(StatusEmptyBody)
		return
	}
	_, err = NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("dummy", true, "status", "400", "event", "rejected", "reason", "invalid_json", "error", err.Error(), "content", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
		w.WriteHeader(http.StatusBadRequest)
		w.Write(StatusInvalidJson)
		return
	}
	ctx.Log().Info("dummy", true, "status", "200", "event", "accepted", "content", string(body), "remote_address", r.RemoteAddr, "user_agent", r.UserAgent())
	w.WriteHeader(http.StatusOK)
	w.Write(StatusProcessedDummy)
}

// FortyTwoHandler http handler providing 42 as a service.
func FortyTwoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	ctx.Log().Info("event", "42aas")
	//body, _ := ioutil.ReadAll(r.Body)
	//fmt.Printf("body: %s\n", body)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `42`)
}

// FortyTwoHandler http handler providing 42 as a service (in JSON encoding).
func FortyTwoJsonHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	ctx.Log().Info("event", "42aasjson")
	//body, _ := ioutil.ReadAll(r.Body)
	//fmt.Printf("body: %s\n", body)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{ "accountId" : "42", "foo": "bar" }`)
}

var post_handler_mux sync.RWMutex

// PostHandler http handler accepts any http POST and sticks body into global context under the key 'debug_post_body'.
func PostHandler(w http.ResponseWriter, r *http.Request) {
	post_handler_mux.Lock()
	defer post_handler_mux.Unlock()
	ctx := Gctx.SubContext()
	body, _ := ioutil.ReadAll(r.Body)
	Gctx.AddValue("debug_post_body", body)
	if Gctx.Value("debug_post_body_array") == nil {
		Gctx.AddValue("debug_post_body_array", make([][]byte, 0))
	}
	Gctx.AddValue("debug_post_body_array", append(Gctx.Value("debug_post_body_array").([][]byte), body))
	ctx.Log().Info("event", "storing_post_body", "body", string(body), "len", len(Gctx.Value("debug_post_body_array").([][]byte)))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{ "status" : "success" }`)
}

// ProcessExpressionHandler http handler to process jpath expression given as part of the URL and writes results to w.
func ProcessExpressionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	if !strings.HasPrefix(r.URL.Path, "/test/process/") {
		ctx.Log().Error("status", "500", "event", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	expression := r.URL.Path[len("/test/process/"):]
	if expression == "" {
		ctx.Log().Error("status", "500", "event", "blank_expression")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank expression"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("status", "500", "event", "error_reading_message", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("status", "400", "event", "blank_message")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"empty body"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	msg, err := NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("status", "400", "event", "invalid_json", "error", err.Error(), "content", string(body))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid json"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	result := ToFlatString(msg.ParseExpression(ctx, expression))
	ctx.Log().Info("event", "process", "expression", expression, "result", result)
	w.WriteHeader(http.StatusOK)
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
	} else {
		fmt.Fprintf(w, result)
	}
}

// GetASTJsonHandler http handler for step debugging of jpath expressions. Processes iter number of iterations of collapsing
// the AST of a jpath expression given as part of the URL. The jpath expression operates against a JSON document stored at
// events/asttest.json and writes D3 JSON results to w.
func GetASTJsonHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	doc, err := NewJDocFromFile(filepath.Join(BasePath, "events/asttest.json"))
	if err != nil {
		ctx.Log().Error("status", "500", "event", "error_loading_asttest_doc", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/test/astjson/") {
		ctx.Log().Error("status", "500", "event", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	data := r.URL.Path[len("/test/astjson/"):]
	if data == "" {
		ctx.Log().Error("status", "500", "event", "blank_expression")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank expression"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	if strings.Index(data, "/") <= 0 {
		ctx.Log().Error("status", "500", "event", "blank_iteration_step")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank iteration step"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	iter, err := strconv.Atoi(data[0:strings.Index(data, "/")])
	if err != nil {
		ctx.Log().Error("status", "500", "event", "error_parsing_iteration", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	expression := data[strings.Index(data, "/")+1:]
	jexpr, err := NewJExpr(expression)
	if err != nil {
		ctx.Log().Error("status", "500", "event", "error_parsing_expression", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	for i := 0; i < iter; i++ {
		jexpr.CollapseNextLeafDebug(ctx, doc)
	}
	buf, err := json.MarshalIndent(jexpr.GetD3Json(nil), "", "\t")
	if err != nil {
		ctx.Log().Error("status", "500", "event", "error_rendering_json", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	ctx.Log().Info("event", "astjson", "expression", expression, "iter", iter)
	w.WriteHeader(http.StatusOK)
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
	} else {
		fmt.Fprintf(w, string(buf))
	}
}

// ParserDebugVizHandler http handler for D3 vizualization of jpath expression AST
func ParserDebugVizHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	if !strings.HasPrefix(r.URL.Path, "/test/asttree/") {
		ctx.Log().Error("status", "500", "event", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	expression := r.URL.Path[len("/test/asttree/"):]
	if expression == "" {
		ctx.Log().Error("status", "500", "event", "blank_expression")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank expression"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/asttree.html"))
	var ta AstTest
	ta.Expression = expression
	t.Execute(w, ta)
}

// ParserDebugHandler http handler to display HTML version of AST for jpath expression
func ParserDebugHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/ast.html"))
	message := r.FormValue("message")
	if message == "" {
		message = ""
	}
	expression := r.FormValue("expression")
	transformations := r.FormValue("transformations")
	customproperties := r.FormValue("customproperties")
	var ta AstTest
	ta.Lists = make([][][]string, 0)
	ta.Message = message
	ta.CustomProperties = customproperties
	if message != "" && expression != "" {
		var h HandlerConfiguration
		if customproperties != "" {
			var ct map[string]interface{}
			err := json.Unmarshal([]byte(customproperties), &ct)
			if err != nil {
				ta.ErrorMessage = "error parsing custom properties: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(ct, "", "\t")
				ta.CustomProperties = string(buf)
			}
			h.CustomProperties = ct
			ctx.AddValue(EelCustomProperties, ct)
		}
		if transformations != "" {
			ta.Transformations = transformations
			var nts map[string]*Transformation
			err := json.Unmarshal([]byte(transformations), &nts)
			if err != nil {
				ta.ErrorMessage = "error parsing named transformations: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(nts, "", "\t")
				ta.Transformations = string(buf)
			}
			for _, v := range nts {
				tf, err := NewJDocFromInterface(v.Transformation)
				if err != nil {
					ta.ErrorMessage = "error parsing named transformations: " + err.Error()
				}
				v.SetTransformation(tf)
			}
			h.Transformations = nts
		}
		ctx.AddValue(EelHandlerConfig, &h)
		ta.Expression = expression
		mIn, err := NewJDocFromString(message)
		if err != nil {
			ta.ErrorMessage = "error parsing message: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
			t.Execute(w, ta)
			return
		} else {
			ta.Message = mIn.StringPretty()
		}
		jexpr, err := NewJExpr(expression)
		if err != nil {
			ta.ErrorMessage = "error parsing expression: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
			t.Execute(w, ta)
			return
		}
		r, l := jexpr.ExecuteDebug(ctx, mIn)
		ta.Result = ToFlatString(r)
		ta.Lists = l
		if errs := GetErrors(ctx); errs != nil {
			for _, e := range errs {
				ta.ErrorMessage += e.Error() + "<br/>"
			}
		}
	}
	err := t.Execute(w, ta)
	if err != nil {
		ctx.Log().Error("event", "template_error", "error", err.Error())
	}
}

// TopicTestHandler http handler for Web Form based transformation testing
func TopicTestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/test.html"))
	message := r.FormValue("message")
	if message == "" {
		message = ""
	}
	transformation := r.FormValue("transformation")
	if transformation == "" {
		transformation = `{
	"{{/}}":"{{/}}"
}`
	}
	transformations := r.FormValue("transformations")
	customproperties := r.FormValue("customproperties")
	filter := r.FormValue("filter")
	istbe := false
	if r.FormValue("istbe") == "on" {
		istbe = true
	}
	isfbe := false
	if r.FormValue("isfbe") == "on" {
		isfbe = true
	}
	isfinvt := false
	if r.FormValue("isfinvt") == "on" {
		isfinvt = true
	}
	var tht TopicHandlerTest
	tht.Message = message
	tht.Transformation = transformation
	tht.Transformations = transformations
	tht.CustomProperties = customproperties
	tht.Filter = filter
	tht.IsTBE = istbe
	tht.IsFBE = isfbe
	tht.IsFInvt = isfinvt
	hc := new(HandlerConfiguration)
	err := json.Unmarshal([]byte(transformation), &hc.Transformation)
	if err != nil {
		tht.ErrorMessage = err.Error()
	}
	hc.IsTransformationByExample = istbe
	hc.IsFilterByExample = isfbe
	hc.IsFilterInverted = isfinvt
	hc.IsTransformationByExample = istbe
	hc.Version = "1.0"
	hc.Name = "DEBUG"
	if filter != "" {
		err = json.Unmarshal([]byte(filter), &hc.Filter)
		if err != nil {
			tht.ErrorMessage = "error parsing filter: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(hc.Filter, "", "\t")
			tht.Filter = string(buf)
		}
	}
	hf, _ := NewHandlerFactory(ctx, nil)
	topicHandler, errs := hf.GetHandlerConfigurationFromJson(ctx, "", *hc)
	for _, e := range errs {
		tht.ErrorMessage += e.Error() + "<br/>"
	}
	if hc.Transformation != nil {
		tp, err := json.MarshalIndent(hc.Transformation, "", "\t")
		if err == nil {
			tht.Transformation = string(tp)
		}
	}
	mdoc, err := NewJDocFromString(message)
	if err == nil {
		tht.Message = mdoc.StringPretty()
	}
	if message != "" && topicHandler != nil {
		if customproperties != "" {
			var ct map[string]interface{}
			err := json.Unmarshal([]byte(customproperties), &ct)
			if err != nil {
				tht.ErrorMessage = "error parsing custom properties: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(ct, "", "\t")
				tht.CustomProperties = string(buf)
			}
			topicHandler.CustomProperties = ct
		}
		if transformations != "" {
			var nts map[string]*Transformation
			err := json.Unmarshal([]byte(transformations), &nts)
			if err != nil {
				tht.ErrorMessage = "error parsing named transformations: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(nts, "", "\t")
				tht.Transformations = string(buf)
			}
			for _, v := range nts {
				tf, err := NewJDocFromInterface(v.Transformation)
				if err != nil {
					tht.ErrorMessage = "error parsing named transformations: " + err.Error()
				}
				v.SetTransformation(tf)
			}
			topicHandler.Transformations = nts
			ctx.AddValue(EelHandlerConfig, &topicHandler)
		}
		mIn, err := NewJDocFromString(message)
		if err != nil {
			tht.ErrorMessage = "error parsing message: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
		}
		publishers, err := topicHandler.ProcessEvent(ctx, mIn)
		out := ""
		if err != nil {
			tht.ErrorMessage = "error processing message: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
		} else if publishers != nil && len(publishers) > 0 {
			tht.Response = publishers[0].GetPayload()
			tht.Path = publishers[0].GetPath()
			out = tht.Response
			if errs := GetErrors(ctx); errs != nil {
				for _, e := range errs {
					tht.ErrorMessage += e.Error() + "<br/>"
				}
			}
		}
		ctx.Log().Info("event", "test_transform", "in", message, "out", out, "topic_handler", topicHandler)
	}
	err = t.Execute(w, tht)
	if err != nil {
		ctx.Log().Error("event", "template_error", "error", err.Error())
	}
}

// HandlersTestHandler http handler for Web Form based transformation testing
func HandlersTestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/handlers.html"))
	message := r.FormValue("message")
	if message == "" {
		message = ""
	}
	transformation := r.FormValue("transformation")
	if transformation == "" {
		transformation = `{
	"{{/}}":"{{/}}"
}`
	}
	transformations := r.FormValue("transformations")
	customproperties := r.FormValue("customproperties")
	filter := r.FormValue("filter")
	istbe := false
	if r.FormValue("istbe") == "on" {
		istbe = true
	}
	isfbe := false
	if r.FormValue("isfbe") == "on" {
		isfbe = true
	}
	isfinvt := false
	if r.FormValue("isfinvt") == "on" {
		isfinvt = true
	}
	var tht TopicHandlerTest
	tht.Message = message
	tht.Transformation = transformation
	tht.Transformations = transformations
	tht.CustomProperties = customproperties
	tht.Filter = filter
	tht.IsTBE = istbe
	tht.IsFBE = isfbe
	tht.IsFInvt = isfinvt
	hc := new(HandlerConfiguration)
	err := json.Unmarshal([]byte(transformation), &hc.Transformation)
	if err != nil {
		tht.ErrorMessage = err.Error()
	}
	hc.IsTransformationByExample = istbe
	hc.IsFilterByExample = isfbe
	hc.IsFilterInverted = isfinvt
	hc.IsTransformationByExample = istbe
	hc.Version = "1.0"
	hc.Name = "DEBUG"
	if filter != "" {
		err = json.Unmarshal([]byte(filter), &hc.Filter)
		if err != nil {
			tht.ErrorMessage = "error parsing filter: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(hc.Filter, "", "\t")
			tht.Filter = string(buf)
		}
	}
	hf, _ := NewHandlerFactory(ctx, nil)
	topicHandler, errs := hf.GetHandlerConfigurationFromJson(ctx, "", *hc)
	for _, e := range errs {
		tht.ErrorMessage += e.Error() + "<br/>"
	}
	if hc.Transformation != nil {
		tp, err := json.MarshalIndent(hc.Transformation, "", "\t")
		if err == nil {
			tht.Transformation = string(tp)
		}
	}
	mdoc, err := NewJDocFromString(message)
	if err == nil {
		tht.Message = mdoc.StringPretty()
	}
	if message != "" && topicHandler != nil {
		if customproperties != "" {
			var ct map[string]interface{}
			err := json.Unmarshal([]byte(customproperties), &ct)
			if err != nil {
				tht.ErrorMessage = "error parsing custom properties: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(ct, "", "\t")
				tht.CustomProperties = string(buf)
			}
			topicHandler.CustomProperties = ct
		}
		if transformations != "" {
			var nts map[string]*Transformation
			err := json.Unmarshal([]byte(transformations), &nts)
			if err != nil {
				tht.ErrorMessage = "error parsing named transformations: " + err.Error()
			} else {
				buf, _ := json.MarshalIndent(nts, "", "\t")
				tht.Transformations = string(buf)
			}
			for _, v := range nts {
				tf, err := NewJDocFromInterface(v.Transformation)
				if err != nil {
					tht.ErrorMessage = "error parsing named transformations: " + err.Error()
				}
				v.SetTransformation(tf)
			}
			topicHandler.Transformations = nts
			ctx.AddValue(EelHandlerConfig, &topicHandler)
		}
		mIn, err := NewJDocFromString(message)
		if err != nil {
			tht.ErrorMessage = "error parsing message: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
		}
		publishers, err := topicHandler.ProcessEvent(ctx, mIn)
		out := ""
		if err != nil {
			tht.ErrorMessage = "error processing message: " + err.Error()
			ctx.Log().Error("event", "test_handler_error", "error", err.Error())
		} else if publishers != nil && len(publishers) > 0 {
			tht.Response = publishers[0].GetPayload()
			tht.Path = publishers[0].GetPath()
			out = tht.Response
			if errs := GetErrors(ctx); errs != nil {
				for _, e := range errs {
					tht.ErrorMessage += e.Error() + "<br/>"
				}
			}
		}
		ctx.Log().Info("event", "test_transform", "in", message, "out", out, "topic_handler", topicHandler)
	}
	err = t.Execute(w, tht)
	if err != nil {
		ctx.Log().Error("event", "template_error", "error", err.Error())
	}
}

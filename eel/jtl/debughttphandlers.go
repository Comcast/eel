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
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	. "github.com/Comcast/eel/eel/util"
)

type (
	TopicHandlerTest struct {
		Response         string
		Message          string
		Transformation   string
		Transformations  string
		CustomProperties string
		Filters          string
		ErrorMessage     string
		Path             interface{}
		PathExpression   string
		IsTBE            bool
	}
	AllTopicHandlersTest struct {
		TopicHandlerTest
		AllHandlers     map[string]*HandlerConfiguration
		AllHandlerNames []string
		CurrentHandler  *HandlerConfiguration
		SelectedHandler string
		Headers         string
		HeadersOut      string
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
		ctx.Log().Error("comp", "debug", "status", "413", "action", "rejected", "error_type", "rejected", "cause", "message_too_large", "msg.length", r.ContentLength, "msg.max.length", GetConfig(ctx).MaxMessageSize)
		http.Error(w, string(GetResponse(ctx, StatusRequestTooLarge)), http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, GetConfig(ctx).MaxMessageSize)
	defer r.Body.Close()
	if r.Method != "POST" {
		ctx.Log().Error("comp", "debug", "status", "400", "action", "rejected", "error_type", "rejected", "cause", "http_post_required", "method", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusHttpPostRequired))
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_reading_message", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(GetResponse(ctx, map[string]interface{}{"error": err.Error()}))
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("comp", "debug", "status", "400", "action", "rejected", "error_type", "rejected", "cause", "blank_message")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusEmptyBody))
		return
	}
	_, err = NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "400", "action", "rejected", "error_type", "rejected", "cause", "invalid_json", "error", err.Error(), "content", string(body))
		w.WriteHeader(http.StatusBadRequest)
		w.Write(GetResponse(ctx, StatusInvalidJson))
		return
	}
	ctx.Log().Info("comp", "debug", "status", "200", "action", "accepted", "content", string(body))
	w.WriteHeader(http.StatusOK)
	w.Write(GetResponse(ctx, StatusProcessedDummy))
}

// FortyTwoHandler http handler providing 42 as a service.
func FortyTwoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	ctx.Log().Info("action", "42aas")
	//body, _ := ioutil.ReadAll(r.Body)
	//fmt.Printf("body: %s\n", body)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `42`)
}

// FortyTwoHandler http handler providing 42 as a service (in JSON encoding).
func FortyTwoJsonHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	ctx.Log().Info("action", "42aasjson")
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
	ctx.Log().Info("action", "storing_post_body", "body", string(body), "len", len(Gctx.Value("debug_post_body_array").([][]byte)))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{ "status" : "success" }`)
}

// ProcessExpressionHandler http handler to process jpath expression given as part of the URL and writes results to w.
func ProcessExpressionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	if !strings.HasPrefix(r.URL.Path, "/test/process/") {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	expression := r.URL.Path[len("/test/process/"):]
	if expression == "" {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "blank_expression")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank expression"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_reading_message", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	if body == nil || len(body) == 0 {
		ctx.Log().Error("comp", "debug", "status", "400", "action", "rejected", "error_type", "rejected", "cause", "blank_message")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"empty body"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	msg, err := NewJDocFromString(string(body))
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "400", "action", "rejected", "error_type", "rejected", "cause", "invalid_json", "error", err.Error(), "content", string(body))
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid json"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	result := ToFlatString(msg.ParseExpression(ctx, expression))
	ctx.Log().Info("comp", "debug", "action", "process_expression", "expression", expression, "result", result)
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
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_loading_asttest_doc", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/test/astjson/") {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	data := r.URL.Path[len("/test/astjson/"):]
	if data == "" {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "blank_expression")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank expression"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	if strings.Index(data, "/") <= 0 {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "blank_iteration_step")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"blank iteration step"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	iter, err := strconv.Atoi(data[0:strings.Index(data, "/")])
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_parsing_iteration", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	expression := data[strings.Index(data, "/")+1:]
	jexpr, err := NewJExpr(expression)
	if err != nil {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_parsing_expression", "error", err.Error())
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
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "error_rendering_json", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		fmt.Fprintf(w, "\n")
		return
	}
	ctx.Log().Info("comp", "debug", "action", "astjson", "expression", expression, "iter", iter)
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
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "invalid_path")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"invalid path"}`)
		fmt.Fprintf(w, "\n")
		return
	}
	expression := r.URL.Path[len("/test/asttree/"):]
	if expression == "" {
		ctx.Log().Error("comp", "debug", "status", "500", "action", "rejected", "error_type", "rejected", "cause", "blank_expression")
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
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
			t.Execute(w, ta)
			return
		} else {
			ta.Message = mIn.StringPretty()
		}
		jexpr, err := NewJExpr(expression)
		if err != nil {
			ta.ErrorMessage = "error parsing expression: " + err.Error()
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
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
		ctx.Log().Error("comp", "debug", "error_type", "template_error", "error", err.Error())
	}
}

// TopicTestHandler http handler for Web Form based transformation testing
func TopicTestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/test.html"))
	message := r.FormValue("message")
	if message == "" {
		message = "{}"
	}
	transformation := r.FormValue("transformation")
	if transformation == "" {
		transformation = `{
	"{{/}}":"{{/}}"
}`
	}
	transformations := r.FormValue("transformations")
	customproperties := r.FormValue("customproperties")
	filters := r.FormValue("filters")
	istbe := false
	if r.FormValue("istbe") == "on" {
		istbe = true
	}
	var tht TopicHandlerTest
	tht.Message = message
	tht.Transformation = transformation
	tht.Transformations = transformations
	tht.CustomProperties = customproperties
	tht.Filters = filters
	tht.IsTBE = istbe
	hc := new(HandlerConfiguration)
	err := json.Unmarshal([]byte(transformation), &hc.Transformation)
	if err != nil {
		tht.ErrorMessage = err.Error()
	}
	hc.IsTransformationByExample = istbe
	hc.IsTransformationByExample = istbe
	hc.Version = "1.0"
	hc.Name = "DEBUG"
	if filters != "" {
		err = json.Unmarshal([]byte(filters), &hc.Filters)
		if err != nil {
			tht.ErrorMessage = "error parsing filters: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(hc.Filters, "", "\t")
			tht.Filters = string(buf)
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
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
		}
		publishers, err := topicHandler.ProcessEvent(ctx, mIn)
		out := ""
		if err != nil {
			tht.ErrorMessage = "error processing message: " + err.Error()
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
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
		ctx.Log().Info("comp", "debug", "action", "test_transform", "in", message, "out", out, "topic_handler", topicHandler)
	}
	err = t.Execute(w, tht)
	if err != nil {
		ctx.Log().Error("comp", "debug", "error_type", "template_error", "error", err.Error())
	}
}

// HandlersTestHandler http handler for Web Form based transformation testing
func HandlersTestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Gctx.SubContext()
	t, _ := template.ParseFiles(filepath.Join(BasePath, "web/handlers.html"))
	handler := r.FormValue("handler")
	newhandlerselected := r.FormValue("newhandlerselected")
	savechanges := r.FormValue("savechanges")
	headers := r.FormValue("headers")
	message := r.FormValue("message")
	if message == "" {
		message = "{}"
	}
	transformation := r.FormValue("transformation")
	if transformation == "" {
		transformation = `{
	"{{/}}":"{{/}}"
}`
	}
	transformations := r.FormValue("transformations")
	customproperties := r.FormValue("customproperties")
	filters := r.FormValue("filters")
	istbe := false
	if r.FormValue("istbe") == "on" {
		istbe = true
	}
	var tht AllTopicHandlersTest
	allHandlers := GetHandlerFactory(ctx).GetAllHandlers(ctx)
	tht.AllHandlers = make(map[string]*HandlerConfiguration)
	tht.AllHandlerNames = make([]string, 0)
	for _, hdlr := range allHandlers {
		name := hdlr.TenantId + "/" + hdlr.Name
		tht.AllHandlers[name] = hdlr
		tht.AllHandlerNames = append(tht.AllHandlerNames, name)
		sort.Strings(tht.AllHandlerNames)
	}
	if handler != "" {
		tht.SelectedHandler = handler
		tht.CurrentHandler = tht.AllHandlers[handler]
	} else if len(tht.AllHandlerNames) > 0 {
		tht.SelectedHandler = tht.AllHandlerNames[0]
		tht.CurrentHandler = tht.AllHandlers[tht.AllHandlerNames[0]]
	}
	tht.Message = message
	if newhandlerselected == "true" {
		if tht.AllHandlers[tht.SelectedHandler].Transformation != nil {
			buf, err := json.MarshalIndent(tht.AllHandlers[tht.SelectedHandler].Transformation, "", "\t")
			if err != nil {
				tht.Transformation = err.Error()
			} else {
				tht.Transformation = string(buf)
			}
		}
		if tht.AllHandlers[tht.SelectedHandler].Transformations != nil {
			buf, err := json.MarshalIndent(tht.AllHandlers[tht.SelectedHandler].Transformations, "", "\t")
			if err != nil {
				tht.Transformation = err.Error()
			} else {
				tht.Transformations = string(buf)
			}
		}
		if tht.AllHandlers[tht.SelectedHandler].CustomProperties != nil {
			buf, err := json.MarshalIndent(tht.AllHandlers[tht.SelectedHandler].CustomProperties, "", "\t")
			if err != nil {
				tht.CustomProperties = err.Error()
			} else {
				tht.CustomProperties = string(buf)
			}
		}
		if tht.AllHandlers[tht.SelectedHandler].Filters != nil {
			buf, err := json.MarshalIndent(tht.AllHandlers[tht.SelectedHandler].Filters, "", "\t")
			if err != nil {
				tht.Filters = err.Error()
			} else {
				tht.Filters = string(buf)
			}
		}
		if tht.AllHandlers[tht.SelectedHandler].HttpHeaders != nil {
			buf, err := json.MarshalIndent(tht.AllHandlers[tht.SelectedHandler].HttpHeaders, "", "\t")
			if err != nil {
				tht.Headers = err.Error()
			} else {
				tht.Headers = string(buf)
			}
		}
		tht.IsTBE = tht.AllHandlers[tht.SelectedHandler].IsTransformationByExample
	} else {
		tht.Transformation = transformation
		tht.Transformations = transformations
		tht.CustomProperties = customproperties
		tht.Headers = headers
		tht.Filters = filters
		tht.IsTBE = istbe
	}
	hc := new(HandlerConfiguration)
	err := json.Unmarshal([]byte(tht.Transformation), &hc.Transformation)
	if err != nil {
		tht.ErrorMessage = err.Error() + "<br/>"
	} else {
		tp, err := json.MarshalIndent(hc.Transformation, "", "\t")
		if err == nil {
			tht.Transformation = string(tp)
		}
	}
	hc.IsTransformationByExample = tht.IsTBE
	if tht.CurrentHandler != nil {
		hc.Version = tht.CurrentHandler.Version
		hc.Name = tht.CurrentHandler.Name
		hc.Info = tht.CurrentHandler.Info
	} else {
		hc.Version = "1.0"
		hc.Name = "DEBUG"
	}
	hf, _ := NewHandlerFactory(ctx, nil)
	topicHandler, errs := hf.GetHandlerConfigurationFromJson(ctx, "", *hc)
	for _, e := range errs {
		tht.ErrorMessage += e.Error() + "<br/>"
	}
	if tht.Filters != "" {
		var fts []*Filter
		err = json.Unmarshal([]byte(tht.Filters), &fts)
		if err != nil {
			tht.ErrorMessage = "error parsing filter: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(fts, "", "\t")
			tht.Filters = string(buf)
		}
		topicHandler.Filters = fts
	}
	if tht.CustomProperties != "" {
		var ct map[string]interface{}
		err := json.Unmarshal([]byte(tht.CustomProperties), &ct)
		if err != nil {
			tht.ErrorMessage = "error parsing custom properties: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(ct, "", "\t")
			tht.CustomProperties = string(buf)
		}
		topicHandler.CustomProperties = ct
	}
	if tht.Headers != "" {
		var hdr map[string]string
		err := json.Unmarshal([]byte(tht.Headers), &hdr)
		if err != nil {
			tht.ErrorMessage = "error parsing headers: " + err.Error()
		} else {
			buf, _ := json.MarshalIndent(hdr, "", "\t")
			tht.Headers = string(buf)
		}
		topicHandler.HttpHeaders = hdr
	}
	if tht.Transformations != "" {
		var nts map[string]*Transformation
		err := json.Unmarshal([]byte(tht.Transformations), &nts)
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
	if message != "" && topicHandler != nil {
		mIn, err := NewJDocFromString(message)
		if err != nil {
			tht.ErrorMessage = "error parsing message: " + err.Error()
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
		} else {
			tht.Message = mIn.StringPretty()
		}
		publishers, err := topicHandler.ProcessEvent(ctx, mIn)
		out := ""
		if err != nil {
			tht.ErrorMessage = "error processing message: " + err.Error()
			ctx.Log().Error("comp", "debug", "error_type", "test_handler_error", "error", err.Error())
		} else if publishers != nil && len(publishers) > 0 {
			tht.Response = publishers[0].GetPayload()
			tht.Path = publishers[0].GetPath()
			if publishers[0].GetHeaders() != nil {
				buf, _ := json.MarshalIndent(publishers[0].GetHeaders(), "", "\t")
				tht.HeadersOut = string(buf)
			}
			out = tht.Response
			if errs := GetErrors(ctx); errs != nil {
				for _, e := range errs {
					tht.ErrorMessage += e.Error() + "<br/>"
				}
			}
		}
		ctx.Log().Info("comp", "debug", "action", "test_transform", "in", message, "out", out, "topic_handler", topicHandler)
	}
	if len(savechanges) > 0 {
		fail := false
		tht.CurrentHandler.Transformation = transformation
		tht.CurrentHandler.IsTransformationByExample = istbe
		if tht.Filters != "" {
			var fts []*Filter
			err = json.Unmarshal([]byte(filters), &fts)
			if err != nil {
				tht.ErrorMessage = "error parsing filter: " + err.Error()
				fail = true
			} else {
				tht.CurrentHandler.Filters = fts
			}
		}
		if tht.CustomProperties != "" {
			var cp map[string]interface{}
			err = json.Unmarshal([]byte(customproperties), &cp)
			if err != nil {
				tht.ErrorMessage = "error parsing custom properties: " + err.Error()
				fail = true
			} else {
				tht.CurrentHandler.CustomProperties = cp
			}
		}
		if tht.Headers != "" {
			var hdr map[string]string
			err = json.Unmarshal([]byte(headers), &hdr)
			if err != nil {
				tht.ErrorMessage = "error parsing headers: " + err.Error()
				fail = true
			} else {
				tht.CurrentHandler.HttpHeaders = hdr
			}
		}
		if tht.Transformations != "" {
			var tt map[string]*Transformation
			err = json.Unmarshal([]byte(transformations), &tt)
			if err != nil {
				tht.ErrorMessage = "error parsing transformations: " + err.Error()
				fail = true
			} else {
				tht.CurrentHandler.Transformations = tt
			}
		}
		if !fail {
			err = tht.CurrentHandler.Save()
			if err != nil {
				tht.ErrorMessage = "error saving handler: " + err.Error()
			} else {
				tht.ErrorMessage = "saved handler at " + tht.CurrentHandler.File
			}
		}
	}
	err = t.Execute(w, tht)
	if err != nil {
		ctx.Log().Error("comp", "debug", "error_type", "template_error", "error", err.Error())
	}
}

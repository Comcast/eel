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

package util

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
)

const maxStackSize = 16 * 1024

// DefaultContext simple implementation of the Context interface.
type DefaultContext struct {
	lvals map[string]interface{}
	cvals map[string]interface{}
	vals  map[string]interface{}
	id    string
	log   *DefaultLogger
	sync.RWMutex
}

// NewDefaultContext creates a default context.
func NewDefaultContext(level string) Context {
	dctx := new(DefaultContext)
	dctx.lvals = make(map[string]interface{}, 0)
	dctx.vals = make(map[string]interface{}, 0)
	dctx.cvals = make(map[string]interface{}, 0)
	dctx.log = NewDefaultLogger(dctx, level)
	return dctx
}

func (c *DefaultContext) SetId(id string) {
	c.id = id
}

func (c *DefaultContext) Id() string {
	return c.id
}

func (c *DefaultContext) SubContext() Context {
	c.Lock()
	defer c.Unlock()
	sctx := new(DefaultContext)
	if c.id == "" {
		sctx.id, _ = NewUUID()
	} else {
		sctx.id = c.id
	}
	sctx.log = NewDefaultLogger(sctx, c.log.level)
	sctx.vals = make(map[string]interface{}, 0)
	for k, v := range c.vals {
		sctx.vals[k] = v
	}
	sctx.lvals = make(map[string]interface{}, 0)
	for k, v := range c.lvals {
		sctx.lvals[k] = v
	}
	sctx.cvals = make(map[string]interface{}, 0)
	for k, v := range c.cvals {
		sctx.cvals[k] = v
	}
	return sctx
}

func (c *DefaultContext) Log() Logger {
	return c.log
}

func (c *DefaultContext) AddValue(key interface{}, value interface{}) {
	c.Lock()
	c.vals[key.(string)] = value
	c.Unlock()
}

func (c *DefaultContext) AddLogValue(key interface{}, value interface{}) {
	c.Lock()
	c.lvals[key.(string)] = value
	c.Unlock()
}

func (c *DefaultContext) AddConfigValue(key interface{}, value interface{}) {
	c.Lock()
	c.cvals[key.(string)] = value
	c.Unlock()
}

func (c *DefaultContext) Value(key interface{}) interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.vals[key.(string)]
}

func (c *DefaultContext) LogValue(key interface{}) interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.lvals[key.(string)]
}

func (c *DefaultContext) ConfigValue(key interface{}) interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.cvals[key.(string)]
}

func (c *DefaultContext) DisableLogging() {
	c.log.dfw.Lock()
	defer c.log.dfw.Unlock()
	c.log.dfw.enabled = false
}

func (c *DefaultContext) EnableLogging() {
	c.log.dfw.Lock()
	defer c.log.dfw.Unlock()
	c.log.dfw.enabled = true
}

func (c *DefaultContext) HandlePanic() {
	if x := recover(); x != nil {
		panicError := fmt.Sprintf("%#v", x)
		trace := bytes.NewBuffer(debug.Stack()).String()
		//limit the stack track to 16k
		if maxStackSize < len(trace) {
			trace = trace[:maxStackSize]
		}
		c.Log().Error("panicError", panicError, "stackTrace", trace)
		return
	}
}

func (c *DefaultContext) WrapPanicHttpHandler(fn func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer c.HandlePanic()
		fn(w, r)
	}
}

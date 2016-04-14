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
	"sync"
)

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
	c.vals[key.(string)] = value
}

func (c *DefaultContext) AddLogValue(key interface{}, value interface{}) {
	c.lvals[key.(string)] = value
}

func (c *DefaultContext) AddConfigValue(key interface{}, value interface{}) {
	c.cvals[key.(string)] = value
}

func (c *DefaultContext) Value(key interface{}) interface{} {
	return c.vals[key.(string)]
}

func (c *DefaultContext) LogValue(key interface{}) interface{} {
	return c.lvals[key.(string)]
}

func (c *DefaultContext) ConfigValue(key interface{}) interface{} {
	return c.cvals[key.(string)]
}

func (c *DefaultContext) DisableLogging() {
	c.log.dfw.enabled = false
}

func (c *DefaultContext) EnableLogging() {
	c.log.dfw.enabled = true
}

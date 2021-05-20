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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// ToFlatString helper function to convert anything to a flat string representation. Complex types are converted to json.
func ToFlatString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch v.(type) {
	case string:
		return v.(string)
	case int:
		return strconv.Itoa(v.(int))
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', 2, 64)
	case float32:
		return strconv.FormatFloat(float64(v.(float32)), 'f', 2, 32)
	case bool:
		return strconv.FormatBool(v.(bool))
	case interface{}:
		buf, err := json.Marshal(v)
		if err != nil {
			return ""
		} else {
			return string(buf)
		}
	default:
		return ""
	}
}

func DeepEquals(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}
	switch a.(type) {
	case string:
		return a.(string) == b.(string)
	case int:
		return a.(int) == b.(int)
	case float64:
		return a.(float64) == b.(float64)
	case float32:
		return a.(float32) == b.(float32)
	case bool:
		return a.(bool) == b.(bool)
	case []interface{}:
		switch b.(type) {
		case []interface{}:
			if len(a.([]interface{})) != len(b.([]interface{})) {
				return false
			}
			for i, ai := range a.([]interface{}) {
				if !DeepEquals(ai, b.([]interface{})[i]) {
					return false
				}
			}
		default:
			return false
		}
	case map[string]interface{}:
		switch b.(type) {
		case map[string]interface{}:
			if len(a.(map[string]interface{})) != len(b.(map[string]interface{})) {
				return false
			}
			for k, v := range a.(map[string]interface{}) {
				if !DeepEquals(v, b.(map[string]interface{})[k]) {
					return false
				}
			}
		default:
			return false
		}
	default:
		bufa, err := json.Marshal(a)
		if err != nil {
			return false
		}
		bufb, err := json.Marshal(b)
		if err != nil {
			return false
		}
		return string(bufa) == string(bufb)
	}
	return true
}

func NewUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func ExtractAppId(tenant string, allowPartner bool) string {
	if !allowPartner {
		return tenant
	}
	combinedTenant := tenant
	appId := ""
	if strings.LastIndex(combinedTenant, "_") > 0 && strings.LastIndex(combinedTenant, "_") < len(combinedTenant)-1 {
		appId = combinedTenant[:strings.LastIndex(combinedTenant, "_")]
	} else {
		appId = combinedTenant
	}
	return appId
}

func ExtractPartnerId(tenant string, allowPartner bool) string {
	if !allowPartner {
		return ""
	}
	combinedTenant := tenant
	partnerId := "comcast"
	if strings.LastIndex(combinedTenant, "_") > 0 && strings.LastIndex(combinedTenant, "_") < len(combinedTenant)-1 {
		partnerId = combinedTenant[strings.LastIndex(combinedTenant, "_")+1:]
	}
	return partnerId
}

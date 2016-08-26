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
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	. "github.com/Comcast/eel/eel/util"
)

// A simple & efficient JSON transformation library inspired by XPath and XSLT

type (
	JDoc struct {
		orig interface{}            // orignal object
		pmap map[string]interface{} // flat map
	}
	Transformation struct {
		Transformation            interface{}
		IsTransformationByExample bool
		t                         *JDoc
	}
	Filter struct {
		Filter                    map[string]interface{} // only forward event if event matches this pattern (by path or by example)
		IsFilterByExample         bool                   // choose syntax style by path or by example for event filtering
		IsFilterInverted          bool                   // true: filter if event matches pattern, false: filter if event does not match pattern
		FilterAfterTransformation bool                   // true: apply filters after transformation, false (default): apply filter before transformation on raw event
		LogParams                 map[string]string      // extra log parameters
		f                         *JDoc
	}
)

func (t *Transformation) GetTransformation() *JDoc {
	return t.t
}

func (t *Transformation) SetTransformation(tf *JDoc) {
	t.t = tf
}

var (
	JPathSimpleReg *regexp.Regexp
)

const (
	JPathThisSelector = "."
	JPathWildcard     = "*"
	JPathPrefix       = "{{"
	JPathSuffix       = "}}"
	JPathSimple       = "(\\/[a-zA-Z0-9\\.,:_\\-]*)+"
	JPathOr           = "||"
)

// NewJDocFromString returns handle to JSON document given as a string.
func NewJDocFromString(doc string) (*JDoc, error) {
	j := new(JDoc)
	var o interface{}
	j.orig = o
	j.pmap = make(map[string]interface{}, 0)
	if doc == "" {
		return j, nil
	}
	err := json.Unmarshal([]byte(doc), &j.orig)
	if err != nil {
		return nil, err
	}
	j.orig = j.convertFloat2Int(j.orig)
	j.parseMessageMap(j.orig, "")
	return j, nil
}

// NewJDocFromFile returns handle to JSON document given as a path to a file.
func NewJDocFromFile(filePath string) (*JDoc, error) {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	j := new(JDoc)
	var o interface{}
	j.orig = o
	j.pmap = make(map[string]interface{}, 0)
	if len(buf) == 0 {
		return j, nil
	}
	err = json.Unmarshal(buf, &j.orig)
	if err != nil {
		return nil, err
	}
	j.orig = j.convertFloat2Int(j.orig)
	j.parseMessageMap(j.orig, "")
	return j, nil
}

// NewJDocFromMap returns handle to JSON document given as a map.
func NewJDocFromMap(m map[string]interface{}) (*JDoc, error) {
	j := new(JDoc)
	j.orig = m
	j.pmap = make(map[string]interface{}, 0)
	if m == nil || len(m) == 0 {
		return j, nil
	}
	j.orig = j.convertFloat2Int(j.orig)
	j.parseMessageMap(j.orig, "")
	return j, nil
}

// NewJDocFromInterface returns handle to JSON document given as a map.
func NewJDocFromInterface(o interface{}) (*JDoc, error) {
	j := new(JDoc)
	j.orig = o
	j.pmap = make(map[string]interface{}, 0)
	if o == nil {
		return j, nil
	}
	j.orig = j.convertFloat2Int(j.orig)
	j.parseMessageMap(j.orig, "")
	return j, nil
}

// GetOriginalObject returns original document as map.
func (j *JDoc) GetOriginalObject() interface{} {
	return j.orig
}

// GetFlatObject returns exploded version of original document.
func (j *JDoc) GetFlatObject() map[string]interface{} {
	return j.pmap
}

// String converts document to single line JSON string representation.
func (j *JDoc) String() string {
	buf, err := json.Marshal(j.orig)
	if err != nil {
		return ""
	}
	return string(buf)
}

// StringPretty converts document to pretty JSON string representation.
func (j *JDoc) StringPretty() string {
	buf, err := json.MarshalIndent(j.orig, "", "\t")
	if err != nil {
		return ""
	}
	return string(buf)
}

// convertFloat2Int is a helper function to convert floats to ints whereever possible without rounding in the internal document representation.
// Apparently in JSON all numbers are floats which can cause trouble. This is
// a hack to make things look good. Otherwise, "timestamp": 1416505007395 turns
// into "timestamp": 1.416505007395e+12.
func (j *JDoc) convertFloat2Int(o interface{}) interface{} {
	switch o.(type) {
	case map[string]interface{}:
		m := o.(map[string]interface{})
		for k, v := range m {
			switch v.(type) {
			case map[string]interface{}:
				j.convertFloat2Int(v)
			case []interface{}:
				j.convertFloat2Int(v)
			case float64:
				if v.(float64) == float64(int(v.(float64))) {
					m[k] = int(v.(float64))
				}
			case float32:
				if v.(float32) == float32(int(v.(float32))) {
					m[k] = int(v.(float32))
				}
			default:
			}
		}
	case []interface{}:
		a := o.([]interface{})
		for i, v := range a {
			switch v.(type) {
			case map[string]interface{}:
				j.convertFloat2Int(v)
			case []interface{}:
				j.convertFloat2Int(v)
			case float64:
				if v.(float64) == float64(int(v.(float64))) {
					a[i] = int(v.(float64))
				}
			case float32:
				if v.(float32) == float32(int(v.(float32))) {
					a[i] = int(v.(float32))
				}
			default:
			}
		}
	case float64:
		if o.(float64) == float64(int(o.(float64))) {
			o = int(o.(float64))
		}
	case float32:
		if o.(float32) == float32(int(o.(float32))) {
			o = int(o.(float32))
		}
	default:
	}
	return o
}

func (j *JDoc) parseMessageMap(o interface{}, path string) {
	if o == nil {
		return
	}
	if path == "" {
		j.pmap["/"] = o
	}
	switch o.(type) {
	case map[string]interface{}:
		m := o.(map[string]interface{})
		for k, v := range m {
			lpath := path + "/" + k
			switch v.(type) {
			case []interface{}:
				j.pmap[lpath] = v
				j.parseMessageMap(v, lpath)
			case map[string]interface{}:
				j.pmap[lpath] = v
				j.parseMessageMap(v, lpath)
			default:
				j.pmap[lpath] = v
			}
		}
	//TODO: parsing arrays by index may be a bad idea (consider MatchesPattern() below)
	// experimental code for array path treatment
	//case []interface{}:
	//a := o.([]interface{})
	//for i, v := range a {
	//	path = path + "[" + strconv.Itoa(i) + "]"
	//	switch v.(type) {
	//	case map[string]interface{}:
	//		j.pmap[path] = v
	//		j.parseMessageMap(v, path)
	//	case []interface{}:
	//		j.pmap[path] = v
	//		j.parseMessageMap(v, path)
	//	default:
	//		j.pmap[path] = v
	//	}
	default:
		// we completely halt for arrays (so maps inside of arrays will not be parsed!)
		j.pmap[path] = o
	}
}

// Equals performs a deep equal test for two JSON documents.
func (j *JDoc) Equals(j2 *JDoc) bool {
	if j2 == nil {
		return false
	}
	return DeepEquals(j.orig, j2.orig)
}

// MatchesPattern checks if given document matches given pattern. Wildcards are expressed as '*' and boolean or is expressed as '||'.
/*func (j *JDoc) MatchesPattern(pattern *JDoc) bool {
	if pattern == nil || j.pmap == nil || j.orig == nil {
		return false
	}
	// pseudo-deep match via path-map
	for pk, pv := range pattern.pmap {
		if !strings.HasSuffix(pk, "/"+JPathWildcard) && !j.HasPath(pk) {
			return false
		}
		switch pattern.pmap[pk].(type) {
		//TODO: need array match here
		case string:
			alts := strings.Split(pv.(string), JPathOr)
			result := false
			for _, alt := range alts {
				//Gctx.Log().Info("pk", pk, "pv", pv, "alt", alt, "j.pmap[pk]", j.pmap[pk])
				if alt == JPathWildcard || alt == j.pmap[pk] {
					result = true
				}
			}
			if !result {
				return false
			}
		case int:
			if pv != j.pmap[pk] {
				return false
			}
		case float64:
			if pv != j.pmap[pk] {
				return false
			}
		case float32:
			if pv != j.pmap[pk] {
				return false
			}
		case bool:
			if pv != j.pmap[pk] {
				return false
			}
		}
	}
	return true
}*/

// MatchesPattern checks if given document matches given pattern. Wildcards are expressed as '*' and boolean or is expressed as '||'.
func (j *JDoc) MatchesPattern(pattern *JDoc) (bool, int) {
	if pattern == nil || j.pmap == nil || j.orig == nil {
		return false, 0
	}
	c, s := j.contains(j.orig, pattern.orig, 0)
	//Gctx.Log().Info("contains", c, "match_strength", s, "doc", j.orig, "pattern", pattern.orig)
	return c, s
}

// traverseEval traverses all maps and arrays in a json transformation whose top-level map
// is passed in as parameter o, the document to select from is j.
func (j *JDoc) traverseEval(ctx Context, o interface{}) interface{} {
	switch o.(type) {
	case map[string]interface{}:
		for k, v := range o.(map[string]interface{}) {
			switch v.(type) {
			case string:
				o.(map[string]interface{})[k] = j.ParseExpression(ctx, v.(string))
			default:
				j.traverseEval(ctx, v)
			}
		}
	case []interface{}:
		for idx, v := range o.([]interface{}) {
			switch v.(type) {
			case string:
				o.([]interface{})[idx] = j.ParseExpression(ctx, v.(string))
			default:
				j.traverseEval(ctx, v)
			}
		}
	case string:
		return j.ParseExpression(ctx, o.(string))
	}
	return o
}

// ApplyTransformationByExample uses a valid JSON document as a basis for the transformation result.
// All keys have to be constants (strings), values can be either constants or jpath
// expressions. This allows for arbitrarily complex JSON documents including maps and
// arrays. tf contains the transformation and j the document to select values from.
func (j *JDoc) ApplyTransformationByExample(ctx Context, tf *JDoc) *JDoc {
	//ctx.Log().Info("orig", tf.orig)
	if tf == nil || tf.pmap == nil || tf.orig == nil || j.pmap == nil || j.orig == nil {
		return nil
	}
	res, err := NewJDocFromString(tf.String()) // clone
	if err != nil {
		ctx.Log().Error("action", "clone_error", "error", err.Error())
		return nil
	}
	res.orig = j.traverseEval(ctx, res.orig)
	res.parseMessageMap(res.orig, "")
	return res
}

// ApplyTransformation uses a collection of jpath expressions on the left hand side and
// constants or jpath expressions on the right hand side as a basis for the transformation
// result. This allows for arbitrarily complex / nested map structures to be created but
// arrays can not be expressed. This turns out too be an unacceptable limitation which is
// why there is ApplyTransformationByExample() as an alternative.
// Syntax examples for path values in transformation.pmap:
// {{/}} 		-> copy entire document
// {{/a/b/c}} 	-> copy contents from location /a/b/c in source document
// {{/a/b[0]}}  -> copy array element
// "abc" 		-> create string field with value "abc"
// 123 			-> create number field with value 123
func (j *JDoc) ApplyTransformation(ctx Context, transformation *JDoc) *JDoc {
	if transformation == nil || j.pmap == nil || j.orig == nil {
		return nil
	}
	t, _ := NewJDocFromString("")
	switch transformation.orig.(type) {
	case map[string]interface{}:
	default:
		ctx.Log().Error("action", "transformation_not_a_map", "transformation", transformation.orig)
		return t
	}
	for tk, tv := range transformation.orig.(map[string]interface{}) {
		m := t.orig
		if !strings.HasPrefix(tk, JPathPrefix+"/") || !strings.HasSuffix(tk, JPathSuffix) {
			ctx.Log().Info("action", "bad_path_warning", "path", tk, "transformation", transformation.orig)
			continue
		}
		tkPath := strings.Replace(tk, JPathPrefix, "", -1)
		tkPath = strings.Replace(tkPath, JPathSuffix, "", -1)
		// ensure target path exists
		segments := strings.Split(tkPath, "/")
		ns := make([]string, 0)
		for _, s := range segments {
			if s != "" {
				ns = append(ns, s)
			}
		}
		// create maps as needed
		segments = ns
		if len(segments) == 1 && t.orig == nil {
			t.orig = make(map[string]interface{}, 0)
			m = t.orig
		} else if len(segments) > 1 {
			if t.orig == nil {
				t.orig = make(map[string]interface{}, 0)
				m = t.orig
			}
			for i := 0; i < len(segments)-1; i++ {
				if _, ok := m.(map[string]interface{})[segments[i]]; !ok {
					m.(map[string]interface{})[segments[i]] = make(map[string]interface{}, 0)
				}
				m = m.(map[string]interface{})[segments[i]].(map[string]interface{})
			}
		}
		// insert appropriate value
		switch tv.(type) {
		case string:
			if tkPath == "/" {
				//ctx.Log().Info("expression", tv.(string), "result", j.ParseExpression(ctx, tv.(string)), "top", true)
				t.orig = j.merge(t.orig, j.ParseExpression(ctx, tv.(string)))
			} else {
				//ctx.Log().Info("expression", tv.(string), "result", j.ParseExpression(ctx, tv.(string)))
				m.(map[string]interface{})[segments[len(segments)-1]] = j.merge(m.(map[string]interface{})[segments[len(segments)-1]], j.ParseExpression(ctx, tv.(string)))
			}
		default:
			if tkPath == "/" {
				//ctx.Log().Info("tv", tv)
				t.orig = j.merge(t.orig, tv)
			} else {
				//ctx.Log().Info("tv", tv)
				m.(map[string]interface{})[segments[len(segments)-1]] = j.merge(m.(map[string]interface{})[segments[len(segments)-1]], tv)
			}
		}
	}
	t.parseMessageMap(t.orig, "")
	return t
}

// choose is a helper function which chooses array elements or map elements if they match the pattern
func (j *JDoc) choose(doc interface{}, pattern interface{}) interface{} {
	if doc == nil || pattern == nil {
		return doc
	}
	switch doc.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, 0)
		for k, v := range doc.(map[string]interface{}) {
			c, _ := j.contains(v, pattern, 0)
			if c {
				m[k] = v
			}
		}
		return m
	case []interface{}:
		list := make([]interface{}, 0)
		for _, e := range doc.([]interface{}) {
			c, _ := j.contains(e, pattern, 0)
			if c {
				list = append(list, e)
			}
		}
		return list
	default:
		c, _ := j.contains(doc, pattern, 0)
		if c {
			return doc
		} else {
			return nil
		}
	}
}

// crush is a helper function which collapses any json structure into a flat array
func (j *JDoc) crush(doc interface{}, result []interface{}) []interface{} {
	if result == nil {
		result = make([]interface{}, 0)
	}
	if doc == nil {
		return result
	}
	switch doc.(type) {
	case []interface{}:
		for _, e := range doc.([]interface{}) {
			result = j.crush(e, result)
		}
	default:
		result = append(result, doc)
	}
	return result
}

// merge is a helper function to merge maps a and b. b is merged into a and finally a is returned as the result.
func (j *JDoc) merge(a interface{}, b interface{}) interface{} {
	// we deep merge maps and shallow merge arrays - should arrays be treated as sets and be deep-merged as well?
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	switch a.(type) {
	case map[string]interface{}:
		switch b.(type) {
		case map[string]interface{}:
			//Gctx.Log().Info("action", "merging_maps", "a", a, "b", b)
			bc := make(map[string]interface{}, 0)
			for k, v := range b.(map[string]interface{}) {
				bc[k] = v
			}
			for k, va := range a.(map[string]interface{}) {
				vb := bc[k]
				bc[k] = j.merge(va, vb)
			}
			//Gctx.Log().Info("action", "merging_maps_done", "m", b)
			return bc
		case []interface{}:
			Gctx.Log().Info("action", "cannot_merge_map_with_array")
			return b
		default:
			Gctx.Log().Info("action", "cannot_merge_map_with_simple_types")
			return b
		}
	case []interface{}:
		switch b.(type) {
		case map[string]interface{}:
			Gctx.Log().Info("action", "cannot_merge_map_with_array")
			return b
		case []interface{}:
			//Gctx.Log().Info("action", "merging_arrays", "a", a, "b", b)
			res := make([]interface{}, 0)
			res = append(res, a.([]interface{})...)
			res = append(res, b.([]interface{})...)
			return res
		default:
			Gctx.Log().Info("action", "cannot_merge_array_with_simple_type")
			return b
			//return append(a.([]interface{}), b)
		}
	default:
		switch b.(type) {
		case map[string]interface{}:
			Gctx.Log().Info("action", "cannot_merge_simple_type_with_map")
			return b
		case []interface{}:
			Gctx.Log().Info("action", "cannot_merge_array_with_simple_type")
			return b
			//return append(b.([]interface{}), a)
		default:
			//l := make([]interface{}, 2)
			//l[0] = a
			//l[1] = b
			Gctx.Log().Info("action", "cannot_merge_simple_types")
			return b
		}
	}
	Gctx.Log().Info("action", "incompatible_data_types", "a", a, "b", b)
	return b
}

// contains is a helper function to check if b is contained in a (if b is a partial of a)
func (j *JDoc) contains(a interface{}, b interface{}, strength int) (bool, int) {
	if a == nil && b == nil {
		return true, strength
	}
	if a == nil || b == nil {
		return false, strength
	}
	switch b.(type) {
	case map[string]interface{}:
		switch a.(type) {
		case map[string]interface{}:
			for k, vb := range b.(map[string]interface{}) {
				c, s := j.contains(a.(map[string]interface{})[k], vb, 0)
				if !c {
					return false, strength
				}
				strength += s
			}
		case []interface{}:
			return false, strength
		default:
			return false, strength
		}
	case []interface{}:
		switch a.(type) {
		case map[string]interface{}:
			return false, strength
		case []interface{}:
			// check if all fields in b are in a (in any order)
			for _, vb := range b.([]interface{}) {
				present := false
				for _, va := range a.([]interface{}) {
					// this supports partial matches in deeply structured array elements even with wild cards etc.
					c, s := j.contains(va, vb, 0)
					if c {
						strength += s
						present = true
						break
					}
				}
				if !present {
					return false, strength
				}
			}
		default:
			return false, strength
		}
	default:
		switch a.(type) {
		case map[string]interface{}:
			return false, strength
		case []interface{}:
			return false, strength
		default:
			//return a == b
			switch b.(type) {
			// special treatment of flat string matches suporting boolean or and wild cards (mainly here for backward compatibility)
			case string:
				alts := strings.Split(b.(string), JPathOr)
				result := false
				for _, alt := range alts {
					if alt == JPathWildcard || alt == a {
						strength++
						result = true
					}
				}
				if !result {
					return false, strength
				}
			default:
				equal := a == b
				if equal {
					strength++
				}
				return equal, strength
			}
		}
	}
	return true, strength
}

// ListAllPaths list all possible path expressions in document as string slice.
func (j *JDoc) ListAllPaths() []string {
	l := make([]string, len(j.pmap))
	i := 0
	for k, _ := range j.pmap {
		l[i] = k
		i++
	}
	return l
}

// HasPath checks if a path exists in a given document.
// Example path: "/content/siteId".
func (j *JDoc) HasPath(path string) bool {
	if _, ok := j.pmap[path]; ok {
		return true
	}
	return false
}

func (j *JDoc) getChildElement(ctx Context, curr interface{}, segment string, path string) interface{} {
	//TODO: use true array paths
	//Gctx.Log().Info("segment", segment, "path", path, "curr", curr)
	if curr == nil {
		return nil
	}
	if !strings.HasSuffix(segment, "]") {
		// map element
		switch curr.(type) {
		case map[string]interface{}:
			return curr.(map[string]interface{})[segment]
		}
	} else {
		switch curr.(type) {
		case map[string]interface{}:
			// array element
			key := segment[:strings.Index(segment, "[")]
			switch curr.(map[string]interface{})[key].(type) {
			case []interface{}:
				idxStr := segment[strings.Index(segment, "[")+1 : len(segment)-1]
				if strings.Contains(idxStr, "=") {
					// array element by key selector
					kv := strings.Split(idxStr, "=")
					if len(kv) != 2 {
						ctx.Log().Error("action", "key_selector_error", "path", path)
						return nil
					}
					k := strings.TrimSpace(kv[0])
					v := strings.TrimSpace(kv[1])
					for _, ae := range curr.(map[string]interface{})[key].([]interface{}) {
						switch ae.(type) {
						case map[string]interface{}:
							if ae.(map[string]interface{})[k] == v {
								return ae
							}
						}
					}
				} else {
					// array element by index
					idx, err := strconv.Atoi(idxStr)
					if err != nil {
						ctx.Log().Error("action", "int_conversion_error", "path", path)
						return nil
					}
					if idx >= len(curr.(map[string]interface{})[key].([]interface{})) {
						ctx.Log().Error("action", "array_length_error", "path", path)
						return nil
					}
					return curr.(map[string]interface{})[key].([]interface{})[idx]
				}
			}
		case []interface{}:
			idxStr := segment[strings.Index(segment, "[")+1 : len(segment)-1]
			if strings.Contains(idxStr, "=") {
				// array element by key selector
				kv := strings.Split(idxStr, "=")
				if len(kv) != 2 {
					ctx.Log().Error("action", "key_selector_error", "path", path)
					return nil
				}
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				for _, ae := range curr.([]interface{}) {
					switch ae.(type) {
					case map[string]interface{}:
						if ae.(map[string]interface{})[k] == v {
							return ae
						}
					}
				}
			} else {
				// array element by index
				idx, err := strconv.Atoi(idxStr)
				if err != nil {
					ctx.Log().Error("action", "int_conversion_error", "path", path)
					return nil
				}
				if idx >= len(curr.([]interface{})) {
					ctx.Log().Error("action", "array_length_error", "path", path)
					return nil
				}
				return curr.([]interface{})[idx]
			}
		}
	}
	//ctx.Log.Error("action", "flat_type_error", "path", path)
	return nil
}

// evalArrayPath evaluates a jpath expression containing an array selector.
// Examples: "/content/deviceId[0]" or "/content/deviceId[foo=bar]" or "/content/deviceId[foo=b/ar]"
func (j *JDoc) evalArrayPath(ctx Context, path string) interface{} {
	//TODO: use true array paths
	inArr := false
	last := 0
	segments := make([]string, 0)
	for pos, c := range path {
		if c == '[' {
			inArr = true
		} else if c == ']' {
			inArr = false
		}
		if c == '/' && !inArr && pos > last {
			segments = append(segments, path[(last+1):pos])
			last = pos
		} else if pos == len(path)-1 && pos-(last) > 0 {
			segments = append(segments, path[last+1:len(path)])
		}
	}
	var curr interface{}
	curr = j.orig
	//Gctx.Log().Info("array_path", path, "segments", segments)
	for i := 0; i < len(segments); i++ {
		if curr == nil {
			return nil
		}
		elem := strings.TrimSpace(segments[i])
		if elem == "" {
			continue
		}
		curr = j.getChildElement(ctx, curr, elem, path)
	}
	return curr
}

// EvalPath evaluates any jpath expression (with or without array selectors).
// Example path: "/content/deviceId"
func (j *JDoc) EvalPath(ctx Context, path string) interface{} {
	//Gctx.Log().Info("path", path, "j.pmap", j.pmap, "value", j.pmap[path])
	if strings.Contains(path, "[") && strings.Contains(path, "]") {
		return j.evalArrayPath(ctx, path)
	}
	if val, ok := j.pmap[path]; ok {
		return val
	}
	return nil
}

// IsValidPath checks if a given jath is valid or not (does not check if the path exists in a given document).
// Example path: "/content/deviceId"
func (j *JDoc) IsValidPath(expr string) bool {
	if JPathSimpleReg == nil {
		JPathSimpleReg, _ = regexp.Compile(JPathSimple)
	}
	return JPathSimpleReg.MatchString(expr)
}

// IsValidPathExpression checks if a given jpath expression is valid. This includes function calls.
// Examples: "{{/content/deviceId}}" or "{{uuid()}}" etc.
func (j *JDoc) IsValidPathExpression(ctx Context, expr string) bool {
	warning := j.VetExpression(ctx, expr)
	if warning != "" {
		return false
	} else {
		return true
	}
}

// ParseExpression parsed and evaluates a given jpath expression. Results are returned as interface.
// Examples: "{{/content/deviceId}}" or "{{uuid()}}" etc.
func (j *JDoc) ParseExpression(ctx Context, expr interface{}) interface{} {
	if expr == nil {
		return nil
	}
	switch expr.(type) {
	case string:
		prsr, _ := NewJExpr(expr.(string))
		return prsr.Execute(ctx, j)
	default:
		return expr
	}
}

// VetExpression checks if a given jpath expression is valid and returns an error string if it isn't.
// Examples: "{{/content/deviceId}}" or "{{uuid()}}" etc.
func (j *JDoc) VetExpression(ctx Context, expr interface{}) string {
	if expr == nil {
		return ""
	}
	switch expr.(type) {
	case string:
		_, err := NewJExpr(expr.(string))
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

func (j *JDoc) GetValue(path string) interface{} {
	if j.pmap == nil || j.orig == nil {
		return ""
	}
	return j.pmap[path]
}

func (j *JDoc) GetValueForExpression(ctx Context, expr string) interface{} {
	return j.ParseExpression(ctx, expr)
}

func (j *JDoc) GetMapValue(path string) map[string]interface{} {
	if j.pmap == nil || j.orig == nil {
		return nil
	}
	switch j.pmap[path].(type) {
	case map[string]interface{}:
		return j.pmap[path].(map[string]interface{})
	}
	return nil
}

func (j *JDoc) GetMapValueForExpression(ctx Context, expr string) map[string]interface{} {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case map[string]interface{}:
		return er.(map[string]interface{})
	}
	return nil
}

func (j *JDoc) GetStringValue(path string) string {
	if j.pmap == nil || j.orig == nil {
		return ""
	}
	switch j.pmap[path].(type) {
	case string:
		return j.pmap[path].(string)
	}
	return ""
}

func (j *JDoc) GetStringValueForExpression(ctx Context, expr string) string {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case string:
		return er.(string)
	}
	return ""
}

func (j *JDoc) GetSliceValue(path string) []interface{} {
	if j.pmap == nil || j.orig == nil {
		return nil
	}
	switch j.pmap[path].(type) {
	case []interface{}:
		return j.pmap[path].([]interface{})
	}
	return nil
}

func (j *JDoc) GetSliceValueForExpression(ctx Context, expr string) []interface{} {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case []interface{}:
		return er.([]interface{})
	}
	return nil
}

func (j *JDoc) GetStringSliceValue(path string) []string {
	if j.pmap == nil || j.orig == nil {
		return nil
	}
	switch j.pmap[path].(type) {
	case []interface{}:
		res := make([]string, 0)
		for _, s := range j.pmap[path].([]interface{}) {
			switch s.(type) {
			case string:
				res = append(res, s.(string))
			}
		}
		return res
	}
	return nil
}

func (j *JDoc) GetStringSliceValueForExpression(ctx Context, expr string) []string {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case []interface{}:
		res := make([]string, 0)
		for _, s := range er.([]interface{}) {
			switch s.(type) {
			case string:
				res = append(res, s.(string))
			}
		}
		return res
	}
	return nil
}

func (j *JDoc) GetFloatValue(path string) float64 {
	if j.pmap == nil || j.orig == nil {
		return 0.0
	}
	switch j.pmap[path].(type) {
	case float64:
		return j.pmap[path].(float64)
	case float32:
		return float64(j.pmap[path].(float32))
	}
	return 0.0
}

func (j *JDoc) GetFloatValueForExpression(ctx Context, expr string) float64 {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case float64:
		return er.(float64)
	case float32:
		return float64(er.(float32))
	}
	return 0.0
}

func (j *JDoc) GetIntValue(path string) int {
	if j.pmap == nil || j.orig == nil {
		return 0
	}
	switch j.pmap[path].(type) {
	case int:
		return j.pmap[path].(int)
	}
	return 0
}

func (j *JDoc) GetIntValueForExpression(ctx Context, expr string) int {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case float64:
		return er.(int)
	}
	return 0
}

func (j *JDoc) GetNumberValue(path string) float64 {
	if j.pmap == nil || j.orig == nil {
		return 0.0
	}
	switch j.pmap[path].(type) {
	case int:
		return float64(j.pmap[path].(int))
	case float64:
		return j.pmap[path].(float64)
	case float32:
		return float64(j.pmap[path].(float32))
	}
	return 0.0
}

func (j *JDoc) GetNumberValueForExpression(ctx Context, expr string) float64 {
	er := j.ParseExpression(ctx, expr)
	switch er.(type) {
	case int:
		return float64(er.(int))
	case float64:
		return er.(float64)
	case float32:
		return float64(er.(float32))
	}
	return 0.0
}

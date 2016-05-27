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
	"errors"
	"fmt"
	"strconv"
	"strings"

	. "github.com/Comcast/eel/eel/util"
)

type astType int

const (
	astPath     astType = iota // 0 simple path
	astFunction                // 1 curl etc.
	astParam                   // 2 'foo'
	astText                    // 3 plain text
	astAgg                     // 4 concatenation of sub elements
)

// JExprItem represents a node in an abstract syntax tree of a parsed jpath expression. Pointer to root node also used as handle for a parser instance.
type JExprItem struct {
	typ      astType
	val      interface{}
	kids     []*JExprItem
	exploded bool
	mom      *JExprItem
}

// JExprD3Node represents a simplified version of a node on the abstract syntax tree for debug and visulaization purposes.
type JExprD3Node struct {
	Name     string         `json:"name"`
	Parent   string         `json:"parent"`
	Children []*JExprD3Node `json:"children"`
}

func newJExprItem(typ astType, val string, mom *JExprItem) *JExprItem {
	return &JExprItem{typ, val, make([]*JExprItem, 0), false, mom}
}

func newJExprParser() *JExprItem {
	return newJExprItem(astAgg, "", nil)
}

// NewJExpr parses (but does not execute) a jpath expression and returns handle.
func NewJExpr(expr string) (*JExprItem, error) {
	ast := newJExprParser()
	err := ast.parse(expr)
	if err == nil {
		for {
			exploded, err := ast.explodeParams()
			if err != nil {
				return ast, err
			}
			if !exploded {
				break
			}
		}
	}
	return ast, err
}

// GetD3Json returns a simplified AST for display with D3
func (a *JExprItem) GetD3Json(cur *JExprD3Node) *JExprD3Node {
	if cur == nil {
		cur = new(JExprD3Node)
	}
	cur.Name = a.typeString() + " " + ToFlatString(a.val)
	if a.mom != nil {
		cur.Parent = a.mom.typeString() + " " + ToFlatString(a.val)
	}
	if cur.Children == nil {
		cur.Children = make([]*JExprD3Node, 0)
	}
	for _, k := range a.kids {
		c := new(JExprD3Node)
		cur.Children = append(cur.Children, c)
		k.GetD3Json(c)
	}
	return cur
}

// parse parses a jpath expression for validation purposes only. Returns an error (if any) or nil.
func (a *JExprItem) parse(expr string) error {
	_, c := lex("", expr)
	var f *JExprItem
	var fnc *JFunction
	for item := range c {
		switch item.typ {
		case lexItemFunction:
			fnc = NewFunction(item.val)
			if fnc == nil {
				return errors.New("unknown function " + item.val)
			}
			f = newJExprItem(astFunction, item.val, a)
			a.kids = append(a.kids, f)
		case lexItemText:
			t := newJExprItem(astText, item.val, a)
			a.kids = append(a.kids, t)
		case lexItemParam:
			if f != nil {
				p := newJExprItem(astParam, item.val, f)
				f.kids = append(f.kids, p)
			} else {
				return errors.New("param without function")
			}
		case lexItemRightBracket:
			if f != nil {
				if len(f.kids) < fnc.minNumParams || len(f.kids) > fnc.maxNumParams {
					return errors.New("wrong number of params for function " + f.val.(string))
				}
				f = nil
			} else {
				return errors.New("closing bracket without function")
			}
		case lexItemPath:
			if strings.HasPrefix(item.val, "/") || item.val == "." {
				p := newJExprItem(astPath, item.val, a)
				a.kids = append(a.kids, p)
			} else {
				return errors.New("invalid path " + item.val)
			}
		case lexItemError:
			return errors.New(item.val)
		}
	}
	return nil
}

func (a *JExprItem) typeString() string {
	switch a.typ {
	case astPath:
		return "PATH"
	case astFunction:
		return "FUNCTION"
	case astParam:
		return "PARAM"
	case astText:
		return "TEXT"
	case astAgg:
		return "AGG"
	default:
		return "UNKNOWN"
	}
}

// print prints AST for debugging
func (a *JExprItem) print(level int) {
	fmt.Printf("level: %d\ttype: %s\tkids: %d\t val: %v", level, a.typeString(), len(a.kids), a.val)
	if a.mom != nil {
		fmt.Printf("\tmom: %s\t%s\n", a.mom.typeString(), a.mom.val)
	} else {
		fmt.Printf("\tmom: nil\n")
	}
	for _, k := range a.kids {
		k.print(level + 1)
	}
}

func (a *JExprItem) list(level int, list [][]string) [][]string {
	if list == nil {
		list = make([][]string, 0)
	}
	row := make([]string, 0)
	row = append(row, strconv.Itoa(level))
	row = append(row, a.typeString())
	row = append(row, strconv.Itoa(len(a.kids)))
	row = append(row, ToFlatString(a.val))
	if a.mom != nil {
		row = append(row, a.mom.typeString())
		row = append(row, ToFlatString(a.mom.val))
	} else {
		row = append(row, "")
		row = append(row, "")
	}
	list = append(list, row)
	for _, k := range a.kids {
		list = k.list(level+1, list)
	}
	return list
}

// explodeParams recursively builds an AST for the given jpath expression
func (a *JExprItem) explodeParams() (bool, error) {
	exploded := false
	var err error
	if a.typ == astParam && !a.exploded {
		valStr := ToFlatString(a.val)
		if strings.Contains(valStr, leftMeta) || strings.Contains(valStr, rightMeta) {
			a.typ = astAgg
			//err = a.parse(extractStringParam(valStr))
			err = a.parse(valStr)
			if err != nil {
				return false, err
			}
			a.val = ""
			a.exploded = true
			exploded = true
		}
	}
	for _, k := range a.kids {
		explodedKid, err := k.explodeParams()
		if err != nil {
			return false, err
		}
		exploded = exploded || explodedKid
	}
	return exploded, err
}

func (a *JExprItem) isLeaf() bool {
	if a.kids == nil || len(a.kids) == 0 {
		return true
	}
	return false
}

func (a *JExprItem) isRoot() bool {
	if a.mom == nil {
		return true
	}
	return false
}

// collapseLeaf collapes a single leaf of the AST
func (a *JExprItem) collapseLeaf(ctx Context, doc *JDoc) bool {
	if !a.isLeaf() {
		return false
	}
	switch a.typ {
	case astPath: // simple path selector
		if a.mom == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		a.val = doc.EvalPath(ctx, ToFlatString(a.val)) // retain type of selected path
		if a.val == nil {
			a.val = ""
		}
		if a.mom.typ == astAgg {
			a.typ = astText
		} else if a.mom.typ == astFunction {
			a.typ = astParam
		} else {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "wrong_type", "val", a.val, "type", a.typeString())
		}
		return true
	case astFunction: // parameter-less function
		if a.mom == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		f := NewFunction(ToFlatString(a.val))
		if f == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "unknown_function", "val", a.val, "type", a.typeString())
			return false
		}
		// odd: currently parameter-less functions require a single blank string as parameter
		param := NewJParam("")
		a.val = f.ExecuteFunction(ctx, doc, []*JParam{param}) // retain type of function return values
		if a.val == nil {
			a.val = ""
		}
		if a.mom.typ == astAgg {
			a.typ = astText
		} else if a.mom.typ == astFunction {
			a.typ = astParam
		}
		return true
	case astParam: // function with parameters
		if a.mom == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.mom == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "grandma_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.typ != astFunction {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_must_be_function", "val", a.val, "type", a.typeString())
			return false
		}
		f := NewFunction(ToFlatString(a.mom.val))
		if f == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "unknown_function", "val", a.val, "type", a.typeString())
			return false
		}
		params := make([]*JParam, 0)
		for _, k := range a.mom.kids { // only execute if all paramters are ready
			if k.typ != astParam {
				return false
			}
			p := NewJParam(ToFlatString(k.val))
			if debugLexer {
				p.Log()
			}
			params = append(params, p)
		}
		a.mom.val = f.ExecuteFunction(ctx, doc, params) // retain type of function return values
		a.mom.kids = make([]*JExprItem, 0)
		if a.mom.mom.typ == astFunction {
			a.mom.typ = astParam
		} else if a.mom.mom.typ == astAgg {
			a.mom.typ = astText
		} else {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_must_be_function_or_aggregation", "val", a.val, "type", a.typeString())
		}
		return true
	case astText: // aggregation of text fields
		if a.mom == nil {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.typ != astAgg {
			//ctx.Log.Debug("event", "ast_collapse_failure", "reason", "mom_must_be_aggregation", "val", a.val, "type", a.typeString())
			return false
		}
		if len(a.mom.kids) == 1 { // retain type for single value aggregations, except for nil which is converted to ""
			if a.mom.mom != nil && a.mom.mom.typ == astFunction {
				//a.mom.val = "'" + ToFlatString(a.val) + "'"
				a.mom.val = ToFlatString(a.val)
				a.mom.typ = astParam
			} else {
				a.mom.val = a.val
				a.mom.typ = astText
			}
			if a.mom.val == nil {
				a.mom.val = ""
			}
		} else { // for multiple value aggregations, convert to string and concatenate
			txt := ""
			for _, k := range a.mom.kids { // only aggregate if all texts ready
				if k.typ != astText {
					return false
				}
				txt += ToFlatString(k.val)
			}
			if a.mom.mom != nil && a.mom.mom.typ == astFunction {
				a.mom.val = "'" + txt + "'"
				a.mom.typ = astParam
			} else {
				a.mom.val = txt
				a.mom.typ = astText
			}
		}
		a.mom.kids = make([]*JExprItem, 0)
		return true
	}
	return false
}

func (a *JExprItem) collapseLeaves(ctx Context, doc *JDoc, ast *JExprItem, trees *[][][]string) bool {
	collapsed := false
	// it is ok to modify tree while we traverse because we are starting from the leaves
	for _, k := range a.kids {
		collapsed = collapsed || k.collapseLeaves(ctx, doc, ast, trees)
	}
	if a.isLeaf() {
		cl := a.collapseLeaf(ctx, doc)
		collapsed = collapsed || cl
		if trees != nil && cl {
			*trees = append(*trees, ast.list(0, nil))
		}
		//fmt.Printf("collapsed:\n")
		//ast.print(0)
	}
	return collapsed
}

// CollapseNextLeafDebug is used for step-debugging of a jpath expression.
func (a *JExprItem) CollapseNextLeafDebug(ctx Context, doc *JDoc) bool {
	for _, k := range a.kids {
		if k.CollapseNextLeafDebug(ctx, doc) {
			return true
		}
	}
	if a.isLeaf() {
		return a.collapseLeaf(ctx, doc)
	}
	return false
}

func (a *JExprItem) countItems(cnt *int) int {
	(*cnt)++
	for _, k := range a.kids {
		k.countItems(cnt)
	}
	return *cnt
}

// Execute executes parsed jpath expression and returns result as interface.
func (a *JExprItem) Execute(ctx Context, doc *JDoc) interface{} {
	//fmt.Printf("ast:\n")
	//a.print(0)
	for {
		if !a.collapseLeaves(ctx, doc, a, nil) {
			break
		}
	}
	return a.val
}

// ExecuteDebug executes parsed jpath expression in debug mode and returns result as interface as well as detailed tabular debug information.
func (a *JExprItem) ExecuteDebug(ctx Context, doc *JDoc) (interface{}, [][][]string) {
	trees := make([][][]string, 0)
	trees = append(trees, a.list(0, nil))
	for {
		if !a.collapseLeaves(ctx, doc, a, &trees) {
			break
		}
	}
	return a.val, trees
}

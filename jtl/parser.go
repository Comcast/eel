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

	. "github.com/Comcast/eel/util"
)

type astType int

const (
	astPath     astType = iota // 0 simple path
	astFunction                // 1 curl etc.
	astParam                   // 2 'foo'
	astText                    // 3 plain text
	astAgg                     // 4 concatenation of sub elements
)

const (
	parserDebug = false
)

// JExprItem represents a node in an abstract syntax tree of a parsed jpath expression. Pointer to root node also used as handle for a parser instance.
type JExprItem struct {
	typ      astType
	val      interface{}
	kids     []*JExprItem
	exploded bool
	mom      *JExprItem
	level    int // level in AST
}

// JExprD3Node represents a simplified version of a node on the abstract syntax tree for debug and visulaization purposes.
type JExprD3Node struct {
	Name     string         `json:"name"`
	Parent   string         `json:"parent"`
	Children []*JExprD3Node `json:"children"`
}

func newJExprItem(typ astType, val string, mom *JExprItem) *JExprItem {
	return &JExprItem{typ, val, make([]*JExprItem, 0), false, mom, 0}
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

// parse parses a single jpath expression. Returns an error (if any) or nil.
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
				p := newJExprItem(astParam, item.val, f) //strings.Replace(item.val, "\\", "", -1), f)
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

// updateLevels updates the level parameters of all nodes in the AST tree. Call on root node with level 0.
func (a *JExprItem) updateLevels(level int) {
	a.level = level
	for _, k := range a.kids {
		k.updateLevels(level + 1)
	}
}

// getDeepestConditional gets the lowest level conditional (ifte(), case(), alt()) for AST optimization
func (a *JExprItem) getDeepestConditional(cond **JExprItem) *JExprItem {
	if cond == nil {
		cond = new(*JExprItem)
	}
	if a.typ == astFunction && (a.val == "ifte" || a.val == "alt" || a.val == "case") {
		if *cond == nil {
			*cond = a
		} else if a.level > (*cond).level {
			*cond = a
		}
	}
	for _, k := range a.kids {
		k.getDeepestConditional(cond)
	}
	return *cond
}

// getHighestConditional gets the highest level conditional (ifte(), case(), alt()) for AST optimization
func (a *JExprItem) getHighestConditional() *JExprItem {
	if a.typ == astFunction && (a.val == "ifte" || a.val == "alt" || a.val == "case") {
		return a
	}
	for _, k := range a.kids {
		c := k.getHighestConditional()
		if c != nil {
			return c
		}
	}
	return nil
}

// optimizeAllConditionals oprtstrd on the entire AST to optimize all conditional sub trees bottom up
func (a *JExprItem) optimizeAllConditionals(ctx Context, doc *JDoc) error {
	a.updateLevels(0)
	for {
		// unfortunately we cannot use getHigestConditional() to go top-down here because eel
		// allows expressions like "if ( if (a==a) then a else b ) == c then d"
		cond := a.getDeepestConditional(nil)
		if cond == nil {
			break
		}
		cond.print(0, "CONDITION")
		_, err := cond.optimizeConditional(ctx, doc)
		if err != nil {
			return err
		}
		a.updateLevels(0)
		a.print(0, "OPTIMIZEDTREE")
	}
	return nil
}

func (a *JExprItem) adjustParameter() {
	if a.typ == astParam {
		a.typ = astText
		switch a.val.(type) {
		case string:
			valStr := a.val.(string)
			if len(valStr) >= 2 && strings.HasPrefix(valStr, "'") && strings.HasSuffix(valStr, "'") {
				valStr = valStr[1 : len(valStr)-1]
				a.val = valStr
			}
		}
	}
}

func (a *JExprItem) collapseIntoSingleNode(ctx Context, doc *JDoc) {
	for {
		if !a.collapseLeaves(ctx, doc, a, nil) {
			break
		}
	}
}

func (a *JExprItem) getMotherIdx() int {
	if a.mom == nil {
		return -1
	}
	for idx, c := range a.mom.kids {
		if a == c {
			return idx
		}
	}
	return -1
}

// optimizeConditional works on the lowest conditional in the AST (obtained with getDeepestConditional), evaluates the condition and replaces it
// with the proper results to ptimize the AST
func (a *JExprItem) optimizeConditional(ctx Context, doc *JDoc) (*JExprItem, error) {
	//TODO: debug support
	stats := ctx.Value(EelTotalStats).(*ServiceStats)
	if a.typ == astFunction && a.val == "ifte" {
		if len(a.kids) != 3 {
			ctx.Log().Error("error_type", "parser", "cause", "wrong_number_of_parameters", "type", a.typ, "val", a.val, "num_params", len(a.kids), "error", "wrong number of parameters")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("wrong number of parameters"), "ifte", nil})
			return nil, errors.New("ifte has wrong number of parameters")
		}
		if a.mom == nil {
			ctx.Log().Error("error_type", "parser", "cause", "conditional_orphan", "type", a.typ, "val", a.val, "error", "conditional orphan")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("conditional orphan"), "ifte", nil})
			return nil, errors.New("conditional orphan")
		}
		childIdx := a.getMotherIdx()
		// detach condition node
		cond := a.kids[0]
		cond.mom = nil
		// evaluate condition and restructure AST
		cond.collapseIntoSingleNode(ctx, doc)
		// pull up the chosen node and adjust some metadata
		var chosenChild *JExprItem
		if cond.val == true || cond.val == "true" || cond.val == "'true'" {
			chosenChild = a.kids[1]
		} else if cond.val == false || cond.val == "false" || cond.val == "'false'" {
			chosenChild = a.kids[2]
		} else {
			ctx.Log().Error("error_type", "parser", "cause", "non_boolean_condition", "type", cond.typ, "val", cond.val, "error", "non boolean condition")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("non-boolean condition"), "ifte", nil})
			return nil, errors.New("non-boolean condition")
		}
		chosenChild.adjustParameter()
		// hack to resurrect json inteface type
		switch chosenChild.val.(type) {
		case string:
			strVal := chosenChild.val.(string)
			if strings.Contains(strVal, "{") && strings.Contains(strVal, "}") {
				obj, err := NewJDocFromString(strVal)
				if err != nil {
					ctx.Log().Error("error_type", "parser", "cause", "invalid_json", "type", cond.typ, "val", cond.val, "error", "invalid json")
					stats.IncErrors()
					AddError(ctx, RuntimeError{fmt.Sprintf("non json parameter"), "ifte", nil})
					return nil, errors.New("non json parameters in call to ifte function")
				}
				chosenChild.val = obj.GetOriginalObject()
			}
		}
		// end hack
		a.mom.kids[childIdx] = chosenChild
		chosenChild.mom = a.mom
		chosenChild.print(0, "CHOSENCHILD")
		return a.mom.kids[childIdx], nil
	} else if a.typ == astFunction && a.val == "alt" {
		if len(a.kids) < 2 {
			ctx.Log().Error("error_type", "parser", "cause", "wrong_number_of_parameters", "type", a.typ, "val", a.val, "num_params", len(a.kids), "error", "wrong number of parameters")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("wrong number of parameters"), "alt", nil})
			return nil, errors.New("alt has wrong number of parameters")
		}
		if a.mom == nil {
			ctx.Log().Error("error_type", "parser", "cause", "conditional_orphan", "type", a.typ, "val", a.val, "error", "conditional orphan")
			return nil, errors.New("conditional orphan")
		}
		childIdx := a.getMotherIdx()
		for _, cand := range a.kids {
			cand.mom = nil
			cand.collapseIntoSingleNode(ctx, doc)
			if cand.val != "''" && cand.val != "" {
				cand.adjustParameter()
				a.mom.kids[childIdx] = cand
				cand.mom = a.mom
				cand.print(0, "CHOSENCHILD")
				return a.mom.kids[childIdx], nil
			}
		}
		// otherwise enter blank default node
		chosenChild := new(JExprItem)
		chosenChild.mom = a.mom
		chosenChild.typ = astText
		chosenChild.val = ""
		a.mom.kids[childIdx] = chosenChild
		chosenChild.print(0, "CHOSENCHILD")
		return a.mom.kids[childIdx], nil
	} else if a.typ == astFunction && a.val == "case" {
		if len(a.kids) < 3 || len(a.kids)%3 > 1 {
			ctx.Log().Error("error_type", "parser", "cause", "wrong_number_of_parameters", "type", a.typ, "val", a.val, "num_params", len(a.kids), "error", "wrong number of parameters")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("wrong number of parameters"), "case", nil})
			return nil, errors.New("case has wrong number of parameters")
		}
		if a.mom == nil {
			ctx.Log().Error("error_type", "parser", "cause", "conditional_orphan", "type", a.typ, "val", a.val, "error", "conditional orphan")
			stats.IncErrors()
			AddError(ctx, RuntimeError{fmt.Sprintf("conditional orphan"), "case", nil})
			return nil, errors.New("conditional orphan")
		}
		childIdx := a.getMotherIdx()
		for i := 0; i < len(a.kids)/3; i++ {
			candA := a.kids[i*3]
			candA.mom = nil
			candA.collapseIntoSingleNode(ctx, doc)
			candA.adjustParameter()
			candB := a.kids[i*3+1]
			candB.mom = nil
			candB.collapseIntoSingleNode(ctx, doc)
			candB.adjustParameter()
			//ctx.Log().Debug("action", "eel_parser", "val_a", candA.val, "val_b", candB.val)
			if candA.val == candB.val ||
				(candA.val == true && candB.val == "true") ||
				(candA.val == false && candB.val == "false") ||
				(candA.val == "true" && candB.val == true) ||
				(candA.val == "false" && candB.val == false) {
				chosenChild := a.kids[i*3+2]
				chosenChild.adjustParameter()
				a.mom.kids[childIdx] = chosenChild
				chosenChild.mom = a.mom
				chosenChild.print(0, "CHOSENCHILD")
				return a.mom.kids[childIdx], nil
			}
		}
		if len(a.kids)%3 == 1 {
			chosenChild := a.kids[len(a.kids)-1]
			chosenChild.adjustParameter()
			a.mom.kids[childIdx] = chosenChild
			chosenChild.mom = a.mom
			chosenChild.print(0, "CHOSENCHILD")
			return a.mom.kids[childIdx], nil
		}
		// otherwise enter blank default node
		chosenChild := new(JExprItem)
		chosenChild.mom = a.mom
		chosenChild.typ = astText
		chosenChild.val = ""
		a.mom.kids[childIdx] = chosenChild
		chosenChild.print(0, "CHOSENCHILD")
		return a.mom.kids[childIdx], nil
	} else {
		ctx.Log().Error("error_type", "parser", "cause", "unsupported_conditional", "type", a.typ, "val", a.val, "error", "unsupported conditional")
		stats.IncErrors()
		AddError(ctx, RuntimeError{fmt.Sprintf("unsupported conditional"), "", nil})
		return nil, errors.New("unsupported conditional")
	}
	return nil, nil
}

// print prints AST for debugging
func (a *JExprItem) print(level int, title string) {
	if !parserDebug {
		return
	}
	if title != "" && level == 0 {
		fmt.Println(title)
	}
	fmt.Printf("level: %d\ttype: %s\tkids: %d\t val: %v", level, a.typeString(), len(a.kids), a.val)
	if a.mom != nil {
		fmt.Printf("\tmom: %s\t%s\n", a.mom.typeString(), a.mom.val)
	} else {
		fmt.Printf("\tmom: nil\n")
	}
	for _, k := range a.kids {
		k.print(level+1, title)
	}
	if level == 0 {
		fmt.Println()
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
			err = a.parse(extractStringParam(valStr))
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

// collapseLeaf collapses a single leaf of the AST
func (a *JExprItem) collapseLeaf(ctx Context, doc *JDoc) bool {
	if !a.isLeaf() {
		return false
	}
	switch a.typ {
	case astPath: // simple path selector
		if a.mom == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
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
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "wrong_type", "val", a.val, "type", a.typeString())
		}
		return true
	case astFunction: // parameter-less function
		if a.mom == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		f := NewFunction(ToFlatString(a.val))
		if f == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "unknown_function", "val", a.val, "type", a.typeString())
			return false
		}
		// odd: currently parameter-less functions require a single blank string as parameter
		a.val = f.ExecuteFunction(ctx, doc, []string{""}) // retain type of function return values
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
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.mom == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "grandma_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.typ != astFunction {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_must_be_function", "val", a.val, "type", a.typeString())
			return false
		}
		f := NewFunction(ToFlatString(a.mom.val))
		if f == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "unknown_function", "val", a.val, "type", a.typeString())
			return false
		}
		params := make([]string, 0)
		for _, k := range a.mom.kids { // only execute if all parameters are ready
			if k.typ != astParam {
				return false
			}
			params = append(params, ToFlatString(k.val))
		}
		a.mom.val = f.ExecuteFunction(ctx, doc, params) // retain type of function return values
		a.mom.kids = make([]*JExprItem, 0)
		if a.mom.mom.typ == astFunction {
			a.mom.typ = astParam
		} else if a.mom.mom.typ == astAgg {
			a.mom.typ = astText
		} else {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_must_be_function_or_aggregation", "val", a.val, "type", a.typeString())
		}
		return true
	case astText: // aggregation of text fields
		if a.mom == nil {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_is_nil", "val", a.val, "type", a.typeString())
			return false
		}
		if a.mom.typ != astAgg {
			//ctx.Log().Debug("action", "ast_collapse_failure", "reason", "mom_must_be_aggregation", "val", a.val, "type", a.typeString())
			return false
		}
		if len(a.mom.kids) == 1 { // retain type for single value aggregations, except for nil which is converted to ""
			if a.mom.mom != nil && a.mom.mom.typ == astFunction {
				a.mom.val = "'" + ToFlatString(a.val) + "'"
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
		ast.print(0, "COLLAPSEDLEAF")
	}
	return collapsed
}

// CollapseNextLeafDebug is used for step-debugging of a jpath expression.
func (a *JExprItem) CollapseNextLeafDebug(ctx Context, doc *JDoc) bool {
	//TODO: add support for conditional optimization
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
	a.print(0, "AST")
	a.optimizeAllConditionals(ctx, doc)
	for {
		if !a.collapseLeaves(ctx, doc, a, nil) {
			break
		}
	}
	a.print(0, "AST")
	// remove escape indicators in final step
	switch a.val.(type) {
	case string:
		a.val = strings.Replace(a.val.(string), "\\", "", -1)
	}
	return a.val
}

// ExecuteDebug executes parsed jpath expression in debug mode and returns result as interface as well as detailed tabular debug information.
func (a *JExprItem) ExecuteDebug(ctx Context, doc *JDoc) (interface{}, [][][]string) {
	trees := make([][][]string, 0)
	trees = append(trees, a.list(0, nil))
	a.print(0, "AST")
	a.optimizeAllConditionals(ctx, doc)
	for {
		if !a.collapseLeaves(ctx, doc, a, &trees) {
			break
		}
	}
	a.print(0, "AST")
	// remove escape indicators in final step
	switch a.val.(type) {
	case string:
		a.val = strings.Replace(a.val.(string), "\\", "", -1)
	}
	return a.val, trees
}

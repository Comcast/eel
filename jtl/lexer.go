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
	"fmt"
	"strings"
	"unicode/utf8"
)

// lexer and parser for EEL based on http://cuddle.googlecode.com/hg/talk/lex.html#title-slide

// stateFn represents the state of the scanner
// as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexItem represents a token returned from the scanner.
type lexItem struct {
	typ lexItemType // type such as lexItemFunction.
	val string      // value, such as "uuid".
}

// lexItemType identifies the type of lex items.
type lexItemType int

const (
	lexItemError        lexItemType = iota // 0 error occurred, value is text of error
	lexItemEOF                             // 1
	lexItemLeftMeta                        // 2 left meta-string
	lexItemRightMeta                       // 3 right meta-string
	lexItemLeftBracket                     // 4 (
	lexItemRightBracket                    // 5 )
	lexItemPath                            // 6 simple jpath
	lexItemFunction                        // 7 curl etc.
	lexItemParam                           // 8 'foo'
	lexItemDot                             // 9 the cursor, spelled '.'
	lexItemText                            // 10 plain text
	lexItemInsideText                      // 11
)

func (i *lexItem) typeString() string {
	switch i.typ {
	case lexItemError:
		return "ERROR"
	case lexItemEOF:
		return "EOF"
	case lexItemLeftMeta:
		return "LMETA"
	case lexItemRightMeta:
		return "RMETA"
	case lexItemLeftBracket:
		return "LBRACKET"
	case lexItemRightBracket:
		return "RBRACKET"
	case lexItemPath:
		return "PATH"
	case lexItemFunction:
		return "FUNCTION"
	case lexItemParam:
		return "PARAM"
	case lexItemDot:
		return "DOT"
	case lexItemText:
		return "TEXT"
	case lexItemInsideText:
		return "INTEXT"
	default:
		return "UNKNOWN"
	}
}

func (i *lexItem) print() {
	fmt.Printf("type:%s\ttoken: %s\n", i.typeString(), i.String())
}

const (
	leftMeta         = "{{"
	rightMeta        = "}}"
	escapedLeftMeta  = "${{"
	escapedRightMeta = "$}}"
)

func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' }

func isLetter(ch rune) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

func isAlphaNumeric(ch rune) bool { return isLetter(ch) || isDigit(ch) }

// eof represents a marker rune for the end of the reader.
var eof = rune(0)

func (i lexItem) String() string {
	switch i.typ {
	case lexItemEOF:
		return "EOF"
	case lexItemError:
		return i.val
	}
	return fmt.Sprintf("%q", i.val)
}

// lexer holds the state of the scanner.
type lexer struct {
	name  string       // used only for error reports.
	input string       // the string being scanned.
	start int          // start position of this item.
	pos   int          // current position in the input.
	width int          // width of last rune read from input.
	items chan lexItem // channel of scanned items.
}

func lex(name, input string) (*lexer, chan lexItem) {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan lexItem),
	}
	go l.run() // Concurrently run state machine.
	return l, l.items
}

// run lexes the input by executing state functions until the state is nil.
func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// emit passes an item back to the client.
func (l *lexer) emit(t lexItemType) {
	// leave the original text in its own flavor, and don't trim tab and newline and white space
	token := l.input[l.start:l.pos]
	//token = strings.TrimLeft(token, "\t\n")
	//token = strings.TrimRight(token, "\t\n")
	// handle escapes
	if token == escapedLeftMeta {
		token = leftMeta
	} else if token == escapedRightMeta {
		token = rightMeta
	}
	l.items <- lexItem{t, token}
	l.start = l.pos
}

func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], leftMeta) {
			if l.pos > l.start {
				l.emit(lexItemText)
			}
			return lexLeftMeta
		}
		// escaped left meta
		if strings.HasPrefix(l.input[l.pos:], escapedLeftMeta) {
			if l.pos > l.start {
				l.emit(lexItemText)
			}
			l.next()
			l.next()
			l.next()
			l.emit(lexItemText)
			return lexText
		}
		if strings.HasPrefix(l.input[l.pos:], rightMeta) {
			return l.errorf("unexpected closing action at %d: %s\n", l.pos, l.input)
		}
		// escaped right meta
		if strings.HasPrefix(l.input[l.pos:], escapedRightMeta) {
			if l.pos > l.start {
				l.emit(lexItemText)
			}
			l.next()
			l.next()
			l.next()
			l.emit(lexItemText)
			return lexText
		}
		switch r := l.next(); {
		case r == eof:
			if l.pos > l.start {

				l.emit(lexItemText)
			}
			l.emit(lexItemEOF)
			return nil
		case r == '\\':
			l.next()
			l.next()
		}
	}
}

func lexLeftMeta(l *lexer) stateFn {
	if l.next() == '{' && l.next() == '{' {
		l.emit(lexItemLeftMeta)
		return lexInsideAction
	}
	return l.errorf("expected left meta at %d: %s\n", l.pos, l.input)
}

func lexRightMeta(l *lexer) stateFn {
	if l.next() == '}' && l.next() == '}' {
		l.emit(lexItemRightMeta)
		return lexText
	}
	return l.errorf("expected right meta at %d: %s\n", l.pos, l.input)
}

func lexPath(l *lexer) stateFn {
	l.acceptRun("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890/-_,.:|!@#$%^&*+=<>?[] ")
	l.emit(lexItemPath)
	return lexRightMeta
}

func lexParam(l *lexer) stateFn {
	bc := 0 // bracket count
	for {
		// push across nested function calls - these will be dealt with in separate
		// recursive calls to the lexer by the parser - this may seem a bit unorthodox
		// but it works very well
		if bc < 0 {
			return l.errorf("unbalanced brackets at %d: %s\n", l.pos, l.input)
		}
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unclosed param at %d: %s\n", l.pos, l.input)
		case r == '{':
			bc++
			break
		case r == '}':
			bc--
			break
		case r == '\\':
			// skip over character following escape character
			l.next()
			l.next()
			break
		case r == '\'' && bc == 0:
			l.emit(lexItemParam)
			return lexParamList
		}
	}
}

func lexOpenParamList(l *lexer) stateFn {
	if l.next() == '(' {
		l.emit(lexItemLeftBracket)
	} else {
		l.errorf("missing opening bracket at %d: %s\n", l.pos, l.input)
	}
	return lexParamList
}

func lexParamList(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unclosed function at %d: %s\n", l.pos, l.input)
		case isWhitespace(r):
			l.ignore()
			break
		case r == ')':
			l.emit(lexItemRightBracket)
			return lexRightMeta
		case r == ',':
			l.ignore()
			break
		case r == '\\':
			// skip over character following escape character
			l.next()
			l.next()
			break
		case r == '\'':
			return lexParam
		}
	}
}

func lexInsideAction(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unclosed action at %d: %s\n", l.pos, l.input)
		case isWhitespace(r):
			l.ignore()
		case r == '/':
			l.backup()
			return lexPath
		case r == '.':
			l.emit(lexItemDot)
			return lexRightMeta
		case r == '(':
			l.backup()
			l.emit(lexItemFunction)
			return lexOpenParamList
		case r == '}':
			l.errorf("illegal path at %d: %s\n", l.pos, l.input)
		}
	}
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	rune := l.next()
	l.backup()
	return rune
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan
// by passing back a nil pointer that will be the next
// state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- lexItem{
		lexItemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

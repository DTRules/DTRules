// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package el provides an Expression Language (EL) compiler that converts
// human-readable expressions to postfix notation for DTRules.
//
// EL supports natural language syntax like:
//   - "job.program is equal to CHIP"
//   - "applicant.age is greater than 18"
//   - "for all members in household where member.income is not null"
//
// The compiler uses ANTLR4 to parse EL expressions and emit postfix notation.
package el

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// Compiler compiles Expression Language (EL) to postfix notation.
type Compiler struct {
	symbols map[string]string // symbol table for type resolution
	errors  []error
}

// NewCompiler creates a new EL compiler.
func NewCompiler() *Compiler {
	return &Compiler{
		symbols: make(map[string]string),
	}
}

// SetSymbols sets the symbol table for type resolution.
// The map keys are identifier names and values are types (entity, long, double, etc.)
func (c *Compiler) SetSymbols(symbols map[string]string) {
	c.symbols = symbols
}

// CompileCondition compiles an EL condition expression to postfix.
// The input should be a boolean expression like "applicant.age >= 18".
func (c *Compiler) CompileCondition(el string) (string, error) {
	return c.compile("condition " + el)
}

// CompileAction compiles an EL action statement to postfix.
// The input should be one or more statements like "set result.eligible = true".
// If the input is just an identifier (table name), it's treated as "perform TableName".
func (c *Compiler) CompileAction(el string) (string, error) {
	el = strings.TrimSpace(el)
	if el == "" {
		return "", nil
	}

	// If it looks like just a table name (single identifier), wrap with perform
	if isIdentifier(el) {
		return c.compile("action perform " + el + ";")
	}

	// Ensure the statement ends with a semicolon
	if !strings.HasSuffix(el, ";") {
		el = el + ";"
	}

	// Try with action prefix
	result, err := c.compile("action " + el)
	if err == nil {
		return result, nil
	}

	// Try without action prefix (might be a raw statement)
	result, err = c.compile(el)
	if err == nil {
		return result, nil
	}

	// Return original error
	return "", err
}

// isIdentifier checks if the string is a simple identifier (table name)
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// Compile compiles a raw EL expression to postfix.
// For conditions, use CompileCondition. For actions, use CompileAction.
func (c *Compiler) Compile(el string) (string, error) {
	return c.compile(el)
}

// compile is the internal compilation method.
func (c *Compiler) compile(el string) (string, error) {
	el = strings.TrimSpace(el)
	if el == "" {
		return "", nil
	}

	// Create lexer and parser
	input := antlr.NewInputStream(el)
	lexer := NewELLexer(input)

	// Set up error listener
	errorListener := &errorCollector{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errorListener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := NewELParser(tokens)

	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)

	// Parse the expression
	tree := parser.Done()

	// Check for parse errors
	if len(errorListener.errors) > 0 {
		return "", fmt.Errorf("parse errors: %s", strings.Join(errorListener.errors, "; "))
	}

	// Emit postfix
	emitter := NewPostfixEmitter()
	emitter.SetSymbols(c.symbols)
	emitter.Visit(tree)

	// Check for emission errors
	if errs := emitter.Errors(); len(errs) > 0 {
		errStrs := make([]string, len(errs))
		for i, err := range errs {
			errStrs[i] = err.Error()
		}
		return "", fmt.Errorf("emission errors: %s", strings.Join(errStrs, "; "))
	}

	return emitter.Emit(), nil
}

// errorCollector collects ANTLR parse errors.
type errorCollector struct {
	*antlr.DefaultErrorListener
	errors []string
}

func (e *errorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{},
	line, column int, msg string, ex antlr.RecognitionException) {
	e.errors = append(e.errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}

// MustCompileCondition compiles a condition and panics on error.
// Use only for testing or when the expression is known to be valid.
func MustCompileCondition(el string) string {
	c := NewCompiler()
	result, err := c.CompileCondition(el)
	if err != nil {
		panic(err)
	}
	return result
}

// MustCompileAction compiles an action and panics on error.
// Use only for testing or when the expression is known to be valid.
func MustCompileAction(el string) string {
	c := NewCompiler()
	result, err := c.CompileAction(el)
	if err != nil {
		panic(err)
	}
	return result
}

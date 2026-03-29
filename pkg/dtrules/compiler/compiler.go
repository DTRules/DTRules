// Copyright 2004-2011 DTRules.com, Inc.
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

// Package compiler implements the postfix expression compiler for DTRules.
package compiler

import (
	"fmt"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// Compiler compiles postfix expressions into executable code.
type Compiler struct {
	session dtrules.Session
	factory *entity.Factory
}

// NewCompiler creates a new compiler.
func NewCompiler(session dtrules.Session, factory *entity.Factory) *Compiler {
	return &Compiler{
		session: session,
		factory: factory,
	}
}

// Compile compiles a postfix expression string into executable code.
// The expression is expected to be in postfix notation (like PostScript).
func (c *Compiler) Compile(expr string) (dtrules.Object, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		// Empty expression - return a no-op array
		return dtrules.NewArray(c.session, true, false)
	}

	// Tokenize the expression
	tokens := c.tokenize(expr)
	if len(tokens) == 0 {
		return dtrules.NewArray(c.session, true, false)
	}

	// Build an executable array from tokens
	elements := make([]dtrules.Object, 0, len(tokens))

	for _, token := range tokens {
		obj, err := c.compileToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to compile token '%s': %w", token, err)
		}
		elements = append(elements, obj)
	}

	// Create an executable array
	return dtrules.NewArrayWithElements(c.session, true, elements, false)
}

// tokenize breaks an expression into tokens.
// This is a simplified tokenizer that handles:
// - Whitespace-separated tokens
// - Quoted strings
// - Braces for code blocks
func (c *Compiler) tokenize(expr string) []string {
	var tokens []string
	var current strings.Builder
	inString := false
	stringDelim := byte(0)
	braceDepth := 0

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		if inString {
			current.WriteByte(ch)
			if ch == stringDelim && (i == 0 || expr[i-1] != '\\') {
				inString = false
				// Only emit token if not inside a brace block
				if braceDepth == 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inString = true
			stringDelim = ch
			current.WriteByte(ch)

		case '{':
			if braceDepth == 0 && current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			braceDepth++
			current.WriteByte(ch)

		case '}':
			current.WriteByte(ch)
			braceDepth--
			if braceDepth == 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

		case ' ', '\t', '\n', '\r':
			if braceDepth > 0 {
				current.WriteByte(ch)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// compileToken compiles a single token into an executable object.
func (c *Compiler) compileToken(token string) (dtrules.Object, error) {
	// Handle empty token
	if token == "" {
		return dtrules.GetRNull(), nil
	}

	// Handle quoted strings
	if len(token) >= 2 && (token[0] == '"' || token[0] == '\'') {
		// Remove quotes and return string
		return dtrules.NewRString(token[1 : len(token)-1]), nil
	}

	// Handle code blocks (braces)
	if len(token) >= 2 && token[0] == '{' && token[len(token)-1] == '}' {
		// Recursively compile the block content
		inner := token[1 : len(token)-1]
		return c.Compile(inner)
	}

	// Handle special literals
	switch strings.ToLower(token) {
	case "true":
		return dtrules.True, nil
	case "false":
		return dtrules.False, nil
	case "null":
		return dtrules.GetRNull(), nil
	}

	// Try as integer
	if i, err := dtrules.GetRIntegerValueFromString(token); err == nil {
		return i, nil
	}

	// Try as double
	if d, err := dtrules.GetRDoubleValueFromString(token); err == nil {
		return d, nil
	}

	// Handle literal (non-executable) names starting with /
	// In PostScript convention: /name is a literal name, name is executable
	if strings.HasPrefix(token, "/") {
		// Strip the / and return a non-executable name
		literalName := token[1:]
		name := dtrules.GetRName(literalName)
		if name == nil {
			return nil, fmt.Errorf("invalid name syntax: %s", literalName)
		}
		return name.GetNonExecutable(), nil
	}

	// Get the name for this token - used for operator, table, and name lookups
	name := dtrules.GetRName(token)
	if name == nil {
		return nil, fmt.Errorf("invalid name syntax: %s", token)
	}

	// Check if it's an operator
	if op, ok := operators.Get(name); ok {
		return op, nil
	}

	// Check if it's a decision table name
	if dt, err := c.factory.GetDecisionTable(name); err == nil && dt != nil {
		return dt, nil
	}

	// Otherwise, treat it as a name (which will be looked up at runtime)
	return name.GetExecutable(), nil
}

// CompilePostfix compiles an already-postfix string.
// This is used when loading pre-compiled decision tables.
func (c *Compiler) CompilePostfix(postfix string) (dtrules.Object, error) {
	return c.Compile(postfix)
}

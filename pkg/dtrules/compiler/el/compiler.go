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
	"errors"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// errEmission marks a failure that occurred after a clean parse, while the
// emitter was producing postfix (e.g. a type error). CompileAction uses it to
// distinguish a definitive emission failure from a parse-structure mismatch:
// the former must not trigger the raw-statement fallback, which would reparse
// and report a confusing parse error that masks the real diagnostic.
var errEmission = errors.New("emission errors")

// Compiler compiles Expression Language (EL) to postfix notation.
type Compiler struct {
	symbols map[string]string // symbol table for type resolution
	emitter *PostfixEmitter   // persistent emitter to track local variables
	errors  []error
}

// NewCompiler creates a new EL compiler.
func NewCompiler() *Compiler {
	emitter := NewPostfixEmitter()
	return &Compiler{
		symbols: make(map[string]string),
		emitter: emitter,
	}
}

// SetSymbols sets the symbol table for type resolution.
// The map keys are identifier names and values are types (entity, long, double, etc.)
func (c *Compiler) SetSymbols(symbols map[string]string) {
	c.symbols = symbols
	c.emitter.SetSymbols(symbols)
}

// SetOperatorChecker supplies the lookup used to reject statement-form calls to
// operators the engine does not implement.
//
// Without it a misspelled or invented operator compiles clean, writes postfix,
// passes build, and fails only when that row executes — reported as "The Name
// 'subests' was not defined by any Entity on the Entity Stack", which does not
// read like a typo (#1020).
//
// Injected rather than imported: pkg/dtrules/operators imports pkg/dtrules, and
// pkg/dtrules's tests import this package, so importing the registry here
// closes that loop. Every path that compiles rules for real should set this —
// pass operators.GetByString.
func (c *Compiler) SetOperatorChecker(exists func(string) bool) {
	c.emitter.operatorExists = exists
}

// SetOperatorArity supplies the lookup used to reject statement-form calls
// with the wrong number of arguments.
//
// Separate from SetOperatorChecker rather than folded into it: the name check
// applies to every operator, while arity is declared only for those reachable
// as `name(a, b, …)`, and a caller that wants one may not have the other. The
// lookup returns ok == false for an operator with no declared arity, which is
// not an error — it means unchecked (#1105).
//
// Injected for the same reason as the name check; pass a closure over
// operators.ArityOf.
func (c *Compiler) SetOperatorArity(arity func(name string) (min, max int, ok bool)) {
	c.emitter.operatorArity = arity
}

// ResetLocals clears per-table local variable state (names + slot indices).
// Callers that reuse a Compiler across multiple tables must call this between
// tables so slot indices don't bleed across independent compilation scopes.
// Within a single table, locals persist across Context/Condition/Action calls
// so a condition can see the slot declared by the context.
func (c *Compiler) ResetLocals() {
	c.emitter.ResetLocals()
}

// CollectionResolver maps an entity-type name (the element type of an array
// field in the EDD) to the fully qualified `owner.field` path of the array
// that contains entities of that type. If more than one collection holds
// entities of the requested type, the resolver must return an error listing
// every candidate so the author can disambiguate.
type CollectionResolver func(entityType string) (ownerEntity, fieldName string, err error)

// SetCollectionResolver wires a resolver that the `for all <type> entities`
// DSL form uses to rewrite a bare type name to its EDD-declared owning
// collection.
func (c *Compiler) SetCollectionResolver(fn CollectionResolver) {
	c.emitter.SetCollectionResolver(fn)
}

// CompileCondition compiles an EL condition expression to postfix.
// The input should be a boolean expression like "applicant.age >= 18".
func (c *Compiler) CompileCondition(el string) (string, error) {
	if isCommentOnly(el) {
		// Documentation rows: a comment-only condition compiles to
		// nothing; the loader substitutes "true always" at load time.
		return "", nil
	}
	return c.compile("condition " + el)
}

// isCommentOnly reports whether the input is empty or consists solely of
// a comment (// line, # line, or a /* block */) — a documentation row
// with no executable content.
func isCommentOnly(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return true
	}
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") {
		return true
	}
	return strings.HasPrefix(t, "/*") && strings.HasSuffix(t, "*/") &&
		!strings.Contains(t[2:len(t)-2], "*/")
}

// CompileContext compiles an EL context statement to postfix.
// Context statements set up local variables and iteration contexts.
// This method is primarily used to register local variables before
// compiling conditions and actions in the same table.
func (c *Compiler) CompileContext(el string) (string, error) {
	el = strings.TrimSpace(el)
	if isCommentOnly(el) {
		// Same rule CompileCondition applies: a row that is only a comment
		// is documentation and compiles to nothing. Without this, a table
		// carrying a commented-out context row could not be written through
		// the authoring API at all — `table put` rejected the whole table on
		// a parse error for a row that was never meant to execute.
		return "", nil
	}
	return c.compile("context " + el)
}

// CompileAction compiles an EL action statement to postfix.
// The input should be one or more statements like "set result.eligible = true".
// If the input is just an identifier (table name), it's treated as "perform TableName".
func (c *Compiler) CompileAction(el string) (string, error) {
	el = strings.TrimSpace(el)
	if isCommentOnly(el) {
		// Documentation row — see CompileCondition.
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

	// A clean parse that failed only at emission (e.g. a type error) is
	// definitive — don't fall back to the raw-statement attempt, which would
	// reparse and report a confusing parse error that hides the real one.
	if errors.Is(err, errEmission) {
		return "", err
	}

	// Try without action prefix (might be a raw statement)
	prefixErr := err
	result, err = c.compile(el)
	if err == nil {
		return result, nil
	}

	// Report the PREFIXED attempt's error. Nearly every action cell is meant
	// to be read as `action <statement>`, so that is the parse the author
	// cares about. Returning the raw attempt's error instead reported
	// "mismatched input '{' expecting {'action', 'condition', ...}" for a
	// statement that merely needed a semicolon inside its block — an error
	// pointing at the entry rule rather than the actual mistake.
	return "", prefixErr
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

	// EOF anchor: the parser may accept a prefix of the input and silently
	// drop trailing tokens. Fail loudly if any remain so authors see broken DSL
	// rather than quietly wrong postfix.
	if tok := tokens.LA(1); tok != antlr.TokenEOF {
		rest := tokens.GetTextFromInterval(antlr.NewInterval(tokens.Index(), tokens.Size()-1))
		return "", fmt.Errorf("unexpected tokens after parse: %s", strings.TrimSpace(rest))
	}

	// Emit postfix using persistent emitter (preserves local variable state)
	c.emitter.Reset() // Reset output buffer but preserve locals
	c.emitter.Visit(tree)

	// Check for emission errors
	if errs := c.emitter.Errors(); len(errs) > 0 {
		errStrs := make([]string, len(errs))
		for i, err := range errs {
			errStrs[i] = err.Error()
		}
		return "", fmt.Errorf("%w: %s", errEmission, strings.Join(errStrs, "; "))
	}

	return c.emitter.Emit(), nil
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

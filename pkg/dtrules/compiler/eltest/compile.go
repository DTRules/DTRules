// Package eltest provides internal helpers for compiling EL expressions and
// capturing the resulting postfix for documentation and testing purposes.
//
// This is NOT an authoring API. Rule authors should use `dtrules docs el`.
// Engineers debugging the compiler or runtime use this to verify compilation.
package eltest

import (
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
)

// Result holds the outcome of a compilation.
type Result struct {
	EL      string
	Postfix string
	Err     error
}

// Compile compiles an EL condition string and returns the postfix.
// symbols maps identifier names to DTRules types (e.g., "applicant.age" -> "long").
func Compile(elStr string, symbols map[string]string) Result {
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	postfix, err := c.CompileCondition(elStr)
	return Result{EL: elStr, Postfix: postfix, Err: err}
}

// CompileAction compiles an EL action string and returns the postfix.
func CompileAction(elStr string, symbols map[string]string) Result {
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	postfix, err := c.CompileAction(elStr)
	return Result{EL: elStr, Postfix: postfix, Err: err}
}

// MustCompile compiles an EL condition string; panics on error.
// Use only for known-valid expressions in tests or doc generation.
func MustCompile(elStr string, symbols map[string]string) string {
	r := Compile(elStr, symbols)
	if r.Err != nil {
		panic("eltest.MustCompile(" + elStr + "): " + r.Err.Error())
	}
	return r.Postfix
}

// MustCompileAction compiles an EL action string; panics on error.
func MustCompileAction(elStr string, symbols map[string]string) string {
	r := CompileAction(elStr, symbols)
	if r.Err != nil {
		panic("eltest.MustCompileAction(" + elStr + "): " + r.Err.Error())
	}
	return r.Postfix
}

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

// Package authoring provides a typed Go SDK for editing DTRules projects —
// decision tables, EDD, and mapping — without touching XML directly.
//
// Every mutation is validated before it is committed; authored EL is
// compiled to postfix at the API boundary so invalid expressions surface
// as named errors before they ever reach a file on disk.
//
// This is the primary authoring interface. Consumers should not edit the
// underlying XML files directly; doing so goes against the #504 policy and
// round-trip semantics enforced by `dtrules build`.
package authoring

import (
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
)

// CheckCondition compiles an EL condition expression to postfix using the
// given symbol table. Returns the postfix on success or a position-annotated
// error on failure. symbols may be nil; the parser works loosely without them.
func CheckCondition(elStr string, symbols map[string]string) (postfix string, err error) {
	elStr = strings.TrimSpace(elStr)
	if elStr == "" {
		return "", nil
	}
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	return c.CompileCondition(elStr)
}

// CheckAction compiles an EL action statement to postfix. See CheckCondition.
func CheckAction(elStr string, symbols map[string]string) (postfix string, err error) {
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	return c.CompileAction(strings.TrimSpace(elStr))
}

// CheckContext compiles an EL context statement to postfix. See CheckCondition.
func CheckContext(elStr string, symbols map[string]string) (postfix string, err error) {
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	return c.CompileContext(strings.TrimSpace(elStr))
}

// tableCompiler compiles every row of one decision table through a single EL
// compiler, so locals a context row declares are in scope for the conditions
// and actions beneath it (#965).
//
// `Compiler.ResetLocals` documents the rule — "within a single table, locals
// persist across Context/Condition/Action calls" — but CheckCondition and
// friends each build a fresh compiler, so a table written like
//
//	context:     local entity ApplyingClient = client
//	condition 1: ApplyingClient == client
//
// compiled the condition with no knowledge of the slot and emitted
// `ApplyingClient` as a bare name. It parsed, it produced plausible postfix,
// and it died at execute with "The Name 'ApplyingClient' was not defined by
// any Entity on the Entity Stack". CHIP's Calculate_Group_Size had been dead
// that way for as long as the sample existed.
//
// Rows must be compiled in table order — contexts first — for the slots to be
// declared before they are referenced. syncToXML already writes them that way.
type tableCompiler struct {
	c *el.Compiler
}

// newTableCompiler starts a fresh local scope for one table.
func newTableCompiler(symbols map[string]string) *tableCompiler {
	c := el.NewCompiler()
	if symbols != nil {
		c.SetSymbols(symbols)
	}
	// Slot indices must not bleed in from whatever was compiled before.
	c.ResetLocals()
	return &tableCompiler{c: c}
}

// compile returns the postfix for one row, or "" when the DSL is empty or does
// not compile. An empty result lets the loader's hand-coded-postfix check flag
// the table rather than silently preserving a stale prior compile.
func (tc *tableCompiler) compile(dsl, kind string) string {
	if strings.TrimSpace(dsl) == "" {
		return ""
	}
	var (
		postfix string
		err     error
	)
	switch kind {
	case "context":
		postfix, err = tc.c.CompileContext(strings.TrimSpace(dsl))
	case "condition":
		postfix, err = tc.c.CompileCondition(strings.TrimSpace(dsl))
	case "action":
		postfix, err = tc.c.CompileAction(strings.TrimSpace(dsl))
	}
	if err != nil {
		return ""
	}
	return postfix
}

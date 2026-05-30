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

package el_test

import (
	"strings"
	"testing"

	el "github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"

	// Ensure operator registration init() functions run.
	_ "github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// Issue #835: four string-comparison visitors were emitting operator
// names that have no runtime registration:
//   - VisitBoolStrNeq          (`<a> != <b>`)        emitted `strneq`
//   - VisitBoolStrIsNot        (`<a> is not <b>`)    emitted `strneq not` (doubly broken — even if strneq existed, the extra `not` would re-invert)
//   - VisitBoolStrEqIc         (case-insensitive ==) emitted `sic==`
//   - VisitBoolStrNeqIc        (case-insensitive !=) emitted `sic== not`
//
// The runtime registers only `s==` (alias `streq`) and `s==i` (alias
// `streqignorecase`). The fix replaces the bad names with composed
// `s== not` / `s==i` / `s==i not` forms.

func issue835Symbols() map[string]string {
	return map[string]string{
		"client.state":  "string",
		"client.region": "string",
	}
}

// TestIssue835_StrNeq_UsesRegisteredOps: `<a> != <b>` on strings must
// emit operators that the runtime actually registers. Pre-fix it
// emitted `strneq`, which would error out at runtime with
// `operator not found`.
func TestIssue835_StrNeq_UsesRegisteredOps(t *testing.T) {
	c := el.NewCompiler()
	c.SetSymbols(issue835Symbols())
	got, err := c.CompileCondition(`client.state != client.region`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "strneq") {
		t.Errorf("emitter still emits unregistered `strneq`: %s", got)
	}
	if !strings.Contains(got, "not") {
		t.Errorf("expected `not` composition for string !=, got: %s", got)
	}
	assertOpsRegistered(t, got)
}

// TestIssue835_StrIsNot_UsesRegisteredOps: `<a> is not <b>`. Pre-fix
// emitted `strneq not` (double inversion bug).
func TestIssue835_StrIsNot_UsesRegisteredOps(t *testing.T) {
	c := el.NewCompiler()
	c.SetSymbols(issue835Symbols())
	got, err := c.CompileCondition(`client.state is not client.region`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "strneq") {
		t.Errorf("emitter still emits unregistered `strneq`: %s", got)
	}
	assertOpsRegistered(t, got)
}

// TestIssue835_StrEqIc_UsesRegisteredOps: `<a> is equal to ignore
// case <b>`. Pre-fix emitted `sic==`.
func TestIssue835_StrEqIc_UsesRegisteredOps(t *testing.T) {
	c := el.NewCompiler()
	c.SetSymbols(issue835Symbols())
	got, err := c.CompileCondition(`client.state is equal to ignore case client.region`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "sic==") {
		t.Errorf("emitter still emits unregistered `sic==`: %s", got)
	}
	if !strings.Contains(got, "s==i") {
		t.Errorf("expected `s==i` for case-insensitive equality, got: %s", got)
	}
	assertOpsRegistered(t, got)
}

// TestIssue835_StrNeqIc_UsesRegisteredOps: `<a> is not equal to
// ignore case <b>`. Pre-fix emitted `sic== not`.
func TestIssue835_StrNeqIc_UsesRegisteredOps(t *testing.T) {
	c := el.NewCompiler()
	c.SetSymbols(issue835Symbols())
	got, err := c.CompileCondition(`client.state is not equal to ignore case client.region`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if strings.Contains(got, "sic==") {
		t.Errorf("emitter still emits unregistered `sic==`: %s", got)
	}
	if !strings.Contains(got, "s==i") || !strings.Contains(got, "not") {
		t.Errorf("expected `s==i not` for case-insensitive inequality, got: %s", got)
	}
	assertOpsRegistered(t, got)
}

// assertOpsRegistered: cheap end-to-end check — every non-quoted,
// non-bracket token in the postfix must either parse as a literal or
// resolve via operators.GetByString. Pre-fix `strneq` / `sic==`
// would have surfaced here as unregistered.
func assertOpsRegistered(t *testing.T, postfix string) {
	t.Helper()
	tokens := strings.Fields(postfix)
	for _, tok := range tokens {
		// Skip blocks and string/numeric literals.
		if tok == "{" || tok == "}" {
			continue
		}
		if strings.HasPrefix(tok, "\"") || strings.HasPrefix(tok, "/") || strings.HasPrefix(tok, "$") {
			continue
		}
		if isNumericToken(tok) {
			continue
		}
		// Field references (a.b.c) and entity names are valid even
		// without operator registration — they resolve via the
		// dictionary at runtime, not the operator table.
		if strings.Contains(tok, ".") {
			continue
		}
		// Anything that looks like a multi-segment identifier
		// (uppercase first letter, e.g. an entity name like Client)
		// is also dictionary-resolved.
		if len(tok) > 0 && tok[0] >= 'A' && tok[0] <= 'Z' {
			continue
		}
		// Everything else should be a registered operator.
		if _, ok := operators.GetByString(tok); !ok {
			t.Errorf("unregistered operator token %q in postfix %q", tok, postfix)
		}
	}
}

func isNumericToken(tok string) bool {
	if tok == "" {
		return false
	}
	for i, r := range tok {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if (r >= '0' && r <= '9') || r == '.' || r == 'e' || r == 'E' {
			continue
		}
		return false
	}
	return true
}

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

package docs_test

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

const elRefEDDDir = "testdata"

var (
	elExampleLine = regexp.MustCompile("^\\*\\*Example \\(EL\\)\\*\\*:\\s*`(.+)`\\s*$")
	elPostfixLine = regexp.MustCompile("^\\*\\*Compiled postfix\\*\\*:\\s*`(.+)`\\s*$")
	// The page also uses a one-line form for the domain-flavored examples:
	//   **Tax example**: `EL` → `postfix`
	// Only those two labels carry postfix. A plain `**Example**:` line in the
	// hashing and encoding sections shows a computed VALUE, and checking it
	// as postfix would demand the page document the wrong thing.
	elInlineLine = regexp.MustCompile("^\\*\\*(?:Tax|Eligibility) example\\*\\*[^:]*:\\s*`([^`]+)`\\s*→\\s*`([^`]+)`\\s*$")
)

// probeTargets let a bare expression be compiled: an expression like
// `copy of household.members` is not a cell on its own, so it is wrapped in an
// assignment and the store is stripped back off. One field per type, so the
// wrapper never adds a conversion the expression would not have had.
var probeTargets = []string{"probe.i", "probe.d", "probe.s", "probe.b", "probe.dt", "probe.e", "probe.a"}

// TestELReference_PostfixMatchesCompiler is what keeps el-reference.md honest.
//
// Before it existed, 70 of the page's 107 documented postfix strings were
// wrong: some were stale, and most were copied from the DTEligibility sample
// — a project added by accident inside a documentation commit, whose postfix
// was hand-written in a calling convention this compiler never used, and
// which was removed in #959. Anyone reading a trace against that page was
// looking for tokens the compiler does not emit.
//
// Every **Example (EL)** is compiled here and its **Compiled postfix** line
// must be exactly what came out.
// notImplementedMarker labels an entry whose syntax the grammar accepts but
// the compiler lowers to a runtime-error stub. Such entries are documented —
// the grammar really does have them — but a rule author must be able to tell
// them from working syntax at a glance, which `map ... through` claiming a
// `mapthrough` postfix it never emitted did not allow (#1021).
const notImplementedMarker = "**Status**: NOT IMPLEMENTED"

func TestELReference_PostfixMatchesCompiler(t *testing.T) {
	symbols := authoring.LoadEDDSymbols(elRefEDDDir)
	if len(symbols) == 0 {
		t.Fatalf("%s declares no symbols — the reference's fixture EDD is missing", elRefEDDDir)
	}

	f, err := os.Open(elRefFile)
	if err != nil {
		t.Fatalf("open %s: %v", elRefFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var pendingEL string
	var pendingLine, lineNo, checked, unlabelled int
	var entryMarkedUnimplemented bool
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// A #### heading starts a new entry, so the NOT IMPLEMENTED label
		// cannot leak from the entry above it.
		if strings.HasPrefix(line, "####") {
			entryMarkedUnimplemented = false
		}
		if strings.HasPrefix(line, notImplementedMarker) {
			entryMarkedUnimplemented = true
		}
		if m := elExampleLine.FindStringSubmatch(line); m != nil {
			pendingEL, pendingLine = m[1], lineNo
			continue
		}
		if m := elPostfixLine.FindStringSubmatch(line); m != nil && pendingEL != "" {
			checked++
			got, err := compileELExample(pendingEL, symbols)
			if err != nil {
				t.Errorf("%s:%d: EL does not compile: %s\n  %v", elRefFile, pendingLine, pendingEL, err)
			} else if strings.Contains(got, "elstmterror") && !entryMarkedUnimplemented {
				unlabelled++
				t.Errorf("%s:%d: documented syntax compiles to a runtime-error stub but is not labelled\n"+
					"  EL:     %s\n  emits:  %s\n"+
					"Add a %q line to the entry, or remove the entry. Documenting dead syntax as usable\n"+
					"is how `map ... through` sat in this reference claiming a `mapthrough` postfix it\n"+
					"never emitted (#1021).", elRefFile, pendingLine, pendingEL, got, notImplementedMarker)
			} else if got != normalizePostfix(m[1]) {
				t.Errorf("%s:%d: documented postfix is not what the compiler emits\n  EL:       %s\n  document: %s\n  compiler: %s",
					elRefFile, pendingLine, pendingEL, normalizePostfix(m[1]), got)
			}
			pendingEL = ""
			continue
		}
		if m := elInlineLine.FindStringSubmatch(line); m != nil {
			checked++
			got, err := compileELExample(m[1], symbols)
			if err != nil {
				t.Errorf("%s:%d: EL does not compile: %s\n  %v", elRefFile, lineNo, m[1], err)
			} else if got != normalizePostfix(m[2]) {
				t.Errorf("%s:%d: documented postfix is not what the compiler emits\n  EL:       %s\n  document: %s\n  compiler: %s",
					elRefFile, lineNo, m[1], normalizePostfix(m[2]), got)
			}
			continue
		}
		// A blank line ends an example block, so a stray postfix line further
		// down is never paired with an unrelated EL example.
		if strings.TrimSpace(line) == "" {
			pendingEL = ""
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", elRefFile, err)
	}

	// The page carries 116 pairs today. The floor is a format guard, not a
	// content count: if the example markup changes shape the extractor stops
	// matching and silently checks nothing, which is the one way this test
	// could pass while the page rots.
	_ = unlabelled
	if checked < 110 {
		t.Errorf("only %d EL/postfix pairs checked in %s — the extractor has drifted from the page's example format",
			checked, elRefFile)
	}
}

// compileELExample compiles one example, trying each cell kind and then the
// bare-expression wrapper.
func compileELExample(el string, symbols map[string]string) (string, error) {
	if got, err := authoring.CheckCondition(el, symbols); err == nil {
		return normalizePostfix(got), nil
	}
	if got, err := authoring.CheckAction(el, symbols); err == nil {
		return normalizePostfix(got), nil
	}
	if got, err := authoring.CheckContext(el, symbols); err == nil {
		return normalizePostfix(got), nil
	}
	for _, target := range probeTargets {
		got, err := authoring.CheckAction(fmt.Sprintf("set %s = %s", target, el), symbols)
		if err != nil {
			continue
		}
		out := normalizePostfix(got)
		if i := strings.LastIndex(out, " /"+target+" xdef"); i > 0 {
			out = out[:i]
			// Drop the conversion the assignment inserted, if any.
			for _, cv := range []string{" cvi", " cvd", " cvs", " cvb", " cvdate", " cve"} {
				if strings.HasSuffix(out, cv) {
					out = strings.TrimSuffix(out, cv)
					break
				}
			}
			return out, nil
		}
	}
	_, err := authoring.CheckCondition(el, symbols)
	return "", err
}

func normalizePostfix(s string) string { return strings.Join(strings.Fields(s), " ") }

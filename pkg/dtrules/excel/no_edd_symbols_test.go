// Copyright 2026 Paul Snow
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

package excel

import (
	"strings"
	"testing"
)

// stubCompiler stands in for the EL compiler. The guard under test runs before
// any compilation, so the returned postfix does not matter — what matters is
// whether we get here at all.
type stubCompiler struct{ symbols map[string]string }

func (c *stubCompiler) SetSymbols(s map[string]string)          { c.symbols = s }
func (c *stubCompiler) CompileCondition(string) (string, error) { return "x", nil }
func (c *stubCompiler) CompileAction(string) (string, error)    { return "x", nil }
func (c *stubCompiler) CompileContext(string) (string, error)   { return "x", nil }

// TestNoEDDSymbolsIsRefused pins #1029.
//
// With no symbol table every field reference compiles as an integer: a `fixed`
// subtraction emits `-` instead of `fp-` and the assignment stores `cvi`
// instead of `cvfp`. A money calculation silently changes its arithmetic while
// the build reports "no drops" and exits 0.
//
// The Accumulate staking rules hit this: their EDD workbook was deleted in
// May 2026 as a supposed duplicate of the decision-table workbook — it was
// not, it was the only Excel copy of the EDD — so an Excel-authored rebuild
// imported 35 tables with entities=0 and downgraded every fixed-point
// operator. It stayed invisible because the only path that regenerates
// postfix was itself a no-op (#1010).
//
// Compiling untyped is not a degraded mode, it is a wrong one, so this is an
// error rather than a warning.
func TestNoEDDSymbolsIsRefused(t *testing.T) {
	imp := NewDTImporter()
	imp.SetELCompiler(&stubCompiler{})
	stats := &ImportStats{}
	imp.SetStats(stats)

	err := imp.compileTableEL(tableWithDSL())
	if err == nil {
		t.Fatal("compiled a table with DSL and no EDD symbols — every field would be typed as integer")
	}
	if !strings.Contains(err.Error(), "no EDD symbols") {
		t.Errorf("error should name the cause, got: %v", err)
	}
	if len(stats.Drops) != 1 {
		t.Fatalf("want 1 drop, got %d: %+v", len(stats.Drops), stats.Drops)
	}
	// The drop has to be actionable: it must say what goes wrong, not just
	// that something did.
	reason := stats.Drops[0].Reason
	for _, want := range []string{"4 DSL row(s)", "cvfp into cvi", "sync export"} {
		if !strings.Contains(reason, want) {
			t.Errorf("drop reason missing %q; got: %s", want, reason)
		}
	}
}

// TestSymbolsPresentCompilesNormally is the control: the guard must not fire
// on the ordinary path, which is every project that has an EDD.
func TestSymbolsPresentCompilesNormally(t *testing.T) {
	imp := NewDTImporter()
	imp.SetELCompiler(&stubCompiler{})
	imp.SetSymbols(map[string]string{"client.age": "integer"})
	stats := &ImportStats{}
	imp.SetStats(stats)

	if err := imp.compileTableEL(tableWithDSL()); err != nil {
		t.Fatalf("refused a table that has symbols: %v", err)
	}
	if len(stats.Drops) != 0 {
		t.Errorf("unexpected drops on the normal path: %+v", stats.Drops)
	}
}

// TestNoSymbolsNoDSLIsFine keeps the guard narrow: a table with nothing to
// compile has nothing to mistype.
func TestNoSymbolsNoDSLIsFine(t *testing.T) {
	imp := NewDTImporter()
	imp.SetELCompiler(&stubCompiler{})
	stats := &ImportStats{}
	imp.SetStats(stats)

	if err := imp.compileTableEL(&DecisionTableXML{TableName: "Empty"}); err != nil {
		t.Errorf("refused a table with no DSL: %v", err)
	}
	if len(stats.Drops) != 0 {
		t.Errorf("unexpected drops for a table with no DSL: %+v", stats.Drops)
	}
}

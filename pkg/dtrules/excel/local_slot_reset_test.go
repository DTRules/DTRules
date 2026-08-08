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
	"fmt"
	"testing"
)

// countingCompiler records how many times ResetLocals was called and emits a
// slot index that climbs unless it is reset — the same shape as the real
// compiler's local counter.
type countingCompiler struct {
	resets int
	slot   int
}

func (c *countingCompiler) SetSymbols(map[string]string) {}
func (c *countingCompiler) ResetLocals()                 { c.resets++; c.slot = 0 }
func (c *countingCompiler) next() string {
	s := fmt.Sprintf("%d local@", c.slot)
	c.slot++
	return s
}
func (c *countingCompiler) CompileCondition(string) (string, error) { return c.next(), nil }
func (c *countingCompiler) CompileAction(string) (string, error)    { return c.next(), nil }
func (c *countingCompiler) CompileContext(string) (string, error)   { return c.next(), nil }

// TestLocalsResetBetweenTables pins #1047.
//
// Local slot indices are numbered per table, and this importer reuses one
// compiler for every table in a workbook. Without a reset the counter keeps
// climbing, so the second table's `local@` indices come out one higher than
// the frame the runtime allocates, the third two higher, and so on.
//
// The rules still build — the postfix is well formed and `drops: none` — and
// fail at execution with `[OutOfBounds] GetFrameValue`. On CHIP every index in
// Evaluate_CHIP_Eligibility shifted by one and five tests that pass against the
// committed XML failed against the rebuild.
//
// Compiler.ResetLocals documents the rule and the authoring path already obeyed
// it. This path never did, which is why it surfaced only when a project was
// rebuilt from Excel rather than authored through the API.
func TestLocalsResetBetweenTables(t *testing.T) {
	c := &countingCompiler{}
	imp := NewDTImporter()
	imp.SetELCompiler(c)
	imp.SetSymbols(map[string]string{"x": "integer"})

	first := tableWithDSL()
	first.TableName = "First"
	second := tableWithDSL()
	second.TableName = "Second"

	if err := imp.compileTableEL(first); err != nil {
		t.Fatalf("first table: %v", err)
	}
	if err := imp.compileTableEL(second); err != nil {
		t.Fatalf("second table: %v", err)
	}

	if c.resets < 2 {
		t.Errorf("ResetLocals called %d time(s) for 2 tables — slot indices carry over between them", c.resets)
	}

	// Two structurally identical tables must compile to identical slot
	// numbering. Asserting "the second starts at 0" would be wrong — within a
	// table, contexts and initial actions take slots before conditions do, so
	// the condition legitimately lands on a later slot. What must not happen
	// is the second table continuing where the first left off.
	if a, b := first.Conditions[0].Postfix, second.Conditions[0].Postfix; a != b {
		t.Errorf("identical tables compiled to different slots:\n  first:  %q\n  second: %q", a, b)
	}
	if a, b := first.Actions[0].Postfix, second.Actions[0].Postfix; a != b {
		t.Errorf("identical tables compiled to different action slots:\n  first:  %q\n  second: %q", a, b)
	}
}

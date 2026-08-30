package el

import (
	"strings"
	"testing"
)

// Local slot indices are frame-relative, and a condition or action that
// declares a local emits its own `allocate` — reserving one slot at the same
// offset every sibling row starts from. So the numbering has to restart from
// whatever the table's contexts declared, not keep climbing across rows.
//
// Without the rewind the second action to declare a local emits `1 local@`
// against a frame holding a single slot, and execution dies with
// "[OutOfBounds] GetFrameValue". That is #1047 one scope down: #1047 stopped
// the counter bleeding between tables, this stops it bleeding between the rows
// of one table. It stayed hidden because it needs two rows in the same table
// that each declare a local, and TaxReturn's
// Build_State_Tax_Result_For_Period — the first such table to be executed —
// was orphaned the whole time it had the defect (#234).
func TestSiblingActionsReuseTheSameLocalSlot(t *testing.T) {
	c := NewCompiler()
	c.ResetLocals()
	c.MarkLocalScope()

	const row = `create state_tax_result as st_result;` +
		`add st_result to job.state_tax_results;`

	c.ResetToLocalScope()
	first, err := c.CompileAction(row)
	if err != nil {
		t.Fatalf("first action: %v", err)
	}

	c.ResetToLocalScope()
	second, err := c.CompileAction(row)
	if err != nil {
		t.Fatalf("second action: %v", err)
	}

	if !strings.Contains(first, "0 local@") {
		t.Errorf("first action should use slot 0, got: %s", first)
	}
	if !strings.Contains(second, "0 local@") {
		t.Errorf("the second action allocates its own frame, so it must reuse slot 0; got: %s", second)
	}
	if strings.Contains(second, "1 local@") {
		t.Errorf("second action indexes past the one slot its allocate reserves: %s", second)
	}
	if first != second {
		t.Errorf("identical rows should compile identically:\n first=%s\nsecond=%s", first, second)
	}
}

// A local a context declares wraps the whole table, so rows below it must keep
// seeing that slot — and their own locals must be numbered after it, not on
// top of it.
func TestContextLocalsSurviveTheRewind(t *testing.T) {
	c := NewCompiler()
	c.ResetLocals()

	if _, err := c.CompileContext(`for all taxpayers as tp`); err != nil {
		t.Fatalf("context: %v", err)
	}
	c.MarkLocalScope()

	c.ResetToLocalScope()
	act, err := c.CompileAction(`create state_tax_result as st_result;` +
		`add st_result to job.state_tax_results;`)
	if err != nil {
		t.Fatalf("action: %v", err)
	}
	if !strings.Contains(act, "1 local@") {
		t.Errorf("the context holds slot 0, so an action local must take slot 1; got: %s", act)
	}

	// A second row must land on the same slot as the first, not slot 2.
	c.ResetToLocalScope()
	act2, err := c.CompileAction(`create state_tax_result as st_result;` +
		`add st_result to job.state_tax_results;`)
	if err != nil {
		t.Fatalf("second action: %v", err)
	}
	if act != act2 {
		t.Errorf("sibling rows diverged:\n first=%s\nsecond=%s", act, act2)
	}
}

// ResetLocals starts a new table, so a baseline marked for the previous one
// must not survive into it.
func TestResetLocalsDropsTheScope(t *testing.T) {
	c := NewCompiler()
	c.ResetLocals()
	if _, err := c.CompileContext(`for all taxpayers as tp`); err != nil {
		t.Fatalf("context: %v", err)
	}
	c.MarkLocalScope()

	c.ResetLocals()
	c.ResetToLocalScope() // no baseline for this table: must be a no-op

	act, err := c.CompileAction(`create state_tax_result as st_result;` +
		`add st_result to job.state_tax_results;`)
	if err != nil {
		t.Fatalf("action: %v", err)
	}
	if !strings.Contains(act, "0 local@") {
		t.Errorf("a fresh table starts at slot 0, got: %s", act)
	}
}

package el

import (
	"strings"
	"testing"
)

// The runtime's `if` and `ifelse` pop the TEST from the top of the data
// stack — the same convention the lazy and/or emission (`over if`) and the
// hasrelationship-where form rely on. If-statements therefore must emit
// their bodies first and the bexpr last. These pins guard against a
// regression to the old `bexpr { body } if` order, which made every
// compiled if-statement fail at runtime with a BooleanValue conversion
// error on the block.
func TestIfStatementEmitsTestOnTop(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.a": "double", "job.b": "double",
	})

	pf, err := c.CompileAction("if job.a > 1.0 then { set job.b = 2.0; } endif")
	if err != nil {
		t.Fatalf("compile if-then: %v", err)
	}
	toks := strings.Fields(pf)
	if len(toks) < 2 || toks[len(toks)-1] != "ifelse" && toks[len(toks)-1] != "if" {
		t.Fatalf("expected trailing if/ifelse, got %q", pf)
	}
	// The test expression must directly precede the if/ifelse operator.
	tail := strings.Join(toks[len(toks)-5:], " ")
	if !strings.Contains(tail, "job.a 1.0 f>") {
		t.Fatalf("test expression not on top of stack before if/ifelse: %q", pf)
	}
	if strings.HasPrefix(pf, "job.a") {
		t.Fatalf("test expression emitted before bodies (old broken order): %q", pf)
	}
}

func TestIfThenElseEmitsTestOnTop(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"job.a": "double", "job.b": "double",
	})

	pf, err := c.CompileAction("if job.a > 1.0 then { set job.b = 2.0; } else { set job.b = 3.0; } endif")
	if err != nil {
		t.Fatalf("compile if-then-else: %v", err)
	}
	toks := strings.Fields(pf)
	if toks[len(toks)-1] != "ifelse" {
		t.Fatalf("expected trailing ifelse, got %q", pf)
	}
	tail := strings.Join(toks[len(toks)-5:], " ")
	if !strings.Contains(tail, "job.a 1.0 f> ifelse") {
		t.Fatalf("test expression must directly precede ifelse: %q", pf)
	}
}

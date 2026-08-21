package el

import (
	"strings"
	"testing"
)

func TestPerformWithDefaultLowersToFallback(t *testing.T) {
	out, err := NewCompiler().CompileAction(
		`perform table named ("Determine_" + apportionment.state_code + "_Filing_Requirement") with default Determine_No_Filing_Requirement`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, want := range []string{"strconcat", "/Determine_No_Filing_Requirement", "performtableordefault"} {
		if !strings.Contains(out, want) {
			t.Errorf("postfix missing %q: %s", want, out)
		}
	}
}

func TestPlainDynamicPerformStillCompiles(t *testing.T) {
	out, err := NewCompiler().CompileAction(`perform table named ("A" + job.x + "B")`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(out, "performtable") || strings.Contains(out, "performtableordefault") {
		t.Errorf("the form without a default changed shape: %s", out)
	}
}

// `among <list>` — the author's complete set of legitimate targets, stated at
// the call site with no smartness required (#776).
func TestPerformAmongLowersToBoundedDispatch(t *testing.T) {
	out, err := NewCompiler().CompileAction(
		`perform table named ("Handle_" + job.kind) among Handle_AA, Handle_BB with default Handle_Default`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, want := range []string{"/Handle_AA", "/Handle_BB", "2", "/Handle_Default", "performtableamongdefault"} {
		if !strings.Contains(out, want) {
			t.Errorf("postfix missing %q: %s", want, out)
		}
	}
}

func TestPerformAmongWithoutDefault(t *testing.T) {
	out, err := NewCompiler().CompileAction(
		`perform table named ("Handle_" + job.kind) among Handle_AA, Handle_BB`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(out, "performtableamong") || strings.Contains(out, "amongdefault") {
		t.Errorf("among alone must use the erroring form: %s", out)
	}
}

// The two earlier forms must be untouched.
func TestPlainAndDefaultFormsUnchangedByAmong(t *testing.T) {
	out, _ := NewCompiler().CompileAction(`perform table named ("A" + job.x)`)
	if !strings.HasSuffix(strings.TrimSpace(out), "performtable") {
		t.Errorf("plain form changed: %s", out)
	}
	out2, _ := NewCompiler().CompileAction(`perform table named ("A" + job.x) with default D`)
	if !strings.Contains(out2, "performtableordefault") || strings.Contains(out2, "among") {
		t.Errorf("default-only form changed: %s", out2)
	}
}

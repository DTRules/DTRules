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

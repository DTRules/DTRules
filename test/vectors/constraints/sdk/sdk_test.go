// Vectors for the Go SDK write paths of #1209. Written BEFORE the fix; not the implementer's to edit.
// A host that embeds DTRules writes fields with Project.SetAttribute and DebugSession.SetAttribute. Those are
// writes from OUTSIDE the rules, so a declared constraint must refuse them exactly as --data and --input do.
// Run through run_sdk.py, which prepares the constrained project and sets DTR_VECTOR_PROJECT.
package sdk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func open(t *testing.T) *authoring.Project {
	t.Helper()
	dir := os.Getenv("DTR_VECTOR_PROJECT")
	if dir == "" {
		t.Skip("DTR_VECTOR_PROJECT not set; run test/vectors/constraints/run_sdk.py")
	}
	p, err := authoring.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	return p
}

func healthy(t *testing.T, p *authoring.Project) {
	t.Helper()
	for k, v := range map[string]any{"age": 40, "lean_body_weight": 80.0, "pcr": 0.9, "penicillin_allergic": false} {
		if err := p.SetAttribute("patient", k, v); err != nil {
			t.Fatalf("S3 unconstrained field %s must be writable: %v", k, err)
		}
	}
}

func refused(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: a value outside allowed_values was accepted", name)
	}
	for _, want := range []string{"patient.diagnosis", "Banana", "Chronic Sinusitis"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error must name %q; got: %v", name, want, err)
		}
	}
}

func TestS1_SetAttribute_InSet_CaseInsensitive(t *testing.T) {
	p := open(t)
	if err := p.SetAttribute("patient", "diagnosis", "chronic sinusitis"); err != nil {
		t.Fatalf("a value in the set (any case) must be accepted: %v", err)
	}
}

func TestS2_SetAttribute_OutsideSet_Refused(t *testing.T) {
	p := open(t)
	refused(t, "Project.SetAttribute", p.SetAttribute("patient", "diagnosis", "Banana"))
}

func TestS3_SetAttribute_Unconstrained_Untouched(t *testing.T) { healthy(t, open(t)) }

func TestS4_Refused_Write_Leaves_The_Old_Value(t *testing.T) {
	p := open(t)
	healthy(t, p)
	if err := p.SetAttribute("patient", "diagnosis", "Acute Sinusitis"); err != nil {
		t.Fatal(err)
	}
	_ = p.SetAttribute("patient", "diagnosis", "Banana")
	ok, err := p.EvalCondition(`patient.diagnosis == "Acute Sinusitis"`)
	if err != nil || !ok {
		t.Fatalf("after a refused write the field must still hold its previous value (ok=%v err=%v)", ok, err)
	}
}

func TestS5_DebugSession_SetAttribute_OutsideSet_Refused(t *testing.T) {
	p := open(t)
	if err := p.LoadTestData(filepath.Join(os.Getenv("DTR_VECTOR_PROJECT"), "testfiles", "TestScenarios", "AdultStandard", "input.xml")); err != nil {
		t.Fatalf("LoadTestData: %v", err)
	}
	trace, err := p.ExecuteEntry("Determine_Therapy")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}
	dbg, err := p.ResumeAt(trace, 0)
	if err != nil {
		t.Fatalf("ResumeAt: %v", err)
	}
	defer dbg.Close()
	if err := dbg.SetAttribute("patient", "diagnosis", "Chronic Sinusitis"); err != nil {
		t.Fatalf("DebugSession: a value in the set must be accepted: %v", err)
	}
	refused(t, "DebugSession.SetAttribute", dbg.SetAttribute("patient", "diagnosis", "Banana"))
}

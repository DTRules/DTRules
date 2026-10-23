// Copyright 2026 DTRules.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dtrules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestValidateStateTaxWritesFail proves the expected_state_tax check (#1317)
// is wired in. The coverage ratchet only counts clean scenarios, so if
// Validate_State_Tax vanished (a merge that regenerated the XML from a
// workbook without it, say) nothing would fail; the clean count would rise.
// This runs AR_01 three ways: as written (PASS line), with a wrong figure
// (Validate_State_Tax's FAIL line), and with a roster that has no entry for
// job.state (the FAIL line Validate_Results writes).
func TestValidateStateTaxWritesFail(t *testing.T) {
	rs, xmlDir := loadTaxReturn(t)
	path := filepath.Join(xmlDir, "..", "testfiles", "TestScenarios", "AR", "TestCase_AR_01_Single_W2.xml")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const right = "<expected_state_tax>1483</expected_state_tax>"
	if strings.Count(string(src), right) != 1 {
		t.Fatalf("AR_01 should carry %s once", right)
	}

	t.Run("right figure passes", func(t *testing.T) {
		lines := auditTrailLines(runScenarioForStateTaxCheck(t, rs, xmlDir, src))
		if !anyLineContains(lines, "PASS: State Tax (AR)") {
			t.Errorf("no PASS line for AR; audit trail:\n%s", strings.Join(lines, "\n"))
		}
		if anyLineContains(lines, "FAIL:") {
			t.Errorf("AR_01 as written should not FAIL; audit trail:\n%s", strings.Join(lines, "\n"))
		}
	})

	t.Run("wrong figure fails", func(t *testing.T) {
		wrong := strings.Replace(string(src), right, "<expected_state_tax>9999</expected_state_tax>", 1)
		lines := auditTrailLines(runScenarioForStateTaxCheck(t, rs, xmlDir, []byte(wrong)))
		if !anyLineContains(lines, "FAIL: State tax mismatch (AR) - calculated $") || !anyLineContains(lines, "vs expected $9999") {
			t.Errorf("a wrong expected_state_tax should write Validate_State_Tax's FAIL line; audit trail:\n%s", strings.Join(lines, "\n"))
		}
	})

	t.Run("no roster entry fails", func(t *testing.T) {
		// A state_period builds the roster instead of job.state, so a
		// Kentucky-only period leaves no AR entry to check against.
		const taxpayers = "<taxpayers>"
		if strings.Count(string(src), taxpayers) != 1 {
			t.Fatalf("AR_01 should contain %s once", taxpayers)
		}
		periods := `<state_periods>
      <state_period id="1">
         <id>1</id>
         <state_code>KY</state_code>
         <start_date>1/1/2025</start_date>
         <end_date>12/31/2025</end_date>
         <resident_status>full_year</resident_status>
      </state_period>
   </state_periods>
   ` + taxpayers
		noEntry := strings.Replace(string(src), taxpayers, periods, 1)
		lines := auditTrailLines(runScenarioForStateTaxCheck(t, rs, xmlDir, []byte(noEntry)))
		if !anyLineContains(lines, "FAIL: State tax mismatch (AR) - no state_tax_results entry for the resident state") {
			t.Errorf("a roster without the resident state should write Validate_Results' FAIL line; audit trail:\n%s", strings.Join(lines, "\n"))
		}
	})
}

// runScenarioForStateTaxCheck loads one scenario through the TaxReturn mapping and
// runs Compute_Tax_Return on it.
func runScenarioForStateTaxCheck(t *testing.T, rs *session.RuleSet, xmlDir string, src []byte) dtrules.State {
	t.Helper()
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	mp := mapping.NewMapping(sess)
	mf, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer mf.Close()
	if err := mp.LoadMapping(mf); err != nil {
		t.Fatal(err)
	}
	if err := mp.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := mp.LoadData(strings.NewReader(string(src))); err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Compute_Tax_Return: %v", err)
	}
	if err := dt.Execute(state); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return state
}

// auditTrailLines returns job.audit_trail, trimmed.
func auditTrailLines(state dtrules.State) []string {
	var lines []string
	for i := 0; i < state.EntityDepth(); i++ {
		e, _ := state.EntityFetch(i)
		if e == nil || e.GetName().StringValue() != "job" {
			continue
		}
		v, _ := e.Get(dtrules.GetRName("audit_trail"))
		if v == nil {
			continue
		}
		arr, err := v.ArrayValue()
		if err != nil {
			continue
		}
		for _, line := range arr {
			lines = append(lines, strings.TrimSpace(line.StringValue()))
		}
	}
	return lines
}

func anyLineContains(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.Contains(l, prefix) {
			return true
		}
	}
	return false
}

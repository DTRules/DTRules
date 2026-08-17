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

// TaxReturn ships 505 scenarios and its tests name 25 of them. The other 480
// are data nothing reads, and a scenario nothing runs can hold any figures at
// all: TestCase_Credits_03_MFS_Standard asserted a $15,000 standard deduction
// and the $45,000 taxable income that follows from it, both wrong, and nothing
// noticed because nothing executed it (#1140).
//
// Running all 505 as ordinary tests would fail 312 of them today, which is a
// wall rather than a signal. So this is a ratchet: it runs every scenario and
// holds the line on how many come out clean. The floor can only be raised.
//
// A scenario is "clean" when it loads, executes, and the rules' own
// Validate_Summary writes no FAIL line — the rule set validates itself, and
// this listens to it.
const (
	scenariosCleanFloor = 175
	scenariosRunFloor   = 500
)

func TestTaxReturnScenarioCoverage(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")
	if _, err := os.Stat(xmlDir); err != nil {
		t.Skip("TaxReturn sample not present")
	}

	rs := session.NewRuleSet("TaxReturn")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load: %v", err)
	}

	var total, ran, clean int
	root := filepath.Join(sampleDir, "testfiles", "TestScenarios")
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".xml") {
			return nil
		}
		total++
		if runScenarioCleanly(t, rs, xmlDir, p, &ran) {
			clean++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	t.Logf("scenarios=%d executed=%d clean=%d", total, ran, clean)

	if ran < scenariosRunFloor {
		t.Errorf("only %d of %d scenarios load and execute, floor is %d — "+
			"a scenario that stopped loading is a regression",
			ran, total, scenariosRunFloor)
	}
	if clean < scenariosCleanFloor {
		t.Errorf("%d scenarios validate cleanly, floor is %d — "+
			"a scenario whose figures stopped agreeing with the rules is a regression. "+
			"If the rules changed on purpose, the scenario's expected values change with them",
			clean, scenariosCleanFloor)
	}
	if clean > scenariosCleanFloor {
		t.Logf("clean count %d is above the floor of %d — raise scenariosCleanFloor to hold the gain",
			clean, scenariosCleanFloor)
	}
}

// runScenarioCleanly executes one scenario and reports whether the rules'
// self-validation stayed quiet.
func runScenarioCleanly(t *testing.T, rs *session.RuleSet, xmlDir, path string, ran *int) bool {
	t.Helper()
	sess, err := rs.NewSession()
	if err != nil {
		return false
	}
	m := mapping.NewMapping(sess)
	mf, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	if err != nil {
		return false
	}
	defer mf.Close()
	if m.LoadMapping(mf) != nil || m.Initialize() != nil {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	if m.LoadData(f) != nil {
		return false
	}
	*ran++

	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		return false
	}
	if dt.Execute(state) != nil {
		return false
	}
	return !auditHasFailure(state)
}

// auditHasFailure reports whether the rules wrote a FAIL line about their own
// figures. Validate_Summary compares computed values against the scenario's
// expectations and says so; nothing used to listen (#1000).
func auditHasFailure(state dtrules.State) bool {
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
			if strings.HasPrefix(strings.TrimSpace(line.StringValue()), "FAIL:") {
				return true
			}
		}
	}
	return false
}

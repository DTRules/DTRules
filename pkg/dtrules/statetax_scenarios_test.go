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

package dtrules_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// expectedTaxOwed pulls the answer out of a scenario file's header comment:
//
//	Expected: taxOwed = $1,425
//
// Every StateTax scenario carries one. Keeping the expectation in the file it
// describes is what makes these cases self-documenting; this test just stops
// them from being decorative.
var expectedTaxOwed = regexp.MustCompile(`Expected:\s*taxOwed\s*=\s*\$?([\d,]+)`)

// TestStateTaxScenarios runs all 51 state scenarios end to end and checks the
// computed tax against the expectation each file states.
//
// Before this existed, the only StateTax test loaded the rules and logged
// whatever happened during execution, so the project could compute nothing at
// all and still look green — which is exactly what it did: `filing_status ==
// SINGLE` compiled to a name lookup that failed at runtime, and every table
// downstream of it never ran.
func TestStateTaxScenarios(t *testing.T) {
	dir := findStateTaxDir(t)
	if dir == "" {
		t.Skip("StateTax sample project not found")
	}

	scenarioDir := filepath.Join(dir, "testfiles/TestScenarios")
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		t.Fatalf("read scenarios: %v", err)
	}

	var scenarios []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".xml") {
			scenarios = append(scenarios, e.Name())
		}
	}
	if len(scenarios) == 0 {
		t.Fatal("no scenario files found")
	}

	for _, name := range scenarios {
		t.Run(strings.TrimSuffix(name, ".xml"), func(t *testing.T) {
			path := filepath.Join(scenarioDir, name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			m := expectedTaxOwed.FindSubmatch(raw)
			if m == nil {
				t.Fatalf("%s has no `Expected: taxOwed = $N` comment", name)
			}
			want, err := strconv.Atoi(strings.ReplaceAll(string(m[1]), ",", ""))
			if err != nil {
				t.Fatalf("%s: unparsable expectation %q: %v", name, m[1], err)
			}

			got := runStateTaxScenario(t, dir, path)
			if got != want {
				t.Errorf("taxOwed = %d, want %d", got, want)
			}
		})
	}
}

// runStateTaxScenario loads the rules fresh, feeds in one scenario, runs
// Compute_Tax, and returns the taxOwed it recorded on the result entity.
func runStateTaxScenario(t *testing.T, dir, scenarioPath string) int {
	t.Helper()

	rs := session.NewRuleSet("StateTax")
	if err := rs.LoadFromDirectory(filepath.Join(dir, "xml")); err != nil {
		t.Fatalf("LoadFromDirectory: %v", err)
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	mapFile, err := os.Open(filepath.Join(dir, "xml/StateTax_map.xml"))
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if err := m.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	dataFile, err := os.Open(scenarioPath)
	if err != nil {
		t.Fatalf("open scenario: %v", err)
	}
	defer dataFile.Close()

	if err := m.LoadDataAndPush(dataFile, []string{"state_config", "job", "taxpayer"}); err != nil {
		t.Fatalf("LoadDataAndPush: %v", err)
	}

	dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Compute_Tax"))
	if err != nil {
		t.Fatalf("GetDecisionTable: %v", err)
	}

	state := sess.GetState()
	if err := dtObj.Execute(state); err != nil {
		t.Fatalf("Compute_Tax: %v", err)
	}

	return resultTaxOwed(t, state)
}

// resultTaxOwed digs the computed tax out of job.results, which Evaluate_Results
// appends a result entity to.
func resultTaxOwed(t *testing.T, state dtrules.State) int {
	t.Helper()

	jobEntity, err := state.FindEntity(dtrules.GetRName("results"))
	if err != nil {
		t.Fatalf("no entity on the stack carries `results`: %v", err)
	}
	resultsObj, err := jobEntity.Get(dtrules.GetRName("results"))
	if err != nil {
		t.Fatalf("job.results: %v", err)
	}
	results, err := resultsObj.RArrayValue()
	if err != nil {
		t.Fatalf("job.results is not an array: %v", err)
	}
	if results.Size() != 1 {
		t.Fatalf("job.results holds %d entries, want 1", results.Size())
	}
	elem, err := results.Get(0)
	if err != nil {
		t.Fatalf("job.results[0]: %v", err)
	}
	result, ok := elem.(dtrules.Entity)
	if !ok {
		t.Fatalf("job.results[0] is %T, want an entity", elem)
	}
	owed, err := result.Get(dtrules.GetRName("taxOwed"))
	if err != nil {
		t.Fatalf("result.taxOwed: %v", err)
	}
	v, err := owed.IntValue()
	if err != nil {
		t.Fatalf("result.taxOwed is not an integer (%s): %v", owed.StringValue(), err)
	}
	return int(v)
}

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

package authoring

import (
	"fmt"
	"strings"
)

// Scenario describes a single test case: inputs, an entry table, and expected
// final state. It is designed to be constructed in code and run against a Project.
type Scenario struct {
	Name       string
	Inputs     map[string]any // "entity.attribute" -> value
	EntryTable string
	Expected   map[string]any // "entity.attribute" -> expected value
}

// ScenarioResult holds the outcome of running a Scenario.
type ScenarioResult struct {
	Scenario *Scenario
	Trace    *ExecutionTrace
	Pass     bool
	Failures []AssertionFailure
	Err      error
}

// AssertionFailure records a single mismatch between expected and actual state.
type AssertionFailure struct {
	Attribute string
	Expected  any
	Actual    any
}

// Run executes the scenario against p and returns the result.
// It resets state first so each Run is isolated.
func (s *Scenario) Run(p *Project) *ScenarioResult {
	result := &ScenarioResult{Scenario: s}

	p.ResetState()

	for key, val := range s.Inputs {
		entity, attr, err := splitEntityAttr(key)
		if err != nil {
			result.Err = fmt.Errorf("input %q: %w", key, err)
			return result
		}
		if err := p.SetAttribute(entity, attr, val); err != nil {
			result.Err = fmt.Errorf("SetAttribute %q: %w", key, err)
			return result
		}
	}

	tbl := p.Table(s.EntryTable)
	if tbl == nil {
		result.Err = fmt.Errorf("table %q not found", s.EntryTable)
		return result
	}

	trace, err := tbl.Execute(p)
	if err != nil {
		result.Err = err
		return result
	}
	result.Trace = trace

	// Build actual map[string]any from trace.FinalState (string values).
	actual := make(map[string]any, len(trace.FinalState))
	for k, v := range trace.FinalState {
		actual[k] = v
	}

	result.Failures = AssertState(actual, s.Expected)
	result.Pass = len(result.Failures) == 0
	return result
}

// AssertState compares actual state against expected. actual keys are
// "entity.attribute" strings with any values; expected keys follow the same
// convention. Returns one AssertionFailure per mismatch or missing key.
func AssertState(actual, expected map[string]any) []AssertionFailure {
	var failures []AssertionFailure
	for key, want := range expected {
		got, ok := actual[key]
		if !ok {
			failures = append(failures, AssertionFailure{
				Attribute: key,
				Expected:  want,
				Actual:    nil,
			})
			continue
		}
		if !semanticEqual(got, want) {
			failures = append(failures, AssertionFailure{
				Attribute: key,
				Expected:  want,
				Actual:    got,
			})
		}
	}
	return failures
}

// semanticEqual compares two values for equality, normalising string
// representations of booleans and numbers so callers can pass typed Go values
// and compare them against string snapshots from FinalState.
func semanticEqual(a, b any) bool {
	if a == b {
		return true
	}
	// Normalise to strings for cross-type comparison (e.g. int 25 vs "25").
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}

// splitEntityAttr splits "entity.attribute" on the first dot.
func splitEntityAttr(key string) (entity, attr string, err error) {
	idx := strings.Index(key, ".")
	if idx < 0 {
		return "", "", fmt.Errorf("expected \"entity.attribute\" format")
	}
	return key[:idx], key[idx+1:], nil
}

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
	"sort"
	"strings"
)

// DiffReport holds the results of a regression diff across two project versions.
type DiffReport struct {
	Total    int
	Matching int
	Diverged []ScenarioDivergence
}

// ScenarioDivergence records one scenario that produced different FinalState
// between the two projects.
type ScenarioDivergence struct {
	Scenario *Scenario
	Diffs    []AttributeDiff
}

// AttributeDiff records a single attribute whose value differs between the two
// project versions.
type AttributeDiff struct {
	Attribute string
	ValueIn1  any
	ValueIn2  any
}

// Summary returns a human-readable summary of the diff report.
func (d *DiffReport) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d total, %d matching, %d diverged", d.Total, d.Matching, len(d.Diverged))
	for _, div := range d.Diverged {
		fmt.Fprintf(&b, "\n  scenario %q:", div.Scenario.Name)
		for _, ad := range div.Diffs {
			fmt.Fprintf(&b, "\n    %s: %v -> %v", ad.Attribute, ad.ValueIn1, ad.ValueIn2)
		}
	}
	return b.String()
}

// Diff runs each scenario against p1 and p2, compares their FinalState maps,
// and returns a DiffReport. Scenarios that error in both projects are excluded
// from both Matching and Diverged counts (shared failure).
func Diff(p1, p2 *Project, scenarios []*Scenario) *DiffReport {
	report := &DiffReport{}

	for _, s := range scenarios {
		r1 := s.Run(p1)
		r2 := s.Run(p2)

		// Shared failure — skip entirely (not divergence, not match)
		if r1.Err != nil && r2.Err != nil {
			continue
		}

		report.Total++

		diffs := diffFinalState(r1, r2)
		if len(diffs) == 0 {
			report.Matching++
			continue
		}

		report.Diverged = append(report.Diverged, ScenarioDivergence{
			Scenario: s,
			Diffs:    diffs,
		})
	}

	return report
}

// diffFinalState compares the FinalState maps from two ScenarioResults and
// returns one AttributeDiff per attribute that differs. Attributes present in
// either result are compared; missing values are treated as empty string.
func diffFinalState(r1, r2 *ScenarioResult) []AttributeDiff {
	keys := make(map[string]struct{})

	if r1.Trace != nil {
		for k := range r1.Trace.FinalState {
			keys[k] = struct{}{}
		}
	}
	if r2.Trace != nil {
		for k := range r2.Trace.FinalState {
			keys[k] = struct{}{}
		}
	}

	// If one errored but the other didn't, treat as full divergence with a
	// synthetic attribute recording the error difference.
	if r1.Err != nil || r2.Err != nil {
		var v1, v2 any
		if r1.Err != nil {
			v1 = r1.Err.Error()
		} else {
			v1 = "<ok>"
		}
		if r2.Err != nil {
			v2 = r2.Err.Error()
		} else {
			v2 = "<ok>"
		}
		return []AttributeDiff{{Attribute: "<execution>", ValueIn1: v1, ValueIn2: v2}}
	}

	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var diffs []AttributeDiff
	for _, k := range sorted {
		v1 := r1.Trace.FinalState[k]
		v2 := r2.Trace.FinalState[k]
		if !semanticEqual(v1, v2) {
			diffs = append(diffs, AttributeDiff{
				Attribute: k,
				ValueIn1:  v1,
				ValueIn2:  v2,
			})
		}
	}
	return diffs
}

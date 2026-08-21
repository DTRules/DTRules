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

package analysis

import "testing"

// Dynamic dispatch resolves through the literal segments of its expression:
// the literals ARE the bound, with nothing declared and nothing to keep in
// sync (#776). CorporateTax's orchestrator is the shape:
//
//	perform table named ("Determine_" + apportionment.state_code + "_Filing_Requirement")
//
// derives ^determine_.*_filing_requirement$ and matches exactly the 51 state
// tables.

func dispatchGraph() *TableCallGraph {
	g := &TableCallGraph{
		Tables: map[string]bool{}, byFold: map[string]string{},
		Calls: map[string]map[string]bool{}, DTFile: map[string]string{},
	}
	for _, n := range []string{"Determine_CA_Filing_Requirement", "Determine_NV_Filing_Requirement",
		"Calculate_CA_State_Tax", "Handle_Default", "Orchestrator"} {
		g.Tables[n] = true
		g.byFold[foldName(n)] = n
	}
	return g
}

func foldName(n string) string {
	out := make([]rune, 0, len(n))
	for _, r := range n {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}

func TestDispatchBoundFromLiterals(t *testing.T) {
	g := dispatchGraph()
	recordDynamicCalls(g, "Orchestrator", "orch_dt.xml",
		`perform table named ("Determine_" + apportionment.state_code + "_Filing_Requirement")`)

	calls := g.Calls["Orchestrator"]
	for _, want := range []string{"Determine_CA_Filing_Requirement", "Determine_NV_Filing_Requirement"} {
		if !calls[want] {
			t.Errorf("bound should reach %s; calls=%v", want, calls)
		}
	}
	if calls["Calculate_CA_State_Tax"] {
		t.Error("the bound must not match tables of a different shape")
	}
	if len(g.UnboundedDispatch) != 0 {
		t.Errorf("a bounded site was reported unbounded: %v", g.UnboundedDispatch)
	}
}

func TestDispatchDefaultIsAnEdge(t *testing.T) {
	g := dispatchGraph()
	recordDynamicCalls(g, "Orchestrator", "orch_dt.xml",
		`perform table named ("Determine_" + r.code + "_Filing_Requirement") with default Handle_Default`)
	if !g.Calls["Orchestrator"]["Handle_Default"] {
		t.Error("the with-default fallback is a reachable target and must be an edge")
	}
}

// An expression with no literal parts derives no bound: the analyzer cannot
// reason about it, and says so rather than staying silent.
func TestUnboundedDispatchIsReported(t *testing.T) {
	g := dispatchGraph()
	recordDynamicCalls(g, "Orchestrator", "orch_dt.xml",
		`perform table named (job.whatever)`)
	if len(g.UnboundedDispatch) != 1 {
		t.Fatalf("want 1 unbounded site, got %v", g.UnboundedDispatch)
	}
	if g.UnboundedDispatch[0].Caller != "Orchestrator" {
		t.Errorf("site should name its caller: %+v", g.UnboundedDispatch[0])
	}
}

// A bound that matches nothing is as unreachable as no bound.
func TestBoundMatchingNothingIsReported(t *testing.T) {
	g := dispatchGraph()
	recordDynamicCalls(g, "Orchestrator", "orch_dt.xml",
		`perform table named ("NoSuch_" + r.x + "_Table")`)
	if len(g.UnboundedDispatch) != 1 {
		t.Errorf("a bound matching no table must be reported: %v", g.UnboundedDispatch)
	}
}

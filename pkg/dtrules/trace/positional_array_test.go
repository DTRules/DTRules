// Copyright 2026 DTRules contributors
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

package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestReplayPositionalArrayOps pins replay of the addat / removeat events
// (positional array mutations). No sample ruleset uses these operators, so
// a hand-built trace is the only coverage: an entity's array attribute is
// bound, values are appended and positionally inserted/removed, and the
// replayed array must land in the recorded order.
func TestReplayPositionalArrayOps(t *testing.T) {
	eddPath := getXMLPath("kidaid_edd.xml")
	dtPath := getXMLPath("kidaid_dt.xml")
	if eddPath == "" {
		t.Skip("Sample project not available")
	}

	rs := session.NewRuleSet("KidAid")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("load EDD: %v", err)
	}
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("open DT: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Fatalf("load DT: %v", err)
	}

	// job.results: [10] -> addat(0, 20) -> [20 10] -> addat(1, 30) ->
	// [20 30 10] -> removeat(0) -> [30 10]
	traceXML := strings.Join([]string{
		`<DTRulesTrace format="2" dtrules_version="test">`,
		`<entitypush entity="job" id="9001"/>`,
		`<arraybind id="9001" attr="results" arrayId="7001"/>`,
		`<addto arrayId="7001">10</addto>`,
		`<addat arrayId="7001" index="0">20</addat>`,
		`<addat arrayId="7001" index="1">30</addat>`,
		`<removeat arrayId="7001" index="0"/>`,
		`<finalState></finalState>`,
		`</DTRulesTrace>`,
	}, "\n")

	path := filepath.Join(t.TempDir(), "positional.trace.xml")
	if err := os.WriteFile(path, []byte(traceXML), 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}

	tr := NewTrace()
	if _, err := tr.Load(path); err != nil {
		t.Fatalf("load trace: %v", err)
	}
	if p := tr.Provenance(); p.Format != "2" {
		t.Errorf("format attribute did not round-trip: %+v", p)
	}

	// Replay to the finalState marker — replay processes every node
	// BEFORE the target, so the marker puts all mutations in scope.
	sess, err := tr.SetState(rs, tr.FinalState())
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	state := sess.GetState()
	e, err := state.FindEntity(dtrules.GetRName("results"))
	if err != nil || e == nil {
		t.Fatalf("no entity with a results attribute on the replayed stack: %v", err)
	}
	v, err := e.Get(dtrules.GetRName("results"))
	if err != nil || v == nil {
		t.Fatalf("get results: %v", err)
	}
	got := strings.Join(strings.Fields(v.StringValue()), " ")
	want := "[ 30 10 ]"
	if got != want {
		t.Errorf("replayed array = %q, want %q", got, want)
	}
}

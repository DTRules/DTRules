// Copyright 2025 DTRules contributors
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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestGoTraceRoundTrip pins the full trace loop against a Go-produced trace
// (the older tests use Java-era fixtures): execute KidAid with real input
// data while tracing, load the trace back, check the recorded structure,
// replay to the end, and verify the replayed state against the recorded
// final state.
func TestGoTraceRoundTrip(t *testing.T) {
	base := getTestDataPath()
	if base == "" {
		t.Skip("Sample project not available")
	}
	xmlDir := filepath.Join(base, "xml")
	input := filepath.Join(base, "testfiles", "TestCase_001.xml")
	if _, err := os.Stat(input); err != nil {
		t.Skip("KidAid test input not available")
	}

	// --- Execute with tracing ---
	rs := session.NewRuleSet("kidaid")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load rules: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("state is not *interpreter.DTState")
	}

	tracePath := filepath.Join(t.TempDir(), "roundtrip_trace.xml")
	f, err := os.Create(tracePath)
	if err != nil {
		t.Fatalf("create trace: %v", err)
	}
	fingerprint, err := FingerprintRules(xmlDir)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	WriteHeader(f, Provenance{DTRulesVersion: "test", RulesFingerprint: fingerprint})
	state.SetOutput(f, nil)
	state.EnableTrace()

	// Load mapping + input data (the same path `dtrules run` takes), so
	// the trace records the initial data as def events.
	mapFile, err := os.Open(filepath.Join(xmlDir, "kidaid_map.xml"))
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()
	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("load mapping: %v", err)
	}
	if err := m.Initialize(); err != nil {
		t.Fatalf("init mapping: %v", err)
	}
	dataFile, err := os.Open(input)
	if err != nil {
		t.Fatalf("open input: %v", err)
	}
	defer dataFile.Close()
	if err := m.LoadData(dataFile); err != nil {
		t.Fatalf("load data: %v", err)
	}

	rsess := sess.(*session.RSession)
	if err := rsess.Execute("Compute_Eligibility"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	WriteFinalState(f, state)
	WriteFooter(f)
	f.Close()

	// --- Load the trace back ---
	tr := NewTrace()
	root, err := tr.Load(tracePath)
	if err != nil {
		t.Fatalf("load trace: %v", err)
	}

	p := tr.Provenance()
	if p.DTRulesVersion != "test" || p.RulesFingerprint != fingerprint {
		t.Errorf("provenance did not round-trip: %+v", p)
	}

	// Structure: the entry table with an execute_table pass, a fired
	// column with actions, condition results, and recorded defs.
	counts := map[string]int{}
	var walk func(n *TraceNode)
	walk = func(n *TraceNode) {
		counts[n.Name]++
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	for _, tag := range []string{"decisiontable", "execute_table", "column", "action", "condition", "def", "entitypush"} {
		if counts[tag] == 0 {
			t.Errorf("trace records no <%s> events", tag)
		}
	}
	if fsNode := tr.FinalState(); fsNode == nil || len(fsNode.Children) == 0 {
		t.Fatalf("trace records no final state")
	}

	// --- Replay to the very end and verify against the recorded state ---
	rs2 := session.NewRuleSet("kidaid-replay")
	if err := rs2.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("load rules for replay: %v", err)
	}
	// Replay everything before the finalState node.
	fsNode := tr.FinalState()
	replaySess, err := tr.SetState(rs2, fsNode)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	mismatches := tr.VerifyFinalState(replaySess.GetState())
	if len(mismatches) > 0 {
		for i, m := range mismatches {
			if i >= 15 {
				t.Errorf("... and %d more", len(mismatches)-15)
				break
			}
			t.Errorf("final-state mismatch: %s", m)
		}
	}
}

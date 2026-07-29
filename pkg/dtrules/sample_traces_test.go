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
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// sampleRun describes a project the way `dtrules run` sees it: an entry table
// declared in DTRules.xml, and an input file to feed through the mapping.
type sampleRun struct {
	project string
	input   string
	// minColumns guards against a run that loads everything and then
	// executes almost nothing — the failure mode that hid a missing
	// operator and an empty bracket schedule for as long as it did.
	minColumns int
}

// TestSampleProjectsProduceLoadableTraces is the campaign's definition of a
// working sample: it runs from the command line on a scenario, executes real
// rules, and leaves a trace an editor can open (#948).
//
// This exercises the same sequence `dtrules run --trace` does, including
// LoadDataAndPushSingletons — the load path where entities are built from the
// input before the singletons are pushed. Poker's map had no createentity for
// its root <game> tag, so the game entity was created after its players and
// game.players came out empty; the project ran, wrote a trace, exited 0, and
// had decided nothing.
func TestSampleProjectsProduceLoadableTraces(t *testing.T) {
	samples := []sampleRun{
		{
			project:    "TestProject",
			input:      "testfiles/TestScenarios/TestCase_001.xml",
			minColumns: 4, // one per thing
		},
		{
			project:    "StateTax",
			input:      "testfiles/TestScenarios/TestCase_AL_progressive.xml",
			minColumns: 5, // the tables Compute_Tax routes through
		},
		{
			project:    "SinusitisTherapy",
			input:      "testfiles/TestScenarios/AdultStandard/input.xml",
			minColumns: 5, // orchestrator + medication, dose, CCr, interactions
		},
		{
			project:    "CHIP",
			input:      "testfiles/TestScenarios/TestCase_001.xml",
			minColumns: 8, // income, group size, and the three program tables
		},
		{
			project:    "Poker",
			input:      "testfiles/TestScenarios/basic_scenarios.xml",
			minColumns: 24, // 12 players, at least two tables deep
		},
	}

	for _, s := range samples {
		t.Run(s.project, func(t *testing.T) {
			dir := sampleProjectDir(t, s.project)
			if dir == "" {
				t.Skipf("%s sample project not found", s.project)
			}

			entry := declaredEntry(t, dir)
			if entry == "" {
				t.Fatalf("%s/DTRules.xml declares no <entry> — `dtrules debug` and the editor have nothing to run", s.project)
			}

			tracePath := filepath.Join(t.TempDir(), "run.trace.xml")
			runSampleWithTrace(t, dir, entry, filepath.Join(dir, s.input), tracePath)

			root, err := trace.LoadFile(tracePath)
			if err != nil {
				t.Fatalf("trace does not load: %v", err)
			}
			if columns := countTraceNodes(root, "column"); columns < s.minColumns {
				t.Errorf("trace records %d fired columns, want at least %d — the run executed almost nothing",
					columns, s.minColumns)
			}
		})
	}
}

// runSampleWithTrace mirrors cmd/dtrules run: open the project, wire the trace
// writer, load the input through the mapping, execute the entry table.
func runSampleWithTrace(t *testing.T, dir, entry, input, tracePath string) {
	t.Helper()

	xmlDir := filepath.Join(dir, "xml")
	rs := session.NewRuleSet(filepath.Base(dir))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("LoadFromDirectory: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	traceFile, err := os.Create(tracePath)
	if err != nil {
		t.Fatalf("create trace: %v", err)
	}
	defer traceFile.Close()

	fingerprint, err := trace.FingerprintRules(xmlDir)
	if err != nil {
		t.Fatalf("FingerprintRules: %v", err)
	}
	trace.WriteHeader(traceFile, trace.Provenance{
		DTRulesVersion:   "test",
		RulesFingerprint: fingerprint,
	})

	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("session state is %T, want *interpreter.DTState", sess.GetState())
	}
	state.SetOutput(traceFile, nil)
	state.EnableTrace()

	maps, _ := filepath.Glob(filepath.Join(xmlDir, "*_map.xml"))
	if len(maps) == 0 {
		t.Fatalf("%s has no mapping file", dir)
	}
	mapFile, err := os.Open(maps[0])
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	inputFile, err := os.Open(input)
	if err != nil {
		t.Fatalf("open input: %v", err)
	}
	defer inputFile.Close()
	if err := m.LoadDataAndPushSingletons(inputFile); err != nil {
		t.Fatalf("LoadDataAndPushSingletons: %v", err)
	}

	dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))
	if err != nil {
		t.Fatalf("entry table %q: %v", entry, err)
	}
	if err := dtObj.Execute(sess.GetState()); err != nil {
		t.Fatalf("%s: %v", entry, err)
	}

	trace.WriteFinalState(traceFile, sess.GetState())
	trace.WriteFooter(traceFile)
}

// declaredEntry reads <entry> out of a project's DTRules.xml.
//
// A malformed marker is reported as such rather than as a missing entry.
// They are easy to confuse and the fix is completely different: XML forbids
// "--" inside a comment, so a marker documenting a flag like the interactive
// one parses as garbage and every <entry> in it silently disappears.
func declaredEntry(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "DTRules.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		Entry string `xml:"entry"`
	}
	if err := xml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("%s is not well-formed XML: %v", path, err)
	}
	return strings.TrimSpace(cfg.Entry)
}

// countTraceNodes counts nodes of a tag anywhere in the trace tree.
func countTraceNodes(n *trace.TraceNode, tag string) int {
	if n == nil {
		return 0
	}
	count := 0
	if n.Name == tag {
		count++
	}
	for _, child := range n.Children {
		count += countTraceNodes(child, tag)
	}
	return count
}

// sampleProjectDir locates a sample project relative to the test.
func sampleProjectDir(t *testing.T, name string) string {
	t.Helper()
	for _, p := range []string{"../../sampleprojects", "../sampleprojects", "sampleprojects"} {
		abs, err := filepath.Abs(filepath.Join(p, name))
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "xml")); err == nil {
			return abs
		}
	}
	return ""
}

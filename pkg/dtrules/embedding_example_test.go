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

// Embedding example and its doc gate (#1211).
//
// README.md ("Embedding in a Go application") and `dtrules docs embedding`
// both show the supported way to drive the engine from Go. Before this file
// they showed it and nothing checked it: the README's snippet was framed as a
// stopgap until `pkg/dtrules/sdk` landed, and that package was deleted as the
// wrong approach (69774f70, #493) — DTRules loads entity values from XML
// through the EDD, so a parallel programmatic entity API had nothing to add.
//
// The two functions below ARE the documented path. They compile against the
// current packages, they produce the same answers `dtrules run` does, and
// TestEmbeddingDocsShowTheSupportedPath fails if the prose drifts away from
// them.
package dtrules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/datafile"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// removalCommit deleted pkg/dtrules/sdk: "DTRules loads entity arrays from XML
// using mapping and EDD, not via a programmatic SDK API."
const removalCommit = "69774f70"

// runMapped executes an entry table over an input document that has to be
// reconciled against the EDD by the project's mapping — the path `dtrules run
// --input` takes, and the one to use when the data comes from somewhere whose
// tag names are not yours to choose.
func runMapped(xmlDir, mapFile, inputFile, entry string) (dtrules.Entity, error) {
	rs := session.NewRuleSet(filepath.Base(filepath.Dir(xmlDir)))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		return nil, err
	}
	sess, err := rs.NewSession()
	if err != nil {
		return nil, err
	}

	mf, err := os.Open(mapFile)
	if err != nil {
		return nil, err
	}
	defer mf.Close()
	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mf); err != nil {
		return nil, err
	}
	in, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	// Reads the document, then pushes the cardinality-1 entities it created,
	// so the stack holds the loaded instances rather than empty singletons.
	if err := m.LoadDataAndPushSingletons(in); err != nil {
		return nil, err
	}

	state := sess.GetState()
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))
	if err != nil {
		return nil, err
	}
	if err := dt.Execute(state); err != nil {
		return nil, err
	}
	return state.FindEntity(dtrules.GetRName("result"))
}

// runCanonical executes an entry table over canonical data XML — tags 1:1 with
// the EDD, so no mapping is consulted. This is the path for an embedder that
// owns its own data: push the singletons the rules resolve bare names against
// (the same set a mapping's <initialentity> names), read the data into them,
// execute, read the result.
func runCanonical(xmlDir, dataFile, entry string, singletons []string) (dtrules.Entity, error) {
	rs := session.NewRuleSet(filepath.Base(filepath.Dir(xmlDir)))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		return nil, err
	}
	sess, err := rs.NewSession()
	if err != nil {
		return nil, err
	}
	state := sess.GetState()
	for _, name := range singletons {
		e, err := sess.CreateEntity(dtrules.GetRName(name))
		if err != nil {
			return nil, err
		}
		if err := state.EntityPush(e); err != nil {
			return nil, err
		}
	}

	df, err := os.Open(dataFile)
	if err != nil {
		return nil, err
	}
	defer df.Close()
	find := func(name string) *entity.REntity {
		e, err := state.FindEntity(dtrules.GetRName(name))
		if err != nil || e == nil {
			return nil
		}
		re, _ := e.(*entity.REntity)
		return re
	}
	create := func(subtype string) (*entity.REntity, error) {
		e, err := sess.CreateEntity(dtrules.GetRName(subtype))
		if err != nil {
			return nil, err
		}
		re, _ := e.(*entity.REntity)
		return re, nil
	}
	// Authoritative: the values are final, so an interactive run would not
	// re-ask for them. Review loads the same values but leaves them defaulted.
	if err := datafile.Read(df, find, create, datafile.Authoritative); err != nil {
		return nil, err
	}

	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))
	if err != nil {
		return nil, err
	}
	if err := dt.Execute(state); err != nil {
		return nil, err
	}
	return state.FindEntity(dtrules.GetRName("result"))
}

// sinusitisDir locates the sample project both embedding tests run against.
func sinusitisDir(t *testing.T) string {
	t.Helper()
	base := findSampleProjectsDir(t)
	if base == "" {
		t.Skip("Sample projects directory not found")
	}
	dir := filepath.Join(base, "SinusitisTherapy")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("SinusitisTherapy project not found")
	}
	return dir
}

// TestEmbeddingExampleMappedInput is test vector 3 of #1211: the documented
// embedding path, run on SinusitisTherapy + TestScenarios/AdultStandard, must
// answer exactly what `dtrules run` answers.
func TestEmbeddingExampleMappedInput(t *testing.T) {
	sample := sinusitisDir(t)
	result, err := runMapped(
		filepath.Join(sample, "xml"),
		filepath.Join(sample, "xml", "sinusitis_map.xml"),
		filepath.Join(sample, "testfiles", "TestScenarios", "AdultStandard", "input.xml"),
		"Determine_Therapy",
	)
	if err != nil {
		t.Fatalf("documented embedding path failed: %v", err)
	}
	assertAdultStandard(t, result)
}

// TestEmbeddingExampleCanonicalData is test vector 4: the same scenario
// expressed as canonical data XML and loaded with no mapping at all must reach
// the same answer, so an embedder that owns its data can skip the mapping.
func TestEmbeddingExampleCanonicalData(t *testing.T) {
	sample := sinusitisDir(t)
	result, err := runCanonical(
		filepath.Join(sample, "xml"),
		filepath.Join("testdata", "embedding", "adult_standard_data.xml"),
		"Determine_Therapy",
		[]string{"constants", "result", "patient"},
	)
	if err != nil {
		t.Fatalf("documented canonical embedding path failed: %v", err)
	}
	assertAdultStandard(t, result)
}

// assertAdultStandard pins the two figures #1211 names, plus the dosing the
// scenario's own comment declares.
func assertAdultStandard(t *testing.T, result dtrules.Entity) {
	t.Helper()
	if result == nil {
		t.Fatal("no result entity on the stack after execution")
	}
	if got := getStringAttr(result, "recommended_drug"); got != "Amoxicillin" {
		t.Errorf("recommended_drug = %q, want %q", got, "Amoxicillin")
	}
	if got := getFloatAttr(result, "dose_mg"); got != 500 {
		t.Errorf("dose_mg = %v, want 500", got)
	}
	if got := getFloatAttr(result, "frequency_hours"); got != 24 {
		t.Errorf("frequency_hours = %v, want 24", got)
	}
	if got := getFloatAttr(result, "duration_days"); got != 14 {
		t.Errorf("duration_days = %v, want 14", got)
	}
	if got := getStringArrayAttr(result, "warnings"); len(got) != 0 {
		t.Errorf("warnings = %v, want none", got)
	}
}

// TestEmbeddingDocsShowTheSupportedPath is test vector 1 plus the guard that
// the prose still describes these functions.
//
// The grep half is the one that matters: README.md and .claude/CLAUDE.md told
// readers to wait for `pkg/dtrules/sdk` (#757) long after the package was
// deleted, and CLAUDE.md's reader is an AI session that takes "is being
// extracted" as a live instruction.
//
// The issue's vector 1 asks for no mention of the package at all, and its
// proposal asks for one sentence on why the package is gone. Those pull
// against each other, and the sentence wins: an idea that was tried and
// rejected has to be named to stay rejected. So what is banned is the package
// cited as FORTHCOMING — the closed issue number, and the phrases that make a
// reader wait. A doc may name `pkg/dtrules/sdk` only alongside the commit that
// deleted it.
func TestEmbeddingDocsShowTheSupportedPath(t *testing.T) {
	repo := filepath.Join("..", "..")
	docs := []string{
		filepath.Join(repo, "README.md"),
		filepath.Join(repo, ".claude", "CLAUDE.md"),
		filepath.Join(repo, "docs", "SPEC.md"),
		filepath.Join(repo, "cmd", "dtrules", "doc_embedding.go"),
		filepath.Join(repo, "cmd", "dtrules", "doc_architecture.go"),
		filepath.Join(repo, "cmd", "dtrules", "docs.go"),
	}
	for _, path := range docs {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(body)
		name := filepath.Base(path)
		for _, dead := range []string{"#757", "issues/757"} {
			if strings.Contains(text, dead) {
				t.Errorf("%s cites %s; it is closed, and the SDK it tracked was removed in 69774f70 (#1211)",
					name, dead)
			}
		}
		for _, waiting := range []string{
			"sdk package for embeddable engine wiring is being",
			"is in progress",
			"until it lands",
			"until the sdk lands",
			"sdk is being extracted",
		} {
			if strings.Contains(strings.ToLower(text), waiting) {
				t.Errorf("%s tells the reader to wait (%q); nothing is coming (#1211)", name, waiting)
			}
		}
		if strings.Contains(text, "pkg/dtrules/sdk") && !strings.Contains(text, removalCommit) {
			t.Errorf("%s names pkg/dtrules/sdk without saying it was removed in %s (#1211)",
				name, removalCommit)
		}
	}

	readme, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	// The README's snippet is this file's code. If a call in the snippet stops
	// being the call the example makes, they have drifted.
	for _, call := range []string{
		"session.NewRuleSet(",
		"rs.LoadFromDirectory(",
		"rs.NewSession()",
		"m.LoadDataAndPushSingletons(",
		"datafile.Read(",
		"state.EntityPush(",
		"dt.Execute(state)",
		"state.FindEntity(",
	} {
		if !strings.Contains(string(readme), call) {
			t.Errorf("README embedding section no longer shows %s, which the example uses", call)
		}
	}
}

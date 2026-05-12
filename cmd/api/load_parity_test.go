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

package main

import (
	"testing"
)

// Load-parity tests for #757. The premise of #757 was that
// cmd/dtrules and cmd/api each glue the engine pipeline together
// independently and could drift. Rather than collapsing the two
// surfaces into an SDK package (heavier refactor, complicated by
// the binaries' different error policies — cmd/dtrules fatals,
// cmd/api logs-and-continues), these tests pin both load surfaces
// to the same observable contract against the CHIP sample project:
//
//   1. buildRuleSetFromXML(...) returns a non-nil RuleSet
//   2. The rule set lists Compute_Eligibility and at least one entity
//   3. NewSession() succeeds against the loaded rule set
//
// The companion test cmd/dtrules/load_parity_test.go asserts the
// same invariants against cmd/dtrules' loadRuleSet helper. Any change
// to either load path that breaks the contract fails its package's
// test, so a PR that touches one without the other surfaces the
// drift immediately rather than at the next time a downstream caller
// hits an unexpected RuleSet shape.

const chipProjectPath = "../../sampleprojects/CHIP/xml"

// TestBuildRuleSetFromXML_CHIP loads the CHIP sample through the API
// server's helper and asserts the runtime contract. The CHIP fixture
// is small, deterministic, and exercises both EDD and DT loading.
func TestBuildRuleSetFromXML_CHIP(t *testing.T) {
	rs := buildRuleSetFromXML(
		"test", chipProjectPath,
		[]string{"CHIP_edd.xml"},
		[]string{"CHIP_dt.xml"},
		func(string, ...any) {}, // discard warnings — CHIP loads cleanly
	)
	if rs == nil {
		t.Fatalf("buildRuleSetFromXML returned nil")
	}

	tables := rs.GetDecisionTableNames()
	if len(tables) == 0 {
		t.Errorf("expected at least one decision table loaded, got 0")
	}
	foundEligibility := false
	for _, n := range tables {
		if n.StringValue() == "Compute_Eligibility" {
			foundEligibility = true
			break
		}
	}
	if !foundEligibility {
		t.Errorf("CHIP rule set missing Compute_Eligibility table; got %v", tables)
	}

	entities := rs.GetEntityNames()
	if len(entities) == 0 {
		t.Errorf("expected at least one entity loaded, got 0")
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession on loaded rule set: %v", err)
	}
	if sess == nil {
		t.Fatalf("NewSession returned nil session")
	}
}

// TestBuildRuleSetFromXML_WarningsOnMissingFile verifies the
// log-and-continue policy: a missing file produces a warning and
// the build still returns a usable rule set for whatever DID load.
// This is the deliberate policy difference from cmd/dtrules and the
// reason the load logic isn't shared verbatim.
func TestBuildRuleSetFromXML_WarningsOnMissingFile(t *testing.T) {
	var warnings []string
	collect := func(format string, args ...any) {
		warnings = append(warnings, formatLoadWarning(format, args...))
	}
	rs := buildRuleSetFromXML(
		"test", chipProjectPath,
		[]string{"DOES_NOT_EXIST_edd.xml"},
		[]string{"CHIP_dt.xml"},
		collect,
	)
	if rs == nil {
		t.Fatalf("buildRuleSetFromXML returned nil despite missing EDD")
	}
	if len(warnings) == 0 {
		t.Errorf("expected a warning for the missing EDD file, got none")
	}
}

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

// Load-parity test for #757. Pins cmd/dtrules' loadRuleSet helper to
// the same observable contract that cmd/api's buildRuleSetFromXML
// asserts in cmd/api/load_parity_test.go:
//
//   1. loadRuleSet returns a non-nil RuleSet with no error against
//      the CHIP fixture.
//   2. The rule set lists Compute_Eligibility and at least one entity.
//   3. NewSession() succeeds against the loaded rule set.
//
// If a future change to either binary's load path drifts away from
// this contract — e.g. a new loading step is added in cmd/api but
// not here, or vice versa — the parity test for the diverging caller
// fails and the PR review surfaces it.
//
// The error policies of the two callers are intentionally different
// (cmd/dtrules returns a hard error, cmd/api logs warnings and
// continues). Drift in behaviour, not policy, is what these tests
// catch.

func TestLoadRuleSet_CHIP(t *testing.T) {
	rs, err := loadRuleSet(
		"../../sampleprojects/CHIP/xml/CHIP_edd.xml",
		"../../sampleprojects/CHIP/xml/CHIP_dt.xml",
	)
	if err != nil {
		t.Fatalf("loadRuleSet on CHIP: %v", err)
	}
	if rs == nil {
		t.Fatalf("loadRuleSet returned nil rule set")
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

// TestLoadRuleSet_MissingFile_Fatals verifies the hard-error policy
// for cmd/dtrules: a missing EDD or DT file fails the call rather
// than logging-and-continuing. This is the deliberate policy
// difference from cmd/api's buildRuleSetFromXML and the reason the
// two callers aren't sharing one implementation.
func TestLoadRuleSet_MissingFile_Fatals(t *testing.T) {
	_, err := loadRuleSet(
		"/does/not/exist/edd.xml",
		"../../sampleprojects/CHIP/xml/CHIP_dt.xml",
	)
	if err == nil {
		t.Errorf("expected error from loadRuleSet with missing EDD, got nil")
	}
}

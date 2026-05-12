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

package dtrules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// TestTaxReturn_NoLegacyPostfix is a regression guard: every decision table
// in the TaxReturn project must carry EL DSL alongside any postfix. The
// loader emits a "hand-coded postfix without EL DSL" WARNING for tables
// that fail this check, but a WARNING isn't a test failure. This test
// turns the warning into an error so a future hand-coded postfix table
// can't slip in unnoticed.
//
// "Legacy" here means: the table has non-empty <action_postfix> /
// <condition_postfix> / <initial_action_postfix> / <context_postfix>
// content but no corresponding EL DSL. Mixed tables (postfix AND DSL)
// are NOT flagged — that's the supported state where DSL is the
// authoritative form and postfix is the generated artifact.
func TestTaxReturn_NoLegacyPostfix(t *testing.T) {
	cwd, _ := os.Getwd()
	xmlDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn", "xml")

	result, err := sync.ValidateELCompliance(xmlDir)
	if err != nil {
		t.Fatalf("ValidateELCompliance failed: %v", err)
	}

	if !result.IsCompliant() {
		t.Errorf("TaxReturn project has %d legacy-postfix file(s); each must be re-authored in EL DSL so the loader can compile postfix from EL:", len(result.LegacyFiles))
		for _, f := range result.LegacyFiles {
			rel, err := filepath.Rel(xmlDir, f)
			if err != nil {
				rel = f
			}
			t.Errorf("  - %s", rel)
		}
		if len(result.Errors) > 0 {
			t.Errorf("Validation errors:")
			for _, e := range result.Errors {
				t.Errorf("  - %v", e)
			}
		}
	}
}

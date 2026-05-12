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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// buildRuleSetFromXML wires up the session.RuleSet pipeline that the
// HTTP server uses to execute decision tables. It opens each EDD and
// DT file under projectPath, loads them in order (EDD before DT), and
// returns the resulting rule set. Failures are reported via warn so
// the caller can choose its policy — the API server logs and keeps
// going, cmd/dtrules fatals.
//
// This helper exists alongside cmd/dtrules' loadRuleSet (#757) so
// both binaries route through code that's directly comparable. The
// parity tests in load_parity_test.go (cmd/api) and
// cmd/dtrules/load_parity_test.go pin both surfaces against the
// same CHIP fixture, so drift between them shows up as a failing
// test rather than a silent behaviour difference.
func buildRuleSetFromXML(name, projectPath string, eddFiles, dtFiles []string, warn func(fmtStr string, args ...any)) *session.RuleSet {
	rs := session.NewRuleSet(name)

	for _, eddFile := range eddFiles {
		path := filepath.Join(projectPath, eddFile)
		loadFileInto(rs, path, eddFile, "EDD", warn)
	}
	for _, dtFile := range dtFiles {
		path := filepath.Join(projectPath, dtFile)
		loadFileInto(rs, path, dtFile, "DT", warn)
	}
	return rs
}

// loadFileInto handles the open + load + close + warn sequence for one
// file. Extracted so EDD and DT paths use the same error semantics.
func loadFileInto(rs *session.RuleSet, path, relPath, kind string, warn func(string, ...any)) {
	f, err := os.Open(path)
	if err != nil {
		warn("Warning: Failed to open %s file %s for rule set: %v", kind, relPath, err)
		return
	}
	defer f.Close()

	switch kind {
	case "EDD":
		if err := rs.LoadEDD(f); err != nil {
			warn("Warning: Failed to load %s file %s into rule set: %v", kind, relPath, err)
		}
	case "DT":
		if err := rs.LoadDecisionTables(f); err != nil {
			warn("Warning: Failed to load %s file %s into rule set: %v", kind, relPath, err)
		}
	default:
		warn("Warning: unknown rule-set file kind %q for %s", kind, relPath)
	}
}

// logLoadWarning is the default warning sink for buildRuleSetFromXML
// — routes through the standard library logger, matching the API
// server's existing log style.
func logLoadWarning(format string, args ...any) {
	log.Printf(format, args...)
}

// formatLoadWarning is a non-logging sink suitable for tests that
// want to inspect warnings rather than emit them. Returns the
// formatted string for the caller to collect.
func formatLoadWarning(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

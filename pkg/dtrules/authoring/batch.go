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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScenarioFile is the JSON schema for a scenario file on disk.
type ScenarioFile struct {
	Name       string         `json:"name"`
	EntryTable string         `json:"entryTable"`
	Inputs     map[string]any `json:"inputs"`
	Expected   map[string]any `json:"expected"`
}

// BatchResult holds the aggregate outcome of RunAllScenarios.
type BatchResult struct {
	Results []ScenarioResult
	Passed  int
	Failed  int
	Skipped int
}

// AllPassed returns true only when there are no failures and no skipped scenarios.
func (b *BatchResult) AllPassed() bool {
	return b.Failed == 0 && b.Skipped == 0
}

// Summary returns a one-line human-readable summary. When there are failures,
// each failing scenario is listed on its own line.
func (b *BatchResult) Summary() string {
	line := fmt.Sprintf("%d passed, %d failed, %d skipped", b.Passed, b.Failed, b.Skipped)
	if b.Failed == 0 {
		return line
	}
	var sb strings.Builder
	sb.WriteString(line)
	for i := range b.Results {
		r := &b.Results[i]
		if r.Pass || r.Err != nil {
			continue
		}
		name := ""
		if r.Scenario != nil {
			name = r.Scenario.Name
		}
		sb.WriteString("\n  FAIL: ")
		sb.WriteString(name)
		for _, f := range r.Failures {
			sb.WriteString(fmt.Sprintf(" [%s: want %v got %v]", f.Attribute, f.Expected, f.Actual))
		}
	}
	return sb.String()
}

// RunAllScenarios walks dir non-recursively, reads every *.json file as a
// Scenario, runs it against p, and returns the aggregate BatchResult.
// Files that cannot be parsed contribute a Skipped entry; all other files
// still run regardless of parse or execution errors in sibling files.
func (p *Project) RunAllScenarios(dir string) (*BatchResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	batch := &BatchResult{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			batch.Results = append(batch.Results, ScenarioResult{Err: fmt.Errorf("read %s: %w", entry.Name(), err)})
			batch.Skipped++
			continue
		}

		var sf ScenarioFile
		if err := json.Unmarshal(data, &sf); err != nil {
			batch.Results = append(batch.Results, ScenarioResult{Err: fmt.Errorf("parse %s: %w", entry.Name(), err)})
			batch.Skipped++
			continue
		}

		s := &Scenario{
			Name:       sf.Name,
			EntryTable: sf.EntryTable,
			Inputs:     sf.Inputs,
			Expected:   sf.Expected,
		}

		result := s.Run(p)
		batch.Results = append(batch.Results, *result)

		switch {
		case result.Err != nil:
			batch.Skipped++
		case result.Pass:
			batch.Passed++
		default:
			batch.Failed++
		}
	}

	return batch, nil
}

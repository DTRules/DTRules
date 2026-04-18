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

package sync

import "fmt"

// BuildSummary holds counters and drop records for a complete build run.
// It covers both steps of an XML-authored build (export + re-import) or
// the single step of an Excel-authored build (import only).
type BuildSummary struct {
	ExportStep *StepSummary
	ImportStep *StepSummary
}

// StepSummary holds the artifact counts and drop list for one pipeline step.
type StepSummary struct {
	Tables          int    // number of decision tables processed
	Actions         int    // number of action rows across all tables
	Conditions      int    // number of condition rows across all tables
	Entities        int    // number of EDD entities processed
	Mappings        int    // number of MAP entries processed
	PostfixStripped int    // export step: <*_postfix> blocks absent from Excel output
	Compiled        int    // import step: postfix blocks regenerated from EL
	Drops           []Drop // anything lost with a reason
	FilesWritten    int
}

// Drop records one artifact that was silently discarded, with a location and reason.
type Drop struct {
	Table  string // decision table name, or "" for EDD/MAP artifacts
	Column int    // 0 if not applicable
	Item   string // e.g. "action 3", "condition 2", "entity foo"
	Reason string
}

// HasErrors returns true when any drops were recorded in either step.
func (s *BuildSummary) HasErrors() bool {
	if s == nil {
		return false
	}
	drops := 0
	if s.ExportStep != nil {
		drops += len(s.ExportStep.Drops)
	}
	if s.ImportStep != nil {
		drops += len(s.ImportStep.Drops)
	}
	return drops > 0
}

// AddDrop appends a drop record to this step.
func (s *StepSummary) AddDrop(table string, column int, item, reason string) {
	s.Drops = append(s.Drops, Drop{
		Table:  table,
		Column: column,
		Item:   item,
		Reason: reason,
	})
}

// Format returns a multi-line human-readable summary string.
func (s *BuildSummary) Format() string {
	if s == nil {
		return "Build summary: (none)\n"
	}

	out := "Build Summary\n"
	out += "=============\n"

	if s.ExportStep != nil {
		out += "\nExport step (XML → Excel):\n"
		out += s.ExportStep.format()
	}
	if s.ImportStep != nil {
		out += "\nImport step (Excel → XML):\n"
		out += s.ImportStep.format()
	}

	totalDrops := 0
	if s.ExportStep != nil {
		totalDrops += len(s.ExportStep.Drops)
	}
	if s.ImportStep != nil {
		totalDrops += len(s.ImportStep.Drops)
	}

	out += "\n"
	if totalDrops == 0 {
		out += "Result: OK — no drops\n"
	} else {
		out += fmt.Sprintf("Result: FAIL — %d drop(s) detected\n", totalDrops)
	}
	return out
}

func (s *StepSummary) format() string {
	if s == nil {
		return "  (no data)\n"
	}
	out := fmt.Sprintf("  tables=%d  actions=%d  conditions=%d  entities=%d  mappings=%d\n",
		s.Tables, s.Actions, s.Conditions, s.Entities, s.Mappings)
	if s.PostfixStripped > 0 {
		out += fmt.Sprintf("  postfix-stripped=%d\n", s.PostfixStripped)
	}
	if s.Compiled > 0 {
		out += fmt.Sprintf("  compiled=%d\n", s.Compiled)
	}
	out += fmt.Sprintf("  files-written=%d\n", s.FilesWritten)
	if len(s.Drops) == 0 {
		out += "  drops: none\n"
	} else {
		out += fmt.Sprintf("  drops: %d\n", len(s.Drops))
		for _, d := range s.Drops {
			if d.Table != "" {
				out += fmt.Sprintf("    table=%q  col=%d  item=%q  reason=%s\n",
					d.Table, d.Column, d.Item, d.Reason)
			} else {
				out += fmt.Sprintf("    item=%q  reason=%s\n", d.Item, d.Reason)
			}
		}
	}
	return out
}

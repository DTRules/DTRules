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

package excel

// StatDrop records one artifact that was dropped (e.g. EL compile failure).
type StatDrop struct {
	Table  string // decision table name, or "" for non-DT artifacts
	Column int    // 0 if not applicable
	Item   string // e.g. "action 3", "condition 2", "entity foo"
	Reason string
}

// ImportStats collects counters accumulated during an Excel→XML import.
type ImportStats struct {
	Tables     int
	Actions    int
	Conditions int
	Entities   int
	Mappings   int
	Compiled   int // EL expressions successfully compiled to postfix
	Drops      []StatDrop
	Warnings   []StatDrop // non-fatal (e.g., legacy prose-DSL pre-#504)
	Files      int
}

// ExportStats collects counters accumulated during an XML→Excel export.
type ExportStats struct {
	Tables          int
	Actions         int
	Conditions      int
	Entities        int
	Mappings        int
	PostfixStripped int // postfix blocks present in XML but not written to Excel
	Drops           []StatDrop
	Files           int
}

// AddDrop appends a fatal drop record.
func (s *ImportStats) AddDrop(table string, column int, item, reason string) {
	s.Drops = append(s.Drops, StatDrop{
		Table:  table,
		Column: column,
		Item:   item,
		Reason: reason,
	})
}

// AddWarning appends a non-fatal issue. Used for legacy prose-DSL entries
// (DSL field equals the comment text verbatim) that fail to compile but
// predate #504; the round-trip preserves them and the build stays green.
func (s *ImportStats) AddWarning(table string, column int, item, reason string) {
	s.Warnings = append(s.Warnings, StatDrop{
		Table:  table,
		Column: column,
		Item:   item,
		Reason: reason,
	})
}

// AddDrop appends a drop record.
func (s *ExportStats) AddDrop(table string, column int, item, reason string) {
	s.Drops = append(s.Drops, StatDrop{
		Table:  table,
		Column: column,
		Item:   item,
		Reason: reason,
	})
}

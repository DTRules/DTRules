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

package authoring

import (
	"fmt"
	"strings"
)

// OtherwiseRule is the one-line statement of what '*' means, quoted into every
// error the authoring surfaces raise about it so an author (or an AI session
// reading the error) learns the rule from the refusal itself.
const OtherwiseRule = "'*' marks the otherwise column, which is only allowed in the last column " +
	"and only when that column has no Y or N"

// checkOtherwise refuses a table whose use of '*' the engine would refuse at
// load, and does it before anything is written.
//
// The rule is the engine's, character for character (see
// RDecisionTable.validateOtherwiseColumn): '*' does not mean "don't care" --
// that is '-'. It marks the otherwise column, which fires iff no other column
// fired, in every table type. It is legal only in the last column the author
// wrote, and only when that column holds no Y or N.
//
// `table put` used to accept '*' in any cell of any column, which meant the
// XML was already on disk by the time the engine's loader objected, and
// `table patch` refused '*' outright, which meant the otherwise column could
// not be authored at all. Both are wrong, and both now ask this (#1215).
func (t *Table) checkOtherwise() error {
	last := t.lastSpecifiedColumn()
	if last < 1 {
		return nil
	}

	starRow := 0
	for _, c := range t.Conditions {
		for col, v := range c.Columns {
			if !isStarCell(v) {
				continue
			}
			if col != last {
				return fmt.Errorf("condition %d, column %d: %s (the last column is column %d)",
					c.Number, col, OtherwiseRule, last)
			}
			if starRow == 0 || c.Number < starRow {
				starRow = c.Number
			}
		}
	}
	if starRow == 0 {
		return nil
	}
	for _, c := range t.Conditions {
		v := strings.ToUpper(strings.TrimSpace(c.Columns[last]))
		if v == "Y" || v == "N" {
			return fmt.Errorf("condition %d, column %d is %q, but column %d is marked '*' at condition %d: %s",
				c.Number, last, v, last, starRow, OtherwiseRule)
		}
	}
	return nil
}

// lastSpecifiedColumn is the authoring view of the engine's rule for "the last
// column": the rightmost column holding a Y, N or '*' in a condition, or an X
// in an action. Columns past it are padding -- a row may carry a '-' out to
// the table's width -- and the otherwise column is the last real column, not
// the last cell of the padding.
func (t *Table) lastSpecifiedColumn() int {
	last := 0
	for _, c := range t.Conditions {
		for col, v := range c.Columns {
			if col <= last {
				continue
			}
			switch {
			case isStarCell(v), strings.EqualFold(strings.TrimSpace(v), "y"), strings.EqualFold(strings.TrimSpace(v), "n"):
				last = col
			}
		}
	}
	for _, a := range t.Actions {
		for col, on := range a.Columns {
			if on && col > last {
				last = col
			}
		}
	}
	return last
}

// otherwiseColumn is the table's otherwise column (1-based), or 0 when it has
// none: the last specified column, when a condition marks it '*'.
func (t *Table) otherwiseColumn() int {
	last := t.lastSpecifiedColumn()
	if last < 1 {
		return 0
	}
	for _, c := range t.Conditions {
		if isStarCell(c.Columns[last]) {
			return last
		}
	}
	return 0
}

// shiftColumnsRight moves every condition and action cell in column from and
// beyond one column to the right, leaving column from empty.
func (t *Table) shiftColumnsRight(from int) {
	for i := range t.Conditions {
		shifted := make(map[int]string, len(t.Conditions[i].Columns))
		for n, v := range t.Conditions[i].Columns {
			if n >= from {
				n++
			}
			shifted[n] = v
		}
		t.Conditions[i].Columns = shifted
	}
	for i := range t.Actions {
		shifted := make(map[int]bool, len(t.Actions[i].Columns))
		for n, v := range t.Actions[i].Columns {
			if n >= from {
				n++
			}
			shifted[n] = v
		}
		t.Actions[i].Columns = shifted
	}
}

// isStarCell reports whether a condition cell is the otherwise marker.
func isStarCell(v string) bool {
	return strings.TrimSpace(v) == "*"
}

// cellSnapshot is a deep copy of every condition and action cell, so a column
// mutation that breaks the otherwise rule can be undone whole. The mutators
// that rewrite a column in place have no cheaper undo, and a refused edit must
// leave the table exactly as it was.
type cellSnapshot struct {
	conditions []map[int]string
	actions    []map[int]bool
}

func (t *Table) snapshotCells() cellSnapshot {
	s := cellSnapshot{
		conditions: make([]map[int]string, len(t.Conditions)),
		actions:    make([]map[int]bool, len(t.Actions)),
	}
	for i, c := range t.Conditions {
		cols := make(map[int]string, len(c.Columns))
		for k, v := range c.Columns {
			cols[k] = v
		}
		s.conditions[i] = cols
	}
	for i, a := range t.Actions {
		cols := make(map[int]bool, len(a.Columns))
		for k, v := range a.Columns {
			cols[k] = v
		}
		s.actions[i] = cols
	}
	return s
}

func (t *Table) restoreCells(s cellSnapshot) {
	for i := range t.Conditions {
		if i < len(s.conditions) {
			t.Conditions[i].Columns = s.conditions[i]
		}
	}
	for i := range t.Actions {
		if i < len(s.actions) {
			t.Actions[i].Columns = s.actions[i]
		}
	}
}

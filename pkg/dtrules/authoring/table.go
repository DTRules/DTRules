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
	"fmt"
	"strconv"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// Table is a typed view of a single decision table. All mutations validate EL
// before committing, so invalid expressions are rejected at the API boundary.
type Table struct {
	Name           string
	Policy         string
	Contexts       []Context
	InitialActions []InitialAction
	Conditions     []Condition
	Actions        []Action

	xml     *excel.DecisionTableXML
	symbols map[string]string
	project *Project // nil when constructed outside a project context
}

// Context holds a context entity reference for a decision table.
type Context struct {
	DSL string
}

// InitialAction holds an action that runs before the decision table columns.
type InitialAction struct {
	DSL string
}

// Condition holds one condition row of a decision table.
type Condition struct {
	Number  int
	Comment string
	DSL     string
	Columns map[int]string // col -> "Y" / "N" / "-"
}

// Action holds one action row of a decision table.
type Action struct {
	Number  int
	Comment string
	DSL     string
	Columns map[int]bool // col -> executes on that column?
}

// newTable builds a Table view from an underlying XML struct.
func newTable(x *excel.DecisionTableXML, symbols map[string]string) *Table {
	t := &Table{
		Name:    x.TableName,
		Policy:  x.AttributeFields.Type,
		xml:     x,
		symbols: symbols,
	}
	t.syncFromXML()
	return t
}

// newTableWithProject builds a Table view linked to its owning project (enables Callers).
func newTableWithProject(x *excel.DecisionTableXML, symbols map[string]string, p *Project) *Table {
	t := newTable(x, symbols)
	t.project = p
	return t
}

// syncFromXML refreshes the typed view from the underlying XML model.
func (t *Table) syncFromXML() {
	t.Name = t.xml.TableName
	t.Policy = t.xml.AttributeFields.Type

	t.Contexts = nil
	if t.xml.Contexts != "" {
		t.Contexts = []Context{{DSL: t.xml.Contexts}}
	}

	t.InitialActions = nil
	for _, ia := range t.xml.InitialActions {
		t.InitialActions = append(t.InitialActions, InitialAction{DSL: ia.DSL})
	}

	t.Conditions = nil
	for _, c := range t.xml.Conditions {
		num, _ := strconv.Atoi(c.Number)
		cols := make(map[int]string)
		for _, cv := range c.Columns {
			cols[cv.Number] = cv.Value
		}
		t.Conditions = append(t.Conditions, Condition{
			Number:  num,
			Comment: c.Comment,
			DSL:     c.DSL,
			Columns: cols,
		})
	}

	t.Actions = nil
	for _, a := range t.xml.Actions {
		num, _ := strconv.Atoi(a.Number)
		cols := make(map[int]bool)
		for _, cv := range a.Columns {
			cols[cv.Number] = cv.Value != ""
		}
		t.Actions = append(t.Actions, Action{
			Number:  num,
			Comment: a.Comment,
			DSL:     a.DSL,
			Columns: cols,
		})
	}
}

// syncToXML writes the typed view back into the underlying XML model.
func (t *Table) syncToXML() {
	t.xml.TableName = t.Name
	t.xml.AttributeFields.Type = t.Policy

	if len(t.Contexts) > 0 {
		t.xml.Contexts = t.Contexts[0].DSL
	} else {
		t.xml.Contexts = ""
	}

	t.xml.InitialActions = nil
	for _, ia := range t.InitialActions {
		t.xml.InitialActions = append(t.xml.InitialActions, excel.InitialActionXML{DSL: ia.DSL})
	}

	t.xml.Conditions = nil
	for _, c := range t.Conditions {
		var cols []excel.ColumnValueXML
		for n, v := range c.Columns {
			if v != "" && v != "-" {
				cols = append(cols, excel.ColumnValueXML{Number: n, Value: v})
			}
		}
		t.xml.Conditions = append(t.xml.Conditions, excel.ConditionXML{
			Number:  strconv.Itoa(c.Number),
			Comment: c.Comment,
			DSL:     c.DSL,
			Columns: cols,
		})
	}

	t.xml.Actions = nil
	for _, a := range t.Actions {
		var cols []excel.ColumnValueXML
		for n, ok := range a.Columns {
			if ok {
				cols = append(cols, excel.ColumnValueXML{Number: n, Value: "X"})
			}
		}
		t.xml.Actions = append(t.xml.Actions, excel.ActionXML{
			Number:  strconv.Itoa(a.Number),
			Comment: a.Comment,
			DSL:     a.DSL,
			Columns: cols,
		})
	}
}

// Columns returns the number of rule columns in this table.
func (t *Table) Columns() int {
	max := 0
	for _, c := range t.Conditions {
		for n := range c.Columns {
			if n > max {
				max = n
			}
		}
	}
	for _, a := range t.Actions {
		for n := range a.Columns {
			if n > max {
				max = n
			}
		}
	}
	return max
}

// nextConditionNumber returns 1 + the current max condition number.
func (t *Table) nextConditionNumber() int {
	max := 0
	for _, c := range t.Conditions {
		if c.Number > max {
			max = c.Number
		}
	}
	return max + 1
}

// nextActionNumber returns 1 + the current max action number.
func (t *Table) nextActionNumber() int {
	max := 0
	for _, a := range t.Actions {
		if a.Number > max {
			max = a.Number
		}
	}
	return max + 1
}

// --- Condition mutations ---

// AddCondition validates and appends a condition. If c.Number == 0, it is auto-assigned.
func (t *Table) AddCondition(c Condition) error {
	if _, err := CheckCondition(c.DSL, t.symbols); err != nil {
		return err
	}
	if c.Number == 0 {
		c.Number = t.nextConditionNumber()
	}
	if c.Columns == nil {
		c.Columns = make(map[int]string)
	}
	t.Conditions = append(t.Conditions, c)
	t.syncToXML()
	return nil
}

// UpdateCondition replaces the condition with the given number.
func (t *Table) UpdateCondition(num int, c Condition) error {
	if _, err := CheckCondition(c.DSL, t.symbols); err != nil {
		return err
	}
	for i, existing := range t.Conditions {
		if existing.Number == num {
			c.Number = num
			if c.Columns == nil {
				c.Columns = existing.Columns
			}
			t.Conditions[i] = c
			t.syncToXML()
			return nil
		}
	}
	return fmt.Errorf("condition number %d not found", num)
}

// DeleteCondition removes the condition with the given number.
func (t *Table) DeleteCondition(num int) error {
	for i, c := range t.Conditions {
		if c.Number == num {
			t.Conditions = append(t.Conditions[:i], t.Conditions[i+1:]...)
			t.syncToXML()
			return nil
		}
	}
	return fmt.Errorf("condition number %d not found", num)
}

// --- Action mutations ---

// AddAction validates and appends an action. If a.Number == 0, it is auto-assigned.
func (t *Table) AddAction(a Action) error {
	if _, err := CheckAction(a.DSL, t.symbols); err != nil {
		return err
	}
	if a.Number == 0 {
		a.Number = t.nextActionNumber()
	}
	if a.Columns == nil {
		a.Columns = make(map[int]bool)
	}
	t.Actions = append(t.Actions, a)
	t.syncToXML()
	return nil
}

// UpdateAction replaces the action with the given number.
func (t *Table) UpdateAction(num int, a Action) error {
	if _, err := CheckAction(a.DSL, t.symbols); err != nil {
		return err
	}
	for i, existing := range t.Actions {
		if existing.Number == num {
			a.Number = num
			if a.Columns == nil {
				a.Columns = existing.Columns
			}
			t.Actions[i] = a
			t.syncToXML()
			return nil
		}
	}
	return fmt.Errorf("action number %d not found", num)
}

// DeleteAction removes the action with the given number.
func (t *Table) DeleteAction(num int) error {
	for i, a := range t.Actions {
		if a.Number == num {
			t.Actions = append(t.Actions[:i], t.Actions[i+1:]...)
			t.syncToXML()
			return nil
		}
	}
	return fmt.Errorf("action number %d not found", num)
}

// --- InitialAction mutations ---

// AddInitialAction validates and appends an initial action.
func (t *Table) AddInitialAction(a InitialAction) error {
	if _, err := CheckAction(a.DSL, t.symbols); err != nil {
		return err
	}
	t.InitialActions = append(t.InitialActions, a)
	t.syncToXML()
	return nil
}

// UpdateInitialAction replaces the initial action at idx.
func (t *Table) UpdateInitialAction(idx int, a InitialAction) error {
	if idx < 0 || idx >= len(t.InitialActions) {
		return fmt.Errorf("initial action index %d out of range (len %d)", idx, len(t.InitialActions))
	}
	if _, err := CheckAction(a.DSL, t.symbols); err != nil {
		return err
	}
	t.InitialActions[idx] = a
	t.syncToXML()
	return nil
}

// DeleteInitialAction removes the initial action at idx.
func (t *Table) DeleteInitialAction(idx int) error {
	if idx < 0 || idx >= len(t.InitialActions) {
		return fmt.Errorf("initial action index %d out of range (len %d)", idx, len(t.InitialActions))
	}
	t.InitialActions = append(t.InitialActions[:idx], t.InitialActions[idx+1:]...)
	t.syncToXML()
	return nil
}

// --- Context mutations ---

// AddContext validates and appends a context.
func (t *Table) AddContext(c Context) error {
	if _, err := CheckContext(c.DSL, t.symbols); err != nil {
		return err
	}
	t.Contexts = append(t.Contexts, c)
	t.syncToXML()
	return nil
}

// UpdateContext replaces the context at idx.
func (t *Table) UpdateContext(idx int, c Context) error {
	if idx < 0 || idx >= len(t.Contexts) {
		return fmt.Errorf("context index %d out of range (len %d)", idx, len(t.Contexts))
	}
	if _, err := CheckContext(c.DSL, t.symbols); err != nil {
		return err
	}
	t.Contexts[idx] = c
	t.syncToXML()
	return nil
}

// DeleteContext removes the context at idx.
func (t *Table) DeleteContext(idx int) error {
	if idx < 0 || idx >= len(t.Contexts) {
		return fmt.Errorf("context index %d out of range (len %d)", idx, len(t.Contexts))
	}
	t.Contexts = append(t.Contexts[:idx], t.Contexts[idx+1:]...)
	t.syncToXML()
	return nil
}

// --- Column operations ---

// validColumnValue checks that a condition column value is one of the legal tokens.
func validColumnValue(v string) bool {
	return v == "Y" || v == "N" || v == "-"
}

// AddColumn adds a new rule column. conditions maps condition Number -> "Y"/"N"/"-".
// actions is the list of action Numbers that execute on this column.
func (t *Table) AddColumn(conditions map[int]string, actions []int) error {
	if err := t.validateColumnArgs(conditions, actions); err != nil {
		return err
	}
	col := t.Columns() + 1
	t.applyColumn(col, conditions, actions)
	t.syncToXML()
	return nil
}

// UpdateColumn replaces an existing column.
func (t *Table) UpdateColumn(col int, conditions map[int]string, actions []int) error {
	if col < 1 || col > t.Columns() {
		return fmt.Errorf("column %d out of range (table has %d columns)", col, t.Columns())
	}
	if err := t.validateColumnArgs(conditions, actions); err != nil {
		return err
	}
	// Remove col from all conditions and actions, then re-apply.
	for i := range t.Conditions {
		delete(t.Conditions[i].Columns, col)
	}
	for i := range t.Actions {
		delete(t.Actions[i].Columns, col)
	}
	t.applyColumn(col, conditions, actions)
	t.syncToXML()
	return nil
}

// DeleteColumn removes column col. Columns after it are renumbered downward by 1.
func (t *Table) DeleteColumn(col int) error {
	numCols := t.Columns()
	if col < 1 || col > numCols {
		return fmt.Errorf("column %d out of range (table has %d columns)", col, numCols)
	}
	for i := range t.Conditions {
		delete(t.Conditions[i].Columns, col)
		// Renumber columns > col
		newCols := make(map[int]string)
		for n, v := range t.Conditions[i].Columns {
			if n > col {
				newCols[n-1] = v
			} else {
				newCols[n] = v
			}
		}
		t.Conditions[i].Columns = newCols
	}
	for i := range t.Actions {
		delete(t.Actions[i].Columns, col)
		newCols := make(map[int]bool)
		for n, v := range t.Actions[i].Columns {
			if n > col {
				newCols[n-1] = v
			} else {
				newCols[n] = v
			}
		}
		t.Actions[i].Columns = newCols
	}
	t.syncToXML()
	return nil
}

// validateColumnArgs checks that all referenced condition/action numbers exist and
// that condition values are legal.
func (t *Table) validateColumnArgs(conditions map[int]string, actions []int) error {
	condIdx := make(map[int]bool, len(t.Conditions))
	for _, c := range t.Conditions {
		condIdx[c.Number] = true
	}
	actionIdx := make(map[int]bool, len(t.Actions))
	for _, a := range t.Actions {
		actionIdx[a.Number] = true
	}

	for num, val := range conditions {
		if !condIdx[num] {
			return fmt.Errorf("condition number %d does not exist in table", num)
		}
		if !validColumnValue(val) {
			return fmt.Errorf("invalid column value %q for condition %d: must be Y, N, or -", val, num)
		}
	}
	for _, num := range actions {
		if !actionIdx[num] {
			return fmt.Errorf("action number %d does not exist in table", num)
		}
	}
	return nil
}

// applyColumn writes col into every affected condition and action.
func (t *Table) applyColumn(col int, conditions map[int]string, actions []int) {
	for i, c := range t.Conditions {
		if val, ok := conditions[c.Number]; ok {
			if t.Conditions[i].Columns == nil {
				t.Conditions[i].Columns = make(map[int]string)
			}
			t.Conditions[i].Columns[col] = val
		}
	}
	actionSet := make(map[int]bool, len(actions))
	for _, num := range actions {
		actionSet[num] = true
	}
	for i, a := range t.Actions {
		if t.Actions[i].Columns == nil {
			t.Actions[i].Columns = make(map[int]bool)
		}
		t.Actions[i].Columns[col] = actionSet[a.Number]
	}
}

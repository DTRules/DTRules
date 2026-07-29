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
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// Table is a typed view of a single decision table. All mutations validate EL
// before committing, so invalid expressions are rejected at the API boundary.
type Table struct {
	Name             string
	Number           int // TABLE_NUMBER — load/sheet ordering; 0 means unset
	Policy           string
	Contexts         []Context
	InitialActions   []InitialAction
	Conditions       []Condition
	Actions          []Action
	PolicyStatements []PolicyStatement

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
//
// Postfix is intentionally NOT a field on this struct. Per #817,
// postfix is a compiled artifact of the EL DSL — the authoring view
// only models authoring inputs. On write-out, syncToXML runs the EL
// compiler over `DSL` and the result becomes the `<action_postfix>`
// bytes; whatever postfix was on disk before is overwritten. If the
// DSL is empty, the postfix is also empty.
type Action struct {
	Number  int
	Comment string
	DSL     string
	Columns map[int]bool // col -> executes on that column?
}

// PolicyStatement is the statement a rule column contributes when it fires.
//
// Description is a template, not an EL expression: `{expr}` substitutes the
// runtime value of expr, so
//
//	State {state_config.state_code} has no income tax
//
// reports the actual state. Like every other postfix in the authoring view,
// the compiled form is regenerated from Description on write-out — see
// excel.CompilePolicyStatement — so hand-written statement postfix that has
// drifted from its description is replaced, not preserved (#817).
type PolicyStatement struct {
	Column      int // 1-based rule column
	Description string
}

// HandCodedRows reports rows that carry postfix but no DSL, as
// "<kind> <number>" strings.
//
// These rows are the one thing a recompile destroys. syncToXML regenerates
// every postfix from its DSL (#817), so a row whose only content IS postfix
// comes back empty — its logic deleted, silently, by an operation that looks
// like normalization. Four sample projects lost rows that way before anyone
// noticed, because the emptied rows were not covered by any scenario.
//
// Author the EL first (read the stored postfix, write the DSL that compiles
// to it, check they match), then recompile.
func (t *Table) HandCodedRows() []string {
	var out []string
	for i, c := range t.xml.Contexts.Details {
		if strings.TrimSpace(c.Postfix) != "" && strings.TrimSpace(c.DSL) == "" {
			out = append(out, fmt.Sprintf("context %d", i+1))
		}
	}
	for i, ia := range t.xml.InitialActions {
		if strings.TrimSpace(ia.EffectivePostfix()) != "" && strings.TrimSpace(ia.EffectiveDSL()) == "" {
			out = append(out, fmt.Sprintf("initial action %d", i+1))
		}
	}
	for _, c := range t.xml.Conditions {
		if strings.TrimSpace(c.Postfix) != "" && strings.TrimSpace(c.DSL) == "" {
			out = append(out, "condition "+c.Number)
		}
	}
	for _, a := range t.xml.Actions {
		if strings.TrimSpace(a.Postfix) != "" && strings.TrimSpace(a.DSL) == "" {
			out = append(out, "action "+a.Number)
		}
	}
	return out
}

// newTable builds a Table view from an underlying XML struct.
func newTable(x *excel.DecisionTableXML, symbols map[string]string) *Table {
	t := &Table{
		Name:    x.TableName,
		Policy:  x.AttributeFields.EffectiveType(),
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
//
// The typed view exposes only DSL / comment / column fields; the XML model
// (postfix, legacy alternate tags, context_entity directives, context number/
// description, etc.) is preserved on the backing `t.xml` struct so syncToXML
// can carry it back through write-out.
func (t *Table) syncFromXML() {
	t.Name = t.xml.TableName
	t.Policy = t.xml.AttributeFields.EffectiveType()
	t.Number, _ = strconv.Atoi(strings.TrimSpace(t.xml.AttributeFields.TableNumber))

	t.Contexts = nil
	// Surface every <context_details> entry to the typed view, even ones
	// whose DSL is empty — they're targets for `update-context` patches
	// during EL authoring of legacy postfix-only contexts.
	for _, d := range t.xml.Contexts.Details {
		t.Contexts = append(t.Contexts, Context{DSL: strings.TrimSpace(d.DSL)})
	}

	t.InitialActions = nil
	for _, ia := range t.xml.InitialActions {
		t.InitialActions = append(t.InitialActions, InitialAction{DSL: ia.EffectiveDSL()})
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

	t.PolicyStatements = nil
	for _, ps := range t.xml.PolicyStatements {
		col, _ := strconv.Atoi(strings.TrimSpace(ps.Column))
		t.PolicyStatements = append(t.PolicyStatements, PolicyStatement{
			Column:      col,
			Description: ps.Description,
		})
	}
	sort.Slice(t.PolicyStatements, func(i, j int) bool {
		return t.PolicyStatements[i].Column < t.PolicyStatements[j].Column
	})
}

// syncToXML writes the typed view back into the underlying XML model,
// preserving every field the typed view doesn't expose (legacy alternate
// tag content, context_entity directives, context number/comment/
// description, …) by matching against the original XML records.
//
// Postfix is NOT preserved from the original XML — it is regenerated
// from each typed entry's current DSL via the EL compiler (#817).
// An empty DSL produces an empty postfix; a malformed DSL produces an
// empty postfix too (mutations already validated through CheckXxx, so
// in practice this is unreachable from authoring callers).
//
// Matching rules for non-postfix carry-through:
//   - Conditions / Actions: by `Number`. Carries Number, Comment from
//     original when present.
//   - InitialActions: by position. Carries Comment, ActionDSL from
//     original. ActionPostfix is regenerated from the carried ActionDSL.
//   - Contexts: position-by-position over the Details slice.
func (t *Table) syncToXML() {
	// One compiler for the whole table, so a local declared in a context row
	// is in scope for the conditions and actions beneath it (#965). Rows are
	// compiled below in table order — contexts, initial actions, conditions,
	// actions — which is the order the slots have to be declared in.
	tc := newTableCompiler(t.symbols)

	t.xml.TableName = t.Name
	// Write the canonical spelling and clear the legacy ones, so a table
	// cannot end up carrying two different types.
	t.xml.AttributeFields.Type = t.Policy
	t.xml.AttributeFields.TypeUppercase = ""
	t.xml.AttributeFields.TypeLowercase = ""
	// Only write a number when one is set, so an unspecified number keeps the
	// value AddTable auto-assigned rather than clobbering it with 0.
	if t.Number > 0 {
		t.xml.AttributeFields.TableNumber = strconv.Itoa(t.Number)
	}

	// Contexts.
	origDetails := t.xml.Contexts.Details
	newDetails := make([]excel.ContextDetailXML, 0, len(t.Contexts))
	for i, c := range t.Contexts {
		entry := excel.ContextDetailXML{Number: i + 1, DSL: c.DSL}
		if i < len(origDetails) {
			entry.Number = origDetails[i].Number
			if entry.Number == 0 {
				entry.Number = i + 1
			}
			entry.Comment = origDetails[i].Comment
			entry.Name = origDetails[i].Name
			entry.Description = origDetails[i].Description
		}
		entry.Postfix = tc.compile(c.DSL, "context")
		newDetails = append(newDetails, entry)
	}
	t.xml.Contexts.Details = newDetails

	// Initial actions.
	origInits := t.xml.InitialActions
	newInits := make([]excel.InitialActionXML, 0, len(t.InitialActions))
	for i, ia := range t.InitialActions {
		entry := excel.InitialActionXML{DSL: ia.DSL}
		if i < len(origInits) {
			entry.Comment = origInits[i].Comment
			entry.ActionDSL = origInits[i].ActionDSL
			entry.ActionPostfix = tc.compile(origInits[i].ActionDSL, "action")
		}
		entry.Postfix = tc.compile(ia.DSL, "action")
		newInits = append(newInits, entry)
	}
	t.xml.InitialActions = newInits

	// Conditions.
	origConds := map[int]excel.ConditionXML{}
	for _, c := range t.xml.Conditions {
		n, _ := strconv.Atoi(c.Number)
		origConds[n] = c
	}
	t.xml.Conditions = nil
	for _, c := range t.Conditions {
		var cols []excel.ColumnValueXML
		for n, v := range c.Columns {
			if v != "" {
				cols = append(cols, excel.ColumnValueXML{Number: n, Value: v})
			}
		}
		// Go map iteration is randomized, so writing columns in map order
		// reshuffled them on every save and every authoring write produced a
		// spurious diff. Emit them in column order.
		sortColumns(cols)
		entry := excel.ConditionXML{
			Number:  strconv.Itoa(c.Number),
			Comment: c.Comment,
			DSL:     c.DSL,
			Columns: cols,
		}
		entry.Postfix = tc.compile(c.DSL, "condition")
		t.xml.Conditions = append(t.xml.Conditions, entry)
	}

	// Actions.
	t.xml.Actions = nil
	for _, a := range t.Actions {
		var cols []excel.ColumnValueXML
		for n, ok := range a.Columns {
			if ok {
				cols = append(cols, excel.ColumnValueXML{Number: n, Value: "X"})
			}
		}
		sortColumns(cols)
		entry := excel.ActionXML{
			Number:  strconv.Itoa(a.Number),
			Comment: a.Comment,
			DSL:     a.DSL,
			Columns: cols,
		}
		entry.Postfix = tc.compile(a.DSL, "action")
		t.xml.Actions = append(t.xml.Actions, entry)
	}

	// Policy statements. Their postfix is compiled from the description
	// template for the same reason every other postfix is (#817): a stored
	// form that has drifted from the text it claims to render is a lie the
	// next build would erase anyway.
	t.xml.PolicyStatements = nil
	for _, ps := range t.PolicyStatements {
		t.xml.PolicyStatements = append(t.xml.PolicyStatements, excel.PolicyStatementXML{
			Column:      strconv.Itoa(ps.Column),
			Description: ps.Description,
			Postfix:     excel.CompilePolicyStatement(ps.Description),
		})
	}
}

// sortColumns orders column entries by column number so write-out is
// deterministic regardless of Go's randomized map iteration.
func sortColumns(cols []excel.ColumnValueXML) {
	sort.Slice(cols, func(i, j int) bool { return cols[i].Number < cols[j].Number })
}

// SetPolicyStatement sets the statement for a rule column, replacing any
// existing one. Statements stay sorted by column.
func (t *Table) SetPolicyStatement(column int, description string) error {
	if column < 1 {
		return fmt.Errorf("policy statement column must be >= 1, got %d", column)
	}
	for i := range t.PolicyStatements {
		if t.PolicyStatements[i].Column == column {
			t.PolicyStatements[i].Description = description
			t.syncToXML()
			return nil
		}
	}
	t.PolicyStatements = append(t.PolicyStatements, PolicyStatement{
		Column:      column,
		Description: description,
	})
	sort.Slice(t.PolicyStatements, func(i, j int) bool {
		return t.PolicyStatements[i].Column < t.PolicyStatements[j].Column
	})
	t.syncToXML()
	return nil
}

// DeletePolicyStatement removes the statement for a rule column.
func (t *Table) DeletePolicyStatement(column int) error {
	for i := range t.PolicyStatements {
		if t.PolicyStatements[i].Column == column {
			t.PolicyStatements = append(t.PolicyStatements[:i], t.PolicyStatements[i+1:]...)
			t.syncToXML()
			return nil
		}
	}
	return fmt.Errorf("no policy statement for column %d", column)
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

// SetNumber sets the table's TABLE_NUMBER, which controls load and Excel-sheet
// ordering. Authors set it explicitly to insert a table between two existing
// ones (the auto-assigned numbers leave gaps of 10 for exactly this) or to
// reorganize. When the table belongs to a project, the number is validated:
// it must fall inside the file's declared range (if any) and be unique.
func (t *Table) SetNumber(n int) error {
	if t.project != nil {
		if err := t.project.validateNumberFor(t.Name, n); err != nil {
			return err
		}
	}
	t.Number = n
	t.syncToXML()
	return nil
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
//
// Postfix is regenerated from DSL on every syncToXML — there is no
// way to carry a non-DSL-derived postfix through this path (#817).
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

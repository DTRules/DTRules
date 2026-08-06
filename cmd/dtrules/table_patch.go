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

// Patch ops are a flat discriminated union keyed on "op". Each op carries
// the arguments it needs as optional siblings; fields for other ops stay at
// their zero value. This keeps the JSON shape trivial for AI callers and
// cheap to unmarshal.

import (
	"fmt"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// tablePatch is the wire shape for `dtrules table patch`.
type tablePatch struct {
	Op string `json:"op"`

	// shared
	Column          int    `json:"column,omitempty"`
	ConditionNumber int    `json:"condition_number,omitempty"`
	ActionNumber    int    `json:"action_number,omitempty"`
	Index           int    `json:"index,omitempty"`
	Value           string `json:"value,omitempty"`
	On              bool   `json:"on,omitempty"`
	Name            string `json:"name,omitempty"`
	Number          int    `json:"number,omitempty"`
	Policy          string `json:"policy,omitempty"`
	DSL             string `json:"dsl,omitempty"`
	Comment         string `json:"comment,omitempty"`
	Description     string `json:"description,omitempty"`
	File            string `json:"file,omitempty"`
	Range           string `json:"range,omitempty"`
	Reason          string `json:"reason,omitempty"`

	// column ops
	Conditions map[string]string `json:"conditions,omitempty"`
	Actions    []int             `json:"actions,omitempty"`

	// add/update-condition whole-row
	Columns map[string]string `json:"columns,omitempty"` // for condition
	// add/update-action whole-row
	ActionColumns map[string]bool `json:"action_columns,omitempty"`
}

func (p *tablePatch) hint() string {
	return "see `dtrules table schema --patch` for the argument shape of each op"
}

// apply runs the requested patch op against t (proj is needed for the
// project-scoped ops set-file and set-range).
func (p *tablePatch) apply(proj *authoring.Project, t *authoring.Table) error {
	switch p.Op {
	case "set-file":
		if p.File == "" {
			return fmt.Errorf("set-file requires \"file\"")
		}
		if strings.TrimSpace(p.Reason) == "" {
			return fmt.Errorf("set-file requires \"reason\"")
		}
		if err := ensureFile(proj, p.File, p.Range, p.Reason); err != nil {
			return err
		}
		return proj.MoveTable(t.Name, p.File, p.Reason)

	case "set-range":
		file := p.File
		if file == "" {
			file = proj.FileOf(t.Name)
		}
		lo, hi, err := parseRange(p.Range)
		if err != nil {
			return err
		}
		return proj.SetFileRange(file, lo, hi, p.Reason)

	case "set-name":
		if p.Name == "" {
			return fmt.Errorf("set-name requires a non-empty \"name\"")
		}
		// Through the project, not the view: a bare `t.Name = ...` mutates a
		// throwaway view and never reaches the XML — set-name used to be a
		// silent no-op that reported "patched".
		if err := proj.RenameTable(t.Name, p.Name); err != nil {
			return err
		}
		t.Name = p.Name
		return nil

	case "set-policy":
		t.Policy = p.Policy
		return nil

	case "set-number":
		if p.Number < 1 {
			return fmt.Errorf("set-number requires \"number\" >= 1")
		}
		return t.SetNumber(p.Number)

	case "set-condition-cell":
		return setConditionCell(t, p.ConditionNumber, p.Column, p.Value)

	case "set-action-cell":
		return setActionCell(t, p.ActionNumber, p.Column, p.On)

	case "add-column":
		conds, err := stringMapToIntString(p.Conditions)
		if err != nil {
			return err
		}
		return t.AddColumn(conds, p.Actions)

	case "update-column":
		if p.Column < 1 {
			return fmt.Errorf("update-column requires column >= 1")
		}
		conds, err := stringMapToIntString(p.Conditions)
		if err != nil {
			return err
		}
		return t.UpdateColumn(p.Column, conds, p.Actions)

	case "delete-column":
		if p.Column < 1 {
			return fmt.Errorf("delete-column requires column >= 1")
		}
		return t.DeleteColumn(p.Column)

	case "add-condition":
		cols, err := stringMapToIntString(p.Columns)
		if err != nil {
			return err
		}
		return t.AddCondition(authoring.Condition{
			Number:  p.ConditionNumber,
			Comment: p.Comment,
			DSL:     p.DSL,
			Columns: cols,
		})

	case "update-condition":
		if p.ConditionNumber == 0 {
			return fmt.Errorf("update-condition requires condition_number")
		}
		cols, err := stringMapToIntString(p.Columns)
		if err != nil {
			return err
		}
		return t.UpdateCondition(p.ConditionNumber, authoring.Condition{
			Comment: p.Comment,
			DSL:     p.DSL,
			Columns: cols,
		})

	case "update-condition-dsl":
		if p.ConditionNumber == 0 {
			return fmt.Errorf("update-condition-dsl requires condition_number")
		}
		existing, err := findCondition(t, p.ConditionNumber)
		if err != nil {
			return err
		}
		return t.UpdateCondition(p.ConditionNumber, authoring.Condition{
			Comment: existing.Comment,
			DSL:     p.DSL,
			Columns: existing.Columns,
		})

	case "delete-condition":
		if p.ConditionNumber == 0 {
			return fmt.Errorf("delete-condition requires condition_number")
		}
		return t.DeleteCondition(p.ConditionNumber)

	case "add-action":
		cols, err := stringMapToIntBool(p.ActionColumns)
		if err != nil {
			return err
		}
		return t.AddAction(authoring.Action{
			Number:  p.ActionNumber,
			Comment: p.Comment,
			DSL:     p.DSL,
			Columns: cols,
		})

	case "update-action":
		if p.ActionNumber == 0 {
			return fmt.Errorf("update-action requires action_number")
		}
		cols, err := stringMapToIntBool(p.ActionColumns)
		if err != nil {
			return err
		}
		return t.UpdateAction(p.ActionNumber, authoring.Action{
			Comment: p.Comment,
			DSL:     p.DSL,
			Columns: cols,
		})

	case "update-action-dsl":
		if p.ActionNumber == 0 {
			return fmt.Errorf("update-action-dsl requires action_number")
		}
		existing, err := findAction(t, p.ActionNumber)
		if err != nil {
			return err
		}
		return t.UpdateAction(p.ActionNumber, authoring.Action{
			Comment: existing.Comment,
			DSL:     p.DSL,
			Columns: existing.Columns,
		})

	case "delete-action":
		if p.ActionNumber == 0 {
			return fmt.Errorf("delete-action requires action_number")
		}
		return t.DeleteAction(p.ActionNumber)

	case "add-initial-action":
		return t.AddInitialAction(authoring.InitialAction{DSL: p.DSL})

	case "update-initial-action":
		return t.UpdateInitialAction(p.Index, authoring.InitialAction{DSL: p.DSL})

	case "delete-initial-action":
		return t.DeleteInitialAction(p.Index)

	case "add-context":
		return t.AddContext(authoring.Context{DSL: p.DSL})

	case "update-context":
		return t.UpdateContext(p.Index, authoring.Context{DSL: p.DSL})

	case "delete-context":
		return t.DeleteContext(p.Index)

	case "set-policy-statement":
		if p.Column < 1 {
			return fmt.Errorf("set-policy-statement requires \"column\" >= 1")
		}
		return t.SetPolicyStatement(p.Column, p.Description)

	case "delete-policy-statement":
		if p.Column < 1 {
			return fmt.Errorf("delete-policy-statement requires \"column\" >= 1")
		}
		return t.DeletePolicyStatement(p.Column)

	default:
		return fmt.Errorf("unknown op %q (see `dtrules table schema --patch`)", p.Op)
	}
}

// setConditionCell uses UpdateCondition to flip a single cell — that exercises
// the SDK's EL validation path, which is what we want.
func setConditionCell(t *authoring.Table, num, col int, value string) error {
	if num == 0 {
		return fmt.Errorf("set-condition-cell requires condition_number")
	}
	if col < 1 {
		return fmt.Errorf("set-condition-cell requires column >= 1")
	}
	if value != "Y" && value != "N" && value != "-" {
		return fmt.Errorf("value must be one of Y, N, -")
	}
	existing, err := findCondition(t, num)
	if err != nil {
		return err
	}
	// Clone the column map so mutation stays local until UpdateCondition commits.
	newCols := make(map[int]string, len(existing.Columns)+1)
	for k, v := range existing.Columns {
		newCols[k] = v
	}
	newCols[col] = value
	return t.UpdateCondition(num, authoring.Condition{
		Comment: existing.Comment,
		DSL:     existing.DSL,
		Columns: newCols,
	})
}

func setActionCell(t *authoring.Table, num, col int, on bool) error {
	if num == 0 {
		return fmt.Errorf("set-action-cell requires action_number")
	}
	if col < 1 {
		return fmt.Errorf("set-action-cell requires column >= 1")
	}
	existing, err := findAction(t, num)
	if err != nil {
		return err
	}
	newCols := make(map[int]bool, len(existing.Columns)+1)
	for k, v := range existing.Columns {
		newCols[k] = v
	}
	newCols[col] = on
	return t.UpdateAction(num, authoring.Action{
		Comment: existing.Comment,
		DSL:     existing.DSL,
		Columns: newCols,
	})
}

func findCondition(t *authoring.Table, num int) (authoring.Condition, error) {
	for _, c := range t.Conditions {
		if c.Number == num {
			return c, nil
		}
	}
	return authoring.Condition{}, fmt.Errorf("condition number %d not found", num)
}

func findAction(t *authoring.Table, num int) (authoring.Action, error) {
	for _, a := range t.Actions {
		if a.Number == num {
			return a, nil
		}
	}
	return authoring.Action{}, fmt.Errorf("action number %d not found", num)
}

// --- EDD patch ---

// eddPatchOp is the wire shape for `dtrules edd patch`.
type eddPatchOp struct {
	Op string `json:"op"`

	Entity  string         `json:"entity,omitempty"`
	Field   *AttributeJSON `json:"field,omitempty"`
	Name    string         `json:"name,omitempty"`
	Comment string         `json:"comment,omitempty"`
}

func (p *eddPatchOp) hint() string {
	return "see `dtrules edd schema --patch` for the argument shape of each op"
}

func (p *eddPatchOp) apply(e *authoring.EDD) error {
	switch p.Op {
	case "add-entity":
		if p.Entity == "" {
			return fmt.Errorf("add-entity requires \"entity\"")
		}
		_, err := e.AddEntity(p.Entity)
		return err

	case "delete-entity":
		if p.Entity == "" {
			return fmt.Errorf("delete-entity requires \"entity\"")
		}
		return e.DeleteEntity(p.Entity)

	case "add-field":
		if p.Entity == "" || p.Field == nil {
			return fmt.Errorf("add-field requires \"entity\" and \"field\"")
		}
		ent := e.Entity(p.Entity)
		if ent == nil {
			return fmt.Errorf("entity %q not found", p.Entity)
		}
		return ent.AddAttribute(attributeFromJSON(*p.Field))

	case "update-field":
		if p.Entity == "" || p.Field == nil || p.Field.Name == "" {
			return fmt.Errorf("update-field requires \"entity\" and \"field.name\"")
		}
		ent := e.Entity(p.Entity)
		if ent == nil {
			return fmt.Errorf("entity %q not found", p.Entity)
		}
		return ent.UpdateAttribute(p.Field.Name, attributeFromJSON(*p.Field))

	case "delete-field":
		if p.Entity == "" || p.Name == "" {
			return fmt.Errorf("delete-field requires \"entity\" and \"name\"")
		}
		ent := e.Entity(p.Entity)
		if ent == nil {
			return fmt.Errorf("entity %q not found", p.Entity)
		}
		return ent.DeleteAttribute(p.Name)

	case "set-comment":
		// Comment on a specific attribute; preserves all other fields.
		if p.Entity == "" || p.Name == "" {
			return fmt.Errorf("set-comment requires \"entity\" and \"name\"")
		}
		ent := e.Entity(p.Entity)
		if ent == nil {
			return fmt.Errorf("entity %q not found", p.Entity)
		}
		return ent.UpdateAttribute(p.Name, authoring.Attribute{Comment: p.Comment})

	default:
		return fmt.Errorf("unknown op %q (see `dtrules edd schema --patch`)", p.Op)
	}
}

func attributeFromJSON(a AttributeJSON) authoring.Attribute {
	attr := authoring.Attribute{
		Name:            a.Name,
		Type:            a.Type,
		Subtype:         a.Subtype,
		Default:         a.Default,
		Access:          a.Access,
		Input:           a.Input,
		Comment:         a.Comment,
		Collect:         a.Collect,
		QuestionText:    a.QuestionText,
		QuestionType:    a.QuestionType,
		QuestionRefLow:  a.QuestionRefLow,
		QuestionRefHigh: a.QuestionRefHigh,
		QuestionUnits:   a.QuestionUnits,
	}
	for _, o := range a.Options {
		attr.Options = append(attr.Options, authoring.Option{Value: o.Value, Label: o.Label})
	}
	return attr
}

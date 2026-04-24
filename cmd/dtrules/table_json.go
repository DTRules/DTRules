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

// DTO layer for the JSON table/EDD API.
//
// The live authoring SDK structs hold unexported fields (xml pointers,
// symbol tables, project back-references) — those must not leak into JSON.
// The types in this file are the stable, serialization-friendly shape that
// `dtrules table get` emits and `dtrules table put` consumes.

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// TableJSON is the JSON projection of a decision table.
type TableJSON struct {
	Name           string              `json:"name"`
	Policy         string              `json:"policy,omitempty"`
	Contexts       []ContextJSON       `json:"contexts,omitempty"`
	InitialActions []InitialActionJSON `json:"initial_actions,omitempty"`
	Conditions     []ConditionJSON     `json:"conditions,omitempty"`
	Actions        []ActionJSON        `json:"actions,omitempty"`
}

// ContextJSON is a single context statement for a table.
type ContextJSON struct {
	DSL string `json:"dsl"`
}

// InitialActionJSON is a single initial-action statement.
type InitialActionJSON struct {
	DSL string `json:"dsl"`
}

// ConditionJSON mirrors authoring.Condition but uses string-keyed column maps
// to survive JSON round-tripping (JSON object keys are always strings).
type ConditionJSON struct {
	Number  int               `json:"number"`
	Comment string            `json:"comment,omitempty"`
	DSL     string            `json:"dsl"`
	Columns map[string]string `json:"columns,omitempty"`
}

// ActionJSON mirrors authoring.Action. Column bool is emitted literally.
type ActionJSON struct {
	Number  int             `json:"number"`
	Comment string          `json:"comment,omitempty"`
	DSL     string          `json:"dsl"`
	Columns map[string]bool `json:"columns,omitempty"`
}

// EDDJSON is the JSON projection of an EDD (entity data dictionary).
type EDDJSON struct {
	Entities []EntityJSON `json:"entities"`
}

// EntityJSON holds the attribute list for one entity.
type EntityJSON struct {
	Name   string          `json:"name"`
	Fields []AttributeJSON `json:"fields,omitempty"`
}

// AttributeJSON mirrors authoring.Attribute.
type AttributeJSON struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
	Default string `json:"default,omitempty"`
	Access  string `json:"access,omitempty"`
	Input   string `json:"input,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// tableToJSON converts an authoring.Table into its JSON-safe projection.
func tableToJSON(t *authoring.Table) TableJSON {
	out := TableJSON{
		Name:   t.Name,
		Policy: t.Policy,
	}
	for _, c := range t.Contexts {
		out.Contexts = append(out.Contexts, ContextJSON{DSL: c.DSL})
	}
	for _, ia := range t.InitialActions {
		out.InitialActions = append(out.InitialActions, InitialActionJSON{DSL: ia.DSL})
	}
	for _, c := range t.Conditions {
		out.Conditions = append(out.Conditions, ConditionJSON{
			Number:  c.Number,
			Comment: c.Comment,
			DSL:     c.DSL,
			Columns: intStringMapToString(c.Columns),
		})
	}
	for _, a := range t.Actions {
		out.Actions = append(out.Actions, ActionJSON{
			Number:  a.Number,
			Comment: a.Comment,
			DSL:     a.DSL,
			Columns: intBoolMapToString(a.Columns),
		})
	}
	return out
}

// eddToJSON projects an authoring.EDD to its JSON shape.
func eddToJSON(e *authoring.EDD) EDDJSON {
	out := EDDJSON{}
	for _, ent := range e.Entities() {
		ej := EntityJSON{Name: ent.Name}
		for _, a := range ent.Attributes {
			ej.Fields = append(ej.Fields, AttributeJSON{
				Name:    a.Name,
				Type:    a.Type,
				Subtype: a.Subtype,
				Default: a.Default,
				Access:  a.Access,
				Input:   a.Input,
				Comment: a.Comment,
			})
		}
		out.Entities = append(out.Entities, ej)
	}
	return out
}

// ApplyTo rebuilds the given authoring.Table in place to match this TableJSON.
// Existing conditions/actions/contexts are cleared; new ones are added in the
// order they appear. Every EL expression is compiled via the SDK's existing
// validation, so invalid DSL fails here before Save is called.
func (tj *TableJSON) ApplyTo(t *authoring.Table) error {
	t.Name = tj.Name
	t.Policy = tj.Policy

	// Clear everything that we're about to rewrite, working from the back
	// so we don't walk off the live slice as the SDK mutates it.
	for i := len(t.Contexts) - 1; i >= 0; i-- {
		if err := t.DeleteContext(i); err != nil {
			return err
		}
	}
	for i := len(t.InitialActions) - 1; i >= 0; i-- {
		if err := t.DeleteInitialAction(i); err != nil {
			return err
		}
	}
	for len(t.Conditions) > 0 {
		num := t.Conditions[len(t.Conditions)-1].Number
		if err := t.DeleteCondition(num); err != nil {
			return err
		}
	}
	for len(t.Actions) > 0 {
		num := t.Actions[len(t.Actions)-1].Number
		if err := t.DeleteAction(num); err != nil {
			return err
		}
	}

	for i, c := range tj.Contexts {
		if err := t.AddContext(authoring.Context{DSL: c.DSL}); err != nil {
			return fmt.Errorf("contexts[%d]: %w", i, err)
		}
	}
	for i, ia := range tj.InitialActions {
		if err := t.AddInitialAction(authoring.InitialAction{DSL: ia.DSL}); err != nil {
			return fmt.Errorf("initial_actions[%d]: %w", i, err)
		}
	}
	for i, c := range tj.Conditions {
		cols, err := stringMapToIntString(c.Columns)
		if err != nil {
			return fmt.Errorf("conditions[%d].columns: %w", i, err)
		}
		if err := t.AddCondition(authoring.Condition{
			Number:  c.Number,
			Comment: c.Comment,
			DSL:     c.DSL,
			Columns: cols,
		}); err != nil {
			return fmt.Errorf("conditions[%d]: %w", i, err)
		}
	}
	for i, a := range tj.Actions {
		cols, err := stringMapToIntBool(a.Columns)
		if err != nil {
			return fmt.Errorf("actions[%d].columns: %w", i, err)
		}
		if err := t.AddAction(authoring.Action{
			Number:  a.Number,
			Comment: a.Comment,
			DSL:     a.DSL,
			Columns: cols,
		}); err != nil {
			return fmt.Errorf("actions[%d]: %w", i, err)
		}
	}
	return nil
}

// ApplyTo rebuilds the EDD to match this EDDJSON. Entities not present in the
// JSON are removed; entities present are re-added with their fields.
func (ej *EDDJSON) ApplyTo(e *authoring.EDD) error {
	existing := map[string]bool{}
	for _, ent := range e.Entities() {
		existing[ent.Name] = true
	}
	wanted := map[string]bool{}
	for _, ent := range ej.Entities {
		wanted[ent.Name] = true
	}
	for name := range existing {
		if !wanted[name] {
			if err := e.DeleteEntity(name); err != nil {
				return fmt.Errorf("delete entity %q: %w", name, err)
			}
		}
	}
	for _, ent := range ej.Entities {
		cur := e.Entity(ent.Name)
		if cur == nil {
			newEnt, err := e.AddEntity(ent.Name)
			if err != nil {
				return fmt.Errorf("add entity %q: %w", ent.Name, err)
			}
			cur = newEnt
		} else {
			// Clear existing attributes so we can rewrite them in the requested order.
			for _, a := range cur.Attributes {
				if err := cur.DeleteAttribute(a.Name); err != nil {
					return fmt.Errorf("entity %q: %w", ent.Name, err)
				}
			}
		}
		for _, f := range ent.Fields {
			if err := cur.AddAttribute(authoring.Attribute{
				Name:    f.Name,
				Type:    f.Type,
				Subtype: f.Subtype,
				Default: f.Default,
				Access:  f.Access,
				Input:   f.Input,
				Comment: f.Comment,
			}); err != nil {
				return fmt.Errorf("entity %q field %q: %w", ent.Name, f.Name, err)
			}
		}
	}
	return nil
}

// intStringMapToString copies an int->string map to a string-keyed one,
// sorting keys for stable JSON output.
func intStringMapToString(in map[int]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	keys := make([]int, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		out[strconv.Itoa(k)] = in[k]
	}
	return out
}

func intBoolMapToString(in map[int]bool) map[string]bool {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]bool, len(in))
	keys := make([]int, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		out[strconv.Itoa(k)] = in[k]
	}
	return out
}

func stringMapToIntString(in map[string]string) (map[int]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[int]string, len(in))
	for k, v := range in {
		n, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("column key %q is not an integer", k)
		}
		out[n] = v
	}
	return out, nil
}

func stringMapToIntBool(in map[string]bool) (map[int]bool, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[int]bool, len(in))
	for k, v := range in {
		n, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("column key %q is not an integer", k)
		}
		out[n] = v
	}
	return out, nil
}

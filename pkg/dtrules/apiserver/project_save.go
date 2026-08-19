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

package apiserver

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// saveViaProject is the editor's save, collapsed onto authoring.Project
// (#1084). The apiserver used to carry its own model→XML serializers
// (saveEDDFile / saveDTFile); two implementations of "turn an edited rule
// set into XML" drift, and #928/#804 were both that drift. Now the editor's
// in-memory model is reconciled onto a freshly-loaded Project and saved
// through the same funnel `dtrules table put` uses — guard, DSL recompile,
// XML write, Excel refresh, and the authoring-notes journal, learned once.
//
// Reconcile order is what makes a failed save atomic: every table mutation
// validates and compiles in memory first (authoring's mutations refuse DSL
// that does not compile), so a compile error aborts before anything touches
// disk — strictly better than the old per-file sequential writes.
func (s *Server) saveViaProject() (saved []string, err error) {
	modifiedEDD := []string{}
	modifiedDT := []string{}
	for _, f := range s.eddFiles {
		if s.modified[f] {
			modifiedEDD = append(modifiedEDD, f)
		}
	}
	for _, f := range s.dtFiles {
		if s.modified[f] {
			modifiedDT = append(modifiedDT, f)
		}
	}
	if len(modifiedEDD)+len(modifiedDT) == 0 {
		return nil, nil
	}

	p, err := authoring.OpenProject(s.projectPath)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}

	// Tables first: their mutations compile DSL and can refuse, and nothing
	// has been written yet when they do.
	if err := s.reconcileTables(p); err != nil {
		return nil, err
	}

	// EDD files. Project holds one EDD at a time; multi-EDD projects select
	// each in turn. All but the last are written through SaveEDD (guarded,
	// Excel-refreshing); the last stays loaded so Project.Save covers it
	// together with the tables in one operation.
	for i, rel := range modifiedEDD {
		if err := p.UseEDDFile(rel); err != nil {
			return nil, fmt.Errorf("select EDD %s: %w", rel, err)
		}
		if err := s.reconcileEDD(p, rel); err != nil {
			return nil, err
		}
		if i < len(modifiedEDD)-1 {
			if err := p.SaveEDD(); err != nil {
				return nil, fmt.Errorf("save %s: %w", rel, err)
			}
		}
	}

	if err := p.Save(); err != nil {
		return nil, err
	}
	return append(modifiedEDD, modifiedDT...), nil
}

// reconcileTables applies the editor model's table state onto the Project:
// tables the editor deleted go, tables it created come, and every kept
// table's editable fields — policy, comments, number, and the condition and
// action rows — are written through the authoring mutations, which compile
// the DSL as they go. Contexts, initial actions, and policy statements are
// read-only in the editor and left exactly as loaded.
func (s *Server) reconcileTables(p *authoring.Project) error {
	keep := make(map[string]*DecisionTableData, len(s.tables))
	for _, t := range s.tables {
		keep[t.TableName] = t
	}
	for _, name := range p.Tables() {
		if keep[name] == nil {
			if err := p.DeleteTable(name, "deleted in the browser editor"); err != nil {
				return fmt.Errorf("delete table %s: %w", name, err)
			}
		}
	}

	for _, td := range s.tables {
		t := p.Table(td.TableName)
		if t == nil {
			file := filepath.Base(td.Source)
			created, err := p.AddTable(td.TableName, file, "created in the browser editor")
			if err != nil {
				return fmt.Errorf("create table %s: %w", td.TableName, err)
			}
			t = created
		}
		if err := applyTableData(t, td); err != nil {
			return &compileError{Table: td.TableName, Err: err}
		}
	}
	return nil
}

// applyTableData writes one editor table onto its authoring view.
func applyTableData(t *authoring.Table, td *DecisionTableData) error {
	t.Policy = td.Type
	t.Comments = td.Comments
	if n, err := strconv.Atoi(strings.TrimSpace(td.TableNumber)); err == nil && n > 0 && n != t.Number {
		if err := t.SetNumber(n); err != nil {
			return fmt.Errorf("table number: %w", err)
		}
	}

	// Conditions, keyed by row number.
	haveCond := map[int]bool{}
	for _, c := range t.Conditions {
		haveCond[c.Number] = true
	}
	wantCond := map[int]bool{}
	for _, r := range td.Conditions {
		wantCond[r.Number] = true
		c := authoring.Condition{
			Number:  r.Number,
			Comment: r.Comment,
			DSL:     r.Description,
			Columns: condColumns(r.Columns),
		}
		var err error
		if haveCond[r.Number] {
			err = t.UpdateCondition(r.Number, c)
		} else {
			err = t.AddCondition(c)
		}
		if err != nil {
			return fmt.Errorf("condition %d: %w", r.Number, err)
		}
	}
	for _, c := range append([]authoring.Condition(nil), t.Conditions...) {
		if !wantCond[c.Number] {
			if err := t.DeleteCondition(c.Number); err != nil {
				return fmt.Errorf("delete condition %d: %w", c.Number, err)
			}
		}
	}

	// Actions, keyed by row number.
	haveAct := map[int]bool{}
	for _, a := range t.Actions {
		haveAct[a.Number] = true
	}
	wantAct := map[int]bool{}
	for _, r := range td.Actions {
		wantAct[r.Number] = true
		a := authoring.Action{
			Number:  r.Number,
			Comment: r.Comment,
			DSL:     r.Description,
			Columns: actColumns(r.Columns),
		}
		var err error
		if haveAct[r.Number] {
			err = t.UpdateAction(r.Number, a)
		} else {
			err = t.AddAction(a)
		}
		if err != nil {
			return fmt.Errorf("action %d: %w", r.Number, err)
		}
	}
	for _, a := range append([]authoring.Action(nil), t.Actions...) {
		if !wantAct[a.Number] {
			if err := t.DeleteAction(a.Number); err != nil {
				return fmt.Errorf("delete action %d: %w", a.Number, err)
			}
		}
	}
	return nil
}

// condColumns converts the editor's string-keyed cell map to the authoring
// form. Empty and don't-care cells are simply absent in both.
func condColumns(cols map[string]string) map[int]string {
	out := make(map[int]string, len(cols))
	for k, v := range cols {
		n, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil || n <= 0 {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" || v == "-" {
			continue
		}
		out[n] = v
	}
	return out
}

// actColumns: any marked cell (canonically "X") means the action fires on
// that column.
func actColumns(cols map[string]string) map[int]bool {
	out := make(map[int]bool, len(cols))
	for k, v := range cols {
		n, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil || n <= 0 {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" || v == "-" {
			continue
		}
		out[n] = true
	}
	return out
}

// reconcileEDD applies the editor model's entities for one EDD file onto
// the Project's loaded EDD. Attribute metadata the editor does not carry
// (collect flags, question definitions) survives because UpdateAttribute
// merges by name with keep-existing semantics — an improvement on the old
// serializer, which matched that metadata by row index.
func (s *Server) reconcileEDD(p *authoring.Project, relPath string) error {
	keep := make(map[string]*EntityData)
	for _, e := range s.entities {
		if e.Source == relPath {
			keep[e.Name] = e
		}
	}

	edd := p.EDD()
	for _, ent := range edd.Entities() {
		if keep[ent.Name] == nil {
			if err := edd.DeleteEntity(ent.Name); err != nil {
				return fmt.Errorf("delete entity %s: %w", ent.Name, err)
			}
		}
	}

	for _, e := range s.entities {
		if e.Source != relPath {
			continue
		}
		ent := edd.Entity(e.Name)
		if ent == nil {
			created, err := edd.AddEntity(e.Name)
			if err != nil {
				return fmt.Errorf("create entity %s: %w", e.Name, err)
			}
			ent = created
		}
		ent.SetNumber(e.Number)
		ent.SetAccess(e.Access)
		ent.SetComment(e.Comment)

		haveAttr := map[string]bool{}
		for _, a := range ent.Attributes {
			haveAttr[a.Name] = true
		}
		wantAttr := map[string]bool{}
		for _, f := range e.Fields {
			wantAttr[f.Name] = true
			a := authoring.Attribute{
				Name:    f.Name,
				Type:    f.Type,
				Subtype: f.Subtype,
				Access:  f.Access,
				Input:   f.Input,
				Default: f.DefaultValue,
				Comment: f.Comment,
			}
			var err error
			if haveAttr[f.Name] {
				err = ent.UpdateAttribute(f.Name, a)
			} else {
				err = ent.AddAttribute(a)
			}
			if err != nil {
				return fmt.Errorf("entity %s field %s: %w", e.Name, f.Name, err)
			}
		}
		for _, a := range append([]authoring.Attribute(nil), ent.Attributes...) {
			if !wantAttr[a.Name] {
				if err := ent.DeleteAttribute(a.Name); err != nil {
					return fmt.Errorf("entity %s delete field %s: %w", e.Name, a.Name, err)
				}
			}
		}
	}
	return nil
}

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

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// A mapping entry that resolves against nothing does not fail at load. The
// loader looks up the enclosure, misses, records an empty pending attribute,
// and the value is dropped -- the run continues and produces a plausible wrong
// answer. A `list` naming an array that does not exist is the same shape:
// updateReferences finds no array by that name on the entity stack and appends
// nowhere, so the array stays empty and every `for all` over it iterates zero
// times.
//
// That is how TaxReturn's multi-state path computed nothing for months and how
// CorporateTax's per-state documents loaded to all zeroes (#1094). Each time it
// was found by noticing the answers were wrong.
//
// So a map write is checked against the EDD before it lands. authoring.Mapping
// has had this check for setattributes since it was written, and nothing ever
// called it -- the type is referenced only from docs.go (#1173).

// eddField is what the check needs to know about one declared field.
type eddField struct {
	Type    string
	Subtype string
}

// eddModel is the declared shape of a project: entity name -> field name ->
// field, both lower-cased, because EL is case-insensitive.
type eddModel struct {
	entities map[string]map[string]eddField
}

// loadEDDModel reads every *_edd.xml under xmlDir. Returns nil when the
// project declares nothing, which is not an error: a mapping can legitimately
// be authored before its EDD exists, and refusing to write one then would make
// the ordering of two files load-bearing.
func loadEDDModel(xmlDir string) *eddModel {
	type fieldXML struct {
		Name    string `xml:"name,attr"`
		Type    string `xml:"type,attr"`
		Subtype string `xml:"subtype,attr"`
	}
	type entityXML struct {
		Name   string     `xml:"name,attr"`
		Fields []fieldXML `xml:"field"`
	}
	type fileXML struct {
		Entities []entityXML `xml:"entity"`
	}

	m := &eddModel{entities: make(map[string]map[string]eddField)}
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_edd.xml") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		var f fileXML
		if xml.Unmarshal(data, &f) != nil {
			return nil
		}
		for _, ent := range f.Entities {
			name := strings.ToLower(strings.TrimSpace(ent.Name))
			if name == "" {
				continue
			}
			fields := m.entities[name]
			if fields == nil {
				fields = make(map[string]eddField)
				m.entities[name] = fields
			}
			for _, fld := range ent.Fields {
				fn := strings.ToLower(strings.TrimSpace(fld.Name))
				if fn == "" {
					continue
				}
				fields[fn] = eddField{
					Type:    strings.ToLower(strings.TrimSpace(fld.Type)),
					Subtype: strings.ToLower(strings.TrimSpace(fld.Subtype)),
				}
			}
		}
		return nil
	})
	if len(m.entities) == 0 {
		return nil
	}
	return m
}

// hasEntity reports whether the EDD declares this entity.
func (e *eddModel) hasEntity(name string) bool {
	_, ok := e.entities[strings.ToLower(name)]
	return ok
}

// field returns a declared field, if any.
func (e *eddModel) field(entity, name string) (eddField, bool) {
	fields, ok := e.entities[strings.ToLower(entity)]
	if !ok {
		return eddField{}, false
	}
	f, ok := fields[strings.ToLower(name)]
	return f, ok
}

// arrayHolding reports whether any entity declares an array field of this name
// whose subtype is the given entity -- the array a createentity's `list` names.
func (e *eddModel) arrayHolding(list, entity string) bool {
	list = strings.ToLower(list)
	entity = strings.ToLower(entity)
	for _, fields := range e.entities {
		f, ok := fields[list]
		if !ok || f.Type != "array" {
			continue
		}
		// An untyped array cannot be contradicted, so it is accepted.
		if f.Subtype == "" || f.Subtype == entity {
			return true
		}
	}
	return false
}

// entityNames returns the declared entity names, sorted, for error messages.
func (e *eddModel) entityNames() []string {
	out := make([]string, 0, len(e.entities))
	for n := range e.entities {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// validateMapOp checks a patch against the EDD before it is applied. A nil
// model means the project declares no EDD yet and nothing can be checked.
func validateMapOp(m *excel.MapXML, op mapPatchOp, edd *eddModel) error {
	if edd == nil {
		return nil
	}

	switch op.Op {
	case "add-attribute":
		a := op.Attribute
		if a == nil {
			return nil // shape errors belong to applyMapOp
		}
		if !edd.hasEntity(a.Enclosure) {
			return fmt.Errorf("enclosure %q is not an entity the EDD declares; "+
				"a setattribute whose enclosure resolves against nothing drops its value at load "+
				"without erroring", a.Enclosure)
		}
		rattr := defaultString(a.RAttribute, a.Tag)
		declared, ok := edd.field(a.Enclosure, rattr)
		if !ok {
			return fmt.Errorf("%s has no field %q in the EDD; declare it with `dtrules edd patch` first",
				a.Enclosure, rattr)
		}
		if a.Type != "" && declared.Type != "" && !strings.EqualFold(a.Type, declared.Type) {
			return fmt.Errorf("type mismatch for %s.%s: the EDD declares %q, the mapping says %q",
				a.Enclosure, rattr, declared.Type, a.Type)
		}

	case "add-entity", "add-create-entity":
		if op.Entity == "" {
			return nil
		}
		if !edd.hasEntity(op.Entity) {
			return fmt.Errorf("entity %q is not declared in the EDD (declared: %s)",
				op.Entity, strings.Join(edd.entityNames(), ", "))
		}
		return validateList(m, op, edd)
	}
	return nil
}

// validateList checks the array a createentity appends its instances to.
//
// Only for a number='*' entity: a singleton is bound by the initialization
// stack and belongs to no array. When `list` is omitted the loader falls back
// to the entity name plus "s", so that fallback is what gets checked -- an
// entity whose plural does not exist has to say where its instances go.
func validateList(m *excel.MapXML, op mapPatchOp, edd *eddModel) error {
	cardinality := op.Number
	if op.Op == "add-create-entity" {
		cardinality = ""
		for _, d := range m.EntityDecls {
			if strings.EqualFold(d.Name, op.Entity) {
				cardinality = d.Number
				break
			}
		}
	}
	if cardinality == "" {
		cardinality = "1"
	}
	if cardinality != "*" {
		return nil
	}

	if op.List != "" {
		if !edd.arrayHolding(op.List, op.Entity) {
			return fmt.Errorf("list %q is not an array of %s in the EDD; instances would be "+
				"created and appended nowhere, leaving every `for all` over it to iterate zero times",
				op.List, op.Entity)
		}
		return nil
	}

	fallback := op.Entity + "s"
	if !edd.arrayHolding(fallback, op.Entity) {
		return fmt.Errorf("a number='*' entity needs a list: with none given the loader falls back "+
			"to %q, which the EDD does not declare as an array of %s, so instances would be "+
			"appended nowhere", fallback, op.Entity)
	}
	return nil
}

// validateWholeMap checks a mapping written in one piece, the way
// validateMapOp checks one written a patch at a time. Reported together
// rather than one at a time: an author fixing a whole document wants the
// whole list, not to rediscover it a run at a time.
func validateWholeMap(m *excel.MapXML, edd *eddModel) error {
	if edd == nil {
		return nil
	}
	declared := make(map[string]string, len(m.EntityDecls))
	for _, d := range m.EntityDecls {
		declared[strings.ToLower(d.Name)] = d.Number
	}

	var problems []string
	for _, d := range m.EntityDecls {
		if !edd.hasEntity(d.Name) {
			problems = append(problems,
				fmt.Sprintf("entity %q is not declared in the EDD", d.Name))
		}
	}
	for _, e := range m.InitialEntities {
		if _, ok := declared[strings.ToLower(e.Entity)]; !ok {
			problems = append(problems,
				fmt.Sprintf("initialentity %q is pushed but never declared in <entities>", e.Entity))
		}
	}
	seenTag := make(map[string]string, len(m.CreateEntities))
	for _, c := range m.CreateEntities {
		key := strings.ToLower(c.Tag)
		if prior, ok := seenTag[key]; ok {
			problems = append(problems,
				fmt.Sprintf("tag %q creates both %q and %q; a tag roots one entity", c.Tag, prior, c.Entity))
		}
		seenTag[key] = c.Entity
		if _, ok := declared[strings.ToLower(c.Entity)]; !ok {
			problems = append(problems,
				fmt.Sprintf("createentity %q is not declared in <entities>", c.Entity))
			continue
		}
		op := mapPatchOp{Op: "add-create-entity", Entity: c.Entity, Tag: c.Tag, ID: c.ID, List: c.List}
		if err := validateList(m, op, edd); err != nil {
			problems = append(problems, err.Error())
		}
	}
	for _, e := range m.Entries {
		if e.IsSection {
			continue
		}
		op := mapPatchOp{Op: "add-attribute", Attribute: &mapAttributeJSON{
			Tag: e.Tag, RAttribute: e.RAttribute, Enclosure: e.Enclosure, Type: e.Type}}
		if err := validateMapOp(m, op, edd); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if len(problems) == 0 {
		return nil
	}
	if len(problems) > 12 {
		extra := len(problems) - 12
		problems = problems[:12]
		problems = append(problems, fmt.Sprintf("... and %d more", extra))
	}
	return fmt.Errorf("%d problem(s):\n  - %s", len(problems), strings.Join(problems, "\n  - "))
}

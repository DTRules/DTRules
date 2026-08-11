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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// A mapping decides which entity each external tag lands in, and which
// entities exist on the stack at all. It was the one authoring surface with no
// write API -- CorporateTax's map file carries a comment saying its entity
// block was "added by hand because the mapping authoring API covers only
// setattribute entries" (#1103).

func newMap() *excel.MapXML {
	return &excel.MapXML{
		EntityDecls:     []excel.MapEntityDecl{{Name: "job", Number: "1"}},
		InitialEntities: []excel.MapInitialEntity{{Entity: "job", EPush: true}},
	}
}

// "This entity exists" is two declarations: a cardinality entry and a push.
// An entity with only the first is declared and never created, and every rule
// that reads it fails with "not defined by any Entity on the Entity Stack".
// Making the caller remember both would be an invitation to that bug.
func TestAddEntityAlsoPushesIt(t *testing.T) {
	m := newMap()

	if err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "nv_result"}); err != nil {
		t.Fatalf("add-entity: %v", err)
	}

	var declared, pushed bool
	for _, e := range m.EntityDecls {
		if e.Name == "nv_result" {
			declared = true
			if e.Number != "1" {
				t.Errorf("cardinality defaulted to %q, want 1", e.Number)
			}
		}
	}
	for _, e := range m.InitialEntities {
		if e.Entity == "nv_result" && e.EPush {
			pushed = true
		}
	}
	if !declared {
		t.Error("entity was not declared")
	}
	if !pushed {
		t.Error("entity was declared but never pushed, so no rule could read it")
	}
}

// A "*" entity is created per input element, not pushed as a singleton, so it
// must not go on the initialization stack.
func TestAManyEntityIsNotPushed(t *testing.T) {
	m := newMap()

	if err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "bracket", Number: "*"}); err != nil {
		t.Fatalf("add-entity: %v", err)
	}
	for _, e := range m.InitialEntities {
		if e.Entity == "bracket" {
			t.Error("a cardinality-* entity must not be pushed as a singleton")
		}
	}
}

func TestAddingAnEntityTwiceIsRefused(t *testing.T) {
	m := newMap()
	if err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "job"}); err == nil {
		t.Fatal("re-declaring an entity should be refused, not silently duplicated")
	}
}

func TestDeleteEntityRemovesBothDeclarations(t *testing.T) {
	m := newMap()
	_ = applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "nv_result"})

	if err := applyMapOp(m, mapPatchOp{Op: "delete-entity", Entity: "nv_result"}); err != nil {
		t.Fatalf("delete-entity: %v", err)
	}
	for _, e := range m.EntityDecls {
		if e.Name == "nv_result" {
			t.Error("declaration left behind")
		}
	}
	for _, e := range m.InitialEntities {
		if e.Entity == "nv_result" {
			t.Error("initialization push left behind, so the stack still builds it")
		}
	}
}

// An attribute with no RAttribute maps to a field of the same name; that
// default is what the XML format itself does.
func TestAddAttributeDefaultsRAttributeToTheTag(t *testing.T) {
	m := newMap()

	err := applyMapOp(m, mapPatchOp{Op: "add-attribute", Attribute: &mapAttributeJSON{
		Tag: "gross_revenue", Enclosure: "nv_result", Type: "double"}})
	if err != nil {
		t.Fatalf("add-attribute: %v", err)
	}
	e := m.Entries[len(m.Entries)-1]
	if e.RAttribute != "gross_revenue" {
		t.Errorf("RAttribute = %q, want the tag", e.RAttribute)
	}
	if e.Enclosure != "nv_result" {
		t.Errorf("enclosure = %q", e.Enclosure)
	}
}

func TestUnknownOpIsNamed(t *testing.T) {
	err := applyMapOp(newMap(), mapPatchOp{Op: "rename-everything"})
	if err == nil || !strings.Contains(err.Error(), "rename-everything") {
		t.Errorf("an unknown op should be named back, got: %v", err)
	}
}

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
	"os"
	"path/filepath"
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

	// And the third declaration: a tag, so a document can name the entity.
	// Without it the entity works but no data can be addressed to it -- an
	// element no mapping knows is unknown markup, so the loader skips it and
	// every child tag inside it, saying nothing.
	//
	// CorporateTax's per-state documents landed exactly there: <me_result>
	// wrapping three mapped tags loaded as all zeroes, the rules ran, and the
	// Maine tax came out 0 on $4,000,000 of income (#1094).
	var tagged bool
	for _, c := range m.CreateEntities {
		if c.Entity == "nv_result" {
			tagged = true
			if c.Tag != "nv_result" {
				t.Errorf("tag defaulted to %q, want the entity's own name", c.Tag)
			}
		}
	}
	if !tagged {
		t.Error("entity has no tag, so no document could load data into it")
	}
}

// A caller who wants the tag to differ from the entity name says so.
func TestAddEntityHonoursAnExplicitTag(t *testing.T) {
	m := newMap()

	if err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "nv_result", Tag: "nevada"}); err != nil {
		t.Fatalf("add-entity: %v", err)
	}
	for _, c := range m.CreateEntities {
		if c.Entity == "nv_result" && c.Tag != "nevada" {
			t.Errorf("tag = %q, want nevada", c.Tag)
		}
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
	for _, c := range m.CreateEntities {
		if c.Entity == "nv_result" {
			t.Error("tag left behind, so the mapping creates an entity nothing declares")
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

// A number='*' entity is appended to an array field on its enclosing entity,
// and list=” names that array. Without it the created instances belong to
// nothing: the array stays empty and every `for all` over it iterates zero
// times, silently computing nothing. TaxReturn's state_period is unmapped for
// exactly this reason, so its multi-state rules run against an empty list
// (#1164, #234).
func TestAddEntityCarriesTheListAttribute(t *testing.T) {
	m := newMap()

	err := applyMapOp(m, mapPatchOp{
		Op: "add-entity", Entity: "state_period", Number: "*", List: "state_periods"})
	if err != nil {
		t.Fatalf("add-entity: %v", err)
	}

	var found bool
	for _, c := range m.CreateEntities {
		if c.Entity != "state_period" {
			continue
		}
		found = true
		if c.List != "state_periods" {
			t.Errorf("list = %q, want state_periods -- without it the array stays empty", c.List)
		}
		if c.ID != "id" {
			t.Errorf("id = %q, want the default id", c.ID)
		}
	}
	if !found {
		t.Fatal("no createentity was written, so no document can be rooted at the entity")
	}
}

func TestAddEntityHonoursAnExplicitID(t *testing.T) {
	m := newMap()

	err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "bracket", Number: "*", ID: "bracket_no"})
	if err != nil {
		t.Fatalf("add-entity: %v", err)
	}

	var found bool
	for _, c := range m.CreateEntities {
		if c.Entity != "bracket" {
			continue
		}
		found = true
		if c.ID != "bracket_no" {
			t.Errorf("id = %q, want bracket_no", c.ID)
		}
	}
	if !found {
		t.Fatal("no createentity was written, so the assertion above checked nothing")
	}
}

// add-create-entity exists for what add-entity cannot express: a second tag
// rooting an entity that is already declared.
func TestAddCreateEntityAddsASecondTag(t *testing.T) {
	m := newMap()
	if err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "period", Number: "*",
		List: "periods"}); err != nil {
		t.Fatalf("add-entity: %v", err)
	}

	err := applyMapOp(m, mapPatchOp{Op: "add-create-entity", Entity: "period",
		Tag: "state_period", List: "periods"})
	if err != nil {
		t.Fatalf("add-create-entity: %v", err)
	}

	var tags []string
	for _, c := range m.CreateEntities {
		if c.Entity == "period" {
			tags = append(tags, c.Tag)
		}
	}
	if len(tags) != 2 {
		t.Fatalf("tags rooting period = %v, want two", tags)
	}
}

// A createentity for an entity nothing declares creates instances the entity
// stack has no room for -- the mirror of the delete-entity symmetry.
func TestAddCreateEntityRefusesAnUndeclaredEntity(t *testing.T) {
	err := applyMapOp(newMap(), mapPatchOp{Op: "add-create-entity", Entity: "ghost"})
	if err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Errorf("want a refusal naming the undeclared entity, got: %v", err)
	}
}

func TestAddCreateEntityRefusesADuplicateTag(t *testing.T) {
	m := newMap()
	_ = applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "period", Number: "*"})

	err := applyMapOp(m, mapPatchOp{Op: "add-create-entity", Entity: "period"})
	if err == nil || !strings.Contains(err.Error(), "already creates") {
		t.Errorf("a tag can only root one entity, got: %v", err)
	}
}

func TestDeleteCreateEntityRemovesOnlyThatTag(t *testing.T) {
	m := newMap()
	_ = applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "period", Number: "*", List: "periods"})
	_ = applyMapOp(m, mapPatchOp{Op: "add-create-entity", Entity: "period", Tag: "state_period"})

	if err := applyMapOp(m, mapPatchOp{Op: "delete-create-entity", Tag: "state_period"}); err != nil {
		t.Fatalf("delete-create-entity: %v", err)
	}

	for _, c := range m.CreateEntities {
		if c.Tag == "state_period" {
			t.Error("the deleted tag is still present")
		}
	}
	var kept bool
	for _, c := range m.CreateEntities {
		if c.Tag == "period" {
			kept = true
		}
	}
	if !kept {
		t.Error("delete-create-entity removed a tag it was not asked to remove")
	}
}

func TestDeleteCreateEntityNamesAMissingTag(t *testing.T) {
	err := applyMapOp(newMap(), mapPatchOp{Op: "delete-create-entity", Tag: "nowhere"})
	if err == nil || !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("want the missing tag named back, got: %v", err)
	}
}

// A tag roots exactly one entity. add-entity used to skip the createentity
// whenever the tag was taken, report success, and leave the entity declared
// and never created -- the failure the three-in-one op exists to prevent --
// silently discarding the caller's list along with it.
func TestAddEntityRefusesATagAnotherEntityRoots(t *testing.T) {
	m := newMap()
	m.CreateEntities = append(m.CreateEntities,
		excel.MapCreateEntity{Entity: "income", Tag: "income", ID: "id"})

	err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "state_period",
		Number: "*", Tag: "income", List: "state_periods"})
	if err == nil {
		t.Fatal("a tag already rooting another entity must be refused, not silently skipped")
	}
	if !strings.Contains(err.Error(), "income") {
		t.Errorf("the conflict should name the tag and the entity holding it, got: %v", err)
	}
}

// When the createentity is already there for this same entity, the mapping was
// missing only its declaration. That is a repair, not a conflict -- but an
// id or list the caller supplies still has to land, or the call reports
// success having written neither.
func TestAddEntityRepairsAMissingDeclaration(t *testing.T) {
	m := newMap()
	m.CreateEntities = append(m.CreateEntities,
		excel.MapCreateEntity{Entity: "state_period", Tag: "state_period", ID: "id"})

	err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "state_period",
		Number: "*", List: "state_periods"})
	if err != nil {
		t.Fatalf("declaring an entity whose createentity already exists is a repair: %v", err)
	}

	var declared bool
	for _, e := range m.EntityDecls {
		if e.Name == "state_period" {
			declared = true
		}
	}
	if !declared {
		t.Error("the missing declaration was not added")
	}
	if n := len(m.CreateEntities); n != 1 {
		t.Errorf("createentity count = %d, want the existing one reused", n)
	}
	if got := m.CreateEntities[0].List; got != "state_periods" {
		t.Errorf("list = %q, want state_periods -- a supplied list must land even on the repair path", got)
	}
}

// delete-create-entity identifies the row by tag, with entity as an optional
// filter -- the same way delete-attribute reads them. Falling back to the
// entity name as the tag would miss every tag not named after its entity and
// report the miss against a tag the caller never supplied.
func TestDeleteCreateEntityNeedsATagNotAnEntity(t *testing.T) {
	m := newMap()
	_ = applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "period", Number: "*"})
	_ = applyMapOp(m, mapPatchOp{Op: "add-create-entity", Entity: "period", Tag: "state_period"})

	err := applyMapOp(m, mapPatchOp{Op: "delete-create-entity", Entity: "period"})
	if err == nil {
		t.Fatal("a delete with no tag should be refused, not resolved to the entity name")
	}
	if !strings.Contains(err.Error(), "delete-entity") {
		t.Errorf("the refusal should point at the op that removes an entity outright, got: %v", err)
	}
	if len(m.CreateEntities) != 2 {
		t.Errorf("a refused delete must leave the mapping untouched, have %d rows", len(m.CreateEntities))
	}
}

// entity narrows a tag that more than one mapping could match.
func TestDeleteCreateEntityFiltersByEntity(t *testing.T) {
	m := newMap()
	_ = applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "period", Number: "*"})

	err := applyMapOp(m, mapPatchOp{Op: "delete-create-entity", Tag: "period", Entity: "somethingelse"})
	if err == nil {
		t.Error("the entity filter should have excluded the only row with that tag")
	}
	if len(m.CreateEntities) != 1 {
		t.Errorf("nothing should have been removed, have %d rows", len(m.CreateEntities))
	}
}

// A refused add-entity must not leave the declaration behind: validation comes
// before any mutation, or a rejected op still half-lands.
func TestAddEntityConflictLeavesNothingBehind(t *testing.T) {
	m := newMap()
	m.CreateEntities = append(m.CreateEntities,
		excel.MapCreateEntity{Entity: "income", Tag: "income", ID: "id"})

	before := len(m.EntityDecls)
	err := applyMapOp(m, mapPatchOp{Op: "add-entity", Entity: "state_period",
		Number: "1", Tag: "income"})
	if err == nil {
		t.Fatal("expected the tag conflict to be refused")
	}
	if len(m.EntityDecls) != before {
		t.Errorf("the refused op still declared the entity: %+v", m.EntityDecls)
	}
	for _, e := range m.InitialEntities {
		if e.Entity == "state_period" {
			t.Error("the refused op still pushed the entity onto the initialization stack")
		}
	}
}

// A mapping entry that resolves against nothing is silent at load: the loader
// looks up the enclosure, misses, and drops the value. So a map write is
// checked against the EDD before it lands (#1173).
func testEDD(t *testing.T) *eddModel {
	t.Helper()
	dir := t.TempDir()
	edd := `<entity_data_dictionary>
  <entity name="job"><field name="state" type="string"/><field name="periods" type="array" subtype="state_period"/></entity>
  <entity name="state_period"><field name="id" type="integer"/><field name="state_code" type="string"/></entity>
</entity_data_dictionary>`
	if err := os.WriteFile(filepath.Join(dir, "x_edd.xml"), []byte(edd), 0o644); err != nil {
		t.Fatal(err)
	}
	m := loadEDDModel(dir)
	if m == nil {
		t.Fatal("fixture EDD did not load")
	}
	return m
}

func TestAddAttributeRefusesAnUndeclaredEnclosure(t *testing.T) {
	err := validateMapOp(newMap(), mapPatchOp{Op: "add-attribute",
		Attribute: &mapAttributeJSON{Tag: "x", Enclosure: "nowhere", Type: "double"}}, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("want a refusal naming the enclosure, got: %v", err)
	}
}

func TestAddAttributeRefusesAnUndeclaredField(t *testing.T) {
	err := validateMapOp(newMap(), mapPatchOp{Op: "add-attribute",
		Attribute: &mapAttributeJSON{Tag: "nonesuch", Enclosure: "job", Type: "string"}}, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "nonesuch") {
		t.Errorf("want a refusal naming the field, got: %v", err)
	}
}

func TestAddAttributeRefusesATypeMismatch(t *testing.T) {
	err := validateMapOp(newMap(), mapPatchOp{Op: "add-attribute",
		Attribute: &mapAttributeJSON{Tag: "state", Enclosure: "job", Type: "double"}}, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "type mismatch") {
		t.Errorf("want a type mismatch, got: %v", err)
	}
}

func TestAddAttributeAcceptsADeclaredField(t *testing.T) {
	if err := validateMapOp(newMap(), mapPatchOp{Op: "add-attribute",
		Attribute: &mapAttributeJSON{Tag: "state", Enclosure: "job", Type: "string"}}, testEDD(t)); err != nil {
		t.Errorf("a declared field must be accepted: %v", err)
	}
}

// The list has to be an array that actually holds this entity, or the
// instances are appended nowhere and every `for all` over it iterates zero
// times.
func TestListMustBeAnArrayOfTheEntity(t *testing.T) {
	err := validateMapOp(newMap(), mapPatchOp{Op: "add-entity",
		Entity: "state_period", Number: "*", List: "not_an_array"}, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "not_an_array") {
		t.Errorf("want a refusal naming the list, got: %v", err)
	}

	if err := validateMapOp(newMap(), mapPatchOp{Op: "add-entity",
		Entity: "state_period", Number: "*", List: "periods"}, testEDD(t)); err != nil {
		t.Errorf("an array declared with this entity's subtype must be accepted: %v", err)
	}
}

// With no list the loader falls back to the entity name plus "s". An entity
// whose plural is not the array name -- unreported_tips, address -- has to say
// where its instances go, and is told so at authoring time rather than
// discovering it as an empty array at run time.
func TestOmittedListIsRefusedWhenThePluralDoesNotExist(t *testing.T) {
	err := validateMapOp(newMap(), mapPatchOp{Op: "add-entity",
		Entity: "state_period", Number: "*"}, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "state_periods") {
		t.Errorf("want the attempted fallback named, got: %v", err)
	}
}

// A singleton is bound by the initialization stack and belongs to no array.
func TestASingletonNeedsNoList(t *testing.T) {
	if err := validateMapOp(newMap(), mapPatchOp{Op: "add-entity",
		Entity: "job", Number: "1"}, testEDD(t)); err != nil {
		t.Errorf("a singleton must not be asked for a list: %v", err)
	}
}

// A project with no EDD yet cannot be checked, and refusing to write its
// mapping would make the order the two files are authored in load-bearing.
func TestNoEDDMeansNoCheck(t *testing.T) {
	if err := validateMapOp(newMap(), mapPatchOp{Op: "add-attribute",
		Attribute: &mapAttributeJSON{Tag: "x", Enclosure: "anything"}}, nil); err != nil {
		t.Errorf("with no EDD declared the check must stand down: %v", err)
	}
}

// put takes exactly what get emits. Round-trip symmetry is the contract that
// makes the pair usable: read a mapping, change one thing, write it back.
func TestMapJSONRoundTripsThroughTheModel(t *testing.T) {
	original := &excel.MapXML{
		MapName:         "Sample",
		EntityDecls:     []excel.MapEntityDecl{{Name: "job", Number: "1"}, {Name: "applicant", Number: "*"}},
		InitialEntities: []excel.MapInitialEntity{{Entity: "job", EPush: true}},
		CreateEntities: []excel.MapCreateEntity{
			{Entity: "applicant", Tag: "applicant", ID: "id", List: "applicants"}},
		Entries: []excel.MapEntry{
			{Tag: "program", RAttribute: "program", Enclosure: "job", Type: "string"}},
	}

	back, err := mapFromJSON(mapToJSON(original))
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if back.MapName != original.MapName {
		t.Errorf("name = %q, want %q", back.MapName, original.MapName)
	}
	if len(back.EntityDecls) != 2 || back.EntityDecls[1].Number != "*" {
		t.Errorf("entity declarations did not survive: %+v", back.EntityDecls)
	}
	if len(back.InitialEntities) != 1 || !back.InitialEntities[0].EPush {
		t.Errorf("initialization stack did not survive: %+v", back.InitialEntities)
	}
	if len(back.CreateEntities) != 1 || back.CreateEntities[0].List != "applicants" {
		t.Errorf("createentity did not survive: %+v", back.CreateEntities)
	}
	if len(back.Entries) != 1 || back.Entries[0].Enclosure != "job" {
		t.Errorf("attributes did not survive: %+v", back.Entries)
	}
}

func TestMapFromJSONDefaultsTagAndID(t *testing.T) {
	m, err := mapFromJSON(mapJSON{
		Entities:       []mapEntityJSON{{Name: "applicant", Number: "*"}},
		CreateEntities: []mapCreateJSON{{Entity: "applicant", List: "applicants"}},
		Attributes:     []mapAttributeJSON{{Tag: "age", Enclosure: "applicant"}},
	})
	if err != nil {
		t.Fatalf("mapFromJSON: %v", err)
	}
	if got := m.CreateEntities[0].Tag; got != "applicant" {
		t.Errorf("tag = %q, want the entity name", got)
	}
	if got := m.CreateEntities[0].ID; got != "id" {
		t.Errorf("id = %q, want the default", got)
	}
	if got := m.Entries[0].RAttribute; got != "age" {
		t.Errorf("rattribute = %q, want the tag", got)
	}
}

func TestMapFromJSONRefusesABadCardinality(t *testing.T) {
	_, err := mapFromJSON(mapJSON{Entities: []mapEntityJSON{{Name: "job", Number: "many"}}})
	if err == nil || !strings.Contains(err.Error(), "many") {
		t.Errorf("want the bad cardinality named, got: %v", err)
	}
}

// A whole-document write reports every problem at once. An author fixing a
// mapping wants the list, not to rediscover it one run at a time.
func TestWholeMapReportsEveryProblemTogether(t *testing.T) {
	m := &excel.MapXML{
		EntityDecls:     []excel.MapEntityDecl{{Name: "job", Number: "1"}, {Name: "ghost", Number: "*"}},
		InitialEntities: []excel.MapInitialEntity{{Entity: "never_declared", EPush: true}},
		Entries: []excel.MapEntry{
			{Tag: "nonesuch", RAttribute: "nonesuch", Enclosure: "job", Type: "string"}},
	}

	err := validateWholeMap(m, testEDD(t))
	if err == nil {
		t.Fatal("expected the undeclared references to be refused")
	}
	for _, want := range []string{"ghost", "never_declared", "nonesuch"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the report should name %q: %v", want, err)
		}
	}
}

// A tag roots one entity, in a whole document as much as in a patch.
func TestWholeMapCatchesADuplicateTag(t *testing.T) {
	m := &excel.MapXML{
		EntityDecls: []excel.MapEntityDecl{{Name: "job", Number: "1"}, {Name: "state_period", Number: "*"}},
		CreateEntities: []excel.MapCreateEntity{
			{Entity: "job", Tag: "thing", ID: "id"},
			{Entity: "state_period", Tag: "thing", ID: "id", List: "periods"}},
	}
	err := validateWholeMap(m, testEDD(t))
	if err == nil || !strings.Contains(err.Error(), "roots one entity") {
		t.Errorf("want the duplicate tag refused, got: %v", err)
	}
}

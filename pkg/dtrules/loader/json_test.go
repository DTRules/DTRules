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

package loader

import (
	"errors"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

func TestJSONEDDLoaderBasic(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "person",
				"access": "rw",
				"fields": [
					{"name": "name", "type": "string", "access": "rw"},
					{"name": "age", "type": "integer", "access": "rw"},
					{"name": "active", "type": "boolean", "access": "r"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	personName := dtrules.GetRName("person")
	personEntity, _ := factory.GetReferenceEntity(personName)
	if personEntity == nil {
		t.Fatal("Expected person entity to be created")
	}

	if !personEntity.ContainsAttribute(dtrules.GetRName("name")) {
		t.Error("Expected 'name' attribute")
	}
	if !personEntity.ContainsAttribute(dtrules.GetRName("age")) {
		t.Error("Expected 'age' attribute")
	}
	if !personEntity.ContainsAttribute(dtrules.GetRName("active")) {
		t.Error("Expected 'active' attribute")
	}
}

func TestJSONEDDLoaderMultipleEntities(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "order",
				"access": "rw",
				"fields": [
					{"name": "order_id", "type": "integer", "access": "rw"},
					{"name": "total", "type": "double", "access": "rw"}
				]
			},
			{
				"name": "customer",
				"access": "rw",
				"fields": [
					{"name": "customer_id", "type": "integer", "access": "rw"},
					{"name": "email", "type": "string", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	orderEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("order"))
	if orderEntity == nil {
		t.Error("Expected order entity")
	}

	customerEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("customer"))
	if customerEntity == nil {
		t.Error("Expected customer entity")
	}
}

func TestJSONEDDLoaderMalformedJSON(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{"entities": [{"name": "broken"`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestJSONEDDLoaderUnknownType(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "test",
				"access": "rw",
				"fields": [
					{"name": "field1", "type": "unknowntype", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for unknown type")
	}

	var loadErr *EDDLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("Expected EDDLoadError, got: %T", err)
	}

	if len(loadErr.Errors) == 0 {
		t.Error("Expected at least one error in EDDLoadError")
	}
}

func TestJSONEDDLoaderEmptyJSON(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{"entities": []}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Empty JSON EDD should load without error: %v", err)
	}
}

func TestJSONEDDLoaderDefaultValues(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "defaults",
				"access": "rw",
				"fields": [
					{"name": "count", "type": "integer", "access": "rw", "defaultValue": "42"},
					{"name": "rate", "type": "double", "access": "rw", "defaultValue": "3.14"},
					{"name": "flag", "type": "boolean", "access": "rw", "defaultValue": "true"},
					{"name": "label", "type": "string", "access": "rw", "defaultValue": "hello"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("defaults"))
	if ent == nil {
		t.Fatal("Expected defaults entity")
	}

	if !ent.ContainsAttribute(dtrules.GetRName("count")) {
		t.Error("Expected 'count' attribute")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("rate")) {
		t.Error("Expected 'rate' attribute")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("flag")) {
		t.Error("Expected 'flag' attribute")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("label")) {
		t.Error("Expected 'label' attribute")
	}
}

func TestJSONEDDLoaderFieldAccess(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "accesstest",
				"access": "rw",
				"fields": [
					{"name": "readonly", "type": "string", "access": "r"},
					{"name": "writeonly", "type": "string", "access": "w"},
					{"name": "readwrite", "type": "string", "access": "rw"},
					{"name": "nospec", "type": "string"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("accesstest"))
	if ent == nil {
		t.Fatal("Expected accesstest entity")
	}

	fields := []string{"readonly", "writeonly", "readwrite", "nospec"}
	for _, f := range fields {
		if !ent.ContainsAttribute(dtrules.GetRName(f)) {
			t.Errorf("Expected '%s' attribute", f)
		}
	}
}

func TestJSONEDDLoaderReadError(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	err := loader.Load(&errorReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

func TestJSONEDDLoaderSizeLimit(t *testing.T) {
	originalMax := MaxJSONSize
	defer func() { MaxJSONSize = originalMax }()

	MaxJSONSize = 50

	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "test",
				"access": "rw",
				"fields": [
					{"name": "value", "type": "integer", "access": "rw", "comment": "Makes this JSON larger than the limit"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for oversized JSON")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

func TestJSONEDDLoaderSizeLimitDisabled(t *testing.T) {
	originalMax := MaxJSONSize
	defer func() { MaxJSONSize = originalMax }()

	MaxJSONSize = 0

	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "test",
				"access": "rw",
				"fields": [
					{"name": "value", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Expected no error with size limit disabled: %v", err)
	}
}

func TestJSONEDDLoaderInvalidEntityName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": ".invalid",
				"fields": [
					{"name": "value", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for invalid entity name")
	}
	if !strings.Contains(err.Error(), "invalid entity name syntax") {
		t.Errorf("Expected 'invalid entity name syntax' error, got: %v", err)
	}
}

func TestJSONEDDLoaderInvalidFieldName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "test",
				"fields": [
					{"name": "value.", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for invalid field name")
	}
	errs := loader.GetErrors()
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "invalid field name syntax") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'invalid field name syntax' error in collected errors")
	}
}

func TestJSONEDDLoaderGetErrors(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	if len(loader.GetErrors()) != 0 {
		t.Error("Expected no errors initially")
	}

	jsonData := `{
		"entities": [
			{
				"name": "test",
				"access": "rw",
				"fields": [
					{"name": "bad", "type": "invalidtype", "access": "rw"}
				]
			}
		]
	}`

	loader.Load(strings.NewReader(jsonData))

	errs := loader.GetErrors()
	if len(errs) == 0 {
		t.Error("Expected errors after loading invalid type")
	}
}

func TestJSONEDDLoaderWithSubtype(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "container",
				"access": "rw",
				"fields": [
					{"name": "items", "type": "array", "subtype": "string", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("container"))
	if ent == nil {
		t.Fatal("Expected container entity")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("items")) {
		t.Error("Expected 'items' attribute")
	}
}

func TestJSONEDDLoaderWithComment(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "documented",
				"access": "rw",
				"comment": "A well-documented entity",
				"fields": [
					{"name": "value", "type": "string", "access": "rw", "comment": "A value field"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("documented"))
	if ent == nil {
		t.Fatal("Expected documented entity")
	}
}

// Test singularize helper
// Test goValueToDTRulesObject
// Test JSONDataLoadError
// Test goValueToDTRulesObject with array values
// Test goValueToDTRulesObject with nested map values
// Test goValueToDTRulesObject with false boolean
// Test goValueToDTRulesObject with negative integer
// Test goValueToDTRulesObject with zero
// Test goValueToDTRulesObject with empty string
// Test goValueToDTRulesObject with empty array
// Test goValueToDTRulesObject with empty map
// Test JSONDataLoader size limit
// Test JSONDataLoader read error
// Test JSONDataLoader malformed JSON
// Test JSONDataLoader GetErrors and GetWarnings on fresh loader
// Test JSONDataLoadError with warnings
// Test singularize with additional edge cases
// Test NewJSONEDDLoader returns valid loader
func TestNewJSONEDDLoader(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)
	if loader == nil {
		t.Fatal("NewJSONEDDLoader returned nil")
	}
}

// Test NewJSONDataLoader returns valid loader
// Test JSON EDD loader with empty entity name
func TestJSONEDDLoaderEmptyEntityName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "",
				"fields": [
					{"name": "value", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for empty entity name")
	}
}

// Test JSON EDD loader with whitespace-only entity name
func TestJSONEDDLoaderWhitespaceEntityName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "   ",
				"fields": [
					{"name": "value", "type": "integer", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for whitespace-only entity name")
	}
}

// Test JSON EDD loader with entity that has no fields
func TestJSONEDDLoaderEntityNoFields(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "emptyentity",
				"access": "rw",
				"fields": []
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Entity with no fields should load without error: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("emptyentity"))
	if ent == nil {
		t.Fatal("Expected emptyentity to be created")
	}
}

// Test JSON EDD loader with all field types
func TestJSONEDDLoaderAllFieldTypes(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "alltypes",
				"access": "rw",
				"fields": [
					{"name": "str", "type": "string", "access": "rw"},
					{"name": "num", "type": "integer", "access": "rw"},
					{"name": "dbl", "type": "double", "access": "rw"},
					{"name": "flag", "type": "boolean", "access": "rw"},
					{"name": "dt", "type": "date", "access": "rw"},
					{"name": "ent", "type": "entity", "access": "rw"},
					{"name": "arr", "type": "array", "access": "rw"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD with all field types: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("alltypes"))
	if ent == nil {
		t.Fatal("Expected alltypes entity")
	}

	types := []string{"str", "num", "dbl", "flag", "dt", "ent", "arr"}
	for _, typ := range types {
		if !ent.ContainsAttribute(dtrules.GetRName(typ)) {
			t.Errorf("Expected '%s' attribute", typ)
		}
	}
}

// Test JSON EDD loader with invalid default values
func TestJSONEDDLoaderInvalidDefaultValues(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	// Invalid integer default should fall through to null
	jsonData := `{
		"entities": [
			{
				"name": "baddefaults",
				"access": "rw",
				"fields": [
					{"name": "intfield", "type": "integer", "access": "rw", "defaultValue": "notanumber"},
					{"name": "doublefield", "type": "double", "access": "rw", "defaultValue": "notadouble"},
					{"name": "boolfield", "type": "boolean", "access": "rw", "defaultValue": "notabool"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Should load even with bad defaults: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("baddefaults"))
	if ent == nil {
		t.Fatal("Expected baddefaults entity")
	}
}

// Test compatibility: verify JSON EDD produces same entities as XML EDD
func TestJSONEDDMatchesXMLEDD(t *testing.T) {
	// Load with XML
	xmlFactory := entity.NewFactory(nil)
	xmlLoader := NewEDDLoader(nil, xmlFactory)

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="person" access="rw">
    <field name="name" type="string" access="rw"/>
    <field name="age" type="integer" access="rw"/>
    <field name="salary" type="double" access="rw"/>
    <field name="active" type="boolean" access="r"/>
  </entity>
</entity_data_dictionary>`

	if err := xmlLoader.Load(strings.NewReader(xmlData)); err != nil {
		t.Fatalf("Failed to load XML EDD: %v", err)
	}

	// Load with JSON
	jsonFactory := entity.NewFactory(nil)
	jsonLoader := NewJSONEDDLoader(nil, jsonFactory)

	jsonData := `{
		"entities": [
			{
				"name": "person",
				"access": "rw",
				"fields": [
					{"name": "name", "type": "string", "access": "rw"},
					{"name": "age", "type": "integer", "access": "rw"},
					{"name": "salary", "type": "double", "access": "rw"},
					{"name": "active", "type": "boolean", "access": "r"}
				]
			}
		]
	}`

	if err := jsonLoader.Load(strings.NewReader(jsonData)); err != nil {
		t.Fatalf("Failed to load JSON EDD: %v", err)
	}

	// Verify both produce the same entity structure
	personName := dtrules.GetRName("person")

	xmlEntity, _ := xmlFactory.GetReferenceEntity(personName)
	jsonEntity, _ := jsonFactory.GetReferenceEntity(personName)

	if xmlEntity == nil || jsonEntity == nil {
		t.Fatal("Both factories should have person entity")
	}

	// Compare attribute names
	xmlAttrs := xmlEntity.GetAttributeNames()
	jsonAttrs := jsonEntity.GetAttributeNames()

	if len(xmlAttrs) != len(jsonAttrs) {
		t.Errorf("Attribute count mismatch: XML=%d, JSON=%d", len(xmlAttrs), len(jsonAttrs))
	}

	// Check that all XML attributes exist in JSON entity
	for _, attr := range xmlAttrs {
		if !jsonEntity.ContainsAttribute(attr) {
			t.Errorf("JSON entity missing attribute: %s", attr.StringValue())
		}
	}
}

// =============================================================================
// BigInt JSON Serialization/Deserialization Tests
// =============================================================================

// TestJSONEDDLoaderBigIntField tests that BigInt fields can be defined in JSON EDD
func TestJSONEDDLoaderBigIntField(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "blockchain",
				"access": "rw",
				"fields": [
					{"name": "block_number", "type": "bigint", "access": "rw"},
					{"name": "balance", "type": "biginteger", "access": "rw"},
					{"name": "hash_value", "type": "bigint", "access": "rw", "defaultValue": "12345678901234567890"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD with bigint fields: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("blockchain"))
	if ent == nil {
		t.Fatal("Expected blockchain entity")
	}

	// Check that BigInt fields are present
	if !ent.ContainsAttribute(dtrules.GetRName("block_number")) {
		t.Error("Expected 'block_number' attribute")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("balance")) {
		t.Error("Expected 'balance' attribute (using biginteger alias)")
	}
	if !ent.ContainsAttribute(dtrules.GetRName("hash_value")) {
		t.Error("Expected 'hash_value' attribute")
	}
}

// TestJSONEDDLoaderBigIntDefaultValue tests that BigInt default values are parsed correctly
func TestJSONEDDLoaderBigIntDefaultValue(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	// Test with a large number that exceeds int64
	largeValue := "123456789012345678901234567890"
	jsonData := `{
		"entities": [
			{
				"name": "bigtest",
				"access": "rw",
				"fields": [
					{"name": "large_num", "type": "bigint", "access": "rw", "defaultValue": "` + largeValue + `"}
				]
			}
		]
	}`

	err := loader.Load(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to load JSON EDD with bigint default value: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("bigtest"))
	if ent == nil {
		t.Fatal("Expected bigtest entity")
	}

	if !ent.ContainsAttribute(dtrules.GetRName("large_num")) {
		t.Error("Expected 'large_num' attribute")
	}
}

// TestJSONEDDLoaderFixedDefaultValue guards the TypeFixed arm of
// JSONEDDLoader.computeDefaultValue — fp fields declared in JSON EDD must
// pick up the default value as an RFixed on the 10^-8 grid (parallels
// the XML-EDD coverage in TestEDDLoaderFixedDefaultValue).
func TestJSONEDDLoaderFixedDefaultValue(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)

	jsonData := `{
		"entities": [
			{
				"name": "pool",
				"access": "rw",
				"fields": [
					{"name": "amount",   "type": "fixed", "access": "rw", "defaultValue": "1680748.45091643"},
					{"name": "rate",     "type": "fixed", "access": "rw", "defaultValue": "0.00250000"},
					{"name": "negative", "type": "fixed", "access": "rw", "defaultValue": "-0.00000001"}
				]
			}
		]
	}`

	if err := loader.Load(strings.NewReader(jsonData)); err != nil {
		t.Fatalf("Load JSON EDD with fixed fields: %v", err)
	}

	ent, _ := factory.GetReferenceEntity(dtrules.GetRName("pool"))
	if ent == nil {
		t.Fatal("expected pool entity")
	}
	expect := map[string]string{
		"amount":   "1680748.45091643",
		"rate":     "0.00250000",
		"negative": "-0.00000001",
	}
	for name, want := range expect {
		v, err := ent.Get(dtrules.GetRName(name))
		if err != nil {
			t.Errorf("Get(%s): %v", name, err)
			continue
		}
		fp, ok := v.(*dtrules.RFixed)
		if !ok {
			t.Errorf("%s: expected RFixed default, got %T", name, v)
			continue
		}
		if got := fp.StringValue(); got != want {
			t.Errorf("%s default: got %q, want %q", name, got, want)
		}
	}
}

// TestGoValueToDTRulesObjectLargeNumber tests that large numeric strings
// that exceed float64 precision are handled appropriately
// TestBigIntStringInput documents that large numbers passed as strings
// can be converted to BigInt properly
func TestBigIntStringInput(t *testing.T) {
	largeNum := "123456789012345678901234567890"

	// Convert string to RBigInt
	bi, err := dtrules.GetRBigIntFromString(largeNum)
	if err != nil {
		t.Fatalf("Failed to parse large number string: %v", err)
	}

	// Verify it maintains full precision
	if bi.StringValue() != largeNum {
		t.Errorf("Expected %s, got %s", largeNum, bi.StringValue())
	}

	// Verify it can't be converted to int64 (overflow)
	_, err = bi.LongValue()
	if err == nil {
		t.Error("Expected overflow error when converting large BigInt to int64")
	}
}

// TestBigIntJSONSerialization documents that BigInt serializes to string
// to preserve precision in JSON
func TestBigIntJSONSerialization(t *testing.T) {
	largeNum := "123456789012345678901234567890"
	bi, err := dtrules.GetRBigIntFromString(largeNum)
	if err != nil {
		t.Fatalf("Failed to create BigInt: %v", err)
	}

	// When converted to string for JSON output, precision is preserved
	serialized := bi.StringValue()
	if serialized != largeNum {
		t.Errorf("Expected %s, got %s", largeNum, serialized)
	}

	// Can be round-tripped
	roundTripped, err := dtrules.GetRBigIntFromString(serialized)
	if err != nil {
		t.Fatalf("Failed to round-trip BigInt: %v", err)
	}

	eq, err := bi.Equals(roundTripped)
	if err != nil {
		t.Fatalf("Equals failed: %v", err)
	}
	if !eq {
		t.Error("Round-tripped BigInt not equal to original")
	}
}

// TestBigIntFromVariousTypes documents BigInt conversion from various input types
func TestBigIntFromVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"from int64 string", "42", "42"},
		{"from large string", "99999999999999999999", "99999999999999999999"},
		{"from negative string", "-12345678901234567890", "-12345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi, err := dtrules.GetRBigIntFromString(tt.input.(string))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}
}

// TestJSONNumberPrecisionLoss documents that JSON numbers (float64) can lose precision
// for very large integers, motivating the use of strings for BigInt values
func TestJSONNumberPrecisionLoss(t *testing.T) {
	// This is a known limitation of JSON: numbers are float64
	// Integers larger than 2^53-1 (9007199254740991) may lose precision

	// Example: 9007199254740993 loses precision when parsed as float64
	// float64(9007199254740993) == float64(9007199254740992)
	f := float64(9007199254740993)
	if f == float64(9007199254740992) {
		// This demonstrates precision loss
		t.Log("As expected: float64 loses precision for integers > 2^53-1")
	}

	// When receiving large numbers from JSON, they should be sent as strings
	// to preserve precision, then converted to BigInt
	largeAsString := "9007199254740993"
	bi, err := dtrules.GetRBigIntFromString(largeAsString)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if bi.StringValue() != largeAsString {
		t.Errorf("Expected %s, got %s", largeAsString, bi.StringValue())
	}
}

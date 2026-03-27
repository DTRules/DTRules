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

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
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
func TestSingularize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"orders", "order"},
		{"entities", "entity"},
		{"boxes", "box"},
		{"classes", "class"},
		{"person", "person"},
		{"s", "s"},
		{"", ""},
		{"items", "item"},
		{"addresses", "address"},
	}

	for _, tt := range tests {
		result := singularize(tt.input)
		if result != tt.expected {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test goValueToDTRulesObject
func TestGoValueToDTRulesObject(t *testing.T) {
	// nil
	obj := goValueToDTRulesObject(nil)
	if obj.Type() != dtrules.TypeNull {
		t.Errorf("Expected null for nil, got %v", obj.Type())
	}

	// bool
	obj = goValueToDTRulesObject(true)
	if v, err := obj.BooleanValue(); err != nil || !v {
		t.Errorf("Expected true boolean, got %v", obj)
	}

	// integer (float64 that is a whole number)
	obj = goValueToDTRulesObject(float64(42))
	if v, err := obj.IntValue(); err != nil || v != 42 {
		t.Errorf("Expected integer 42, got %v", obj)
	}

	// double (float64 with decimal)
	obj = goValueToDTRulesObject(float64(3.14))
	if v, err := obj.DoubleValue(); err != nil || v != 3.14 {
		t.Errorf("Expected double 3.14, got %v", obj)
	}

	// string
	obj = goValueToDTRulesObject("hello")
	if obj.StringValue() != "hello" {
		t.Errorf("Expected string 'hello', got %v", obj.StringValue())
	}
}

// Test JSONDataLoadError
func TestJSONDataLoadError(t *testing.T) {
	// Single error
	singleErr := &JSONDataLoadError{
		Errors: []error{errors.New("single error")},
	}
	if !strings.Contains(singleErr.Error(), "single error") {
		t.Errorf("Expected single error message: %s", singleErr.Error())
	}

	// Multiple errors
	multiErr := &JSONDataLoadError{
		Errors: []error{errors.New("first"), errors.New("second")},
	}
	if !strings.Contains(multiErr.Error(), "2 errors") {
		t.Errorf("Expected multiple error message: %s", multiErr.Error())
	}
	if !strings.Contains(multiErr.Error(), "first") {
		t.Errorf("Expected first error in message: %s", multiErr.Error())
	}

	// Unwrap
	innerErr := errors.New("inner")
	unwrapErr := &JSONDataLoadError{
		Errors: []error{innerErr},
	}
	if unwrapErr.Unwrap() != innerErr {
		t.Error("Expected Unwrap to return first error")
	}

	// Unwrap empty
	emptyErr := &JSONDataLoadError{}
	if emptyErr.Unwrap() != nil {
		t.Error("Expected Unwrap to return nil for empty errors")
	}
}

// Test goValueToDTRulesObject with array values
func TestGoValueToDTRulesObjectArray(t *testing.T) {
	arr := []interface{}{"a", float64(1), true}
	obj := goValueToDTRulesObject(arr)
	if obj == nil {
		t.Fatal("Expected non-nil object for array")
	}
	// The result should be an RArray containing 3 items
	sv := obj.StringValue()
	if sv == "" {
		t.Error("Expected non-empty string value for array")
	}
}

// Test goValueToDTRulesObject with nested map values
func TestGoValueToDTRulesObjectMap(t *testing.T) {
	m := map[string]interface{}{"key": "value", "num": float64(42)}
	obj := goValueToDTRulesObject(m)
	if obj == nil {
		t.Fatal("Expected non-nil object for map")
	}
	// Maps are serialized as JSON strings
	sv := obj.StringValue()
	if !strings.Contains(sv, "key") || !strings.Contains(sv, "value") {
		t.Errorf("Expected JSON string containing 'key' and 'value', got: %s", sv)
	}
}

// Test goValueToDTRulesObject with false boolean
func TestGoValueToDTRulesObjectFalse(t *testing.T) {
	obj := goValueToDTRulesObject(false)
	v, err := obj.BooleanValue()
	if err != nil || v {
		t.Errorf("Expected false boolean, got %v (err: %v)", v, err)
	}
}

// Test goValueToDTRulesObject with negative integer
func TestGoValueToDTRulesObjectNegativeInt(t *testing.T) {
	obj := goValueToDTRulesObject(float64(-5))
	v, err := obj.IntValue()
	if err != nil || v != -5 {
		t.Errorf("Expected -5, got %d (err: %v)", v, err)
	}
}

// Test goValueToDTRulesObject with zero
func TestGoValueToDTRulesObjectZero(t *testing.T) {
	obj := goValueToDTRulesObject(float64(0))
	v, err := obj.IntValue()
	if err != nil || v != 0 {
		t.Errorf("Expected 0, got %d (err: %v)", v, err)
	}
}

// Test goValueToDTRulesObject with empty string
func TestGoValueToDTRulesObjectEmptyString(t *testing.T) {
	obj := goValueToDTRulesObject("")
	if obj.StringValue() != "" {
		t.Errorf("Expected empty string, got %q", obj.StringValue())
	}
}

// Test goValueToDTRulesObject with empty array
func TestGoValueToDTRulesObjectEmptyArray(t *testing.T) {
	arr := []interface{}{}
	obj := goValueToDTRulesObject(arr)
	if obj == nil {
		t.Fatal("Expected non-nil object for empty array")
	}
}

// Test goValueToDTRulesObject with empty map
func TestGoValueToDTRulesObjectEmptyMap(t *testing.T) {
	m := map[string]interface{}{}
	obj := goValueToDTRulesObject(m)
	if obj == nil {
		t.Fatal("Expected non-nil object for empty map")
	}
	sv := obj.StringValue()
	if sv != "{}" {
		t.Errorf("Expected '{}', got %q", sv)
	}
}

// Test JSONDataLoader size limit
func TestJSONDataLoaderSizeLimit(t *testing.T) {
	originalMax := MaxJSONSize
	defer func() { MaxJSONSize = originalMax }()

	MaxJSONSize = 10

	loader := NewJSONDataLoader(nil, nil)

	// This JSON is definitely larger than 10 bytes
	jsonData := `{"person": {"name": "John", "age": 30}}`
	err := loader.Load(strings.NewReader(jsonData))
	if err == nil {
		t.Fatal("Expected error for oversized JSON data")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

// Test JSONDataLoader read error
func TestJSONDataLoaderReadError(t *testing.T) {
	loader := NewJSONDataLoader(nil, nil)
	err := loader.Load(&errorReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

// Test JSONDataLoader malformed JSON
func TestJSONDataLoaderMalformedJSON(t *testing.T) {
	loader := NewJSONDataLoader(nil, nil)
	err := loader.Load(strings.NewReader(`{"broken`))
	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// Test JSONDataLoader GetErrors and GetWarnings on fresh loader
func TestJSONDataLoaderGettersEmpty(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONDataLoader(nil, factory)

	errs := loader.GetErrors()
	if len(errs) != 0 {
		t.Errorf("Expected no errors initially, got %d", len(errs))
	}

	warnings := loader.GetWarnings()
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings initially, got %d", len(warnings))
	}
}

// Test JSONDataLoadError with warnings
func TestJSONDataLoadErrorWithWarnings(t *testing.T) {
	err := &JSONDataLoadError{
		Errors:   []error{errors.New("err1")},
		Warnings: []string{"warn1", "warn2"},
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
}

// Test singularize with additional edge cases
func TestSingularizeEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a", "a"},                // Single character
		{"ss", "ss"},             // Double s (not plural)
		{"boss", "boss"},         // Ends in ss
		{"buses", "bus"},         // Ends in ses
		{"foxes", "fox"},         // Ends in xes
		{"babies", "baby"},       // Ends in ies
		{"policies", "policy"},   // Ends in ies
		{"Categories", "Category"}, // Case preserved
	}

	for _, tt := range tests {
		result := singularize(tt.input)
		if result != tt.expected {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Test NewJSONEDDLoader returns valid loader
func TestNewJSONEDDLoader(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONEDDLoader(nil, factory)
	if loader == nil {
		t.Fatal("NewJSONEDDLoader returned nil")
	}
}

// Test NewJSONDataLoader returns valid loader
func TestNewJSONDataLoader(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewJSONDataLoader(nil, factory)
	if loader == nil {
		t.Fatal("NewJSONDataLoader returned nil")
	}
}

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

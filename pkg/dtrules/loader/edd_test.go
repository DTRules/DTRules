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

func TestEDDLoaderBasic(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="person" access="rw">
    <field name="name" type="string" access="rw"/>
    <field name="age" type="integer" access="rw"/>
    <field name="active" type="boolean" access="r"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Verify entity was created
	personName := dtrules.GetRName("person")
	personEntity, _ := factory.GetReferenceEntity(personName)
	if personEntity == nil {
		t.Fatal("Expected person entity to be created")
	}

	// Verify attributes
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

func TestEDDLoaderMultipleEntities(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="order" access="rw">
    <field name="order_id" type="integer" access="rw"/>
    <field name="total" type="double" access="rw"/>
  </entity>
  <entity name="customer" access="rw">
    <field name="customer_id" type="integer" access="rw"/>
    <field name="email" type="string" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Verify both entities were created
	orderEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("order"))
	if orderEntity == nil {
		t.Error("Expected order entity")
	}

	customerEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("customer"))
	if customerEntity == nil {
		t.Error("Expected customer entity")
	}
}

func TestEDDLoaderMalformedXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0"?>
<entity_data_dictionary>
  <entity name="broken"
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for malformed XML")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestEDDLoaderUnknownType(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="test" access="rw">
    <field name="field1" type="unknowntype" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
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

func TestEDDLoaderEmptyXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Empty EDD should load without error: %v", err)
	}
}

func TestEDDLoaderDefaultValues(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="defaults" access="rw">
    <field name="count" type="integer" access="rw" default_value="42"/>
    <field name="rate" type="double" access="rw" default_value="3.14"/>
    <field name="flag" type="boolean" access="rw" default_value="true"/>
    <field name="label" type="string" access="rw" default_value="hello"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	entity, _ := factory.GetReferenceEntity(dtrules.GetRName("defaults"))
	if entity == nil {
		t.Fatal("Expected defaults entity")
	}

	// Check default values through attribute entries
	if !entity.ContainsAttribute(dtrules.GetRName("count")) {
		t.Error("Expected 'count' attribute")
	}
}

func TestEDDLoaderFieldAccess(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="accesstest" access="rw">
    <field name="readonly" type="string" access="r"/>
    <field name="writeonly" type="string" access="w"/>
    <field name="readwrite" type="string" access="rw"/>
    <field name="nospec" type="string"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	entity, _ := factory.GetReferenceEntity(dtrules.GetRName("accesstest"))
	if entity == nil {
		t.Fatal("Expected accesstest entity")
	}

	// Verify all fields exist
	fields := []string{"readonly", "writeonly", "readwrite", "nospec"}
	for _, f := range fields {
		if !entity.ContainsAttribute(dtrules.GetRName(f)) {
			t.Errorf("Expected '%s' attribute", f)
		}
	}
}

func TestEDDLoaderReadError(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	// Use a reader that returns an error
	err := loader.Load(&errorReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func TestEDDLoadErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	loadErr := &EDDLoadError{
		Errors: []error{innerErr, errors.New("second error")},
	}

	// Test Error() method
	errStr := loadErr.Error()
	if !strings.Contains(errStr, "inner error") {
		t.Errorf("Expected error message to contain inner error: %s", errStr)
	}

	// Test Unwrap()
	unwrapped := loadErr.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Expected Unwrap to return first error")
	}

	// Test errors.Is
	if !errors.Is(loadErr, innerErr) {
		t.Error("Expected errors.Is to match inner error")
	}
}

func TestEDDLoadErrorSingleError(t *testing.T) {
	innerErr := errors.New("single error")
	loadErr := &EDDLoadError{
		Errors: []error{innerErr},
	}

	errStr := loadErr.Error()
	if !strings.Contains(errStr, "single error") {
		t.Errorf("Expected single error message format: %s", errStr)
	}
	if strings.Contains(errStr, "errors") {
		t.Errorf("Single error should not use plural: %s", errStr)
	}
}

func TestGetErrors(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	// Initially no errors
	if len(loader.GetErrors()) != 0 {
		t.Error("Expected no errors initially")
	}

	// Load something with errors
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="test" access="rw">
    <field name="bad" type="invalidtype" access="rw"/>
  </entity>
</entity_data_dictionary>`

	loader.Load(strings.NewReader(xml))

	errs := loader.GetErrors()
	if len(errs) == 0 {
		t.Error("Expected errors after loading invalid type")
	}
}

func TestEDDLoaderInvalidEntityName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	// Entity name with invalid syntax (leading dot)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
    <entity name=".invalid">
        <field name="value" type="integer" access="rw"/>
    </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for invalid entity name")
	}
	if !strings.Contains(err.Error(), "invalid entity name syntax") {
		t.Errorf("Expected 'invalid entity name syntax' error, got: %v", err)
	}
}

func TestEDDLoaderInvalidFieldName(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	// Field name with invalid syntax (trailing dot)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
    <entity name="test">
        <field name="value." type="integer" access="rw"/>
    </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for invalid field name")
	}
	// The error is collected, so we check via GetErrors
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

func TestEDDLoaderXMLSizeLimit(t *testing.T) {
	// Save original value and restore after test
	originalMax := MaxXMLSize
	defer func() { MaxXMLSize = originalMax }()

	// Set a small limit for testing
	MaxXMLSize = 100

	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	// Create XML larger than the limit
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
    <entity name="test">
        <field name="value" type="integer" access="rw" comment="This comment makes the XML larger than the limit"/>
    </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for oversized XML")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

func TestEDDLoaderXMLSizeLimitDisabled(t *testing.T) {
	// Save original value and restore after test
	originalMax := MaxXMLSize
	defer func() { MaxXMLSize = originalMax }()

	// Disable the limit
	MaxXMLSize = 0

	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
    <entity name="test">
        <field name="value" type="integer" access="rw"/>
    </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Expected no error with size limit disabled: %v", err)
	}
}

// =============================================================================
// BigInt Field Tests
// =============================================================================

func TestEDDLoaderBigIntField(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="budget" access="rw">
    <field name="supply_limit" type="bigint" access="rw"/>
    <field name="amount_issued" type="biginteger" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with bigint fields: %v", err)
	}

	// Verify entity was created
	budgetEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("budget"))
	if budgetEntity == nil {
		t.Fatal("Expected budget entity to be created")
	}

	// Verify bigint attributes exist
	if !budgetEntity.ContainsAttribute(dtrules.GetRName("supply_limit")) {
		t.Error("Expected 'supply_limit' bigint attribute")
	}

	if !budgetEntity.ContainsAttribute(dtrules.GetRName("amount_issued")) {
		t.Error("Expected 'amount_issued' biginteger attribute")
	}
}

func TestEDDLoaderBigIntDefaultValue(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="budget" access="rw">
    <field name="small_value" type="bigint" access="rw" default_value="12345"/>
    <field name="large_value" type="bigint" access="rw" default_value="99999999999999999999999999999999"/>
    <field name="negative_value" type="bigint" access="rw" default_value="-50000000000000000000"/>
    <field name="zero_value" type="bigint" access="rw" default_value="0"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with bigint default values: %v", err)
	}

	budgetEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("budget"))
	if budgetEntity == nil {
		t.Fatal("Expected budget entity to be created")
	}

	// Verify all fields were created
	fields := []string{"small_value", "large_value", "negative_value", "zero_value"}
	for _, f := range fields {
		if !budgetEntity.ContainsAttribute(dtrules.GetRName(f)) {
			t.Errorf("Expected '%s' bigint attribute", f)
		}
	}
}

func TestEDDLoaderBigIntInvalidDefaultValue(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="test" access="rw">
    <field name="bad_value" type="bigint" access="rw" default_value="not-a-number"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	// The loader should still succeed but may log a warning or use default
	// Check that the entity was created even with invalid default
	if err != nil {
		// If error is returned, that's also acceptable behavior
		return
	}

	testEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("test"))
	if testEntity == nil {
		t.Fatal("Expected test entity to be created")
	}
}

func TestEDDLoaderBigIntWithOtherTypes(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="mixed" access="rw">
    <field name="name" type="string" access="rw" default_value="test"/>
    <field name="count" type="integer" access="rw" default_value="42"/>
    <field name="balance" type="bigint" access="rw" default_value="50000000000000000000"/>
    <field name="rate" type="double" access="rw" default_value="3.14"/>
    <field name="active" type="boolean" access="rw" default_value="true"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with mixed types including bigint: %v", err)
	}

	mixedEntity, _ := factory.GetReferenceEntity(dtrules.GetRName("mixed"))
	if mixedEntity == nil {
		t.Fatal("Expected mixed entity to be created")
	}

	// Verify all fields exist
	fields := []string{"name", "count", "balance", "rate", "active"}
	for _, f := range fields {
		if !mixedEntity.ContainsAttribute(dtrules.GetRName(f)) {
			t.Errorf("Expected '%s' attribute", f)
		}
	}
}

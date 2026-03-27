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

package entity

import (
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// mockSession implements minimal Session for testing
type mockSession struct {
	uniqueID int
}

func (m *mockSession) GetState() dtrules.State                 { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory { return nil }
func (m *mockSession) GetUniqueID() int                        { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return nil, nil
}
func (m *mockSession) Compile(expr string) (dtrules.Object, error) {
	return nil, nil
}

func TestEntityFactory(t *testing.T) {
	factory := NewFactory(nil)
	if factory == nil {
		t.Fatal("NewFactory returned nil")
	}

	// Create a reference entity
	entityName := dtrules.GetRName("person")
	refEntity, err := factory.FindCreateRefEntity(false, entityName)
	if err != nil {
		t.Fatalf("FindCreateRefEntity failed: %v", err)
	}
	if refEntity == nil {
		t.Fatal("Reference entity is nil")
	}

	// Verify we get the same entity back
	refEntity2, err := factory.FindCreateRefEntity(false, entityName)
	if err != nil {
		t.Fatalf("FindCreateRefEntity failed: %v", err)
	}
	if refEntity != refEntity2 {
		t.Error("Expected same reference entity instance")
	}

	// Get entity names
	names := factory.GetEntityNames()
	if len(names) != 1 {
		t.Errorf("Expected 1 entity name, got %d", len(names))
	}
}

func TestEntityAttributes(t *testing.T) {
	factory := NewFactory(nil)
	entityName := dtrules.GetRName("person")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	// Add attributes
	attrName := dtrules.GetRName("name")
	errStr := refEntity.AddAttribute(
		attrName,
		"",
		dtrules.NewRString(""),
		true, // writable
		true, // readable
		dtrules.TypeString,
		"",
		"Person's name",
		"",
		"",
	)
	if errStr != "" {
		t.Fatalf("AddAttribute failed: %s", errStr)
	}

	attrAge := dtrules.GetRName("age")
	errStr = refEntity.AddAttribute(
		attrAge,
		"0",
		dtrules.GetRIntegerValue(0),
		true,
		true,
		dtrules.TypeInteger,
		"",
		"Person's age",
		"",
		"",
	)
	if errStr != "" {
		t.Fatalf("AddAttribute failed: %s", errStr)
	}

	// Verify attribute exists
	if !refEntity.ContainsAttribute(attrName) {
		t.Error("Entity should contain 'name' attribute")
	}
	if !refEntity.ContainsAttribute(attrAge) {
		t.Error("Entity should contain 'age' attribute")
	}
	if refEntity.ContainsAttribute(dtrules.GetRName("nonexistent")) {
		t.Error("Entity should not contain 'nonexistent' attribute")
	}
}

func TestEntityInstance(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	// Create reference entity with attributes
	entityName := dtrules.GetRName("person")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	nameAttr := dtrules.GetRName("name")
	refEntity.AddAttribute(nameAttr, "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")

	ageAttr := dtrules.GetRName("age")
	refEntity.AddAttribute(ageAttr, "0", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	// Create instance
	instance, err := factory.CreateEntity(session, entityName)
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	if instance == nil {
		t.Fatal("Instance is nil")
	}

	// Verify instance has unique ID
	if instance.GetID() == 0 {
		t.Error("Instance should have non-zero ID")
	}

	// Set and get values
	err = instance.Put(nameAttr, dtrules.NewRString("Alice"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, err := instance.Get(nameAttr)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value.StringValue() != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", value.StringValue())
	}

	// Set age
	err = instance.Put(ageAttr, dtrules.GetRIntegerValue(30))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, err = instance.Get(ageAttr)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	val, _ := value.IntValue()
	if val != 30 {
		t.Errorf("Expected 30, got %d", val)
	}
}

func TestEntityClone(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	// Create reference entity with attributes
	entityName := dtrules.GetRName("person")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	nameAttr := dtrules.GetRName("name")
	refEntity.AddAttribute(nameAttr, "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")

	// Create instance and set value
	instance, _ := factory.CreateEntity(session, entityName)
	instance.Put(nameAttr, dtrules.NewRString("Alice"))

	// Clone the instance
	cloned, err := instance.Clone(session)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	clonedEntity, ok := cloned.(dtrules.Entity)
	if !ok {
		t.Fatal("Cloned object is not an Entity")
	}

	// Cloned entity should have different ID
	if clonedEntity.GetID() == instance.GetID() {
		t.Error("Cloned entity should have different ID")
	}

	// Cloned entity should have same value
	value, _ := clonedEntity.Get(nameAttr)
	if value.StringValue() != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", value.StringValue())
	}

	// Modifying clone should not affect original
	clonedEntity.Put(nameAttr, dtrules.NewRString("Bob"))

	originalValue, _ := instance.Get(nameAttr)
	if originalValue.StringValue() != "Alice" {
		t.Error("Modifying clone affected original entity")
	}

	clonedValue, _ := clonedEntity.Get(nameAttr)
	if clonedValue.StringValue() != "Bob" {
		t.Error("Clone value not updated correctly")
	}
}

func TestMultipleInstances(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	// Create reference entity
	entityName := dtrules.GetRName("counter")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	valueAttr := dtrules.GetRName("value")
	refEntity.AddAttribute(valueAttr, "0", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	// Create multiple instances
	instance1, _ := factory.CreateEntity(session, entityName)
	instance2, _ := factory.CreateEntity(session, entityName)

	// Set different values
	instance1.Put(valueAttr, dtrules.GetRIntegerValue(100))
	instance2.Put(valueAttr, dtrules.GetRIntegerValue(200))

	// Verify values are independent
	val1, _ := instance1.Get(valueAttr)
	val2, _ := instance2.Get(valueAttr)

	v1, _ := val1.IntValue()
	v2, _ := val2.IntValue()

	if v1 != 100 {
		t.Errorf("Instance 1 should have value 100, got %d", v1)
	}
	if v2 != 200 {
		t.Errorf("Instance 2 should have value 200, got %d", v2)
	}

	// Verify different IDs
	if instance1.GetID() == instance2.GetID() {
		t.Error("Instances should have different IDs")
	}
}

func TestEntityType(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("test")
	factory.FindCreateRefEntity(false, entityName)

	instance, _ := factory.CreateEntity(session, entityName)

	if instance.Type() != dtrules.TypeEntity {
		t.Errorf("Expected TypeEntity, got %v", instance.Type())
	}
}

func TestEntityGetName(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("myentity")
	factory.FindCreateRefEntity(false, entityName)

	instance, _ := factory.CreateEntity(session, entityName)

	if instance.GetName() != entityName {
		t.Errorf("Expected name 'myentity', got '%s'", instance.GetName().StringValue())
	}
}

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

	"github.com/DTRules/DTRules/pkg/dtrules"
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
func (m *mockSession) GetEntityByID(id int) dtrules.Entity { return nil }

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

// TestEntityPutFixedCoercion guards the TypeFixed branch of REntity.Put:
// integer and bigint values assigned to a fixed-typed field must coerce to
// RFixed via PromoteToRFixed; double values must error per the "no silent
// down-coercion" principle shared with the bigint wiring (#675).
func TestEntityPutFixedCoercion(t *testing.T) {
	factory := NewFactory(nil)
	entityName := dtrules.GetRName("pool")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	zero, _ := dtrules.GetRFixedFromInt64(0)
	refEntity.AddAttribute(dtrules.GetRName("amount"), "",
		zero, true, true, dtrules.TypeFixed, "", "", "", "")

	amountName := dtrules.GetRName("amount")

	t.Run("int promotes to RFixed", func(t *testing.T) {
		if err := refEntity.Put(amountName, dtrules.GetRIntegerValue(42)); err != nil {
			t.Fatalf("Put integer: %v", err)
		}
		v, _ := refEntity.Get(amountName)
		fp, ok := v.(*dtrules.RFixed)
		if !ok {
			t.Fatalf("stored value should be RFixed, got %T", v)
		}
		if got := fp.StringValue(); got != "42.00000000" {
			t.Errorf("int 42 → %q, want 42.00000000", got)
		}
	})

	t.Run("bigint promotes to RFixed", func(t *testing.T) {
		if err := refEntity.Put(amountName, dtrules.GetRBigIntFromInt64(1_000_000)); err != nil {
			t.Fatalf("Put bigint: %v", err)
		}
		v, _ := refEntity.Get(amountName)
		fp, ok := v.(*dtrules.RFixed)
		if !ok {
			t.Fatalf("stored value should be RFixed, got %T", v)
		}
		if got := fp.StringValue(); got != "1000000.00000000" {
			t.Errorf("bigint 1e6 → %q, want 1000000.00000000", got)
		}
	})

	t.Run("double is rejected — requires explicit cvfp", func(t *testing.T) {
		err := refEntity.Put(amountName, dtrules.GetRDoubleValue(1.5))
		if err == nil {
			t.Fatal("Put of double into fixed field must error")
		}
	})

	t.Run("RFixed passes through unchanged", func(t *testing.T) {
		fp, _ := dtrules.GetRFixedFromString("3.14159265")
		if err := refEntity.Put(amountName, fp); err != nil {
			t.Fatalf("Put fp: %v", err)
		}
		v, _ := refEntity.Get(amountName)
		got, ok := v.(*dtrules.RFixed)
		if !ok {
			t.Fatalf("stored value should be RFixed, got %T", v)
		}
		if got.StringValue() != "3.14159265" {
			t.Errorf("fp passthrough: got %q, want 3.14159265", got.StringValue())
		}
	})
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

// =============================================================================
// BigInt Field Tests
// =============================================================================

func TestEntityBigIntField(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	// Create reference entity with bigint attribute
	entityName := dtrules.GetRName("budget")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	balanceAttr := dtrules.GetRName("balance")
	refEntity.AddAttribute(balanceAttr, "0", dtrules.GetRBigIntFromInt64(0), true, true, dtrules.TypeBigInt, "", "Balance in nanoUnits", "", "")

	// Create instance
	instance, err := factory.CreateEntity(session, entityName)
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}

	// Set a BigInt value
	bigValue := dtrules.GetRBigIntFromInt64(1000000000000)
	err = instance.Put(balanceAttr, bigValue)
	if err != nil {
		t.Fatalf("Put BigInt failed: %v", err)
	}

	// Get and verify
	value, err := instance.Get(balanceAttr)
	if err != nil {
		t.Fatalf("Get BigInt failed: %v", err)
	}

	bi, err := value.RBigIntValue()
	if err != nil {
		t.Fatalf("RBigIntValue failed: %v", err)
	}

	if bi.StringValue() != "1000000000000" {
		t.Errorf("Expected 1000000000000, got %s", bi.StringValue())
	}
}

func TestEntityBigIntFieldLargeValue(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("budget")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	supplyAttr := dtrules.GetRName("supply_limit")
	refEntity.AddAttribute(supplyAttr, "0", dtrules.GetRBigIntFromInt64(0), true, true, dtrules.TypeBigInt, "", "", "", "")

	instance, _ := factory.CreateEntity(session, entityName)

	// Set a very large BigInt value (larger than int64)
	largeValue, _ := dtrules.GetRBigIntFromString("50000000000000000000")
	err := instance.Put(supplyAttr, largeValue)
	if err != nil {
		t.Fatalf("Put large BigInt failed: %v", err)
	}

	value, _ := instance.Get(supplyAttr)
	bi, _ := value.RBigIntValue()

	if bi.StringValue() != "50000000000000000000" {
		t.Errorf("Expected 50000000000000000000, got %s", bi.StringValue())
	}
}

func TestEntityBigIntCoercionFromInteger(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("account")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	amountAttr := dtrules.GetRName("amount")
	refEntity.AddAttribute(amountAttr, "0", dtrules.GetRBigIntFromInt64(0), true, true, dtrules.TypeBigInt, "", "", "", "")

	instance, _ := factory.CreateEntity(session, entityName)

	// Put an RInteger - should be coerced to BigInt
	intValue := dtrules.GetRIntegerValue(12345)
	err := instance.Put(amountAttr, intValue)
	if err != nil {
		t.Fatalf("Put RInteger to BigInt field failed: %v", err)
	}

	value, _ := instance.Get(amountAttr)

	// Should be a BigInt now
	if value.Type() != dtrules.TypeBigInt {
		t.Errorf("Expected TypeBigInt, got %v", value.Type())
	}

	bi, _ := value.RBigIntValue()
	if bi.StringValue() != "12345" {
		t.Errorf("Expected 12345, got %s", bi.StringValue())
	}
}

func TestEntityBigIntCoercionFromString(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("account")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	amountAttr := dtrules.GetRName("amount")
	refEntity.AddAttribute(amountAttr, "0", dtrules.GetRBigIntFromInt64(0), true, true, dtrules.TypeBigInt, "", "", "", "")

	instance, _ := factory.CreateEntity(session, entityName)

	// Put an RString - should be coerced to BigInt
	strValue := dtrules.NewRString("99999999999999999999")
	err := instance.Put(amountAttr, strValue)
	if err != nil {
		t.Fatalf("Put RString to BigInt field failed: %v", err)
	}

	value, _ := instance.Get(amountAttr)
	bi, _ := value.RBigIntValue()

	if bi.StringValue() != "99999999999999999999" {
		t.Errorf("Expected 99999999999999999999, got %s", bi.StringValue())
	}
}

func TestEntityBigIntCoercionFromDouble(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("account")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	amountAttr := dtrules.GetRName("amount")
	refEntity.AddAttribute(amountAttr, "0", dtrules.GetRBigIntFromInt64(0), true, true, dtrules.TypeBigInt, "", "", "", "")

	instance, _ := factory.CreateEntity(session, entityName)

	// Put an RDouble - should be coerced to BigInt (truncated)
	doubleValue := dtrules.GetRDoubleValue(123.99)
	err := instance.Put(amountAttr, doubleValue)
	if err != nil {
		t.Fatalf("Put RDouble to BigInt field failed: %v", err)
	}

	value, _ := instance.Get(amountAttr)
	bi, _ := value.RBigIntValue()

	// Should be truncated to 123
	if bi.StringValue() != "123" {
		t.Errorf("Expected 123 (truncated), got %s", bi.StringValue())
	}
}

func TestEntityBigIntDefaultValue(t *testing.T) {
	factory := NewFactory(nil)
	session := &mockSession{}

	entityName := dtrules.GetRName("budget")
	refEntity, _ := factory.FindCreateRefEntity(false, entityName)

	// Set a default value
	defaultValue, _ := dtrules.GetRBigIntFromString("1000000000000000000")
	limitAttr := dtrules.GetRName("limit")
	refEntity.AddAttribute(limitAttr, "1000000000000000000", defaultValue, true, true, dtrules.TypeBigInt, "", "", "", "")

	// Create instance - should have default value
	instance, _ := factory.CreateEntity(session, entityName)

	value, err := instance.Get(limitAttr)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	bi, _ := value.RBigIntValue()
	if bi.StringValue() != "1000000000000000000" {
		t.Errorf("Expected default 1000000000000000000, got %s", bi.StringValue())
	}
}

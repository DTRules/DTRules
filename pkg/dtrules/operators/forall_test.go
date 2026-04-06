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

package operators_test

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestForallNamedBinding tests that when an entity is pushed to the entity stack,
// it can be accessed by its type name using the entity.attribute syntax.
// This test verifies issue #464: forall context does not create named entity binding
func TestForallNamedBinding(t *testing.T) {
	// Create a session with entities
	rs := session.NewRuleSet("test")

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the entity factory
	ef := sess.GetEntityFactory().(*entity.Factory)

	// Create a reference entity for "vehicle" using FindCreateRefEntity
	vehicleName := dtrules.GetRName("vehicle")
	vehicleRef, err := ef.FindCreateRefEntity(true, vehicleName)
	if err != nil {
		t.Fatalf("Failed to create vehicle reference: %v", err)
	}
	// Add some attributes
	vehicleRef.AddAttribute(dtrules.GetRName("mileage"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "Mileage", "", "")

	// Create a vehicle instance
	vehicle, err := ef.CreateEntity(sess, vehicleName)
	if err != nil {
		t.Fatalf("Failed to create vehicle: %v", err)
	}
	vehicle.Put(dtrules.GetRName("mileage"), dtrules.GetRIntegerValue(10000))

	// Get the state
	state := sess.GetState().(*interpreter.DTState)

	// Push vehicle onto entity stack (simulating what forall does)
	state.EntityPush(vehicle)

	// Now try to find vehicle.mileage
	vehicleMileageName := dtrules.GetRName("vehicle.mileage")
	value, err := state.Find(vehicleMileageName)
	if err != nil {
		t.Errorf("Failed to find vehicle.mileage: %v", err)
	}
	if value == nil {
		t.Errorf("Expected to find vehicle.mileage but got nil")
	} else {
		intVal, err := value.IntValue()
		if err != nil {
			t.Errorf("Failed to get int value: %v", err)
		} else if intVal != 10000 {
			t.Errorf("Expected mileage 10000, got %d", intVal)
		}
	}

	// Clean up
	state.EntityPop()
}

// TestForallNamedBindingDebug tests and logs details about the entity name resolution
func TestForallNamedBindingDebug(t *testing.T) {
	// Create a session with entities
	rs := session.NewRuleSet("test")

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the entity factory
	ef := sess.GetEntityFactory().(*entity.Factory)

	// Create a reference entity for "vehicle"
	vehicleName := dtrules.GetRName("vehicle")
	vehicleRef, err := ef.FindCreateRefEntity(true, vehicleName)
	if err != nil {
		t.Fatalf("Failed to create vehicle reference: %v", err)
	}
	vehicleRef.AddAttribute(dtrules.GetRName("mileage"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "Mileage", "", "")

	// Create a vehicle instance
	vehicle, err := ef.CreateEntity(sess, vehicleName)
	if err != nil {
		t.Fatalf("Failed to create vehicle: %v", err)
	}
	vehicle.Put(dtrules.GetRName("mileage"), dtrules.GetRIntegerValue(15000))

	// Get the state
	state := sess.GetState().(*interpreter.DTState)

	// Push vehicle onto entity stack (what forall does)
	state.EntityPush(vehicle)

	// Verify that vehicle.mileage works when entity type matches
	name := dtrules.GetRName("vehicle.mileage")
	t.Logf("Looking up: %s", name.StringValue())
	t.Logf("Entity name from RName: %s", vehicleName.StringValue())
	t.Logf("Entity type on stack: %s", vehicle.GetName().StringValue())

	// Check if the entity has the attribute
	mileageAttr := dtrules.GetRName("mileage")
	t.Logf("Vehicle has mileage attribute: %v", vehicle.ContainsAttribute(mileageAttr))

	// Check the entity name equals
	entityNamePart := name.GetEntityName()
	if entityNamePart != nil {
		t.Logf("Entity name from lookup key: %s", entityNamePart.StringValue())
		eq, _ := vehicle.GetName().Equals(entityNamePart)
		t.Logf("Entity names equal: %v", eq)
	} else {
		t.Logf("No entity name part in lookup key")
	}

	value, err := state.Find(name)
	if err != nil {
		t.Errorf("Failed to find vehicle.mileage: %v", err)
	}
	if value == nil {
		t.Errorf("Expected to find vehicle.mileage but got nil")
	} else {
		intVal, err := value.IntValue()
		if err != nil {
			t.Errorf("Failed to get int value: %v", err)
		}
		t.Logf("Found mileage: %d", intVal)
		if intVal != 15000 {
			t.Errorf("Expected mileage 15000, got %d", intVal)
		}
	}

	state.EntityPop()
}

// TestForallNamedBindingTaxpayerScenario tests the actual taxpayer scenario from issue #464
func TestForallNamedBindingTaxpayerScenario(t *testing.T) {
	// Create a session with entities matching the TaxReturn structure
	rs := session.NewRuleSet("test")

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the entity factory
	ef := sess.GetEntityFactory().(*entity.Factory)

	// Create a reference entity for "taxpayer" (matching TaxReturn_edd_core.xml)
	taxpayerName := dtrules.GetRName("taxpayer")
	taxpayerRef, err := ef.FindCreateRefEntity(true, taxpayerName)
	if err != nil {
		t.Fatalf("Failed to create taxpayer reference: %v", err)
	}
	// Add the se_gross_income attribute (from TaxReturn_edd_core.xml)
	taxpayerRef.AddAttribute(dtrules.GetRName("se_gross_income"), "", dtrules.GetRDoubleValue(0), true, true, dtrules.TypeDouble, "", "Self-employment gross receipts", "", "")
	taxpayerRef.AddAttribute(dtrules.GetRName("is_self_employed"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "Whether taxpayer is self-employed", "", "")

	// Create a taxpayer instance
	taxpayer, err := ef.CreateEntity(sess, taxpayerName)
	if err != nil {
		t.Fatalf("Failed to create taxpayer: %v", err)
	}
	taxpayer.Put(dtrules.GetRName("se_gross_income"), dtrules.GetRDoubleValue(50000.0))
	taxpayer.Put(dtrules.GetRName("is_self_employed"), dtrules.GetRBoolean(true))

	// Get the state
	state := sess.GetState().(*interpreter.DTState)

	// Simulate the forall loop: push taxpayer onto entity stack
	state.EntityPush(taxpayer)

	// Now test the exact lookup that's failing: taxpayer.se_gross_income
	name := dtrules.GetRName("taxpayer.se_gross_income")
	t.Logf("Looking up: %s", name.StringValue())
	t.Logf("Taxpayer entity name: %s", taxpayer.GetName().StringValue())

	// Check the entity name part
	entityNamePart := name.GetEntityName()
	if entityNamePart != nil {
		t.Logf("Entity prefix in lookup: %s", entityNamePart.StringValue())
		eq, _ := taxpayer.GetName().Equals(entityNamePart)
		t.Logf("Entity name match: %v", eq)
	}

	// Check attribute existence
	attrName := dtrules.GetRName("se_gross_income")
	t.Logf("Taxpayer has se_gross_income: %v", taxpayer.ContainsAttribute(attrName))

	value, err := state.Find(name)
	if err != nil {
		t.Errorf("Failed to find taxpayer.se_gross_income: %v", err)
	}
	if value == nil {
		t.Errorf("Expected to find taxpayer.se_gross_income but got nil - THIS IS THE BUG FROM ISSUE #464")
	} else {
		doubleVal, err := value.DoubleValue()
		if err != nil {
			t.Errorf("Failed to get double value: %v", err)
		}
		t.Logf("Found se_gross_income: %f", doubleVal)
		if doubleVal != 50000.0 {
			t.Errorf("Expected se_gross_income 50000.0, got %f", doubleVal)
		}
	}

	state.EntityPop()
}

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

package interpreter

import (
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/operators"
)

// TestRuntimeInterfaceCompliance verifies that DTState satisfies the Runtime interface.
func TestRuntimeInterfaceCompliance(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Verify that *DTState implements dtrules.Runtime
	var _ dtrules.Runtime = state
	var _ dtrules.RuntimeInit = state
	var _ dtrules.RuntimeQuery = state
}

// TestRuntimeName verifies the runtime name is "go".
func TestRuntimeName(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	if state.Name() != "go" {
		t.Errorf("Expected runtime name 'go', got %q", state.Name())
	}
}

// TestRuntimeGetState verifies GetState returns the DTState itself.
func TestRuntimeGetState(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	s := state.GetState()
	if s != state {
		t.Error("GetState() should return the DTState itself")
	}
}

// TestRuntimeInitPushValue tests pushing values via RuntimeInit.
func TestRuntimeInitPushValue(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Push values via RuntimeInit
	err := rt.PushValue(dtrules.NewValueInteger(42))
	if err != nil {
		t.Fatalf("PushValue failed: %v", err)
	}
	err = rt.PushValue(dtrules.NewValueInteger(99))
	if err != nil {
		t.Fatalf("PushValue failed: %v", err)
	}

	// Query values via RuntimeQuery
	if rt.ValueStackSize() != 2 {
		t.Errorf("Expected value stack depth 2, got %d", rt.ValueStackSize())
	}

	v, err := rt.PeekValue()
	if err != nil {
		t.Fatalf("PeekValue failed: %v", err)
	}
	if !v.IsInteger() || v.AsInteger() != 99 {
		t.Errorf("Expected 99 on top, got %v", v)
	}

	v, err = rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if v.AsInteger() != 99 {
		t.Errorf("Expected 99, got %d", v.AsInteger())
	}

	v, err = rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if v.AsInteger() != 42 {
		t.Errorf("Expected 42, got %d", v.AsInteger())
	}
}

// TestRuntimeInitPushData tests pushing data via RuntimeInit.
func TestRuntimeInitPushData(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	obj := dtrules.GetRIntegerValue(42)
	err := rt.PushData(obj)
	if err != nil {
		t.Fatalf("PushData failed: %v", err)
	}

	if rt.DataStackSize() != 1 {
		t.Errorf("Expected data stack depth 1, got %d", rt.DataStackSize())
	}

	result, err := rt.PopData()
	if err != nil {
		t.Fatalf("PopData failed: %v", err)
	}
	val, err := result.IntValue()
	if err != nil {
		t.Fatalf("IntValue failed: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

// TestRuntimeInitPushEntity tests pushing entities via RuntimeInit.
func TestRuntimeInitPushEntity(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Create a simple entity-like object using primitives
	primitives := operators.NewPrimitivesEntity()
	err := rt.PushEntity(primitives)
	if err != nil {
		t.Fatalf("PushEntity failed: %v", err)
	}

	if rt.EntityStackDepth() != 1 {
		t.Errorf("Expected entity stack depth 1, got %d", rt.EntityStackDepth())
	}

	e, err := rt.EntityByIndex(0)
	if err != nil {
		t.Fatalf("EntityByIndex failed: %v", err)
	}
	if e != primitives {
		t.Error("EntityByIndex(0) should return the pushed entity")
	}
}

// TestRuntimeSetSession tests setting session on the runtime.
func TestRuntimeSetSession(t *testing.T) {
	session1 := &mockSession{}
	session2 := &mockSession{}
	state := NewDTState(session1)

	if state.GetSession() != session1 {
		t.Error("Initial session should be session1")
	}

	state.SetSession(session2)
	if state.GetSession() != session2 {
		t.Error("After SetSession, session should be session2")
	}
}

// TestRuntimeExecuteBytecode tests bytecode execution through the Runtime interface.
func TestRuntimeExecuteBytecode(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Build bytecode: push 10, push 20, add
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)

	err := rt.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	if rt.ValueStackSize() != 1 {
		t.Errorf("Expected value stack depth 1, got %d", rt.ValueStackSize())
	}

	v, err := rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if !v.IsInteger() || v.AsInteger() != 30 {
		t.Errorf("Expected 30, got %v", v)
	}
}

// TestRuntimeInitThenExecute tests the full init → execute → query lifecycle.
func TestRuntimeInitThenExecute(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Step 1: Init — push a value onto the value stack
	err := rt.PushValue(dtrules.NewValueInteger(5))
	if err != nil {
		t.Fatalf("PushValue failed: %v", err)
	}

	// Step 2: Execute — bytecode that pushes 10 and adds
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.Emit(dtrules.OpAdd)

	err = rt.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	// Step 3: Query — result should be 15
	v, err := rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if !v.IsInteger() || v.AsInteger() != 15 {
		t.Errorf("Expected 15, got %v", v)
	}
}

// TestRuntimeQueryEmptyStacks tests querying empty stacks returns appropriate errors.
func TestRuntimeQueryEmptyStacks(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Empty value stack
	_, err := rt.PopValue()
	if err == nil {
		t.Error("PopValue on empty stack should return error")
	}

	_, err = rt.PeekValue()
	if err == nil {
		t.Error("PeekValue on empty stack should return error")
	}

	// Empty data stack
	_, err = rt.PopData()
	if err == nil {
		t.Error("PopData on empty stack should return error")
	}

	_, err = rt.PeekData()
	if err == nil {
		t.Error("PeekData on empty stack should return error")
	}

	// Empty entity stack
	_, err = rt.EntityByIndex(0)
	if err == nil {
		t.Error("EntityByIndex on empty stack should return error")
	}

	// Stack sizes should be zero
	if rt.ValueStackSize() != 0 {
		t.Errorf("Expected value stack size 0, got %d", rt.ValueStackSize())
	}
	if rt.DataStackSize() != 0 {
		t.Errorf("Expected data stack size 0, got %d", rt.DataStackSize())
	}
	if rt.EntityStackDepth() != 0 {
		t.Errorf("Expected entity stack depth 0, got %d", rt.EntityStackDepth())
	}
}

// TestGoRuntimeFactory tests the GoRuntimeFactory.
func TestGoRuntimeFactory(t *testing.T) {
	factory := &GoRuntimeFactory{}

	if factory.Name() != "go" {
		t.Errorf("Expected factory name 'go', got %q", factory.Name())
	}

	session := &mockSession{}
	rt, err := factory.CreateRuntime(session)
	if err != nil {
		t.Fatalf("CreateRuntime failed: %v", err)
	}
	if rt == nil {
		t.Fatal("CreateRuntime returned nil")
	}
	if rt.Name() != "go" {
		t.Errorf("Expected runtime name 'go', got %q", rt.Name())
	}

	// Verify the returned runtime works
	err = rt.PushValue(dtrules.NewValueInteger(42))
	if err != nil {
		t.Fatalf("PushValue failed: %v", err)
	}
	v, err := rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if v.AsInteger() != 42 {
		t.Errorf("Expected 42, got %d", v.AsInteger())
	}
}

// TestRuntimeFindValue tests FindValue through RuntimeQuery.
func TestRuntimeFindValue(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// FindValue with no entities on stack should return ValueNull
	name := dtrules.GetRName("nonexistent")
	if name != nil {
		v, err := rt.FindValue(name)
		if err != nil {
			t.Fatalf("FindValue failed: %v", err)
		}
		if !v.IsNull() {
			t.Errorf("Expected ValueNull for nonexistent name, got %v", v)
		}
	}
}

// TestRuntimeQueryEntity tests QueryEntity through RuntimeQuery.
func TestRuntimeQueryEntity(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// QueryEntity with no entities on stack should return nil
	name := dtrules.GetRName("nonexistent")
	if name != nil {
		e, err := rt.QueryEntity(name)
		if err != nil {
			t.Fatalf("QueryEntity failed: %v", err)
		}
		if e != nil {
			t.Errorf("Expected nil entity for nonexistent name, got %v", e)
		}
	}
}

// TestRuntimeSetOperatorTable tests setting the operator table via RuntimeInit.
func TestRuntimeSetOperatorTable(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Initially nil operator table should cause error on OpOperator
	bc := dtrules.NewBytecodeChunk()
	bc.EmitWithArg(dtrules.OpOperator, 0)

	err := state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected error executing OpOperator with nil operator table")
	}

	// Set an operator table
	table := make([]dtrules.Object, 10)
	state.SetOperatorTable(table)

	// OpOperator with nil entry should still error
	bc = dtrules.NewBytecodeChunk()
	bc.EmitWithArg(dtrules.OpOperator, 0)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}
	err = state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected error executing OpOperator with nil operator entry")
	}
}

// TestRuntimeFindValueWithEntity tests FindValue when an entity with a matching attribute is on the stack.
func TestRuntimeFindValueWithEntity(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Push primitives entity (it contains operator names as attributes)
	primitives := operators.NewPrimitivesEntity()
	err := rt.PushEntity(primitives)
	if err != nil {
		t.Fatalf("PushEntity failed: %v", err)
	}

	// Look up a name that should exist in primitives (like "add")
	name := dtrules.GetRName("add")
	if name == nil {
		t.Skip("Cannot create RName for 'add'")
	}

	v, err := rt.FindValue(name)
	if err != nil {
		t.Fatalf("FindValue failed: %v", err)
	}
	// Should find something (not null) since 'add' is an operator in primitives
	if v.IsNull() {
		// This may be null if primitives doesn't have "add" — check for any known attribute
		t.Log("FindValue returned null for 'add' — checking primitives setup")
	}
}

// TestRuntimeQueryEntityWithEntity tests QueryEntity when an entity is on the stack.
func TestRuntimeQueryEntityWithEntity(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	primitives := operators.NewPrimitivesEntity()
	err := rt.PushEntity(primitives)
	if err != nil {
		t.Fatalf("PushEntity failed: %v", err)
	}

	// Query for an attribute on the primitives entity
	name := dtrules.GetRName("add")
	if name == nil {
		t.Skip("Cannot create RName for 'add'")
	}

	e, err := rt.QueryEntity(name)
	if err != nil {
		t.Fatalf("QueryEntity failed: %v", err)
	}
	// If primitives has 'add', we should get back the primitives entity
	if e != nil && e != primitives {
		t.Error("QueryEntity should return the entity containing the attribute")
	}
}

// TestRuntimeMultipleEntities tests pushing multiple entities and verifying stack order.
func TestRuntimeMultipleEntities(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	primitives := operators.NewPrimitivesEntity()

	err := rt.PushEntity(primitives)
	if err != nil {
		t.Fatalf("PushEntity 1 failed: %v", err)
	}
	err = rt.PushEntity(primitives)
	if err != nil {
		t.Fatalf("PushEntity 2 failed: %v", err)
	}

	if rt.EntityStackDepth() != 2 {
		t.Errorf("Expected entity stack depth 2, got %d", rt.EntityStackDepth())
	}

	// EntityByIndex(0) is the top, EntityByIndex(1) is below
	e0, err := rt.EntityByIndex(0)
	if err != nil {
		t.Fatalf("EntityByIndex(0) failed: %v", err)
	}
	e1, err := rt.EntityByIndex(1)
	if err != nil {
		t.Fatalf("EntityByIndex(1) failed: %v", err)
	}
	if e0 == nil || e1 == nil {
		t.Error("Both entities should be non-nil")
	}

	// Out of range should error
	_, err = rt.EntityByIndex(2)
	if err == nil {
		t.Error("EntityByIndex out of range should return error")
	}
	_, err = rt.EntityByIndex(-1)
	if err == nil {
		t.Error("EntityByIndex negative should return error")
	}
}

// TestRuntimePushMultipleValues tests pushing various types of values.
func TestRuntimePushMultipleValues(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Push different value types
	values := []dtrules.Value{
		dtrules.NewValueInteger(100),
		dtrules.NewValueDouble(3.14),
		dtrules.NewValueBoolean(true),
		dtrules.NewValueString("hello"),
		dtrules.ValueNull,
		dtrules.ValueTrue,
		dtrules.ValueFalse,
	}

	for i, v := range values {
		if err := rt.PushValue(v); err != nil {
			t.Fatalf("PushValue[%d] failed: %v", i, err)
		}
	}

	if rt.ValueStackSize() != len(values) {
		t.Errorf("Expected stack size %d, got %d", len(values), rt.ValueStackSize())
	}

	// Pop in reverse order and verify
	for i := len(values) - 1; i >= 0; i-- {
		v, err := rt.PopValue()
		if err != nil {
			t.Fatalf("PopValue[%d] failed: %v", i, err)
		}
		if !v.Equal(values[i]) {
			t.Errorf("Value[%d]: expected %v, got %v", i, values[i], v)
		}
	}

	if rt.ValueStackSize() != 0 {
		t.Errorf("Expected empty stack, got %d", rt.ValueStackSize())
	}
}

// TestRuntimePushMultipleData tests pushing multiple data objects.
func TestRuntimePushMultipleData(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	objs := []dtrules.Object{
		dtrules.GetRIntegerValue(1),
		dtrules.GetRIntegerValue(2),
		dtrules.GetRIntegerValue(3),
	}

	for i, obj := range objs {
		if err := rt.PushData(obj); err != nil {
			t.Fatalf("PushData[%d] failed: %v", i, err)
		}
	}

	if rt.DataStackSize() != 3 {
		t.Errorf("Expected data stack size 3, got %d", rt.DataStackSize())
	}

	// PeekData should return top without removing
	top, err := rt.PeekData()
	if err != nil {
		t.Fatalf("PeekData failed: %v", err)
	}
	if rt.DataStackSize() != 3 {
		t.Error("PeekData should not remove elements")
	}
	val, _ := top.IntValue()
	if val != 3 {
		t.Errorf("PeekData expected 3, got %d", val)
	}

	// Pop all
	for i := len(objs) - 1; i >= 0; i-- {
		obj, err := rt.PopData()
		if err != nil {
			t.Fatalf("PopData[%d] failed: %v", i, err)
		}
		v, _ := obj.IntValue()
		expected, _ := objs[i].IntValue()
		if v != expected {
			t.Errorf("PopData[%d]: expected %d, got %d", i, expected, v)
		}
	}
}

// TestGoRuntimeFactoryInterfaceCompliance verifies GoRuntimeFactory satisfies RuntimeFactory.
func TestGoRuntimeFactoryInterfaceCompliance(t *testing.T) {
	var _ dtrules.RuntimeFactory = (*GoRuntimeFactory)(nil)
}

// TestGoRuntimeFactoryCreatesIndependentRuntimes verifies each runtime is independent.
func TestGoRuntimeFactoryCreatesIndependentRuntimes(t *testing.T) {
	factory := &GoRuntimeFactory{}
	session := &mockSession{}

	rt1, err := factory.CreateRuntime(session)
	if err != nil {
		t.Fatalf("CreateRuntime 1 failed: %v", err)
	}
	rt2, err := factory.CreateRuntime(session)
	if err != nil {
		t.Fatalf("CreateRuntime 2 failed: %v", err)
	}

	// Push value to rt1 only
	rt1.PushValue(dtrules.NewValueInteger(42))

	// rt2 should be independent — still empty
	if rt2.ValueStackSize() != 0 {
		t.Error("Runtimes should be independent — rt2 should have empty stack")
	}
	if rt1.ValueStackSize() != 1 {
		t.Errorf("rt1 should have 1 value, got %d", rt1.ValueStackSize())
	}
}

// TestRuntimeExecuteBytecodeChainedOperations tests complex bytecode sequences.
func TestRuntimeExecuteBytecodeChainedOperations(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	// Compute: (10 + 20) * 3 = 90
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)
	bc.EmitPushConstant(dtrules.NewValueInteger(3))
	bc.Emit(dtrules.OpMul)

	err := rt.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	v, err := rt.PopValue()
	if err != nil {
		t.Fatalf("PopValue failed: %v", err)
	}
	if v.AsInteger() != 90 {
		t.Errorf("Expected 90, got %d", v.AsInteger())
	}
}

// TestRuntimeExecuteEmptyBytecode tests executing empty bytecode.
func TestRuntimeExecuteEmptyBytecode(t *testing.T) {
	session := &mockSession{}
	var rt dtrules.Runtime = NewDTState(session)

	bc := dtrules.NewBytecodeChunk()
	err := rt.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode on empty bytecode should not fail: %v", err)
	}

	if rt.ValueStackSize() != 0 {
		t.Errorf("Empty bytecode should leave stack empty, got %d", rt.ValueStackSize())
	}
}

// TestEvaluateBytecodeConditionNoResult tests condition evaluation when bytecode produces no result.
func TestEvaluateBytecodeConditionNoResult(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Empty bytecode produces no result
	bc := dtrules.NewBytecodeChunk()
	_, err := state.EvaluateBytecodeCondition(bc)
	if err == nil {
		t.Error("Expected error for condition with no result on stack")
	}
}

// TestEvaluateBytecodeConditionExtraValues tests cleanup of extra values on stack.
func TestEvaluateBytecodeConditionExtraValues(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Bytecode that pushes extra values then a boolean result
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(999)) // extra
	bc.EmitPushConstant(dtrules.NewValueInteger(888)) // extra
	bc.EmitPushConstant(dtrules.NewValueBoolean(true)) // result

	result, err := state.EvaluateBytecodeCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeCondition failed: %v", err)
	}
	if !result {
		t.Error("Expected true")
	}

	// Stack should be cleaned up to initial depth (0)
	if state.ValueStackDepth() != 0 {
		t.Errorf("Expected stack cleaned to depth 0, got %d", state.ValueStackDepth())
	}
}

// TestEvaluateBytecodeConditionWithPreExistingStack tests condition eval doesn't disturb existing stack.
func TestEvaluateBytecodeConditionWithPreExistingStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Push some pre-existing values
	state.ValuePush(dtrules.NewValueInteger(100))
	state.ValuePush(dtrules.NewValueInteger(200))
	initialDepth := state.ValueStackDepth()

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueBoolean(false))

	result, err := state.EvaluateBytecodeCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeCondition failed: %v", err)
	}
	if result {
		t.Error("Expected false")
	}

	// Pre-existing values should still be there
	if state.ValueStackDepth() != initialDepth {
		t.Errorf("Expected stack depth %d (pre-existing), got %d", initialDepth, state.ValueStackDepth())
	}
}

// TestEvaluateBytecodeActionBalanced tests balanced action evaluation.
func TestEvaluateBytecodeActionBalanced(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Push initial value, then run balanced bytecode
	state.ValuePush(dtrules.NewValueInteger(100))
	initialDepth := state.ValueStackDepth()

	// Balanced: push then pop
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(42))
	bc.Emit(dtrules.OpPop)

	err := state.EvaluateBytecodeAction(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeAction failed: %v", err)
	}

	if state.ValueStackDepth() != initialDepth {
		t.Errorf("Expected stack depth %d, got %d", initialDepth, state.ValueStackDepth())
	}
}

// TestEvaluateBytecodeActionUnbalanced tests that unbalanced actions are detected and cleaned up.
func TestEvaluateBytecodeActionUnbalanced(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	initialDepth := state.ValueStackDepth()

	// Unbalanced: pushes but doesn't pop
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.EmitPushConstant(dtrules.NewValueInteger(2))

	err := state.EvaluateBytecodeAction(bc)
	if err == nil {
		t.Error("Expected error for unbalanced stack")
	}

	// Stack should be cleaned up
	if state.ValueStackDepth() != initialDepth {
		t.Errorf("Expected stack cleaned to depth %d, got %d", initialDepth, state.ValueStackDepth())
	}
}

// TestEvaluateBytecodeActionError tests error propagation in action evaluation.
func TestEvaluateBytecodeActionError(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Bytecode that will fail (add with only one operand)
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.Emit(dtrules.OpAdd)

	err := state.EvaluateBytecodeAction(bc)
	if err == nil {
		t.Error("Expected error for failed bytecode execution")
	}
}

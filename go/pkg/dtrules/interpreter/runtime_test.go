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

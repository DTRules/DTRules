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

func TestDataStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test empty stack
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got size %d", state.DataStackDepth())
	}

	// Test push
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))
	state.DataPush(dtrules.GetRIntegerValue(3))

	if state.DataStackDepth() != 3 {
		t.Errorf("Expected stack size 3, got %d", state.DataStackDepth())
	}

	// Test pop (LIFO order)
	obj, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}
	val, _ := obj.IntValue()
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}

	obj, err = state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 2 {
		t.Errorf("Expected 2, got %d", val)
	}

	// Test peek (top of stack)
	obj, err = state.DataPeek()
	if err != nil {
		t.Fatalf("DataPeek failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	// Stack should still have 1 element
	if state.DataStackDepth() != 1 {
		t.Errorf("Expected stack size 1, got %d", state.DataStackDepth())
	}

	// Pop last element
	state.DataPop()

	// Test underflow
	_, err = state.DataPop()
	if err == nil {
		t.Error("Expected stack underflow error")
	}
}

func TestDataStackIndexed(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRIntegerValue(20))
	state.DataPush(dtrules.GetRIntegerValue(30))

	// GetDataStack uses 0 as bottom of stack
	// Get at index 0 (bottom)
	obj, err := state.GetDataStack(0)
	if err != nil {
		t.Fatalf("GetDataStack(0) failed: %v", err)
	}
	val, _ := obj.IntValue()
	if val != 10 {
		t.Errorf("Expected 10 at index 0 (bottom), got %d", val)
	}

	// Get at index 1 (middle)
	obj, err = state.GetDataStack(1)
	if err != nil {
		t.Fatalf("GetDataStack(1) failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 20 {
		t.Errorf("Expected 20 at index 1, got %d", val)
	}

	// Get at index 2 (top)
	obj, err = state.GetDataStack(2)
	if err != nil {
		t.Fatalf("GetDataStack(2) failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 30 {
		t.Errorf("Expected 30 at index 2 (top), got %d", val)
	}

	// Get out of bounds
	_, err = state.GetDataStack(3)
	if err == nil {
		t.Error("Expected error for out of bounds get")
	}
}

func TestControlStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test empty control stack
	if state.CtrlDepth() != 0 {
		t.Errorf("Expected empty control stack, got size %d", state.CtrlDepth())
	}

	// Push to control stack
	state.CtrlPush(dtrules.NewRString("test"))
	if state.CtrlDepth() != 1 {
		t.Errorf("Expected control stack size 1, got %d", state.CtrlDepth())
	}

	// Pop from control stack
	obj, err := state.CtrlPop()
	if err != nil {
		t.Fatalf("CtrlPop failed: %v", err)
	}
	if obj == nil {
		t.Fatal("CtrlPop returned nil")
	}
	if obj.StringValue() != "test" {
		t.Errorf("Expected 'test', got '%s'", obj.StringValue())
	}
}

func TestEntityStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Initial entity stack should be empty
	if state.EntityDepth() != 0 {
		t.Errorf("Expected empty entity stack, got size %d", state.EntityDepth())
	}
}

func TestFrame(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Initial frame is 0
	initialFrame := state.GetCurrentFrame()
	if initialFrame != 0 {
		t.Errorf("Expected initial frame 0, got %d", initialFrame)
	}

	// Push some control stack items
	state.CtrlPush(dtrules.NewRString("before_frame"))

	// Push a frame
	err := state.PushFrame()
	if err != nil {
		t.Fatalf("PushFrame failed: %v", err)
	}

	// Frame should be updated
	newFrame := state.GetCurrentFrame()
	if newFrame <= initialFrame {
		t.Errorf("Expected frame to increase after push, got %d", newFrame)
	}

	// Push items in new frame
	state.CtrlPush(dtrules.NewRString("in_frame"))

	// Pop the frame
	err = state.PopFrame()
	if err != nil {
		t.Fatalf("PopFrame failed: %v", err)
	}

	// Frame should be back to initial
	if state.GetCurrentFrame() != initialFrame {
		t.Errorf("Expected frame %d after pop, got %d", initialFrame, state.GetCurrentFrame())
	}
}

func TestStateFlags(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test initial state
	if state.TestState(1) {
		t.Error("Flag 1 should not be set initially")
	}

	// Set a flag
	state.SetState(1)
	if !state.TestState(1) {
		t.Error("Flag 1 should be set")
	}

	// Clear the flag
	state.ClearState(1)
	if state.TestState(1) {
		t.Error("Flag 1 should be cleared")
	}
}

func TestGetSession(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	if state.GetSession() != session {
		t.Error("GetSession should return the session")
	}
}

// =============================================================================
// Value Stack Tests
// =============================================================================

func TestValueStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test empty stack
	if state.ValueStackDepth() != 0 {
		t.Errorf("Expected empty value stack, got size %d", state.ValueStackDepth())
	}

	// Test push
	state.ValuePush(dtrules.NewValueInteger(1))
	state.ValuePush(dtrules.NewValueInteger(2))
	state.ValuePush(dtrules.NewValueInteger(3))

	if state.ValueStackDepth() != 3 {
		t.Errorf("Expected value stack size 3, got %d", state.ValueStackDepth())
	}

	// Test pop (LIFO order)
	v, err := state.ValuePop()
	if err != nil {
		t.Fatalf("ValuePop failed: %v", err)
	}
	if v.AsInteger() != 3 {
		t.Errorf("Expected 3, got %d", v.AsInteger())
	}

	v, err = state.ValuePop()
	if err != nil {
		t.Fatalf("ValuePop failed: %v", err)
	}
	if v.AsInteger() != 2 {
		t.Errorf("Expected 2, got %d", v.AsInteger())
	}

	// Pop remaining and test underflow
	state.ValuePop()
	_, err = state.ValuePop()
	if err == nil {
		t.Error("Expected value stack underflow error")
	}
}

func TestValueStackDup(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	state.ValuePush(dtrules.NewValueInteger(42))
	err := state.ValueDup()
	if err != nil {
		t.Fatalf("ValueDup failed: %v", err)
	}

	if state.ValueStackDepth() != 2 {
		t.Errorf("Expected depth 2 after dup, got %d", state.ValueStackDepth())
	}

	v1, _ := state.ValuePop()
	v2, _ := state.ValuePop()
	if v1.AsInteger() != 42 || v2.AsInteger() != 42 {
		t.Error("Dup should duplicate the value")
	}
}

func TestValueStackSwap(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	state.ValuePush(dtrules.NewValueInteger(1))
	state.ValuePush(dtrules.NewValueInteger(2))

	err := state.ValueSwap()
	if err != nil {
		t.Fatalf("ValueSwap failed: %v", err)
	}

	v1, _ := state.ValuePop()
	v2, _ := state.ValuePop()
	if v1.AsInteger() != 1 || v2.AsInteger() != 2 {
		t.Errorf("Expected [2,1] after swap, got [%d,%d]", v2.AsInteger(), v1.AsInteger())
	}
}

func TestValueStackRot(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	state.ValuePush(dtrules.NewValueInteger(1))
	state.ValuePush(dtrules.NewValueInteger(2))
	state.ValuePush(dtrules.NewValueInteger(3))

	err := state.ValueRot()
	if err != nil {
		t.Fatalf("ValueRot failed: %v", err)
	}

	// After rot: 2 3 1 (top is 1)
	v1, _ := state.ValuePop()
	v2, _ := state.ValuePop()
	v3, _ := state.ValuePop()

	if v1.AsInteger() != 1 || v2.AsInteger() != 3 || v3.AsInteger() != 2 {
		t.Errorf("Expected [2,3,1], got [%d,%d,%d]", v3.AsInteger(), v2.AsInteger(), v1.AsInteger())
	}
}

func TestValueStackUnderflowErrors(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Dup on empty
	if err := state.ValueDup(); err == nil {
		t.Error("Expected error for dup on empty stack")
	}

	// Swap with one element
	state.ValuePush(dtrules.NewValueInteger(1))
	if err := state.ValueSwap(); err == nil {
		t.Error("Expected error for swap with one element")
	}

	// Rot with two elements
	state.ValuePush(dtrules.NewValueInteger(2))
	if err := state.ValueRot(); err == nil {
		t.Error("Expected error for rot with two elements")
	}
}

// =============================================================================
// Bytecode VM Tests
// =============================================================================

func TestBytecodeExecutionPushOperations(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	tests := []struct {
		name     string
		setup    func(*dtrules.BytecodeChunk)
		expected dtrules.Value
	}{
		{
			name: "push true",
			setup: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushTrue)
			},
			expected: dtrules.ValueTrue,
		},
		{
			name: "push false",
			setup: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushFalse)
			},
			expected: dtrules.ValueFalse,
		},
		{
			name: "push null",
			setup: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushNull)
			},
			expected: dtrules.ValueNull,
		},
		{
			name: "push zero",
			setup: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushZero)
			},
			expected: dtrules.NewValueInteger(0),
		},
		{
			name: "push one",
			setup: func(bc *dtrules.BytecodeChunk) {
				bc.Emit(dtrules.OpPushOne)
			},
			expected: dtrules.NewValueInteger(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := dtrules.NewBytecodeChunk()
			tt.setup(bc)

			// Clear state
			for state.ValueStackDepth() > 0 {
				state.ValuePop()
			}

			err := state.ExecuteBytecode(bc)
			if err != nil {
				t.Fatalf("ExecuteBytecode failed: %v", err)
			}

			result, err := state.ValuePop()
			if err != nil {
				t.Fatalf("ValuePop failed: %v", err)
			}

			if !result.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBytecodeExecutionArithmetic(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	tests := []struct {
		name     string
		a, b     int64
		op       dtrules.Opcode
		expected int64
	}{
		{"add", 10, 20, dtrules.OpAdd, 30},
		{"sub", 50, 20, dtrules.OpSub, 30},
		{"mul", 6, 7, dtrules.OpMul, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(tt.a))
			bc.EmitPushConstant(dtrules.NewValueInteger(tt.b))
			bc.Emit(tt.op)

			// Clear state
			for state.ValueStackDepth() > 0 {
				state.ValuePop()
			}

			err := state.ExecuteBytecode(bc)
			if err != nil {
				t.Fatalf("ExecuteBytecode failed: %v", err)
			}

			result, _ := state.ValuePop()
			if result.AsInteger() != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result.AsInteger())
			}
		})
	}
}

func TestBytecodeExecutionDivision(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(100))
	bc.EmitPushConstant(dtrules.NewValueInteger(5))
	bc.Emit(dtrules.OpDiv)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	result, _ := state.ValuePop()
	// Division always returns double
	if result.AsDouble() != 20.0 {
		t.Errorf("Expected 20.0, got %f", result.AsDouble())
	}
}

func TestBytecodeExecutionComparison(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	tests := []struct {
		name     string
		a, b     int64
		op       dtrules.Opcode
		expected bool
	}{
		{"eq true", 5, 5, dtrules.OpEq, true},
		{"eq false", 5, 10, dtrules.OpEq, false},
		{"ne true", 5, 10, dtrules.OpNe, true},
		{"ne false", 5, 5, dtrules.OpNe, false},
		{"lt true", 5, 10, dtrules.OpLt, true},
		{"lt false", 10, 5, dtrules.OpLt, false},
		{"le true equal", 5, 5, dtrules.OpLe, true},
		{"le true less", 5, 10, dtrules.OpLe, true},
		{"le false", 10, 5, dtrules.OpLe, false},
		{"gt true", 10, 5, dtrules.OpGt, true},
		{"gt false", 5, 10, dtrules.OpGt, false},
		{"ge true equal", 5, 5, dtrules.OpGe, true},
		{"ge true greater", 10, 5, dtrules.OpGe, true},
		{"ge false", 5, 10, dtrules.OpGe, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueInteger(tt.a))
			bc.EmitPushConstant(dtrules.NewValueInteger(tt.b))
			bc.Emit(tt.op)

			// Clear state
			for state.ValueStackDepth() > 0 {
				state.ValuePop()
			}

			err := state.ExecuteBytecode(bc)
			if err != nil {
				t.Fatalf("ExecuteBytecode failed: %v", err)
			}

			result, _ := state.ValuePop()
			if result.AsBoolean() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result.AsBoolean())
			}
		})
	}
}

func TestBytecodeExecutionBoolean(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	tests := []struct {
		name     string
		a, b     bool
		op       dtrules.Opcode
		expected bool
	}{
		{"and true true", true, true, dtrules.OpAnd, true},
		{"and true false", true, false, dtrules.OpAnd, false},
		{"and false true", false, true, dtrules.OpAnd, false},
		{"and false false", false, false, dtrules.OpAnd, false},
		{"or true true", true, true, dtrules.OpOr, true},
		{"or true false", true, false, dtrules.OpOr, true},
		{"or false true", false, true, dtrules.OpOr, true},
		{"or false false", false, false, dtrules.OpOr, false},
		{"xor true true", true, true, dtrules.OpXor, false},
		{"xor true false", true, false, dtrules.OpXor, true},
		{"xor false true", false, true, dtrules.OpXor, true},
		{"xor false false", false, false, dtrules.OpXor, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := dtrules.NewBytecodeChunk()
			bc.EmitPushConstant(dtrules.NewValueBoolean(tt.a))
			bc.EmitPushConstant(dtrules.NewValueBoolean(tt.b))
			bc.Emit(tt.op)

			// Clear state
			for state.ValueStackDepth() > 0 {
				state.ValuePop()
			}

			err := state.ExecuteBytecode(bc)
			if err != nil {
				t.Fatalf("ExecuteBytecode failed: %v", err)
			}

			result, _ := state.ValuePop()
			if result.AsBoolean() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result.AsBoolean())
			}
		})
	}
}

func TestBytecodeExecutionNot(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	tests := []struct {
		input    bool
		expected bool
	}{
		{true, false},
		{false, true},
	}

	for _, tt := range tests {
		bc := dtrules.NewBytecodeChunk()
		bc.EmitPushConstant(dtrules.NewValueBoolean(tt.input))
		bc.Emit(dtrules.OpNot)

		// Clear state
		for state.ValueStackDepth() > 0 {
			state.ValuePop()
		}

		err := state.ExecuteBytecode(bc)
		if err != nil {
			t.Fatalf("ExecuteBytecode failed: %v", err)
		}

		result, _ := state.ValuePop()
		if result.AsBoolean() != tt.expected {
			t.Errorf("not %v: expected %v, got %v", tt.input, tt.expected, result.AsBoolean())
		}
	}
}

func TestBytecodeExecutionNegIncDec(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test neg
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(42))
	bc.Emit(dtrules.OpNeg)

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (neg) failed: %v", err)
	}

	result, _ := state.ValuePop()
	if result.AsInteger() != -42 {
		t.Errorf("neg 42: expected -42, got %d", result.AsInteger())
	}

	// Test inc
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.Emit(dtrules.OpInc)

	err = state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (inc) failed: %v", err)
	}

	result, _ = state.ValuePop()
	if result.AsInteger() != 11 {
		t.Errorf("inc 10: expected 11, got %d", result.AsInteger())
	}

	// Test dec
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.Emit(dtrules.OpDec)

	err = state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (dec) failed: %v", err)
	}

	result, _ = state.ValuePop()
	if result.AsInteger() != 9 {
		t.Errorf("dec 10: expected 9, got %d", result.AsInteger())
	}
}

func TestBytecodeExecutionStackOps(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test pop
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.EmitPushConstant(dtrules.NewValueInteger(2))
	bc.Emit(dtrules.OpPop)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (pop) failed: %v", err)
	}

	if state.ValueStackDepth() != 1 {
		t.Errorf("Expected stack depth 1 after pop, got %d", state.ValueStackDepth())
	}

	result, _ := state.ValuePop()
	if result.AsInteger() != 1 {
		t.Errorf("Expected 1 remaining after pop, got %d", result.AsInteger())
	}

	// Test dup
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(42))
	bc.Emit(dtrules.OpDup)

	err = state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (dup) failed: %v", err)
	}

	if state.ValueStackDepth() != 2 {
		t.Errorf("Expected stack depth 2 after dup, got %d", state.ValueStackDepth())
	}

	v1, _ := state.ValuePop()
	v2, _ := state.ValuePop()
	if v1.AsInteger() != 42 || v2.AsInteger() != 42 {
		t.Error("Dup should duplicate the value")
	}

	// Test swap
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.EmitPushConstant(dtrules.NewValueInteger(2))
	bc.Emit(dtrules.OpSwap)

	err = state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (swap) failed: %v", err)
	}

	v1, _ = state.ValuePop()
	v2, _ = state.ValuePop()
	if v1.AsInteger() != 1 || v2.AsInteger() != 2 {
		t.Errorf("Expected [2,1] after swap, got [%d,%d]", v2.AsInteger(), v1.AsInteger())
	}

	// Test rot
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.EmitPushConstant(dtrules.NewValueInteger(2))
	bc.EmitPushConstant(dtrules.NewValueInteger(3))
	bc.Emit(dtrules.OpRot)

	err = state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode (rot) failed: %v", err)
	}

	v1, _ = state.ValuePop()
	v2, _ = state.ValuePop()
	v3, _ := state.ValuePop()
	if v1.AsInteger() != 1 || v2.AsInteger() != 3 || v3.AsInteger() != 2 {
		t.Errorf("Expected [2,3,1], got [%d,%d,%d]", v3.AsInteger(), v2.AsInteger(), v1.AsInteger())
	}
}

func TestBytecodeExecutionConstantPool(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	bc := dtrules.NewBytecodeChunk()
	// Add a large integer that goes into constant pool
	bc.EmitPushConstant(dtrules.NewValueInteger(1000000))
	bc.EmitPushConstant(dtrules.NewValueDouble(3.14159))
	bc.EmitPushConstant(dtrules.NewValueString("hello"))

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	if state.ValueStackDepth() != 3 {
		t.Errorf("Expected 3 values on stack, got %d", state.ValueStackDepth())
	}

	// Pop in reverse order
	v3, _ := state.ValuePop()
	v2, _ := state.ValuePop()
	v1, _ := state.ValuePop()

	if v1.AsInteger() != 1000000 {
		t.Errorf("Expected 1000000, got %d", v1.AsInteger())
	}
	if v2.AsDouble() != 3.14159 {
		t.Errorf("Expected 3.14159, got %f", v2.AsDouble())
	}
	if v3.AsString() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", v3.AsString())
	}
}

func TestBytecodeExecutionErrors(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test stack underflow on binary op
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.Emit(dtrules.OpAdd) // Only one operand

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected stack underflow error")
	}

	// Test invalid constant index
	bc = dtrules.NewBytecodeChunk()
	bc.EmitWithArg(dtrules.OpConstant, 999) // Invalid index

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err = state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected out of bounds error for invalid constant index")
	}

	// Test invalid name index
	bc = dtrules.NewBytecodeChunk()
	bc.EmitWithArg(dtrules.OpName, 999) // Invalid index

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err = state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected out of bounds error for invalid name index")
	}

	// Test invalid opcode
	bc = dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.Opcode(255)) // Invalid opcode

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err = state.ExecuteBytecode(bc)
	if err == nil {
		t.Error("Expected error for invalid opcode")
	}
}

func TestBytecodeExecutionNamePush(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	bc := dtrules.NewBytecodeChunk()
	testName := dtrules.GetRName("testattr")
	bc.EmitPushName(testName)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	result, _ := state.ValuePop()
	if result.AsName() != testName {
		t.Errorf("Expected name 'testattr', got %v", result.AsName())
	}
}

func TestBytecodeConditionEvaluation(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test true condition
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(5))
	bc.EmitPushConstant(dtrules.NewValueInteger(3))
	bc.Emit(dtrules.OpGt)

	result, err := state.EvaluateBytecodeCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeCondition failed: %v", err)
	}
	if !result {
		t.Error("Expected true for 5 > 3")
	}

	// Test false condition
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(3))
	bc.EmitPushConstant(dtrules.NewValueInteger(5))
	bc.Emit(dtrules.OpGt)

	result, err = state.EvaluateBytecodeCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeCondition failed: %v", err)
	}
	if result {
		t.Error("Expected false for 3 > 5")
	}
}

func TestBytecodeActionEvaluation(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test action that leaves stack balanced
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	bc.Emit(dtrules.OpPop)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.EvaluateBytecodeAction(bc)
	if err != nil {
		t.Fatalf("EvaluateBytecodeAction failed: %v", err)
	}

	// Test action that leaves extra values on stack - should be cleaned up
	bc = dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(1))
	// Don't pop - this value should be cleaned up automatically

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	initialDepth := state.ValueStackDepth()
	err = state.EvaluateBytecodeAction(bc)
	if err != nil {
		t.Errorf("EvaluateBytecodeAction should not error for unbalanced stack: %v", err)
	}
	// Stack should be cleaned up to initial depth
	if state.ValueStackDepth() != initialDepth {
		t.Errorf("Stack depth should be restored to %d, got %d", initialDepth, state.ValueStackDepth())
	}
}

func TestBytecodeNop(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpNop)
	bc.EmitPushConstant(dtrules.NewValueInteger(42))
	bc.Emit(dtrules.OpNop)

	for state.ValueStackDepth() > 0 {
		state.ValuePop()
	}

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	if state.ValueStackDepth() != 1 {
		t.Errorf("Expected 1 value on stack, got %d", state.ValueStackDepth())
	}

	result, _ := state.ValuePop()
	if result.AsInteger() != 42 {
		t.Errorf("Expected 42, got %d", result.AsInteger())
	}
}

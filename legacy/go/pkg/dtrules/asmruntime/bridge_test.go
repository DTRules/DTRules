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

package asmruntime

import (
	"testing"
	"unsafe"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func TestInit(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if !IsInitialized() {
		t.Error("IsInitialized() returned false after Init()")
	}

	// Calling Init() again should be a no-op
	err = Init()
	if err != nil {
		t.Errorf("Second Init() call failed: %v", err)
	}
}

func TestStackOperations(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Start fresh
	Reset()

	// Test pushing and popping integers
	v1 := dtrules.NewValueInteger(42)
	err = PushValue(v1)
	if err != nil {
		t.Fatalf("PushValue() failed: %v", err)
	}

	depth := StackDepth()
	if depth != 1 {
		t.Errorf("StackDepth() = %d, want 1", depth)
	}

	// Push another
	v2 := dtrules.NewValueInteger(100)
	err = PushValue(v2)
	if err != nil {
		t.Fatalf("PushValue(100) failed: %v", err)
	}

	// Peek should return 100
	peeked, err := PeekValue()
	if err != nil {
		t.Fatalf("PeekValue() failed: %v", err)
	}
	if peeked.AsInteger() != 100 {
		t.Errorf("PeekValue() = %d, want 100", peeked.AsInteger())
	}

	// Pop should return 100
	popped, err := PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}
	if popped.AsInteger() != 100 {
		t.Errorf("PopValue() = %d, want 100", popped.AsInteger())
	}

	// Pop should return 42
	popped, err = PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}
	if popped.AsInteger() != 42 {
		t.Errorf("PopValue() = %d, want 42", popped.AsInteger())
	}

	// Stack should be empty
	depth = StackDepth()
	if depth != 0 {
		t.Errorf("StackDepth() after pop = %d, want 0", depth)
	}
}

func TestSimpleBytecode(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Test: Push 1, Push 2, Add -> expect 3 on stack
	chunk := dtrules.NewBytecodeChunk()
	chunk.Emit(dtrules.OpPushOne)           // Push 1
	chunk.EmitWithArg(dtrules.OpPushInt, 2) // Push 2
	chunk.Emit(dtrules.OpAdd)               // Add

	err = ExecuteBytecode(chunk)
	if err != nil {
		t.Fatalf("ExecuteBytecode() failed: %v", err)
	}

	depth := StackDepth()
	if depth != 1 {
		t.Errorf("After 1+2: StackDepth() = %d, want 1", depth)
	}

	result, err := PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}

	if !result.IsInteger() {
		t.Errorf("Result is not integer, tag=%d", result.Tag())
	} else if result.AsInteger() != 3 {
		t.Errorf("1 + 2 = %d, want 3", result.AsInteger())
	}
}

func TestComparisonOps(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Test: 5 > 3 -> true
	chunk := dtrules.NewBytecodeChunk()
	chunk.EmitWithArg(dtrules.OpPushInt, 5)  // Push 5
	chunk.EmitWithArg(dtrules.OpPushInt, 3)  // Push 3
	chunk.Emit(dtrules.OpGt)                 // 5 > 3

	err = ExecuteBytecode(chunk)
	if err != nil {
		t.Fatalf("ExecuteBytecode() failed: %v", err)
	}

	result, err := PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}

	if !result.IsBoolean() {
		t.Errorf("Result is not boolean, tag=%d", result.Tag())
	} else if !result.AsBoolean() {
		t.Error("5 > 3 should be true")
	}
}

func TestBooleanOps(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Test: true and false -> false
	chunk := dtrules.NewBytecodeChunk()
	chunk.Emit(dtrules.OpPushTrue)   // Push true
	chunk.Emit(dtrules.OpPushFalse)  // Push false
	chunk.Emit(dtrules.OpAnd)        // true && false

	err = ExecuteBytecode(chunk)
	if err != nil {
		t.Fatalf("ExecuteBytecode() failed: %v", err)
	}

	result, err := PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}

	if !result.IsBoolean() {
		t.Errorf("Result is not boolean, tag=%d", result.Tag())
	} else if result.AsBoolean() {
		t.Error("true && false should be false")
	}
}

func TestConstantPool(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Test using constant pool with larger values that require constant pool
	// (values >= 64 or < -64 use OpConstant instead of OpPushInt)
	chunk := dtrules.NewBytecodeChunk()

	// Add constants to pool
	chunk.AddConstant(dtrules.NewValueInteger(100))
	chunk.AddConstant(dtrules.NewValueInteger(200))

	// Emit OpConstant with indices
	chunk.EmitWithArg(dtrules.OpConstant, 0)
	chunk.EmitWithArg(dtrules.OpConstant, 1)
	chunk.Emit(dtrules.OpAdd)

	err = ExecuteBytecode(chunk)
	if err != nil {
		t.Fatalf("ExecuteBytecode() failed: %v", err)
	}

	result, err := PopValue()
	if err != nil {
		t.Fatalf("PopValue() failed: %v", err)
	}

	if result.AsInteger() != 300 {
		t.Errorf("100 + 200 = %d, want 300", result.AsInteger())
	}
}

func BenchmarkSimpleArithmetic(b *testing.B) {
	err := Init()
	if err != nil {
		b.Fatalf("Init() failed: %v", err)
	}

	// Pre-build bytecode: 1 + 2
	chunk := dtrules.NewBytecodeChunk()
	chunk.Emit(dtrules.OpPushOne)
	chunk.EmitWithArg(dtrules.OpPushInt, 2)
	chunk.Emit(dtrules.OpAdd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Reset()
		ExecuteBytecode(chunk)
	}
}

func BenchmarkComplexExpression(b *testing.B) {
	err := Init()
	if err != nil {
		b.Fatalf("Init() failed: %v", err)
	}

	// Pre-build bytecode: ((10 + 20) * 3) - 5
	chunk := dtrules.NewBytecodeChunk()
	chunk.EmitWithArg(dtrules.OpPushInt, 10)
	chunk.EmitWithArg(dtrules.OpPushInt, 20)
	chunk.Emit(dtrules.OpAdd)
	chunk.EmitWithArg(dtrules.OpPushInt, 3)
	chunk.Emit(dtrules.OpMul)
	chunk.EmitWithArg(dtrules.OpPushInt, 5)
	chunk.Emit(dtrules.OpSub)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Reset()
		ExecuteBytecode(chunk)
	}
}

func TestEntityStackOperations(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Test entity stack depth (should be 0 initially)
	depth := EntityStackDepth()
	if depth != 0 {
		t.Errorf("Initial EntityStackDepth() = %d, want 0", depth)
	}

	// Create a simple entity in ASM heap
	typeName := CreateString("test_entity")
	if typeName == nil {
		t.Fatal("CreateString() returned nil")
	}

	entity := EntityAlloc(typeName)
	if entity == nil {
		t.Fatal("EntityAlloc() returned nil")
	}

	// Create a Value wrapping the entity
	entityValue := HeapAlloc(24)
	if entityValue == nil {
		t.Fatal("HeapAlloc(24) returned nil")
	}

	// Set tag to VTAG_ENTITY (7) and ptr to entity
	*(*byte)(entityValue) = 7                                                   // tag
	*(*uintptr)(unsafe.Pointer(uintptr(entityValue) + 16)) = uintptr(entity)   // ptr

	// Push entity onto stack
	err = EntityStackPush(entityValue)
	if err != nil {
		t.Fatalf("EntityStackPush() failed: %v", err)
	}

	depth = EntityStackDepth()
	if depth != 1 {
		t.Errorf("EntityStackDepth() after push = %d, want 1", depth)
	}

	// Peek should return same pointer
	peeked := EntityStackPeek(0)
	if peeked != entityValue {
		t.Errorf("EntityStackPeek(0) = %p, want %p", peeked, entityValue)
	}

	// Pop should return same pointer
	popped := EntityStackPop()
	if popped != entityValue {
		t.Errorf("EntityStackPop() = %p, want %p", popped, entityValue)
	}

	depth = EntityStackDepth()
	if depth != 0 {
		t.Errorf("EntityStackDepth() after pop = %d, want 0", depth)
	}
}

func TestEntityAttributes(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Create an entity
	typeName := CreateString("client")
	entity := EntityAlloc(typeName)
	if entity == nil {
		t.Fatal("EntityAlloc() returned nil")
	}

	// Create an attribute name
	attrName := CreateString("age")
	if attrName == nil {
		t.Fatal("CreateString('age') returned nil")
	}

	// Create an integer value (42)
	ageValue := HeapAlloc(24)
	if ageValue == nil {
		t.Fatal("HeapAlloc(24) returned nil")
	}
	*(*byte)(ageValue) = 1                                              // VTagInteger
	*(*int64)(unsafe.Pointer(uintptr(ageValue) + 8)) = 42               // num = 42

	// Set attribute
	err = EntitySetAttr(entity, attrName, ageValue)
	if err != nil {
		t.Fatalf("EntitySetAttr() failed: %v", err)
	}

	// Get attribute back
	gotValue := EntityGetAttr(entity, attrName)
	if gotValue == nil {
		t.Fatal("EntityGetAttr() returned nil")
	}

	// Verify value
	gotTag := *(*byte)(gotValue)
	gotNum := *(*int64)(unsafe.Pointer(uintptr(gotValue) + 8))

	if gotTag != 1 {
		t.Errorf("Got tag %d, want 1 (VTagInteger)", gotTag)
	}
	if gotNum != 42 {
		t.Errorf("Got num %d, want 42", gotNum)
	}
}

// TestCHIPLikeOperations tests the bytecode operations needed by the CHIP sample project.
// This verifies ASM can handle arithmetic, comparisons, and boolean logic.
func TestCHIPLikeOperations(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test 1: Income calculation: (10000 + 20000) / 12 = 2500
	t.Run("IncomeCalculation", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		chunk.AddConstant(dtrules.NewValueInteger(10000))
		chunk.AddConstant(dtrules.NewValueInteger(20000))
		chunk.EmitWithArg(dtrules.OpConstant, 0) // Push 10000
		chunk.EmitWithArg(dtrules.OpConstant, 1) // Push 20000
		chunk.Emit(dtrules.OpAdd)                // 30000
		chunk.EmitWithArg(dtrules.OpPushInt, 12) // Push 12
		chunk.Emit(dtrules.OpDiv)                // 2500

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if result.AsInteger() != 2500 {
			t.Errorf("(10000 + 20000) / 12 = %d, want 2500", result.AsInteger())
		}
	})

	// Test 2: Eligibility check: age >= 18 && income < 50000
	t.Run("EligibilityCheck", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		// age = 25
		chunk.EmitWithArg(dtrules.OpPushInt, 25) // Push age
		chunk.EmitWithArg(dtrules.OpPushInt, 18) // Push 18
		chunk.Emit(dtrules.OpGe)                 // age >= 18 -> true

		// income = 30000
		chunk.AddConstant(dtrules.NewValueInteger(30000))
		chunk.AddConstant(dtrules.NewValueInteger(50000))
		chunk.EmitWithArg(dtrules.OpConstant, 0) // Push 30000
		chunk.EmitWithArg(dtrules.OpConstant, 1) // Push 50000
		chunk.Emit(dtrules.OpLt)                 // income < 50000 -> true

		chunk.Emit(dtrules.OpAnd) // true && true -> true

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if !result.IsBoolean() || !result.AsBoolean() {
			t.Errorf("Eligibility check should be true, got %v", result)
		}
	})

	// Test 3: Complex calculation with stack operations
	t.Run("StackOperations", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		chunk.EmitWithArg(dtrules.OpPushInt, 10) // Push 10
		chunk.Emit(dtrules.OpDup)                // Duplicate -> 10, 10
		chunk.Emit(dtrules.OpMul)                // 100

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if result.AsInteger() != 100 {
			t.Errorf("10 * 10 = %d, want 100", result.AsInteger())
		}
	})

	// Test 4: Null check (common in CHIP)
	t.Run("NullCheck", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		chunk.Emit(dtrules.OpPushNull)  // Push null
		chunk.Emit(dtrules.OpPushNull)  // Push another null
		chunk.Emit(dtrules.OpEq)        // null == null -> true

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if !result.IsBoolean() || !result.AsBoolean() {
			t.Errorf("null == null should be true, got %v", result)
		}
	})

	// Test 5: Boolean OR (used in CHIP for multi-condition checks)
	t.Run("BooleanOr", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		chunk.Emit(dtrules.OpPushFalse)
		chunk.Emit(dtrules.OpPushTrue)
		chunk.Emit(dtrules.OpOr) // false || true -> true

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if !result.IsBoolean() || !result.AsBoolean() {
			t.Errorf("false || true should be true, got %v", result)
		}
	})

	// Test 6: NOT operator (used in CHIP for negation)
	t.Run("BooleanNot", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		chunk.Emit(dtrules.OpPushFalse)
		chunk.Emit(dtrules.OpNot) // !false -> true

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if !result.IsBoolean() || !result.AsBoolean() {
			t.Errorf("!false should be true, got %v", result)
		}
	})

	// Test 7: Multiple comparisons in sequence
	t.Run("ComparisonSequence", func(t *testing.T) {
		Reset()
		chunk := dtrules.NewBytecodeChunk()
		// Test: 100 <= 200 && 200 < 300 && 300 != 400
		chunk.AddConstant(dtrules.NewValueInteger(100))
		chunk.AddConstant(dtrules.NewValueInteger(200))
		chunk.AddConstant(dtrules.NewValueInteger(300))
		chunk.AddConstant(dtrules.NewValueInteger(400))

		chunk.EmitWithArg(dtrules.OpConstant, 0)
		chunk.EmitWithArg(dtrules.OpConstant, 1)
		chunk.Emit(dtrules.OpLe) // 100 <= 200

		chunk.EmitWithArg(dtrules.OpConstant, 1)
		chunk.EmitWithArg(dtrules.OpConstant, 2)
		chunk.Emit(dtrules.OpLt) // 200 < 300

		chunk.Emit(dtrules.OpAnd) // first two results

		chunk.EmitWithArg(dtrules.OpConstant, 2)
		chunk.EmitWithArg(dtrules.OpConstant, 3)
		chunk.Emit(dtrules.OpNe) // 300 != 400

		chunk.Emit(dtrules.OpAnd) // all three

		err := ExecuteBytecode(chunk)
		if err != nil {
			t.Fatalf("ExecuteBytecode() failed: %v", err)
		}

		result, err := PopValue()
		if err != nil {
			t.Fatalf("PopValue() failed: %v", err)
		}

		if !result.IsBoolean() || !result.AsBoolean() {
			t.Errorf("Comparison sequence should be true, got %v", result)
		}
	})
}

// TestCHIPEntityWorkflow tests entity create, attribute set/get which CHIP uses extensively.
func TestCHIPEntityWorkflow(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	Reset()

	// Simulate CHIP workflow: Create client entity with age and income

	// 1. Create client entity
	clientType := CreateString("client")
	clientEntity := EntityAlloc(clientType)
	if clientEntity == nil {
		t.Fatal("Failed to allocate client entity")
	}

	// 2. Set age = 35
	ageName := CreateString("age")
	ageValue := HeapAlloc(24)
	*(*byte)(ageValue) = 1 // VTagInteger
	*(*int64)(unsafe.Pointer(uintptr(ageValue) + 8)) = 35

	err = EntitySetAttr(clientEntity, ageName, ageValue)
	if err != nil {
		t.Fatalf("Failed to set age: %v", err)
	}

	// 3. Set income = 45000
	incomeName := CreateString("income")
	incomeValue := HeapAlloc(24)
	*(*byte)(incomeValue) = 1 // VTagInteger
	*(*int64)(unsafe.Pointer(uintptr(incomeValue) + 8)) = 45000

	err = EntitySetAttr(clientEntity, incomeName, incomeValue)
	if err != nil {
		t.Fatalf("Failed to set income: %v", err)
	}

	// 4. Set eligible = false initially
	eligibleName := CreateString("eligible")
	eligibleValue := HeapAlloc(24)
	*(*byte)(eligibleValue) = 3 // VTagBoolean
	*(*int64)(unsafe.Pointer(uintptr(eligibleValue) + 8)) = 0 // false

	err = EntitySetAttr(clientEntity, eligibleName, eligibleValue)
	if err != nil {
		t.Fatalf("Failed to set eligible: %v", err)
	}

	// 5. Retrieve and verify attributes
	gotAge := EntityGetAttr(clientEntity, ageName)
	if gotAge == nil {
		t.Fatal("Failed to get age attribute")
	}
	gotAgeVal := *(*int64)(unsafe.Pointer(uintptr(gotAge) + 8))
	if gotAgeVal != 35 {
		t.Errorf("Age = %d, want 35", gotAgeVal)
	}

	gotIncome := EntityGetAttr(clientEntity, incomeName)
	if gotIncome == nil {
		t.Fatal("Failed to get income attribute")
	}
	gotIncomeVal := *(*int64)(unsafe.Pointer(uintptr(gotIncome) + 8))
	if gotIncomeVal != 45000 {
		t.Errorf("Income = %d, want 45000", gotIncomeVal)
	}

	gotEligible := EntityGetAttr(clientEntity, eligibleName)
	if gotEligible == nil {
		t.Fatal("Failed to get eligible attribute")
	}
	gotEligibleTag := *(*byte)(gotEligible)
	gotEligibleVal := *(*int64)(unsafe.Pointer(uintptr(gotEligible) + 8))
	if gotEligibleTag != 3 || gotEligibleVal != 0 {
		t.Errorf("Eligible = (tag:%d, val:%d), want (tag:3, val:0)", gotEligibleTag, gotEligibleVal)
	}

	// 6. Now simulate setting eligible = true based on calculation
	*(*int64)(unsafe.Pointer(uintptr(eligibleValue) + 8)) = 1 // true
	err = EntitySetAttr(clientEntity, eligibleName, eligibleValue)
	if err != nil {
		t.Fatalf("Failed to update eligible: %v", err)
	}

	// Verify update
	gotEligible2 := EntityGetAttr(clientEntity, eligibleName)
	gotEligibleVal2 := *(*int64)(unsafe.Pointer(uintptr(gotEligible2) + 8))
	if gotEligibleVal2 != 1 {
		t.Errorf("Updated eligible = %d, want 1", gotEligibleVal2)
	}

	t.Log("CHIP entity workflow completed successfully")
}

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
	"unsafe"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// TestDTStateMemoryLayout verifies the exact memory layout of the DTState struct.
// This test ensures the layout remains stable for potential ASM access.
//
// DTState is the core interpreter state. For ASM-level access to the stacks
// (especially the value stack for hot-path bytecode execution), these offsets
// must remain stable. If any assertion fails, assembly code must be updated.
//
// Each Go slice header is 24 bytes: {ptr uintptr, len int, cap int}.
// To access the underlying array from ASM, dereference the pointer at the
// slice offset. The length is at slice_offset+8, capacity at slice_offset+16.
func TestDTStateMemoryLayout(t *testing.T) {
	var s DTState

	// Total size: 304 bytes
	if got := unsafe.Sizeof(s); got != 304 {
		t.Errorf("sizeof(DTState) = %d, want 304", got)
	}

	// Alignment: 8-byte aligned
	if got := unsafe.Alignof(s); got != 8 {
		t.Errorf("alignof(DTState) = %d, want 8", got)
	}

	// === The Three Stacks + Value Stack ===

	// ctrlStk ([]Object): offset 0, size 24
	// Control stack for decision table call frames and local variables.
	// Slice header: ptr at +0, len at +8, cap at +16.
	if got := unsafe.Offsetof(s.ctrlStk); got != 0 {
		t.Errorf("offsetof(DTState.ctrlStk) = %d, want 0", got)
	}

	// dataStk ([]Object): offset 24, size 24
	// Data stack for passing operands and returning results.
	// Slice header: ptr at +24, len at +32, cap at +40.
	if got := unsafe.Offsetof(s.dataStk); got != 24 {
		t.Errorf("offsetof(DTState.dataStk) = %d, want 24", got)
	}

	// entityStk ([]Entity): offset 48, size 24
	// Entity stack for scoped name resolution context.
	// Slice header: ptr at +48, len at +56, cap at +64.
	if got := unsafe.Offsetof(s.entityStk); got != 48 {
		t.Errorf("offsetof(DTState.entityStk) = %d, want 48", got)
	}

	// valueStk ([]Value): offset 72, size 24
	// Optimized value stack for bytecode VM execution.
	// This is the primary stack for ASM hot paths.
	// Slice header: ptr at +72, len at +80, cap at +88.
	// Elements are 24 bytes each (Value struct stride).
	if got := unsafe.Offsetof(s.valueStk); got != 72 {
		t.Errorf("offsetof(DTState.valueStk) = %d, want 72", got)
	}

	// === Frame Management ===

	// frames ([]int): offset 96, size 24
	// Stack of saved frame pointers for nested call frames.
	if got := unsafe.Offsetof(s.frames); got != 96 {
		t.Errorf("offsetof(DTState.frames) = %d, want 96", got)
	}

	// currentFrame (int): offset 120, size 8
	// Index into ctrlStk marking the start of the active frame.
	if got := unsafe.Offsetof(s.currentFrame); got != 120 {
		t.Errorf("offsetof(DTState.currentFrame) = %d, want 120", got)
	}

	// === Session and State ===

	// session (Session interface): offset 128, size 16
	// Two-word interface: (type pointer, data pointer).
	if got := unsafe.Offsetof(s.session); got != 128 {
		t.Errorf("offsetof(DTState.session) = %d, want 128", got)
	}

	// state (int): offset 144, size 8
	// Bit flags: DEBUG=0x1, TRACE=0x2, ECHO=0x4, VERBOSE=0x8.
	if got := unsafe.Offsetof(s.state); got != 144 {
		t.Errorf("offsetof(DTState.state) = %d, want 144", got)
	}

	// === Operator Table (for bytecode VM) ===

	// operatorTable ([]Object): offset 280, size 24
	// Pre-indexed operator lookup table for OpOperator bytecode instruction.
	// Slice header: ptr at +280, len at +288, cap at +296.
	if got := unsafe.Offsetof(s.operatorTable); got != 280 {
		t.Errorf("offsetof(DTState.operatorTable) = %d, want 280", got)
	}
}

// TestDTStateSliceHeaderLayout verifies Go slice header structure.
// Go slices are represented as: {Data unsafe.Pointer, Len int, Cap int}.
// This is the standard Go runtime layout (reflect.SliceHeader).
func TestDTStateSliceHeaderLayout(t *testing.T) {
	// Slice header is always 24 bytes on 64-bit: ptr(8) + len(8) + cap(8)
	var sl []byte
	if got := unsafe.Sizeof(sl); got != 24 {
		t.Errorf("sizeof([]byte) = %d, want 24", got)
	}

	// Verify pointer size is 8 bytes (64-bit)
	if got := unsafe.Sizeof(uintptr(0)); got != 8 {
		t.Errorf("sizeof(uintptr) = %d, want 8", got)
	}

	// Verify int size is 8 bytes (64-bit)
	if got := unsafe.Sizeof(int(0)); got != 8 {
		t.Errorf("sizeof(int) = %d, want 8", got)
	}
}

// TestDTStateValueStackAccess verifies that the value stack's underlying
// array can be accessed via unsafe pointer arithmetic, as ASM code would.
func TestDTStateValueStackAccess(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Push three values
	state.ValuePush(dtrules.NewValueInteger(10))
	state.ValuePush(dtrules.NewValueInteger(20))
	state.ValuePush(dtrules.NewValueInteger(30))

	// Simulate ASM access: get pointer to DTState, navigate to valueStk
	statePtr := unsafe.Pointer(state)

	// valueStk slice header is at offset 72.
	// A Go slice header is {Data unsafe.Pointer, Len int, Cap int}.
	// We read each field individually at its known offset.
	valueStkBase := unsafe.Add(statePtr, 72)

	// Data pointer (offset +0 within slice header)
	dataPtr := *(*unsafe.Pointer)(valueStkBase)

	// Length (offset +8 within slice header)
	stkLen := *(*int)(unsafe.Add(valueStkBase, 8))
	if stkLen != 3 {
		t.Errorf("value stack len via unsafe = %d, want 3", stkLen)
	}

	// Capacity (offset +16 within slice header)
	stkCap := *(*int)(unsafe.Add(valueStkBase, 16))
	if stkCap != stackLimit {
		t.Errorf("value stack cap via unsafe = %d, want %d", stkCap, stackLimit)
	}

	// Access the last element (top of stack) via pointer arithmetic.
	// Element stride is 24 bytes (sizeof(Value)).
	topIdx := stkLen - 1
	topValuePtr := unsafe.Add(dataPtr, uintptr(topIdx)*24)

	// Read the tag byte at offset 0 of the Value
	tag := *(*uint8)(topValuePtr)
	if tag != dtrules.VTagInteger {
		t.Errorf("top value tag via unsafe = %d, want %d (VTagInteger)", tag, dtrules.VTagInteger)
	}

	// Read the num field at offset 8 of the Value
	num := *(*int64)(unsafe.Add(topValuePtr, 8))
	if num != 30 {
		t.Errorf("top value num via unsafe = %d, want 30", num)
	}

	// Read element at index 0 (bottom of stack)
	bottomValuePtr := dataPtr
	bottomTag := *(*uint8)(bottomValuePtr)
	if bottomTag != dtrules.VTagInteger {
		t.Errorf("bottom value tag via unsafe = %d, want %d", bottomTag, dtrules.VTagInteger)
	}
	bottomNum := *(*int64)(unsafe.Add(bottomValuePtr, 8))
	if bottomNum != 10 {
		t.Errorf("bottom value num via unsafe = %d, want 10", bottomNum)
	}
}

// TestDTStateStateFlags verifies the state flags field can be read via
// unsafe access at offset 144, as ASM code would to check TRACE/DEBUG.
func TestDTStateStateFlags(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	statePtr := unsafe.Pointer(state)

	// state flags at offset 144
	flagsPtr := (*int)(unsafe.Add(statePtr, 144))

	// Initially all flags clear
	if *flagsPtr != 0 {
		t.Errorf("initial state flags = %d, want 0", *flagsPtr)
	}

	// Set TRACE flag (0x2)
	state.SetState(TRACE)
	if *flagsPtr != TRACE {
		t.Errorf("state flags after SetState(TRACE) = %d, want %d", *flagsPtr, TRACE)
	}

	// Set DEBUG flag too (0x1)
	state.SetState(DEBUG)
	if *flagsPtr != (TRACE | DEBUG) {
		t.Errorf("state flags after SetState(DEBUG) = %d, want %d", *flagsPtr, TRACE|DEBUG)
	}
}

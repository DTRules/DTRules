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

package dtrules

import (
	"testing"
	"unsafe"
)

// TestValueMemoryLayout verifies the exact memory layout of the Value struct.
// This test ensures the layout remains stable for potential ASM access.
// If any of these assertions fail, any assembly code accessing Value fields
// must be updated to match the new offsets.
func TestValueMemoryLayout(t *testing.T) {
	var v Value

	// Total size: 24 bytes (3 x 8-byte words)
	if got := unsafe.Sizeof(v); got != 24 {
		t.Errorf("sizeof(Value) = %d, want 24", got)
	}

	// Alignment: 8-byte aligned (required for int64 and pointer fields)
	if got := unsafe.Alignof(v); got != 8 {
		t.Errorf("alignof(Value) = %d, want 8", got)
	}

	// Field: tag (uint8) at offset 0
	// Type discriminator: VTagNull(0), VTagInteger(1), VTagDouble(2),
	// VTagBoolean(3), VTagString(4), VTagName(5), VTagArray(6),
	// VTagEntity(7), VTagObject(8)
	if got := unsafe.Offsetof(v.tag); got != 0 {
		t.Errorf("offsetof(Value.tag) = %d, want 0", got)
	}
	if got := unsafe.Sizeof(v.tag); got != 1 {
		t.Errorf("sizeof(Value.tag) = %d, want 1", got)
	}

	// Field: num (int64) at offset 8
	// Stores: integer values, float64 bits (via Float64bits), boolean as 0/1,
	// or type pointer word for VTagObject
	if got := unsafe.Offsetof(v.num); got != 8 {
		t.Errorf("offsetof(Value.num) = %d, want 8", got)
	}
	if got := unsafe.Sizeof(v.num); got != 8 {
		t.Errorf("sizeof(Value.num) = %d, want 8", got)
	}

	// Field: ptr (unsafe.Pointer) at offset 16
	// Stores: pointer to string header, *RName, *RArray, or data pointer
	// for VTagObject
	if got := unsafe.Offsetof(v.ptr); got != 16 {
		t.Errorf("offsetof(Value.ptr) = %d, want 16", got)
	}
	if got := unsafe.Sizeof(v.ptr); got != 8 {
		t.Errorf("sizeof(Value.ptr) = %d, want 8", got)
	}
}

// TestValueTagConstants verifies the tag constant values used as type
// discriminators. ASM code can use these values to branch on type.
func TestValueTagConstants(t *testing.T) {
	tests := []struct {
		name string
		tag  uint8
		want uint8
	}{
		{"VTagNull", VTagNull, 0},
		{"VTagInteger", VTagInteger, 1},
		{"VTagDouble", VTagDouble, 2},
		{"VTagBoolean", VTagBoolean, 3},
		{"VTagString", VTagString, 4},
		{"VTagName", VTagName, 5},
		{"VTagArray", VTagArray, 6},
		{"VTagEntity", VTagEntity, 7},
		{"VTagObject", VTagObject, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tag != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.tag, tt.want)
			}
		})
	}
}

// TestValueSliceLayout verifies that a []Value slice has the expected
// element stride for pointer arithmetic in ASM.
func TestValueSliceLayout(t *testing.T) {
	values := make([]Value, 3)

	// Each element is 24 bytes apart
	stride := uintptr(unsafe.Pointer(&values[1])) - uintptr(unsafe.Pointer(&values[0]))
	if stride != 24 {
		t.Errorf("Value slice stride = %d, want 24", stride)
	}

	// Verify contiguous layout
	stride2 := uintptr(unsafe.Pointer(&values[2])) - uintptr(unsafe.Pointer(&values[1]))
	if stride2 != 24 {
		t.Errorf("Value slice stride (1->2) = %d, want 24", stride2)
	}
}

// TestBytecodeChunkLayout verifies the memory layout of BytecodeChunk.
// ASM code reading bytecode needs to know where the code, constants,
// and names slices are stored.
func TestBytecodeChunkLayout(t *testing.T) {
	var bc BytecodeChunk

	// Total size: 72 bytes (3 slice headers x 24 bytes each)
	if got := unsafe.Sizeof(bc); got != 72 {
		t.Errorf("sizeof(BytecodeChunk) = %d, want 72", got)
	}

	// Field: code ([]byte) at offset 0
	if got := unsafe.Offsetof(bc.code); got != 0 {
		t.Errorf("offsetof(BytecodeChunk.code) = %d, want 0", got)
	}

	// Field: constants ([]Value) at offset 24
	if got := unsafe.Offsetof(bc.constants); got != 24 {
		t.Errorf("offsetof(BytecodeChunk.constants) = %d, want 24", got)
	}

	// Field: names ([]*RName) at offset 48
	if got := unsafe.Offsetof(bc.names); got != 48 {
		t.Errorf("offsetof(BytecodeChunk.names) = %d, want 48", got)
	}
}

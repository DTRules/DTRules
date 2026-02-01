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
)

func TestBytecodeChunk(t *testing.T) {
	bc := NewBytecodeChunk()
	if bc == nil {
		t.Fatal("NewBytecodeChunk returned nil")
	}
	if len(bc.Code()) != 0 {
		t.Error("expected empty code")
	}
}

func TestBytecodeEmit(t *testing.T) {
	bc := NewBytecodeChunk()
	bc.Emit(OpAdd)
	bc.Emit(OpSub)
	bc.Emit(OpMul)

	code := bc.Code()
	if len(code) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(code))
	}
	if Opcode(code[0]) != OpAdd {
		t.Errorf("expected OpAdd, got %d", code[0])
	}
	if Opcode(code[1]) != OpSub {
		t.Errorf("expected OpSub, got %d", code[1])
	}
	if Opcode(code[2]) != OpMul {
		t.Errorf("expected OpMul, got %d", code[2])
	}
}

func TestBytecodeEmitWithArg(t *testing.T) {
	tests := []struct {
		name string
		arg  int
	}{
		{"small", 42},
		{"medium", 1000},
		{"large", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBytecodeChunk()
			bc.EmitWithArg(OpPushInt, tt.arg)

			reader := NewBytecodeReader(bc.Code())
			op := reader.ReadOpcode()
			if op != OpPushInt {
				t.Errorf("expected OpPushInt, got %d", op)
			}
			arg := reader.ReadVarint()
			if arg != tt.arg {
				t.Errorf("expected %d, got %d", tt.arg, arg)
			}
		})
	}
}

func TestBytecodeConstants(t *testing.T) {
	bc := NewBytecodeChunk()

	// Add constants
	idx1 := bc.AddConstant(NewValueInteger(42))
	idx2 := bc.AddConstant(NewValueDouble(3.14))
	idx3 := bc.AddConstant(NewValueString("hello"))

	if idx1 != 0 || idx2 != 1 || idx3 != 2 {
		t.Errorf("unexpected indices: %d, %d, %d", idx1, idx2, idx3)
	}

	constants := bc.Constants()
	if len(constants) != 3 {
		t.Errorf("expected 3 constants, got %d", len(constants))
	}
	if constants[0].AsInteger() != 42 {
		t.Error("constant 0 mismatch")
	}
}

func TestBytecodeNames(t *testing.T) {
	bc := NewBytecodeChunk()

	name1 := GetRName("foo")
	name2 := GetRName("bar")

	idx1 := bc.AddName(name1)
	idx2 := bc.AddName(name2)
	idx3 := bc.AddName(name1) // Should return same index

	if idx1 != 0 || idx2 != 1 {
		t.Errorf("unexpected indices: %d, %d", idx1, idx2)
	}
	if idx3 != 0 {
		t.Errorf("expected reuse of index 0, got %d", idx3)
	}
}

func TestBytecodeEmitPushConstant(t *testing.T) {
	tests := []struct {
		name       string
		value      Value
		expectOp   Opcode
		expectPool bool
	}{
		{"true", ValueTrue, OpPushTrue, false},
		{"false", ValueFalse, OpPushFalse, false},
		{"null", ValueNull, OpPushNull, false},
		{"zero", NewValueInteger(0), OpPushZero, false},
		{"one", NewValueInteger(1), OpPushOne, false},
		{"small_int", NewValueInteger(10), OpPushInt, false},
		{"large_int", NewValueInteger(1000), OpConstant, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBytecodeChunk()
			bc.EmitPushConstant(tt.value)

			reader := NewBytecodeReader(bc.Code())
			op := reader.ReadOpcode()
			if op != tt.expectOp {
				t.Errorf("expected opcode %d, got %d", tt.expectOp, op)
			}
			if tt.expectPool && len(bc.Constants()) == 0 {
				t.Error("expected constant in pool")
			}
		})
	}
}

func TestBytecodeReader(t *testing.T) {
	bc := NewBytecodeChunk()
	bc.Emit(OpPushTrue)
	bc.Emit(OpPushFalse)
	bc.Emit(OpAnd)

	reader := NewBytecodeReader(bc.Code())

	if !reader.HasMore() {
		t.Error("expected HasMore() to be true")
	}

	op1 := reader.ReadOpcode()
	op2 := reader.ReadOpcode()
	op3 := reader.ReadOpcode()

	if op1 != OpPushTrue || op2 != OpPushFalse || op3 != OpAnd {
		t.Errorf("unexpected opcodes: %d, %d, %d", op1, op2, op3)
	}

	if reader.HasMore() {
		t.Error("expected HasMore() to be false")
	}
}

func TestBytecodeReaderVarint(t *testing.T) {
	tests := []int{0, 1, 127, 128, 255, 256, 10000, 100000}

	for _, expected := range tests {
		bc := NewBytecodeChunk()
		bc.EmitWithArg(OpPushInt, expected)

		reader := NewBytecodeReader(bc.Code())
		reader.ReadOpcode() // Skip opcode
		got := reader.ReadVarint()

		if got != expected {
			t.Errorf("varint: expected %d, got %d", expected, got)
		}
	}
}

func TestBytecodeSize(t *testing.T) {
	bc := NewBytecodeChunk()
	bc.EmitPushConstant(NewValueInteger(100))
	bc.EmitPushConstant(NewValueInteger(200))
	bc.Emit(OpAdd)

	size := bc.Size()
	if size == 0 {
		t.Error("expected non-zero size")
	}
}

func TestBytecodeSetPC(t *testing.T) {
	bc := NewBytecodeChunk()
	bc.Emit(OpPushTrue)
	bc.Emit(OpPushFalse)
	bc.Emit(OpAnd)

	reader := NewBytecodeReader(bc.Code())
	reader.ReadOpcode()
	reader.ReadOpcode()

	if reader.PC() != 2 {
		t.Errorf("expected PC=2, got %d", reader.PC())
	}

	reader.SetPC(0)
	if reader.PC() != 0 {
		t.Errorf("expected PC=0 after SetPC, got %d", reader.PC())
	}
}

func TestBytecodeReaderBoundsChecking(t *testing.T) {
	// Test TryReadOpcode on empty bytecode
	reader := NewBytecodeReader([]byte{})
	_, err := reader.TryReadOpcode()
	if err != ErrBytecodeOutOfBounds {
		t.Errorf("expected ErrBytecodeOutOfBounds, got %v", err)
	}

	// Test TryReadOpcode at end of bytecode
	reader = NewBytecodeReader([]byte{byte(OpAdd)})
	reader.ReadOpcode() // consume the one byte
	_, err = reader.TryReadOpcode()
	if err != ErrBytecodeOutOfBounds {
		t.Errorf("expected ErrBytecodeOutOfBounds at end, got %v", err)
	}

	// Test TryReadVarint on empty bytecode
	reader = NewBytecodeReader([]byte{})
	_, err = reader.TryReadVarint()
	if err != ErrBytecodeOutOfBounds {
		t.Errorf("expected ErrBytecodeOutOfBounds for varint, got %v", err)
	}

	// Test TryReadVarint with truncated varint (continuation bit set but no more bytes)
	reader = NewBytecodeReader([]byte{0x80}) // continuation bit set
	_, err = reader.TryReadVarint()
	if err != ErrBytecodeOutOfBounds {
		t.Errorf("expected ErrBytecodeOutOfBounds for truncated varint, got %v", err)
	}
}

func TestBytecodeReaderReadOpcodeSuccess(t *testing.T) {
	bc := NewBytecodeChunk()
	bc.Emit(OpAdd)
	bc.Emit(OpSub)

	reader := NewBytecodeReader(bc.Code())
	op, err := reader.TryReadOpcode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != OpAdd {
		t.Errorf("expected OpAdd, got %d", op)
	}
}

func TestBytecodeReaderReadVarintSuccess(t *testing.T) {
	// Manually create bytecode with a varint
	// Varint encoding: 42 = 0x2A (single byte, high bit clear)
	code := []byte{byte(OpPushInt), 0x2A}

	reader := NewBytecodeReader(code)
	// Skip the opcode
	_, err := reader.TryReadOpcode()
	if err != nil {
		t.Fatalf("unexpected error reading opcode: %v", err)
	}
	// Read the varint
	v, err := reader.TryReadVarint()
	if err != nil {
		t.Fatalf("unexpected error reading varint: %v", err)
	}
	if v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
}

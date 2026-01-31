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

// Bytecode provides a compact representation for compiled expressions.
// Instead of storing Object interfaces, bytecode uses:
// - Opcodes for operators (1 byte)
// - Immediate values for small integers
// - Indices into constant pools for strings/names/etc.
//
// This improves cache locality and reduces memory usage.

// Opcode represents a bytecode instruction.
type Opcode uint8

// Bytecode opcodes
const (
	// Stack operations
	OpNop     Opcode = 0
	OpPush    Opcode = 1  // Push constant from pool
	OpPushInt Opcode = 2  // Push immediate integer (followed by varint)
	OpPop     Opcode = 3  // Pop and discard
	OpDup     Opcode = 4  // Duplicate top
	OpSwap    Opcode = 5  // Swap top two
	OpRot     Opcode = 6  // Rotate top three
	OpRoll    Opcode = 7  // Roll n items
	OpIndex   Opcode = 8  // Copy nth item to top
	OpClear   Opcode = 9  // Clear stack to mark
	OpMark    Opcode = 10 // Push mark

	// Arithmetic
	OpAdd    Opcode = 20 // a + b
	OpSub    Opcode = 21 // a - b
	OpMul    Opcode = 22 // a * b
	OpDiv    Opcode = 23 // a / b
	OpMod    Opcode = 24 // a % b
	OpNeg    Opcode = 25 // -a
	OpAbs    Opcode = 26 // |a|
	OpInc    Opcode = 27 // a + 1
	OpDec    Opcode = 28 // a - 1

	// Comparison
	OpEq  Opcode = 30 // a == b
	OpNe  Opcode = 31 // a != b
	OpLt  Opcode = 32 // a < b
	OpLe  Opcode = 33 // a <= b
	OpGt  Opcode = 34 // a > b
	OpGe  Opcode = 35 // a >= b

	// Boolean
	OpAnd Opcode = 40 // a && b
	OpOr  Opcode = 41 // a || b
	OpNot Opcode = 42 // !a
	OpXor Opcode = 43 // a ^ b

	// Control flow
	OpExec    Opcode = 50 // Execute top of stack
	OpIf      Opcode = 51 // Conditional execution
	OpIfElse  Opcode = 52 // if-else
	OpWhile   Opcode = 53 // while loop
	OpFor     Opcode = 54 // for loop
	OpForAll  Opcode = 55 // iterate array
	OpReturn  Opcode = 56 // Return from procedure
	OpJump    Opcode = 57 // Unconditional jump (followed by offset)
	OpJumpIf  Opcode = 58 // Conditional jump
	OpCall    Opcode = 59 // Call operator by index

	// Entity operations
	OpEntityPush Opcode = 60 // Push entity onto entity stack
	OpEntityPop  Opcode = 61 // Pop entity from entity stack
	OpDef        Opcode = 62 // Define variable
	OpLookup     Opcode = 63 // Lookup name in context
	OpNewEntity  Opcode = 64 // Create new entity

	// Array operations
	OpNewArray  Opcode = 70 // Create new array
	OpAddTo     Opcode = 71 // Add to array
	OpLength    Opcode = 72 // Get array length
	OpGet       Opcode = 73 // Get array element
	OpPut       Opcode = 74 // Put array element

	// Table operations
	OpNewTable  Opcode = 80 // Create new table
	OpTableGet  Opcode = 81 // Get table value
	OpTablePut  Opcode = 82 // Put table value

	// String operations
	OpConcat    Opcode = 90 // String concatenation
	OpSubstring Opcode = 91 // Substring

	// Constants
	OpPushTrue  Opcode = 100 // Push true
	OpPushFalse Opcode = 101 // Push false
	OpPushNull  Opcode = 102 // Push null
	OpPushZero  Opcode = 103 // Push 0
	OpPushOne   Opcode = 104 // Push 1

	// Extended opcodes (for operators with indices)
	OpOperator Opcode = 200 // Call operator by index (followed by varint index)
	OpConstant Opcode = 201 // Push constant by index (followed by varint index)
	OpName     Opcode = 202 // Push name by index (followed by varint index)
)

// BytecodeChunk holds compiled bytecode and its constant pool.
type BytecodeChunk struct {
	code      []byte    // Bytecode instructions
	constants []Value   // Constant pool (immediate values, strings, etc.)
	names     []*RName  // Name pool (for lookup operations)
}

// NewBytecodeChunk creates a new bytecode chunk.
func NewBytecodeChunk() *BytecodeChunk {
	return &BytecodeChunk{
		code:      make([]byte, 0, 64),
		constants: make([]Value, 0, 16),
		names:     make([]*RName, 0, 16),
	}
}

// Emit adds an opcode to the chunk.
func (bc *BytecodeChunk) Emit(op Opcode) {
	bc.code = append(bc.code, byte(op))
}

// EmitWithArg adds an opcode with a varint argument.
func (bc *BytecodeChunk) EmitWithArg(op Opcode, arg int) {
	bc.code = append(bc.code, byte(op))
	bc.emitVarint(arg)
}

// emitVarint encodes a variable-length integer.
func (bc *BytecodeChunk) emitVarint(v int) {
	for v >= 0x80 {
		bc.code = append(bc.code, byte(v)|0x80)
		v >>= 7
	}
	bc.code = append(bc.code, byte(v))
}

// AddConstant adds a constant to the pool and returns its index.
func (bc *BytecodeChunk) AddConstant(v Value) int {
	idx := len(bc.constants)
	bc.constants = append(bc.constants, v)
	return idx
}

// AddName adds a name to the pool and returns its index.
func (bc *BytecodeChunk) AddName(n *RName) int {
	// Check if name already exists
	for i, existing := range bc.names {
		if existing == n {
			return i
		}
	}
	idx := len(bc.names)
	bc.names = append(bc.names, n)
	return idx
}

// EmitPushConstant emits code to push a constant.
func (bc *BytecodeChunk) EmitPushConstant(v Value) {
	// Optimize common constants
	switch {
	case v.tag == VTagBoolean && v.num != 0:
		bc.Emit(OpPushTrue)
	case v.tag == VTagBoolean && v.num == 0:
		bc.Emit(OpPushFalse)
	case v.tag == VTagNull:
		bc.Emit(OpPushNull)
	case v.tag == VTagInteger && v.num == 0:
		bc.Emit(OpPushZero)
	case v.tag == VTagInteger && v.num == 1:
		bc.Emit(OpPushOne)
	case v.tag == VTagInteger && v.num >= -64 && v.num < 64:
		// Small integers as immediate values
		bc.EmitWithArg(OpPushInt, int(v.num))
	default:
		// Add to constant pool
		idx := bc.AddConstant(v)
		bc.EmitWithArg(OpConstant, idx)
	}
}

// EmitPushName emits code to push a name.
func (bc *BytecodeChunk) EmitPushName(n *RName) {
	idx := bc.AddName(n)
	bc.EmitWithArg(OpName, idx)
}

// EmitOperator emits code to call an operator by index.
func (bc *BytecodeChunk) EmitOperator(opIdx int) {
	// Check for common operators with dedicated opcodes
	bc.EmitWithArg(OpOperator, opIdx)
}

// Code returns the bytecode.
func (bc *BytecodeChunk) Code() []byte {
	return bc.code
}

// Constants returns the constant pool.
func (bc *BytecodeChunk) Constants() []Value {
	return bc.constants
}

// Names returns the name pool.
func (bc *BytecodeChunk) Names() []*RName {
	return bc.names
}

// Size returns the total size in bytes.
func (bc *BytecodeChunk) Size() int {
	return len(bc.code) + len(bc.constants)*24 + len(bc.names)*8
}

// BytecodeReader reads bytecode instructions.
type BytecodeReader struct {
	code []byte
	pc   int
}

// NewBytecodeReader creates a reader for bytecode.
func NewBytecodeReader(code []byte) *BytecodeReader {
	return &BytecodeReader{code: code, pc: 0}
}

// HasMore returns true if there are more instructions.
func (r *BytecodeReader) HasMore() bool {
	return r.pc < len(r.code)
}

// ReadOpcode reads the next opcode.
func (r *BytecodeReader) ReadOpcode() Opcode {
	op := Opcode(r.code[r.pc])
	r.pc++
	return op
}

// ReadVarint reads a variable-length integer.
func (r *BytecodeReader) ReadVarint() int {
	v := 0
	shift := 0
	for {
		b := r.code[r.pc]
		r.pc++
		v |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return v
}

// PC returns the current program counter.
func (r *BytecodeReader) PC() int {
	return r.pc
}

// SetPC sets the program counter.
func (r *BytecodeReader) SetPC(pc int) {
	r.pc = pc
}

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

import "errors"

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
	OpAdd Opcode = 20 // a + b
	OpSub Opcode = 21 // a - b
	OpMul Opcode = 22 // a * b
	OpDiv Opcode = 23 // a / b
	OpMod Opcode = 24 // a % b
	OpNeg Opcode = 25 // -a
	OpAbs Opcode = 26 // |a|
	OpInc Opcode = 27 // a + 1
	OpDec Opcode = 28 // a - 1

	// Comparison
	OpEq Opcode = 30 // a == b
	OpNe Opcode = 31 // a != b
	OpLt Opcode = 32 // a < b
	OpLe Opcode = 33 // a <= b
	OpGt Opcode = 34 // a > b
	OpGe Opcode = 35 // a >= b

	// Boolean
	OpAnd Opcode = 40 // a && b
	OpOr  Opcode = 41 // a || b
	OpNot Opcode = 42 // !a
	OpXor Opcode = 43 // a ^ b

	// Control flow
	OpExec   Opcode = 50 // Execute top of stack
	OpIf     Opcode = 51 // Conditional execution
	OpIfElse Opcode = 52 // if-else
	OpWhile  Opcode = 53 // while loop
	OpFor    Opcode = 54 // for loop
	OpForAll Opcode = 55 // iterate array
	OpReturn Opcode = 56 // Return from procedure
	OpJump   Opcode = 57 // Unconditional jump (followed by offset)
	OpJumpIf Opcode = 58 // Conditional jump
	OpCall   Opcode = 59 // Call operator by index

	// Entity operations
	OpEntityPush Opcode = 60 // Push entity onto entity stack
	OpEntityPop  Opcode = 61 // Pop entity from entity stack
	OpDef        Opcode = 62 // Define variable
	OpLookup     Opcode = 63 // Lookup name in context
	OpNewEntity  Opcode = 64 // Create new entity

	// Array operations
	OpNewArray Opcode = 70 // Create new array
	OpAddTo    Opcode = 71 // Add to array
	OpLength   Opcode = 72 // Get array length
	OpGet      Opcode = 73 // Get array element
	OpPut      Opcode = 74 // Put array element

	// String operations
	OpConcat    Opcode = 90 // String concatenation
	OpSubstring Opcode = 91 // Substring

	// Bytes operations (stack effect noted as [a b -> result])
	OpBytesConcat Opcode = 110 // [bytes bytes -> bytes] concatenate two byte sequences
	OpBytesLen    Opcode = 111 // [bytes -> int]         number of bytes
	OpBytesSlice  Opcode = 112 // [bytes from to -> bytes] slice [from, to)
	OpBytesIndex  Opcode = 113 // [bytes i -> int]       byte at index i (0-255)
	OpBytesEq     Opcode = 114 // [bytes bytes -> bool]  constant-time equality
	OpBytesNe     Opcode = 115 // [bytes bytes -> bool]  constant-time inequality

	// Hash operations (stack effect: [bytes -> bytes])
	OpSha256    Opcode = 116 // [bytes -> bytes(32)] SHA-256
	OpKeccak256 Opcode = 117 // [bytes -> bytes(32)] Keccak-256 (Ethereum)
	OpRipemd160 Opcode = 118 // [bytes -> bytes(20)] RIPEMD-160
	OpSha3      Opcode = 119 // [bytes -> bytes(32)] SHA3-256

	// Encoding operations (starting at 120)
	OpHex         Opcode = 120 // [bytes -> string]         hex encode (lowercase, no 0x prefix)
	OpCvHex       Opcode = 121 // [string -> bytes]         decode hex (accepts optional 0x prefix)
	OpB58Check    Opcode = 122 // [bytes version -> string]  base58check encode
	OpCvB58Check  Opcode = 123 // [string -> bytes version]  base58check decode; pushes bytes then version
	OpBech32      Opcode = 124 // [bytes hrp -> string]      bech32 encode
	OpCvBech32    Opcode = 125 // [string -> bytes hrp]      bech32 decode; pushes bytes then hrp
	OpBigIntBytes Opcode = 126 // [bigint size -> bytes]     bigint → big-endian zero-padded bytes
	OpBytesBigInt Opcode = 127 // [bytes -> bigint]          bytes → bigint (unsigned big-endian)

	// EL statement opcodes
	OpError Opcode = 128 // [string --] halt execution, surface string as ELStatementError
	OpWarn  Opcode = 129 // [string --] non-fatal; logs string and continues

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
	code      []byte   // Bytecode instructions
	constants []Value  // Constant pool (immediate values, strings, etc.)
	names     []*RName // Name pool (for lookup operations)
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
	case v.tag == VTagInteger && v.num >= 0 && v.num < 128:
		// Small non-negative integers as immediate values
		// Negative integers go to constant pool due to varint encoding limitations
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

// ErrBytecodeOutOfBounds is returned when bytecode reading goes past the end.
var ErrBytecodeOutOfBounds = errors.New("bytecode read out of bounds")

// ErrBytecodeVarintOverflow is returned when a varint exceeds 64 bits.
var ErrBytecodeVarintOverflow = errors.New("bytecode varint overflow")

// ReadOpcode reads the next opcode.
// Panics if reading past end of bytecode (use TryReadOpcode for error handling).
func (r *BytecodeReader) ReadOpcode() Opcode {
	op, err := r.TryReadOpcode()
	if err != nil {
		panic(err)
	}
	return op
}

// TryReadOpcode reads the next opcode with bounds checking.
func (r *BytecodeReader) TryReadOpcode() (Opcode, error) {
	if r.pc >= len(r.code) {
		return 0, ErrBytecodeOutOfBounds
	}
	op := Opcode(r.code[r.pc])
	r.pc++
	return op, nil
}

// ReadVarint reads a variable-length integer.
// Panics on error (use TryReadVarint for error handling).
func (r *BytecodeReader) ReadVarint() int {
	v, err := r.TryReadVarint()
	if err != nil {
		panic(err)
	}
	return v
}

// TryReadVarint reads a variable-length integer with bounds and overflow checking.
func (r *BytecodeReader) TryReadVarint() (int, error) {
	v := 0
	shift := 0
	for {
		if r.pc >= len(r.code) {
			return 0, ErrBytecodeOutOfBounds
		}
		if shift >= 63 {
			return 0, ErrBytecodeVarintOverflow
		}
		b := r.code[r.pc]
		r.pc++
		v |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return v, nil
}

// PC returns the current program counter.
func (r *BytecodeReader) PC() int {
	return r.pc
}

// SetPC sets the program counter.
func (r *BytecodeReader) SetPC(pc int) {
	r.pc = pc
}

// Bytecode file format constants
const (
	BytecodeMagic   = "DTBC"
	BytecodeVersion = 1
)

// Serialize writes the bytecode chunk to a binary format.
// Format:
//   - Magic: 4 bytes "DTBC"
//   - Version: 1 byte
//   - Names count: varint
//   - Names: [length:varint, bytes:utf8]...
//   - Constants count: varint
//   - Constants: [type:byte, data:varies]...
//   - Code length: varint
//   - Code: bytes
func (bc *BytecodeChunk) Serialize() []byte {
	buf := make([]byte, 0, len(bc.code)+256)

	// Magic and version
	buf = append(buf, []byte(BytecodeMagic)...)
	buf = append(buf, BytecodeVersion)

	// Names
	buf = appendVarint(buf, len(bc.names))
	for _, name := range bc.names {
		s := name.StringValue()
		buf = appendVarint(buf, len(s))
		buf = append(buf, []byte(s)...)
	}

	// Constants
	buf = appendVarint(buf, len(bc.constants))
	for _, c := range bc.constants {
		buf = serializeValue(buf, c)
	}

	// Code
	buf = appendVarint(buf, len(bc.code))
	buf = append(buf, bc.code...)

	return buf
}

// serializeValue writes a Value to the buffer.
func serializeValue(buf []byte, v Value) []byte {
	buf = append(buf, byte(v.tag))
	switch v.tag {
	case VTagNull:
		// No data needed
	case VTagBoolean:
		if v.num != 0 {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	case VTagInteger:
		buf = appendSvarint(buf, v.num)
	case VTagDouble:
		// Write as 8 bytes (IEEE 754)
		bits := v.num
		for i := 0; i < 8; i++ {
			buf = append(buf, byte(bits>>(i*8)))
		}
	case VTagString:
		s := v.AsString()
		buf = appendVarint(buf, len(s))
		buf = append(buf, []byte(s)...)
	case VTagName:
		n := v.AsName()
		if n != nil {
			s := n.StringValue()
			buf = appendVarint(buf, len(s))
			buf = append(buf, []byte(s)...)
		} else {
			buf = appendVarint(buf, 0)
		}
	default:
		// Unknown type - write as null
		buf[len(buf)-1] = byte(VTagNull)
	}
	return buf
}

// appendVarint appends a variable-length unsigned integer.
func appendVarint(buf []byte, v int) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	buf = append(buf, byte(v))
	return buf
}

// appendSvarint appends a signed variable-length integer using zigzag encoding.
func appendSvarint(buf []byte, v int64) []byte {
	// Zigzag encode: (v << 1) ^ (v >> 63)
	uv := uint64((v << 1) ^ (v >> 63))
	for uv >= 0x80 {
		buf = append(buf, byte(uv)|0x80)
		uv >>= 7
	}
	buf = append(buf, byte(uv))
	return buf
}

// DeserializeBytecode reads a bytecode chunk from its serialized binary format.
// Returns an error if the data is invalid or corrupted.
func DeserializeBytecode(data []byte) (*BytecodeChunk, error) {
	if len(data) < 5 {
		return nil, errors.New("bytecode too short")
	}

	// Check magic
	if string(data[0:4]) != BytecodeMagic {
		return nil, errors.New("invalid bytecode magic")
	}

	// Check version
	if data[4] != BytecodeVersion {
		return nil, errors.New("unsupported bytecode version")
	}

	pos := 5
	bc := NewBytecodeChunk()

	// Read names
	nameCount, n := readVarintFrom(data[pos:])
	if n <= 0 {
		return nil, errors.New("invalid name count")
	}
	pos += n

	for i := 0; i < nameCount; i++ {
		strLen, n := readVarintFrom(data[pos:])
		if n <= 0 {
			return nil, errors.New("invalid name length")
		}
		pos += n

		if pos+strLen > len(data) {
			return nil, errors.New("name data truncated")
		}
		nameStr := string(data[pos : pos+strLen])
		pos += strLen

		bc.names = append(bc.names, GetRName(nameStr))
	}

	// Read constants
	constCount, n := readVarintFrom(data[pos:])
	if n <= 0 {
		return nil, errors.New("invalid constant count")
	}
	pos += n

	for i := 0; i < constCount; i++ {
		if pos >= len(data) {
			return nil, errors.New("constant data truncated")
		}

		tag := data[pos]
		pos++

		var v Value
		switch tag {
		case VTagNull:
			v = ValueNull
		case VTagBoolean:
			if pos >= len(data) {
				return nil, errors.New("boolean data truncated")
			}
			v = NewValueBoolean(data[pos] != 0)
			pos++
		case VTagInteger:
			intVal, n := readSvarintFrom(data[pos:])
			if n <= 0 {
				return nil, errors.New("invalid integer value")
			}
			pos += n
			v = NewValueInteger(intVal)
		case VTagDouble:
			if pos+8 > len(data) {
				return nil, errors.New("double data truncated")
			}
			var bits uint64
			for j := 0; j < 8; j++ {
				bits |= uint64(data[pos+j]) << (j * 8)
			}
			pos += 8
			v = Value{tag: VTagDouble, num: int64(bits)}
		case VTagString:
			strLen, n := readVarintFrom(data[pos:])
			if n <= 0 {
				return nil, errors.New("invalid string length")
			}
			pos += n
			if pos+strLen > len(data) {
				return nil, errors.New("string data truncated")
			}
			v = NewValueString(string(data[pos : pos+strLen]))
			pos += strLen
		case VTagName:
			strLen, n := readVarintFrom(data[pos:])
			if n <= 0 {
				return nil, errors.New("invalid name length")
			}
			pos += n
			if strLen > 0 {
				if pos+strLen > len(data) {
					return nil, errors.New("name data truncated")
				}
				v = NewValueName(GetRName(string(data[pos : pos+strLen])))
				pos += strLen
			} else {
				v = ValueNull
			}
		default:
			// Unknown type - treat as null
			v = ValueNull
		}
		bc.constants = append(bc.constants, v)
	}

	// Read code
	codeLen, n := readVarintFrom(data[pos:])
	if n <= 0 {
		return nil, errors.New("invalid code length")
	}
	pos += n

	if pos+codeLen > len(data) {
		return nil, errors.New("code data truncated")
	}
	bc.code = make([]byte, codeLen)
	copy(bc.code, data[pos:pos+codeLen])

	return bc, nil
}

// readVarintFrom reads a varint from a byte slice.
// Returns the value and number of bytes read, or (0, -1) on error.
func readVarintFrom(data []byte) (int, int) {
	v := 0
	shift := 0
	for i, b := range data {
		v |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			return v, i + 1
		}
		shift += 7
		if shift >= 63 {
			return 0, -1
		}
	}
	return 0, -1
}

// readSvarintFrom reads a signed varint (zigzag encoded) from a byte slice.
// Returns the value and number of bytes read, or (0, -1) on error.
func readSvarintFrom(data []byte) (int64, int) {
	uv, n := readVarintFrom(data)
	if n <= 0 {
		return 0, n
	}
	// Zigzag decode: (uv >> 1) ^ -(uv & 1)
	return int64((uint64(uv) >> 1) ^ uint64(-int64(uv&1))), n
}

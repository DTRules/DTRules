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

import "unsafe"

// Assembly subroutines for hot-path bytecode operations on DTState.
// These operate directly on DTState's fixed-size dataStk array and dataSP.
//
// Return codes:
//   0 = success
//   1 = stack underflow
//   2 = stack overflow
//   3 = not integer (caller should fall back to Go for mixed-type handling)
//   4 = division by zero
//   5 = unknown opcode
//
// All routines take a pointer to DTState. They access:
//   dataStk at offset 0 (array of 24-byte Values)
//   dataSP  at offset 245760 (0x3C000)
//
// Value layout (24 bytes):
//   byte  0:    tag (0=null, 1=integer, 2=double, 3=boolean)
//   bytes 1-7:  padding
//   bytes 8-15: num (int64 for integers/booleans, float64 bits for doubles)
//   bytes 16-23: ptr (unsafe.Pointer, always nil for numeric values)

// Per-opcode functions (used by Go runtime dispatch loop)

// Arithmetic
func asmAdd(state *DTState) int
func asmSub(state *DTState) int
func asmMul(state *DTState) int
func asmDiv(state *DTState) int
func asmNeg(state *DTState) int
func asmInc(state *DTState) int
func asmDec(state *DTState) int
func asmMod(state *DTState) int
func asmAbs(state *DTState) int

// Comparison (result is boolean on stack)
func asmEq(state *DTState) int
func asmNe(state *DTState) int
func asmLt(state *DTState) int
func asmLe(state *DTState) int
func asmGt(state *DTState) int
func asmGe(state *DTState) int

// Boolean
func asmAnd(state *DTState) int
func asmOr(state *DTState) int
func asmNot(state *DTState) int
func asmXor(state *DTState) int

// Stack operations
func asmPop(state *DTState) int
func asmDup(state *DTState) int
func asmSwap(state *DTState) int
func asmRot(state *DTState) int

// Full dispatch loop (used by NativeASM runtime)
//
// asmDispatchLoop owns the entire bytecode execution.
// It handles all opcodes inline for arithmetic, comparison, boolean,
// and stack operations. For opcodes that need Go data structures
// (lookup, def, operator, exec), it calls Go helper functions.
//
// Parameters:
//
//	state:    DTState pointer
//	code:     pointer to bytecode array
//	codeLen:  length of bytecode
//	constPtr: pointer to first element of constants array ([]Value)
//	namePtr:  pointer to first element of names array ([]*RName)
//
// Returns: 0 for success, non-zero for error
func asmDispatchLoop(state *DTState, code unsafe.Pointer, codeLen int, constPtr unsafe.Pointer, namePtr unsafe.Pointer) int

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

// Plan 9 assembly subroutines for hot-path bytecode operations.
// These operate directly on DTState's fixed-size arrays.
//
// DTState layout:
//   dataStk  @ offset 0x000000  - [10240]Value, each Value = 24 bytes
//   dataSP   @ offset 0x03C000  - int (stack pointer)
//
// Value layout (24 bytes):
//   offset 0:  tag   (uint8: 0=null, 1=integer, 2=double, 3=boolean)
//   offset 8:  num   (int64)
//   offset 16: ptr   (pointer, always nil for numeric ops)
//
// Return codes (in AX):
//   0 = success
//   1 = stack underflow
//   2 = stack overflow
//   3 = not integer (fall back to Go)
//   4 = division by zero
//
// Register conventions:
//   R15 = DTState pointer (loaded from first argument)
//   CX  = dataSP value
//   AX  = return value / scratch

#include "textflag.h"
#include "funcdata.h"

// Constants
#define DATASP_OFF  0x3C000    // offset of dataSP in DTState
#define VALUE_SIZE  24         // sizeof(Value)
#define TAG_OFF     0          // offset of tag in Value
#define NUM_OFF     8          // offset of num in Value
#define PTR_OFF     16         // offset of ptr in Value
#define TAG_INTEGER 1
#define TAG_DOUBLE  2
#define TAG_BOOLEAN 3
#define MAX_DEPTH   10240

#define RET_OK        0
#define RET_UNDERFLOW 1
#define RET_OVERFLOW  2
#define RET_NOT_INT   3
#define RET_DIV_ZERO  4
#define RET_UNKNOWN   5

// Opcode constants (must match bytecode.go)
#define OP_NOP        0
#define OP_PUSH       1
#define OP_PUSHINT    2
#define OP_POP        3
#define OP_DUP        4
#define OP_SWAP       5
#define OP_ROT        6

#define OP_ADD        20
#define OP_SUB        21
#define OP_MUL        22
#define OP_DIV        23
#define OP_MOD        24
#define OP_NEG        25
#define OP_ABS        26
#define OP_INC        27
#define OP_DEC        28

#define OP_EQ         30
#define OP_NE         31
#define OP_LT         32
#define OP_LE         33
#define OP_GT         34
#define OP_GE         35

#define OP_AND        40
#define OP_OR         41
#define OP_NOT        42
#define OP_XOR        43

#define OP_EXEC       50
#define OP_LOOKUP     63
#define OP_DEF        62

#define OP_PUSHTRUE   100
#define OP_PUSHFALSE  101
#define OP_PUSHNULL   102
#define OP_PUSHZERO   103
#define OP_PUSHONE    104

#define OP_OPERATOR   200
#define OP_CONSTANT   201
#define OP_NAME       202

#define TAG_NAME      5

// ============================================================================
// Helper macros
// ============================================================================

// LOAD_STATE loads the DTState pointer and dataSP.
// After: R15 = state, CX = dataSP
#define LOAD_STATE \
	MOVQ state+0(FP), R15 \
	MOVQ DATASP_OFF(R15), CX

// STORE_SP stores CX back to dataSP
#define STORE_SP \
	MOVQ CX, DATASP_OFF(R15)

// VALUE_ADDR computes address of dataStk[index] into register dst.
// index must be in a register. Clobbers DX.
// dataStk[i] = R15 + i * 24 = R15 + i * 8 * 3
// We compute: dst = R15 + index*24
#define VALUE_ADDR(index, dst) \
	LEAQ (index)(index*2), DX   /* DX = index * 3 */ \
	SHLQ $3, DX                 /* DX = index * 24 */ \
	LEAQ (R15)(DX*1), dst

// ============================================================================
// Arithmetic operations
// ============================================================================

// func asmAdd(state *DTState) int
// Integer fast path: pops two, pushes sum. Returns 3 if not both integers.
TEXT ·asmAdd(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   add_underflow

	// Address of TOS (index = CX-1) and NOS (index = CX-2)
	MOVQ CX, R8
	DECQ R8               // R8 = CX-1 (TOS index)
	VALUE_ADDR(R8, R9)    // R9 = &dataStk[CX-1]

	MOVQ CX, R10
	SUBQ $2, R10          // R10 = CX-2 (NOS index)
	VALUE_ADDR(R10, R11)  // R11 = &dataStk[CX-2]

	// Check both are integers (tag == 1)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  add_not_int
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_INTEGER
	JNE  add_not_int

	// a.num + b.num -> NOS.num
	MOVQ NUM_OFF(R9), AX   // b.num
	ADDQ AX, NUM_OFF(R11)  // a.num += b.num

	// Clear TOS (tag=0, num=0, ptr=0)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	// dataSP--
	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

add_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

add_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmSub(state *DTState) int
TEXT ·asmSub(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   sub_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)    // TOS

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)  // NOS

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  sub_not_int
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_INTEGER
	JNE  sub_not_int

	// a.num - b.num
	MOVQ NUM_OFF(R9), AX
	SUBQ AX, NUM_OFF(R11)

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

sub_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

sub_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmMul(state *DTState) int
TEXT ·asmMul(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   mul_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  mul_not_int
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_INTEGER
	JNE  mul_not_int

	// a * b -> NOS
	MOVQ NUM_OFF(R9), AX
	IMULQ NUM_OFF(R11), AX
	MOVQ AX, NUM_OFF(R11)

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

mul_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

mul_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmDiv(state *DTState) int
TEXT ·asmDiv(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   div_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  div_not_int
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_INTEGER
	JNE  div_not_int

	// Check divisor != 0
	MOVQ NUM_OFF(R9), BX
	TESTQ BX, BX
	JZ   div_zero

	// a / b (signed) -> NOS
	MOVQ NUM_OFF(R11), AX
	CQO                    // sign-extend AX into DX:AX
	IDIVQ BX               // AX = quotient, DX = remainder
	MOVQ AX, NUM_OFF(R11)

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

div_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

div_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

div_zero:
	MOVQ $RET_DIV_ZERO, ret+8(FP)
	RET

// func asmNeg(state *DTState) int
TEXT ·asmNeg(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   neg_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  neg_not_int

	NEGQ NUM_OFF(R9)

	MOVQ $RET_OK, ret+8(FP)
	RET

neg_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

neg_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmInc(state *DTState) int
TEXT ·asmInc(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   inc_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  inc_not_int

	INCQ NUM_OFF(R9)

	MOVQ $RET_OK, ret+8(FP)
	RET

inc_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

inc_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmDec(state *DTState) int
TEXT ·asmDec(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   dec_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dec_not_int

	DECQ NUM_OFF(R9)

	MOVQ $RET_OK, ret+8(FP)
	RET

dec_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

dec_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmMod(state *DTState) int
TEXT ·asmMod(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   mod_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  mod_not_int
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_INTEGER
	JNE  mod_not_int

	MOVQ NUM_OFF(R9), BX
	TESTQ BX, BX
	JZ   mod_div_zero

	MOVQ NUM_OFF(R11), AX
	CQO
	IDIVQ BX
	MOVQ DX, NUM_OFF(R11)  // remainder

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

mod_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

mod_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

mod_div_zero:
	MOVQ $RET_DIV_ZERO, ret+8(FP)
	RET

// func asmAbs(state *DTState) int
TEXT ·asmAbs(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   abs_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  abs_not_int

	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	JGE  abs_done
	NEGQ AX
	MOVQ AX, NUM_OFF(R9)

abs_done:
	MOVQ $RET_OK, ret+8(FP)
	RET

abs_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

abs_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// ============================================================================
// Comparison operations
// All produce a boolean Value (tag=3, num=0 or 1, ptr=0)
// ============================================================================

// func asmEq(state *DTState) int
TEXT ·asmEq(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   eq_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  eq_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  eq_not_int

	// Compare nums
	MOVQ NUM_OFF(R9), AX
	CMPQ AX, NUM_OFF(R11)

	// Set boolean result in NOS
	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)       // default false
	MOVQ $0, PTR_OFF(R11)
	JNE  eq_false
	MOVQ $1, NUM_OFF(R11)       // true
eq_false:

	// Clear TOS
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

eq_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

eq_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmNe(state *DTState) int
TEXT ·asmNe(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   ne_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  ne_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  ne_not_int

	MOVQ NUM_OFF(R9), AX
	CMPQ AX, NUM_OFF(R11)

	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)
	MOVQ $0, PTR_OFF(R11)
	JE   ne_equal
	MOVQ $1, NUM_OFF(R11)
ne_equal:

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

ne_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

ne_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmLt(state *DTState) int
TEXT ·asmLt(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   lt_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  lt_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  lt_not_int

	// a < b (signed comparison)
	MOVQ NUM_OFF(R11), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a < b ?

	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)
	MOVQ $0, PTR_OFF(R11)
	JGE  lt_false
	MOVQ $1, NUM_OFF(R11)
lt_false:

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

lt_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

lt_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmLe(state *DTState) int
TEXT ·asmLe(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   le_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  le_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  le_not_int

	MOVQ NUM_OFF(R11), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a <= b ?

	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)
	MOVQ $0, PTR_OFF(R11)
	JG   le_false
	MOVQ $1, NUM_OFF(R11)
le_false:

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

le_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

le_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmGt(state *DTState) int
TEXT ·asmGt(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   gt_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  gt_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  gt_not_int

	MOVQ NUM_OFF(R11), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a > b ?

	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)
	MOVQ $0, PTR_OFF(R11)
	JLE  gt_false
	MOVQ $1, NUM_OFF(R11)
gt_false:

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

gt_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

gt_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmGe(state *DTState) int
TEXT ·asmGe(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   ge_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  ge_not_int
	MOVBQZX TAG_OFF(R11), BX
	CMPB BX, $TAG_INTEGER
	JNE  ge_not_int

	MOVQ NUM_OFF(R11), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a >= b ?

	MOVB $TAG_BOOLEAN, TAG_OFF(R11)
	MOVQ $0, NUM_OFF(R11)
	MOVQ $0, PTR_OFF(R11)
	JL   ge_false
	MOVQ $1, NUM_OFF(R11)
ge_false:

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

ge_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

ge_not_int:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// ============================================================================
// Boolean operations
// Operate on boolean Values (tag=3, num=0 or 1)
// ============================================================================

// func asmAnd(state *DTState) int
TEXT ·asmAnd(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   and_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	// Check both are booleans
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  and_not_bool
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  and_not_bool

	// a AND b
	MOVQ NUM_OFF(R9), AX
	ANDQ NUM_OFF(R11), AX
	MOVQ AX, NUM_OFF(R11)
	// tag stays TAG_BOOLEAN, ptr stays 0

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

and_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

and_not_bool:
	MOVQ $RET_NOT_INT, ret+8(FP)    // reuse code 3 for "not expected type"
	RET

// func asmOr(state *DTState) int
TEXT ·asmOr(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   or_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  or_not_bool
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  or_not_bool

	MOVQ NUM_OFF(R9), AX
	ORQ  AX, NUM_OFF(R11)

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

or_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

or_not_bool:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmNot(state *DTState) int
TEXT ·asmNot(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   not_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  not_not_bool

	// Flip: 0->1, 1->0
	MOVQ NUM_OFF(R9), AX
	XORQ $1, AX
	MOVQ AX, NUM_OFF(R9)

	MOVQ $RET_OK, ret+8(FP)
	RET

not_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

not_not_bool:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// func asmXor(state *DTState) int
TEXT ·asmXor(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   xor_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)

	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  xor_not_bool
	MOVBQZX TAG_OFF(R11), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  xor_not_bool

	MOVQ NUM_OFF(R9), AX
	XORQ AX, NUM_OFF(R11)

	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	DECQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

xor_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

xor_not_bool:
	MOVQ $RET_NOT_INT, ret+8(FP)
	RET

// ============================================================================
// Stack operations
// These work with any Value type (no tag checking needed).
// ============================================================================

// func asmPop(state *DTState) int
TEXT ·asmPop(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   pop_underflow

	DECQ CX
	VALUE_ADDR(CX, R9)

	// Clear the popped slot for GC
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)

	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

pop_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

// func asmDup(state *DTState) int
TEXT ·asmDup(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $1
	JL   dup_underflow
	CMPQ CX, $MAX_DEPTH
	JGE  dup_overflow

	// Copy TOS to new TOS
	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)    // source = dataStk[CX-1]

	VALUE_ADDR(CX, R11)   // dest = dataStk[CX]

	// Copy 24 bytes
	MOVQ TAG_OFF(R9), AX
	MOVQ AX, TAG_OFF(R11)
	MOVQ NUM_OFF(R9), AX
	MOVQ AX, NUM_OFF(R11)
	MOVQ PTR_OFF(R9), AX
	MOVQ AX, PTR_OFF(R11)

	INCQ CX
	STORE_SP

	MOVQ $RET_OK, ret+8(FP)
	RET

dup_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

dup_overflow:
	MOVQ $RET_OVERFLOW, ret+8(FP)
	RET

// func asmSwap(state *DTState) int
TEXT ·asmSwap(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $2
	JL   swap_underflow

	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)    // TOS

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)  // NOS

	// Swap 24 bytes using AX and BX as temp
	// tag+padding (8 bytes)
	MOVQ TAG_OFF(R9), AX
	MOVQ TAG_OFF(R11), BX
	MOVQ BX, TAG_OFF(R9)
	MOVQ AX, TAG_OFF(R11)

	// num (8 bytes)
	MOVQ NUM_OFF(R9), AX
	MOVQ NUM_OFF(R11), BX
	MOVQ BX, NUM_OFF(R9)
	MOVQ AX, NUM_OFF(R11)

	// ptr (8 bytes)
	MOVQ PTR_OFF(R9), AX
	MOVQ PTR_OFF(R11), BX
	MOVQ BX, PTR_OFF(R9)
	MOVQ AX, PTR_OFF(R11)

	MOVQ $RET_OK, ret+8(FP)
	RET

swap_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

// func asmRot(state *DTState) int
// ( a b c -- b c a ) rotate top three elements
TEXT ·asmRot(SB), NOSPLIT, $0-16
	LOAD_STATE
	CMPQ CX, $3
	JL   rot_underflow

	// Compute addresses of TOS, TOS-1, TOS-2
	MOVQ CX, R8
	DECQ R8
	VALUE_ADDR(R8, R9)    // c = dataStk[CX-1]

	MOVQ CX, R10
	SUBQ $2, R10
	VALUE_ADDR(R10, R11)  // b = dataStk[CX-2]

	MOVQ CX, R12
	SUBQ $3, R12
	VALUE_ADDR(R12, R13)  // a = dataStk[CX-3]

	// Save 'a' in scratch (use stack-local - but NOSPLIT, so use registers)
	// We need to save 24 bytes of 'a'. Use R14, BX, DI.
	MOVQ TAG_OFF(R13), R14
	MOVQ NUM_OFF(R13), BX
	MOVQ PTR_OFF(R13), DI

	// a = b
	MOVQ TAG_OFF(R11), AX
	MOVQ AX, TAG_OFF(R13)
	MOVQ NUM_OFF(R11), AX
	MOVQ AX, NUM_OFF(R13)
	MOVQ PTR_OFF(R11), AX
	MOVQ AX, PTR_OFF(R13)

	// b = c
	MOVQ TAG_OFF(R9), AX
	MOVQ AX, TAG_OFF(R11)
	MOVQ NUM_OFF(R9), AX
	MOVQ AX, NUM_OFF(R11)
	MOVQ PTR_OFF(R9), AX
	MOVQ AX, PTR_OFF(R11)

	// c = saved 'a'
	MOVQ R14, TAG_OFF(R9)
	MOVQ BX, NUM_OFF(R9)
	MOVQ DI, PTR_OFF(R9)

	MOVQ $RET_OK, ret+8(FP)
	RET

rot_underflow:
	MOVQ $RET_UNDERFLOW, ret+8(FP)
	RET

// ============================================================================
// Full bytecode dispatch loop
//
// func asmDispatchLoop(state *DTState, code unsafe.Pointer, codeLen int,
//                      constPtr unsafe.Pointer, namePtr unsafe.Pointer) int
//
// Owns the entire bytecode execution. All simple opcodes are handled inline.
// Complex opcodes (lookup, def, operator, exec) call Go helper functions.
//
// Register allocation within the dispatch loop:
//   R15 = DTState pointer
//   R14 = bytecode base pointer
//   R13 = bytecode length
//   R12 = PC (program counter)
//   DI  = dataSP (kept in sync with DTState)
//   SI  = constants base pointer
//   BX  = names base pointer
//
// Stack frame: just the Go call area. No save area.
// PC is stored in DTState.asmPC. Everything else reloads from
// function arguments (FP) or DTState. Stack machine -- all operands
// are on the data stack, not in registers.
// ============================================================================

#define ASMPC_OFF  0xA00F8   // offset of asmPC in DTState

// GO_CALL_SAVE: write dataSP and PC back to DTState.
#define GO_CALL_SAVE \
	MOVQ DI, DATASP_OFF(R15) \
	MOVQ R12, ASMPC_OFF(R15)

// GO_CALL_RESTORE: reload from function args and DTState.
#define GO_CALL_RESTORE \
	MOVQ state+0(FP), R15 \
	MOVQ code+8(FP), R14 \
	MOVQ codeLen+16(FP), R13 \
	MOVQ constPtr+24(FP), SI \
	MOVQ namePtr+32(FP), BX \
	MOVQ DATASP_OFF(R15), DI \
	MOVQ ASMPC_OFF(R15), R12

// SYNC_SP writes DI back to DTState.dataSP.
#define SYNC_SP \
	MOVQ DI, DATASP_OFF(R15)

// DL_VALUE_ADDR computes address of dataStk[index] into register dst.
// Same as VALUE_ADDR but doesn't clobber DX (uses CX instead).
#define DL_VALUE_ADDR(index, dst) \
	LEAQ (index)(index*2), CX \
	SHLQ $3, CX \
	LEAQ (R15)(CX*1), dst

TEXT ·asmDispatchLoop(SB), $24-48
	NO_LOCAL_POINTERS
	// Load arguments
	MOVQ state+0(FP), R15       // DTState pointer
	MOVQ code+8(FP), R14        // bytecode base pointer
	MOVQ codeLen+16(FP), R13    // bytecode length
	MOVQ constPtr+24(FP), SI    // constants base pointer
	MOVQ namePtr+32(FP), BX     // names base pointer
	XORQ R12, R12               // PC = 0
	MOVQ DATASP_OFF(R15), DI    // DI = dataSP

dl_dispatch:
	CMPQ R12, R13
	JGE  dl_done                // PC >= codeLen -> done

	// Fetch opcode
	MOVBQZX (R14)(R12*1), AX   // AX = code[PC]
	INCQ R12                    // PC++

	// Dispatch (if-else chain ordered by frequency)
	// Push operations
	CMPB AX, $OP_PUSHTRUE
	JE   dl_pushtrue
	CMPB AX, $OP_PUSHFALSE
	JE   dl_pushfalse
	CMPB AX, $OP_PUSHNULL
	JE   dl_pushnull
	CMPB AX, $OP_PUSHZERO
	JE   dl_pushzero
	CMPB AX, $OP_PUSHONE
	JE   dl_pushone
	CMPB AX, $OP_PUSHINT
	JE   dl_pushint
	CMPB AX, $OP_CONSTANT
	JE   dl_constant
	CMPB AX, $OP_NAME
	JE   dl_name

	// Arithmetic
	CMPB AX, $OP_ADD
	JE   dl_add
	CMPB AX, $OP_SUB
	JE   dl_sub
	CMPB AX, $OP_MUL
	JE   dl_mul
	CMPB AX, $OP_DIV
	JE   dl_div
	CMPB AX, $OP_MOD
	JE   dl_mod
	CMPB AX, $OP_NEG
	JE   dl_neg
	CMPB AX, $OP_ABS
	JE   dl_abs
	CMPB AX, $OP_INC
	JE   dl_inc
	CMPB AX, $OP_DEC
	JE   dl_dec

	// Comparison
	CMPB AX, $OP_EQ
	JE   dl_eq
	CMPB AX, $OP_NE
	JE   dl_ne
	CMPB AX, $OP_LT
	JE   dl_lt
	CMPB AX, $OP_LE
	JE   dl_le
	CMPB AX, $OP_GT
	JE   dl_gt
	CMPB AX, $OP_GE
	JE   dl_ge

	// Boolean
	CMPB AX, $OP_AND
	JE   dl_and
	CMPB AX, $OP_OR
	JE   dl_or
	CMPB AX, $OP_NOT
	JE   dl_not
	CMPB AX, $OP_XOR
	JE   dl_xor

	// Stack
	CMPB AX, $OP_POP
	JE   dl_pop
	CMPB AX, $OP_DUP
	JE   dl_dup
	CMPB AX, $OP_SWAP
	JE   dl_swap
	CMPB AX, $OP_ROT
	JE   dl_rot

	// Complex opcodes (call Go)
	CMPB AX, $OP_LOOKUP
	JE   dl_lookup
	CMPB AX, $OP_DEF
	JE   dl_def
	CMPB AX, $OP_OPERATOR
	JE   dl_operator
	CMPB AX, $OP_EXEC
	JE   dl_exec

	// Nop
	CMPB AX, $OP_NOP
	JE   dl_dispatch

	// Unknown opcode
	MOVQ $RET_UNKNOWN, ret+40(FP)
	RET

dl_done:
	SYNC_SP
	MOVQ $RET_OK, ret+40(FP)
	RET

// ---- Push operations (inline) ----

dl_pushtrue:
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	DL_VALUE_ADDR(DI, R8)
	MOVB $TAG_BOOLEAN, TAG_OFF(R8)
	MOVQ $1, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushfalse:
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	DL_VALUE_ADDR(DI, R8)
	MOVB $TAG_BOOLEAN, TAG_OFF(R8)
	MOVQ $0, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushnull:
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	DL_VALUE_ADDR(DI, R8)
	MOVQ $0, TAG_OFF(R8)
	MOVQ $0, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushzero:
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	DL_VALUE_ADDR(DI, R8)
	MOVB $TAG_INTEGER, TAG_OFF(R8)
	MOVQ $0, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushone:
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	DL_VALUE_ADDR(DI, R8)
	MOVB $TAG_INTEGER, TAG_OFF(R8)
	MOVQ $1, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushint:
	// Read varint, push as integer
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	// Inline varint read: result in AX, updates R12
	XORQ AX, AX
	MOVBQZX (R14)(R12*1), AX
	INCQ R12
	TESTB $0x80, AX
	JNZ  dl_pushint_multi
dl_pushint_store:
	DL_VALUE_ADDR(DI, R8)
	MOVB $TAG_INTEGER, TAG_OFF(R8)
	MOVQ AX, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	INCQ DI
	JMP  dl_dispatch

dl_pushint_multi:
	// Multi-byte varint
	ANDQ $0x7F, AX
	MOVQ $7, CX              // shift = 7
dl_pushint_loop:
	MOVBQZX (R14)(R12*1), DX
	INCQ R12
	MOVQ DX, R8
	ANDQ $0x7F, R8
	SHLQ CL, R8
	ORQ  R8, AX
	ADDQ $7, CX
	TESTB $0x80, DX
	JNZ  dl_pushint_loop
	JMP  dl_pushint_store

dl_constant:
	// Read varint index, push constants[idx]
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	// Read varint -> AX
	XORQ AX, AX
	MOVBQZX (R14)(R12*1), AX
	INCQ R12
	TESTB $0x80, AX
	JNZ  dl_constant_multi
dl_constant_load:
	// AX = constant index. Load constants[AX] (24 bytes each)
	// Address = SI + AX * 24
	LEAQ (AX)(AX*2), CX      // CX = AX * 3
	SHLQ $3, CX              // CX = AX * 24
	LEAQ (SI)(CX*1), R8      // R8 = &constants[AX]
	DL_VALUE_ADDR(DI, R9)    // R9 = &dataStk[DI]
	// Copy 24 bytes
	MOVQ TAG_OFF(R8), R10
	MOVQ R10, TAG_OFF(R9)
	MOVQ NUM_OFF(R8), R10
	MOVQ R10, NUM_OFF(R9)
	MOVQ PTR_OFF(R8), R10
	MOVQ R10, PTR_OFF(R9)
	INCQ DI
	JMP  dl_dispatch

dl_constant_multi:
	ANDQ $0x7F, AX
	MOVQ $7, CX
dl_constant_mloop:
	MOVBQZX (R14)(R12*1), DX
	INCQ R12
	MOVQ DX, R8
	ANDQ $0x7F, R8
	SHLQ CL, R8
	ORQ  R8, AX
	ADDQ $7, CX
	TESTB $0x80, DX
	JNZ  dl_constant_mloop
	JMP  dl_constant_load

dl_name:
	// Read varint index, push names[idx] as Value with tag=VTagName
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	// Read varint -> AX
	XORQ AX, AX
	MOVBQZX (R14)(R12*1), AX
	INCQ R12
	TESTB $0x80, AX
	JNZ  dl_name_multi
dl_name_load:
	// AX = name index. Load names[AX] (pointer, 8 bytes each)
	// Address = BX + AX * 8
	MOVQ (BX)(AX*8), R8      // R8 = names[AX] (*RName)
	DL_VALUE_ADDR(DI, R9)    // R9 = &dataStk[DI]
	MOVB $TAG_NAME, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ R8, PTR_OFF(R9)
	INCQ DI
	JMP  dl_dispatch

dl_name_multi:
	ANDQ $0x7F, AX
	MOVQ $7, CX
dl_name_mloop:
	MOVBQZX (R14)(R12*1), DX
	INCQ R12
	MOVQ DX, R8
	ANDQ $0x7F, R8
	SHLQ CL, R8
	ORQ  R8, AX
	ADDQ $7, CX
	TESTB $0x80, DX
	JNZ  dl_name_mloop
	JMP  dl_name_load

// ---- Stack operations (inline) ----

dl_pop:
	CMPQ DI, $1
	JL   dl_underflow
	DECQ DI
	DL_VALUE_ADDR(DI, R8)
	MOVQ $0, TAG_OFF(R8)
	MOVQ $0, NUM_OFF(R8)
	MOVQ $0, PTR_OFF(R8)
	JMP  dl_dispatch

dl_dup:
	CMPQ DI, $1
	JL   dl_underflow
	CMPQ DI, $MAX_DEPTH
	JGE  dl_overflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)    // source
	DL_VALUE_ADDR(DI, R10)   // dest (CX clobbered, but we saved DI)
	MOVQ TAG_OFF(R9), R11
	MOVQ R11, TAG_OFF(R10)
	MOVQ NUM_OFF(R9), R11
	MOVQ R11, NUM_OFF(R10)
	MOVQ PTR_OFF(R9), R11
	MOVQ R11, PTR_OFF(R10)
	INCQ DI
	JMP  dl_dispatch

dl_swap:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)    // TOS
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)   // NOS
	// Swap 24 bytes
	MOVQ TAG_OFF(R9), R11
	MOVQ TAG_OFF(R10), R8
	MOVQ R8, TAG_OFF(R9)
	MOVQ R11, TAG_OFF(R10)
	MOVQ NUM_OFF(R9), R11
	MOVQ NUM_OFF(R10), R8
	MOVQ R8, NUM_OFF(R9)
	MOVQ R11, NUM_OFF(R10)
	MOVQ PTR_OFF(R9), R11
	MOVQ PTR_OFF(R10), R8
	MOVQ R8, PTR_OFF(R9)
	MOVQ R11, PTR_OFF(R10)
	JMP  dl_dispatch

dl_rot:
	CMPQ DI, $3
	JL   dl_underflow
	// ( a b c -- b c a )
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)    // c = dataStk[DI-1]
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)   // b = dataStk[DI-2]
	MOVQ DI, R8
	SUBQ $3, R8
	DL_VALUE_ADDR(R8, R11)   // a = dataStk[DI-3]
	// Save a
	MOVQ TAG_OFF(R11), AX
	MOVQ NUM_OFF(R11), CX
	MOVQ PTR_OFF(R11), DX
	// a = b
	MOVQ TAG_OFF(R10), R8
	MOVQ R8, TAG_OFF(R11)
	MOVQ NUM_OFF(R10), R8
	MOVQ R8, NUM_OFF(R11)
	MOVQ PTR_OFF(R10), R8
	MOVQ R8, PTR_OFF(R11)
	// b = c
	MOVQ TAG_OFF(R9), R8
	MOVQ R8, TAG_OFF(R10)
	MOVQ NUM_OFF(R9), R8
	MOVQ R8, NUM_OFF(R10)
	MOVQ PTR_OFF(R9), R8
	MOVQ R8, PTR_OFF(R10)
	// c = saved a
	MOVQ AX, TAG_OFF(R9)
	MOVQ CX, NUM_OFF(R9)
	MOVQ DX, PTR_OFF(R9)
	JMP  dl_dispatch

// ---- Arithmetic (inline, integer fast path + SSE2 double fallback) ----
//
// Binary arithmetic: integer fast path, then SSE2 double fallback.
// Both operands are loaded as doubles (CVTSQ2SD for int, MOVSD for double).
// Result is TAG_DOUBLE. No Go calls needed.

dl_add:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)    // TOS
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)   // NOS
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_add_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_add_mixed
	// Both integers: fast path
	MOVQ NUM_OFF(R9), AX
	ADDQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_add_mixed:
	// SSE2 double fallback: load both as double, ADDSD, store as TAG_DOUBLE
	// Load TOS as double into X0
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_add_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_add_load_nos
dl_add_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_add_load_nos:
	// Load NOS as double into X1
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_add_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_add_compute
dl_add_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_add_compute:
	ADDSD X0, X1
	// Store result in NOS as TAG_DOUBLE
	MOVB $TAG_DOUBLE, TAG_OFF(R10)
	MOVSD X1, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	// Clear TOS
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_sub:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_sub_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_sub_mixed
	MOVQ NUM_OFF(R9), AX
	SUBQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_sub_mixed:
	// Load TOS into X0
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_sub_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_sub_load_nos
dl_sub_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_sub_load_nos:
	// Load NOS into X1
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_sub_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_sub_compute
dl_sub_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_sub_compute:
	SUBSD X0, X1            // X1 = NOS - TOS
	MOVB $TAG_DOUBLE, TAG_OFF(R10)
	MOVSD X1, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_mul:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_mul_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_mul_mixed
	MOVQ NUM_OFF(R9), AX
	IMULQ NUM_OFF(R10), AX
	MOVQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_mul_mixed:
	// Load TOS into X0
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_mul_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_mul_load_nos
dl_mul_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_mul_load_nos:
	// Load NOS into X1
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_mul_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_mul_compute
dl_mul_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_mul_compute:
	MULSD X0, X1
	MOVB $TAG_DOUBLE, TAG_OFF(R10)
	MOVSD X1, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_div:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_div_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_div_mixed
	// Both integers: check for zero, do integer division
	MOVQ NUM_OFF(R9), CX
	TESTQ CX, CX
	JZ   dl_div_zero
	MOVQ NUM_OFF(R10), AX
	CQO
	IDIVQ CX
	MOVQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_div_mixed:
	// Load TOS (divisor) into X0
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_div_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_div_check_zero
dl_div_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_div_check_zero:
	// Check for zero divisor
	XORPD X2, X2
	UCOMISD X2, X0
	JP   dl_div_do            // NaN is not zero, proceed
	JE   dl_div_zero          // divisor is zero
dl_div_do:
	// Load NOS (dividend) into X1
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_div_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_div_compute
dl_div_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_div_compute:
	DIVSD X0, X1            // X1 = NOS / TOS
	MOVB $TAG_DOUBLE, TAG_OFF(R10)
	MOVSD X1, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_div_zero:
	SYNC_SP
	MOVQ $RET_DIV_ZERO, ret+40(FP)
	RET

dl_mod:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_mod_type_err
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_mod_type_err
	MOVQ NUM_OFF(R9), CX
	TESTQ CX, CX
	JZ   dl_div_zero
	MOVQ NUM_OFF(R10), AX
	CQO
	IDIVQ CX
	MOVQ DX, NUM_OFF(R10)   // remainder
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_mod_type_err:
	SYNC_SP
	MOVQ $RET_NOT_INT, ret+40(FP)
	RET

dl_neg:
	CMPQ DI, $1
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_neg_dbl
	NEGQ NUM_OFF(R9)
	JMP  dl_dispatch
dl_neg_dbl:
	// Double negate: XOR the sign bit (bit 63)
	CMPB AX, $TAG_DOUBLE
	JNE  dl_dispatch          // non-numeric: no-op (safety)
	MOVQ NUM_OFF(R9), AX
	MOVQ $0x8000000000000000, DX
	XORQ DX, AX
	MOVQ AX, NUM_OFF(R9)
	JMP  dl_dispatch

dl_abs:
	CMPQ DI, $1
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_abs_dbl
	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	JGE  dl_dispatch          // already positive
	NEGQ AX
	MOVQ AX, NUM_OFF(R9)
	JMP  dl_dispatch
dl_abs_dbl:
	// Double abs: AND away the sign bit (bit 63)
	CMPB AX, $TAG_DOUBLE
	JNE  dl_dispatch          // non-numeric: no-op (safety)
	MOVQ NUM_OFF(R9), AX
	MOVQ $0x7FFFFFFFFFFFFFFF, DX
	ANDQ DX, AX
	MOVQ AX, NUM_OFF(R9)
	JMP  dl_dispatch

dl_inc:
	CMPQ DI, $1
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_inc_dbl
	INCQ NUM_OFF(R9)
	JMP  dl_dispatch
dl_inc_dbl:
	// Double inc: convert to double, add 1.0
	CMPB AX, $TAG_DOUBLE
	JNE  dl_dispatch          // non-numeric: no-op
	MOVSD NUM_OFF(R9), X0
	MOVQ $0x3FF0000000000000, AX   // 1.0 in float64 bits
	MOVQ AX, X1
	ADDSD X1, X0
	MOVSD X0, NUM_OFF(R9)
	JMP  dl_dispatch

dl_dec:
	CMPQ DI, $1
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_dec_dbl
	DECQ NUM_OFF(R9)
	JMP  dl_dispatch
dl_dec_dbl:
	// Double dec: convert to double, subtract 1.0
	CMPB AX, $TAG_DOUBLE
	JNE  dl_dispatch          // non-numeric: no-op
	MOVSD NUM_OFF(R9), X0
	MOVQ $0x3FF0000000000000, AX   // 1.0 in float64 bits
	MOVQ AX, X1
	SUBSD X1, X0
	MOVSD X0, NUM_OFF(R9)
	JMP  dl_dispatch

// ---- Comparison (inline, integer fast path + SSE2 double fallback) ----
//
// For numeric types (int+double mix), convert both to double and UCOMISD.
// For non-numeric types, result is false for eq (not equal), true for ne.

dl_eq:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_eq_check_dbl
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_eq_mixed
	// Both integers: fast path
	MOVQ NUM_OFF(R9), AX
	CMPQ AX, NUM_OFF(R10)
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JNE  dl_eq_done
	MOVQ $1, NUM_OFF(R10)
dl_eq_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_eq_check_dbl:
	// TOS is not integer. Check if both are numeric (int or double)
	CMPB AX, $TAG_DOUBLE
	JNE  dl_eq_non_numeric
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JE   dl_eq_mixed
	CMPB DX, $TAG_DOUBLE
	JE   dl_eq_mixed
	JMP  dl_eq_non_numeric
dl_eq_mixed:
	// At least one numeric, load both as double
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_eq_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_eq_load_nos
dl_eq_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_eq_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_eq_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_eq_ucomi
dl_eq_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_eq_ucomi:
	UCOMISD X0, X1
	// Result: equal if ZF=1 and PF=0
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_eq_mixed_done     // parity = NaN -> false
	JNE  dl_eq_mixed_done     // not equal -> false
	MOVQ $1, NUM_OFF(R10)
dl_eq_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_eq_non_numeric:
	// Non-numeric eq: false (different types or unsupported)
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_ne:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ne_check_dbl
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_ne_mixed
	// Both integers: fast path
	MOVQ NUM_OFF(R9), AX
	CMPQ AX, NUM_OFF(R10)
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JE   dl_ne_done
	MOVQ $1, NUM_OFF(R10)
dl_ne_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_ne_check_dbl:
	CMPB AX, $TAG_DOUBLE
	JNE  dl_ne_non_numeric
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JE   dl_ne_mixed
	CMPB DX, $TAG_DOUBLE
	JE   dl_ne_mixed
	JMP  dl_ne_non_numeric
dl_ne_mixed:
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ne_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_ne_load_nos
dl_ne_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_ne_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ne_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_ne_ucomi
dl_ne_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_ne_ucomi:
	UCOMISD X0, X1
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_ne_mixed_true     // parity = NaN -> not equal = true
	JE   dl_ne_mixed_done     // equal -> false
dl_ne_mixed_true:
	MOVQ $1, NUM_OFF(R10)
dl_ne_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_ne_non_numeric:
	// Non-numeric ne: true (different types)
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $1, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_lt:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_lt_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_lt_mixed
	// Both integers
	MOVQ NUM_OFF(R10), AX    // a (NOS)
	CMPQ AX, NUM_OFF(R9)     // a < b ?
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JGE  dl_lt_done
	MOVQ $1, NUM_OFF(R10)
dl_lt_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_lt_mixed:
	// Load TOS (b) into X0, NOS (a) into X1
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_lt_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_lt_load_nos
dl_lt_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_lt_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_lt_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_lt_ucomi
dl_lt_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_lt_ucomi:
	// UCOMISD X0, X1 compares X1 vs X0 (X1 < X0 means NOS < TOS)
	UCOMISD X0, X1
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_lt_mixed_done     // NaN -> false
	JAE  dl_lt_mixed_done     // X1 >= X0 -> false
	MOVQ $1, NUM_OFF(R10)
dl_lt_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_le:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_le_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_le_mixed
	MOVQ NUM_OFF(R10), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a <= b ?
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JG   dl_le_done
	MOVQ $1, NUM_OFF(R10)
dl_le_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_le_mixed:
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_le_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_le_load_nos
dl_le_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_le_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_le_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_le_ucomi
dl_le_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_le_ucomi:
	// a <= b: UCOMISD X0, X1 (X0=TOS=b, X1=NOS=a), true if X1 <= X0 (CF=1 or ZF=1)
	UCOMISD X0, X1
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_le_mixed_done     // NaN -> false
	JA   dl_le_mixed_done     // X1 > X0 -> false
	MOVQ $1, NUM_OFF(R10)
dl_le_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_gt:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_gt_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_gt_mixed
	MOVQ NUM_OFF(R10), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a > b ?
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JLE  dl_gt_done
	MOVQ $1, NUM_OFF(R10)
dl_gt_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_gt_mixed:
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_gt_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_gt_load_nos
dl_gt_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_gt_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_gt_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_gt_ucomi
dl_gt_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_gt_ucomi:
	// a > b: UCOMISD X1, X0 (X0=TOS=b, X1=NOS=a), true if X1 > X0 (CF=0 and ZF=0)
	UCOMISD X1, X0
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_gt_mixed_done     // NaN -> false
	JAE  dl_gt_mixed_done     // X0 >= X1 -> a <= b -> false
	MOVQ $1, NUM_OFF(R10)
dl_gt_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_ge:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ge_mixed
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_INTEGER
	JNE  dl_ge_mixed
	MOVQ NUM_OFF(R10), AX    // a
	CMPQ AX, NUM_OFF(R9)     // a >= b ?
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JL   dl_ge_done
	MOVQ $1, NUM_OFF(R10)
dl_ge_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_ge_mixed:
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ge_tos_dbl
	MOVQ NUM_OFF(R9), AX
	CVTSQ2SD AX, X0
	JMP  dl_ge_load_nos
dl_ge_tos_dbl:
	MOVSD NUM_OFF(R9), X0
dl_ge_load_nos:
	MOVBQZX TAG_OFF(R10), AX
	CMPB AX, $TAG_INTEGER
	JNE  dl_ge_nos_dbl
	MOVQ NUM_OFF(R10), AX
	CVTSQ2SD AX, X1
	JMP  dl_ge_ucomi
dl_ge_nos_dbl:
	MOVSD NUM_OFF(R10), X1
dl_ge_ucomi:
	// a >= b: UCOMISD X1, X0 (X0=TOS=b, X1=NOS=a), true if X1 >= X0 (CF=0)
	UCOMISD X1, X0
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ $0, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	JP   dl_ge_mixed_done     // NaN -> false
	JA   dl_ge_mixed_done     // X0 > X1 -> a < b -> false
	MOVQ $1, NUM_OFF(R10)
dl_ge_mixed_done:
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

// ---- Boolean (inline, boolean fast path + coercion fallback) ----
//
// For non-boolean types, coerce with TESTQ num: non-zero = true (1), zero = false (0).
// No Go calls needed.

dl_and:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  dl_and_coerce
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_BOOLEAN
	JNE  dl_and_coerce
	// Both booleans: fast path
	MOVQ NUM_OFF(R9), AX
	ANDQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_and_coerce:
	// Coerce both to boolean: TESTQ num, SETNE -> 0 or 1
	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	SETNE AX                  // AX = (num != 0) ? 1 : 0
	MOVQ NUM_OFF(R10), DX
	TESTQ DX, DX
	SETNE DX
	ANDB DX, AX               // AND the booleans
	MOVBQZX AX, AX
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ AX, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_or:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  dl_or_coerce
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_BOOLEAN
	JNE  dl_or_coerce
	MOVQ NUM_OFF(R9), AX
	ORQ  AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_or_coerce:
	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	SETNE AX
	MOVQ NUM_OFF(R10), DX
	TESTQ DX, DX
	SETNE DX
	ORB  DX, AX
	MOVBQZX AX, AX
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ AX, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

dl_not:
	CMPQ DI, $1
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  dl_not_coerce
	// Boolean fast path
	MOVQ NUM_OFF(R9), AX
	XORQ $1, AX
	MOVQ AX, NUM_OFF(R9)
	JMP  dl_dispatch
dl_not_coerce:
	// Coerce: num==0 -> true(1), num!=0 -> false(0)
	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	SETEQ AX                  // AX = (num == 0) ? 1 : 0
	MOVBQZX AX, AX
	MOVB $TAG_BOOLEAN, TAG_OFF(R9)
	MOVQ AX, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	JMP  dl_dispatch

dl_xor:
	CMPQ DI, $2
	JL   dl_underflow
	MOVQ DI, R8
	DECQ R8
	DL_VALUE_ADDR(R8, R9)
	MOVQ DI, R8
	SUBQ $2, R8
	DL_VALUE_ADDR(R8, R10)
	MOVBQZX TAG_OFF(R9), AX
	CMPB AX, $TAG_BOOLEAN
	JNE  dl_xor_coerce
	MOVBQZX TAG_OFF(R10), DX
	CMPB DX, $TAG_BOOLEAN
	JNE  dl_xor_coerce
	MOVQ NUM_OFF(R9), AX
	XORQ AX, NUM_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch
dl_xor_coerce:
	MOVQ NUM_OFF(R9), AX
	TESTQ AX, AX
	SETNE AX
	MOVQ NUM_OFF(R10), DX
	TESTQ DX, DX
	SETNE DX
	XORB DX, AX
	MOVBQZX AX, AX
	MOVB $TAG_BOOLEAN, TAG_OFF(R10)
	MOVQ AX, NUM_OFF(R10)
	MOVQ $0, PTR_OFF(R10)
	MOVQ $0, TAG_OFF(R9)
	MOVQ $0, NUM_OFF(R9)
	MOVQ $0, PTR_OFF(R9)
	DECQ DI
	JMP  dl_dispatch

// ---- Complex opcodes (call Go helpers) ----

dl_lookup:
	SYNC_SP
	GO_CALL_SAVE
	MOVQ R15, 0(SP)
	CALL ·goOpLookup(SB)
	MOVQ 8(SP), AX
	GO_CALL_RESTORE
	TESTQ AX, AX
	JNZ  dl_error_from_go
	JMP  dl_dispatch

dl_def:
	SYNC_SP
	GO_CALL_SAVE
	MOVQ R15, 0(SP)
	CALL ·goOpDef(SB)
	MOVQ 8(SP), AX
	GO_CALL_RESTORE
	TESTQ AX, AX
	JNZ  dl_error_from_go
	JMP  dl_dispatch

dl_operator:
	// Read varint for operator index first
	XORQ AX, AX
	MOVBQZX (R14)(R12*1), AX
	INCQ R12
	TESTB $0x80, AX
	JNZ  dl_operator_multi
dl_operator_call:
	// AX = operator index
	SYNC_SP
	GO_CALL_SAVE
	MOVQ R15, 0(SP)           // arg1: state
	MOVQ AX, 8(SP)            // arg2: opIdx
	CALL ·goOpOperator(SB)
	MOVQ 16(SP), AX
	GO_CALL_RESTORE
	TESTQ AX, AX
	JNZ  dl_error_from_go
	JMP  dl_dispatch

dl_operator_multi:
	ANDQ $0x7F, AX
	MOVQ $7, CX
dl_operator_mloop:
	MOVBQZX (R14)(R12*1), DX
	INCQ R12
	MOVQ DX, R8
	ANDQ $0x7F, R8
	SHLQ CL, R8
	ORQ  R8, AX
	ADDQ $7, CX
	TESTB $0x80, DX
	JNZ  dl_operator_mloop
	JMP  dl_operator_call

dl_exec:
	SYNC_SP
	GO_CALL_SAVE
	MOVQ R15, 0(SP)
	CALL ·goOpExec(SB)
	MOVQ 8(SP), AX
	GO_CALL_RESTORE
	TESTQ AX, AX
	JNZ  dl_error_from_go
	JMP  dl_dispatch

// ---- Error exits ----

dl_underflow:
	SYNC_SP
	MOVQ $RET_UNDERFLOW, ret+40(FP)
	RET

dl_overflow:
	SYNC_SP
	MOVQ $RET_OVERFLOW, ret+40(FP)
	RET

dl_error_from_go:
	// Go helper set state.lastError, return non-zero
	SYNC_SP
	MOVQ AX, ret+40(FP)
	RET

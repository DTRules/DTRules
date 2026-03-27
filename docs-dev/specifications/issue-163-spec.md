# Specification: NASM VM Stack Operations

**Issue**: #163
**Version**: 1.0
**Status**: Implemented

## Overview

This specification defines the stack operations for the DTRules NASM VM, a stack-based bytecode interpreter implemented in x86-64 assembly using NASM.

## Architecture

### Value Representation

Each value on the stack occupies 16 bytes:

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| +0 | 8 bytes | tag | Type tag (VTAG_*) |
| +8 | 8 bytes | value | Integer, boolean, or pointer |

### Type Tags

| Tag | Value | Description |
|-----|-------|-------------|
| VTAG_NULL | 0 | Null value |
| VTAG_BOOLEAN | 1 | Boolean (0 or 1) |
| VTAG_INTEGER | 2 | 64-bit signed integer |
| VTAG_DOUBLE | 3 | 64-bit floating point |
| VTAG_STRING | 4 | String pointer |
| VTAG_NAME | 5 | Name reference |
| VTAG_ARRAY | 6 | Array pointer |
| VTAG_ENTITY | 7 | Entity pointer |
| VTAG_MARK | 8 | Stack mark |

### Register Conventions

| Register | Purpose |
|----------|---------|
| rdi | Bytecode pointer (instruction pointer) |
| rsi | Bytecode end pointer |
| r12 | Data stack pointer (grows upward) |
| r13 | Data stack base (underflow limit) |
| r14 | Data stack end (overflow limit) |
| r15 | Trace buffer pointer (optional) |
| rbx | Context pointer |

## Stack Operations

### OP_NOP (0)

**Signature**: `( -- )`

**INPUT**: None
**OPERATION**: No operation
**OUTPUT**: Stack unchanged

**Precision Rules**: None

**Example 1**:
- Before: `[42]`
- After: `[42]`

**Example 2**:
- Before: `[]`
- After: `[]`

---

### OP_PUSH_INT (2)

**Signature**: `( -- n )`

**INPUT**:
- Varint encoded integer from bytecode stream

**OPERATION**:
1. Read varint from bytecode
2. Push integer with VTAG_INTEGER tag

**OUTPUT**:
- Integer value on stack

**Precision Rules**:
- Varint encoding: continuation bit in MSB (0x80)
- Value bits in lower 7 bits
- Sign extension not implemented (unsigned only)

**Example 1**:
- Bytecode: `[0x02, 42]`
- Before: `[]`
- After: `[42]`

**Example 2**:
- Bytecode: `[0x02, 0xC8, 0x01]` (200 in varint)
- Before: `[]`
- After: `[200]`

---

### OP_POP (3) / OP_DROP (107)

**Signature**: `( a -- )`

**INPUT**:
- One value on stack (minimum)

**OPERATION**:
1. Check for underflow (r12 == r13)
2. Decrement stack pointer by 16

**OUTPUT**:
- Top value removed

**Precision Rules**:
- Returns ERR_STACK_UNDERFLOW (1) if stack empty

**Example 1**:
- Before: `[42, 100]`
- After: `[42]`

**Example 2** (error case):
- Before: `[]`
- Result: ERR_STACK_UNDERFLOW

---

### OP_DUP (4)

**Signature**: `( a -- a a )`

**INPUT**:
- One value on stack (minimum)

**OPERATION**:
1. Check for underflow
2. Check for overflow
3. Copy tag and value from TOS
4. Push copy

**OUTPUT**:
- Top value duplicated

**Precision Rules**:
- Returns ERR_STACK_UNDERFLOW if stack empty
- Returns ERR_STACK_OVERFLOW if stack full

**Example 1**:
- Before: `[42]`
- After: `[42, 42]`

**Example 2**:
- Before: `[1, 2, 3]`
- After: `[1, 2, 3, 3]`

---

### OP_SWAP (5)

**Signature**: `( a b -- b a )`

**INPUT**:
- Two values on stack (minimum)

**OPERATION**:
1. Check for underflow (need 32 bytes)
2. Load both values
3. Store in swapped positions

**OUTPUT**:
- Top two values swapped

**Precision Rules**:
- Returns ERR_STACK_UNDERFLOW if fewer than 2 values

**Example 1**:
- Before: `[1, 2]`
- After: `[2, 1]`

**Example 2**:
- Before: `[10, 20, 30]`
- After: `[10, 30, 20]`

---

### OP_ROT (6)

**Signature**: `( a b c -- b c a )`

**INPUT**:
- Three values on stack (minimum)

**OPERATION**:
1. Check for underflow (need 48 bytes)
2. Load all three values
3. Rotate: third to top, shift others down

**OUTPUT**:
- Top three values rotated

**Precision Rules**:
- Returns ERR_STACK_UNDERFLOW if fewer than 3 values

**Example 1**:
- Before: `[1, 2, 3]`
- After: `[2, 3, 1]`

**Example 2**:
- Before: `[10, 20, 30, 40]`
- After: `[10, 30, 40, 20]`

---

### OP_ROLL (7)

**Signature**: `( ... n -- ... )`

**INPUT**:
- n: number of elements to roll
- n+1 elements on stack

**OPERATION**:
1. Pop n from stack
2. Roll n elements (NOT FULLY IMPLEMENTED)

**OUTPUT**:
- Elements rolled

**Precision Rules**:
- Current implementation is a placeholder (just pops n)

**Example**: (Placeholder - needs full implementation)

---

### OP_INDEX (8) / OP_PICK (106)

**Signature**: `( ... n -- ... nth )`

**INPUT**:
- n: 0-based index from top of stack
- n+2 elements on stack minimum

**OPERATION**:
1. Pop n from stack
2. Calculate offset: (n+1)*16 bytes
3. Check bounds
4. Copy nth element to top

**OUTPUT**:
- Copy of nth element pushed

**Precision Rules**:
- n=0 copies top element (like DUP after pop)
- n=1 copies second element
- Returns ERR_STACK_UNDERFLOW if index out of bounds

**Example 1**:
- Before: `[100, 200, 300, 1]` (pick index 1)
- After: `[100, 200, 300, 200]`

**Example 2**:
- Before: `[10, 20, 30, 0]` (pick index 0)
- After: `[10, 20, 30, 30]`

---

### OP_CLEAR (9)

**Signature**: `( ... -- )`

**INPUT**:
- Any number of values

**OPERATION**:
1. Reset stack pointer to base (r12 = r13)

**OUTPUT**:
- Empty stack

**Precision Rules**:
- Always succeeds
- Does not clear to mark (clears entire stack)

**Example 1**:
- Before: `[1, 2, 3, 4, 5]`
- After: `[]`

**Example 2**:
- Before: `[]`
- After: `[]`

---

### OP_MARK (10)

**Signature**: `( -- mark )`

**INPUT**:
- None

**OPERATION**:
1. Check for overflow
2. Push value with VTAG_MARK tag

**OUTPUT**:
- Mark value on stack

**Precision Rules**:
- Mark value field is 0
- Returns ERR_STACK_OVERFLOW if stack full

**Example 1**:
- Before: `[1, 2]`
- After: `[1, 2, <mark>]`

**Example 2**:
- Before: `[]`
- After: `[<mark>]`

---

### OP_OVER (105)

**Signature**: `( a b -- a b a )`

**INPUT**:
- Two values on stack (minimum)

**OPERATION**:
1. Check for underflow (need 32 bytes)
2. Check for overflow
3. Copy second element to top

**OUTPUT**:
- Copy of second element pushed

**Precision Rules**:
- Returns ERR_STACK_UNDERFLOW if fewer than 2 values
- Returns ERR_STACK_OVERFLOW if stack full

**Example 1**:
- Before: `[1, 2]`
- After: `[1, 2, 1]`

**Example 2**:
- Before: `[10, 20, 30]`
- After: `[10, 20, 30, 20]`

---

## Constant Push Operations

### OP_PUSH_TRUE (100)

**Signature**: `( -- true )`

**INPUT**: None
**OPERATION**: Push boolean true (tag=VTAG_BOOLEAN, value=1)
**OUTPUT**: Boolean true on stack

### OP_PUSH_FALSE (101)

**Signature**: `( -- false )`

**INPUT**: None
**OPERATION**: Push boolean false (tag=VTAG_BOOLEAN, value=0)
**OUTPUT**: Boolean false on stack

### OP_PUSH_NULL (102)

**Signature**: `( -- null )`

**INPUT**: None
**OPERATION**: Push null (tag=VTAG_NULL, value=0)
**OUTPUT**: Null value on stack

### OP_PUSH_ZERO (103)

**Signature**: `( -- 0 )`

**INPUT**: None
**OPERATION**: Push integer 0 (tag=VTAG_INTEGER, value=0)
**OUTPUT**: Integer 0 on stack

### OP_PUSH_ONE (104)

**Signature**: `( -- 1 )`

**INPUT**: None
**OPERATION**: Push integer 1 (tag=VTAG_INTEGER, value=1)
**OUTPUT**: Integer 1 on stack

---

## Error Handling

| Error Code | Name | Condition |
|------------|------|-----------|
| 0 | ERR_NONE | Success |
| 1 | ERR_STACK_UNDERFLOW | Operation requires more values |
| 2 | ERR_STACK_OVERFLOW | Stack capacity exceeded |
| 3 | ERR_INVALID_OPCODE | Unknown opcode |
| 4 | ERR_DIV_BY_ZERO | Division by zero |
| 5 | ERR_OUT_OF_BOUNDS | Array/entity index invalid |

---

## Edge Cases

1. **Empty stack operations**: POP, DUP, SWAP on empty stack return underflow error
2. **Full stack operations**: DUP, OVER, MARK on full stack return overflow error
3. **CLEAR on empty stack**: No-op, succeeds
4. **PICK with index 0**: Equivalent to DUP (copies TOS)
5. **ROT with exactly 3 elements**: Works correctly
6. **SWAP with exactly 2 elements**: Works correctly

---

## Implementation Notes

1. All operations use jump table dispatch for performance
2. Stack grows upward (higher addresses)
3. Each operation ends with `jmp dispatch`
4. Trace recording is not implemented for stack operations
5. ROLL operation is a placeholder (needs full implementation)

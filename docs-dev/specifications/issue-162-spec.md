# Specification: NASM VM Scaffolding (Issue #162)

## Overview

The DTRules NASM VM provides a high-performance bytecode interpreter implemented in x86-64 NASM assembly. The VM executes DTRules bytecode using a threaded interpreter with jump table dispatch.

## Architecture

### Design Principles

1. **Single Entry/Exit**: One call to `vm_execute` runs bytecode to completion - no CGO-style per-operation calls
2. **Jump Table Dispatch**: `jmp [table + opcode*8]` - not if-else chains
3. **Threaded Interpretation**: Each handler ends with `dispatch` macro (jmp back to dispatcher)
4. **Trace Recording**: Writes to buffer, not function calls
5. **System V AMD64 ABI**: Standard Linux x86-64 calling convention

### Register Allocation

| Register | Purpose |
|----------|---------|
| r12 | VM context pointer (preserved across calls) |
| r13 | Program counter (bytecode position) |
| r14 | Data stack pointer |
| r15 | Jump table base address |

### Memory Layout

#### Value Structure (24 bytes)
```
Offset  Size  Field     Description
0       1     tag       Type discriminator (VTAG_*)
1       7     padding   Alignment padding
8       8     num       Numeric value (int64 or float64 bits)
16      8     ptr       Pointer to complex types
```

#### VMContext Structure
The context contains:
- Stack pointers (data_sp, entity_sp, ctrl_sp)
- Stack bases (data_base, entity_base, ctrl_base)
- Stack limits (data_limit, entity_limit, ctrl_limit)
- Bytecode execution state (bytecode, bytecode_len, pc)
- Constant pools (constants, const_count, names, name_count)
- Jump table reference
- State flags (debug, trace, echo, verbose)
- Error code
- Trace buffer management
- Statistics (insn_count, cycle_count)

## Value Types

| Tag | Value | Type |
|-----|-------|------|
| VTAG_NULL | 0 | Null/nil value |
| VTAG_INTEGER | 1 | 64-bit signed integer |
| VTAG_DOUBLE | 2 | 64-bit IEEE 754 float |
| VTAG_BOOLEAN | 3 | Boolean (0 or 1) |
| VTAG_STRING | 4 | String (ptr to data) |
| VTAG_NAME | 5 | Name/symbol reference |
| VTAG_ARRAY | 6 | Array reference |
| VTAG_ENTITY | 7 | Entity reference |
| VTAG_OBJECT | 8 | Object reference |

## Opcode Specification

### Stack Operations (0-19)

| Opcode | Name | INPUT | OPERATION | OUTPUT |
|--------|------|-------|-----------|--------|
| 0 | OP_NOP | - | No operation | - |
| 1 | OP_PUSH | varint index | Push constants[index] to data stack | +1 value |
| 2 | OP_PUSH_INT | varint value | Push integer value | +1 integer |
| 3 | OP_POP | -1 value | Discard top | - |
| 4 | OP_DUP | peek top | Duplicate top | +1 value |
| 5 | OP_SWAP | -2 values (a, b) | Exchange positions | +2 values (b, a) |
| 6 | OP_ROT | -3 values (a, b, c) | Rotate: (a b c -- b c a) | +3 values |
| 7 | OP_ROLL | (not implemented) | Roll n items | - |
| 8 | OP_INDEX | (not implemented) | Copy nth item to top | - |
| 9 | OP_CLEAR | (not implemented) | Clear stack to mark | - |
| 10 | OP_MARK | (not implemented) | Push mark | - |

### Arithmetic Operations (20-29)

| Opcode | Name | INPUT | OPERATION | OUTPUT | Precision |
|--------|------|-------|-----------|--------|-----------|
| 20 | OP_ADD | -2 integers (a, b) | a + b | +1 integer | 64-bit signed, wrapping |
| 21 | OP_SUB | -2 integers (a, b) | a - b | +1 integer | 64-bit signed, wrapping |
| 22 | OP_MUL | -2 integers (a, b) | a * b | +1 integer | 64-bit signed, wrapping |
| 23 | OP_DIV | -2 integers (a, b) | a / b (integer division) | +1 integer | Truncates toward zero |
| 24 | OP_MOD | -2 integers (a, b) | a % b (remainder) | +1 integer | Sign follows dividend |
| 25 | OP_NEG | -1 integer (a) | -a | +1 integer (in-place) | 64-bit signed |
| 26 | OP_ABS | -1 integer (a) | \|a\| | +1 integer (in-place) | 64-bit signed |
| 27 | OP_INC | -1 integer (a) | a + 1 | +1 integer (in-place) | 64-bit signed |
| 28 | OP_DEC | -1 integer (a) | a - 1 | +1 integer (in-place) | 64-bit signed |

### Comparison Operations (30-39)

| Opcode | Name | INPUT | OPERATION | OUTPUT |
|--------|------|-------|-----------|--------|
| 30 | OP_EQ | -2 integers (a, b) | a == b | +1 boolean |
| 31 | OP_NE | -2 integers (a, b) | a != b | +1 boolean |
| 32 | OP_LT | -2 integers (a, b) | a < b | +1 boolean |
| 33 | OP_LE | -2 integers (a, b) | a <= b | +1 boolean |
| 34 | OP_GT | -2 integers (a, b) | a > b | +1 boolean |
| 35 | OP_GE | -2 integers (a, b) | a >= b | +1 boolean |

### Boolean Operations (40-49)

| Opcode | Name | INPUT | OPERATION | OUTPUT |
|--------|------|-------|-----------|--------|
| 40 | OP_AND | -2 booleans (a, b) | a && b | +1 boolean |
| 41 | OP_OR | -2 booleans (a, b) | a \|\| b | +1 boolean |
| 42 | OP_NOT | -1 boolean (a) | !a | +1 boolean (in-place) |
| 43 | OP_XOR | -2 booleans (a, b) | a ^ b | +1 boolean |

### Control Flow Operations (50-59)

| Opcode | Name | Status |
|--------|------|--------|
| 50 | OP_EXEC | Not implemented |
| 51 | OP_IF | Not implemented |
| 52 | OP_IF_ELSE | Not implemented |
| 53 | OP_WHILE | Not implemented |
| 54 | OP_FOR | Not implemented |
| 55 | OP_FOR_ALL | Not implemented |
| 56 | OP_RETURN | Not implemented |
| 57 | OP_JUMP | Implemented: add varint offset to PC |
| 58 | OP_JUMP_IF | Implemented: conditional jump if top is true |
| 59 | OP_CALL | Not implemented |

### Entity Operations (60-69)

All currently return ERR_NOT_IMPLEMENTED.

### Array Operations (70-79)

All currently return ERR_NOT_IMPLEMENTED.

### Table Operations (80-89)

All currently return ERR_NOT_IMPLEMENTED.

### String Operations (90-99)

All currently return ERR_NOT_IMPLEMENTED.

### Constant Push Operations (100-109)

| Opcode | Name | INPUT | OPERATION | OUTPUT |
|--------|------|-------|-----------|--------|
| 100 | OP_PUSH_TRUE | - | Push true | +1 boolean (1) |
| 101 | OP_PUSH_FALSE | - | Push false | +1 boolean (0) |
| 102 | OP_PUSH_NULL | - | Push null | +1 null value |
| 103 | OP_PUSH_ZERO | - | Push integer 0 | +1 integer (0) |
| 104 | OP_PUSH_ONE | - | Push integer 1 | +1 integer (1) |

### Extended Operations (200+)

| Opcode | Name | INPUT | OPERATION | OUTPUT |
|--------|------|-------|-----------|--------|
| 200 | OP_OPERATOR | varint index | Call operator handler | varies |
| 201 | OP_CONSTANT | varint index | Push constant by index | +1 value |
| 202 | OP_NAME | varint index | Push name by index | +1 name |
| 255 | OP_HALT | - | Stop execution | - |

## Error Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | ERR_NONE | No error |
| 1 | ERR_STACK_OVERFLOW | Data stack overflow |
| 2 | ERR_STACK_UNDERFLOW | Data stack underflow |
| 3 | ERR_INVALID_OPCODE | Unknown opcode |
| 4 | ERR_TYPE_MISMATCH | Wrong type for operation |
| 5 | ERR_DIV_ZERO | Division by zero |
| 6 | ERR_OUT_OF_BOUNDS | Index out of bounds |
| 7 | ERR_NULL_PTR | Null pointer dereference |
| 8 | ERR_NOT_IMPLEMENTED | Feature not yet implemented |

## Varint Encoding

Variable-length integer encoding:
- Each byte: 7 bits of data + 1 continuation bit (high bit)
- Little-endian byte order
- Maximum 63-bit values supported

Example:
- Value 42: `0x2A` (single byte, no continuation)
- Value 128: `0x80 0x01` (two bytes)

## C API

```c
int vm_execute(VMContext *ctx);
void vm_load_bytecode(VMContext *ctx, void *bytecode, size_t len);
void vm_set_jump_table(VMContext *ctx, void **jump_table);
size_t vm_get_stack_depth(VMContext *ctx);
int vm_peek_stack(VMContext *ctx, size_t index, Value *out);
uint64_t vm_get_insn_count(VMContext *ctx);
int vm_get_error(VMContext *ctx);
void vm_enable_trace(VMContext *ctx);
void vm_disable_trace(VMContext *ctx);
```

## Worked Examples

### Example 1: Calculate 10 + 32

**Bytecode:**
```
OP_PUSH_INT, 10,   ; Push 10
OP_PUSH_INT, 32,   ; Push 32
OP_ADD,            ; Add them
OP_HALT            ; Stop
```

**Execution Trace:**
1. PC=0: OP_PUSH_INT - read varint 10, push integer(10). Stack: [10]
2. PC=2: OP_PUSH_INT - read varint 32, push integer(32). Stack: [10, 32]
3. PC=4: OP_ADD - pop 32, pop 10, push 42. Stack: [42]
4. PC=5: OP_HALT - exit with success

**Result:** Stack contains integer 42

### Example 2: Compare and Branch

**Bytecode:**
```
OP_PUSH_INT, 10,   ; Push 10
OP_PUSH_INT, 20,   ; Push 20
OP_LT,             ; 10 < 20 = true
OP_JUMP_IF, 2,     ; If true, skip next 2 bytes
OP_PUSH_FALSE,     ; (skipped)
OP_HALT,           ; (skipped)
OP_PUSH_TRUE,      ; Push true
OP_HALT            ; Stop
```

**Execution:**
1. Push 10, Push 20, LT -> Stack: [true]
2. JUMP_IF with offset 2 - condition is true, so PC += 2
3. Execute PUSH_TRUE -> Stack: [true, true]
4. HALT

**Result:** Stack contains [true, true]

## Edge Cases

### Division by Zero
- OP_DIV and OP_MOD with divisor 0 return ERR_DIV_ZERO
- Stack state is undefined after error

### Stack Overflow
- All push operations check against data_limit
- Returns ERR_STACK_OVERFLOW if limit exceeded

### Stack Underflow
- All pop operations check against data_base
- Returns ERR_STACK_UNDERFLOW if stack empty

### Type Checking
- Arithmetic operations require VTAG_INTEGER operands
- Boolean operations require VTAG_BOOLEAN operands
- Type mismatch returns ERR_TYPE_MISMATCH

### Integer Overflow
- All arithmetic uses 64-bit signed integers
- Overflow wraps (two's complement behavior)
- No overflow detection is performed

## Known Issues (To Be Fixed)

1. **vm_exit_success does not save data_sp**: The exit path jumps past the code that saves r14 back to VMContext.data_sp, causing stack depth queries to always return 0.

2. **data_pop macro has bug**: Line 131 reads `mov rbx, [rax + Value.num]` but rax was already modified by line 130 which stored the tag. The `data_pop_safe` macro exists as a corrected version.

## File Structure

```
nasm-vm/
├── vm_constants.inc   # Constants: opcodes, error codes, type tags
├── vm_state.inc       # Structure definitions: Value, VMContext, TraceEntry
├── vm_core.asm        # Macros: data_push, data_pop, read_byte, dispatch
├── vm_entry.asm       # Entry point: vm_execute, context initialization
├── vm_jump_table.asm  # Jump table and opcode handlers
├── test_harness.c     # C test harness
└── Makefile           # Build system
```

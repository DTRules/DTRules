# DTRules NASM VM

A high-performance bytecode virtual machine for DTRules, written in x86-64 NASM assembly.

## Overview

This VM implements the DTRules bytecode interpreter in native assembly code using:

- **Jump table dispatch**: Direct opcode-to-handler mapping via `jmp [table + opcode*8]`
- **Single entry/exit**: Clean function boundary with proper callee-saved register handling
- **Tagged values**: 16-byte values with 8-byte tag and 8-byte payload
- **Stack-based execution**: Matches the Go/Java DTRules interpreter semantics

## Architecture

### Register Allocation

| Register | Purpose |
|----------|---------|
| rdi | Bytecode instruction pointer |
| rsi | Bytecode end pointer |
| r12 | Data stack top pointer |
| r13 | Data stack base (underflow limit) |
| r14 | Data stack end (overflow limit) |
| r15 | Trace buffer pointer (optional) |
| rbx | VM context pointer |

### Value Representation

Values are 16 bytes (128 bits):
- **tag** (offset 0, 8 bytes): Type discriminator
- **value** (offset 8, 8 bytes): Integer, double, or pointer

Type tags:
- 0: null
- 1: boolean
- 2: integer
- 3: double
- 4: string
- 5: name
- 6: array
- 7: entity
- 8: mark

## Implemented Operations

### Stack Operations
- `NOP` - No operation
- `POP`/`DROP` - Remove top element
- `DUP` - Duplicate top element
- `SWAP` - Swap top two elements
- `OVER` - Copy second element to top
- `ROT` - Rotate top three elements
- `PICK`/`INDEX` - Copy nth element to top
- `CLEAR` - Clear entire stack
- `MARK` - Push stack mark

### Push Operations
- `PUSH_TRUE` - Push boolean true
- `PUSH_FALSE` - Push boolean false
- `PUSH_NULL` - Push null
- `PUSH_ZERO` - Push integer 0
- `PUSH_ONE` - Push integer 1
- `PUSH_INT` - Push immediate integer (varint encoded)

### Arithmetic
- `ADD`, `SUB`, `MUL`, `DIV`, `MOD`
- `NEG` - Negate
- `ABS` - Absolute value
- `INC`, `DEC` - Increment/decrement

### Comparison
- `EQ`, `NE` - Equal, not equal
- `LT`, `LE`, `GT`, `GE` - Less/greater than (or equal)

### Boolean
- `AND`, `OR`, `NOT`, `XOR`

## Building

Requirements:
- NASM assembler
- GCC (for C test harness)
- Linux x86-64

```bash
make        # Build VM and tests
make test   # Run tests
make clean  # Clean build artifacts
make debug  # Build with debug symbols
```

## Testing

```bash
make test
```

All 45 tests verify:
- Stack operations with underflow/overflow detection
- Arithmetic operations including division by zero
- Boolean and comparison operations
- Combined operation sequences

## Usage

The VM is called from C code:

```c
#include <stdint.h>

typedef struct {
    int64_t tag;
    int64_t value;
} Value;

typedef struct {
    uint8_t*  bytecode;
    size_t    bytecode_len;
    Value*    constants;
    size_t    constants_len;
    void**    names;
    size_t    names_len;
    Value*    data_stack;
    size_t    stack_size;
    size_t    stack_capacity;
    uint8_t*  trace_buf;
    size_t    trace_size;
} VMContext;

extern int vm_execute(VMContext* ctx);
```

## Error Handling

The VM returns error codes:
- 0: Success
- 1: Stack underflow
- 2: Stack overflow
- 3: Invalid opcode
- 4: Division by zero
- 5: Out of bounds

## License

Apache License 2.0 - See LICENSE file.

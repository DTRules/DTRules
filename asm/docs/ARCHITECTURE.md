# DTRules Assembly Architecture

> **Note:** A consolidated guide is available at [docs/asm-guide.md](../../docs/asm-guide.md)

## Overview

Pure x86-64 assembly implementation of the DTRules execution engine. This implementation makes no use of libc - all system interactions are through direct Linux syscalls. The goal is educational: demonstrating how a complete rules engine can be built at the lowest level.

## Design Philosophy

1. **No External Dependencies**: Direct syscalls only, no libc
2. **Value Compatibility**: 24-byte tagged union matches Go implementation exactly
3. **Register-Based VM State**: Critical pointers in callee-saved registers
4. **Arena Allocation**: Simple bump-pointer allocation for predictable memory behavior

## Memory Layout

### Stack Regions (grow downward)

```
┌─────────────────────────────────────┐  High addresses
│                                     │
│         Control Stack               │  ← r14 (grows down)
│         (call frames)               │
│                                     │
├─────────────────────────────────────┤
│                                     │
│         Entity Stack                │  ← r13 (grows down)
│         (context entities)          │
│                                     │
├─────────────────────────────────────┤
│                                     │
│         Data Stack                  │  ← r12 (grows down)
│         (operand values)            │
│                                     │
├─────────────────────────────────────┤
│                                     │
│         Heap (arena)                │  (grows up)
│         (allocated objects)         │
│                                     │
└─────────────────────────────────────┘  Low addresses
```

### Register Convention

| Register | Purpose | Preserved |
|----------|---------|-----------|
| r12 | Data stack pointer | Yes |
| r13 | Entity stack pointer | Yes |
| r14 | Control stack pointer | Yes |
| r15 | State structure pointer | Yes |
| rdi | First argument | No |
| rsi | Second argument | No |
| rdx | Third argument | No |
| rax | Return value | No |
| rbx | General (callee-saved) | Yes |
| rbp | Frame pointer | Yes |

## Value Representation

Each value is a 24-byte tagged union, matching the Go implementation exactly:

```
┌─────────────────────────────────────────────────────────────────┐
│ Byte 0    │ Bytes 1-7   │ Bytes 8-15        │ Bytes 16-23       │
│───────────│─────────────│───────────────────│───────────────────│
│ Tag (1B)  │ Padding(7B) │ Number (int64/f64)│ Pointer (8B)      │
└─────────────────────────────────────────────────────────────────┘
```

### Type Tags

| Tag | Type | Number Field | Pointer Field |
|-----|------|--------------|---------------|
| 0 | NULL | 0 | 0 |
| 1 | INTEGER | 64-bit signed int | 0 |
| 2 | DOUBLE | IEEE 754 double bits | 0 |
| 3 | BOOLEAN | 0 or 1 | 0 |
| 4 | STRING | 0 | → String header |
| 5 | NAME | 0 | → Name entry |
| 6 | ARRAY | 0 | → Array header |
| 7 | ENTITY | 0 | → Entity header |

### String Header

```
┌─────────────────────────────────────────┐
│ Length (8 bytes) │ Data (variable)...   │
└─────────────────────────────────────────┘
```

### Array Header

```
┌─────────────────────────────────────────────────────────────────┐
│ Length (8B) │ Capacity (8B) │ Data Pointer (8B) → Value[]      │
└─────────────────────────────────────────────────────────────────┘
```

## Opcode Encoding

Opcodes are single bytes, optionally followed by operands:

| Range | Category | Examples |
|-------|----------|----------|
| 0x00-0x0F | Stack ops | nop, push, pop, dup, swap |
| 0x10-0x1F | Arithmetic | add, sub, mul, div, mod |
| 0x20-0x2F | Comparison | eq, ne, lt, le, gt, ge, and, or |
| 0x30-0x4F | String ops | concat, substring, split |
| 0x50-0x5F | Array ops | newarray, get, put, length |
| 0x60-0x6F | Control flow | if, while, for, jump |
| 0x70-0x7F | Entity ops | newentity, getattr, setattr |
| 0x80-0x8F | Decision table | executetable, cnode, anode |
| 0xF0-0xFF | Debug | print, trace, halt |

### Variable-Length Operands

Integers use variable-length encoding (similar to Protocol Buffers):
- Each byte has 7 data bits + 1 continuation bit
- Continuation bit = 1 means more bytes follow
- Signed integers use zigzag encoding

## Memory Management

### Arena Allocator

The heap uses a simple bump-pointer allocator:

```
heap_alloc:
    ; rdi = size to allocate
    mov rax, [state + State.heap_ptr]
    add rdi, 7
    and rdi, ~7                    ; Align to 8 bytes
    add rdi, rax
    cmp rdi, [state + State.heap_end]
    ja .out_of_memory
    mov [state + State.heap_ptr], rdi
    ret                            ; rax = allocated address
```

### Object Pools

Fixed-size pools for Values and Names use free lists:
- 24-byte Value pool
- 32-byte Name pool
- Free blocks are linked through their first 8 bytes

## Error Handling

Errors are recorded in the state structure and checked after each operation:

```asm
; Check for error
cmp dword [state + State.error], ERR_NONE
jne .handle_error
```

Error codes are defined in `constants.inc`:
- ERR_NONE (0)
- ERR_STACK_OVERFLOW (1)
- ERR_STACK_UNDERFLOW (2)
- ERR_TYPE_MISMATCH (3)
- ERR_DIV_BY_ZERO (4)
- ERR_OUT_OF_MEMORY (5)
- ...

## Execution Loop

The main execution loop in `bytecode.asm`:

```
vm_execute:
    .fetch:
        ; Check if halted
        cmp dword [state + State.halted], 0
        jne .done

        ; Fetch opcode
        movzx eax, byte [rbx]
        inc rbx

        ; Dispatch via jump table
        lea rcx, [opcode_table]
        jmp [rcx + rax * 8]

        ; Check for errors
        cmp dword [state + State.error], ERR_NONE
        jne .error

        jmp .fetch
```

## File Organization

```
asm/
├── include/
│   ├── constants.inc    # Type tags, opcodes, limits
│   ├── macros.inc       # Common assembly macros
│   ├── syscalls.inc     # Linux syscall numbers
│   └── state.inc        # VM state structure
├── src/
│   ├── boot/
│   │   └── start.asm    # Entry point, init
│   ├── core/
│   │   ├── value.asm    # Value operations
│   │   └── stack_*.asm  # Stack implementations
│   ├── memory/
│   │   ├── heap.asm     # Arena allocator
│   │   └── pool.asm     # Fixed-size pools
│   ├── types/           # Type implementations
│   ├── vm/
│   │   └── bytecode.asm # Execution loop + operators
│   ├── dt/              # Decision table execution
│   ├── xml/             # XML parsing
│   └── io/              # File I/O, printing
└── test/
    ├── unit/            # Assembly unit tests
    └── comparison/      # Go vs ASM comparison
```

## Building

Requirements:
- NASM 2.15+ (assembler)
- GNU ld (linker)
- Linux x86-64

```bash
make           # Build dtrules-asm
make debug     # Build with debug symbols
make test      # Run all tests
make clean     # Remove build artifacts
```

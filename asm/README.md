# DTRules x86-64 Assembly Implementation

An educational x86-64 assembly implementation of the DTRules execution engine - a pure assembly, self-contained rules engine with no external dependencies (no libc).

## Overview

This project demonstrates how a stack-based rules engine works at the machine level. It replicates the architecture of the Go/Java DTRules implementation in pure assembly:

- **Stack-based VM** with three stacks (data, entity, control)
- **24-byte tagged union Values** matching Go's `Value` type
- **183+ operators** for math, strings, arrays, control flow, entities
- **Decision trees** (CNode/ANode) for efficient table execution
- **Pure assembly XML parser** for loading rules

## Requirements

- Linux x86-64 (tested on Ubuntu/Debian)
- NASM (Netwide Assembler) 2.15+
- GNU Make
- ld (GNU linker)

### Install Dependencies

```bash
# Ubuntu/Debian
sudo apt-get install nasm make binutils

# Fedora/RHEL
sudo dnf install nasm make binutils

# Arch Linux
sudo pacman -S nasm make binutils
```

## Building

```bash
# Build the main executable
make

# Build with debug symbols
make debug

# Clean build artifacts
make clean
```

## Usage

```bash
# Run a rule set
./dtrules-asm path/to/rules.xml

# Show usage
./dtrules-asm
```

## Project Structure

```
dtrules-asm/
├── Makefile
├── include/
│   ├── constants.inc       # Opcodes, tags, limits
│   ├── macros.inc          # Common macros
│   └── syscalls.inc        # Linux syscall numbers
├── src/
│   ├── boot/start.asm      # Entry point, memory setup
│   ├── core/
│   │   ├── value.asm       # 24-byte tagged union
│   │   ├── stack_data.asm  # Data stack operations
│   │   ├── stack_entity.asm # Entity stack
│   │   └── stack_control.asm # Control stack + frames
│   ├── memory/
│   │   ├── heap.asm        # Arena allocator
│   │   └── pool.asm        # Fixed-size pools
│   ├── types/
│   │   ├── integer.asm     # 64-bit integers
│   │   ├── double.asm      # IEEE 754 doubles (SSE2)
│   │   ├── string.asm      # Length-prefixed strings
│   │   ├── array.asm       # Dynamic arrays
│   │   └── entity.asm      # Typed dictionaries
│   ├── vm/
│   │   ├── bytecode.asm    # Execution loop
│   │   └── operators/      # Operator implementations
│   ├── dt/
│   │   ├── cnode.asm       # Condition node
│   │   ├── anode.asm       # Action node
│   │   └── table.asm       # Decision table execution
│   ├── xml/
│   │   ├── lexer.asm       # XML tokenizer
│   │   ├── parser.asm      # DOM builder
│   │   ├── edd_loader.asm  # Entity definitions
│   │   └── dt_loader.asm   # Decision tables
│   └── io/
│       ├── file.asm        # File I/O (syscalls)
│       └── print.asm       # Output formatting
├── test/
│   ├── unit/               # Module tests
│   ├── integration/        # Full system tests
│   └── comparison/         # Go vs ASM comparison
└── examples/
    └── simple_rule.xml     # Example rule file
```

## Architecture

### Value Type (24 bytes)

```asm
struc Value
    .tag:   resb 1    ; Type tag (null, int, double, bool, string, name, array, entity)
    .pad:   resb 7    ; Alignment padding
    .num:   resq 1    ; Integer or float64 bits
    .ptr:   resq 1    ; Pointer for complex types
endstruc
```

### Register Conventions

| Register | Purpose |
|----------|---------|
| r12 | Data stack pointer (preserved) |
| r13 | Entity stack pointer (preserved) |
| r14 | Control stack pointer (preserved) |
| r15 | State pointer (preserved) |
| rdi/rsi | Arguments |
| rax | Return value |

### Memory Management

- **Arena allocator**: Simple bump pointer, reset between executions
- **Fixed pools**: 24-byte (Values), 32-byte (Names) pools with free lists
- **No libc**: Direct Linux syscalls (mmap, read, write, exit)

## Testing

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Compare with Go implementation
make test-comparison
```

## Opcodes

The VM supports 183+ opcodes organized into categories:

- **0x00-0x0F**: Stack operations (dup, swap, rot, pop, etc.)
- **0x10-0x1F**: Arithmetic (add, sub, mul, div, mod, etc.)
- **0x20-0x2F**: Comparison/Logic (eq, lt, and, or, not, etc.)
- **0x30-0x4F**: String operations (concat, substring, etc.)
- **0x50-0x5F**: Array operations (get, set, push, pop, etc.)
- **0x60-0x6F**: Control flow (if, while, for, break, etc.)
- **0x70-0x7F**: Entity operations (getattr, setattr, def, etc.)
- **0x80-0x8F**: Decision table operations
- **0xF0-0xFF**: Debug/utility (print, trace, halt)

## Educational Value

This implementation teaches:

1. **Stack machine architecture** - How interpreters work at CPU level
2. **Memory management** - Arena allocation without malloc
3. **Type systems** - Tagged unions, type dispatch
4. **x86-64 ABI** - Calling conventions, register usage
5. **System programming** - Direct syscalls without libc
6. **Parsing** - Hand-written XML lexer/parser
7. **Data structures** - Hash tables, arrays in assembly

## License

MIT License - See LICENSE file for details.

## Related Projects

- [DTRules](https://github.com/DTRules) - Original Java implementation
- [DTRules Go](https://github.com/DTRules/dtrules-go) - Go implementation

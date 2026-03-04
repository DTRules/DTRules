# ASM Runtime

The ASM runtime is a high-performance native x86-64 assembly implementation of the DTRules stack machine.

## Overview

The ASM runtime provides maximum execution speed by implementing the DTRules interpreter directly in assembly language. It operates as a complete, standalone runtime with no shared state with other implementations.

## Architecture

### Design Principles

1. **Complete Implementation** - All core DTRules operations implemented natively
2. **No Shared State** - Fully independent from Go runtime
3. **Service Linkage Only** - Links to Go only for external services (I/O, crypto, DB)
4. **No Callbacks** - Core execution never calls back to Go

### Stack Model

The ASM runtime implements the same three-stack architecture as other DTRules runtimes:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ASM Runtime                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Data Stack              Control Stack         Entity Stack          │
│  ┌─────────────────┐    ┌─────────────────┐   ┌─────────────────┐  │
│  │ RSP-based       │    │ Frame pointers  │   │ Context chain   │  │
│  │ Native push/pop │    │ Return addrs    │   │ Attribute lookup│  │
│  └─────────────────┘    └─────────────────┘   └─────────────────┘  │
│                                                                      │
│  Registers:                                                          │
│  - RAX: Accumulator / Return value                                  │
│  - RBX: Data stack pointer                                          │
│  - RCX: Control stack pointer                                       │
│  - RDX: Scratch / Arguments                                         │
│  - RSI/RDI: Source/Destination for string ops                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
asm/
├── core/                    # Core stack machine
│   ├── stack.asm           # Stack operations
│   ├── dispatch.asm        # Operator dispatch
│   └── main.asm            # Entry points
│
├── operators/              # Native operators
│   ├── math.asm            # Arithmetic (+, -, *, /)
│   ├── boolean.asm         # Boolean (and, or, not)
│   ├── compare.asm         # Comparison (<, >, ==)
│   ├── stack_ops.asm       # Stack manipulation (dup, swap, pop)
│   └── control.asm         # Control flow (if, while, for)
│
├── services/               # Go service linkage
│   ├── io.go               # File I/O service
│   ├── crypto.go           # Cryptography service
│   └── db.go               # Database service
│
├── build/                  # Build outputs
│   ├── dtrules.o           # Object file
│   └── libdtrules.a        # Static library
│
└── Makefile                # Build configuration
```

## Service Linkage

The ASM runtime links to Go for services that require external system access.

### Linkage Model

```
┌────────────────────────────────────────────────────────────────────┐
│                        ASM Execution                                │
│                                                                     │
│   Pure assembly operations:                                         │
│   - Stack push/pop                                                  │
│   - Arithmetic operations                                           │
│   - Boolean logic                                                   │
│   - Control flow                                                    │
│   - Operator dispatch                                               │
│                                                                     │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
                                 │ External service calls
                                 ▼
┌────────────────────────────────────────────────────────────────────┐
│                      Go Service Layer                               │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  I/O Services           Crypto Services        Database Services    │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐   │
│  │ ReadFile()     │    │ SHA256()       │    │ Connect()      │   │
│  │ WriteFile()    │    │ AES256()       │    │ Query()        │   │
│  │ ReadConsole()  │    │ RSASign()      │    │ Execute()      │   │
│  │ WriteConsole() │    │ RSAVerify()    │    │ Close()        │   │
│  └────────────────┘    └────────────────┘    └────────────────┘   │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Service Call Convention

When ASM needs to call a Go service:

1. Arguments are placed in registers (AMD64 calling convention)
2. Service function is called via function pointer
3. Result is returned in RAX
4. No state is shared - all data is passed explicitly

### Why This Model

- **Performance**: Core execution stays in assembly
- **Simplicity**: No complex state synchronization
- **Portability**: Services can be swapped out
- **Testability**: Each layer can be tested independently

## Building

```bash
# Assemble the ASM runtime
cd asm
make

# Run tests
make test

# Build with Go services linked
make libdtrules.a
```

## Performance

The ASM runtime achieves maximum performance by:

1. **Direct register operations** - No interpreter dispatch overhead
2. **Inline operators** - Common operations expanded inline
3. **Branch prediction** - Optimized jump tables for dispatch
4. **Memory locality** - Stack data kept in cache

### Expected Performance Characteristics

| Operation | Go Runtime | ASM Runtime | Speedup |
|-----------|------------|-------------|---------|
| Stack push/pop | 0.5 ns | 0.1 ns | 5x |
| Integer add | 0.8 ns | 0.2 ns | 4x |
| Operator dispatch | 0.6 ns | 0.1 ns | 6x |

Note: Actual performance depends on workload and CPU architecture.

## Comparison with Go Runtime

| Aspect | Go Runtime | ASM Runtime |
|--------|------------|-------------|
| Portability | Any Go platform | x86-64 only |
| Build complexity | Standard Go build | Assembler required |
| Debugging | Go tooling | Lower-level tools |
| Performance | Optimized | Maximum |
| Maintenance | Easier | Requires ASM expertise |

## When to Use

**Use ASM Runtime when:**
- Maximum performance is critical
- Running on x86-64 hardware
- Workload is compute-bound

**Use Go Runtime when:**
- Portability is required
- Easier debugging/maintenance preferred
- Performance is sufficient

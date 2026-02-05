# Native ASM Runtime (Plan 9 Assembly)

## Overview

The native ASM runtime is a high-performance implementation of the DTRules bytecode executor using Go's native Plan 9 assembly. It provides significant performance improvements over both pure Go and CGO-based implementations by eliminating runtime overhead while maintaining full compatibility.

## Performance

| Operation | Native Plan 9 | Pure Go | CGO NASM | Plan 9 Speedup |
|-----------|--------------|---------|----------|----------------|
| Push/Pop | 4.1 ns | 20.9 ns | 81.7 ns | **5x vs Go, 20x vs CGO** |
| Execute 1 op | 3.2 ns | 24.8 ns | 174.1 ns | **8x vs Go, 54x vs CGO** |
| Execute 100 ops | 206 ns | 1730 ns | 767 ns | **8x vs Go, 4x vs CGO** |
| Execute 1000 ops | 1987 ns | 17239 ns | 6144 ns | **9x vs Go, 3x vs CGO** |

**Key benefits:**
- Zero CGO overhead (no C call boundary crossing)
- Zero allocations in hot paths
- Native register-based execution
- Full compatibility with runtime.Runtime interface

## Architecture

### Package Structure

```
go/pkg/dtrules/runtime/nativeasm/
├── vm.go              # Go interface, VMState struct, hybrid executor
├── vm_amd64.s         # Plan 9 assembly bytecode interpreter
├── stack_amd64.s      # Reserved for future stack optimizations
├── runtime.go         # NativeRuntime implementing runtime.Runtime
├── runtime_test.go    # Runtime interface tests
└── vm_test.go         # Comprehensive VM tests
```

### VMState Structure

```go
type VMState struct {
    // Data stack (24 bytes * 1024 = 24,576 bytes)
    Stack [MaxStackDepth]dtrules.Value
    SP    int64

    // Error state
    Error int64

    // Bytecode execution state
    Code     uintptr
    CodeEnd  uintptr

    // Constant and name pools
    Constants  uintptr
    ConstCount int64
    Names      uintptr
    NameCount  int64

    // Entity stack (for name resolution context)
    EntityStack [MaxEntityDepth]dtrules.Entity
    EntitySP    int64

    // Control stack (for loops/calls)
    ControlStack [MaxControlDepth]ControlFrame
    ControlSP    int64
}
```

### Limits

| Resource | Limit |
|----------|-------|
| Data stack depth | 1,024 values |
| Entity stack depth | 256 entities |
| Control stack depth | 256 frames |

## Runtime Interface

The native ASM runtime implements the standard `runtime.Runtime` and `runtime.ExecutionContext` interfaces:

```go
// Create runtime
rt := nativeasm.New()
defer rt.Close()

// Create execution context
ctx, _ := rt.CreateContext()
defer ctx.Close()

// Push values
ctx.Push(dtrules.NewValueInteger(10))
ctx.Push(dtrules.NewValueInteger(32))

// Execute bytecode
bc := dtrules.NewBytecodeChunk()
bc.Emit(dtrules.OpAdd)
ctx.ExecuteBytecode(bc)

// Pop result
result, _ := ctx.Pop()
fmt.Println(result.AsInteger()) // 42
```

### Capabilities

```go
caps := rt.Capabilities()
// caps.ConcurrentContexts = true   (each context has own VMState)
// caps.Tracing = false             (can add later)
// caps.MaxStackDepth = 1024
// caps.MaxEntityDepth = 256
// caps.SupportsAllOperators = false (core ops in assembly, complex in Go)
```

## Implemented Operations

### Assembly-Optimized (Fast Path)

These operations execute entirely in Plan 9 assembly:

| Category | Operations |
|----------|------------|
| Stack | push, pop, dup, swap, rot, over, pick, roll, clear |
| Arithmetic | add, sub, mul, div, mod, neg, abs, inc, dec |
| Comparison | eq, ne, lt, le, gt, ge |
| Boolean | and, or, not, xor |
| Constants | push_true, push_false, push_null, push_zero, push_one |
| Control | jump, jump_if, return |

### Hybrid Execution (Go Fallback)

Complex operations that require Go runtime support:

| Category | Operations | Reason |
|----------|------------|--------|
| Entity | entity_push, entity_pop, def, lookup | Go interface handling |
| Arrays | new_array, add_to, length, get, put | Dynamic allocation |
| Constants | constant, name | Pool access |
| Strings | concat, substring | GC-managed strings |

## Hybrid Executor

The `ExecuteWithEntities` method provides efficient execution by:
1. Running assembly-optimized operations directly
2. Falling back to Go for complex operations
3. Maintaining consistent stack state across boundaries

```go
// Hybrid execution flow
func (vm *VMState) ExecuteWithEntities(code []byte, constants []Value, names []*RName) int {
    for pc < len(code) {
        op := code[pc]
        switch op {
        case OpAdd, OpSub, OpMul, ...:
            // Execute in assembly - fast path
            vm.ExecuteSimple(code[pc:], ...)
        case OpEntityPush, OpNewArray, ...:
            // Handle in Go - complex operations
            handleComplexOp(vm, op, ...)
        }
    }
}
```

## Value Type

Values use a 24-byte tagged union compatible with the Go implementation:

```
+--------+--------+--------+
| tag    | padding| num    |
| 1 byte | 7 bytes| 8 bytes|
+--------+--------+--------+
| ptr                      |
| 8 bytes                  |
+--------------------------+
```

| Tag | Type | num usage | ptr usage |
|-----|------|-----------|-----------|
| 0 | Null | unused | unused |
| 1 | Integer | int64 value | unused |
| 2 | Double | float64 bits | unused |
| 3 | Boolean | 0=false, 1=true | unused |
| 4 | String | unused | *string |
| 5 | Name | unused | *RName |
| 6 | Array | unused | *RArray |
| 7 | Entity | unused | Entity |

## Error Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | None | No error |
| 1 | StackOverflow | Data stack overflow |
| 2 | StackUnderflow | Data stack underflow |
| 3 | TypeMismatch | Type error in operation |
| 4 | DivisionByZero | Division by zero |
| 5 | InvalidOpcode | Unknown opcode |
| 6 | EntityStackOverflow | Entity stack overflow |
| 7 | EntityStackUnderflow | Entity stack underflow |
| 8 | NameNotFound | Name lookup failed |

## Plan 9 Assembly Notes

### Register Usage

| Register | Purpose |
|----------|---------|
| AX | Return value, scratch |
| BX | Bytecode pointer |
| CX | Scratch, shift amounts |
| DX | Scratch |
| SI | Source pointer |
| DI | Destination pointer |
| R8-R11 | Scratch |
| R12-R15 | Preserved across calls |

### Key Patterns

**Multiply by 24 (Value size):**
```asm
#define MUL24(src, dst) \
    MOVQ src, dst; \
    SHLQ $4, dst; \       // dst = src * 16
    MOVQ src, R14; \
    SHLQ $3, R14; \       // R14 = src * 8
    ADDQ R14, dst         // dst = src * 24
```

**Varint reading:**
```asm
read_varint:
    XORQ R10, R10         // value accumulator
    XORQ CX, CX           // shift amount
loop:
    MOVBQZX (SI), R12     // read byte
    INCQ SI
    MOVQ R12, DI
    ANDQ $0x7F, DI        // extract 7 bits
    SHLQ CL, DI           // shift by accumulated amount
    ORQ DI, R10           // add to value
    ADDQ $7, CX           // next shift
    TESTQ $0x80, R12      // more bytes?
    JNZ loop
```

## Testing

```bash
# Run all nativeasm tests
go test -v ./pkg/dtrules/runtime/nativeasm/...

# Run benchmarks
go test -bench=. -benchmem ./pkg/dtrules/runtime/nativeasm/...

# Run specific test
go test -v -run TestContextPushPop ./pkg/dtrules/runtime/nativeasm/...
```

## Usage with Decision Tables

The native ASM runtime integrates with the session layer:

```go
import (
    "github.com/PaulSnow/DTRules/go/pkg/dtrules/runtime/nativeasm"
    "github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// Load rules
rules, _ := repo.LoadFromXML(eddFile, dtFile)

// Create session with native ASM runtime
rt := nativeasm.New()
sess, _ := session.NewSession(rules, rt)

// Execute decision table
sess.Execute("Compute_Eligibility")
```

## Future Enhancements

1. **Full operator coverage**: Implement remaining 109 operators in assembly
2. **Tracing support**: Add optional execution tracing
3. **SIMD optimization**: Use AVX for bulk array operations
4. **ARM64 port**: Add arm64 assembly for Apple Silicon/ARM servers

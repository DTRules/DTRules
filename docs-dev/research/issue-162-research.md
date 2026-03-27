# Research: NASM VM Scaffolding (Issue #162)

## Objective

Build a high-performance bytecode interpreter for DTRules using NASM (Netwide Assembler) targeting x86-64 Linux. The VM should execute the same bytecode as the Go interpreter but with lower overhead.

## Background Research

### Why NASM?

1. **Direct Hardware Access**: No abstraction layers, direct register/memory control
2. **Optimal Dispatch**: Jump table dispatch is the fastest known interpreter technique
3. **Memory Layout Control**: Exact control over data structure layouts
4. **FFI Integration**: Easy integration with C and potentially Go via cgo

### Interpreter Design Patterns

#### Switch Dispatch
```c
while (running) {
    switch (*pc++) {
        case OP_ADD: /* ... */ break;
        case OP_SUB: /* ... */ break;
    }
}
```
- Simple but slow due to branch misprediction
- Compiler may not optimize well

#### Jump Table Dispatch (Computed Goto)
```asm
dispatch:
    movzx eax, byte [pc]
    inc pc
    jmp [jump_table + rax*8]
```
- Used by LuaJIT, CPython (computed goto variant)
- Best performance for bytecode interpreters
- **This is the approach chosen for DTRules NASM VM**

#### Threaded Interpretation
Each handler ends with the dispatch sequence:
```asm
op_add:
    ; ... add logic ...
    dispatch    ; macro that jumps to next opcode
```
- Avoids function call overhead
- Better branch prediction than switch
- No call/ret instruction overhead

### Stack Machine vs Register Machine

DTRules uses a **stack machine** architecture:
- Operands are pushed/popped from a data stack
- No need to encode register operands in bytecode
- Simpler bytecode format
- Natural fit for expression evaluation

### Reference Implementations

#### Fifth/AISynth Pattern
The NASM VM follows the Fifth/AISynth architecture:
- Single entry/exit to minimize FFI overhead
- No per-operation callbacks
- Trace recording to buffer (not callbacks)
- Jump table for opcode dispatch

#### LuaJIT Interpreter
- Uses computed goto in C (similar to our jump table)
- Register-based VM
- Inline assembly for performance-critical paths

#### CPython
- Stack-based like DTRules
- Uses computed goto with gcc extension
- Heavy use of macros for common operations

## Technical Decisions

### Calling Convention: System V AMD64 ABI

Linux x86-64 standard:
- Arguments: rdi, rsi, rdx, rcx, r8, r9
- Return: rax, rdx
- Callee-saved: rbx, rbp, r12-r15
- Caller-saved: rax, rcx, rdx, rsi, rdi, r8-r11

We use callee-saved registers for VM state:
- r12: VMContext pointer (persistent)
- r13: Program counter (persistent, actually stored in context)
- r14: Data stack pointer (persistent)
- r15: Jump table base (persistent)

### Value Representation

24-byte tagged union:
```
struct Value {
    uint8_t tag;      // Type discriminator
    uint8_t pad[7];   // Alignment
    int64_t num;      // Numeric value
    void* ptr;        // Pointer for complex types
}
```

**Why 24 bytes?**
- 8-byte alignment for num and ptr fields
- Allows both int64 and float64 in same field (union semantics)
- Pointer for strings, arrays, entities
- Power of 2 sizes (24 = 8*3) for efficient indexing

### Bytecode Encoding

Variable-length instructions:
- 1-byte opcode
- Optional varint operands

**Varint encoding:**
- 7 bits per byte, high bit = continuation
- Little-endian
- Compact for small values (common case)

### Error Handling

No exceptions in assembly - we use error codes:
1. Check for error condition
2. Set error_code in VMContext
3. Jump to vm_error_return
4. Caller checks return value

### Memory Safety

Stack bounds checking on every push/pop:
- Compare against data_limit for overflow
- Compare against data_base for underflow
- Return error codes, don't crash

Type checking for arithmetic:
- Verify tag before operating
- Return ERR_TYPE_MISMATCH for wrong types

## Performance Considerations

### Critical Path Optimization

The dispatch loop is the hottest code:
```asm
vm_dispatch:
    mov rax, [r12 + VMContext.pc]
    cmp rax, [r12 + VMContext.bytecode_len]
    jge vm_exit_success

    mov rdi, [r12 + VMContext.bytecode]
    movzx eax, byte [rdi + rax]     ; opcode
    inc qword [r12 + VMContext.pc]
    inc qword [r12 + VMContext.insn_count]

    jmp [r15 + rax * 8]             ; dispatch!
```

Key optimizations:
- r15 holds jump_table base (no memory load)
- movzx zero-extends byte to avoid partial register stalls
- PC stored in context, not register (allows simpler handler code)

### Avoiding Common Pitfalls

1. **Partial Register Stalls**: Always use movzx when loading bytes
2. **Memory Alignment**: All structures aligned to natural boundaries
3. **Branch Prediction**: Jump table has better prediction than switch
4. **Cache Locality**: Keep VMContext fields accessed together near each other

## Integration Strategy

### C Test Harness

For initial testing, we use a C test harness:
- Allocates and initializes VMContext
- Loads test bytecode
- Calls vm_execute
- Verifies results

### Future Go Integration

Via cgo:
```go
/*
#include "vm.h"
*/
import "C"

func Execute(ctx *VMContext) error {
    err := C.vm_execute((*C.VMContext)(unsafe.Pointer(ctx)))
    if err != 0 {
        return errors.New(errorString(int(err)))
    }
    return nil
}
```

The single entry/exit design minimizes cgo overhead.

## Testing Strategy

### Unit Tests (Current)

Individual opcode tests via C harness:
- Push/pop operations
- Arithmetic operations
- Comparison operations
- Boolean operations
- Error handling

### Integration Tests (Future)

Compare trace output between:
- Go interpreter execution
- NASM VM execution

For the same bytecode, both should produce identical traces.

### Benchmarks (Future)

Compare performance:
- Go interpreter
- NASM VM
- Target: 5-10x speedup for pure computation

## Files and Structure

```
nasm-vm/
├── include/            # Future C header files
├── src/                # Future source organization
│   ├── core/          # VM core implementation
│   ├── ops/           # Opcode handlers (organized)
│   └── test/          # Test code
├── test/               # Test bytecode files
├── build/              # Build outputs
│   └── obj/           # Object files
├── vm_constants.inc    # Constants
├── vm_state.inc        # Structures
├── vm_core.asm         # Macros
├── vm_entry.asm        # Entry point
├── vm_jump_table.asm   # Handlers
├── test_harness.c      # Tests
└── Makefile            # Build
```

## Open Questions

1. **Entity Stack Implementation**: How should entity references work in NASM?
2. **Control Stack**: Frame management for procedure calls
3. **Memory Allocation**: Should VM allocate memory or use pre-allocated pools?
4. **Trace Format**: Binary trace format for comparison with Go runtime
5. **Float Support**: VTAG_DOUBLE implementation using SSE/AVX

## References

1. System V AMD64 ABI: https://gitlab.com/x86-psABIs/x86-64-ABI
2. NASM Documentation: https://nasm.us/doc/
3. LuaJIT Internals: http://wiki.luajit.org/
4. Threaded Code: https://en.wikipedia.org/wiki/Threaded_code
5. Fifth/AISynth Architecture: aisynth/reference-impl/ (internal)

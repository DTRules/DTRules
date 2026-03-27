# Research: NASM VM Stack Operations

**Issue**: #163
**Date**: 2026-03-21

## Background

This document captures research findings for implementing stack operations in the DTRules NASM VM.

## Stack Machine Fundamentals

### Traditional Stack Operations

Stack-based VMs like Forth, PostScript, and the JVM use common stack manipulation primitives:

| Operation | Forth | PostScript | JVM | Effect |
|-----------|-------|------------|-----|--------|
| Duplicate | DUP | dup | dup | a -- a a |
| Pop/Drop | DROP | pop | pop | a -- |
| Swap | SWAP | exch | swap | a b -- b a |
| Over | OVER | -- | dup_x1 | a b -- a b a |
| Rotate | ROT | roll | -- | a b c -- b c a |
| Pick | PICK | index | -- | ... n -- ... nth |

### Value Representation Choices

1. **Unboxed values**: Direct value storage, no type info
2. **Tagged unions**: Type tag + value (our approach)
3. **NaN boxing**: Encode type in IEEE 754 NaN bits
4. **Pointer boxing**: All values are heap objects

Our choice of tagged unions (16 bytes) provides:
- Type safety at runtime
- Simple implementation
- No heap allocation for primitives
- Compatible with 64-bit integer/double operations

## x86-64 Implementation Considerations

### Register Allocation

The System V AMD64 ABI reserves certain registers:
- **Callee-saved**: rbx, rbp, r12-r15 (we use r12-r15 for VM state)
- **Caller-saved**: rax, rcx, rdx, rsi, rdi, r8-r11 (we use for temporaries)

Our register usage:
- r12: Stack pointer (TOS address)
- r13: Stack base (underflow check)
- r14: Stack limit (overflow check)
- r15: Trace buffer (optional)

### Jump Table Dispatch

```nasm
; Fetch opcode and dispatch
movzx eax, byte [rdi]    ; Load opcode
inc rdi                   ; Advance IP
lea rcx, [rel jump_table]
jmp [rcx + rax*8]         ; Jump to handler
```

Advantages:
- O(1) dispatch regardless of opcode count
- No branch misprediction cascade
- Cache-friendly when opcodes are grouped

### Memory Layout

Stack grows upward (higher addresses):
```
r13 (base)  -> [Value 0] <- bottom of stack
               [Value 1]
               [Value 2]
r12 (TOS)   -> [next free slot]
r14 (limit) -> [end of allocated space]
```

Address calculations:
- TOS value: r12 - 16
- Second value: r12 - 32
- Nth value: r12 - (n+1)*16

## Existing Implementation Analysis

### vm_core.asm

Key findings from the implementation:

1. **Value size**: 16 bytes (8 tag + 8 value)
2. **Underflow check**: Compare r12 against r13
3. **Overflow check**: Calculate new position, compare against r14
4. **Type tags**: Defined in opcodes.inc

### Varint Encoding

PUSH_INT uses variable-length integer encoding:
```
Value 0-127:    1 byte  [0xxxxxxx]
Value 128-16383: 2 bytes [1xxxxxxx] [0xxxxxxx]
etc.
```

The current implementation reads until high bit is 0.

### op_rot Implementation

```nasm
op_rot:
    ; (a b c -- b c a)
    ; Load: a at -48, b at -32, c at -16
    ; Store: b at -48, c at -32, a at -16
```

This matches the Forth ROT semantics (third to top).

## Comparison with Go VM

The Go implementation (in `go/pkg/dtrules/interpreter/`) uses:
- `[]Value` for the data stack
- Interface-based values with type switching
- Garbage collection for memory management

The NASM implementation trades flexibility for:
- Predictable performance (no GC pauses)
- Lower memory overhead
- Direct hardware access

## Known Limitations

1. **ROLL not implemented**: Currently a placeholder that just pops n
2. **No signed varint**: PUSH_INT only handles unsigned values
3. **CLEAR clears entire stack**: Doesn't respect marks
4. **No trace recording**: r15 is allocated but unused for stack ops

## References

1. Forth standard: https://forth-standard.org/
2. PostScript Language Reference: Adobe
3. JVM Specification: Chapter 6 (Instruction Set)
4. System V AMD64 ABI: https://gitlab.com/x86-psABIs/x86-64-ABI
5. Intel x86-64 Manual: Volume 2 (Instruction Reference)

## Future Work

1. Complete ROLL implementation
2. Add signed varint support
3. Implement CLEAR-to-mark behavior
4. Add trace recording for debugging
5. Optimize common sequences (e.g., DUP + ADD)

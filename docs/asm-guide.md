# DTRules Assembly Implementation Guide

This document consolidates documentation for the x86-64 assembly implementation of DTRules.

## Overview

Pure x86-64 assembly implementation of the DTRules execution engine. This implementation makes no use of libc - all system interactions are through direct Linux syscalls. The goal is educational: demonstrating how a complete rules engine can be built at the lowest level.

## Design Philosophy

1. **No External Dependencies**: Direct syscalls only, no libc
2. **Value Compatibility**: 24-byte tagged union matches Go implementation exactly
3. **Register-Based VM State**: Critical pointers in callee-saved registers
4. **Arena Allocation**: Simple bump-pointer allocation for predictable memory behavior

## Architecture

### Memory Layout

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
| rcx | Fourth argument | No |
| rax | Return value | No |

## Compatibility with Go/Java

### Value Type Representation

The 24-byte tagged union structure is identical across implementations:

| Language | Implementation |
|----------|----------------|
| Go | `type Value struct { tag byte; _ [7]byte; num int64; ptr unsafe.Pointer }` |
| ASM | 24-byte struct: `[tag:1][pad:7][num:8][ptr:8]` |
| Java | `class Value { byte tag; long num; Object ptr; }` (JVM handles alignment) |

### Type Tags

All implementations use identical type tag values:

| Tag | Type |
|-----|------|
| 0 | NULL |
| 1 | INTEGER |
| 2 | DOUBLE |
| 3 | BOOLEAN |
| 4 | STRING |
| 5 | NAME |
| 6 | ARRAY |
| 7 | ENTITY |

### Stack Operations

Stack operations behave identically:

| Operation | Stack Effect | Notes |
|-----------|--------------|-------|
| push X | -- X | Push value onto stack |
| pop | X -- | Remove top value |
| dup | X -- X X | Duplicate top |
| swap | X Y -- Y X | Exchange top two |
| rot | X Y Z -- Y Z X | Rotate top three |
| over | X Y -- X Y X | Copy second to top |
| pick n | ... X[n] ... -- ... X[n] ... X[n] | Copy nth element |
| roll n | X[n] ... X[0] -- X[n-1] ... X[0] X[n] | Move nth to top |
| clear | ... -- | Empty the stack |

## Known Differences from Go/Java

### Summary

| Area | Go/Java | Assembly | Impact |
|------|---------|----------|--------|
| Error messages | Verbose, descriptive | Terse error codes | Debugging harder |
| Floating point | IEEE 754 (software) | SSE2 (hardware) | Minor precision differences |
| Memory limits | Dynamic, configurable | Fixed at compile time | May run out earlier |
| Unicode | Full UTF-8 support | ASCII only | Non-ASCII strings unsupported |
| Stack size | 10,000+ values | 4,096 values | Deep recursion limited |

### Memory Limits

| Resource | Go | Assembly | Ratio |
|----------|----|----|-------|
| Data stack | 10,000 values | 4,096 values | 2.4x smaller |
| Entity stack | 1,000 entities | 256 entities | 3.9x smaller |
| Control stack | 10,000 frames | 1,024 frames | 9.8x smaller |
| Total heap | Dynamic (OS limit) | 16 MB fixed | - |
| String pool | Dynamic | 1 MB fixed | - |

### Unicode Support

| Operation | Go | Assembly |
|-----------|----|----|
| Character encoding | UTF-8 | ASCII only |
| String length | Code points | Bytes |
| Substring indexing | Code point based | Byte based |
| Case conversion | Unicode-aware | ASCII only (a-z, A-Z) |

## Testing

### Quick Start

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run comparison tests (requires dtrules-go in PATH)
make test-comparison
```

### Test Categories

1. **Unit Tests** (`test/unit/`): Verify individual functions and operations
2. **Comparison Tests** (`test/comparison/`): Verify output matches Go implementation
3. **Integration Tests** (`test/integration/`): Complete workflow verification

### Test Coverage

| Component | Unit Tests | Comparison Tests |
|-----------|------------|------------------|
| Value types | Yes | - |
| Stack operations | Yes | Yes |
| Arithmetic | Yes | Yes |
| Comparison | Yes | - |
| Boolean | Yes | Yes |
| Strings | Stubbed | No |
| Arrays | Stubbed | No |
| Entities | Stubbed | No |
| Control flow | Stubbed | No |
| Decision tables | Stubbed | No |

## Recommendations for Portable Rules

1. **Avoid Unicode**: Use ASCII-only strings for portability
2. **Limit stack depth**: Stay under 1,000 nested calls
3. **Limit string size**: Keep strings under 64KB
4. **Handle precision**: Don't rely on floating point equality
5. **Test both**: Run comparison tests regularly

## Building

```bash
cd asm
make          # Build everything
make test     # Run tests
make clean    # Clean build artifacts
```

## Future Work

1. **String encoding**: Implement UTF-8 support
2. **Extended stack**: Allow configurable stack sizes
3. **Error messages**: Add optional verbose error mode
4. **Bytecode format**: Define portable bytecode specification

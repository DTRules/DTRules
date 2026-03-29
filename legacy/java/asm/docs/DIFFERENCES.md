# Documented Differences

This document catalogs all known behavioral differences between the Assembly implementation and the Go/Java reference implementations. Understanding these differences is essential for:

1. Debugging cross-implementation issues
2. Writing portable rules
3. Knowing when differences are expected vs. bugs

## Error Handling

### Error Message Format

| Scenario | Go | Assembly |
|----------|----|----|
| Stack underflow | "stack underflow at operation 'add'" | "ERR_STACK_UNDERFLOW" |
| Type mismatch | "expected integer, got string" | "ERR_TYPE_MISMATCH" |
| Division by zero | "division by zero" | "ERR_DIV_ZERO" |
| Out of memory | "failed to allocate 1024 bytes" | "ERR_OUT_OF_MEMORY" |
| Invalid opcode | "unknown opcode 0xFF at offset 42" | "ERR_INVALID_OPCODE" |

### Error Recovery

| Behavior | Go | Assembly |
|----------|----|----|
| Continue after error | Configurable | Halts execution |
| Error stack trace | Full call stack | None |
| Error context | Operation name + position | Error code only |

## Memory Limits

### Stack Sizes

| Resource | Go | Assembly | Ratio |
|----------|----|----|-------|
| Data stack | 10,000 values | 4,096 values | 2.4x smaller |
| Entity stack | 1,000 entities | 256 entities | 3.9x smaller |
| Control stack | 10,000 frames | 1,024 frames | 9.8x smaller |

### Heap Size

| Resource | Go | Assembly |
|----------|----|----|
| Total heap | Dynamic (OS limit) | 16 MB fixed |
| String pool | Dynamic | 1 MB fixed |
| Name table | Dynamic hash map | 4,096 buckets fixed |

### String Limits

| Limit | Go | Assembly |
|-------|----|----|
| Maximum string length | 2^31 - 1 | 65,536 |
| Maximum name length | 2^31 - 1 | 256 |

## Unicode Support

### String Handling

| Operation | Go | Assembly |
|-----------|----|----|
| Character encoding | UTF-8 | ASCII only |
| String length | Code points | Bytes |
| Substring indexing | Code point based | Byte based |
| Case conversion | Unicode-aware | ASCII only (a-z, A-Z) |

### Implications

```
# Works identically in both:
"hello" length     → 5

# Different behavior:
"héllo" length     → Go: 5, ASM: 6 (UTF-8 bytes)
"日本語" length    → Go: 3, ASM: 9 (UTF-8 bytes)
```

## Floating Point

### Precision

| Aspect | Go | Assembly |
|--------|----|----|
| Implementation | Software float64 | SSE2 hardware |
| Precision | IEEE 754 double | IEEE 754 double |
| Rounding mode | Round to nearest | Round to nearest |

### Edge Cases

| Case | Go | Assembly |
|------|----|----|
| 0.1 + 0.2 | 0.30000000000000004 | 0.30000000000000004 |
| Very large × very small | May differ in last bits | May differ in last bits |
| NaN propagation | IEEE compliant | IEEE compliant |
| Infinity handling | IEEE compliant | IEEE compliant |

The normalize.py script truncates to 6 decimal places for comparison, masking minor precision differences.

## Integer Overflow

### Behavior

| Scenario | Go | Assembly |
|----------|----|----|
| 2^63-1 + 1 | Wraps to -2^63 | Wraps to -2^63 |
| -2^63 - 1 | Wraps to 2^63-1 | Wraps to 2^63-1 |
| Overflow detection | None (silent wrap) | None (silent wrap) |

Both implementations use two's complement 64-bit arithmetic with silent overflow.

## Division Semantics

### Integer Division

| Operation | Go | Assembly |
|-----------|----|----|
| 7 / 3 | 2 | 2 |
| -7 / 3 | -2 | -2 |
| 7 / -3 | -2 | -2 |
| -7 / -3 | 2 | 2 |

Both truncate toward zero (not floor division).

### Modulo Operation

| Operation | Go | Assembly |
|-----------|----|----|
| 7 % 3 | 1 | 1 |
| -7 % 3 | -1 | -1 |
| 7 % -3 | 1 | 1 |
| -7 % -3 | -1 | -1 |

Sign follows the dividend in both implementations.

## Array Behavior

### Bounds Checking

| Scenario | Go | Assembly |
|----------|----|----|
| Negative index | Error | Error |
| Index >= length | Error | Error |
| Index on empty array | Error | Error |

### Array Growth

| Aspect | Go | Assembly |
|--------|----|----|
| Initial capacity | 8 | 8 |
| Growth factor | 2x | 2x |
| Maximum size | Memory limited | 65536 elements |

## Entity Behavior

### Attribute Lookup

| Scenario | Go | Assembly |
|----------|----|----|
| Missing attribute | Returns null | Returns null |
| Type coercion | None | None |
| Case sensitivity | Case sensitive | Case sensitive |

### Entity Types

Both implementations support typed entities with the same semantics.

## Control Flow

### Recursion Depth

| Aspect | Go | Assembly |
|--------|----|----|
| Maximum depth | ~10,000 | ~1,000 |
| Tail call optimization | None | None |

### Loop Limits

Neither implementation has built-in loop iteration limits.

## Decision Table Execution

### Column Evaluation

| Behavior | Go | Assembly |
|----------|----|----|
| Short-circuit | Yes | Yes |
| Action execution order | Top to bottom | Top to bottom |
| Multiple matches | Execute all | Execute all |

### Truth Value Interpretation

Both use the same truthy/falsy rules:
- null → false
- 0 → false
- false → false
- "" (empty string) → false
- Everything else → true

## I/O Differences

### Print Output

| Aspect | Go | Assembly |
|--------|----|----|
| Integer format | Decimal, no padding | Decimal, no padding |
| Double format | %g (shortest) | Fixed 6 decimal places |
| Boolean format | "true"/"false" | "true"/"false" |
| Null format | "null" | "null" |

### File Handling

| Aspect | Go | Assembly |
|--------|----|----|
| Path encoding | OS native | ASCII only |
| Maximum file size | Memory limited | Memory limited |
| Error messages | OS error strings | Generic error codes |

## Performance Characteristics

While not semantic differences, these affect practical usage:

| Aspect | Go | Assembly |
|--------|----|----|
| Startup time | ~10ms (runtime init) | ~0.1ms |
| Memory overhead | ~10MB baseline | ~1MB baseline |
| Throughput | Competitive | Competitive |
| GC pauses | Occasional | None (arena reset) |

## Recommendations for Portable Rules

1. **Avoid Unicode**: Use ASCII-only strings for portability
2. **Limit stack depth**: Stay under 1,000 nested calls
3. **Limit string size**: Keep strings under 64KB
4. **Handle precision**: Don't rely on floating point equality
5. **Test both**: Run comparison tests regularly

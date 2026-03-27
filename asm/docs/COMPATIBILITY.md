# Compatibility with Go/Java Implementations

This document describes how the Assembly implementation maintains compatibility with the Go and Java versions of DTRules, and where intentional differences exist.

## Identical Behavior

### Value Type Representation

The 24-byte tagged union structure is identical across implementations:

| Language | Implementation |
|----------|----------------|
| Go | `type Value struct { tag byte; _ [7]byte; num int64; ptr unsafe.Pointer }` |
| ASM | 24-byte struct: `[tag:1][pad:7][num:8][ptr:8]` |
| Java | `class Value { byte tag; long num; Object ptr; }` (JVM handles alignment) |

This ensures values can theoretically be serialized and shared between implementations.

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

### Arithmetic Operators

Integer arithmetic produces identical results:

| Operation | Behavior |
|-----------|----------|
| add | a b -- (a+b) |
| sub | a b -- (a-b) |
| mul | a b -- (a*b) |
| div | a b -- (a/b) (integer division, truncates toward zero) |
| mod | a b -- (a%b) (sign follows dividend) |
| neg | a -- (-a) |
| abs | a -- |a| |

### Comparison Operators

Comparison semantics match across implementations:

| Operation | Result |
|-----------|--------|
| eq | Push true if equal |
| ne | Push true if not equal |
| lt | Push true if less than |
| le | Push true if less or equal |
| gt | Push true if greater than |
| ge | Push true if greater or equal |

### Boolean Operators

| Operation | Truth Table |
|-----------|-------------|
| and | T∧T=T, T∧F=F, F∧T=F, F∧F=F |
| or | T∨T=T, T∨F=T, F∨T=T, F∨F=F |
| not | ¬T=F, ¬F=T |
| xor | T⊕T=F, T⊕F=T, F⊕T=T, F⊕F=F |

### Decision Table Structure

The CNode/ANode tree structure for decision tables is identical:
- CNodes represent conditions with true/false branches
- ANodes represent actions to execute
- Tables are traversed top-to-bottom, executing matching actions

## Known Differences

See [DIFFERENCES.md](DIFFERENCES.md) for detailed documentation of behavioral differences.

### Summary of Differences

| Area | Go/Java | Assembly | Impact |
|------|---------|----------|--------|
| Error messages | Verbose, descriptive | Terse error codes | Debugging harder |
| Floating point | IEEE 754 (software) | SSE2 (hardware) | Minor precision differences |
| Memory limits | Dynamic, configurable | Fixed at compile time | May run out earlier |
| Unicode | Full UTF-8 support | ASCII only | Non-ASCII strings unsupported |
| Stack size | 10,000+ values | 4,096 values | Deep recursion limited |

## Testing Compatibility

The comparison test suite verifies behavioral compatibility:

```bash
# Run comparison tests (requires dtrules-go in PATH)
make test-comparison

# Output shows differences
# PASS: Outputs match after normalization
# FAIL: Semantic differences found
```

### Output Normalization

The `normalize.py` script handles known acceptable differences:
- Timestamp removal
- Floating point precision (normalized to 6 decimal places)
- Error message canonicalization
- Whitespace normalization

## Bytecode Compatibility

The assembly implementation uses the same opcode values as Go/Java, enabling potential bytecode sharing:

```
0x10 = OP_ADD
0x11 = OP_SUB
0x12 = OP_MUL
...
```

However, full bytecode compatibility requires:
1. String/Name interning format agreement
2. Decision table binary format specification
3. Entity serialization format

These are not yet standardized across implementations.

## Future Compatibility Work

1. **String encoding**: Implement UTF-8 support in assembly
2. **Extended stack**: Allow configurable stack sizes
3. **Error messages**: Add optional verbose error mode
4. **Bytecode format**: Define portable bytecode specification

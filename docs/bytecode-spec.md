# DTRules Bytecode Specification

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2026-02

## Overview

DTRules bytecode is a compact, portable representation of compiled rule expressions. It is designed to be:

- **Language-agnostic**: Executable by any runtime (Go, Java, x86-64 ASM, ARM, WASM)
- **Compact**: Minimizes memory footprint and improves cache locality
- **Versioned**: Supports forward/backward compatibility
- **Serializable**: Can be stored, transmitted, and pre-compiled

## Binary Format

### File Structure

```
+------------------+
| Magic (4 bytes)  |  "DTBC"
+------------------+
| Version (1 byte) |  Currently 1
+------------------+
| Name Pool        |
+------------------+
| Constant Pool    |
+------------------+
| Bytecode         |
+------------------+
```

### Magic Number

- **Bytes**: `0x44 0x54 0x42 0x43` ("DTBC")
- **Purpose**: Identifies the file as DTRules bytecode

### Version

- **Byte**: Version number (currently `1`)
- **Compatibility**: Readers should reject unknown major versions

### Variable-Length Integer Encoding (Varint)

Integers are encoded using a variable-length format similar to Protocol Buffers:

```
For each byte:
  - Bits 0-6: 7 bits of data
  - Bit 7: Continuation flag (1 = more bytes follow)
```

**Examples**:
| Value | Encoding |
|-------|----------|
| 0     | `0x00`   |
| 127   | `0x7F`   |
| 128   | `0x80 0x01` |
| 300   | `0xAC 0x02` |

### Signed Varint (Svarint)

Signed integers use zigzag encoding before varint encoding:

```
encoded = (value << 1) ^ (value >> 63)
```

This maps small negative numbers to small positive encodings:
| Value | Zigzag | Encoding |
|-------|--------|----------|
| 0     | 0      | `0x00`   |
| -1    | 1      | `0x01`   |
| 1     | 2      | `0x02`   |
| -2    | 3      | `0x03`   |

## Name Pool

The name pool stores RName references used by the bytecode.

### Format

```
+-------------------+
| Count (varint)    |  Number of names
+-------------------+
| Name entries...   |
+-------------------+
```

### Name Entry

```
+-------------------+
| Length (varint)   |  String length in bytes
+-------------------+
| UTF-8 data        |  String bytes
+-------------------+
```

Names are interned at load time - identical strings resolve to the same RName instance.

## Constant Pool

The constant pool stores literal values used by the bytecode.

### Format

```
+-------------------+
| Count (varint)    |  Number of constants
+-------------------+
| Constant entries  |
+-------------------+
```

### Constant Entry

```
+-------------------+
| Type tag (1 byte) |  Value type discriminator
+-------------------+
| Type-specific data|
+-------------------+
```

### Type Tags

| Tag | Type    | Data Format |
|-----|---------|-------------|
| 0   | Null    | (none) |
| 1   | Integer | Signed varint |
| 2   | Double  | 8 bytes, little-endian IEEE 754 |
| 3   | Boolean | 1 byte (0=false, 1=true) |
| 4   | String  | Length (varint) + UTF-8 bytes |
| 5   | Name    | Length (varint) + UTF-8 bytes |
| 6   | Array   | Reserved for future use |
| 7   | Entity  | Reserved for future use |
| 8   | Object  | Reserved for runtime-specific extensions |

## Bytecode Section

### Format

```
+-------------------+
| Length (varint)   |  Bytecode length in bytes
+-------------------+
| Instructions      |  Bytecode stream
+-------------------+
```

## Instruction Set

### Instruction Format

Each instruction consists of:
1. **Opcode** (1 byte)
2. **Operands** (optional, varint-encoded)

### Opcode Categories

#### Stack Operations (0x00-0x0F)

| Opcode | Mnemonic | Operands | Description |
|--------|----------|----------|-------------|
| 0x00   | NOP      | -        | No operation |
| 0x01   | PUSH     | index    | Push constant from pool |
| 0x02   | PUSH_INT | value    | Push immediate integer |
| 0x03   | POP      | -        | Pop and discard top |
| 0x04   | DUP      | -        | Duplicate top |
| 0x05   | SWAP     | -        | Swap top two elements |
| 0x06   | ROT      | -        | Rotate top three (a b c -> b c a) |
| 0x07   | ROLL     | n        | Roll n elements |
| 0x08   | INDEX    | n        | Copy nth element to top |
| 0x09   | CLEAR    | -        | Clear stack to mark |
| 0x0A   | MARK     | -        | Push stack mark |

#### Arithmetic Operations (0x14-0x1F)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x14   | ADD      | (a b -- a+b) | Addition |
| 0x15   | SUB      | (a b -- a-b) | Subtraction |
| 0x16   | MUL      | (a b -- a*b) | Multiplication |
| 0x17   | DIV      | (a b -- a/b) | Division |
| 0x18   | MOD      | (a b -- a%b) | Modulo |
| 0x19   | NEG      | (a -- -a)    | Negation |
| 0x1A   | ABS      | (a -- |a|)   | Absolute value |
| 0x1B   | INC      | (a -- a+1)   | Increment |
| 0x1C   | DEC      | (a -- a-1)   | Decrement |

#### Comparison Operations (0x1E-0x23)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x1E   | EQ       | (a b -- a==b) | Equal |
| 0x1F   | NE       | (a b -- a!=b) | Not equal |
| 0x20   | LT       | (a b -- a<b)  | Less than |
| 0x21   | LE       | (a b -- a<=b) | Less or equal |
| 0x22   | GT       | (a b -- a>b)  | Greater than |
| 0x23   | GE       | (a b -- a>=b) | Greater or equal |

#### Boolean Operations (0x28-0x2B)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x28   | AND      | (a b -- a&&b) | Logical AND |
| 0x29   | OR       | (a b -- a\|\|b) | Logical OR |
| 0x2A   | NOT      | (a -- !a)     | Logical NOT |
| 0x2B   | XOR      | (a b -- a^b)  | Logical XOR |

#### Control Flow (0x32-0x3B)

| Opcode | Mnemonic | Operands | Description |
|--------|----------|----------|-------------|
| 0x32   | EXEC     | -        | Execute object on stack |
| 0x33   | IF       | -        | Pop bool and block, execute if true |
| 0x34   | IFELSE   | -        | Pop bool and two blocks, execute conditionally |
| 0x35   | WHILE    | -        | Pop condition and body blocks, loop |
| 0x36   | FOR      | -        | For loop with counter |
| 0x37   | FORALL   | -        | Iterate over array |
| 0x38   | RETURN   | -        | Return from procedure |
| 0x39   | JUMP     | offset   | Unconditional jump |
| 0x3A   | JUMP_IF  | offset   | Conditional jump (if top is true) |
| 0x3B   | CALL     | index    | Call operator by index |

#### Entity Operations (0x3C-0x40)

| Opcode | Mnemonic | Operands | Description |
|--------|----------|----------|-------------|
| 0x3C   | ENTITY_PUSH | -     | Push entity to entity stack |
| 0x3D   | ENTITY_POP  | -     | Pop entity from entity stack |
| 0x3E   | DEF      | -        | Define variable (name value --) |
| 0x3F   | LOOKUP   | -        | Lookup name in context (name -- value) |
| 0x40   | NEW_ENTITY | -      | Create new entity (name -- entity) |

#### Array Operations (0x46-0x4A)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x46   | NEW_ARRAY | (-- array)    | Create empty array |
| 0x47   | ADD_TO   | (array elem -- array) | Add element to array |
| 0x48   | LENGTH   | (array -- len) | Get array length |
| 0x49   | GET      | (array idx -- elem) | Get element at index |
| 0x4A   | PUT      | (array idx elem -- array) | Set element at index |

#### Table Operations (0x50-0x52)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x50   | NEW_TABLE | (-- table)   | Create empty hashtable |
| 0x51   | TABLE_GET | (table key -- value) | Get value by key |
| 0x52   | TABLE_PUT | (table key value -- table) | Set key-value pair |

#### String Operations (0x5A-0x5B)

| Opcode | Mnemonic | Stack Effect | Description |
|--------|----------|--------------|-------------|
| 0x5A   | CONCAT   | (a b -- a+b) | String concatenation |
| 0x5B   | SUBSTRING | (str start len -- substr) | Extract substring |

#### Constant Shortcuts (0x64-0x68)

| Opcode | Mnemonic | Description |
|--------|----------|-------------|
| 0x64   | PUSH_TRUE  | Push boolean true |
| 0x65   | PUSH_FALSE | Push boolean false |
| 0x66   | PUSH_NULL  | Push null |
| 0x67   | PUSH_ZERO  | Push integer 0 |
| 0x68   | PUSH_ONE   | Push integer 1 |

#### Extended Operations (0xC8-0xCA)

| Opcode | Mnemonic | Operands | Description |
|--------|----------|----------|-------------|
| 0xC8   | OPERATOR | index    | Call operator from runtime table |
| 0xC9   | CONSTANT | index    | Push constant by pool index |
| 0xCA   | NAME     | index    | Push name by pool index |

## Value Types

### Memory Layout (24 bytes)

```
+--------+--------+--------+
| tag    | padding| num    |
| 1 byte | 7 bytes| 8 bytes|
+--------+--------+--------+
| ptr                      |
| 8 bytes                  |
+--------------------------+
```

### Type Tags

| Tag | Type    | num usage | ptr usage |
|-----|---------|-----------|-----------|
| 0   | Null    | unused    | unused |
| 1   | Integer | int64 value | unused |
| 2   | Double  | float64 bits | unused |
| 3   | Boolean | 0=false, 1=true | unused |
| 4   | String  | unused | pointer to string header |
| 5   | Name    | unused | pointer to RName |
| 6   | Array   | unused | pointer to RArray |
| 7   | Entity  | unused | pointer to Entity |
| 8   | Object  | type ptr (as int64) | data ptr |

## Execution Model

### Data Stack

- Main stack for expression evaluation
- Holds Value instances
- Operations consume and produce values

### Entity Stack

- Provides name resolution context
- Entities pushed/popped during execution
- Lookup searches from top to bottom

### Control Stack (Implementation-Specific)

- Used for control flow constructs
- May hold frames, return addresses, marks

## Error Codes

| Code | Name | Description |
|------|------|-------------|
| 0    | ERR_NONE | No error |
| 1    | ERR_STACK_OVERFLOW | Data stack overflow |
| 2    | ERR_STACK_UNDERFLOW | Data stack underflow |
| 3    | ERR_TYPE_MISMATCH | Type error in operation |
| 4    | ERR_DIV_BY_ZERO | Division by zero |
| 5    | ERR_OUT_OF_MEMORY | Memory allocation failed |
| 6    | ERR_INVALID_OPCODE | Unknown opcode |
| 7    | ERR_INDEX_BOUNDS | Array index out of bounds |
| 8    | ERR_NAME_NOT_FOUND | Name lookup failed |
| 9    | ERR_ENTITY_STACK_UNDERFLOW | Entity stack underflow |
| 10   | ERR_INVALID_ENTITY | Invalid entity reference |

## Versioning

### Current Version: 1

### Compatibility Rules

1. **Major version changes** indicate breaking changes
2. **Minor additions** (new opcodes in reserved ranges) are backward-compatible
3. **Unknown opcodes** should cause execution to fail with ERR_INVALID_OPCODE
4. **Unknown value types** should be treated as Null or cause an error

### Reserved Opcode Ranges

| Range | Purpose |
|-------|---------|
| 0x00-0x0F | Stack operations |
| 0x10-0x1F | Arithmetic |
| 0x20-0x2F | Comparison/Boolean |
| 0x30-0x3F | Control flow |
| 0x40-0x4F | Entity/Array |
| 0x50-0x5F | Table/String |
| 0x60-0x6F | Constant shortcuts |
| 0x70-0xBF | Reserved for future use |
| 0xC0-0xCF | Extended operations |
| 0xD0-0xEF | Reserved for extensions |
| 0xF0-0xFF | Implementation-specific |

## Examples

### Simple Expression: `3 + 4`

```
PUSH_INT 3     ; 0x02 0x03
PUSH_INT 4     ; 0x02 0x04
ADD            ; 0x14
```

Serialized (5 bytes + headers):
```
44 54 42 43    ; Magic "DTBC"
01             ; Version 1
00             ; 0 names
00             ; 0 constants
05             ; Code length = 5
02 03 02 04 14 ; Instructions
```

### Variable Lookup: `age > 18`

```
NAME 0         ; 0xCA 0x00 (lookup "age" from name pool)
LOOKUP         ; 0x3F
PUSH_INT 18    ; 0x02 0x12
GT             ; 0x22
```

Name pool contains: ["age"]

### Conditional: `if condition then action`

```
; Push condition result (assumed already on stack)
; Push action bytecode block
IF             ; 0x33
```

## Implementation Notes

### Optimization Hints

1. **Constant folding**: Pre-compute constant expressions at compile time
2. **Peephole optimization**: Combine common instruction sequences
3. **Name interning**: Deduplicate RName instances
4. **Small integer caching**: Reuse common integer values

### Thread Safety

- BytecodeChunk is immutable after creation
- Each ExecutionContext maintains independent stack state
- Multiple contexts can share the same BytecodeChunk

### Memory Management

- Values with pointers (String, Array, Entity) follow language GC rules
- ASM runtime uses explicit reference counting or arena allocation
- Cross-runtime calls must marshal values appropriately

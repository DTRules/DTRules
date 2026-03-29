# DTRules Bytecode Format Specification

This document describes the bytecode format used by the DTRules decision table engine for compiled expressions.

## Overview

DTRules bytecode provides a compact, cache-friendly representation for compiled postfix expressions. Instead of storing Object interfaces, bytecode uses:
- Single-byte opcodes for operators
- Variable-length integers (varints) for arguments
- Constant and name pools for complex values

## Bytecode Chunk Structure

A `BytecodeChunk` consists of three parts:

| Component | Description |
|-----------|-------------|
| `code`    | Bytecode instructions (byte array) |
| `constants` | Constant pool for immediate values, strings, etc. |
| `names`   | Name pool for variable/entity lookups |

## Instruction Format

Instructions follow one of three formats:

1. **Simple opcode** (1 byte): `[opcode]`
2. **Opcode with varint argument**: `[opcode][varint]`
3. **Opcode with pool index**: `[opcode][varint_index]`

### Variable-Length Integer (Varint) Encoding

Arguments are encoded using unsigned varints:
- Each byte uses 7 bits for data, 1 bit (high bit) as continuation flag
- High bit = 1 means more bytes follow
- High bit = 0 means this is the last byte
- Bytes are stored in little-endian order

Example encodings:
| Value | Encoding |
|-------|----------|
| 42    | `0x2A` (1 byte) |
| 127   | `0x7F` (1 byte) |
| 128   | `0x80 0x01` (2 bytes) |
| 300   | `0xAC 0x02` (2 bytes) |

## Opcode Reference

### Stack Operations (0-10)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpNop` | 0 | No operation | ( -- ) |
| `OpPush` | 1 | Push constant from pool | ( -- value ) |
| `OpPushInt` | 2 | Push immediate integer (varint follows) | ( -- int ) |
| `OpPop` | 3 | Pop and discard top | ( a -- ) |
| `OpDup` | 4 | Duplicate top of stack | ( a -- a a ) |
| `OpSwap` | 5 | Swap top two elements | ( a b -- b a ) |
| `OpRot` | 6 | Rotate top three | ( a b c -- b c a ) |
| `OpRoll` | 7 | Roll n items | ( ... n -- ... ) |
| `OpIndex` | 8 | Copy nth item to top | ( ... n -- ... item ) |
| `OpClear` | 9 | Clear stack to mark | ( mark ... -- ) |
| `OpMark` | 10 | Push mark onto stack | ( -- mark ) |

### Arithmetic Operations (20-28)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpAdd` | 20 | Addition | ( a b -- a+b ) |
| `OpSub` | 21 | Subtraction | ( a b -- a-b ) |
| `OpMul` | 22 | Multiplication | ( a b -- a*b ) |
| `OpDiv` | 23 | Division | ( a b -- a/b ) |
| `OpMod` | 24 | Modulo | ( a b -- a%b ) |
| `OpNeg` | 25 | Negate | ( a -- -a ) |
| `OpAbs` | 26 | Absolute value | ( a -- |a| ) |
| `OpInc` | 27 | Increment by 1 | ( a -- a+1 ) |
| `OpDec` | 28 | Decrement by 1 | ( a -- a-1 ) |

### Comparison Operations (30-35)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpEq` | 30 | Equal | ( a b -- bool ) |
| `OpNe` | 31 | Not equal | ( a b -- bool ) |
| `OpLt` | 32 | Less than | ( a b -- bool ) |
| `OpLe` | 33 | Less than or equal | ( a b -- bool ) |
| `OpGt` | 34 | Greater than | ( a b -- bool ) |
| `OpGe` | 35 | Greater than or equal | ( a b -- bool ) |

### Boolean Operations (40-43)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpAnd` | 40 | Logical AND | ( a b -- bool ) |
| `OpOr` | 41 | Logical OR | ( a b -- bool ) |
| `OpNot` | 42 | Logical NOT | ( a -- bool ) |
| `OpXor` | 43 | Logical XOR | ( a b -- bool ) |

### Control Flow Operations (50-59)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpExec` | 50 | Execute top of stack | ( code -- ... ) |
| `OpIf` | 51 | Conditional execution | ( bool code -- ... ) |
| `OpIfElse` | 52 | If-else | ( bool then else -- ... ) |
| `OpWhile` | 53 | While loop | ( cond body -- ... ) |
| `OpFor` | 54 | For loop | ( init cond step body -- ... ) |
| `OpForAll` | 55 | Iterate array | ( array code -- ... ) |
| `OpReturn` | 56 | Return from procedure | ( -- ) |
| `OpJump` | 57 | Unconditional jump (offset follows) | ( -- ) |
| `OpJumpIf` | 58 | Conditional jump | ( bool -- ) |
| `OpCall` | 59 | Call operator by index | ( -- ... ) |

### Entity Operations (60-64)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpEntityPush` | 60 | Push entity onto entity stack | ( entity -- ) |
| `OpEntityPop` | 61 | Pop entity from entity stack | ( -- entity ) |
| `OpDef` | 62 | Define variable in context | ( value name -- ) |
| `OpLookup` | 63 | Lookup name in context | ( name -- value ) |
| `OpNewEntity` | 64 | Create new entity | ( type -- entity ) |

### Array Operations (70-74)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpNewArray` | 70 | Create new array | ( -- array ) |
| `OpAddTo` | 71 | Add element to array | ( array value -- array ) |
| `OpLength` | 72 | Get array length | ( array -- length ) |
| `OpGet` | 73 | Get array element | ( array index -- value ) |
| `OpPut` | 74 | Put array element | ( array index value -- array ) |

### Table Operations (80-82)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpNewTable` | 80 | Create new table | ( -- table ) |
| `OpTableGet` | 81 | Get table value | ( table key -- value ) |
| `OpTablePut` | 82 | Put table value | ( table key value -- table ) |

### String Operations (90-91)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpConcat` | 90 | String concatenation | ( a b -- str ) |
| `OpSubstring` | 91 | Substring | ( str start end -- substr ) |

### Constant Push Operations (100-104)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpPushTrue` | 100 | Push boolean true | ( -- true ) |
| `OpPushFalse` | 101 | Push boolean false | ( -- false ) |
| `OpPushNull` | 102 | Push null | ( -- null ) |
| `OpPushZero` | 103 | Push integer 0 | ( -- 0 ) |
| `OpPushOne` | 104 | Push integer 1 | ( -- 1 ) |

### Extended Operations (200-202)

| Opcode | Value | Description | Stack Effect |
|--------|-------|-------------|--------------|
| `OpOperator` | 200 | Call operator by index (varint follows) | ( ... -- ... ) |
| `OpConstant` | 201 | Push constant by pool index (varint follows) | ( -- value ) |
| `OpName` | 202 | Push name by pool index (varint follows) | ( -- name ) |

## Serialization Format

Bytecode chunks can be serialized to a binary format for storage or transmission.

### File Header

| Offset | Size | Description |
|--------|------|-------------|
| 0 | 4 bytes | Magic: "DTBC" |
| 4 | 1 byte | Version (currently 1) |

### Names Section

| Field | Encoding |
|-------|----------|
| Count | varint |
| Names | For each: [length:varint][bytes:utf8] |

### Constants Section

| Field | Encoding |
|-------|----------|
| Count | varint |
| Constants | For each: [type:byte][data:varies] |

#### Constant Type Tags

| Tag | Type | Data Format |
|-----|------|-------------|
| 0 | Null | (no data) |
| 1 | Integer | signed varint (zigzag encoded) |
| 2 | Double | 8 bytes (IEEE 754, little-endian) |
| 3 | Boolean | 1 byte (0=false, 1=true) |
| 4 | String | [length:varint][bytes:utf8] |
| 5 | Name | [length:varint][bytes:utf8] |

### Code Section

| Field | Encoding |
|-------|----------|
| Length | varint |
| Code | raw bytes |

## Example Bytecode Sequences

### Example 1: Simple Addition (3 + 4)
Postfix: `3 4 +`

```
OpPushInt 3    ; 02 03     ( -- 3 )
OpPushInt 4    ; 02 04     ( 3 -- 3 4 )
OpAdd          ; 14        ( 3 4 -- 7 )
```
Total: 5 bytes

### Example 2: Boolean Expression (x > 5 and y < 10)
Postfix: `x 5 > y 10 < and`

```
OpName 0       ; CA 00     ( -- /x )
OpLookup       ; 3F        ( /x -- x_value )
OpPushInt 5    ; 02 05     ( x_value -- x_value 5 )
OpGt           ; 22        ( x_value 5 -- bool1 )
OpName 1       ; CA 01     ( bool1 -- bool1 /y )
OpLookup       ; 3F        ( bool1 /y -- bool1 y_value )
OpPushInt 10   ; 02 0A     ( ... -- ... 10 )
OpLt           ; 20        ( y_value 10 -- bool2 )
OpAnd          ; 28        ( bool1 bool2 -- result )
```

### Example 3: Variable Assignment
Postfix: `100 /total def`

```
OpPushInt 100  ; 02 64     ( -- 100 )
OpName 0       ; CA 00     ( 100 -- 100 /total )
OpDef          ; 3E        ( 100 /total -- )
```

### Example 4: Optimized Constants
The compiler optimizes common values:

| Expression | Bytecode | Description |
|------------|----------|-------------|
| `true` | `64` | OpPushTrue |
| `false` | `65` | OpPushFalse |
| `null` | `66` | OpPushNull |
| `0` | `67` | OpPushZero |
| `1` | `68` | OpPushOne |
| `42` | `02 2A` | OpPushInt + varint(42) |
| `1000` | `C9 [idx]` | OpConstant + pool index |

## Compiler Optimizations

The bytecode compiler applies several optimizations:

1. **Constant folding**: Common values (true, false, null, 0, 1) use dedicated opcodes
2. **Small integer immediates**: Integers 0-127 are encoded inline with OpPushInt
3. **Operator mapping**: Common operators (+, -, *, /, comparisons) map to dedicated opcodes
4. **Name deduplication**: Repeated names share a single pool entry

## Implementation Files

- `bytecode.go`: Opcode definitions, BytecodeChunk, serialization
- `interpreter/vm.go`: VM execution of bytecode (ExecuteBytecode)
- `compiler/bytecode_compiler.go`: Compilation from postfix to bytecode

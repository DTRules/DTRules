> **NOT FOR RULE AUTHORS.** Rule authors MUST use `dtrules docs el`.
> This file covers how the EL compiler and runtime work internally.
> It is useful for engineers debugging the compiler, the VM, or performance
> issues. It is not a rule authoring reference.

# Compiler Internals

## 1. Architecture Overview

EL source text travels through four distinct stages before it runs:

```
EL source text
     │
     ▼
┌─────────────────────────────────────────────────────────────┐
│  ANTLR4 layer  (pkg/dtrules/compiler/el/)                   │
│                                                             │
│  EL.g4 grammar → el_lexer.go + el_parser.go                │
│  PostfixEmitter visits parse tree → postfix token stream    │
└────────────────────────┬────────────────────────────────────┘
                         │  postfix token string
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Postfix compiler  (pkg/dtrules/compiler/)                  │
│                                                             │
│  Tokeniser → per-token dispatch → executable Object array   │
│  Operators resolved by name; DTs resolved from Factory      │
└────────────────────────┬────────────────────────────────────┘
                         │  []Object (RArray, executable)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Bytecode compiler  (pkg/dtrules/bytecode.go)               │
│                                                             │
│  BytecodeChunk: []byte code + []Value constants + []*RName  │
│  Opcodes: 1 byte each; arguments: unsigned LEB128 varints   │
└────────────────────────┬────────────────────────────────────┘
                         │  *BytecodeChunk
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  VM / runtime  (pkg/dtrules/interpreter/ + runtime/)        │
│                                                             │
│  DTState runs the Object-based path                         │
│  ExecuteBytecode runs the Value-based fast path             │
│  GoRuntime / NativeRuntime wrap the state as contexts       │
└─────────────────────────────────────────────────────────────┘
```

The two execution paths (Object-based and Value-based) are intentionally
maintained in parallel. The Object-based path handles the full operator
catalogue including complex entity operations. The Value-based (bytecode)
path is the hot path for numeric and boolean expressions.

---

## 2. ANTLR4 Layer

### Grammar file

`pkg/dtrules/compiler/el/EL.g4` – case-insensitive ANTLR4 grammar.

Generated files (do not edit by hand):
- `el_lexer.go` – tokeniser
- `el_parser.go` – recursive-descent parser
- `el_listener.go` / `el_visitor.go` – visitor/listener interfaces
- `el_base_listener.go` / `el_base_visitor.go` – no-op base implementations

### Top-level rule

```
done
    : ACTION statementList          // action cells
    | CONDITION bexpr               // condition cells
    | CONTEXT contextForTable       // iteration/local-variable context
    | POLICYSTATEMENT <expr>        // policy output cells
    ;
```

Each `done` alternative maps to one cell type in a decision table.

### Expression rules

| Rule | Go type tag | Produces |
|------|-------------|---------|
| `bexpr` | boolean | Conditions, `if` guards |
| `iexpr` | integer / long | Integer arithmetic |
| `fexpr` | double / float | Floating-point arithmetic |
| `dexpr` | date | Date arithmetic and comparisons |
| `strexpr` | string | String operations |
| `eexpr` | entity | Entity references |
| `nexpr` | name | Name (`$name`) references |
| `arrayExpr` / `arrayExpr2` | array | Array expressions |
| `bigexpr` | bigint | Arbitrary-precision integers |
| `texpr` | table | Decision table references |

All typed identifiers (`typedLong`, `typedDouble`, etc.) resolve to `IDENT`
at the grammar level; type resolution happens inside `PostfixEmitter` by
consulting the symbol table loaded from the EDD.

### Visitor

`PostfixEmitter` (implements `ELVisitor`) walks the parse tree and emits
postfix tokens. It holds:
- `symbols map[string]string` – attribute name → type string from EDD
- `locals map[string]LocalVar` – local variable name → stack frame index
- `localCnt int` – next local variable slot

`PostfixEmitter.SetSymbols(m)` is called before compilation so the emitter
knows what type each attribute has.

`PostfixEmitter.Emit()` returns the complete postfix string after visiting.

---

## 3. Postfix IR

The postfix token stream uses whitespace-separated tokens. Code blocks are
delimited by `{` … `}` and are compiled recursively.

Stack effect notation: `(before -- after)` where the rightmost value is top.

### 3.1 Literal / load tokens

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `true` | `( -- bool)` | Boolean true literal |
| `false` | `( -- bool)` | Boolean false literal |
| `otherwise` | `( -- bool)` | Synonym for `true` in conditions |
| `null` | `( -- null)` | Null literal |
| `<integer>` | `( -- int)` | Decimal integer literal |
| `<float>` | `( -- double)` | Decimal float literal |
| `"<str>"` | `( -- str)` | Quoted string literal |
| `/<name>` | `( -- name)` | Non-executable name (literal) |
| `<name>` | `( -- obj)` | Executable name – looked up at runtime on entity stack |

### 3.2 Attribute fetch / store

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `/entity.field` | `( -- name)` | Push interned `RName` for an attribute |
| `xdef` | `(val name -- )` | Assign `val` to attribute named by `name` in entity context |

In generated postfix, attribute reads appear as `/entity.attr` followed by
an implicit lookup, and writes appear as `<val> /entity.attr xdef`.

### 3.3 Local variable tokens

Local variables are stack-frame slots inside the control stack.

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `<n> local@` | `( -- val)` | Read local variable at slot `n` |
| `<n> local!` | `(val -- )` | Write local variable at slot `n` |

### 3.4 Type conversion operators

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `cvi` | `(a -- int)` | Convert to integer |
| `cvd` | `(a -- double)` | Convert to double |
| `cvs` | `(a -- str)` | Convert to string |
| `cvb` | `(a -- bool)` | Convert to boolean |
| `cve` | `(a -- entity)` | Convert to entity |
| `cvr` | `(a -- name)` | Convert to RName |
| `cvbi` | `(a -- bigint)` | Convert to BigInteger |

### 3.5 Integer arithmetic

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `+` | `(a b -- a+b)` | Integer addition |
| `-` | `(a b -- a-b)` | Integer subtraction |
| `*` | `(a b -- a*b)` | Integer multiplication |
| `/` | `(a b -- a/b)` | Integer division (truncates) |
| `neg` | `(a -- -a)` | Integer negation |
| `addto` | `(dest val -- dest)` | `dest += val` in-place |

### 3.6 Float arithmetic

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `f+` | `(a b -- a+b)` | Float addition |
| `f-` | — | (expressed as `fnegate` + `f+`) |
| `fmul` | `(a b -- a*b)` | Float multiplication |
| `fdiv` | `(a b -- a/b)` | Float division |
| `bnegate` | `(a -- -a)` | Float negation (prefix `b` for "binary float") |
| `b+` | `(a b -- a+b)` | BigInteger addition |
| `b-` | `(a b -- a-b)` | BigInteger subtraction |
| `b*` | `(a b -- a*b)` | BigInteger multiplication |
| `b/` | `(a b -- a/b)` | BigInteger division |
| `babs` | `(a -- |a|)` | BigInteger absolute value |
| `round` | `(double places -- double)` | Round to n decimal places |

### 3.7 Comparison – integers / BigIntegers

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `==` | `(a b -- bool)` | Integer equal |
| `!=` | — | Expressed as `==` + `not` |
| `<` | `(a b -- bool)` | Integer less-than |
| `<=` | `(a b -- bool)` | Integer less-than-or-equal |
| `>` | `(a b -- bool)` | Integer greater-than |
| `>=` | `(a b -- bool)` | Integer greater-than-or-equal |
| `b==` | `(a b -- bool)` | BigInteger equal |
| `b!=` | — | `b==` + `not` |
| `b<` | `(a b -- bool)` | BigInteger less-than |
| `b<=` | `(a b -- bool)` | BigInteger less-than-or-equal |
| `b>` | `(a b -- bool)` | BigInteger greater-than |
| `b>=` | `(a b -- bool)` | BigInteger greater-than-or-equal |

### 3.8 Comparison – floats

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `f==` | `(a b -- bool)` | Float equal |
| `f<` | `(a b -- bool)` | Float less-than |
| `f<=` | `(a b -- bool)` | Float less-than-or-equal |
| `f>` | `(a b -- bool)` | Float greater-than |
| `f>=` | `(a b -- bool)` | Float greater-than-or-equal |

### 3.9 Comparison – strings / names / entities

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `streq` | `(a b -- bool)` | Case-sensitive string equal |
| `strneq` | `(a b -- bool)` | Case-sensitive string not-equal |
| `sic==` | `(a b -- bool)` | Case-insensitive string equal |
| `req` | `(a b -- bool)` | Reference (identity) equal |
| `beq` | `(a b -- bool)` | Boolean equal |

### 3.10 Comparison – dates

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `d==` | `(a b -- bool)` | Date equal |
| `d<` | `(a b -- bool)` | Date less-than (before) |
| `d>` | `(a b -- bool)` | Date greater-than (after) |

### 3.11 Boolean logic

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `not` | `(a -- !a)` | Boolean NOT |
| `isnull` | `(a -- bool)` | True if top is null |

Short-circuit AND / OR use the stack manipulation pattern:

```
// a AND b:
<eval a>
{ pop <eval b> } over if
// Leaves (a && b) on stack; short-circuits if a is false

// a OR b:
<eval a>
{ pop <eval b> } over not if
// Short-circuits if a is true
```

The `{` `}` tokens denote a code block pushed as an executable array.

### 3.12 Control flow

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `{...}` | `( -- block)` | Push executable block |
| `if` | `(bool block -- )` | Execute block if bool is true |
| `ifelse` | `(bool tblock fblock -- )` | Execute tblock or fblock |
| `{}` | `( -- )` | Empty block (no-op) |
| `over` | `(a b -- a b a)` | Copy second item to top |
| `pop` | `(a -- )` | Discard top |
| `swap` | `(a b -- b a)` | Swap top two |

### 3.13 Entity / session operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `entitypush` | `(entity -- )` | Push entity onto entity stack |
| `entitypop` | `( -- entity)` | Pop entity from entity stack |
| `allocate` | `(name -- entity)` | Create new entity instance |
| `deallocate` | `(entity -- )` | Remove entity from session |
| `createentity` | `(name -- entity)` | Synonym for `allocate` |
| `clone` | `(entity -- entity)` | Shallow clone of entity |
| `execute` | `(block -- )` | Execute the top block |

### 3.14 Array operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `addarray` | `(src dst -- dst)` | Append all elements of `src` to `dst` |
| `addto` | `(val array -- array)` | Append `val` to `array` |
| `numberof` | `(array -- int)` | Element count |
| `length` | `(array -- int)` | Synonym for `numberof` |
| `copy` | `(array -- copy)` | Shallow copy |
| `memberof` | `(val array -- bool)` | Membership test |
| `findmatch` | `(array cond -- entity)` | Find first matching element |
| `policystatements` | `( -- array)` | Push policy-statements array |

### 3.15 Date operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `currentdate` | `( -- date)` | Today's date |
| `days` | `(int -- interval)` | Convert integer to day interval |
| `adddays` | `(date int -- date)` | Add N days |
| `subdays` | `(date int -- date)` | Subtract N days |
| `addmonths` | `(date int -- date)` | Add N months |
| `submonths` | `(date int -- date)` | Subtract N months |
| `addyears` | `(date int -- date)` | Add N years |
| `subyears` | `(date int -- date)` | Subtract N years |

### 3.16 String operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `strconcat` | `(a b -- str)` | Concatenate strings |
| `trim` | `(str -- str)` | Trim whitespace |
| `tolower` | `(str -- str)` | Lowercase |
| `toupper` | `(str -- str)` | Uppercase |

### 3.17 Name operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `nameof` | `(entity -- name)` | Get entity's `RName` |

### 3.18 Relationship operations

| Token | Stack effect | Notes |
|-------|-------------|-------|
| `getrelationship` | `(str entity -- str)` | Attribute-as-relationship lookup |
| `hasrelationship` | `(str entity -- bool)` | True if relationship exists |
| `relationships` | `(entity -- array)` | All relationship names |

### 3.19 Slot notation used by context / entity references

`/type`, `/source`, `/target` are slot tokens used internally for building
entity references in context expressions. They are not standalone opcodes
but postfix tokens that the postfix compiler resolves.

---

## 4. VM: Bytecode Layer

### 4.1 BytecodeChunk

`dtrules.BytecodeChunk` is the compiled form of a single expression:

```
type BytecodeChunk struct {
    code      []byte   // instruction bytes
    constants []Value  // constant pool
    names     []*RName // name pool
}
```

Arguments to opcodes use unsigned LEB128 (varint) encoding.
Signed integers are zigzag-encoded before varint.

Binary file format (magic `DTBC`, version `1`):
```
[4]byte magic | [1]byte version
varint namesCount | (varint len + utf8)...
varint constsCount | (tag byte + type-specific bytes)...
varint codeLen | code bytes
```

### 4.2 Opcode table

Opcodes are defined in `pkg/dtrules/bytecode.go` as `type Opcode uint8`.

**Stack operations**

| Opcode | Value | Stack effect | Operand |
|--------|-------|-------------|---------|
| `OpNop` | 0 | `( -- )` | — |
| `OpPush` | 1 | `( -- val)` | — |
| `OpPushInt` | 2 | `( -- int)` | varint value |
| `OpPop` | 3 | `(a -- )` | — |
| `OpDup` | 4 | `(a -- a a)` | — |
| `OpSwap` | 5 | `(a b -- b a)` | — |
| `OpRot` | 6 | `(a b c -- b c a)` | — |
| `OpRoll` | 7 | roll n items | — |
| `OpIndex` | 8 | copy nth to top | — |
| `OpClear` | 9 | clear to mark | — |
| `OpMark` | 10 | push mark | — |

**Arithmetic**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpAdd` | 20 | `(a b -- a+b)` |
| `OpSub` | 21 | `(a b -- a-b)` |
| `OpMul` | 22 | `(a b -- a*b)` |
| `OpDiv` | 23 | `(a b -- a/b)` – int division for int/int; float otherwise |
| `OpMod` | 24 | `(a b -- a%b)` |
| `OpNeg` | 25 | `(a -- -a)` |
| `OpAbs` | 26 | `(a -- \|a\|)` |
| `OpInc` | 27 | `(a -- a+1)` |
| `OpDec` | 28 | `(a -- a-1)` |

**Comparison**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpEq` | 30 | `(a b -- a==b)` |
| `OpNe` | 31 | `(a b -- a!=b)` |
| `OpLt` | 32 | `(a b -- a<b)` |
| `OpLe` | 33 | `(a b -- a<=b)` |
| `OpGt` | 34 | `(a b -- a>b)` |
| `OpGe` | 35 | `(a b -- a>=b)` |

**Boolean**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpAnd` | 40 | `(a b -- a&&b)` |
| `OpOr` | 41 | `(a b -- a\|\|b)` |
| `OpNot` | 42 | `(a -- !a)` |
| `OpXor` | 43 | `(a b -- a^b)` |

**Control flow**

| Opcode | Value | Notes |
|--------|-------|-------|
| `OpExec` | 50 | Execute top of stack |
| `OpIf` | 51 | `(bool block -- )` |
| `OpIfElse` | 52 | `(bool tblock fblock -- )` |
| `OpWhile` | 53 | while loop |
| `OpFor` | 54 | for loop |
| `OpForAll` | 55 | iterate array |
| `OpReturn` | 56 | Return from procedure |
| `OpJump` | 57 | Unconditional jump; operand: offset varint |
| `OpJumpIf` | 58 | Conditional jump |
| `OpCall` | 59 | Call operator by index |

**Entity operations**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpEntityPush` | 60 | `(entity -- )` |
| `OpEntityPop` | 61 | `( -- entity)` |
| `OpDef` | 62 | `(val name -- )` |
| `OpLookup` | 63 | `(name -- val)` |
| `OpNewEntity` | 64 | `(name -- entity)` |

**Array operations**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpNewArray` | 70 | `( -- array)` |
| `OpAddTo` | 71 | `(val array -- array)` |
| `OpLength` | 72 | `(array -- int)` |
| `OpGet` | 73 | `(array idx -- val)` |
| `OpPut` | 74 | `(array idx val -- )` |

**Table operations**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpNewTable` | 80 | `( -- table)` |
| `OpTableGet` | 81 | `(table key -- val)` |
| `OpTablePut` | 82 | `(table key val -- )` |

**String operations**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpConcat` | 90 | `(a b -- str)` |
| `OpSubstring` | 91 | `(str start end -- str)` |

**Constant shortcuts**

| Opcode | Value | Stack effect |
|--------|-------|-------------|
| `OpPushTrue` | 100 | `( -- true)` |
| `OpPushFalse` | 101 | `( -- false)` |
| `OpPushNull` | 102 | `( -- null)` |
| `OpPushZero` | 103 | `( -- 0)` |
| `OpPushOne` | 104 | `( -- 1)` |

**Extended (index-bearing) opcodes**

| Opcode | Value | Notes |
|--------|-------|-------|
| `OpOperator` | 200 | Call operator at `operatorTable[varint]` |
| `OpConstant` | 201 | Push `constants[varint]` |
| `OpName` | 202 | Push `names[varint]` as a `Value` |

### 4.3 Small-integer fast path

`BytecodeChunk.EmitPushConstant(v)` applies inline peephole optimisations:
- `true` → `OpPushTrue`
- `false` → `OpPushFalse`
- `null` → `OpPushNull`
- `0` → `OpPushZero`
- `1` → `OpPushOne`
- integers 2–127 → `OpPushInt` + varint (2 bytes total)
- everything else → `OpConstant` + pool index

### 4.4 Value type

`dtrules.Value` is a 24-byte tagged union that avoids interface boxing:

```
struct Value {
    tag uint8           // type discriminator (VTag*)
    _   [7]byte         // alignment padding
    num int64           // integer, boolean (0/1), or float64 bits
    ptr unsafe.Pointer  // string, array, entity, *big.Int, or iface words
}
```

| Tag constant | Go type stored |
|---|---|
| `VTagNull` (0) | — |
| `VTagInteger` (1) | `int64` in `num` |
| `VTagDouble` (2) | `float64` bits in `num` |
| `VTagBoolean` (3) | 0/1 in `num` |
| `VTagString` (4) | `*string` in `ptr` |
| `VTagName` (5) | `*RName` in `ptr` |
| `VTagArray` (6) | `*RArray` in `ptr` |
| `VTagEntity` (7) | `Object` interface (two-word) in `num`+`ptr` |
| `VTagObject` (8) | Any `Object` interface as two-word pair |
| `VTagBigInt` (9) | `*big.Int` in `ptr` |

Integer and boolean arithmetic never allocates. `Add`, `Sub`, `Mul` return
inline `Value` structs. Only float/BigInt operations may allocate.

---

## 5. Runtime Types

### 5.1 Core Object types (pkg/dtrules/)

All rule values implement `dtrules.Object`. Concrete types:

| Type | File | Description |
|------|------|-------------|
| `RInteger` | `integer.go` | 64-bit signed integer |
| `RDouble` | `double.go` | 64-bit IEEE 754 float |
| `RBoolean` | `boolean.go` | True / false (singletons) |
| `RString` | `string.go` | Interned string |
| `RName` | `name.go` | Interned symbol; executable/non-executable variants |
| `RDate` | `date.go` | `time.Time` wrapper |
| `RArray` | `array.go` | Ordered, mutable list of `Object` |
| `RBigInt` | `bigint.go` | `math/big.Int` wrapper |
| `RNull` | `null.go` | Null sentinel |
| `RMark` | `mark.go` | Stack mark (for `clear` to mark) |
| `RTable` | `table.go` | Key→value map |
| `RType` | `types.go` | Type metadata singleton |
| `RInterval` | `interval.go` | Date interval (days/months/years) |
| `RXmlValue` | `xmlvalue_stub.go` | XML attribute value (stub) |
| `BaseObject` | `base.go` | No-op base for embedded structs |

### 5.2 Interfaces (pkg/dtrules/object.go)

| Interface | Implemented by |
|-----------|---------------|
| `Object` | All rule value types |
| `State` | `interpreter.DTState` |
| `Session` | `session.RSession` |
| `Entity` | `entity.REntity` |
| `DecisionTable` | decision table implementations |
| `EntityFactory` | `entity.Factory` |
| `DateParser` | `session.DateParser` |
| `RuleSet` | `session.RuleSet` |
| `RuntimeFactory` | `runtime.GoRuntime`, `runtime.NativeRuntime` |

### 5.3 Error types

| Type | File | Use |
|------|------|-----|
| `RulesError` | `errors.go` | All engine errors; carries `ErrorType`, `Location`, `Message`, optional `Cause` |
| `RuntimeError` | `runtime/runtime.go` | Runtime-layer errors with numeric codes |

Constructor helpers in `errors.go`:
- `ConversionError(loc, msg)` – type mismatch
- `UndefinedError(loc, msg)` – name not found
- `TypeCheckError(loc, msg)` – type assertion failure
- `StackUnderflowError(loc)` – empty stack pop
- `StackOverflowError(loc, msg)` – limit exceeded
- `OutOfBoundsError(loc, msg)` – array/frame index out of range
- `ParsingError(loc, msg)` – parse failure

### 5.4 Runtime implementations

**Context configuration** (`runtime/runtime.go`)

| Type | Role |
|------|------|
| `ContextOption` | `func(*ContextConfig)` functional option applied at context creation |
| `ContextConfig` | Holds `DataStackSize`, `EntityStackSize`, `EnableTracing`, `TraceWriter` |
| `TraceWriter` | Interface for receiving per-instruction trace events |

`WithDataStackSize`, `WithEntityStackSize`, and `WithTracing` return
`ContextOption` values.

**GoRuntime** (`runtime/goruntime/`)

| Type | Role |
|------|------|
| `GoRuntime` | `Runtime` factory; thread-safe, supports concurrent contexts |
| `GoContext` | `ExecutionContext` wrapping a `DTState` |

`GoRuntime.Capabilities()` reports `ConcurrentContexts: true`, `MaxStackDepth: 1000`.

**NativeRuntime** (`runtime/nativeasm/`)

| Type | Role |
|------|------|
| `NativeRuntime` | x86-64 assembly runtime (CGO) |
| `NativeContext` | Context backed by assembly VM |
| `Executor` | Assembly executor implementation |
| `Factory` | NativeRuntime factory |

`NativeRuntime` does not support concurrent contexts (`ConcurrentContexts: false`).
It uses the same `BytecodeChunk` format as `GoRuntime`.

---

## 6. Interpreter: DTState

`interpreter.DTState` (`pkg/dtrules/interpreter/state.go`) is the central
execution environment.

### Stack layout

```
┌──────────────────────────────┐  ◄ top
│  data stack []Object          │  Expression results, operator I/O
├──────────────────────────────┤
│  value stack []Value          │  Fast path – mirrors data stack
├──────────────────────────────┤
│  entity stack []Entity        │  Name resolution scope (like PostScript dict stack)
├──────────────────────────────┤
│  control stack []Object       │  Stack frames, local variables
└──────────────────────────────┘
```

All stacks have a hard limit of 1000 (`stackLimit`).

### Control stack frames

`PushFrame()` records `currentFrame = len(ctrlStk)` and saves the old
frame pointer. Local variables in the current frame are accessed via
`GetFrameValue(i)` / `SetFrameValue(i, v)`. `PopFrame()` restores the
slice and frame pointer.

Local variable postfix tokens (`0 local@`, `0 local!`) are translated by
the bytecode compiler to `GetFrameValue` / `SetFrameValue` calls.

### Name resolution

`DTState.Find(name)` → `FindEntity(name)` → scans entity stack from top
to bottom looking for an entity that `ContainsAttribute(attrName)`.
If the `RName` includes an entity prefix (`entity.attribute`), only entities
whose name matches the prefix are searched.

`DTState.Def(name, value, trace)` uses the same search and calls
`entity.Put(attrName, value)` on the match.

### State flags

| Flag constant | Bit | Effect |
|---|---|---|
| `DEBUG` | 0x01 | Enable debug print statements |
| `TRACE` | 0x02 | Emit XML trace events per operation |
| `ECHO` | 0x04 | Mirror debug output to stdout |
| `VERBOSE` | 0x08 | Trace every push/pop |

---

## 7. Entity / Session Model

### RuleSet ↔ REntity

```
RuleSet
  └── entity.Factory
        ├── reference entities (id=0, readonly)
        │     └── REntity{name, attributes map[*RName]*EntityEntry, values []Object}
        └── decision tables entity (holds DT references)

RSession (one per rule execution)
  ├── entityInstances map[int]Entity  (cloned from reference entities)
  └── DTState
        └── entityStk  (reference entities + cloned instances)
```

`entity.Factory` is created at `RuleSet` load time and holds the reference
(prototype) entities. `NewSession` clones entities as needed and pushes the
operator primitives entity and decision-tables entity onto the entity stack.

### REntity internals

`REntity` stores attribute definitions as `*EntityEntry` values keyed by
`*RName` (pointer-equality, since names are interned). Values are stored in
`[]Object values` and indexed via `EntityEntry.index`. This means attribute
lookup is O(1) after the `map[*RName]` lookup.

Every entity has a synthetic self-reference attribute (name = entity name,
value = `self`) and a `mapping*key` attribute (a string, initially null).
The mapping key is used by operators that correlate entities across arrays
(e.g., `FORALL … IN entity`).

### Session lifecycle

1. `NewRuleSet(name)` → zero-population rule set
2. `rs.LoadEDD(r)` → parse XML, populate `entity.Factory` with reference entities
3. `rs.LoadDT(r)` → parse decision table XML, compile postfix to `[]Object`
4. `session.NewSession(rs)` → clone session state, push operator + DT entities
5. `session.Execute(tableName, entityInstances)` → push entity instances, invoke DT
6. Read attribute values back from entity instances

---

## 8. Error Model

### Parse errors

ANTLR4 parse errors are collected by a custom error listener attached to
the `ELParser`. Each error is wrapped in `dtrules.ParsingError` and returned
to the caller of `el.Compiler.compile*`. The first error terminates
compilation.

### Compile errors

`PostfixEmitter` accumulates errors in `emitter.errors []error`. After
visiting, `Errors()` returns them. The postfix compiler (`compiler.Compiler`)
wraps each token error in `fmt.Errorf("failed to compile token '%s': %w", ...)`.

### Runtime errors

`dtrules.RulesError` is the canonical runtime error. Fields:
- `ErrorType` – category string (e.g. `"Stack Overflow"`, `"Division By Zero"`)
- `Location` – function or operator name
- `Message` – human-readable detail
- `PostFix` – the postfix fragment being executed (set by callers when available)
- `Cause error` – wrapped underlying error

`runtime.RuntimeError` carries a numeric `Code` for machine consumption.
The standard codes are 1–12 and map to the `Err*` sentinel variables in
`runtime/runtime.go`.

---

## 9. Performance Notes

### Hot paths

1. **Numeric conditions** – most condition cells are numeric comparisons
   (`>=`, `==`, etc.). These execute entirely in the Value-based bytecode
   path with no heap allocation for integers.

2. **Entity attribute reads** – `entity.Get(name)` resolves via a
   `map[*RName]` lookup (pointer key, O(1)) then a slice index. Interning
   `RName` values is critical to keeping this O(1).

3. **Short-circuit AND / OR** – the emitter uses the `over`/`if` pattern
   to skip right-hand evaluation. This is entirely postfix; no special
   compiler pass.

### Allocation patterns

- `Value` arithmetic (integer, boolean) – zero allocation.
- `Value` string / BigInt creation – one allocation per new value.
- `RString`, `RInteger`, `RDouble`, `RBoolean`, `RName` – pooled via
  `Get*Value` helpers; common values (integers –128..127 and common strings)
  return pool singletons.
- Each `NewSession` clones entity value slices (`copy(e.values, source.values)`);
  allocation is proportional to total attribute count, not session count.

### Stack depth

Hard limit is 1000 for all three stacks. Deeply recursive decision tables
(each DT invocation pushes the called DT's entities) can hit the entity
stack limit before the data stack limit. Each `FORALL` block adds one entity
stack frame for the loop variable.

### NativeASM vs GoRuntime

The `NativeRuntime` x86-64 assembly backend executes the same
`BytecodeChunk` byte stream as `GoRuntime`. It does not support concurrent
contexts (global register state). Use `GoRuntime` when executing rules in
parallel across goroutines.

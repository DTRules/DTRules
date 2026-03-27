# DTRules Bytecode Runtime -- Porting Guide

This document specifies what a DTRules bytecode runtime must implement. It is
intended as a roadmap for porting the runtime to any language.

The Go implementation with Plan 9 assembly is the reference. The Java
implementation is the original. Both produce identical results on all CHIP
decision table test cases.

---

## 1. Execution Model

DTRules is a PostScript-style stack machine. If you know PostScript or FORTH,
you already understand the model. If not, here are the essential rules:

- There is **one data stack** (the operand stack). All computation happens here.
- There is **one entity stack** (the dictionary stack). Name resolution
  searches here, top to bottom.
- There is **one control stack** for stack frames (decision table calls,
  local variables).
- **Executable objects auto-execute on lookup.** When `lookup` finds a
  procedure or decision table, it invokes it immediately. It does not push
  it. This is the single most important semantic rule.
- Operators are named functions that manipulate the stacks through an
  abstract State interface. They never touch stack internals directly.

There is ONE state object. It owns all three stacks. There are no shadow
copies, no syncing, no fallbacks to another runtime. This is the architecture
that works. An earlier attempt that maintained parallel state and synced
between them was deleted because it was both wrong and slow.

---

## 2. The Value Type

Every entry on the data stack is a Value. This is a tagged union.

### Layout (24 bytes, 64-bit aligned)

```
Byte offset   Size    Field   Contents
-----------   ----    -----   --------
0             1       tag     Type discriminator (see table below)
1-7           7       pad     Padding for alignment
8-15          8       num     64-bit integer, or IEEE 754 double bits, or boolean
16-23         8       ptr     Pointer to heap object (nil for primitive types)
```

### Type Tags

| Tag | Type    | num field                      | ptr field          |
|-----|---------|--------------------------------|--------------------|
| 0   | Null    | 0                              | nil                |
| 1   | Integer | signed 64-bit integer          | nil                |
| 2   | Double  | IEEE 754 double, stored as bits| nil                |
| 3   | Boolean | 0 = false, 1 = true            | nil                |
| 4   | String  | unused                         | pointer to string  |
| 5   | Name    | unused                         | pointer to RName   |
| 6   | Array   | unused                         | pointer to RArray  |
| 7   | Entity  | unused                         | pointer to Entity  |
| 8   | Object  | type word (for interface)      | data pointer       |

Tags 0-3 are the primitive types. For these, `ptr` is always nil. This is
what makes fast-path assembly possible: the assembly only touches `tag` and
`num`, never `ptr`, and never interacts with the heap or garbage collector.

### Adapting the Value Type

The 24-byte layout is the Go implementation's choice. A port does not need
the same byte layout. What matters is:

- You can distinguish types by tag.
- Integers are signed 64-bit.
- Doubles are IEEE 754 64-bit.
- Booleans are integer 0 or 1.
- Heap types carry a pointer/reference the GC can trace.

In C++, this might be a `struct` with `uint8_t tag`, `int64_t num`, `void* ptr`.
In Rust, an `enum`. In FORTH, a cell pair (tag + value) on a dedicated stack
with a side table for pointers. In Python, just use native objects. The
tagged-union optimization only matters for low-level languages.

---

## 3. The State

The State owns three stacks and provides the interface that operators use.

### Data Stack

- Fixed-size array of Values (10240 entries in the Go implementation).
- Explicit integer stack pointer (SP). SP points to the next free slot.
- `push`: store at `stk[SP]`, increment SP.
- `pop`: decrement SP, read from `stk[SP]`, clear the slot.
- Clearing popped slots matters for garbage-collected languages. The GC must
  see that the pointer is gone.

### Entity Stack

- Fixed-size array of Entity references.
- Entities are typed dictionaries. Each has a name and a set of named
  attributes with typed values.
- `lookup` searches the entity stack from top (SP-1) to bottom (0). The
  first entity that contains the attribute wins.
- Entity names can be qualified: `job.results` means "find the entity named
  `job`, get its `results` attribute." Unqualified names search all entities.

### Control Stack

- Stores stack frames for decision table calls and local variables.
- Frame management tracks the data stack depth at call time, so the callee
  can clean up after itself.

### State Interface

Operators access the state through an abstract interface. The essential
methods are:

```
DataPush(object)           -- push an Object onto the data stack
DataPop() -> object        -- pop an Object from the data stack
DataPeek() -> object       -- read top without popping
DataStackDepth() -> int    -- current depth

EntityPush(entity)         -- push an entity context
EntityPop() -> entity      -- pop an entity context
EntityDepth() -> int       -- current depth
Find(name) -> object       -- search entity stack for a name
Def(name, value) -> bool   -- set a name's value in the entity stack

ControlPush(frame)         -- push a control frame
ControlPop() -> frame      -- pop a control frame
```

This abstraction is what lets operators work unchanged across different
runtime implementations. A port must provide this interface.

---

## 4. Bytecode Format

### Opcodes

Single-byte opcodes. Some are followed by a variable-length integer argument.

#### Varint Encoding

Same as Protocol Buffers:
- Each byte: bits 0-6 = 7 bits of data, bit 7 = continuation flag.
- Unsigned. Little-endian byte order within the varint.

```
Value  Encoding
-----  --------
0      0x00
127    0x7F
128    0x80 0x01
300    0xAC 0x02
```

#### Opcode Table

**Stack operations:**

| Code | Name       | Arg?   | Stack effect       |
|------|------------|--------|--------------------|
| 0    | nop        | --     | --                 |
| 1    | push       | --     | (legacy, unused)   |
| 2    | push_int   | varint | -- n               |
| 3    | pop        | --     | a --               |
| 4    | dup        | --     | a -- a a           |
| 5    | swap       | --     | a b -- b a         |
| 6    | rot        | --     | a b c -- b c a     |

**Arithmetic:**

| Code | Name | Stack effect       | Notes                               |
|------|------|--------------------|-------------------------------------|
| 20   | add  | a b -- (a+b)       | int+int=int, otherwise double       |
| 21   | sub  | a b -- (a-b)       | same promotion rules                |
| 22   | mul  | a b -- (a*b)       | same promotion rules                |
| 23   | div  | a b -- (a/b)       | int/int=int (truncated toward zero) |
| 24   | mod  | a b -- (a%b)       | integers only                       |
| 25   | neg  | a -- (-a)          | preserves type                      |
| 26   | abs  | a -- (\|a\|)       | preserves type                      |
| 27   | inc  | a -- (a+1)         | integer result                      |
| 28   | dec  | a -- (a-1)         | integer result                      |

**Comparison** (all produce a boolean):

| Code | Name | Stack effect       |
|------|------|--------------------|
| 30   | eq   | a b -- (a==b)      |
| 31   | ne   | a b -- (a!=b)      |
| 32   | lt   | a b -- (a\<b)      |
| 33   | le   | a b -- (a\<=b)     |
| 34   | gt   | a b -- (a\>b)      |
| 35   | ge   | a b -- (a\>=b)     |

**Boolean:**

| Code | Name | Stack effect       |
|------|------|--------------------|
| 40   | and  | a b -- (a&&b)      |
| 41   | or   | a b -- (a\|\|b)    |
| 42   | not  | a -- (!a)          |
| 43   | xor  | a b -- (a!=b)      |

**Entity operations:**

| Code | Name    | Stack effect       | Notes                                  |
|------|---------|--------------------|-----------------------------------------|
| 62   | def     | value name --      | set name's value in entity stack        |
| 63   | lookup  | name -- result     | find name; auto-execute if executable   |

**Constants:**

| Code | Name        | Stack effect |
|------|-------------|-------------|
| 100  | push_true   | -- true     |
| 101  | push_false  | -- false    |
| 102  | push_null   | -- null     |
| 103  | push_zero   | -- 0        |
| 104  | push_one    | -- 1        |

**Extended (with varint index):**

| Code | Name     | Arg?   | Stack effect        | Notes                    |
|------|----------|--------|---------------------|--------------------------|
| 200  | operator | varint | (varies)            | call operator by index   |
| 201  | constant | varint | -- value            | push from constant pool  |
| 202  | name     | varint | -- name             | push from name pool      |

### Bytecode Chunk

A bytecode chunk is a self-contained unit of execution. It carries:

1. **Code**: byte array of opcodes and varint arguments.
2. **Constant pool**: array of Values, indexed by the `constant` opcode.
3. **Name pool**: array of RName references, indexed by the `name` opcode.

---

## 5. Semantic Rules

These rules must hold in any correct implementation. The CHIP decision table
test cases verify them.

### Arithmetic Promotion

- `integer op integer = integer` (for add, sub, mul)
- `integer op double = double`
- `double op anything = double`
- Exception: `integer / integer = integer`, truncated toward zero.
  `17 / 5 = 3`, not `3.4`. This matches PostScript and Java.
- Division by zero is an error, not infinity.
- Mod requires integer operands.

### Lookup Semantics

1. Pop a Name from the data stack.
2. If the name is qualified (`entity.attribute`), find the entity by name,
   then get the attribute.
3. If the name is unqualified, search the entity stack from top to bottom.
   The first entity that contains the attribute wins.
4. If the value is executable (a procedure or decision table), **execute it
   immediately**. Do not push it.
5. If the value is not executable, push it onto the data stack.
6. If not found, push null.

### Def Semantics

1. Pop a Name from the data stack.
2. Pop a Value from the data stack.
3. Find the entity that contains the attribute (same search as lookup).
4. Set the attribute's value.

### Operator Dispatch

1. Read the operator index (varint after the opcode).
2. Look up the operator function in the operator table.
3. Call the operator. It manipulates the stacks through the State interface.
4. Operators are the extensibility mechanism. The 181 built-in operators
   cover math, strings, arrays, control flow, entity manipulation, dates,
   and more. See the Go `operators/` package for the complete list.

---

## 6. Implementation Phases

Build your port in this order. Each phase is independently testable.

### Phase 1: Data Stack and Primitives

Implement the Value type and data stack. Implement the push/pop/dup/swap/rot
opcodes and the constant pushes (push_true, push_false, push_null, push_zero,
push_one, push_int).

**Test**: push values, pop them, verify.

### Phase 2: Arithmetic and Comparison

Implement add, sub, mul, div, mod, neg, abs, inc, dec. Implement eq, ne, lt,
le, gt, ge. Implement and, or, not, xor.

Pay attention to:
- Integer division truncates toward zero.
- Type promotion (int+double=double).
- Division by zero is an error.
- Comparisons produce booleans.

**Test**: the arithmetic and comparison test vectors in `test/vectors/`.

### Phase 3: Entity Stack and Lookup

Implement the entity stack, the Entity type (typed dictionary), and the
lookup/def opcodes.

This is the hardest part because it involves:
- Qualified vs unqualified name resolution.
- Auto-execution of executable objects.
- The interaction between lookup and decision table invocation.

**Test**: create entities with attributes, push them, look up values.

### Phase 4: Operator Table

Implement the operator dispatch opcode and enough operators to run CHIP.
The CHIP decision tables use approximately 30-40 distinct operators. You
don't need all 181 on day one.

Essential operators for CHIP:
- Arithmetic: `+`, `-`, `*`, `/`, `negate`, `abs`
- Comparison: `<`, `<=`, `>`, `>=`, `=`, `!=`
- Boolean: `and`, `or`, `not`
- Stack: `dup`, `pop`, `swap`, `exch`, `clear`
- String: `concat`, `s=` (string equals)
- Entity: `forall`, `member`, `addto`, `length`, `copy`
- Control: `if`, `ifelse`, `executetable`
- Conversion: `cvs` (convert to string), `cvi` (convert to integer)

**Test**: load the CHIP rule set, run all 13 test cases, compare results
against the Java or Go reference output.

### Phase 5: Optimization (Optional)

Once correctness is established:
- Add assembly or intrinsics for the hot path (arithmetic, comparison,
  boolean, stack ops on integer Values).
- The pattern that works: try the fast path, return a status code, let the
  caller fall back to the generic path on type mismatch.
- Profile on CHIP first. The bottleneck is usually operator dispatch and
  entity lookup, not arithmetic.

---

## 7. The Fast-Path Pattern

This section describes how the Go implementation uses assembly. The same
pattern applies to any language where you want a native-code fast path.

Each hot-path operation is a leaf function:

```
function fast_add(state: *State) -> status_code:
    if state.dataSP < 2: return UNDERFLOW
    tos = &state.dataStk[state.dataSP - 1]
    nos = &state.dataStk[state.dataSP - 2]
    if tos.tag != INTEGER: return TYPE_MISMATCH
    if nos.tag != INTEGER: return TYPE_MISMATCH
    nos.num += tos.num
    clear(tos)
    state.dataSP--
    return OK
```

The bytecode loop calls it like this:

```
case OP_ADD:
    status = fast_add(state)
    if status == OK: break
    if status == UNDERFLOW: return error
    if status == TYPE_MISMATCH:
        // fall back to generic add (handles doubles, mixed types)
        b = state.dataStk[state.dataSP - 1]
        a = state.dataStk[state.dataSP - 2]
        state.dataSP--
        state.dataStk[state.dataSP - 1] = generic_add(a, b)
```

Status codes:
- 0 = success
- 1 = stack underflow
- 2 = stack overflow
- 3 = type mismatch (fall back to generic)
- 4 = division by zero

The fast path handles the common case (integer-integer). The generic path
handles everything else. In practice, decision table conditions and actions
are dominated by integer arithmetic and boolean logic, so the fast path
hits most of the time.

### What to Optimize

| Worth optimizing           | Not worth optimizing              |
|----------------------------|-----------------------------------|
| add, sub, mul, div, mod    | push_true, push_false (trivial)   |
| eq, ne, lt, le, gt, ge    | constant, name (pool lookup)      |
| and, or, not, xor         | lookup, def (entity search)       |
| pop, dup, swap, rot        | operator (complex dispatch)       |
| neg, inc, dec, abs         | string/array ops (heap alloc)     |

The left column is 24 operations. In the Go implementation, these are in
Plan 9 assembly. Everything else stays in the host language.

---

## 8. Testing Against the Reference

The CHIP sample project (`sampleprojects/CHIP/`) is the definitive test
suite. It contains:

- Entity definitions (`CHIP_edd.xml`)
- Decision tables (`CHIP_dt.xml`)
- Data mappings (`CHIP_map.xml`)
- 13 test cases with expected results

A correct implementation must:

1. Load the EDD, DT, and MAP files.
2. For each test case, create a session, load test data, execute the
   `Compute_Eligibility` decision table.
3. Produce the same entity attribute values as the Java reference.

Test vectors for individual operations are in `test/vectors/`:
- `arithmetic.json` -- add, sub, mul, div, mod, neg, abs
- (more as they are added)

---

## 9. Reference Implementation Files

The Go implementation is the reference for this document:

```
go/pkg/dtrules/bytecode.go          -- opcode definitions, BytecodeChunk
go/pkg/dtrules/value.go             -- Value tagged union
go/pkg/dtrules/object.go            -- Object, Entity interfaces
go/pkg/dtrules/interpreter/state.go -- State: three stacks, fixed arrays
go/pkg/dtrules/interpreter/vm.go    -- bytecode loop + assembly integration
go/pkg/dtrules/interpreter/asm_amd64.s   -- Plan 9 assembly (24 ops)
go/pkg/dtrules/interpreter/asm_stubs.go  -- Go declarations for assembly
go/pkg/dtrules/operators/           -- 181 operators
go/pkg/dtrules/compiler/            -- Expression Language -> bytecode
```

The Java implementation is the original reference:

```
dtrules/src/main/java/com/dtrules/interpreter/
    DTState.java             -- State with three stacks
    RInteger.java            -- Integer type
    RDouble.java             -- Double type
    RBoolean.java            -- Boolean type
    RString.java             -- String type
    RName.java               -- Name type (with entity.attribute support)
    RArray.java              -- Array/procedure type
```

These are GitHub issues for the DTRules project. Each must be a COMPLETE, SELF-CONTAINED prompt.

Project root: /home/paul/go/src/github.com/DTRules/DTRules
Branch: issue-31-asm-optimization

DTRules x86-64-ASM runtime uses NASM assembled into a shared library, accessed via CGO. The NASM source is in `asm/src/vm/bytecode.asm` with library init in `asm/src/lib/init.asm`. The Go bridge is `go/pkg/dtrules/asmruntime/bridge.go` and executor is `go/pkg/dtrules/asmruntime/executor.go`.

Key structures (must match Go side):
- Value: 24 bytes - tag(uint8, offset 0) + 7pad + num(int64, offset 8) + ptr(pointer, offset 16)
- Type tags: VTagNull(0), VTagInteger(1), VTagDouble(2), VTagBoolean(3), VTagString(4), VTagName(5), VTagArray(6), VTagEntity(7), VTagObject(8)
- DTState has dataStk[10240]Value at offset 0, dataSP follows

Opcodes (from go/pkg/dtrules/bytecode.go):
- Stack: OpNop(0), OpPush(1), OpPushInt(2), OpPop(3), OpDup(4), OpSwap(5), OpRot(6), OpRoll(7), OpIndex(8), OpClear(9), OpMark(10)
- Arithmetic: OpAdd(20)-OpDec(28)
- Comparison: OpEq(30)-OpGe(35)
- Boolean: OpAnd(40)-OpXor(43)
- Control: OpExec(50)-OpCall(59)
- Entity: OpEntityPush(60)-OpNewEntity(64)
- Array: OpNewArray(70)-OpPut(74)
- Table: OpNewTable(80)-OpTablePut(82)
- String: OpConcat(90), OpSubstring(91)
- Constants: OpPushTrue(100)-OpPushOne(104)
- Extended: OpOperator(200), OpConstant(201), OpName(202)

CGO bridge functions in bridge.go provide: Init, Reset, ExecuteBytecode, PushValue, PopValue, StackDepth, EntityStackPush/Pop, HeapAlloc, CreateString, EntityAlloc, MarshalEntity, MarshalValue, TableAlloc, CNode/ANode operations.

NASM bytecode.asm already has 79+ opcodes partially implemented but many have issues with state sync and type handling.

Test vectors: test/vectors/*.json (174 tests across 10 files)

---

## C1: x86-64-ASM arithmetic opcodes

Title: x86-64-ASM: Complete arithmetic opcodes for all type combinations
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read arithmetic section for exact type rules, promotion, error conditions for opcodes 20-28
- [ ] DTState memory offsets documented (produced by A2) -- dataStk offset, dataSP offset, Value layout (tag@0, num@8, ptr@16, size=24). Verify these match the NASM struct definitions in `bytecode.asm`
- [ ] The zero-overhead dispatch loop exists in `asm/src/vm/bytecode.asm` (produced by A4) -- arithmetic handlers must be jump targets. Read the NASM dispatch loop to understand the jump table and register conventions.
- [ ] State sync is eliminated (produced by A3) -- NASM now reads/writes DTState memory directly via pointer, not through copied stacks. Verify the CGO bridge passes the DTState pointer.
- [ ] NativeASM arithmetic (B1) is ideally complete -- use it as reference implementation. Read `go/pkg/dtrules/interpreter/vm_amd64.s` arithmetic handlers.
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: add(20), sub(21), mul(22), div(23), mod(24), neg(25), abs(26), inc(27), dec(28)

Current state: NASM `bytecode.asm` has arithmetic ops that work for integers. Mixed int/double was recently improved but still has issues due to state sync architecture (which A3/A4 fixes).

Task:
1. Read bytecode spec from A1 at `docs/bytecode-spec.md`
2. Read current NASM implementation in `asm/src/vm/bytecode.asm` (search for add/sub/mul etc handlers)
3. After A4 establishes the dispatch loop pattern, implement/fix all arithmetic handlers:
   - int op int -> int (IMUL, IDIV, etc.)
   - double op double -> double (SSE2: addsd, subsd, mulsd, divsd)
   - int op double, double op int -> double (cvtsi2sd to promote, then SSE2)
   - mod: integer only
   - neg: NEG for int, xorpd sign bit for double
   - abs: conditional negate
   - inc/dec: add/sub 1
4. Read directly from DTState memory (after A3 removes sync)
5. Verify against NativeASM behavior (should be identical per spec)

Files to modify: `asm/src/vm/bytecode.asm`
Files to read: `docs/bytecode-spec.md`, `go/pkg/dtrules/interpreter/vm.go`, `test/vectors/arithmetic.json` (28 tests)
Reference: NativeASM implementation in `go/pkg/dtrules/interpreter/vm_amd64.s` (Track B)

ASM test harness: `asm/test/unit/test_arithmetic.asm`

Acceptance criteria:
- All 28 arithmetic test vectors pass
- All type combinations work without state sync
- `asm/test/unit/test_arithmetic.asm` passes
- Results match NativeASM and Go runtimes exactly

---

## C2: x86-64-ASM comparison opcodes

Title: x86-64-ASM: Complete comparison opcodes for all type combinations
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read comparison section for cross-type rules and null handling
- [ ] DTState memory offsets documented (produced by A2) -- Value layout for type tag checking
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] State sync eliminated (produced by A3) -- direct DTState pointer access
- [ ] NativeASM comparison (B2) ideally complete -- reference implementation
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: eq(30), ne(31), lt(32), le(33), gt(34), ge(35)

Task: Same pattern as C1 but for comparisons.
- int cmp int: CMP + SETcc
- double cmp double: UCOMISD + SETcc (handle NaN)
- int cmp double: cvtsi2sd + UCOMISD
- string eq string: CGO callback to Go string comparison
- boolean eq boolean: simple byte compare
- null handling per spec

Files: `asm/src/vm/bytecode.asm`, `asm/test/unit/test_comparison.asm`
Test vectors: `test/vectors/comparison.json` (24 tests)

Acceptance criteria: All 24 tests pass, matches Go/NativeASM.

---

## C3: x86-64-ASM boolean opcodes

Title: x86-64-ASM: Complete boolean opcodes
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- boolean type checking rules
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM boolean (B3) ideally complete -- reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: and(40), or(41), not(42), xor(43)

Task: Verify/implement boolean operations on VTagBoolean values.
- Check operand types are boolean
- AND/OR/XOR on num field (0 or 1)
- NOT: flip num field
- Push result with VTagBoolean

Files: `asm/src/vm/bytecode.asm`
Test vectors: `test/vectors/boolean.json` (17 tests)

Acceptance criteria: All 17 tests pass.

---

## C4: x86-64-ASM stack manipulation opcodes

Title: x86-64-ASM: Complete stack manipulation opcodes
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- stack op semantics, especially OpRoll/OpIndex/OpMark
- [ ] DTState memory offsets documented (produced by A2) -- raw 24-byte Value copies need base offset and size
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM stack ops (B4) ideally complete -- reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: pop(3), dup(4), swap(5), rot(6), roll(7), index(8), clear(9), mark(10)

Task: These are pure memory operations on the data stack array. No type checking needed (operate on raw 24-byte Value slots).
- pop: decrement SP
- dup: copy 24 bytes at [SP-1] to [SP], increment SP
- swap: exchange 24 bytes at [SP-1] and [SP-2]
- rot: rotate top 3 values
- roll: pop count, rotate N values
- index: pop index, copy N-th value to top
- clear: set SP to 0
- mark: push special mark value

Files: `asm/src/vm/bytecode.asm`
Test vectors: `test/vectors/stack.json` (may be 0 tests -- create if needed)

Acceptance criteria: All stack ops work, bounds checking for underflow/overflow.

---

## C5: x86-64-ASM string opcodes

Title: x86-64-ASM: Implement string opcodes via CGO callbacks
Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- string opcode behavior
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4) -- understand CGO callback pattern for NASM
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- string concat creates new strings; follow the allocation strategy
- [ ] State sync eliminated (produced by A3) -- bridge functions operate on DTState directly
- [ ] NativeASM string ops (B5) ideally complete -- reference for Go helper functions in `asm_helpers.go`; the CGO bridge may wrap similar helpers
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpConcat(90), OpSubstring(91), plus operator-table string ops via OpOperator(200)

Task: Strings are Go-managed. NASM cannot manipulate string contents directly. Use CGO callbacks.
1. Define C-callable functions in bridge.go for string operations
2. In bytecode.asm, when dispatch hits OpConcat/OpSubstring, call into C bridge
3. Bridge functions operate on DTState directly (after A3)
4. Operator-table string ops already go through OpOperator -> bridge

Files to modify: `asm/src/vm/bytecode.asm`, `go/pkg/dtrules/asmruntime/bridge.go`
Files to read: `go/pkg/dtrules/operators/string.go`, `test/vectors/string.json` (33 tests)

Acceptance criteria: All 33 string test vectors pass.

---

## C6: x86-64-ASM array opcodes

Title: x86-64-ASM: Implement array opcodes via CGO callbacks
Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- array opcode behavior
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- array creation allocates Go slices
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM array ops (B6) ideally complete -- reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpNewArray(70), OpAddTo(71), OpLength(72), OpGet(73), OpPut(74)

Task: Arrays are Go slices. Use CGO callbacks for all array operations.
Same pattern as C5: dispatch recognizes opcode, calls bridge function, bridge operates on DTState.

Files: `asm/src/vm/bytecode.asm`, `bridge.go`
Reference: `go/pkg/dtrules/operators/array.go`, `test/vectors/array.json` (19 tests)

Acceptance criteria: All 19 array test vectors pass.

---

## C7: x86-64-ASM table opcodes

Title: x86-64-ASM: Implement table opcodes via CGO callbacks
Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- table opcode behavior
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- table creation allocates Go maps
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM table ops (B7) ideally complete -- reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpNewTable(80), OpTableGet(81), OpTablePut(82)

Task: Tables are Go maps. CGO callbacks for all operations.

Files: `asm/src/vm/bytecode.asm`, `bridge.go`
Reference: `go/pkg/dtrules/operators/table.go`, `test/vectors/table.json` (14 tests)

Acceptance criteria: All 14 table test vectors pass.

---

## C8: x86-64-ASM control flow opcodes

Title: x86-64-ASM: Implement control flow opcodes
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- control flow section, especially OpJump offset encoding and OpExec recursive semantics
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4) -- OpJump/OpJumpIf/OpReturn modify the bytecode PC within the loop
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM control flow (B8) ideally complete -- reference for Go helper patterns
- [ ] D5 (OpExec) ideally complete -- or coordinate, since both address recursive execution
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpExec(50), OpIf(51), OpIfElse(52), OpWhile(53), OpFor(54), OpForAll(55), OpReturn(56), OpJump(57), OpJumpIf(58), OpCall(59)

Task:
- OpJump, OpJumpIf, OpReturn: pure NASM (modify instruction pointer within bytecode)
- OpIf, OpIfElse: check boolean on stack, branch to correct bytecode section
- OpWhile, OpFor, OpForAll, OpExec: CGO callbacks (these involve recursive execution)

Files: `asm/src/vm/bytecode.asm`, `bridge.go`
Reference: `go/pkg/dtrules/interpreter/vm.go`, `test/vectors/control.json` (4 tests)

Acceptance criteria: All control tests pass, CHIP decision tables execute correctly.

---

## C9: x86-64-ASM entity opcodes

Title: x86-64-ASM: Implement entity opcodes via CGO callbacks
Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- entity opcode behavior
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4)
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- entity creation allocates Go objects
- [ ] State sync eliminated (produced by A3) -- bridge.go's EntityStackPush/Pop/MarshalEntity should be refactored to operate on runtime-owned state; verify this is done
- [ ] NativeASM entity ops (B9) ideally complete -- reference for Go helper functions
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpEntityPush(60), OpEntityPop(61), OpDef(62), OpLookup(63), OpNewEntity(64)

Task: All entity operations require Go interaction (entity factory, entity stack, name resolution). Use CGO callbacks for everything.

Current bridge.go has: EntityStackPush, EntityStackPop, EntityAlloc, EntitySetAttr, EntityGetAttr, MarshalEntity. After A3, these should operate on runtime-owned state directly instead of marshaling.

Files: `asm/src/vm/bytecode.asm`, `bridge.go`
Reference: `go/pkg/dtrules/operators/entity.go`, `test/vectors/entity.json` (11 tests)

Acceptance criteria: All 11 entity test vectors pass, CHIP entity operations work.

---

## C10: x86-64-ASM datetime opcodes

Title: x86-64-ASM: Implement datetime opcodes via CGO callbacks
Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- check if datetime has dedicated opcodes or only operator-table dispatch
- [ ] Zero-overhead dispatch loop in `asm/src/vm/bytecode.asm` (produced by A4) -- verify OpOperator(200) CGO callback is wired
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- datetime objects may need allocation
- [ ] State sync eliminated (produced by A3)
- [ ] NativeASM datetime (B10) ideally complete -- reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Task: DateTime operations go through OpOperator(200) dispatch to the operator table. Verify that the CGO callback chain (OpOperator -> bridge -> Go operator) works for datetime operations.

Files to read: `go/pkg/dtrules/operators/datetime.go`, `test/vectors/datetime.json` (24 tests)

Acceptance criteria: All 24 datetime test vectors pass through x86-64-ASM runtime.

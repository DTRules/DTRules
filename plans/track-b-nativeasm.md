# Track B: NativeASM Opcode Implementation

These are GitHub issues for the DTRules project. Each issue must be a COMPLETE, SELF-CONTAINED prompt that a Claude Code session can execute without prior context.

Project root: /home/paul/go/src/github.com/DTRules/DTRules
Branch: issue-31-asm-optimization

DTRules is a Decision Table Rules Engine with a PostScript-style stack machine. NativeASM is the Plan 9 assembly runtime that operates directly on DTState's fixed-size arrays.

Key structures:
- DTState in `go/pkg/dtrules/interpreter/state.go`: has dataStk[10240]Value, dataSP, entityStk[10240]Entity, entitySP, ctrlStk[10240]Object, ctrlSP
- Value struct (24 bytes): tag(uint8) + 7pad + num(int64) + ptr(unsafe.Pointer)
- Type tags: VTagNull(0), VTagInteger(1), VTagDouble(2), VTagBoolean(3), VTagString(4), VTagName(5), VTagArray(6), VTagEntity(7), VTagObject(8)

ASM stubs declared in `go/pkg/dtrules/interpreter/asm_stubs.go`, implemented in `go/pkg/dtrules/interpreter/vm_amd64.s` and `go/pkg/dtrules/interpreter/stack_amd64.s`.

Go helpers for complex ops in `go/pkg/dtrules/interpreter/asm_helpers.go`: goOpLookup, goOpDef, goOpOperator, goOpExec.

Return codes: 0=success, 1=underflow, 2=overflow, 3=type-mismatch-fallback, 4=div-by-zero, 5=unknown-opcode.

Reference Go implementation: `go/pkg/dtrules/interpreter/vm.go` (dispatch switch).
Test vectors: `test/vectors/*.json`

---

## B1: NativeASM arithmetic opcodes

Title: NativeASM: Complete arithmetic opcodes for all type combinations

Depends on: A4 (dispatch loop)

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read the arithmetic section for exact type rules, promotion behavior, and error conditions for OpAdd(20) through OpDec(28)
- [ ] DTState memory offsets are documented (produced by A2) -- you need dataStk offset, dataSP offset, and Value layout (tag@0, num@8, ptr@16) to read/write stack values in ASM
- [ ] The zero-overhead dispatch loop exists in `go/pkg/dtrules/interpreter/vm_amd64.s` (produced by A4) -- arithmetic handlers must be jump targets in this loop, not standalone functions. Read the dispatch loop to understand the jump table pattern and register conventions before implementing handlers.
- [ ] `go test ./go/pkg/dtrules/...` passes -- baseline before your changes
- [ ] `docs/asm-memory-strategy.md` is NOT required for this issue (arithmetic doesn't allocate)

Opcodes: OpAdd(20), OpSub(21), OpMul(22), OpDiv(23), OpMod(24), OpNeg(25), OpAbs(26), OpInc(27), OpDec(28)

Current state: These exist in `vm_amd64.s` but only handle integer+integer. Return code 3 (type mismatch) falls back to Go for mixed types.

Task:
1. Read the Go reference implementation in `go/pkg/dtrules/interpreter/vm.go` for each arithmetic opcode to understand the exact behavior
2. Read the bytecode spec (from issue A1) at `docs/bytecode-spec.md`
3. Implement in `go/pkg/dtrules/interpreter/vm_amd64.s`:
   - int + int -> int (already works)
   - double + double -> double (SSE2 ADDSD/SUBSD/MULSD/DIVSD)
   - int + double -> double (promote int to double, then SSE2)
   - double + int -> double (same)
   - OpMod: integer only (return error for double)
   - OpNeg: negate int (NEG) or double (XOR sign bit)
   - OpAbs: absolute value int or double
   - OpInc/OpDec: add/subtract 1 to top of stack (int or double)
4. Handle errors: underflow (dataSP < required operands), div-by-zero for OpDiv and OpMod

Files to modify:
- `go/pkg/dtrules/interpreter/vm_amd64.s` - implement handlers
- `go/pkg/dtrules/interpreter/asm_stubs.go` - verify stub signatures match

Files to read (reference):
- `go/pkg/dtrules/interpreter/vm.go` - Go behavior to match
- `go/pkg/dtrules/operators/arithmetic.go` - operator implementations
- `test/vectors/arithmetic.json` - 28 test vectors to validate against

Acceptance criteria:
- All 28 arithmetic test vectors pass when run through NativeASM runtime
- No fallback to Go (RC=3) for any supported type combination
- `go test ./go/pkg/dtrules/interpreter/... -run TestASM` passes
- `go test ./go/pkg/dtrules/...` passes (no regressions)

---

## B2: NativeASM comparison opcodes

Title: NativeASM: Complete comparison opcodes for all type combinations

Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read the comparison section for type rules per opcode, especially cross-type comparison (int vs double) and null handling
- [ ] DTState memory offsets documented (produced by A2) -- Value layout needed for type tag checking and SSE2 double access
- [ ] Zero-overhead dispatch loop exists in `vm_amd64.s` (produced by A4) -- comparison handlers are jump targets in this loop
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpEq(30), OpNe(31), OpLt(32), OpLe(33), OpGt(34), OpGe(35)

Current state: Exist in `vm_amd64.s` for integer comparisons. Double and mixed-type comparisons fall back to Go.

Task:
1. Read Go reference in `vm.go` for comparison behavior
2. Implement all type combinations:
   - int cmp int -> boolean (CMP instruction, set result)
   - double cmp double -> boolean (UCOMISD instruction)
   - int cmp double -> boolean (promote int, UCOMISD)
   - double cmp int -> boolean (promote int, UCOMISD)
   - string eq string -> boolean (call Go helper for string comparison)
   - boolean eq boolean -> boolean
   - null eq null -> true; null eq anything -> false
3. Push result as Value with VTagBoolean(3), num=0 or num=1

Files to modify:
- `go/pkg/dtrules/interpreter/vm_amd64.s`

Files to read:
- `go/pkg/dtrules/interpreter/vm.go`
- `go/pkg/dtrules/operators/comparison.go`
- `test/vectors/comparison.json` - 24 test vectors

Acceptance criteria:
- All 24 comparison test vectors pass
- No fallback to Go for numeric comparisons
- String comparison calls Go helper (strings are Go-managed)
- `go test ./go/pkg/dtrules/...` passes

---

## B3: NativeASM boolean opcodes

Title: NativeASM: Complete boolean opcodes

Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read boolean section for type checking rules (what happens with non-boolean operands?)
- [ ] DTState memory offsets documented (produced by A2)
- [ ] Zero-overhead dispatch loop exists in `vm_amd64.s` (produced by A4)
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpAnd(40), OpOr(41), OpNot(42), OpXor(43)

Current state: Exist in `vm_amd64.s`. May already be complete since booleans are simple (VTagBoolean with num=0 or 1).

Task:
1. Read Go reference in `vm.go`
2. Verify/implement:
   - OpAnd: pop two booleans, push AND result
   - OpOr: pop two booleans, push OR result
   - OpNot: pop one boolean, push negation
   - OpXor: pop two booleans, push XOR result
3. Type checking: all operands must be VTagBoolean(3), otherwise error
4. Handle: what if operand is not boolean? Check spec from A1.

Files to modify: `go/pkg/dtrules/interpreter/vm_amd64.s`
Files to read: `go/pkg/dtrules/interpreter/vm.go`, `go/pkg/dtrules/operators/boolean.go`, `test/vectors/boolean.json` (17 tests)

Acceptance criteria:
- All 17 boolean test vectors pass
- Type errors for non-boolean operands
- `go test ./go/pkg/dtrules/...` passes

---

## B4: NativeASM stack manipulation opcodes

Title: NativeASM: Complete stack manipulation opcodes

Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read stack section for OpRoll/OpIndex semantics and mark value specification
- [ ] DTState memory offsets documented (produced by A2) -- stack ops are raw 24-byte copies; you need dataStk base offset, dataSP offset, and Value size (24 bytes)
- [ ] Zero-overhead dispatch loop exists in `vm_amd64.s` (produced by A4)
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpPop(3), OpDup(4), OpSwap(5), OpRot(6), OpRoll(7), OpIndex(8), OpClear(9), OpMark(10)

Current state: OpPop, OpDup, OpSwap, OpRot exist in `stack_amd64.s`. OpRoll, OpIndex, OpClear, OpMark may be missing.

Task:
1. Read Go reference for each stack opcode
2. Verify existing: pop (decrement SP), dup (copy top), swap (exchange top two), rot (rotate top three)
3. Implement missing:
   - OpRoll(7): roll n-th item to top (pop count from stack, rotate)
   - OpIndex(8): copy n-th item to top without removing (pop index, copy)
   - OpClear(9): reset dataSP to 0
   - OpMark(10): push a mark value onto stack (used for counttomark operations)
4. Bounds checking: underflow for all, overflow for dup/index/mark

Files to modify: `go/pkg/dtrules/interpreter/stack_amd64.s`
Files to read: `go/pkg/dtrules/interpreter/vm.go`, `go/pkg/dtrules/operators/stack.go`, `test/vectors/stack.json` (note: currently 0 tests -- may need to create test vectors)

Acceptance criteria:
- All stack opcodes work correctly
- Proper underflow/overflow detection
- If stack.json has 0 tests, create test vectors in `test/vectors/stack.json`
- `go test ./go/pkg/dtrules/...` passes

---

## B5: NativeASM string opcodes

Title: NativeASM: Implement string opcodes via Go callbacks

Depends on: A4, A5 (memory management)

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read string section for OpConcat and OpSubstring behavior
- [ ] DTState memory offsets documented (produced by A2)
- [ ] Zero-overhead dispatch loop exists in `vm_amd64.s` (produced by A4) -- understand the Go callback pattern (how goOpLookup is called from ASM) since string ops use the same pattern
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- string concatenation creates new strings; the strategy document tells you whether to allocate via Go heap callback or arena. Follow it.
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpConcat(90), OpSubstring(91). Additional string operations may exist in the operator table (accessed via OpOperator(200)).

Current state: Not implemented in NativeASM assembly. Strings are Go-managed (string header = pointer + length), so ASM cannot manipulate string contents directly.

Task:
1. Read Go reference for string operations in `go/pkg/dtrules/operators/string.go`
2. Strategy: ASM dispatch recognizes string opcodes and calls Go helper functions that operate directly on DTState
3. Implement Go helpers in `go/pkg/dtrules/interpreter/asm_helpers.go`:
   - `goOpConcat(state *DTState) int` - pop two strings, push concatenated result
   - `goOpSubstring(state *DTState) int` - pop string + indices, push substring
4. Wire ASM dispatch in `vm_amd64.s` to CALL these helpers (like existing goOpLookup pattern)
5. Handle operator-table string operations (concat, substring, trim, indexOf, etc.) -- these go through OpOperator(200) which already calls goOpOperator

Files to modify:
- `go/pkg/dtrules/interpreter/asm_helpers.go` - new Go helpers
- `go/pkg/dtrules/interpreter/vm_amd64.s` - dispatch for OpConcat, OpSubstring

Files to read:
- `go/pkg/dtrules/operators/string.go` - reference implementations
- `test/vectors/string.json` - 33 test vectors

Acceptance criteria:
- OpConcat and OpSubstring work in NativeASM
- String operator-table ops work through existing goOpOperator path
- All 33 string test vectors pass
- `go test ./go/pkg/dtrules/...` passes

---

## B6: NativeASM array opcodes

Title: NativeASM: Implement array opcodes via Go callbacks

Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read array section for OpNewArray through OpPut
- [ ] Zero-overhead dispatch loop in `vm_amd64.s` (produced by A4) -- understand Go callback pattern
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- array creation allocates Go slices; follow the strategy
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpNewArray(70), OpAddTo(71), OpLength(72), OpGet(73), OpPut(74)

Current state: Not implemented in NativeASM. Arrays are Go slices ([]Object), requiring Go heap interaction.

Task:
1. Read Go reference in `go/pkg/dtrules/operators/array.go`
2. Implement Go helpers in `asm_helpers.go`:
   - `goOpNewArray(state *DTState) int` - create empty array, push
   - `goOpAddTo(state *DTState) int` - pop value + array, append, push array
   - `goOpLength(state *DTState) int` - pop array, push length as integer
   - `goOpGet(state *DTState) int` - pop index + array, push element
   - `goOpPut(state *DTState) int` - pop value + index + array, set element, push array
3. Wire ASM dispatch to call helpers

Files to modify:
- `go/pkg/dtrules/interpreter/asm_helpers.go`
- `go/pkg/dtrules/interpreter/vm_amd64.s`

Files to read:
- `go/pkg/dtrules/operators/array.go`
- `test/vectors/array.json` - 19 test vectors

Acceptance criteria:
- All 5 array opcodes work
- All 19 array test vectors pass
- `go test ./go/pkg/dtrules/...` passes

---

## B7: NativeASM table/hashmap opcodes

Title: NativeASM: Implement table opcodes via Go callbacks

Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read table section for OpNewTable through OpTablePut
- [ ] Zero-overhead dispatch loop in `vm_amd64.s` (produced by A4) -- understand Go callback pattern
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- table creation allocates Go maps; follow the strategy
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpNewTable(80), OpTableGet(81), OpTablePut(82)

Current state: Not implemented in NativeASM. Tables are Go maps.

Task:
1. Read Go reference in `go/pkg/dtrules/operators/table.go`
2. Implement Go helpers:
   - `goOpNewTable(state *DTState) int`
   - `goOpTableGet(state *DTState) int`
   - `goOpTablePut(state *DTState) int`
3. Wire ASM dispatch

Files to modify: `go/pkg/dtrules/interpreter/asm_helpers.go`, `go/pkg/dtrules/interpreter/vm_amd64.s`
Files to read: `go/pkg/dtrules/operators/table.go`, `test/vectors/table.json` (14 tests)

Acceptance criteria:
- All 3 table opcodes work
- All 14 table test vectors pass

---

## B8: NativeASM control flow opcodes

Title: NativeASM: Implement control flow opcodes

Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read control flow section for OpIf through OpCall. Pay special attention to OpExec semantics (recursive execution) and OpJump offset encoding
- [ ] Zero-overhead dispatch loop in `vm_amd64.s` (produced by A4) -- OpJump/OpJumpIf/OpReturn modify the bytecode PC within the dispatch loop; understand how the PC is stored and advanced
- [ ] Understand the Go callback pattern from A4 -- OpWhile, OpFor, OpForAll, OpExec need Go helpers
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpExec(50), OpIf(51), OpIfElse(52), OpWhile(53), OpFor(54), OpForAll(55), OpReturn(56), OpJump(57), OpJumpIf(58), OpCall(59)

Current state: OpExec is not implemented (returns error). Jump/call opcodes may be partially in dispatch loop. If/while/for likely missing.

Task:
1. Read Go reference in `vm.go` for control flow
2. Implement in assembly or as Go helpers:
   - OpIf(51): pop boolean + executable; if true, execute. Can be ASM: check boolean, if true adjust PC to inline code
   - OpIfElse(52): pop boolean + two executables; execute one. Similar to OpIf.
   - OpWhile(53): pop body + condition; loop while condition is true. Needs Go helper for complex case.
   - OpFor(54): pop body + limit + start; loop. Go helper.
   - OpForAll(55): pop body + collection; iterate. Go helper.
   - OpExec(50): pop executable, execute it. Go helper (recursive dispatch).
   - OpReturn(56): exit current bytecode chunk. Set PC past end.
   - OpJump(57): unconditional jump. Adjust PC by offset.
   - OpJumpIf(58): conditional jump. Pop boolean, adjust PC if true.
   - OpCall(59): call subroutine. Push return address, jump.
3. Control flow within ASM dispatch: OpJump, OpJumpIf, OpReturn can be pure ASM (modify bytecode PC). OpWhile, OpFor, OpForAll likely need Go helpers.

Files to modify: `go/pkg/dtrules/interpreter/vm_amd64.s`, `go/pkg/dtrules/interpreter/asm_helpers.go`
Files to read: `go/pkg/dtrules/interpreter/vm.go`, `test/vectors/control.json` (4 tests)

Acceptance criteria:
- All control flow opcodes work
- All 4 control flow test vectors pass
- CHIP decision tables execute correctly (they use if/ifelse heavily)
- `go test ./go/pkg/dtrules/...` passes

---

## B9: NativeASM entity opcodes

Title: NativeASM: Implement entity opcodes via Go callbacks

Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read entity section for OpEntityPush through OpNewEntity
- [ ] Zero-overhead dispatch loop in `vm_amd64.s` (produced by A4) -- verify goOpLookup and goOpDef are already wired as Go callbacks. Understand the pattern for adding new Go callbacks (goOpEntityPush, etc.)
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- entity creation allocates Go objects via entity factory; follow the strategy
- [ ] RuntimeInit/RuntimeQuery interfaces exist (produced by A3) -- entity stack is managed by the runtime; understand how entities are stored
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Opcodes: OpEntityPush(60), OpEntityPop(61), OpDef(62), OpLookup(63), OpNewEntity(64)

Current state: goOpLookup and goOpDef exist in `asm_helpers.go` and are called from the dispatch loop. OpEntityPush, OpEntityPop, OpNewEntity may be missing.

Task:
1. Read Go reference in `vm.go` and `go/pkg/dtrules/operators/entity.go`
2. Verify existing helpers: goOpLookup (entity stack name resolution), goOpDef (define on entity stack)
3. Implement missing Go helpers:
   - `goOpEntityPush(state *DTState) int` - pop entity from data stack, push onto entity stack
   - `goOpEntityPop(state *DTState) int` - pop entity from entity stack
   - `goOpNewEntity(state *DTState) int` - pop type name, create entity via factory, push
4. Wire ASM dispatch for all 5 entity opcodes

Files to modify: `go/pkg/dtrules/interpreter/asm_helpers.go`, `go/pkg/dtrules/interpreter/vm_amd64.s`
Files to read: `go/pkg/dtrules/interpreter/vm.go`, `go/pkg/dtrules/operators/entity.go`, `test/vectors/entity.json` (11 tests)

Acceptance criteria:
- All 5 entity opcodes work
- All 11 entity test vectors pass
- CHIP decision tables work (entity operations are core to decision table execution)
- `go test ./go/pkg/dtrules/...` passes

---

## B10: NativeASM datetime opcodes

Title: NativeASM: Implement datetime opcodes via Go callbacks

Depends on: A4, A5

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- check whether datetime has dedicated opcodes or only operator-table dispatch
- [ ] Zero-overhead dispatch loop in `vm_amd64.s` (produced by A4) -- verify goOpOperator callback is wired for OpOperator(200) dispatch
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- datetime objects may need allocation
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Current state: No datetime opcodes in the core opcode set (OpXxx constants). DateTime operations go through the operator table via OpOperator(200), which calls goOpOperator.

Task:
1. Read `go/pkg/dtrules/operators/datetime.go` to understand what datetime operators exist
2. Verify that OpOperator(200) + goOpOperator correctly dispatches datetime operations
3. If any datetime operations need direct opcode support, implement Go helpers
4. Run datetime test vectors to verify

Files to read: `go/pkg/dtrules/operators/datetime.go`, `go/pkg/dtrules/interpreter/asm_helpers.go` (goOpOperator), `test/vectors/datetime.json` (24 tests)

Acceptance criteria:
- All 24 datetime test vectors pass through NativeASM
- DateTime operations work via operator table dispatch
- `go test ./go/pkg/dtrules/...` passes

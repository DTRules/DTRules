These are GitHub issues for the DTRules project. Each issue must be a COMPLETE, SELF-CONTAINED prompt that a Claude Code session can execute without prior context. Include all file paths, code references, struct definitions, and acceptance criteria.

Project root: /home/paul/go/src/github.com/DTRules/DTRules
Branch to work from: issue-31-asm-optimization (or create feature branches from it)
Base branch: 5.0-SNAPSHOT

DTRules is a Decision Table Rules Engine with a PostScript-style stack machine. It has three runtimes: Go (reference), NativeASM (Plan 9 assembly), and x86-64-ASM (NASM/CGO). The ASM runtimes are broken because they maintain separate state that syncs with DTState on every operation.

## A1: Define formal bytecode specification

Title: Define formal bytecode specification for all runtimes

This is the FIRST issue to complete. It blocks everything else.

Context: DTRules has a bytecode format defined in `go/pkg/dtrules/bytecode.go` with these opcodes:
- Stack: OpNop(0), OpPush(1), OpPushInt(2), OpPop(3), OpDup(4), OpSwap(5), OpRot(6), OpRoll(7), OpIndex(8), OpClear(9), OpMark(10)
- Arithmetic: OpAdd(20), OpSub(21), OpMul(22), OpDiv(23), OpMod(24), OpNeg(25), OpAbs(26), OpInc(27), OpDec(28)
- Comparison: OpEq(30), OpNe(31), OpLt(32), OpLe(33), OpGt(34), OpGe(35)
- Boolean: OpAnd(40), OpOr(41), OpNot(42), OpXor(43)
- Control: OpExec(50), OpIf(51), OpIfElse(52), OpWhile(53), OpFor(54), OpForAll(55), OpReturn(56), OpJump(57), OpJumpIf(58), OpCall(59)
- Entity: OpEntityPush(60), OpEntityPop(61), OpDef(62), OpLookup(63), OpNewEntity(64)
- Array: OpNewArray(70), OpAddTo(71), OpLength(72), OpGet(73), OpPut(74)
- Table: OpNewTable(80), OpTableGet(81), OpTablePut(82)
- String: OpConcat(90), OpSubstring(91)
- Constants: OpPushTrue(100), OpPushFalse(101), OpPushNull(102), OpPushZero(103), OpPushOne(104)
- Extended: OpOperator(200), OpConstant(201), OpName(202)

Value types (from value.go): VTagNull(0), VTagInteger(1), VTagDouble(2), VTagBoolean(3), VTagString(4), VTagName(5), VTagArray(6), VTagEntity(7), VTagObject(8)

The Value struct is 24 bytes: tag(uint8) + 7 padding + num(int64) + ptr(unsafe.Pointer)

Task: Create a formal specification document at `docs/bytecode-spec.md` that defines for EVERY opcode:
1. Stack effect diagram: `[inputs] -> [outputs]` with types
2. Type rules: what input types are legal for each operand position
3. Type promotion: what happens with mixed types (e.g., int+double -> double result)
4. Error conditions: underflow, overflow, type mismatch, div-by-zero, null handling
5. Return codes for ASM: 0=success, 1=underflow, 2=overflow, 3=type-fallback, 4=div-zero, 5=unknown-opcode
6. Edge cases: NaN, empty strings, null operands, empty arrays

Reference implementation: Read `go/pkg/dtrules/interpreter/vm.go` (the dispatch switch starting around the ExecuteBytecode method) and `go/pkg/dtrules/operators/` directory to understand current Go behavior. The spec should codify what Go currently does as the canonical behavior.

Also reference test vectors in `test/vectors/*.json` (174 tests across 10 categories). Map each test vector to the opcode(s) it exercises.

Acceptance criteria:
- Every opcode has a complete specification
- Every type combination is specified (what happens when you add int+string? Error or coerce?)
- Test vectors are mapped to opcodes
- Any gaps in test vectors are identified (e.g., stack.json has 0 tests currently)

## A2: Verify DTState fixed-size array layout and document offsets

Title: Document and verify DTState memory layout for ASM access

Depends on: A1

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- contains the formal per-opcode specification including Value struct layout and type tag definitions
- [ ] Read the spec to confirm Value size (expected 24 bytes) and type tag assignments

Context: DTState in `go/pkg/dtrules/interpreter/state.go` uses fixed-size arrays:
```go
const MaxStackDepth = 10240

type DTState struct {
    dataStk   [MaxStackDepth]dtrules.Value  // Value is 24 bytes
    dataSP    int
    entityStk [MaxStackDepth]dtrules.Entity
    entitySP  int
    ctrlStk   [MaxStackDepth]dtrules.Object
    ctrlSP    int
    frames    [MaxStackDepth]int
    frameSP   int
    // ... more fields
}
```

Value struct (from `go/pkg/dtrules/value.go`):
```go
type Value struct {
    tag uint8
    _   [7]byte
    num int64
    ptr unsafe.Pointer
}
```

Task:
1. Write a Go test in `go/pkg/dtrules/interpreter/state_layout_test.go` that uses `unsafe.Offsetof` and `unsafe.Sizeof` to compute and print the byte offset of every field in DTState
2. Document the offsets in a constants file or comment block that ASM can reference
3. Verify that the current ASM code (in `go/pkg/dtrules/interpreter/vm_amd64.s`) uses the correct offsets
4. Add compile-time assertions that these offsets don't change (if a field is inserted, tests fail)
5. Document Value struct layout: tag at offset 0, num at offset 8, ptr at offset 16, total size 24

Key files:
- `go/pkg/dtrules/interpreter/state.go` - DTState struct
- `go/pkg/dtrules/value.go` - Value struct
- `go/pkg/dtrules/interpreter/vm_amd64.s` - existing ASM that references offsets
- `go/pkg/dtrules/interpreter/asm_stubs.go` - Go function declarations for ASM

Acceptance criteria:
- Test file exists that computes all offsets
- All offsets are documented
- ASM constants match Go struct layout
- Tests pass on amd64

## A3: Move VMState into all runtimes

Title: Runtimes own their VMState; provide init/query interfaces

Depends on: A1

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- contains the formal opcode contract, return code semantics, and error behavior that the RuntimeInit/RuntimeQuery interfaces must support
- [ ] Read the spec's section on error conditions and return codes to understand what the runtime interfaces need to expose

Context: Currently the x86-64-ASM runtime in `go/pkg/dtrules/asmruntime/executor.go` copies state from DTState into a separate ASM state before execution, then copies results back after. This sync-on-every-operation pattern is the root cause of ASM being slower than Go.

The fix: Each runtime owns its state entirely. Callers use:
- RuntimeInit interface to push entities and values IN before execution
- RuntimeQuery interface to extract results OUT after execution completes
- No mid-execution syncing

Current interfaces in `go/pkg/dtrules/interpreter/state.go`:
```go
type BytecodeExecutor interface {
    ExecuteBytecode(state *DTState, bc *dtrules.BytecodeChunk) error
    Name() string
}
```

Current RuntimeFactory in `go/pkg/dtrules/session/ruleset.go`:
```go
type RuntimeFactory interface {
    Name() string
    CreateState(session Session) (State, error)
}
```

Task:
1. Define new interfaces (in `go/pkg/dtrules/runtime.go` or appropriate location):
   - `RuntimeInit`: methods to push initial entities, set entity schemas, push initial values before execution
   - `RuntimeQuery`: methods to extract entities by name, get values, read data stack after execution
2. Refactor BytecodeExecutor: the runtime receives state via RuntimeInit, not as a DTState pointer
3. Go runtime: DTState becomes the Go runtime's internal VMState, wrapped behind RuntimeInit/RuntimeQuery
4. NativeASM (`go/pkg/dtrules/runtime/nativeasm/executor.go`): already operates on DTState directly -- refactor to own its state via the new interfaces
5. x86-64-ASM (`go/pkg/dtrules/asmruntime/executor.go`): eliminate syncStateToASM/syncStateFromASM. ASM operates on its own state, results extracted via RuntimeQuery.
6. Delete entity marshaling code in `go/pkg/dtrules/asmruntime/bridge.go` (lines ~440-554)
7. Update `go/pkg/dtrules/session/session.go` to use new interfaces

Key files to modify:
- `go/pkg/dtrules/interpreter/state.go`
- `go/pkg/dtrules/interpreter/vm.go`
- `go/pkg/dtrules/runtime/nativeasm/executor.go`
- `go/pkg/dtrules/asmruntime/executor.go`
- `go/pkg/dtrules/asmruntime/bridge.go`
- `go/pkg/dtrules/session/ruleset.go`
- `go/pkg/dtrules/session/session.go`

Acceptance criteria:
- RuntimeInit and RuntimeQuery interfaces defined
- All three runtimes implement both interfaces
- No state syncing occurs during bytecode execution
- Existing Go tests still pass (`go test ./go/pkg/dtrules/...`)
- CHIP decision tables produce correct results on Go runtime

## A4: Zero-overhead ASM dispatch loop

Title: Implement zero-overhead dispatch loop with no per-opcode stack frames

Depends on: A2, A3

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- contains the complete opcode table with numeric values needed to build the jump table
- [ ] `go/pkg/dtrules/interpreter/state_layout_test.go` exists and passes (produced by A2) -- confirms DTState memory layout and byte offsets for ASM to use
- [ ] A2's documented offsets are available (either in a constants file or the test output) -- needed for ASM to address dataStk, dataSP, entityStk, entitySP at correct offsets
- [ ] RuntimeInit/RuntimeQuery interfaces exist (produced by A3) -- the dispatch loop entry point must conform to the new runtime-owned-state model, not the old DTState-pointer model
- [ ] All three runtimes implement RuntimeInit/RuntimeQuery (A3) -- verify `go test ./go/pkg/dtrules/...` passes after A3's refactor before modifying ASM

Context: The ASM dispatch loop must be a tight jump table. Each opcode handler is a labeled block within the loop, reached by direct jump (not CALL/RET). No stack frame setup per opcode.

Current dispatch loop entry point in `go/pkg/dtrules/interpreter/asm_stubs.go`:
```go
func asmDispatchLoop(state *DTState, code unsafe.Pointer, codeLen int, constPtr unsafe.Pointer, namePtr unsafe.Pointer) int
```

Current implementation in `go/pkg/dtrules/interpreter/vm_amd64.s` uses this pattern.

For x86-64/NASM, the dispatch loop is in `asm/src/vm/bytecode.asm`.

Task:
1. NativeASM (Plan 9): Redesign `vm_amd64.s` dispatch loop:
   - Single entry point, one stack frame for the entire loop
   - Read opcode from bytecode stream
   - Jump to handler via computed jump (jump table indexed by opcode)
   - Handler executes, jumps back to dispatch (not CALL/RET)
   - Only exit the loop on: end of bytecode, error, or opcode that needs Go callback
   - Go callbacks (goOpLookup, goOpDef, goOpOperator) are the ONLY function calls
2. x86-64 NASM: Same pattern in `asm/src/vm/bytecode.asm`
   - One CGO entry, tight dispatch loop, one CGO return
   - Jump table for opcode dispatch
   - Opcode handlers are jump targets, not functions
3. Profile: measure cycles per dispatch (just the jump, not the operation) -- target is single indirect jump

Key files:
- `go/pkg/dtrules/interpreter/vm_amd64.s` - NativeASM dispatch
- `go/pkg/dtrules/interpreter/asm_stubs.go` - entry points
- `go/pkg/dtrules/interpreter/asm_helpers.go` - Go callbacks
- `asm/src/vm/bytecode.asm` - NASM dispatch
- `go/pkg/dtrules/bytecode.go` - opcode constants

Acceptance criteria:
- No CALL/RET per opcode in either dispatch loop
- Jump table indexed by opcode value
- Dispatch loop uses one stack frame total
- Go callbacks minimize register save/restore
- Existing tests still pass

## A5: Define ASM memory management strategy

Title: Define memory management strategy for ASM runtimes

Depends on: A1, A3

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- identifies which opcodes allocate memory (string concat, array creation, entity creation, table creation)
- [ ] RuntimeInit/RuntimeQuery interfaces exist (produced by A3) -- the memory strategy must be compatible with the runtime-owned-state model; understand how the runtime holds references to allocated objects

Context: ASM runtimes need to allocate memory for strings (concatenation results), arrays, entities, and tables. Currently the x86-64 runtime has heap functions in `go/pkg/dtrules/asmruntime/bridge.go`:
```go
func HeapAlloc(size int) unsafe.Pointer
func CreateString(s string) unsafe.Pointer
func EntityAlloc(typeName unsafe.Pointer) unsafe.Pointer
```

And the NASM init (`asm/src/lib/init.asm`) provides memory initialization via mmap.

The question: when ASM needs to create a new string (e.g., concat), where does the memory come from? Options:
1. Go heap via callbacks (safest, GC-managed, but callback overhead)
2. ASM-owned arena (fast, but no GC, must manually free)
3. Pre-allocated pool (fast, bounded, but limits apply)

Task:
1. Analyze what operations need memory allocation (string concat, array creation, entity creation, table creation)
2. For each, determine frequency in typical workloads (read CHIP sample decision tables)
3. Recommend a strategy that balances performance with correctness
4. Document the strategy
5. Define the Go callback interface for allocation (if using Go heap)
6. Consider: can NativeASM (Plan 9) and x86-64 (NASM) share the same strategy?

Key files:
- `go/pkg/dtrules/asmruntime/bridge.go` - existing allocation functions
- `asm/src/lib/init.asm` - NASM memory init
- `go/pkg/dtrules/interpreter/asm_helpers.go` - existing Go helpers
- `sampleprojects/CHIP/` - reference workload

Acceptance criteria:
- Strategy document created at `docs/asm-memory-strategy.md`
- Clear decision on Go heap vs arena vs pool
- Interface defined for allocation callbacks
- Strategy accounts for GC interaction (Go objects referenced from ASM)
- Strategy is implementable in both Plan 9 and NASM

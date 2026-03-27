# Plan: Fix All Runtime Issues (Revised)

## Problem Statement

DTRules is a PostScript-style interpreter with three stacks:
- **Data stack** (operand stack) -- expression evaluation
- **Entity stack** (dictionary stack) -- name resolution context (typed attributes, not unrestricted maps)
- **Control stack** -- stack frames for decision tables and local variables

The Go runtime works correctly. Both ASM runtimes are broken because they
invented a **separate VMState** that duplicates DTState's stacks and constantly
syncs between them. This is wrong. The Java reference implementation has one
state. The Go implementation should have one state.

| Runtime | Status | Core Problem |
|---------|--------|--------------|
| **Go** | Working | Reference implementation |
| **NativeASM** | Broken | Invented VMState; syncs with DTState; falls back to Go |
| **x86-64-ASM** | Broken | Same sync problem + incomplete operators; CGO issues |
| **Java** | Working | Reference implementation |

**Current Benchmark Results (CHIP Decision Tables):**
- Go: 12.2us (fastest working)
- NativeASM: 17.6us (45% slower than Go -- should be faster!)
- x86-64-ASM: 38.2us (errors on most tests)
- Java: 78us (6x slower than Go)

---

## Root Cause Analysis

### The Wrong Approach: VMState

Both ASM runtimes invented separate state structs with their own stacks,
then synced between them and DTState on every operator call:

```
VMState (separate)  <--sync-->  DTState
  Stack copy        <--marshal->  Real Stack

  For operators: sync -> execute -> sync = 3x overhead
  For nested calls: FALL BACK TO GO VM
```

This is unnecessary. Assembly can operate directly on DTState's memory.
There should be ONE state, like Java.

### The Correct Approach: Assembly on DTState

```
DTState owns ALL state (one state, like Java):
  Data Stack [10240]  <-- assembly operates here directly
  Entity Stack [10240]
  Control Stack [10240]
  Stack Pointers (int fields)

Assembly subroutines:
  - Take pointer to DTState's stack memory + SP
  - Do the operation (add, compare, etc.)
  - Update SP
  - Return

No VMState. No syncing. No fallbacks.
```

### Why All Operators Already Work

Architectural review confirmed that **all operators use the `dtrules.State`
interface exclusively**. No operator type-asserts to `*interpreter.DTState`.
This means:

1. Changing DTState's internal stack representation is safe
2. Operators will work unchanged
3. Decision table execution (CNode, ANode) uses State interface only
4. RArray (executable arrays / PostScript procedures) uses State interface only

---

## Implementation Plan

### Phase 1: Make DTState Assembly-Friendly

**Goal:** Change DTState from dynamic slices to fixed-size arrays so assembly
can operate directly on the stack memory at known offsets.

#### 1.1 Change Stack Representation

**File:** `go/pkg/dtrules/interpreter/state.go`

Change from:
```go
type DTState struct {
    ctrlStk   []dtrules.Object   // dynamic slice
    dataStk   []dtrules.Object   // dynamic slice
    entityStk []dtrules.Entity   // dynamic slice
    // ...
}
```

To:
```go
const MaxStackDepth = 10240

type DTState struct {
    // Fixed-size arrays -- assembly can access at known offsets
    dataStk   [MaxStackDepth]dtrules.Object
    dataSP    int

    entityStk [MaxStackDepth]dtrules.Entity
    entitySP  int

    ctrlStk   [MaxStackDepth]dtrules.Object
    ctrlSP    int

    // Frame management
    frames       [MaxStackDepth]int
    frameSP      int
    currentFrame int

    // ... rest unchanged ...
}
```

Update all stack methods to use array + SP instead of slice append/truncate:
```go
func (s *DTState) DataPush(obj dtrules.Object) error {
    if s.dataSP >= MaxStackDepth {
        return dtrules.StackOverflowError("DataPush", "Data Stack overflow")
    }
    s.dataStk[s.dataSP] = obj
    s.dataSP++
    return nil
}

func (s *DTState) DataPop() (dtrules.Object, error) {
    if s.dataSP <= 0 {
        return nil, dtrules.StackUnderflowError("DataPop")
    }
    s.dataSP--
    obj := s.dataStk[s.dataSP]
    s.dataStk[s.dataSP] = nil  // clear for GC
    return obj, nil
}

func (s *DTState) DataStackDepth() int {
    return s.dataSP
}
```

Same pattern for entity stack, control stack, and frames.

#### 1.2 Verify Go Runtime Still Works

Run all tests to confirm the internal change doesn't break anything.
All existing methods still satisfy `dtrules.State`.

**Files to modify:**
- `go/pkg/dtrules/interpreter/state.go` -- stack representation change

---

### Phase 2: NativeASM (Plan 9 Assembly on DTState)

**Goal:** Plan 9 assembly subroutines that operate directly on DTState's
fixed-size arrays for hot-path operations. This is the reference ASM
implementation.

#### 2.1 Define Assembly Entry Points

**File:** `go/pkg/dtrules/interpreter/asm_amd64.s` (new)

Assembly subroutines take a pointer to DTState and operate on its stacks
at known offsets:

```asm
// func asmAdd(state *DTState) int
TEXT .asmAdd(SB), NOSPLIT, $0-16
    MOVQ state+0(FP), R15        // R15 = DTState pointer
    MOVQ dataSP_offset(R15), CX  // CX = current SP
    // pop two Objects, extract numeric values, add, push result
    RET
```

Since DTState has fixed-size arrays, the assembly knows exactly where the
data stack starts and where dataSP is.

#### 2.2 Integrate into ExecuteBytecode

**File:** `go/pkg/dtrules/interpreter/vm.go`

For hot-path opcodes, call assembly instead of Go:

```go
case dtrules.OpAdd:
    if result := asmAdd(s); result != 0 {
        return translateAsmError(result)
    }
```

Single ExecuteBytecode method on DTState. No separate executor, no VMState,
no syncing.

#### 2.3 Which Operations Get Assembly

Hot path (assembly):
- Arithmetic: OpAdd, OpSub, OpMul, OpDiv, OpNeg, OpInc, OpDec, OpMod, OpAbs
- Comparison: OpEq, OpNe, OpLt, OpLe, OpGt, OpGe
- Boolean: OpAnd, OpOr, OpNot, OpXor
- Stack: OpDup, OpSwap, OpRot, OpPop
- Push: OpPushTrue, OpPushFalse, OpPushNull, OpPushZero, OpPushOne, OpPushInt

Everything else stays in Go (entity ops, operator dispatch, name lookup).
These involve Go interfaces and GC-managed objects that assembly shouldn't
touch.

**Files to create/modify:**
- `go/pkg/dtrules/interpreter/asm_amd64.s` -- Plan 9 assembly subroutines
- `go/pkg/dtrules/interpreter/asm_stubs.go` -- Go function declarations
- `go/pkg/dtrules/interpreter/vm.go` -- call assembly for hot-path ops

---

### Phase 3: x86-64-ASM (NASM Translation)

**Goal:** The x86-64-ASM runtime is a translation of the Plan 9 assembly
from Phase 2 into NASM syntax, accessed via CGO. Same operations, same
logic, different assembler.

#### 3.1 Translate Plan 9 to NASM

Each Plan 9 subroutine from Phase 2 gets a NASM equivalent:

```
Plan 9 (asm_amd64.s)          NASM (bytecode.asm)
-----------------------        ----------------------
asmAdd                    -->  op_add
asmSub                    -->  op_sub
asmMul                    -->  op_mul
...                            ...
```

Same stack layout, same offsets, same logic. Just different syntax and
calling convention (CGO instead of Go linkage).

#### 3.2 Fix CGO Bridge

**File:** `go/pkg/dtrules/asmruntime/bridge.go`

The bridge passes pointers to DTState's fixed-size arrays to the NASM code.
No separate state, no syncing:

```go
func (e *Executor) ExecuteBytecode(state *interpreter.DTState, bc *dtrules.BytecodeChunk) error {
    // Pass DTState's stack memory directly to C/NASM
    result := C.execute_bytecode(
        (*C.char)(unsafe.Pointer(&state.DataStk[0])),
        (*C.int)(unsafe.Pointer(&state.DataSP)),
        (*C.char)(unsafe.Pointer(&bc.Code()[0])),
        C.int(len(bc.Code())),
        // ... constants, names ...
    )
    // No syncing -- NASM operated on DTState directly
    return translateError(result)
}
```

#### 3.3 Fix Existing Issues

- Complete missing opcodes (translate from Plan 9 versions)
- Fix type conversion (integer + double -- same logic as Plan 9)
- Fix heap management / OOM issues
- Add null checks and pointer validation

#### 3.4 Entity and Operator Dispatch

For entity ops and operator calls, the NASM code calls back into Go
(via CGO callbacks) which operates on DTState through the State interface.
Same as how the Go bytecode loop handles them -- no special cases.

**Files to modify:**
- `asm/src/vm/bytecode.asm` -- translate Plan 9 ops to NASM
- `go/pkg/dtrules/asmruntime/bridge.go` -- pass DTState pointers, no syncing
- `go/pkg/dtrules/asmruntime/executor.go` -- simplify, remove sync code

---

### Phase 4: Clean Up

**Goal:** Delete the broken VMState architecture and unused code.

#### 4.1 Delete VMState

**Delete:**
- `go/pkg/dtrules/runtime/nativeasm/vm.go` -- VMState struct
- `go/pkg/dtrules/runtime/nativeasm/executor.go` -- sync-based executor
- `go/pkg/dtrules/runtime/nativeasm/vm_amd64.s` -- VMState assembly
- `go/pkg/dtrules/runtime/nativeasm/stack_amd64.s` -- VMState stack ops

#### 4.2 Simplify Runtime Wrappers

The NativeASM runtime.go becomes a thin wrapper or merges into the Go
runtime, since the Plan 9 assembly is now inside DTState itself.

The x86-64-ASM runtime keeps its own executor because it uses CGO,
but it no longer maintains separate state.

#### 4.3 Clean Up BytecodeExecutor Interface

Consider changing `BytecodeExecutor` to accept `dtrules.State` instead
of `*interpreter.DTState`, or remove it if no longer needed.

---

### Phase 5: Testing and Validation

#### 5.1 After Phase 1 (Stack Change)

All existing tests must pass:
```bash
cd go && go test ./pkg/dtrules/... > /tmp/test-phase1.log 2>&1
```

#### 5.2 After Phase 2 (NativeASM)

Assembly and Go paths produce identical results:
```bash
go test -v -run TestCHIP ./pkg/dtrules/ > /tmp/test-phase2.log 2>&1
go test -bench=BenchmarkCHIP ./pkg/dtrules/ -benchtime=5s > /tmp/bench-native.log 2>&1
```

#### 5.3 After Phase 3 (x86-64-ASM)

NASM runtime produces identical results to Go and NativeASM:
```bash
go test -v -run TestCHIPAllRuntimes ./pkg/dtrules/ > /tmp/test-phase3.log 2>&1
go test -bench=BenchmarkCHIP ./pkg/dtrules/ -benchtime=5s > /tmp/bench-all.log 2>&1
```

**Success Criteria:**
1. All tests pass with array-based stacks (Phase 1)
2. NativeASM (Plan 9) produces identical results to Go (Phase 2)
3. x86-64-ASM (NASM) produces identical results to Go and NativeASM (Phase 3)
4. No VMState, no syncing, no fallbacks anywhere
5. All CHIP decision tables produce correct results on all runtimes
6. NativeASM faster than pure Go (target: 2x arithmetic-heavy)
7. x86-64-ASM comparable to or faster than NativeASM

---

## File Changes Summary

| File | Action | Phase | Description |
|------|--------|-------|-------------|
| `go/pkg/dtrules/interpreter/state.go` | MODIFY | 1 | Slices to fixed arrays (10240) + SP fields |
| `go/pkg/dtrules/interpreter/asm_amd64.s` | NEW | 2 | Plan 9 assembly subroutines on DTState |
| `go/pkg/dtrules/interpreter/asm_stubs.go` | NEW | 2 | Go declarations for assembly functions |
| `go/pkg/dtrules/interpreter/vm.go` | MODIFY | 2 | Call assembly for hot-path opcodes |
| `asm/src/vm/bytecode.asm` | MODIFY | 3 | Translate Plan 9 ops to NASM |
| `go/pkg/dtrules/asmruntime/bridge.go` | MODIFY | 3 | Pass DTState pointers, remove syncing |
| `go/pkg/dtrules/asmruntime/executor.go` | MODIFY | 3 | Simplify, remove sync code |
| `go/pkg/dtrules/runtime/nativeasm/vm.go` | DELETE | 4 | VMState eliminated |
| `go/pkg/dtrules/runtime/nativeasm/executor.go` | DELETE | 4 | Sync executor eliminated |
| `go/pkg/dtrules/runtime/nativeasm/vm_amd64.s` | DELETE | 4 | VMState assembly eliminated |
| `go/pkg/dtrules/runtime/nativeasm/stack_amd64.s` | DELETE | 4 | VMState stack ops eliminated |
| `go/pkg/dtrules/runtime/nativeasm/runtime.go` | SIMPLIFY | 4 | Thin wrapper or merge |

---

## Priority Order

1. **Phase 1** -- DTState fixed arrays. Low risk, enables everything else.
2. **Phase 2** -- NativeASM (Plan 9). Reference ASM implementation.
3. **Phase 3** -- x86-64-ASM (NASM). Translation of Phase 2.
4. **Phase 4** -- Delete VMState and broken code.
5. **Phase 5** -- Full test and benchmark.

---

## Key Principles

1. **One state, like Java.** DTState owns all execution state. No copies.
2. **PostScript model.** Three stacks, executable objects auto-invoke on lookup,
   operators work through the State interface.
3. **Assembly is an optimization, not an architecture.** Assembly subroutines
   speed up hot-path operations inside DTState. They don't create a parallel
   execution model.
4. **NativeASM first, then translate.** Plan 9 is the reference ASM. x86-64
   NASM is a translation of the same logic into a different assembler/calling
   convention.
5. **10240 stack depth.** Fixed arrays, large enough for any practical use.
   Can be grown with manual memory management if needed.
6. **All operators use State interface.** Confirmed by review. Internal changes
   to DTState are safe.

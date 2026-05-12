// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build amd64

package interpreter

import (
	"unsafe"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// ============================================================================
// Go helper functions called by the assembly dispatch loop.
//
// These exist ONLY as link targets. The assembly dispatch loop handles all
// 18 arithmetic/comparison/boolean operations inline with SSE2. These stubs
// should NEVER execute. If they do, it means the assembly fallback path
// was taken when it shouldn't have been.
//
// The 4 complex opcodes (lookup, def, operator, exec) are handled by
// assembly calling back into Go through these helpers, since they need
// access to Go data structures (entity stack, operator table, etc.).
// ============================================================================

// --- Arithmetic / comparison / boolean fallbacks ---
//
// These are link targets for the assembly. The SSE2 dispatch in
// asm_amd64.s handles every type combination inline; assembly never
// calls these. They exist only because the assembler emits CALL
// instructions to these symbols that are never reached at runtime,
// and the Go linker still needs the symbols resolved.
//
// They used to `panic()` to surface an assembly invariant violation
// as a loud failure. That works in a binary, but DTRules is meant to
// be embedded via `pkg/dtrules/sdk` — panicking inside a library
// takes the host process down (#756). We record the violation as a
// non-fatal runtime error via `setASMInvariantError` so the caller
// of ExecuteBytecodeASM gets an error instead of a crashed process.
// The "should never run" intent stays loud (every message names the
// op and points at the responsible asm path).

// setASMInvariantError stamps state.lastError with a description of
// the broken assembly invariant. The asm dispatch loop returns
// non-zero when it sees state.lastError set, so the host application
// surfaces a normal error rather than a panic.
func setASMInvariantError(state *DTState, op string) {
	state.lastError = dtrules.NewRulesError(
		"ASM Invariant Violation",
		"asm_helpers."+op+"Fallback",
		"assembly dispatch fell through to the Go fallback for "+op+
			" — assembly is expected to handle every type combination inline. "+
			"This is a bug in asm_amd64.s; please file with the failing rule set.",
	)
}

//go:nosplit
func goOpAddFallback(state *DTState) { setASMInvariantError(state, "goOpAdd") }

//go:nosplit
func goOpSubFallback(state *DTState) { setASMInvariantError(state, "goOpSub") }

//go:nosplit
func goOpMulFallback(state *DTState) { setASMInvariantError(state, "goOpMul") }

//go:nosplit
func goOpDivFallback(state *DTState) int {
	setASMInvariantError(state, "goOpDiv")
	return 5 // unknown opcode — see asm_stubs.go return code table
}

//go:nosplit
func goOpNegFallback(state *DTState) { setASMInvariantError(state, "goOpNeg") }

//go:nosplit
func goOpAbsFallback(state *DTState) { setASMInvariantError(state, "goOpAbs") }

//go:nosplit
func goOpIncFallback(state *DTState) { setASMInvariantError(state, "goOpInc") }

//go:nosplit
func goOpDecFallback(state *DTState) { setASMInvariantError(state, "goOpDec") }

//go:nosplit
func goOpEqFallback(state *DTState) { setASMInvariantError(state, "goOpEq") }

//go:nosplit
func goOpNeFallback(state *DTState) { setASMInvariantError(state, "goOpNe") }

//go:nosplit
func goOpLtFallback(state *DTState) { setASMInvariantError(state, "goOpLt") }

//go:nosplit
func goOpLeFallback(state *DTState) { setASMInvariantError(state, "goOpLe") }

//go:nosplit
func goOpGtFallback(state *DTState) { setASMInvariantError(state, "goOpGt") }

//go:nosplit
func goOpGeFallback(state *DTState) { setASMInvariantError(state, "goOpGe") }

//go:nosplit
func goOpAndFallback(state *DTState) { setASMInvariantError(state, "goOpAnd") }

//go:nosplit
func goOpOrFallback(state *DTState) { setASMInvariantError(state, "goOpOr") }

//go:nosplit
func goOpNotFallback(state *DTState) { setASMInvariantError(state, "goOpNot") }

//go:nosplit
func goOpXorFallback(state *DTState) { setASMInvariantError(state, "goOpXor") }

// --- Complex opcodes (called by assembly, do real work) ---

// goOpLookup handles the OpLookup opcode from the assembly dispatch loop.
// Pops a name from the data stack, looks it up in the entity stack context,
// and pushes the result (or executes it if executable).
func goOpLookup(state *DTState) int {
	nameObj, err := state.DataPop()
	if err != nil {
		state.lastError = err
		return 1
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		state.lastError = dtrules.ConversionError("OpLookup", "expected name")
		return 1
	}
	obj, err := state.Find(name)
	if err != nil {
		state.lastError = err
		return 1
	}
	if obj == nil {
		if err := state.DataPush(dtrules.GetRNull()); err != nil {
			state.lastError = err
			return 1
		}
	} else if obj.IsExecutable() {
		if err := obj.Execute(state); err != nil {
			state.lastError = err
			return 1
		}
	} else {
		if err := state.DataPush(obj); err != nil {
			state.lastError = err
			return 1
		}
	}
	return 0
}

// goOpDef handles the OpDef opcode from the assembly dispatch loop.
// Pops name and value from the data stack, defines the value in the entity context.
func goOpDef(state *DTState) int {
	nameObj, err := state.DataPop()
	if err != nil {
		state.lastError = err
		return 1
	}
	value, err := state.DataPop()
	if err != nil {
		state.lastError = err
		return 1
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		state.lastError = dtrules.ConversionError("OpDef", "expected name")
		return 1
	}
	ok, err := state.Def(name, value, state.TestState(TRACE))
	if err != nil {
		state.lastError = err
		return 1
	}
	if !ok {
		state.lastError = dtrules.UndefinedError("OpDef", name.StringValue()+" not found in context")
		return 1
	}
	return 0
}

// goOpOperator handles the OpOperator opcode from the assembly dispatch loop.
// Takes the operator index, looks up the operator, and executes it.
func goOpOperator(state *DTState, opIdx int) int {
	if state.operatorTable == nil || opIdx >= len(state.operatorTable) {
		state.lastError = dtrules.UndefinedError("OpOperator", "operator not found")
		return 1
	}
	op := state.operatorTable[opIdx]
	if op == nil {
		state.lastError = dtrules.UndefinedError("OpOperator", "operator not found")
		return 1
	}
	if err := op.Execute(state); err != nil {
		state.lastError = err
		return 1
	}
	return 0
}

// goOpExec handles the OpExec opcode from the assembly dispatch loop.
// Pops the top of stack and executes it. If it's a bytecode chunk,
// recursively enters the dispatch loop.
func goOpExec(state *DTState) int {
	obj, err := state.DataPop()
	if err != nil {
		state.lastError = err
		return 1
	}
	if err := obj.Execute(state); err != nil {
		state.lastError = err
		return 1
	}
	return 0
}

// ExecuteBytecodeASM runs a bytecode chunk through the assembly dispatch loop.
// This is the entry point called by the NativeASM executor.
func (s *DTState) ExecuteBytecodeASM(bc *dtrules.BytecodeChunk) error {
	code := bc.Code()
	if len(code) == 0 {
		return nil
	}

	constants := bc.Constants()
	names := bc.Names()

	var constPtr unsafe.Pointer
	if len(constants) > 0 {
		constPtr = unsafe.Pointer(&constants[0])
	}

	var namePtr unsafe.Pointer
	if len(names) > 0 {
		namePtr = unsafe.Pointer(&names[0])
	}

	rc := asmDispatchLoop(s, unsafe.Pointer(&code[0]), len(code), constPtr, namePtr)
	if rc != 0 {
		if s.lastError != nil {
			err := s.lastError
			s.lastError = nil
			return err
		}
		switch rc {
		case 1:
			return dtrules.StackUnderflowError("ExecuteBytecodeASM")
		case 2:
			return dtrules.StackOverflowError("ExecuteBytecodeASM", "")
		case 4:
			return dtrules.NewRulesError("Division By Zero", "ExecuteBytecodeASM", "cannot divide by zero")
		case 5:
			return dtrules.NewRulesError("Invalid Opcode", "ExecuteBytecodeASM", "unknown opcode")
		default:
			return dtrules.NewRulesError("Runtime Error", "ExecuteBytecodeASM", "assembly dispatch error")
		}
	}
	return nil
}

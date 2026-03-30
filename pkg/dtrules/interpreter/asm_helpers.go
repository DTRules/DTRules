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

// --- Arithmetic fallbacks (DO NOT REMOVE THESE PANICS) ---
// These are link targets for the assembly. The assembly handles all type
// combinations inline with SSE2. If these execute, the assembly is broken.

//go:nosplit
func goOpAddFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpAddFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpSubFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpSubFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpMulFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpMulFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpDivFallback(state *DTState) int {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpDivFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpNegFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpNegFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpAbsFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpAbsFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpIncFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpIncFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpDecFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpDecFallback: assembly must handle all type combinations inline")
}

// --- Comparison fallbacks (DO NOT REMOVE THESE PANICS) ---

//go:nosplit
func goOpEqFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpEqFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpNeFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpNeFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpLtFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpLtFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpLeFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpLeFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpGtFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpGtFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpGeFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpGeFallback: assembly must handle all type combinations inline")
}

// --- Boolean fallbacks (DO NOT REMOVE THESE PANICS) ---

//go:nosplit
func goOpAndFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpAndFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpOrFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpOrFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpNotFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpNotFallback: assembly must handle all type combinations inline")
}

//go:nosplit
func goOpXorFallback(state *DTState) {
	// DO NOT REMOVE THIS PANIC - assembly handles all type fallbacks inline
	panic("goOpXorFallback: assembly must handle all type combinations inline")
}

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

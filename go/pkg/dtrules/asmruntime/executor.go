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

package asmruntime

import (
	"fmt"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
)

// Executor implements the BytecodeExecutor interface for the x86-64 ASM runtime.
// This is the CGO-based NASM implementation.
//
// WARNING: This implementation is currently incomplete. It should be a
// self-contained runtime following the architecture documented in ARCHITECTURE.md.
// The runtime should NOT fall back to Go or sync state with DTState.
type Executor struct {
	initialized bool
}

// NewExecutor creates a new x86-64 ASM bytecode executor.
func NewExecutor() (*Executor, error) {
	if err := Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize ASM runtime: %w", err)
	}
	return &Executor{initialized: true}, nil
}

// Name returns the runtime name.
func (e *Executor) Name() string {
	return "x86-64-asm"
}

// ExecuteBytecode executes bytecode using the x86-64 ASM runtime.
//
// WARNING: This implementation currently has architectural issues similar to
// NativeASM. It needs to be a self-contained runtime with no Go fallbacks.
// See ARCHITECTURE.md for the correct design.
func (e *Executor) ExecuteBytecode(state *interpreter.DTState, bc *dtrules.BytecodeChunk) error {
	if !e.initialized {
		return ErrNotInitialized
	}

	// Reset the ASM runtime for this execution
	if err := Reset(); err != nil {
		return fmt.Errorf("failed to reset ASM runtime: %w", err)
	}

	// Copy data stack from DTState to ASM
	dataDepth := state.DataStackDepth()
	for i := 0; i < dataDepth; i++ {
		obj, err := state.DataFetch(dataDepth - 1 - i) // Bottom to top
		if err != nil {
			return fmt.Errorf("failed to fetch data stack: %w", err)
		}
		v := objectToValue(obj)
		if err := PushValue(v); err != nil {
			return fmt.Errorf("failed to push to ASM stack: %w", err)
		}
	}

	// Copy entity stack from DTState to ASM
	entityDepth := state.EntityDepth()
	if err := SetupEntityStack(state); err != nil {
		return fmt.Errorf("failed to setup entity stack (Go depth=%d, ASM depth=%d): %w",
			entityDepth, EntityStackDepth(), err)
	}

	// Verify entity stack was set up
	asmEntityDepth := EntityStackDepth()
	if asmEntityDepth == 0 && entityDepth > 0 {
		return fmt.Errorf("entity stack setup failed: Go has %d entities but ASM has 0", entityDepth)
	}

	// Execute bytecode
	if err := ExecuteBytecode(bc); err != nil {
		return fmt.Errorf("ASM execution error (entity depth=%d): %w", asmEntityDepth, err)
	}

	// Copy data stack back from ASM to DTState
	state.DataClear()
	asmDepth := StackDepth()
	values := make([]dtrules.Value, asmDepth)
	for i := asmDepth - 1; i >= 0; i-- {
		v, err := PopValue()
		if err != nil {
			return fmt.Errorf("failed to pop from ASM stack: %w", err)
		}
		values[i] = v
	}

	// Debug: log the result value
	if debugASMExecution && asmDepth > 0 {
		tag, num, ptr := values[asmDepth-1].RawParts()
		fmt.Printf("[ASM DEBUG] Result: tag=%d, num=%d, ptr=%p\n", tag, num, ptr)
	}

	for _, v := range values {
		obj := valueToObject(v)
		if err := state.DataPush(obj); err != nil {
			return fmt.Errorf("failed to push to DTState: %w", err)
		}
	}

	return nil
}

// debugASMExecution enables debug output for ASM execution
var debugASMExecution = false

// SetDebugASMExecution enables or disables ASM execution debugging
func SetDebugASMExecution(enabled bool) {
	debugASMExecution = enabled
}

// objectToValue converts a DTRules Object to a Value for ASM.
func objectToValue(obj dtrules.Object) dtrules.Value {
	if obj == nil {
		return dtrules.ValueNull
	}
	switch obj.Type() {
	case dtrules.TypeNull:
		return dtrules.ValueNull
	case dtrules.TypeInteger:
		v, _ := obj.LongValue()
		return dtrules.NewValueInteger(v)
	case dtrules.TypeDouble:
		v, _ := obj.DoubleValue()
		return dtrules.NewValueDouble(v)
	case dtrules.TypeBoolean:
		v, _ := obj.BooleanValue()
		return dtrules.NewValueBoolean(v)
	case dtrules.TypeString:
		return dtrules.NewValueString(obj.StringValue())
	case dtrules.TypeName:
		n, _ := obj.RNameValue()
		return dtrules.NewValueName(n)
	case dtrules.TypeArray:
		a, _ := obj.RArrayValue()
		return dtrules.NewValueArray(a)
	}
	switch o := obj.(type) {
	case dtrules.Entity:
		return dtrules.NewValueEntity(o)
	default:
		return dtrules.NewValueObject(obj)
	}
}

// valueToObject converts a Value to a DTRules Object.
func valueToObject(v dtrules.Value) dtrules.Object {
	return v.AsObject()
}

// Ensure Executor implements the interface
var _ interpreter.BytecodeExecutor = (*Executor)(nil)

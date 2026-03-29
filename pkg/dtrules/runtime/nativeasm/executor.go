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

package nativeasm

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

// Executor implements the BytecodeExecutor interface for the NativeASM runtime.
//
// The assembly dispatch loop (asmDispatchLoop) owns the entire bytecode
// execution. It handles all opcodes inline -- arithmetic, comparison,
// boolean, stack, and push operations run in assembly with zero Go
// dispatch overhead. For opcodes that interact with Go data structures
// (lookup, def, operator, exec), assembly calls Go helper functions
// and stays in control of the main loop.
type Executor struct{}

// NewExecutor creates a new NativeASM bytecode executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// Name returns the runtime name.
func (e *Executor) Name() string {
	return "nativeasm"
}

// ExecuteBytecode runs bytecode through the assembly dispatch loop.
// Assembly owns the entire execution.
func (e *Executor) ExecuteBytecode(state *interpreter.DTState, bc *dtrules.BytecodeChunk) error {
	return state.ExecuteBytecodeASM(bc)
}

// Ensure Executor implements the interface
var _ interpreter.BytecodeExecutor = (*Executor)(nil)

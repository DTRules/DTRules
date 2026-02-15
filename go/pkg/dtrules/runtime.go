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

package dtrules

// RuntimeInit provides methods to push initial state into a runtime before
// execution. Callers use this interface to set up entities, values, and
// configuration that the runtime needs. All initialization must complete
// before ExecuteBytecode is called — no mid-execution syncing occurs.
type RuntimeInit interface {
	// PushEntity pushes an entity onto the runtime's entity stack.
	// Entities pushed first are searched last (stack order).
	PushEntity(entity Entity) error

	// PushValue pushes a value onto the runtime's value stack.
	PushValue(v Value) error

	// PushData pushes an object onto the runtime's data stack.
	PushData(obj Object) error

	// SetOperatorTable provides the operator table for bytecode execution.
	SetOperatorTable(table []Object)

	// SetSession sets the session reference for the runtime.
	// The runtime uses this for entity creation, compilation, and
	// other session-scoped operations during execution.
	SetSession(session Session)
}

// RuntimeQuery provides methods to extract results from a runtime after
// execution completes. Callers use this interface to read entities, values,
// and data stack contents without accessing the runtime's internal state.
type RuntimeQuery interface {
	// EntityByIndex returns the entity at the given depth from the top of
	// the entity stack. 0 returns the top entity.
	EntityByIndex(index int) (Entity, error)

	// EntityDepth returns the current depth of the entity stack.
	EntityStackDepth() int

	// PopValue pops and returns a value from the runtime's value stack.
	PopValue() (Value, error)

	// PeekValue returns the top value without removing it.
	PeekValue() (Value, error)

	// ValueStackDepth returns the current depth of the value stack.
	ValueStackSize() int

	// PopData pops and returns an object from the runtime's data stack.
	PopData() (Object, error)

	// PeekData returns the top data stack object without removing it.
	PeekData() (Object, error)

	// DataStackDepth returns the current depth of the data stack.
	DataStackSize() int

	// FindValue looks up a name in the entity stack context and returns
	// the value as a Value. Returns ValueNull if not found.
	FindValue(name *RName) (Value, error)

	// FindEntity locates the entity containing the named attribute.
	// Returns nil if no entity on the stack contains the attribute.
	QueryEntity(name *RName) (Entity, error)
}

// Runtime combines initialization, execution, and querying into a single
// interface. Each runtime implementation owns its VMState entirely.
// The lifecycle is:
//  1. Create the runtime
//  2. Use RuntimeInit to push entities and values before execution
//  3. Call Execute to run bytecode or decision tables
//  4. Use RuntimeQuery to extract results after execution
type Runtime interface {
	RuntimeInit
	RuntimeQuery

	// Name returns the name of this runtime implementation (e.g. "go", "nativeasm").
	Name() string

	// ExecuteBytecode executes a bytecode chunk using the runtime's own state.
	// All state must be initialized via RuntimeInit before calling this method.
	ExecuteBytecode(bc *BytecodeChunk) error

	// GetState returns the State interface for this runtime, providing
	// full access to the interpreter state. This allows existing code that
	// operates on State (decision tables, operators, etc.) to continue
	// working during the transition.
	GetState() State
}

// RuntimeFactory creates new Runtime instances for a given session.
type RuntimeFactory interface {
	// Name returns the name of this runtime factory (e.g. "go", "nativeasm").
	Name() string

	// CreateRuntime creates a new Runtime for the given session.
	// The returned runtime owns its own VMState.
	CreateRuntime(session Session) (Runtime, error)
}

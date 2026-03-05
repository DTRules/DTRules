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

// Package goruntime provides a pure Go implementation of the DTRules runtime interface.
//
// This runtime wraps the existing interpreter/vm.go bytecode execution, providing
// a clean interface that matches the runtime.Runtime and runtime.ExecutionContext
// interfaces.
//
// Usage:
//
//	rt := goruntime.New()
//	ctx, err := rt.CreateContext()
//	if err != nil { ... }
//	defer ctx.Close()
//
//	err = ctx.ExecuteBytecode(chunk)
package goruntime

import (
	"sync"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/go/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
)

const (
	runtimeName    = "go"
	runtimeVersion = "1.0.0"
)

// GoRuntime is the pure Go implementation of the Runtime interface.
type GoRuntime struct {
	closed bool
	mu     sync.Mutex
}

// New creates a new GoRuntime instance.
func New() *GoRuntime {
	return &GoRuntime{}
}

// Name returns the runtime name.
func (r *GoRuntime) Name() string {
	return runtimeName
}

// Version returns the runtime version.
func (r *GoRuntime) Version() string {
	return runtimeVersion
}

// Capabilities returns the runtime capabilities.
func (r *GoRuntime) Capabilities() runtime.Capabilities {
	return runtime.Capabilities{
		ConcurrentContexts:   true, // Go runtime supports multiple concurrent contexts
		Tracing:              true,
		MaxStackDepth:        1000, // Matches interpreter.stackLimit
		MaxEntityDepth:       1000,
		SupportsAllOperators: true,
	}
}

// CreateContext creates a new execution context.
func (r *GoRuntime) CreateContext(opts ...runtime.ContextOption) (runtime.ExecutionContext, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, runtime.ErrRuntimeClosed
	}

	config := runtime.DefaultContextConfig()
	config.ApplyOptions(opts)

	ctx := &GoContext{
		state:  interpreter.NewDTState(nil), // No session - pure bytecode execution
		config: config,
	}

	// Set up operator table for bytecode execution
	ctx.state.SetOperatorTable(operators.GetOperatorTable())

	return ctx, nil
}

// Close releases runtime resources.
func (r *GoRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	return nil
}

// GoContext is the Go implementation of ExecutionContext.
type GoContext struct {
	state    *interpreter.DTState
	config   runtime.ContextConfig
	closed   bool
	lastErr  error
	mu       sync.Mutex
}

// --- Data Stack Operations ---
// Note: Following the PostScript model, there is a single operand stack (data stack).
// The runtime interface uses Value types, so we convert at the boundary.

// Push pushes a value onto the data stack.
func (c *GoContext) Push(v dtrules.Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	err := c.state.ValuePush(v)
	if err != nil {
		c.lastErr = runtime.ErrStackOverflow.WithCause(err)
		return c.lastErr
	}
	return nil
}

// Pop removes and returns the top value from the data stack.
func (c *GoContext) Pop() (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	v, err := c.state.ValuePop()
	if err != nil {
		c.lastErr = runtime.ErrStackUnderflow.WithCause(err)
		return dtrules.ValueNull, c.lastErr
	}
	return v, nil
}

// Peek returns the top value without removing it.
func (c *GoContext) Peek() (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	v, err := c.state.ValuePeek()
	if err != nil {
		c.lastErr = runtime.ErrStackUnderflow.WithCause(err)
		return dtrules.ValueNull, c.lastErr
	}
	return v, nil
}

// StackDepth returns the current data stack depth.
func (c *GoContext) StackDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0
	}
	return c.state.ValueStackDepth()
}

// --- Entity Stack Operations ---

// EntityPush pushes an entity onto the entity stack.
func (c *GoContext) EntityPush(e dtrules.Entity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	err := c.state.EntityPush(e)
	if err != nil {
		c.lastErr = runtime.ErrStackOverflow.WithCause(err)
		return c.lastErr
	}
	return nil
}

// EntityPop removes and returns the top entity from the entity stack.
func (c *GoContext) EntityPop() (dtrules.Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, runtime.ErrContextClosed
	}

	e, err := c.state.EntityPop()
	if err != nil {
		c.lastErr = runtime.ErrEntityStackUnderflow.WithCause(err)
		return nil, c.lastErr
	}
	return e, nil
}

// EntityPeek returns the top entity without removing it.
func (c *GoContext) EntityPeek() (dtrules.Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, runtime.ErrContextClosed
	}

	e, err := c.state.EntityFetch(0)
	if err != nil {
		c.lastErr = runtime.ErrEntityStackUnderflow.WithCause(err)
		return nil, c.lastErr
	}
	return e, nil
}

// EntityDepth returns the current entity stack depth.
func (c *GoContext) EntityDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0
	}
	return c.state.EntityDepth()
}

// FindEntity searches the entity stack for an entity containing the named attribute.
func (c *GoContext) FindEntity(name *dtrules.RName) (dtrules.Entity, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, false
	}

	e, err := c.state.FindEntity(name)
	if err != nil || e == nil {
		return nil, false
	}
	return e, true
}

// --- Variable Operations ---

// Def defines or updates a variable in the current entity context.
func (c *GoContext) Def(name *dtrules.RName, value dtrules.Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	obj := value.AsObject()
	ok, err := c.state.Def(name, obj, false)
	if err != nil {
		c.lastErr = err
		return err
	}
	if !ok {
		c.lastErr = runtime.ErrNameNotFound
		return c.lastErr
	}
	return nil
}

// Lookup retrieves a variable's value from the entity stack context.
func (c *GoContext) Lookup(name *dtrules.RName) (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	obj, err := c.state.Find(name)
	if err != nil {
		c.lastErr = err
		return dtrules.ValueNull, err
	}
	if obj == nil {
		return dtrules.ValueNull, nil
	}
	return dtrules.ValueFromObject(obj), nil
}

// --- Execution ---

// ExecuteBytecode executes a bytecode chunk.
func (c *GoContext) ExecuteBytecode(bc *dtrules.BytecodeChunk) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	if bc == nil {
		return nil
	}

	err := c.state.ExecuteBytecode(bc)
	if err != nil {
		c.lastErr = err
		return err
	}
	return nil
}

// EvaluateCondition executes bytecode and returns the result as a boolean.
func (c *GoContext) EvaluateCondition(bc *dtrules.BytecodeChunk) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false, runtime.ErrContextClosed
	}

	if bc == nil {
		return false, nil
	}

	result, err := c.state.EvaluateBytecodeCondition(bc)
	if err != nil {
		c.lastErr = err
		return false, err
	}
	return result, nil
}

// --- State Management ---

// Reset clears all stack state but keeps the context usable.
func (c *GoContext) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	// Clear value stack by popping all values
	for c.state.ValueStackDepth() > 0 {
		c.state.ValuePop()
	}
	// Clear entity stack by popping all entities
	for c.state.EntityDepth() > 0 {
		c.state.EntityPop()
	}
	c.lastErr = nil
	return nil
}

// GetError returns the last error that occurred during execution.
func (c *GoContext) GetError() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lastErr
}

// Close releases resources associated with this context.
func (c *GoContext) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.state = nil
	return nil
}

// GetState returns the underlying DTState for advanced use cases.
// This breaks the abstraction but is useful for integration with existing code.
func (c *GoContext) GetState() *interpreter.DTState {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.state
}

// Verify interface compliance at compile time
var _ runtime.Runtime = (*GoRuntime)(nil)
var _ runtime.ExecutionContext = (*GoContext)(nil)

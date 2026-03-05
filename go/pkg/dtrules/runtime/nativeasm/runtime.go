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

// Package nativeasm provides the NativeASM runtime that uses Plan 9 assembly
// subroutines integrated directly into DTState for hot-path bytecode operations.
//
// Assembly is now an optimization inside DTState (Phase 2), not a separate
// execution model. This package provides the runtime.Runtime wrapper and
// the BytecodeExecutor for the pluggable runtime interface.
package nativeasm

import (
	"sync"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/go/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
)

// NativeRuntime implements the runtime.Runtime interface using Plan 9 assembly.
// Assembly subroutines are integrated directly into DTState.
// No separate VMState, no syncing.
type NativeRuntime struct {
	closed bool
	mu     sync.Mutex
}

// New creates a new native assembly runtime.
func New() *NativeRuntime {
	return &NativeRuntime{}
}

// Name returns the runtime's name.
func (r *NativeRuntime) Name() string {
	return "native-asm"
}

// Version returns the runtime's version.
func (r *NativeRuntime) Version() string {
	return "2.0.0"
}

// Capabilities returns information about what features the runtime supports.
func (r *NativeRuntime) Capabilities() runtime.Capabilities {
	return runtime.Capabilities{
		ConcurrentContexts:   true,
		Tracing:              false,
		MaxStackDepth:        interpreter.MaxStackDepth,
		MaxEntityDepth:       interpreter.MaxStackDepth,
		SupportsAllOperators: true, // Assembly handles hot-path, Go handles the rest
	}
}

// CreateContext creates a new execution context.
func (r *NativeRuntime) CreateContext(opts ...runtime.ContextOption) (runtime.ExecutionContext, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, runtime.ErrRuntimeClosed
	}

	config := runtime.DefaultContextConfig()
	config.ApplyOptions(opts)

	state := interpreter.NewDTState(nil)
	state.SetOperatorTable(operators.GetOperatorTable())
	// NativeASM executor delegates to DTState's built-in Go+ASM path
	state.SetBytecodeExecutor(NewExecutor())

	return &NativeContext{
		state:   state,
		config:  config,
		runtime: r,
	}, nil
}

// Close releases resources held by the runtime.
func (r *NativeRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

// NativeContext implements the runtime.ExecutionContext interface.
// It wraps DTState with the NativeASM executor set.
type NativeContext struct {
	state   *interpreter.DTState
	config  runtime.ContextConfig
	runtime *NativeRuntime
	closed  bool
	lastErr error
	mu      sync.Mutex
}

// --- Data Stack Operations ---

func (c *NativeContext) Push(v dtrules.Value) error {
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

func (c *NativeContext) Pop() (dtrules.Value, error) {
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

func (c *NativeContext) Peek() (dtrules.Value, error) {
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

func (c *NativeContext) StackDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0
	}
	return c.state.ValueStackDepth()
}

// --- Entity Stack Operations ---

func (c *NativeContext) EntityPush(e dtrules.Entity) error {
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

func (c *NativeContext) EntityPop() (dtrules.Entity, error) {
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

func (c *NativeContext) EntityPeek() (dtrules.Entity, error) {
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

func (c *NativeContext) EntityDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0
	}
	return c.state.EntityDepth()
}

func (c *NativeContext) FindEntity(name *dtrules.RName) (dtrules.Entity, bool) {
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

func (c *NativeContext) Def(name *dtrules.RName, value dtrules.Value) error {
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

func (c *NativeContext) Lookup(name *dtrules.RName) (dtrules.Value, error) {
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

func (c *NativeContext) ExecuteBytecode(bc *dtrules.BytecodeChunk) error {
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

func (c *NativeContext) EvaluateCondition(bc *dtrules.BytecodeChunk) (bool, error) {
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

func (c *NativeContext) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return runtime.ErrContextClosed
	}
	for c.state.ValueStackDepth() > 0 {
		c.state.ValuePop()
	}
	for c.state.EntityDepth() > 0 {
		c.state.EntityPop()
	}
	c.lastErr = nil
	return nil
}

func (c *NativeContext) GetError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastErr
}

func (c *NativeContext) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.state = nil
	return nil
}

// Compile-time checks
var _ runtime.Runtime = (*NativeRuntime)(nil)
var _ runtime.ExecutionContext = (*NativeContext)(nil)

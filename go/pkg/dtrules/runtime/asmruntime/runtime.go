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

// Package asmruntime provides an x86-64 assembly implementation of the DTRules runtime interface.
//
// This runtime wraps the existing asmruntime/bridge.go CGO bindings, providing
// a clean interface that matches the runtime.Runtime and runtime.ExecutionContext
// interfaces.
//
// Note: The ASM runtime uses global state and does NOT support concurrent contexts.
// Only one context can execute at a time.
//
// Usage:
//
//	rt, err := asmruntime.New()
//	if err != nil { ... }
//	defer rt.Close()
//
//	ctx, err := rt.CreateContext()
//	if err != nil { ... }
//	defer ctx.Close()
//
//	err = ctx.ExecuteBytecode(chunk)
package asmruntime

import (
	"sync"
	"unsafe"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/asmruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
)

const (
	runtimeName    = "x86-64-asm"
	runtimeVersion = "1.0.0"
)

// ASMRuntime is the x86-64 assembly implementation of the Runtime interface.
type ASMRuntime struct {
	closed        bool
	activeContext *ASMContext // Only one context can be active at a time
	mu            sync.Mutex
}

// New creates a new ASMRuntime instance and initializes the ASM VM.
func New() (*ASMRuntime, error) {
	if err := asmruntime.Init(); err != nil {
		return nil, runtime.NewRuntimeError(12, "failed to initialize ASM runtime: "+err.Error())
	}

	return &ASMRuntime{}, nil
}

// Name returns the runtime name.
func (r *ASMRuntime) Name() string {
	return runtimeName
}

// Version returns the runtime version.
func (r *ASMRuntime) Version() string {
	return runtimeVersion
}

// Capabilities returns the runtime capabilities.
func (r *ASMRuntime) Capabilities() runtime.Capabilities {
	return runtime.Capabilities{
		ConcurrentContexts:   false, // ASM uses global state
		Tracing:              false, // ASM tracing not yet implemented
		MaxStackDepth:        1024,  // Matches ASM stack size
		MaxEntityDepth:       256,   // Matches ASM entity stack size
		SupportsAllOperators: false, // ASM supports subset of operators
	}
}

// CreateContext creates a new execution context.
// Note: Only one context can be active at a time due to global ASM state.
func (r *ASMRuntime) CreateContext(opts ...runtime.ContextOption) (runtime.ExecutionContext, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, runtime.ErrRuntimeClosed
	}

	// Check if there's already an active context
	if r.activeContext != nil && !r.activeContext.closed {
		return nil, runtime.NewRuntimeError(13, "ASM runtime only supports one active context at a time")
	}

	// Reset ASM state for new context
	if err := asmruntime.Reset(); err != nil {
		return nil, runtime.NewRuntimeError(14, "failed to reset ASM state: "+err.Error())
	}

	config := runtime.DefaultContextConfig()
	config.ApplyOptions(opts)

	ctx := &ASMContext{
		runtime: r,
		config:  config,
	}

	r.activeContext = ctx
	return ctx, nil
}

// Close releases runtime resources.
func (r *ASMRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	if r.activeContext != nil {
		r.activeContext.closed = true
		r.activeContext = nil
	}
	return nil
}

// ASMContext is the ASM implementation of ExecutionContext.
type ASMContext struct {
	runtime *ASMRuntime
	config  runtime.ContextConfig
	closed  bool
	lastErr error
	mu      sync.Mutex
}

// --- Data Stack Operations ---

// Push pushes a value onto the data stack.
func (c *ASMContext) Push(v dtrules.Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	err := asmruntime.PushValue(v)
	if err != nil {
		c.lastErr = mapASMError(err)
		return c.lastErr
	}
	return nil
}

// Pop removes and returns the top value from the data stack.
func (c *ASMContext) Pop() (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	v, err := asmruntime.PopValue()
	if err != nil {
		c.lastErr = mapASMError(err)
		return dtrules.ValueNull, c.lastErr
	}
	return v, nil
}

// Peek returns the top value without removing it.
func (c *ASMContext) Peek() (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	v, err := asmruntime.PeekValue()
	if err != nil {
		c.lastErr = mapASMError(err)
		return dtrules.ValueNull, c.lastErr
	}
	return v, nil
}

// StackDepth returns the current data stack depth.
func (c *ASMContext) StackDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0
	}
	return asmruntime.StackDepth()
}

// --- Entity Stack Operations ---

// EntityPush pushes an entity onto the entity stack.
func (c *ASMContext) EntityPush(e dtrules.Entity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	// Marshal entity to ASM format
	asmEntity, err := asmruntime.MarshalEntity(e)
	if err != nil {
		c.lastErr = mapASMError(err)
		return c.lastErr
	}

	// Create a Value wrapping the entity pointer
	entityValue := asmruntime.HeapAlloc(24)
	if entityValue == nil {
		c.lastErr = runtime.ErrOutOfMemory
		return c.lastErr
	}

	// Set tag to VTAG_ENTITY (7) and ptr to entity
	*(*byte)(entityValue) = 7
	*(*unsafe.Pointer)(unsafe.Pointer(uintptr(entityValue) + 16)) = asmEntity

	if err := asmruntime.EntityStackPush(entityValue); err != nil {
		c.lastErr = mapASMError(err)
		return c.lastErr
	}
	return nil
}

// EntityPop removes and returns the top entity from the entity stack.
func (c *ASMContext) EntityPop() (dtrules.Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, runtime.ErrContextClosed
	}

	ptr := asmruntime.EntityStackPop()
	if ptr == nil {
		c.lastErr = runtime.ErrEntityStackUnderflow
		return nil, c.lastErr
	}

	// Note: We cannot easily convert ASM entity back to Go entity
	// For now, return nil - this is primarily used for stack management
	return nil, nil
}

// EntityPeek returns the top entity without removing it.
func (c *ASMContext) EntityPeek() (dtrules.Entity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, runtime.ErrContextClosed
	}

	ptr := asmruntime.EntityStackPeek(0)
	if ptr == nil {
		c.lastErr = runtime.ErrEntityStackUnderflow
		return nil, c.lastErr
	}

	// Note: We cannot easily convert ASM entity back to Go entity
	return nil, nil
}

// EntityDepth returns the current entity stack depth.
func (c *ASMContext) EntityDepth() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0
	}
	return asmruntime.EntityStackDepth()
}

// FindEntity searches the entity stack for an entity containing the named attribute.
// Note: This operation is not directly supported in ASM - returns false.
func (c *ASMContext) FindEntity(name *dtrules.RName) (dtrules.Entity, bool) {
	// ASM runtime doesn't support entity introspection from Go
	return nil, false
}

// --- Variable Operations ---

// Def defines or updates a variable in the current entity context.
// Note: This operation is limited in ASM context.
func (c *ASMContext) Def(name *dtrules.RName, value dtrules.Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	// For ASM, we would need to find the entity and set the attribute
	// This is complex because we need to search the ASM entity stack
	// For now, return an error indicating this is not supported
	return runtime.NewRuntimeError(15, "Def not supported in ASM context - use bytecode OpDef instead")
}

// Lookup retrieves a variable's value from the entity stack context.
// Note: This operation is limited in ASM context.
func (c *ASMContext) Lookup(name *dtrules.RName) (dtrules.Value, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return dtrules.ValueNull, runtime.ErrContextClosed
	}

	// For ASM, we would need to search the ASM entity stack
	// This is complex and better done through bytecode OpLookup
	return dtrules.ValueNull, runtime.NewRuntimeError(16, "Lookup not supported in ASM context - use bytecode OpLookup instead")
}

// --- Execution ---

// ExecuteBytecode executes a bytecode chunk.
func (c *ASMContext) ExecuteBytecode(bc *dtrules.BytecodeChunk) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	if bc == nil {
		return nil
	}

	err := asmruntime.ExecuteBytecode(bc)
	if err != nil {
		c.lastErr = mapASMError(err)
		return c.lastErr
	}
	return nil
}

// EvaluateCondition executes bytecode and returns the result as a boolean.
func (c *ASMContext) EvaluateCondition(bc *dtrules.BytecodeChunk) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false, runtime.ErrContextClosed
	}

	if bc == nil {
		return false, nil
	}

	initialDepth := asmruntime.StackDepth()

	err := asmruntime.ExecuteBytecode(bc)
	if err != nil {
		c.lastErr = mapASMError(err)
		return false, c.lastErr
	}

	// Must have at least one result
	if asmruntime.StackDepth() <= initialDepth {
		c.lastErr = runtime.NewRuntimeError(17, "no result on stack after condition evaluation")
		return false, c.lastErr
	}

	// Pop the result
	result, err := asmruntime.PopValue()
	if err != nil {
		c.lastErr = mapASMError(err)
		return false, c.lastErr
	}

	// Clean up any extra values
	for asmruntime.StackDepth() > initialDepth {
		asmruntime.PopValue()
	}

	return result.AsBoolean(), nil
}

// --- State Management ---

// Reset clears all stack state but keeps the context usable.
func (c *ASMContext) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return runtime.ErrContextClosed
	}

	err := asmruntime.Reset()
	if err != nil {
		c.lastErr = mapASMError(err)
		return c.lastErr
	}
	c.lastErr = nil
	return nil
}

// GetError returns the last error that occurred during execution.
func (c *ASMContext) GetError() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lastErr
}

// Close releases resources associated with this context.
func (c *ASMContext) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	// Clear the active context reference in the runtime
	if c.runtime != nil {
		c.runtime.mu.Lock()
		if c.runtime.activeContext == c {
			c.runtime.activeContext = nil
		}
		c.runtime.mu.Unlock()
	}

	return nil
}

// mapASMError maps ASM bridge errors to runtime errors.
func mapASMError(err error) *runtime.RuntimeError {
	if err == nil {
		return nil
	}

	switch err {
	case asmruntime.ErrStackOverflow:
		return runtime.ErrStackOverflow
	case asmruntime.ErrStackUnderflow:
		return runtime.ErrStackUnderflow
	case asmruntime.ErrTypeMismatch:
		return runtime.ErrTypeMismatch
	case asmruntime.ErrDivByZero:
		return runtime.ErrDivisionByZero
	case asmruntime.ErrOutOfMemory:
		return runtime.ErrOutOfMemory
	case asmruntime.ErrInvalidOpcode:
		return runtime.ErrInvalidOpcode
	case asmruntime.ErrIndexBounds:
		return runtime.ErrIndexOutOfBounds
	case asmruntime.ErrNameNotFound:
		return runtime.ErrNameNotFound
	case asmruntime.ErrNotInitialized:
		return runtime.ErrRuntimeClosed
	default:
		return runtime.NewRuntimeError(99, err.Error())
	}
}

// Verify interface compliance at compile time
var _ runtime.Runtime = (*ASMRuntime)(nil)
var _ runtime.ExecutionContext = (*ASMContext)(nil)

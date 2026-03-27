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

// Package runtime defines the interfaces for DTRules execution runtimes.
//
// A Runtime is a bytecode execution engine that can be implemented in various
// ways: Go interpreter, x86-64 assembly, ARM, WASM, or even remote execution.
// This abstraction allows rules to be compiled once and executed anywhere.
//
// The key interfaces are:
//   - Runtime: Factory for creating execution contexts
//   - ExecutionContext: Stateful environment for executing bytecode
//
// Usage:
//
//	// Choose a runtime
//	rt := goruntime.New()  // or asmruntime.New()
//
//	// Create an execution context
//	ctx, err := rt.CreateContext()
//
//	// Execute bytecode
//	err = ctx.ExecuteBytecode(chunk)
//
//	// Clean up
//	ctx.Close()
//	rt.Close()
package runtime

import (
	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// Runtime represents a bytecode execution engine.
//
// Implementations include:
//   - goruntime: Pure Go interpreter
//   - asmruntime: x86-64 assembly via CGO
//
// Each Runtime can create multiple ExecutionContexts for concurrent execution
// (depending on implementation capabilities).
type Runtime interface {
	// Name returns the runtime's human-readable name.
	Name() string

	// Version returns the runtime's version string.
	Version() string

	// Capabilities returns information about what features the runtime supports.
	Capabilities() Capabilities

	// CreateContext creates a new execution context.
	// Options can be used to configure the context (stack sizes, tracing, etc.).
	CreateContext(opts ...ContextOption) (ExecutionContext, error)

	// Close releases any resources held by the runtime.
	// After Close, CreateContext should not be called.
	Close() error
}

// Capabilities describes what features a runtime supports.
type Capabilities struct {
	// ConcurrentContexts indicates if multiple contexts can execute simultaneously.
	// ASM runtime typically does not support this (global state).
	ConcurrentContexts bool

	// Tracing indicates if the runtime can emit trace events.
	Tracing bool

	// MaxStackDepth is the maximum data stack depth, or 0 for unlimited.
	MaxStackDepth int

	// MaxEntityDepth is the maximum entity stack depth, or 0 for unlimited.
	MaxEntityDepth int

	// SupportsAllOperators indicates if all standard operators are supported.
	// Some lightweight runtimes may only support a subset.
	SupportsAllOperators bool
}

// ExecutionContext represents the state of a single execution thread.
//
// An ExecutionContext maintains:
//   - Data stack: For expression evaluation
//   - Entity stack: For name resolution context
//   - Variable bindings: For def/lookup operations
//
// Each context is independent and should not be used from multiple goroutines
// simultaneously (unless the runtime explicitly supports it).
type ExecutionContext interface {
	// --- Data Stack Operations ---

	// Push pushes a value onto the data stack.
	Push(v dtrules.Value) error

	// Pop removes and returns the top value from the data stack.
	Pop() (dtrules.Value, error)

	// Peek returns the top value without removing it.
	Peek() (dtrules.Value, error)

	// StackDepth returns the current data stack depth.
	StackDepth() int

	// --- Entity Stack Operations ---

	// EntityPush pushes an entity onto the entity stack.
	// This establishes a new name resolution context.
	EntityPush(e dtrules.Entity) error

	// EntityPop removes and returns the top entity from the entity stack.
	EntityPop() (dtrules.Entity, error)

	// EntityPeek returns the top entity without removing it.
	EntityPeek() (dtrules.Entity, error)

	// EntityDepth returns the current entity stack depth.
	EntityDepth() int

	// FindEntity searches the entity stack for an entity containing the named attribute.
	// Returns the entity and true if found, or nil and false if not found.
	FindEntity(name *dtrules.RName) (dtrules.Entity, bool)

	// --- Variable Operations ---

	// Def defines or updates a variable in the current entity context.
	// The name is looked up on the entity stack to find where to store the value.
	Def(name *dtrules.RName, value dtrules.Value) error

	// Lookup retrieves a variable's value from the entity stack context.
	Lookup(name *dtrules.RName) (dtrules.Value, error)

	// --- Execution ---

	// ExecuteBytecode executes a bytecode chunk.
	// The chunk is executed to completion (or error).
	// Stack state is preserved across calls.
	ExecuteBytecode(bc *dtrules.BytecodeChunk) error

	// EvaluateCondition executes bytecode and returns the result as a boolean.
	// The stack should have exactly one boolean value after execution.
	EvaluateCondition(bc *dtrules.BytecodeChunk) (bool, error)

	// --- State Management ---

	// Reset clears all stack state but keeps the context usable.
	// This is useful for reusing a context across multiple executions.
	Reset() error

	// GetError returns the last error that occurred during execution.
	// Returns nil if no error has occurred since the last Reset.
	GetError() error

	// Close releases resources associated with this context.
	// After Close, no other methods should be called.
	Close() error
}

// ContextOption configures an ExecutionContext.
type ContextOption func(*ContextConfig)

// ContextConfig holds configuration for context creation.
type ContextConfig struct {
	DataStackSize   int
	EntityStackSize int
	EnableTracing   bool
	TraceWriter     TraceWriter
}

// DefaultContextConfig returns the default configuration.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		DataStackSize:   1024,
		EntityStackSize: 64,
		EnableTracing:   false,
		TraceWriter:     nil,
	}
}

// ApplyOptions applies options to a config.
func (c *ContextConfig) ApplyOptions(opts []ContextOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithDataStackSize sets the initial data stack capacity.
func WithDataStackSize(size int) ContextOption {
	return func(c *ContextConfig) {
		c.DataStackSize = size
	}
}

// WithEntityStackSize sets the initial entity stack capacity.
func WithEntityStackSize(size int) ContextOption {
	return func(c *ContextConfig) {
		c.EntityStackSize = size
	}
}

// WithTracing enables execution tracing with the given writer.
func WithTracing(w TraceWriter) ContextOption {
	return func(c *ContextConfig) {
		c.EnableTracing = true
		c.TraceWriter = w
	}
}

// TraceWriter receives trace events during execution.
type TraceWriter interface {
	// TraceInstruction is called before each bytecode instruction.
	TraceInstruction(pc int, op dtrules.Opcode, stackDepth int)

	// TraceOperation is called when an operator is executed.
	TraceOperation(name string, args []dtrules.Value, result dtrules.Value)

	// TraceError is called when an error occurs.
	TraceError(err error)
}

// Error types for runtime operations.
var (
	// ErrStackOverflow indicates the data stack exceeded its maximum depth.
	ErrStackOverflow = &RuntimeError{Code: 1, Message: "stack overflow"}

	// ErrStackUnderflow indicates a pop was attempted on an empty stack.
	ErrStackUnderflow = &RuntimeError{Code: 2, Message: "stack underflow"}

	// ErrTypeMismatch indicates an operation received incompatible types.
	ErrTypeMismatch = &RuntimeError{Code: 3, Message: "type mismatch"}

	// ErrDivisionByZero indicates division or modulo by zero.
	ErrDivisionByZero = &RuntimeError{Code: 4, Message: "division by zero"}

	// ErrOutOfMemory indicates memory allocation failed.
	ErrOutOfMemory = &RuntimeError{Code: 5, Message: "out of memory"}

	// ErrInvalidOpcode indicates an unknown bytecode instruction.
	ErrInvalidOpcode = &RuntimeError{Code: 6, Message: "invalid opcode"}

	// ErrIndexOutOfBounds indicates an array index was out of bounds.
	ErrIndexOutOfBounds = &RuntimeError{Code: 7, Message: "index out of bounds"}

	// ErrNameNotFound indicates a variable name was not found in context.
	ErrNameNotFound = &RuntimeError{Code: 8, Message: "name not found"}

	// ErrEntityStackUnderflow indicates the entity stack is empty.
	ErrEntityStackUnderflow = &RuntimeError{Code: 9, Message: "entity stack underflow"}

	// ErrInvalidEntity indicates an invalid entity reference.
	ErrInvalidEntity = &RuntimeError{Code: 10, Message: "invalid entity"}

	// ErrContextClosed indicates the context has been closed.
	ErrContextClosed = &RuntimeError{Code: 11, Message: "context closed"}

	// ErrRuntimeClosed indicates the runtime has been closed.
	ErrRuntimeClosed = &RuntimeError{Code: 12, Message: "runtime closed"}
)

// RuntimeError represents an error from the runtime.
type RuntimeError struct {
	Code    int
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *RuntimeError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *RuntimeError) Unwrap() error {
	return e.Cause
}

// WithCause returns a new error with the given cause.
func (e *RuntimeError) WithCause(cause error) *RuntimeError {
	return &RuntimeError{
		Code:    e.Code,
		Message: e.Message,
		Cause:   cause,
	}
}

// NewRuntimeError creates a new runtime error with a custom message.
func NewRuntimeError(code int, message string) *RuntimeError {
	return &RuntimeError{Code: code, Message: message}
}

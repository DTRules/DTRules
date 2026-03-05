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

package goruntime

import (
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime"
)

func TestGoRuntimeName(t *testing.T) {
	rt := New()
	defer rt.Close()

	if rt.Name() != "go" {
		t.Errorf("expected name 'go', got '%s'", rt.Name())
	}
}

func TestGoRuntimeVersion(t *testing.T) {
	rt := New()
	defer rt.Close()

	if rt.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", rt.Version())
	}
}

func TestGoRuntimeCapabilities(t *testing.T) {
	rt := New()
	defer rt.Close()

	caps := rt.Capabilities()

	if !caps.ConcurrentContexts {
		t.Error("expected ConcurrentContexts true")
	}
	if !caps.Tracing {
		t.Error("expected Tracing true")
	}
	if !caps.SupportsAllOperators {
		t.Error("expected SupportsAllOperators true")
	}
	if caps.MaxStackDepth != 1000 {
		t.Errorf("expected MaxStackDepth 1000, got %d", caps.MaxStackDepth)
	}
}

func TestGoRuntimeCreateContext(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	if ctx.StackDepth() != 0 {
		t.Errorf("expected empty stack, got depth %d", ctx.StackDepth())
	}
}

func TestGoContextPushPop(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Push values
	if err := ctx.Push(dtrules.NewValueInteger(42)); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if err := ctx.Push(dtrules.NewValueDouble(3.14)); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if ctx.StackDepth() != 2 {
		t.Errorf("expected stack depth 2, got %d", ctx.StackDepth())
	}

	// Pop and verify
	v, err := ctx.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if !v.IsDouble() || v.AsDouble() != 3.14 {
		t.Errorf("expected double 3.14, got %v", v)
	}

	v, err = ctx.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if !v.IsInteger() || v.AsInteger() != 42 {
		t.Errorf("expected integer 42, got %v", v)
	}

	if ctx.StackDepth() != 0 {
		t.Errorf("expected empty stack, got depth %d", ctx.StackDepth())
	}
}

func TestGoContextPeek(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	if err := ctx.Push(dtrules.NewValueString("hello")); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Peek should not remove the value
	v, err := ctx.Peek()
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if !v.IsString() || v.AsString() != "hello" {
		t.Errorf("expected string 'hello', got %v", v)
	}

	if ctx.StackDepth() != 1 {
		t.Errorf("expected stack depth 1 after Peek, got %d", ctx.StackDepth())
	}
}

func TestGoContextReset(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Push some values
	ctx.Push(dtrules.NewValueInteger(1))
	ctx.Push(dtrules.NewValueInteger(2))
	ctx.Push(dtrules.NewValueInteger(3))

	if ctx.StackDepth() != 3 {
		t.Errorf("expected stack depth 3, got %d", ctx.StackDepth())
	}

	// Reset
	if err := ctx.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	if ctx.StackDepth() != 0 {
		t.Errorf("expected empty stack after Reset, got depth %d", ctx.StackDepth())
	}
}

func TestGoContextExecuteBytecode(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Create simple bytecode: push 10, push 20, add
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(20))
	bc.Emit(dtrules.OpAdd)

	if err := ctx.ExecuteBytecode(bc); err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	// Verify result
	v, err := ctx.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	// Result could be integer or double depending on implementation
	if v.IsInteger() {
		if v.AsInteger() != 30 {
			t.Errorf("expected 30, got %d", v.AsInteger())
		}
	} else if v.IsDouble() {
		if v.AsDouble() != 30.0 {
			t.Errorf("expected 30.0, got %f", v.AsDouble())
		}
	} else {
		t.Errorf("expected numeric result, got %v", v)
	}
}

func TestGoContextEvaluateCondition(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Create bytecode: push 10, push 5, gt (10 > 5 = true)
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(10))
	bc.EmitPushConstant(dtrules.NewValueInteger(5))
	bc.Emit(dtrules.OpGt)

	result, err := ctx.EvaluateCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}

	if !result {
		t.Error("expected true, got false")
	}

	// Stack should be clean after EvaluateCondition
	if ctx.StackDepth() != 0 {
		t.Errorf("expected empty stack after EvaluateCondition, got depth %d", ctx.StackDepth())
	}
}

func TestGoContextClose(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}

	// Close the context
	if err := ctx.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations after close should fail
	if err := ctx.Push(dtrules.NewValueInteger(1)); err != runtime.ErrContextClosed {
		t.Errorf("expected ErrContextClosed, got %v", err)
	}

	_, err = ctx.Pop()
	if err != runtime.ErrContextClosed {
		t.Errorf("expected ErrContextClosed, got %v", err)
	}
}

func TestGoRuntimeClose(t *testing.T) {
	rt := New()

	// Close the runtime
	if err := rt.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// CreateContext after close should fail
	_, err := rt.CreateContext()
	if err != runtime.ErrRuntimeClosed {
		t.Errorf("expected ErrRuntimeClosed, got %v", err)
	}
}

func TestGoContextConcurrency(t *testing.T) {
	rt := New()
	defer rt.Close()

	// Create multiple contexts
	ctx1, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext 1 failed: %v", err)
	}
	defer ctx1.Close()

	ctx2, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext 2 failed: %v", err)
	}
	defer ctx2.Close()

	// Operations on each context should be independent
	ctx1.Push(dtrules.NewValueInteger(100))
	ctx2.Push(dtrules.NewValueInteger(200))

	if ctx1.StackDepth() != 1 || ctx2.StackDepth() != 1 {
		t.Error("contexts should have independent stacks")
	}

	v1, _ := ctx1.Pop()
	v2, _ := ctx2.Pop()

	if v1.AsInteger() != 100 {
		t.Errorf("ctx1 expected 100, got %d", v1.AsInteger())
	}
	if v2.AsInteger() != 200 {
		t.Errorf("ctx2 expected 200, got %d", v2.AsInteger())
	}
}

func TestGoContextEmptyStackOperations(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Pop from empty stack should fail
	_, err = ctx.Pop()
	if err == nil {
		t.Error("expected error popping from empty stack")
	}

	// Peek on empty stack should fail
	_, err = ctx.Peek()
	if err == nil {
		t.Error("expected error peeking empty stack")
	}
}

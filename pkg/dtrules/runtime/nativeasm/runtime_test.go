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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

// newTestEntity creates a test entity with an attribute "x" set to the given integer value.
func newTestEntity(name string, id int) *entity.REntity {
	e := entity.NewREntity(id, false, dtrules.GetRName(name))
	return e
}

// newTestEntityWithAttr creates a test entity with a named integer attribute.
func newTestEntityWithAttr(name string, id int, attrName string, value int64) *entity.REntity {
	e := entity.NewREntity(id, false, dtrules.GetRName(name))
	attr := dtrules.GetRName(attrName)
	e.AddAttribute(attr, "0", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	e.Put(attr, dtrules.GetRIntegerValue(value))
	return e
}

func TestRuntimeBasics(t *testing.T) {
	rt := New()
	defer rt.Close()

	if rt.Name() != "native-asm" {
		t.Errorf("expected name 'native-asm', got '%s'", rt.Name())
	}

	caps := rt.Capabilities()
	if !caps.ConcurrentContexts {
		t.Error("expected ConcurrentContexts to be true")
	}
	if caps.MaxStackDepth != interpreter.MaxStackDepth {
		t.Errorf("expected MaxStackDepth %d, got %d", interpreter.MaxStackDepth, caps.MaxStackDepth)
	}
}

func TestContextPushPop(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	err = ctx.Push(dtrules.NewValueInteger(42))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if ctx.StackDepth() != 1 {
		t.Errorf("expected depth 1, got %d", ctx.StackDepth())
	}

	v, err := ctx.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	if v.AsInteger() != 42 {
		t.Errorf("expected 42, got %d", v.AsInteger())
	}
}

func TestContextExecuteBytecode(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Push values, then execute add
	err = ctx.Push(dtrules.NewValueInteger(10))
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.Push(dtrules.NewValueInteger(32))
	if err != nil {
		t.Fatal(err)
	}

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpAdd)

	err = ctx.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("ExecuteBytecode failed: %v", err)
	}

	result, err := ctx.Pop()
	if err != nil {
		t.Fatal(err)
	}

	if result.AsInteger() != 42 {
		t.Errorf("expected 42, got %d", result.AsInteger())
	}
}

func TestContextEvaluateCondition(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	// Bytecode must produce its own result for EvaluateCondition
	bc := dtrules.NewBytecodeChunk()
	bc.EmitWithArg(dtrules.OpPushInt, 5)
	bc.EmitWithArg(dtrules.OpPushInt, 3)
	bc.Emit(dtrules.OpGt)

	result, err := ctx.EvaluateCondition(bc)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}

	if !result {
		t.Error("expected true, got false")
	}
}

func TestContextEntityStack(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	defer ctx.Close()

	e := newTestEntityWithAttr("test", 1, "x", 100)
	attrX := dtrules.GetRName("x")

	err = ctx.EntityPush(e)
	if err != nil {
		t.Fatalf("EntityPush failed: %v", err)
	}

	if ctx.EntityDepth() != 1 {
		t.Errorf("expected entity depth 1, got %d", ctx.EntityDepth())
	}

	found, ok := ctx.FindEntity(attrX)
	if !ok {
		t.Fatal("expected to find entity with attribute x")
	}
	if found.GetID() != 1 {
		t.Errorf("expected entity ID 1, got %d", found.GetID())
	}

	v, err := ctx.Lookup(attrX)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if v.AsInteger() != 100 {
		t.Errorf("expected 100, got %d", v.AsInteger())
	}

	err = ctx.Def(attrX, dtrules.NewValueInteger(200))
	if err != nil {
		t.Fatalf("Def failed: %v", err)
	}

	v, err = ctx.Lookup(attrX)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsInteger() != 200 {
		t.Errorf("expected 200, got %d", v.AsInteger())
	}

	popped, err := ctx.EntityPop()
	if err != nil {
		t.Fatal(err)
	}
	if popped.GetID() != 1 {
		t.Errorf("expected entity ID 1, got %d", popped.GetID())
	}
}

func TestContextReset(t *testing.T) {
	rt := New()
	defer rt.Close()

	ctx, err := rt.CreateContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ctx.Push(dtrules.NewValueInteger(1))
	ctx.Push(dtrules.NewValueInteger(2))
	ctx.EntityPush(newTestEntity("test", 1))

	err = ctx.Reset()
	if err != nil {
		t.Fatal(err)
	}

	if ctx.StackDepth() != 0 {
		t.Errorf("expected stack depth 0, got %d", ctx.StackDepth())
	}
	if ctx.EntityDepth() != 0 {
		t.Errorf("expected entity depth 0, got %d", ctx.EntityDepth())
	}
}

func TestContextPooling(t *testing.T) {
	rt := New()
	defer rt.Close()

	for i := 0; i < 10; i++ {
		ctx, err := rt.CreateContext()
		if err != nil {
			t.Fatal(err)
		}

		ctx.Push(dtrules.NewValueInteger(int64(i)))

		err = ctx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkRuntimePushPop(b *testing.B) {
	rt := New()
	defer rt.Close()

	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	v := dtrules.NewValueInteger(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Push(v)
		ctx.Pop()
	}
}

func BenchmarkRuntimeExecute(b *testing.B) {
	rt := New()
	defer rt.Close()

	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)
	bc.Emit(dtrules.OpPushOne)
	bc.Emit(dtrules.OpAdd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

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

//go:build integration
// +build integration

package runtime_test

import (
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/asmruntime"
	"github.com/DTRules/DTRules/go/pkg/dtrules/runtime/goruntime"
)

// Benchmark just push/pop to measure CGO overhead
func BenchmarkGoPushPop(b *testing.B) {
	rt := goruntime.New()
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

func BenchmarkASMPushPop(b *testing.B) {
	if err := asmruntime.Init(); err != nil {
		b.Skip("ASM not available")
	}
	asmruntime.Reset()

	v := dtrules.NewValueInteger(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asmruntime.PushValue(v)
		asmruntime.PopValue()
	}
}

// Benchmark just the execute call with minimal bytecode
func BenchmarkGoExecuteMinimal(b *testing.B) {
	rt := goruntime.New()
	defer rt.Close()
	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	// Just push one value
	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

func BenchmarkASMExecuteMinimal(b *testing.B) {
	if err := asmruntime.Init(); err != nil {
		b.Skip("ASM not available")
	}

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asmruntime.Reset()
		asmruntime.ExecuteBytecode(bc)
		asmruntime.PopValue()
	}
}

// Benchmark longer computation to amortize CGO overhead
func BenchmarkGoLongComputation(b *testing.B) {
	rt := goruntime.New()
	defer rt.Close()
	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	// 100 operations: push 1, then add 1 ninety-nine times
	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)
	for i := 0; i < 99; i++ {
		bc.Emit(dtrules.OpPushOne)
		bc.Emit(dtrules.OpAdd)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

func BenchmarkASMLongComputation(b *testing.B) {
	if err := asmruntime.Init(); err != nil {
		b.Skip("ASM not available")
	}

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)
	for i := 0; i < 99; i++ {
		bc.Emit(dtrules.OpPushOne)
		bc.Emit(dtrules.OpAdd)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asmruntime.Reset()
		asmruntime.ExecuteBytecode(bc)
		asmruntime.PopValue()
	}
}

// Very long computation to really amortize CGO
func BenchmarkGoVeryLongComputation(b *testing.B) {
	rt := goruntime.New()
	defer rt.Close()
	ctx, _ := rt.CreateContext()
	defer ctx.Close()

	// 1000 operations
	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)
	for i := 0; i < 999; i++ {
		bc.Emit(dtrules.OpPushOne)
		bc.Emit(dtrules.OpAdd)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExecuteBytecode(bc)
		ctx.Pop()
	}
}

func BenchmarkASMVeryLongComputation(b *testing.B) {
	if err := asmruntime.Init(); err != nil {
		b.Skip("ASM not available")
	}

	bc := dtrules.NewBytecodeChunk()
	bc.Emit(dtrules.OpPushOne)
	for i := 0; i < 999; i++ {
		bc.Emit(dtrules.OpPushOne)
		bc.Emit(dtrules.OpAdd)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		asmruntime.Reset()
		asmruntime.ExecuteBytecode(bc)
		asmruntime.PopValue()
	}
}

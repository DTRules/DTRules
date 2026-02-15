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

package benchmark

import (
	"os"
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/compiler"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/operators"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// Paths to CHIP sample project
const (
	chipEDD = "../../../../sampleprojects/CHIP/repository/xml/CHIP_edd.xml"
	chipDT  = "../../../../sampleprojects/CHIP/repository/xml/CHIP_dt.xml"
)

// mockSession for micro-benchmarks
type mockSession struct {
	uniqueID int
}

func (m *mockSession) GetState() dtrules.State                                  { return nil }
func (m *mockSession) GetRuntime() dtrules.Runtime                              { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory                  { return nil }
func (m *mockSession) GetUniqueID() int                                         { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser                        { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet                              { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) { return nil, nil }
func (m *mockSession) Compile(expr string) (dtrules.Object, error)              { return nil, nil }

func newTestState() *interpreter.DTState {
	return interpreter.NewDTState(&mockSession{})
}

// =============================================================================
// MICRO-BENCHMARKS: Individual Operations
// =============================================================================

// BenchmarkNameLookup measures the cost of name interning/lookup
func BenchmarkNameLookup(b *testing.B) {
	names := []string{"foo", "bar", "baz", "client", "income", "age", "result"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, n := range names {
			_ = dtrules.GetRName(n)
		}
	}
}

// BenchmarkNameLookupCached measures lookup of already-cached names
func BenchmarkNameLookupCached(b *testing.B) {
	// Pre-cache the names
	names := []*dtrules.RName{
		dtrules.GetRName("foo"),
		dtrules.GetRName("bar"),
		dtrules.GetRName("baz"),
		dtrules.GetRName("client"),
		dtrules.GetRName("income"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, n := range names {
			_ = n.StringValue()
		}
	}
}

// BenchmarkOperatorLookup measures operator dispatch by name (map lookup)
func BenchmarkOperatorLookup(b *testing.B) {
	opNames := []*dtrules.RName{
		dtrules.GetRName("+"),
		dtrules.GetRName("-"),
		dtrules.GetRName("*"),
		dtrules.GetRName("/"),
		dtrules.GetRName("and"),
		dtrules.GetRName("or"),
		dtrules.GetRName("not"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range opNames {
			operators.Get(name)
		}
	}
}

// BenchmarkOperatorLookupIndexed measures operator dispatch by index (array lookup)
func BenchmarkOperatorLookupIndexed(b *testing.B) {
	// Get indices for operators
	opIndices := make([]int, 7)
	opIndices[0], _ = operators.GetIndex(dtrules.GetRName("+"))
	opIndices[1], _ = operators.GetIndex(dtrules.GetRName("-"))
	opIndices[2], _ = operators.GetIndex(dtrules.GetRName("*"))
	opIndices[3], _ = operators.GetIndex(dtrules.GetRName("/"))
	opIndices[4], _ = operators.GetIndex(dtrules.GetRName("and"))
	opIndices[5], _ = operators.GetIndex(dtrules.GetRName("or"))
	opIndices[6], _ = operators.GetIndex(dtrules.GetRName("not"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, idx := range opIndices {
			operators.GetByIndex(idx)
		}
	}
}

// BenchmarkOperatorDirect measures direct function pointer call (no lookup)
func BenchmarkOperatorDirect(b *testing.B) {
	// Get operator pointers directly
	ops := make([]*operators.Operator, 7)
	ops[0], _ = operators.Get(dtrules.GetRName("+"))
	ops[1], _ = operators.Get(dtrules.GetRName("-"))
	ops[2], _ = operators.Get(dtrules.GetRName("*"))
	ops[3], _ = operators.Get(dtrules.GetRName("/"))
	ops[4], _ = operators.Get(dtrules.GetRName("and"))
	ops[5], _ = operators.Get(dtrules.GetRName("or"))
	ops[6], _ = operators.Get(dtrules.GetRName("not"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, op := range ops {
			_ = op // Just access the pointer - no lookup
		}
	}
}

// BenchmarkDataStackPushPop measures stack operations
func BenchmarkDataStackPushPop(b *testing.B) {
	state := newTestState()
	val := dtrules.GetRIntegerValue(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(val)
		state.DataPop()
	}
}

// BenchmarkIntegerArithmetic measures basic math operations (Object-based)
func BenchmarkIntegerArithmetic(b *testing.B) {
	state := newTestState()
	addOp, _ := operators.Get(dtrules.GetRName("+"))
	val1 := dtrules.GetRIntegerValue(100)
	val2 := dtrules.GetRIntegerValue(200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(val1)
		state.DataPush(val2)
		addOp.Execute(state)
		state.DataPop()
	}
}

// BenchmarkValueArithmetic measures Value-based arithmetic (optimized)
func BenchmarkValueArithmetic(b *testing.B) {
	val1 := dtrules.NewValueInteger(100)
	val2 := dtrules.NewValueInteger(200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := val1.Add(val2)
		_ = result
	}
}

// BenchmarkValueArithmeticComplex measures a complex calculation with Values
func BenchmarkValueArithmeticComplex(b *testing.B) {
	a := dtrules.NewValueInteger(100)
	c := dtrules.NewValueInteger(2)
	threshold := dtrules.NewValueInteger(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bv := dtrules.NewValueInteger(int64(i % 100))
		// (a + b) * c < threshold
		sum := a.Add(bv)
		product := sum.Mul(c)
		result := product.Less(threshold)
		_ = result
	}
}

// BenchmarkValueCreation measures Value creation (no allocation)
func BenchmarkValueCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dtrules.NewValueInteger(int64(i))
	}
}

// BenchmarkValueToObject measures Value to Object conversion
func BenchmarkValueToObject(b *testing.B) {
	v := dtrules.NewValueInteger(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.AsObject()
	}
}

// BenchmarkIntegerCreation measures integer object creation
func BenchmarkIntegerCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Small integers should be cached
		_ = dtrules.GetRIntegerValue(int64(i % 256))
	}
}

// BenchmarkIntegerCreationLarge measures large integer creation (not cached)
func BenchmarkIntegerCreationLarge(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dtrules.GetRIntegerValue(int64(i + 1000000))
	}
}

// BenchmarkStringCreation measures string object creation (now with interning)
func BenchmarkStringCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dtrules.NewRString("test string value")
	}
}

// BenchmarkStringCreationCached measures interned string lookup
func BenchmarkStringCreationCached(b *testing.B) {
	// Pre-intern the strings
	strings := []string{"foo", "bar", "baz", "client", "income", "age", "result"}
	for _, s := range strings {
		dtrules.GetRString(s)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range strings {
			_ = dtrules.GetRString(s)
		}
	}
}

// BenchmarkStringCreationUnique measures unique (non-interned) string creation
func BenchmarkStringCreationUnique(b *testing.B) {
	// Create strings longer than MaxInternLength to avoid interning
	longPrefix := "this_is_a_very_long_string_prefix_that_exceeds_the_intern_limit_"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dtrules.NewRString(longPrefix + string(rune('a'+i%26)))
	}
}

// BenchmarkBooleanOperations measures boolean logic
func BenchmarkBooleanOperations(b *testing.B) {
	state := newTestState()
	andOp, _ := operators.Get(dtrules.GetRName("and"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(dtrules.True)
		state.DataPush(dtrules.False)
		andOp.Execute(state)
		state.DataPop()
	}
}

// BenchmarkComparison measures comparison operations
func BenchmarkComparison(b *testing.B) {
	state := newTestState()
	ltOp, _ := operators.Get(dtrules.GetRName("<"))
	val1 := dtrules.GetRIntegerValue(50)
	val2 := dtrules.GetRIntegerValue(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(val1)
		state.DataPush(val2)
		ltOp.Execute(state)
		state.DataPop()
	}
}

// BenchmarkStringConcat measures string concatenation
func BenchmarkStringConcat(b *testing.B) {
	state := newTestState()
	concatOp, _ := operators.Get(dtrules.GetRName("concat"))
	s1 := dtrules.NewRString("Hello ")
	s2 := dtrules.NewRString("World")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(s1)
		state.DataPush(s2)
		concatOp.Execute(state)
		state.DataPop()
	}
}

// =============================================================================
// INTEGRATION BENCHMARKS: Full Rule Execution
// =============================================================================

// loadCHIPRuleSet loads the CHIP sample project
func loadCHIPRuleSet(b *testing.B) *session.RuleSet {
	rs := session.NewRuleSet("CHIP")

	eddFile, err := os.Open(chipEDD)
	if err != nil {
		b.Skipf("Could not open CHIP EDD: %v", err)
		return nil
	}
	defer eddFile.Close()

	if err := rs.LoadEDD(eddFile); err != nil {
		b.Skipf("Could not load CHIP EDD: %v", err)
		return nil
	}

	dtFile, err := os.Open(chipDT)
	if err != nil {
		b.Skipf("Could not open CHIP DT: %v", err)
		return nil
	}
	defer dtFile.Close()

	if err := rs.LoadDecisionTables(dtFile); err != nil {
		b.Skipf("Could not load CHIP DT: %v", err)
		return nil
	}

	return rs
}

// BenchmarkRuleSetLoad measures time to load rules
func BenchmarkRuleSetLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rs := session.NewRuleSet("CHIP")

		eddFile, err := os.Open(chipEDD)
		if err != nil {
			b.Skipf("Could not open CHIP EDD: %v", err)
			return
		}

		if err := rs.LoadEDD(eddFile); err != nil {
			eddFile.Close()
			b.Skipf("Could not load CHIP EDD: %v", err)
			return
		}
		eddFile.Close()

		dtFile, err := os.Open(chipDT)
		if err != nil {
			b.Skipf("Could not open CHIP DT: %v", err)
			return
		}

		if err := rs.LoadDecisionTables(dtFile); err != nil {
			dtFile.Close()
			b.Skipf("Could not load CHIP DT: %v", err)
			return
		}
		dtFile.Close()
	}
}

// BenchmarkSessionCreate measures session creation time
func BenchmarkSessionCreate(b *testing.B) {
	rs := loadCHIPRuleSet(b)
	if rs == nil {
		return
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.NewSession()
	}
}

// BenchmarkDecisionTableExecution measures single table execution
func BenchmarkDecisionTableExecution(b *testing.B) {
	rs := loadCHIPRuleSet(b)
	if rs == nil {
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sess, _ := rs.NewSession()
		rsess := sess.(*session.RSession)
		// Execute without data - measures table traversal overhead
		rsess.Execute("Compute_Eligibility")
	}
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

// BenchmarkMemoryIntegerAlloc measures integer allocation
func BenchmarkMemoryIntegerAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dtrules.GetRIntegerValue(int64(i + 1000000))
	}
}

// BenchmarkMemoryStringAlloc measures string allocation
func BenchmarkMemoryStringAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dtrules.NewRString("test string")
	}
}

// BenchmarkMemoryNameAlloc measures name interning allocation
func BenchmarkMemoryNameAlloc(b *testing.B) {
	b.ReportAllocs()
	// Use unique names to force allocation
	for i := 0; i < b.N; i++ {
		_ = dtrules.GetRName("unique_name_that_wont_be_cached_" + string(rune(i%26+'a')))
	}
}

// BenchmarkMemoryStackOps measures stack operation allocation
func BenchmarkMemoryStackOps(b *testing.B) {
	state := newTestState()
	val := dtrules.GetRIntegerValue(42)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.DataPush(val)
		state.DataPop()
	}
}

// =============================================================================
// COMPLEX SCENARIO: Simulated Heavy Load
// =============================================================================

// BenchmarkComplexScenario simulates processing many records
func BenchmarkComplexScenario(b *testing.B) {
	state := newTestState()

	// Get operators we'll use
	addOp, _ := operators.Get(dtrules.GetRName("+"))
	mulOp, _ := operators.Get(dtrules.GetRName("*"))
	ltOp, _ := operators.Get(dtrules.GetRName("<"))
	andOp, _ := operators.Get(dtrules.GetRName("and"))

	// Simulate a calculation: (a + b) * c < threshold AND flag
	a := dtrules.GetRIntegerValue(100)
	c := dtrules.GetRIntegerValue(2)
	threshold := dtrules.GetRIntegerValue(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b := dtrules.GetRIntegerValue(int64(i % 100))

		// a + b
		state.DataPush(a)
		state.DataPush(b)
		addOp.Execute(state)

		// * c
		state.DataPush(c)
		mulOp.Execute(state)

		// < threshold
		state.DataPush(threshold)
		ltOp.Execute(state)

		// AND true
		state.DataPush(dtrules.True)
		andOp.Execute(state)

		state.DataPop()
	}
}

// BenchmarkBytecodeEmit measures bytecode emission speed
func BenchmarkBytecodeEmit(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bc := dtrules.NewBytecodeChunk()
		// Emit: (100 + i) * 2 < 1000
		bc.EmitPushConstant(dtrules.NewValueInteger(100))
		bc.EmitPushConstant(dtrules.NewValueInteger(int64(i % 100)))
		bc.Emit(dtrules.OpAdd)
		bc.EmitPushConstant(dtrules.NewValueInteger(2))
		bc.Emit(dtrules.OpMul)
		bc.EmitPushConstant(dtrules.NewValueInteger(1000))
		bc.Emit(dtrules.OpLt)
	}
}

// BenchmarkBytecodeRead measures bytecode reading speed
func BenchmarkBytecodeRead(b *testing.B) {
	// Pre-compile bytecode
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(100))
	bc.EmitPushConstant(dtrules.NewValueInteger(50))
	bc.Emit(dtrules.OpAdd)
	bc.EmitPushConstant(dtrules.NewValueInteger(2))
	bc.Emit(dtrules.OpMul)
	bc.EmitPushConstant(dtrules.NewValueInteger(1000))
	bc.Emit(dtrules.OpLt)
	code := bc.Code()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := dtrules.NewBytecodeReader(code)
		for reader.HasMore() {
			op := reader.ReadOpcode()
			switch op {
			case dtrules.OpPushInt, dtrules.OpConstant, dtrules.OpOperator:
				reader.ReadVarint()
			}
		}
	}
}

// BenchmarkBytecodeVMExecution measures bytecode VM execution
func BenchmarkBytecodeVMExecution(b *testing.B) {
	state := newTestState()

	// Compile: (100 + 50) * 2 < 1000
	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueInteger(100))
	bc.EmitPushConstant(dtrules.NewValueInteger(50))
	bc.Emit(dtrules.OpAdd)
	bc.EmitPushConstant(dtrules.NewValueInteger(2))
	bc.Emit(dtrules.OpMul)
	bc.EmitPushConstant(dtrules.NewValueInteger(1000))
	bc.Emit(dtrules.OpLt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.ExecuteBytecode(bc)
		state.ValuePop() // Consume result
	}
}

// BenchmarkBytecodeCompile measures bytecode compilation speed
func BenchmarkBytecodeCompile(b *testing.B) {
	factory := entity.NewFactory(nil)
	comp := compiler.NewCompiler(&mockSession{}, factory)
	expr := "100 50 + 2 * 1000 <"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp.CompileToBytecode(expr)
	}
}

// BenchmarkObjectCompile measures Object-based compilation speed
func BenchmarkObjectCompile(b *testing.B) {
	factory := entity.NewFactory(nil)
	comp := compiler.NewCompiler(&mockSession{}, factory)
	expr := "100 50 + 2 * 1000 <"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp.Compile(expr)
	}
}

// BenchmarkBytecodeFullPath measures compile + execute for bytecode
func BenchmarkBytecodeFullPath(b *testing.B) {
	state := newTestState()
	factory := entity.NewFactory(nil)
	comp := compiler.NewCompiler(&mockSession{}, factory)
	expr := "100 50 + 2 * 1000 <"

	// Pre-compile
	bc, _ := comp.CompileToBytecode(expr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.ExecuteBytecode(bc)
		state.ValuePop()
	}
}

// BenchmarkObjectFullPath measures compile + execute for Object arrays
func BenchmarkObjectFullPath(b *testing.B) {
	state := newTestState()
	factory := entity.NewFactory(nil)
	comp := compiler.NewCompiler(&mockSession{}, factory)
	expr := "100 50 + 2 * 1000 <"

	// Pre-compile
	code, _ := comp.Compile(expr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code.Execute(state)
		state.DataPop()
	}
}

// BenchmarkBytecodeSize compares memory usage
func BenchmarkBytecodeSize(b *testing.B) {
	// Compare bytecode size vs Object array size
	bc := dtrules.NewBytecodeChunk()
	for i := 0; i < 100; i++ {
		bc.EmitPushConstant(dtrules.NewValueInteger(int64(i)))
		bc.Emit(dtrules.OpAdd)
	}
	b.ReportMetric(float64(bc.Size()), "bytecode_bytes")

	// Object array would be approximately 100 * 16 bytes (interface = 2 words)
	b.ReportMetric(float64(100*16), "object_array_bytes")
}

// BenchmarkManyOperations simulates executing many different operations
func BenchmarkManyOperations(b *testing.B) {
	state := newTestState()

	// Cache operators
	ops := make([]dtrules.Object, 0)
	opNames := []string{"+", "-", "*", "/", "and", "or", "not", "dup", "swap", "pop"}
	for _, name := range opNames {
		if op, ok := operators.Get(dtrules.GetRName(name)); ok {
			ops = append(ops, op)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Push some values
		state.DataPush(dtrules.GetRIntegerValue(10))
		state.DataPush(dtrules.GetRIntegerValue(20))
		state.DataPush(dtrules.True)

		// Execute various operations
		for j := 0; j < 10; j++ {
			op := ops[j%len(ops)]
			op.Execute(state)
			// Reset stack for next iteration
			if state.DataStackDepth() < 2 {
				state.DataPush(dtrules.GetRIntegerValue(10))
				state.DataPush(dtrules.GetRIntegerValue(20))
			}
		}

		// Clear stack
		for state.DataStackDepth() > 0 {
			state.DataPop()
		}
	}
}

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

package interpreter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// =============================================================================
// Value convenience method tests
// =============================================================================

func TestValuePushInt(t *testing.T) {
	state := NewDTState(&mockSession{})
	err := state.ValuePushInt(42)
	if err != nil {
		t.Fatalf("ValuePushInt failed: %v", err)
	}
	v, err := state.ValuePop()
	if err != nil {
		t.Fatalf("ValuePop failed: %v", err)
	}
	if !v.IsInteger() || v.AsInteger() != 42 {
		t.Errorf("expected integer 42, got %v", v)
	}
}

func TestValuePushDouble(t *testing.T) {
	state := NewDTState(&mockSession{})
	err := state.ValuePushDouble(3.14)
	if err != nil {
		t.Fatalf("ValuePushDouble failed: %v", err)
	}
	v, err := state.ValuePop()
	if err != nil {
		t.Fatalf("ValuePop failed: %v", err)
	}
	if !v.IsDouble() || v.AsDouble() != 3.14 {
		t.Errorf("expected double 3.14, got %v", v)
	}
}

func TestValuePushBool(t *testing.T) {
	state := NewDTState(&mockSession{})
	err := state.ValuePushBool(true)
	if err != nil {
		t.Fatalf("ValuePushBool failed: %v", err)
	}
	v, err := state.ValuePop()
	if err != nil {
		t.Fatalf("ValuePop failed: %v", err)
	}
	if !v.IsBoolean() || !v.AsBoolean() {
		t.Errorf("expected true, got %v", v)
	}
}

func TestValuePopInt(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushInt(99)

	val, err := state.ValuePopInt()
	if err != nil {
		t.Fatalf("ValuePopInt failed: %v", err)
	}
	if val != 99 {
		t.Errorf("expected 99, got %d", val)
	}
}

func TestValuePopIntTypeError(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushDouble(3.14)

	_, err := state.ValuePopInt()
	if err == nil {
		t.Error("expected error when popping non-integer as int")
	}
}

func TestValuePopIntUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.ValuePopInt()
	if err == nil {
		t.Error("expected underflow error")
	}
}

func TestValuePopDouble(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushDouble(2.71)

	val, err := state.ValuePopDouble()
	if err != nil {
		t.Fatalf("ValuePopDouble failed: %v", err)
	}
	if val != 2.71 {
		t.Errorf("expected 2.71, got %f", val)
	}
}

func TestValuePopDoubleFromInt(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushInt(5)

	val, err := state.ValuePopDouble()
	if err != nil {
		t.Fatalf("ValuePopDouble from int failed: %v", err)
	}
	if val != 5.0 {
		t.Errorf("expected 5.0, got %f", val)
	}
}

func TestValuePopDoubleTypeError(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushBool(true)

	_, err := state.ValuePopDouble()
	if err == nil {
		t.Error("expected error when popping boolean as double")
	}
}

func TestValuePopDoubleUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.ValuePopDouble()
	if err == nil {
		t.Error("expected underflow error")
	}
}

func TestValuePopBool(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushBool(false)

	val, err := state.ValuePopBool()
	if err != nil {
		t.Fatalf("ValuePopBool failed: %v", err)
	}
	if val != false {
		t.Errorf("expected false, got %t", val)
	}
}

func TestValuePopBoolTypeError(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushInt(1)

	_, err := state.ValuePopBool()
	if err == nil {
		t.Error("expected error when popping integer as bool")
	}
}

func TestValuePopBoolUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.ValuePopBool()
	if err == nil {
		t.Error("expected underflow error")
	}
}

// =============================================================================
// ValueClear tests
// =============================================================================

func TestValueClear(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValuePushInt(1)
	state.ValuePushInt(2)
	state.ValuePushInt(3)

	if state.ValueStackDepth() != 3 {
		t.Fatalf("expected depth 3, got %d", state.ValueStackDepth())
	}

	state.ValueClear()
	if state.ValueStackDepth() != 0 {
		t.Errorf("expected depth 0 after clear, got %d", state.ValueStackDepth())
	}
}

func TestValueClearEmpty(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.ValueClear() // Should not panic
	if state.ValueStackDepth() != 0 {
		t.Error("expected depth 0")
	}
}

// =============================================================================
// ValuePeek tests
// =============================================================================

func TestValuePeekUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.ValuePeek()
	if err == nil {
		t.Error("expected underflow error from ValuePeek on empty stack")
	}
}

// =============================================================================
// Enable/Disable flag methods
// =============================================================================

func TestEnableDisableTrace(t *testing.T) {
	state := NewDTState(&mockSession{})
	if state.TestState(TRACE) {
		t.Error("TRACE should not be set initially")
	}
	state.EnableTrace()
	if !state.TestState(TRACE) {
		t.Error("TRACE should be set after EnableTrace")
	}
	state.DisableTrace()
	if state.TestState(TRACE) {
		t.Error("TRACE should not be set after DisableTrace")
	}
}

func TestEnableDisableDebug(t *testing.T) {
	state := NewDTState(&mockSession{})
	if state.TestState(DEBUG) {
		t.Error("DEBUG should not be set initially")
	}
	state.EnableDebug()
	if !state.TestState(DEBUG) {
		t.Error("DEBUG should be set after EnableDebug")
	}
	state.DisableDebug()
	if state.TestState(DEBUG) {
		t.Error("DEBUG should not be set after DisableDebug")
	}
}

func TestEnableDisableVerbose(t *testing.T) {
	state := NewDTState(&mockSession{})
	if state.TestState(VERBOSE) {
		t.Error("VERBOSE should not be set initially")
	}
	state.EnableVerbose()
	if !state.TestState(VERBOSE) {
		t.Error("VERBOSE should be set after EnableVerbose")
	}
	state.DisableVerbose()
	if state.TestState(VERBOSE) {
		t.Error("VERBOSE should not be set after DisableVerbose")
	}
}

func TestMultipleFlags(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.SetState(TRACE | DEBUG)
	if !state.TestState(TRACE) {
		t.Error("TRACE should be set")
	}
	if !state.TestState(DEBUG) {
		t.Error("DEBUG should be set")
	}
	// Clear only TRACE
	state.ClearState(TRACE)
	if state.TestState(TRACE) {
		t.Error("TRACE should be cleared")
	}
	if !state.TestState(DEBUG) {
		t.Error("DEBUG should still be set")
	}
}

// =============================================================================
// Trace output tests
// =============================================================================

func TestTraceInfoWithTrace(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceInfo("def", "entity", "person", "set age")

	output := buf.String()
	if !strings.Contains(output, "<def") {
		t.Error("TraceInfo should output opening tag")
	}
	if !strings.Contains(output, "entity=\"person\"") {
		t.Error("TraceInfo should output attribute")
	}
	if !strings.Contains(output, "set age") {
		t.Error("TraceInfo should output content")
	}
}

func TestTraceInfoWithoutTrace(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	// TRACE not enabled

	state.TraceInfo("def", "entity", "person", "set age")
	if buf.Len() != 0 {
		t.Error("TraceInfo should not output when TRACE is not set")
	}
}

func TestTraceInfoSelfClosing(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceInfo("entitypop", "", "", "")
	output := buf.String()
	if !strings.Contains(output, "/>") {
		t.Error("TraceInfo with empty content should output self-closing tag")
	}
}

func TestTraceTable(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceTable("MyTable", true)
	output := buf.String()
	if !strings.Contains(output, "<table name=\"MyTable\">") {
		t.Errorf("TraceTable entering should output <table name=...>, got %q", output)
	}

	buf.Reset()
	state.TraceTable("MyTable", false)
	output = buf.String()
	if !strings.Contains(output, "</table>") {
		t.Errorf("TraceTable exiting should output </table>, got %q", output)
	}
}

func TestTraceTableWithoutTrace(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	// TRACE not enabled

	state.TraceTable("MyTable", true)
	if buf.Len() != 0 {
		t.Error("TraceTable should not output when TRACE is not set")
	}
}

func TestTraceCondition(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceCondition(1, true)
	output := buf.String()
	if !strings.Contains(output, "condition") {
		t.Error("TraceCondition should output condition element")
	}
	if !strings.Contains(output, "num=\"1\"") {
		t.Error("TraceCondition should include condition number")
	}
	if !strings.Contains(output, "result=\"true\"") {
		t.Error("TraceCondition should include result")
	}
}

func TestTraceAction(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceAction(3)
	output := buf.String()
	if !strings.Contains(output, "action") {
		t.Error("TraceAction should output action element")
	}
	if !strings.Contains(output, "num=\"3\"") {
		t.Error("TraceAction should include action number")
	}
}

func TestTraceMessage(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableTrace()

	state.TraceMessage("test message")
	output := buf.String()
	if !strings.Contains(output, "<!-- test message -->") {
		t.Errorf("TraceMessage should output comment, got %q", output)
	}
}

// =============================================================================
// Extended state tests
// =============================================================================

func TestExtendedState(t *testing.T) {
	state := NewDTState(&mockSession{})
	key := dtrules.GetRName("testkey")

	// Initially nil
	val := state.GetExtendedState(key)
	if val != nil {
		t.Error("GetExtendedState should return nil initially")
	}

	// Set a value
	state.SetExtendedState(key, "hello")
	val = state.GetExtendedState(key)
	if val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}

	// Overwrite
	state.SetExtendedState(key, 42)
	val = state.GetExtendedState(key)
	if val != 42 {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestExtendedStateDifferentKeys(t *testing.T) {
	state := NewDTState(&mockSession{})
	key1 := dtrules.GetRName("key1")
	key2 := dtrules.GetRName("key2")

	state.SetExtendedState(key1, "value1")
	state.SetExtendedState(key2, "value2")

	if state.GetExtendedState(key1) != "value1" {
		t.Error("key1 should have value1")
	}
	if state.GetExtendedState(key2) != "value2" {
		t.Error("key2 should have value2")
	}
}

// =============================================================================
// Output configuration tests
// =============================================================================

func TestSetOutput(t *testing.T) {
	state := NewDTState(&mockSession{})
	var debugBuf, errBuf bytes.Buffer

	state.SetOutput(&debugBuf, &errBuf)
	state.EnableTrace()

	// Trace output should go to debugBuf
	state.TraceMessage("trace message")
	if !strings.Contains(debugBuf.String(), "trace message") {
		t.Error("trace output should go to debug writer")
	}

	// Error output should go to errBuf
	state.Error("error message")
	if !strings.Contains(errBuf.String(), "error message") {
		t.Error("error output should go to error writer")
	}
}

func TestSetOutputNil(t *testing.T) {
	state := NewDTState(&mockSession{})
	var buf bytes.Buffer

	// Setting nil for debug should not change it
	state.SetOutput(nil, &buf)
	// Setting nil for error should not change it
	state.SetOutput(&buf, nil)
	// Should not panic
}

// =============================================================================
// Debug output tests
// =============================================================================

func TestDebugWithFlag(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	state.EnableDebug()

	state.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Error("Debug should output when DEBUG flag is set")
	}
}

func TestDebugWithoutFlag(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)
	// DEBUG not set

	state.Debug("debug message")
	if buf.Len() != 0 {
		t.Error("Debug should not output when DEBUG flag is not set")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	state := NewDTState(&mockSession{})
	state.SetOutput(&buf, &buf)

	result := state.Error("error text")
	if !result {
		t.Error("Error should return true")
	}
	if !strings.Contains(buf.String(), "error text") {
		t.Error("Error should output to error stream")
	}
}

// =============================================================================
// Decision table context tests
// =============================================================================

func TestDecisionTableContext(t *testing.T) {
	state := NewDTState(&mockSession{})

	// Initial state
	if state.GetCurrentTable() != nil {
		t.Error("GetCurrentTable should return nil initially")
	}
	if state.GetCurrentTableSection() != "" {
		t.Error("GetCurrentTableSection should return empty initially")
	}
	if state.GetNumberInSection() != -1 {
		t.Errorf("GetNumberInSection should return -1 initially, got %d", state.GetNumberInSection())
	}

	// Set table
	state.SetCurrentTable("test_table")
	if state.GetCurrentTable() != "test_table" {
		t.Error("GetCurrentTable should return what was set")
	}

	// Set section
	state.SetCurrentTableSection("Condition", 3)
	if state.GetCurrentTableSection() != "Condition" {
		t.Error("GetCurrentTableSection should return 'Condition'")
	}
	if state.GetNumberInSection() != 3 {
		t.Errorf("GetNumberInSection should return 3, got %d", state.GetNumberInSection())
	}
}

// =============================================================================
// Operator table tests
// =============================================================================

func TestSetOperatorTable(t *testing.T) {
	state := NewDTState(&mockSession{})
	table := make([]dtrules.Object, 10)
	state.SetOperatorTable(table)
	// No getter exposed, but it should not panic
}

// =============================================================================
// String representation tests
// =============================================================================

func TestStateString(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))

	str := state.String()
	if !strings.Contains(str, "dstk[2]") {
		t.Errorf("String() should show data stack depth, got %q", str)
	}
}

func TestStateStringEmpty(t *testing.T) {
	state := NewDTState(&mockSession{})
	str := state.String()
	if !strings.Contains(str, "dstk[0]") {
		t.Errorf("String() should show empty data stack, got %q", str)
	}
}

// =============================================================================
// Control stack indexed access tests
// =============================================================================

func TestGetCtrlStack(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.CtrlPush(dtrules.NewRString("a"))
	state.CtrlPush(dtrules.NewRString("b"))

	obj, err := state.GetCtrlStack(0)
	if err != nil {
		t.Fatalf("GetCtrlStack(0) failed: %v", err)
	}
	if obj.StringValue() != "a" {
		t.Errorf("expected 'a' at index 0, got %q", obj.StringValue())
	}

	obj, err = state.GetCtrlStack(1)
	if err != nil {
		t.Fatalf("GetCtrlStack(1) failed: %v", err)
	}
	if obj.StringValue() != "b" {
		t.Errorf("expected 'b' at index 1, got %q", obj.StringValue())
	}
}

func TestGetCtrlStackOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.GetCtrlStack(0)
	if err == nil {
		t.Error("expected error for out of bounds get on empty stack")
	}
	_, err = state.GetCtrlStack(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestSetCtrlStack(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.CtrlPush(dtrules.NewRString("original"))

	err := state.SetCtrlStack(0, dtrules.NewRString("replaced"))
	if err != nil {
		t.Fatalf("SetCtrlStack failed: %v", err)
	}

	obj, _ := state.GetCtrlStack(0)
	if obj.StringValue() != "replaced" {
		t.Errorf("expected 'replaced', got %q", obj.StringValue())
	}
}

func TestSetCtrlStackOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	err := state.SetCtrlStack(0, dtrules.NewRString("x"))
	if err == nil {
		t.Error("expected error for out of bounds set on empty stack")
	}
	err = state.SetCtrlStack(-1, dtrules.NewRString("x"))
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestCtrlStackDepth(t *testing.T) {
	state := NewDTState(&mockSession{})
	if state.CtrlStackDepth() != 0 {
		t.Error("expected 0 initially")
	}
	state.CtrlPush(dtrules.NewRString("x"))
	if state.CtrlStackDepth() != 1 {
		t.Errorf("expected 1, got %d", state.CtrlStackDepth())
	}
}

// =============================================================================
// Control stack underflow
// =============================================================================

func TestCtrlPopUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.CtrlPop()
	if err == nil {
		t.Error("expected underflow error")
	}
}

// =============================================================================
// Data stack underflow for peek
// =============================================================================

func TestDataPeekUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.DataPeek()
	if err == nil {
		t.Error("expected underflow error from DataPeek on empty stack")
	}
}

// =============================================================================
// Data stack negative index
// =============================================================================

func TestGetDataStackNegativeIndex(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.DataPush(dtrules.GetRIntegerValue(1))
	_, err := state.GetDataStack(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

// =============================================================================
// Entity stack tests
// =============================================================================

func TestGetEntityStackOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.GetEntityStack(0)
	if err == nil {
		t.Error("expected error for get on empty entity stack")
	}
	_, err = state.GetEntityStack(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestEntityFetchOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.EntityFetch(0)
	if err == nil {
		t.Error("expected error for fetch on empty entity stack")
	}
}

func TestEntityPopUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	_, err := state.EntityPop()
	if err == nil {
		t.Error("expected underflow error")
	}
}

// =============================================================================
// Frame operations tests
// =============================================================================

func TestFrameValue(t *testing.T) {
	state := NewDTState(&mockSession{})

	// Push some items, then a frame
	state.CtrlPush(dtrules.NewRString("before"))
	state.PushFrame()
	state.CtrlPush(dtrules.NewRString("frame_item"))

	// Get value in frame
	obj, err := state.GetFrameValue(0)
	if err != nil {
		t.Fatalf("GetFrameValue(0) failed: %v", err)
	}
	if obj.StringValue() != "frame_item" {
		t.Errorf("expected 'frame_item', got %q", obj.StringValue())
	}
}

func TestGetFrameValueOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.PushFrame()
	_, err := state.GetFrameValue(0)
	if err == nil {
		t.Error("expected out of bounds error")
	}
}

func TestSetFrameValue(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.PushFrame()
	state.CtrlPush(dtrules.NewRString("original"))

	err := state.SetFrameValue(0, dtrules.NewRString("modified"))
	if err != nil {
		t.Fatalf("SetFrameValue failed: %v", err)
	}

	obj, _ := state.GetFrameValue(0)
	if obj.StringValue() != "modified" {
		t.Errorf("expected 'modified', got %q", obj.StringValue())
	}
}

func TestSetFrameValueOutOfBounds(t *testing.T) {
	state := NewDTState(&mockSession{})
	state.PushFrame()
	err := state.SetFrameValue(0, dtrules.NewRString("x"))
	if err == nil {
		t.Error("expected out of bounds error")
	}
}

func TestPopFrameUnderflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	err := state.PopFrame()
	if err == nil {
		t.Error("expected underflow error")
	}
}

func TestNestedFrames(t *testing.T) {
	state := NewDTState(&mockSession{})

	// Push items and frames
	state.CtrlPush(dtrules.NewRString("outer"))
	state.PushFrame()
	state.CtrlPush(dtrules.NewRString("inner1"))

	frame1 := state.GetCurrentFrame()

	state.PushFrame()
	state.CtrlPush(dtrules.NewRString("inner2"))

	frame2 := state.GetCurrentFrame()
	if frame2 <= frame1 {
		t.Error("nested frame should have higher index")
	}

	// Pop inner frame
	state.PopFrame()
	if state.GetCurrentFrame() != frame1 {
		t.Error("frame should be restored after pop")
	}

	// Pop outer frame
	state.PopFrame()
	if state.GetCurrentFrame() != 0 {
		t.Error("frame should be 0 after popping all frames")
	}
}

// =============================================================================
// Stack overflow tests
// =============================================================================

func TestDataPushOverflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	// Fill to limit
	for i := 0; i < stackLimit; i++ {
		err := state.DataPush(dtrules.GetRIntegerValue(int64(i)))
		if err != nil {
			t.Fatalf("DataPush failed at %d: %v", i, err)
		}
	}
	// One more should overflow
	err := state.DataPush(dtrules.GetRIntegerValue(0))
	if err == nil {
		t.Error("expected stack overflow error")
	}
}

func TestValuePushOverflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	for i := 0; i < stackLimit; i++ {
		err := state.ValuePush(dtrules.NewValueInteger(int64(i)))
		if err != nil {
			t.Fatalf("ValuePush failed at %d: %v", i, err)
		}
	}
	err := state.ValuePush(dtrules.NewValueInteger(0))
	if err == nil {
		t.Error("expected stack overflow error")
	}
}

func TestCtrlPushOverflow(t *testing.T) {
	state := NewDTState(&mockSession{})
	for i := 0; i < stackLimit; i++ {
		err := state.CtrlPush(dtrules.GetRIntegerValue(int64(i)))
		if err != nil {
			t.Fatalf("CtrlPush failed at %d: %v", i, err)
		}
	}
	err := state.CtrlPush(dtrules.GetRIntegerValue(0))
	if err == nil {
		t.Error("expected stack overflow error")
	}
}

// =============================================================================
// InContext tests (without entities — just verify no panic)
// =============================================================================

func TestInContextEmpty(t *testing.T) {
	state := NewDTState(&mockSession{})
	if state.InContext("anything") {
		t.Error("should return false on empty entity stack")
	}
}

func TestInContextInvalidName(t *testing.T) {
	state := NewDTState(&mockSession{})
	// Empty name should return false, not panic
	if state.InContext("") {
		t.Error("should return false for empty name")
	}
}

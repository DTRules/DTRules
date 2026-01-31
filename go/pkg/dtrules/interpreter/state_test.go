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
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// mockSession implements minimal Session for testing
type mockSession struct {
	uniqueID int
}

func (m *mockSession) GetState() dtrules.State                 { return nil }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory { return nil }
func (m *mockSession) GetUniqueID() int                        { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return nil, nil
}
func (m *mockSession) Compile(expr string) (dtrules.Object, error) {
	return nil, nil
}

func TestDataStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test empty stack
	if state.DataStackDepth() != 0 {
		t.Errorf("Expected empty stack, got size %d", state.DataStackDepth())
	}

	// Test push
	state.DataPush(dtrules.GetRIntegerValue(1))
	state.DataPush(dtrules.GetRIntegerValue(2))
	state.DataPush(dtrules.GetRIntegerValue(3))

	if state.DataStackDepth() != 3 {
		t.Errorf("Expected stack size 3, got %d", state.DataStackDepth())
	}

	// Test pop (LIFO order)
	obj, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}
	val, _ := obj.IntValue()
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}

	obj, err = state.DataPop()
	if err != nil {
		t.Fatalf("DataPop failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 2 {
		t.Errorf("Expected 2, got %d", val)
	}

	// Test peek (top of stack)
	obj, err = state.DataPeek()
	if err != nil {
		t.Fatalf("DataPeek failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	// Stack should still have 1 element
	if state.DataStackDepth() != 1 {
		t.Errorf("Expected stack size 1, got %d", state.DataStackDepth())
	}

	// Pop last element
	state.DataPop()

	// Test underflow
	_, err = state.DataPop()
	if err == nil {
		t.Error("Expected stack underflow error")
	}
}

func TestDataStackIndexed(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRIntegerValue(20))
	state.DataPush(dtrules.GetRIntegerValue(30))

	// GetDataStack uses 0 as bottom of stack
	// Get at index 0 (bottom)
	obj, err := state.GetDataStack(0)
	if err != nil {
		t.Fatalf("GetDataStack(0) failed: %v", err)
	}
	val, _ := obj.IntValue()
	if val != 10 {
		t.Errorf("Expected 10 at index 0 (bottom), got %d", val)
	}

	// Get at index 1 (middle)
	obj, err = state.GetDataStack(1)
	if err != nil {
		t.Fatalf("GetDataStack(1) failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 20 {
		t.Errorf("Expected 20 at index 1, got %d", val)
	}

	// Get at index 2 (top)
	obj, err = state.GetDataStack(2)
	if err != nil {
		t.Fatalf("GetDataStack(2) failed: %v", err)
	}
	val, _ = obj.IntValue()
	if val != 30 {
		t.Errorf("Expected 30 at index 2 (top), got %d", val)
	}

	// Get out of bounds
	_, err = state.GetDataStack(3)
	if err == nil {
		t.Error("Expected error for out of bounds get")
	}
}

func TestControlStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test empty control stack
	if state.CtrlDepth() != 0 {
		t.Errorf("Expected empty control stack, got size %d", state.CtrlDepth())
	}

	// Push to control stack
	state.CtrlPush(dtrules.NewRString("test"))
	if state.CtrlDepth() != 1 {
		t.Errorf("Expected control stack size 1, got %d", state.CtrlDepth())
	}

	// Pop from control stack
	obj, err := state.CtrlPop()
	if err != nil {
		t.Fatalf("CtrlPop failed: %v", err)
	}
	if obj == nil {
		t.Fatal("CtrlPop returned nil")
	}
	if obj.StringValue() != "test" {
		t.Errorf("Expected 'test', got '%s'", obj.StringValue())
	}
}

func TestEntityStack(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Initial entity stack should be empty
	if state.EntityDepth() != 0 {
		t.Errorf("Expected empty entity stack, got size %d", state.EntityDepth())
	}
}

func TestFrame(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Initial frame is 0
	initialFrame := state.GetCurrentFrame()
	if initialFrame != 0 {
		t.Errorf("Expected initial frame 0, got %d", initialFrame)
	}

	// Push some control stack items
	state.CtrlPush(dtrules.NewRString("before_frame"))

	// Push a frame
	err := state.PushFrame()
	if err != nil {
		t.Fatalf("PushFrame failed: %v", err)
	}

	// Frame should be updated
	newFrame := state.GetCurrentFrame()
	if newFrame <= initialFrame {
		t.Errorf("Expected frame to increase after push, got %d", newFrame)
	}

	// Push items in new frame
	state.CtrlPush(dtrules.NewRString("in_frame"))

	// Pop the frame
	err = state.PopFrame()
	if err != nil {
		t.Fatalf("PopFrame failed: %v", err)
	}

	// Frame should be back to initial
	if state.GetCurrentFrame() != initialFrame {
		t.Errorf("Expected frame %d after pop, got %d", initialFrame, state.GetCurrentFrame())
	}
}

func TestStateFlags(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	// Test initial state
	if state.TestState(1) {
		t.Error("Flag 1 should not be set initially")
	}

	// Set a flag
	state.SetState(1)
	if !state.TestState(1) {
		t.Error("Flag 1 should be set")
	}

	// Clear the flag
	state.ClearState(1)
	if state.TestState(1) {
		t.Error("Flag 1 should be cleared")
	}
}

func TestGetSession(t *testing.T) {
	session := &mockSession{}
	state := NewDTState(session)

	if state.GetSession() != session {
		t.Error("GetSession should return the session")
	}
}

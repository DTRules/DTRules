// Copyright 2004-2011 DTRules.com, Inc.
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

// Package interpreter implements the DTRules interpreter state and stacks.
package interpreter

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// State flags
const (
	DEBUG   = 0x00000001
	TRACE   = 0x00000002
	ECHO    = 0x00000004
	VERBOSE = 0x00000008
)

const stackLimit = 1000

// DTState implements the interpreter state with three stacks.
// The interpreter is a stack-based interpreter similar to PostScript.
//
// - Control stack: implements stack frames for decision tables and local variables
// - Entity stack: defines the context for associating attributes with values (like PostScript's dictionary stack)
// - Data stack: passes data to operators and returns results
//
// The state also supports an optimized Value-based stack for high-performance execution.
type DTState struct {
	// The three stacks (Object-based, for compatibility)
	ctrlStk   []dtrules.Object
	dataStk   []dtrules.Object
	entityStk []dtrules.Entity

	// Optimized Value-based stack (for bytecode execution)
	valueStk []dtrules.Value

	// Frame management for control stack
	frames       []int
	currentFrame int

	// Session reference
	session dtrules.Session

	// State flags (DEBUG, TRACE, ECHO, VERBOSE)
	state int

	// Extended state for custom operators
	extendedState map[*dtrules.RName]interface{}

	// Random number generator
	Seed int64
	Rand *rand.Rand

	// Output streams
	debugOut io.Writer
	errorOut io.Writer
	traceOut io.Writer

	// Current decision table context
	currentTable        interface{} // *RDecisionTable when implemented
	currentTableSection string
	numberInSection     int
	anode               interface{} // *ANode when implemented

	// Operator table for bytecode execution (set externally to avoid import cycle)
	operatorTable []dtrules.Object
}

// NewDTState creates a new interpreter state for the given session.
func NewDTState(session dtrules.Session) *DTState {
	seed := int64(0x711083186866559)
	return &DTState{
		ctrlStk:         make([]dtrules.Object, 0, stackLimit),
		dataStk:         make([]dtrules.Object, 0, stackLimit),
		entityStk:       make([]dtrules.Entity, 0, stackLimit),
		valueStk:        make([]dtrules.Value, 0, stackLimit),
		frames:          make([]int, 0, stackLimit),
		currentFrame:    0,
		session:         session,
		state:           0,
		extendedState:   make(map[*dtrules.RName]interface{}),
		Seed:            seed,
		Rand:            rand.New(rand.NewSource(seed)),
		debugOut:        os.Stdout,
		errorOut:        os.Stdout,
		traceOut:        os.Stdout,
		numberInSection: -1,
	}
}

// GetSession returns the session associated with this state.
func (s *DTState) GetSession() dtrules.Session {
	return s.session
}

// State flag methods

// TestState checks if a state flag is set.
func (s *DTState) TestState(flag int) bool {
	return (s.state & flag) != 0
}

// SetState sets a state flag.
func (s *DTState) SetState(flag int) {
	s.state |= flag
}

// ClearState clears a state flag.
func (s *DTState) ClearState(flag int) {
	s.state &= ^flag
}

// Data stack operations

// DataPush pushes an object onto the data stack.
// Returns error if stack overflow would occur.
func (s *DTState) DataPush(obj dtrules.Object) error {
	if len(s.dataStk) >= stackLimit {
		return dtrules.StackOverflowError("DataPush", "Data Stack overflow")
	}
	s.dataStk = append(s.dataStk, obj)
	if s.TestState(VERBOSE) {
		s.TraceInfo("datapush", "attribs", obj.PostFix(), "")
	}
	return nil
}

// DataPop pops and returns an object from the data stack.
func (s *DTState) DataPop() (dtrules.Object, error) {
	if len(s.dataStk) <= 0 {
		return nil, dtrules.StackUnderflowError("DataPop")
	}
	idx := len(s.dataStk) - 1
	obj := s.dataStk[idx]
	s.dataStk = s.dataStk[:idx]
	if s.TestState(VERBOSE) {
		s.TraceInfo("datapop", "", "", obj.StringValue())
	}
	return obj, nil
}

// DataPeek returns the top object without removing it.
func (s *DTState) DataPeek() (dtrules.Object, error) {
	if len(s.dataStk) <= 0 {
		return nil, dtrules.StackUnderflowError("DataPeek")
	}
	return s.dataStk[len(s.dataStk)-1], nil
}

// DataStackDepth returns the current depth of the data stack.
func (s *DTState) DataStackDepth() int {
	return len(s.dataStk)
}

// GetDataStack returns the element at the given index (0 is bottom).
func (s *DTState) GetDataStack(i int) (dtrules.Object, error) {
	if i >= len(s.dataStk) {
		return nil, dtrules.NewRulesError("Data Stack Overflow", "GetDataStack", fmt.Sprintf("index out of range: %d", i))
	}
	if i < 0 {
		return nil, dtrules.NewRulesError("Data Stack Underflow", "GetDataStack", fmt.Sprintf("index out of range: %d", i))
	}
	return s.dataStk[i], nil
}

// =============================================================================
// Value stack operations (optimized, allocation-free for primitives)
// =============================================================================

// ValuePush pushes a Value onto the value stack.
func (s *DTState) ValuePush(v dtrules.Value) error {
	if len(s.valueStk) >= stackLimit {
		return dtrules.StackOverflowError("ValuePush", "Value Stack overflow")
	}
	s.valueStk = append(s.valueStk, v)
	return nil
}

// ValuePop pops and returns a Value from the value stack.
func (s *DTState) ValuePop() (dtrules.Value, error) {
	if len(s.valueStk) <= 0 {
		return dtrules.ValueNull, dtrules.StackUnderflowError("ValuePop")
	}
	idx := len(s.valueStk) - 1
	v := s.valueStk[idx]
	s.valueStk = s.valueStk[:idx]
	return v, nil
}

// ValuePeek returns the top Value without removing it.
func (s *DTState) ValuePeek() (dtrules.Value, error) {
	if len(s.valueStk) <= 0 {
		return dtrules.ValueNull, dtrules.StackUnderflowError("ValuePeek")
	}
	return s.valueStk[len(s.valueStk)-1], nil
}

// ValueStackDepth returns the current depth of the value stack.
func (s *DTState) ValueStackDepth() int {
	return len(s.valueStk)
}

// ValuePushInt pushes an integer value (convenience method).
func (s *DTState) ValuePushInt(v int64) error {
	return s.ValuePush(dtrules.NewValueInteger(v))
}

// ValuePushDouble pushes a double value (convenience method).
func (s *DTState) ValuePushDouble(v float64) error {
	return s.ValuePush(dtrules.NewValueDouble(v))
}

// ValuePushBool pushes a boolean value (convenience method).
func (s *DTState) ValuePushBool(v bool) error {
	return s.ValuePush(dtrules.NewValueBoolean(v))
}

// ValuePopInt pops an integer value (convenience method).
func (s *DTState) ValuePopInt() (int64, error) {
	v, err := s.ValuePop()
	if err != nil {
		return 0, err
	}
	if !v.IsInteger() {
		return 0, dtrules.ConversionError("ValuePopInt", "expected integer")
	}
	return v.AsInteger(), nil
}

// ValuePopDouble pops a double value (convenience method).
func (s *DTState) ValuePopDouble() (float64, error) {
	v, err := s.ValuePop()
	if err != nil {
		return 0, err
	}
	if !v.IsNumeric() {
		return 0, dtrules.ConversionError("ValuePopDouble", "expected numeric")
	}
	return v.AsDouble(), nil
}

// ValuePopBool pops a boolean value (convenience method).
func (s *DTState) ValuePopBool() (bool, error) {
	v, err := s.ValuePop()
	if err != nil {
		return false, err
	}
	if !v.IsBoolean() {
		return false, dtrules.ConversionError("ValuePopBool", "expected boolean")
	}
	return v.AsBoolean(), nil
}

// ValueClear clears the value stack.
func (s *DTState) ValueClear() {
	s.valueStk = s.valueStk[:0]
}

// ValueDup duplicates the top value.
func (s *DTState) ValueDup() error {
	if len(s.valueStk) <= 0 {
		return dtrules.StackUnderflowError("ValueDup")
	}
	return s.ValuePush(s.valueStk[len(s.valueStk)-1])
}

// ValueSwap swaps the top two values.
func (s *DTState) ValueSwap() error {
	n := len(s.valueStk)
	if n < 2 {
		return dtrules.StackUnderflowError("ValueSwap")
	}
	s.valueStk[n-1], s.valueStk[n-2] = s.valueStk[n-2], s.valueStk[n-1]
	return nil
}

// ValueRot rotates the top three values (a b c -- b c a).
func (s *DTState) ValueRot() error {
	n := len(s.valueStk)
	if n < 3 {
		return dtrules.StackUnderflowError("ValueRot")
	}
	a := s.valueStk[n-3]
	s.valueStk[n-3] = s.valueStk[n-2]
	s.valueStk[n-2] = s.valueStk[n-1]
	s.valueStk[n-1] = a
	return nil
}

// Entity stack operations

// EntityPush pushes an entity onto the entity stack.
// Returns error if stack overflow would occur.
func (s *DTState) EntityPush(entity dtrules.Entity) error {
	if len(s.entityStk) >= stackLimit {
		return dtrules.StackOverflowError("EntityPush", "Entity Stack overflow")
	}
	if s.TestState(TRACE) {
		s.TraceInfo("entitypush", "entity", entity.GetName().StringValue(), fmt.Sprintf("id=%d", entity.GetID()))
	}
	s.entityStk = append(s.entityStk, entity)
	return nil
}

// EntityPop pops and returns an entity from the entity stack.
func (s *DTState) EntityPop() (dtrules.Entity, error) {
	if len(s.entityStk) <= 0 {
		return nil, dtrules.NewRulesError("Entity Stack Underflow", "EntityPop", "Entity Stack underflow")
	}
	if s.TestState(TRACE) {
		s.TraceInfo("entitypop", "", "", "")
	}
	idx := len(s.entityStk) - 1
	entity := s.entityStk[idx]
	s.entityStk = s.entityStk[:idx]
	return entity, nil
}

// EntityDepth returns the current depth of the entity stack.
func (s *DTState) EntityDepth() int {
	return len(s.entityStk)
}

// EntityFetch returns the nth entity from the top of the entity stack.
// 0 returns the top element, 1 returns one from top, etc.
func (s *DTState) EntityFetch(i int) (dtrules.Entity, error) {
	if i >= len(s.entityStk) {
		return nil, dtrules.NewRulesError("Entity Stack Underflow", "EntityFetch", "Entity Stack underflow")
	}
	return s.entityStk[len(s.entityStk)-1-i], nil
}

// GetEntityStack returns the element at the given index (0 is bottom).
func (s *DTState) GetEntityStack(i int) (dtrules.Entity, error) {
	if i >= len(s.entityStk) {
		return nil, dtrules.NewRulesError("Entity Stack Overflow", "GetEntityStack", fmt.Sprintf("index out of range: %d", i))
	}
	if i < 0 {
		return nil, dtrules.NewRulesError("Entity Stack Underflow", "GetEntityStack", fmt.Sprintf("index out of range: %d", i))
	}
	return s.entityStk[i], nil
}

// Control stack operations

// CtrlPush pushes an object onto the control stack.
// Returns error if stack overflow would occur.
func (s *DTState) CtrlPush(obj dtrules.Object) error {
	if len(s.ctrlStk) >= stackLimit {
		return dtrules.StackOverflowError("CtrlPush", "Control Stack overflow")
	}
	s.ctrlStk = append(s.ctrlStk, obj)
	return nil
}

// CtrlPop pops and returns an object from the control stack.
func (s *DTState) CtrlPop() (dtrules.Object, error) {
	if len(s.ctrlStk) <= 0 {
		return nil, dtrules.StackUnderflowError("CtrlPop")
	}
	idx := len(s.ctrlStk) - 1
	obj := s.ctrlStk[idx]
	s.ctrlStk = s.ctrlStk[:idx]
	return obj, nil
}

// CtrlDepth returns the current depth of the control stack.
func (s *DTState) CtrlDepth() int {
	return len(s.ctrlStk)
}

// CtrlStackDepth returns the current depth of the control stack (interface method).
func (s *DTState) CtrlStackDepth() int {
	return len(s.ctrlStk)
}

// GetCtrlStack returns the element at the given index.
func (s *DTState) GetCtrlStack(i int) (dtrules.Object, error) {
	if i >= len(s.ctrlStk) {
		return nil, dtrules.NewRulesError("Control Stack Overflow", "GetCtrlStack", fmt.Sprintf("index out of range: %d", i))
	}
	if i < 0 {
		return nil, dtrules.NewRulesError("Control Stack Underflow", "GetCtrlStack", fmt.Sprintf("index out of range: %d", i))
	}
	return s.ctrlStk[i], nil
}

// SetCtrlStack sets the element at the given index.
func (s *DTState) SetCtrlStack(i int, v dtrules.Object) error {
	if i >= len(s.ctrlStk) {
		return dtrules.NewRulesError("Control Stack Overflow", "SetCtrlStack", fmt.Sprintf("index out of range: %d", i))
	}
	if i < 0 {
		return dtrules.NewRulesError("Control Stack Underflow", "SetCtrlStack", fmt.Sprintf("index out of range: %d", i))
	}
	s.ctrlStk[i] = v
	return nil
}

// Frame operations

// PushFrame pushes a new frame onto the control stack.
func (s *DTState) PushFrame() error {
	if len(s.frames) >= stackLimit {
		return dtrules.NewRulesError("Control Stack Overflow", "PushFrame", "Control Stack Overflow")
	}
	s.frames = append(s.frames, s.currentFrame)
	s.currentFrame = len(s.ctrlStk)
	return nil
}

// PopFrame pops a frame from the control stack.
func (s *DTState) PopFrame() error {
	if len(s.frames) <= 0 {
		return dtrules.NewRulesError("Control Stack Underflow", "PopFrame", "Control Stack underflow")
	}
	s.ctrlStk = s.ctrlStk[:s.currentFrame]
	idx := len(s.frames) - 1
	s.currentFrame = s.frames[idx]
	s.frames = s.frames[:idx]
	return nil
}

// GetCurrentFrame returns the current frame index.
func (s *DTState) GetCurrentFrame() int {
	return s.currentFrame
}

// GetFrameValue returns a value from the current frame.
func (s *DTState) GetFrameValue(i int) (dtrules.Object, error) {
	if s.currentFrame+i >= len(s.ctrlStk) {
		return nil, dtrules.OutOfBoundsError("GetFrameValue", "")
	}
	return s.ctrlStk[s.currentFrame+i], nil
}

// SetFrameValue sets a value within the current frame.
func (s *DTState) SetFrameValue(i int, value dtrules.Object) error {
	if s.currentFrame+i >= len(s.ctrlStk) {
		return dtrules.OutOfBoundsError("SetFrameValue", "")
	}
	s.ctrlStk[s.currentFrame+i] = value
	return nil
}

// Name resolution

// Find looks up a name on the entity stack and returns its value.
func (s *DTState) Find(name *dtrules.RName) (dtrules.Object, error) {
	entity, err := s.FindEntity(name)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}
	// Use attribute-only name for lookup (without entity prefix)
	attrName := name
	if name.GetEntityName() != nil {
		attrName = dtrules.GetRName(name.GetName())
		if attrName == nil {
			return nil, dtrules.UndefinedError("GetValue", "invalid attribute name: "+name.GetName())
		}
	}
	return entity.Get(attrName)
}

// FindEntity finds the entity containing an attribute.
func (s *DTState) FindEntity(name *dtrules.RName) (dtrules.Entity, error) {
	entityName := name.GetEntityName()

	// Get the attribute-only name for lookup (without entity prefix)
	attrName := name
	if entityName != nil {
		// Extract just the attribute part for ContainsAttribute lookup
		attrName = dtrules.GetRName(name.GetName())
		if attrName == nil {
			return nil, dtrules.UndefinedError("FindEntity", "invalid attribute name: "+name.GetName())
		}
	}

	if entityName == nil {
		// No entity prefix - search all entities
		for i := len(s.entityStk) - 1; i >= 0; i-- {
			e := s.entityStk[i]
			if e.ContainsAttribute(attrName) {
				return e, nil
			}
		}
	} else {
		// Entity prefix specified - must match entity name AND have attribute
		for i := len(s.entityStk) - 1; i >= 0; i-- {
			e := s.entityStk[i]
			eq, err := e.GetName().Equals(entityName)
			if err == nil && eq && e.ContainsAttribute(attrName) {
				return e, nil
			}
		}
	}
	return nil, nil
}

// Def assigns a value to a name in the entity stack context.
// Returns true if successful, false if name not found.
func (s *DTState) Def(name *dtrules.RName, value dtrules.Object, trace bool) (bool, error) {
	entity, err := s.FindEntity(name)
	if err != nil {
		return false, err
	}
	if entity == nil {
		return false, nil
	}

	if trace && s.TestState(TRACE) {
		s.TraceInfo("def",
			"entity", entity.GetName().StringValue(),
			fmt.Sprintf("name=%s id=%d", name.StringValue(), entity.GetID()))
	}

	// Use attribute-only name for Put (without entity prefix)
	attrName := name
	if name.GetEntityName() != nil {
		attrName = dtrules.GetRName(name.GetName())
		if attrName == nil {
			return false, dtrules.UndefinedError("Def", "invalid attribute name: "+name.GetName())
		}
	}

	err = entity.Put(attrName, value)
	if err != nil {
		return false, err
	}
	return true, nil
}

// DefProtected assigns a value with optional protection check.
func (s *DTState) DefProtected(name *dtrules.RName, value dtrules.Object, protect bool) error {
	ok, err := s.Def(name, value, true)
	if err != nil {
		return err
	}
	if !ok {
		return dtrules.UndefinedError("Def", "Name '"+name.StringValue()+"' not found in context")
	}
	return nil
}

// InContext checks if an entity with the given name is on the entity stack.
func (s *DTState) InContext(entityName string) bool {
	rname := dtrules.GetRName(entityName)
	if rname == nil {
		return false
	}
	return s.InContextRName(rname)
}

// InContextRName checks if an entity with the given RName is on the entity stack.
func (s *DTState) InContextRName(entityName *dtrules.RName) bool {
	for i := 0; i < len(s.entityStk); i++ {
		e := s.entityStk[i]
		eq, err := e.GetName().Equals(entityName)
		if err == nil && eq {
			return true
		}
	}
	return false
}

// Evaluation

// EvaluateCondition executes code and returns the resulting boolean value.
// The code should produce exactly one boolean value on the stack.
// If more values are left, the top value is used and extras are cleaned up.
func (s *DTState) EvaluateCondition(c dtrules.Object) (bool, error) {
	stackIndex := len(s.dataStk)
	err := c.Execute(s)
	if err != nil {
		return false, err
	}

	// Must have at least one result
	if len(s.dataStk) <= stackIndex {
		return false, dtrules.NewRulesError("Stack Check Exception", "EvaluateCondition", "No result on stack")
	}

	// Pop the result
	result, err := s.DataPop()
	if err != nil {
		return false, err
	}

	// Clean up any extra values left on the stack (some expressions leave extras)
	for len(s.dataStk) > stackIndex {
		s.DataPop()
	}

	return result.BooleanValue()
}

// Evaluate executes code that should leave the stack balanced.
// If extra values are left on the stack, they are cleaned up.
func (s *DTState) Evaluate(c dtrules.Object) error {
	stackIndex := len(s.dataStk)
	err := c.Execute(s)
	if err != nil {
		return err
	}

	// Clean up any extra values left on the stack (some expressions leave extras)
	for len(s.dataStk) > stackIndex {
		s.DataPop()
	}

	return nil
}

// Trace output

// TraceInfo outputs trace information in XML format.
func (s *DTState) TraceInfo(tag, attr, value, content string) {
	if s.TestState(TRACE) {
		if s.traceOut != nil {
			fmt.Fprintf(s.traceOut, "<%s", tag)
			if attr != "" {
				fmt.Fprintf(s.traceOut, " %s=\"%s\"", attr, value)
			}
			if content != "" {
				fmt.Fprintf(s.traceOut, ">%s</%s>\n", content, tag)
			} else {
				fmt.Fprintf(s.traceOut, "/>\n")
			}
		}
	}
}

// TraceTable traces decision table entry/exit.
func (s *DTState) TraceTable(tableName string, entering bool) {
	if s.TestState(TRACE) {
		if s.traceOut != nil {
			if entering {
				fmt.Fprintf(s.traceOut, "<table name=\"%s\">\n", tableName)
			} else {
				fmt.Fprintf(s.traceOut, "</table>\n")
			}
		}
	}
}

// TraceCondition traces condition evaluation.
func (s *DTState) TraceCondition(condNum int, result bool) {
	if s.TestState(TRACE) {
		if s.traceOut != nil {
			fmt.Fprintf(s.traceOut, "  <condition num=\"%d\" result=\"%t\"/>\n", condNum, result)
		}
	}
}

// TraceAction traces action execution.
func (s *DTState) TraceAction(actionNum int) {
	if s.TestState(TRACE) {
		if s.traceOut != nil {
			fmt.Fprintf(s.traceOut, "  <action num=\"%d\"/>\n", actionNum)
		}
	}
}

// TraceMessage outputs a simple trace message.
func (s *DTState) TraceMessage(msg string) {
	if s.TestState(TRACE) {
		if s.traceOut != nil {
			fmt.Fprintf(s.traceOut, "<!-- %s -->\n", msg)
		}
	}
}

// EnableTrace enables trace output.
func (s *DTState) EnableTrace() {
	s.SetState(TRACE)
}

// DisableTrace disables trace output.
func (s *DTState) DisableTrace() {
	s.ClearState(TRACE)
}

// EnableDebug enables debug output.
func (s *DTState) EnableDebug() {
	s.SetState(DEBUG)
}

// DisableDebug disables debug output.
func (s *DTState) DisableDebug() {
	s.ClearState(DEBUG)
}

// EnableVerbose enables verbose output.
func (s *DTState) EnableVerbose() {
	s.SetState(VERBOSE)
}

// DisableVerbose disables verbose output.
func (s *DTState) DisableVerbose() {
	s.ClearState(VERBOSE)
}

// Extended state

// GetExtendedState returns a custom state object.
func (s *DTState) GetExtendedState(key *dtrules.RName) interface{} {
	return s.extendedState[key]
}

// SetExtendedState sets a custom state object.
func (s *DTState) SetExtendedState(key *dtrules.RName, obj interface{}) {
	s.extendedState[key] = obj
}

// Output configuration

// SetOutput sets the debug and error output streams.
func (s *DTState) SetOutput(debugTrace, error io.Writer) {
	if debugTrace != nil {
		s.debugOut = debugTrace
		s.traceOut = debugTrace
	}
	if error != nil {
		s.errorOut = error
	}
}

// Debug printing

// Debug prints a string if DEBUG is on.
func (s *DTState) Debug(str string) {
	if s.TestState(DEBUG) {
		if s.TestState(ECHO) {
			fmt.Print(str)
		}
		fmt.Fprint(s.debugOut, str)
	}
}

// Error prints a string to the error stream.
func (s *DTState) Error(str string) bool {
	fmt.Fprint(s.errorOut, str)
	return true
}

// Print prints a string to stdout.
func (s *DTState) Print(str string) {
	fmt.Print(str)
}

// PStack prints all stacks for debugging.
func (s *DTState) PStack() {
	fmt.Print(s.String())
}

// Decision table context

// GetCurrentTable returns the current decision table.
func (s *DTState) GetCurrentTable() interface{} {
	return s.currentTable
}

// SetCurrentTable sets the current decision table.
func (s *DTState) SetCurrentTable(table interface{}) {
	s.currentTable = table
}

// GetCurrentTableSection returns the current section (Condition, Action, etc).
func (s *DTState) GetCurrentTableSection() string {
	return s.currentTableSection
}

// SetCurrentTableSection sets the current section and number.
func (s *DTState) SetCurrentTableSection(section string, number int) {
	s.currentTableSection = section
	s.numberInSection = number
}

// GetNumberInSection returns the current number in section.
func (s *DTState) GetNumberInSection() int {
	return s.numberInSection
}

// SetOperatorTable sets the operator table for bytecode execution.
// This should be called during initialization with the operators.GetOperatorTable() result.
func (s *DTState) SetOperatorTable(table []dtrules.Object) {
	s.operatorTable = table
}

// String returns a string representation of the state.
func (s *DTState) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("dstk[%d]: ", len(s.dataStk)))
	for _, obj := range s.dataStk {
		sb.WriteString(obj.StringValue())
		sb.WriteString(" ")
	}

	sb.WriteString(fmt.Sprintf("\nentitystk[%d]: ", len(s.entityStk)))
	for _, e := range s.entityStk {
		sb.WriteString(fmt.Sprintf("%s[%d] ", e.GetName().StringValue(), e.GetID()))
	}

	sb.WriteString(fmt.Sprintf("\nctrlstk[%d]: ", len(s.ctrlStk)))
	for _, obj := range s.ctrlStk {
		sb.WriteString(obj.StringValue())
		sb.WriteString(" ")
	}

	return sb.String()
}

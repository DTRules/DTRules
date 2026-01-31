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

// Package dtrules implements a Go runtime for DTRules that executes
// compiled decision tables and rule sets.
package dtrules

import (
	"time"
)

// Object is the base interface for all Rules Engine objects.
// This interface is used by the Interpreter to execute objects,
// and by the implementation of operators to manipulate objects.
type Object interface {
	// Type returns the type metadata for this object
	Type() *RType

	// Execute defines the executable behavior of an object within the
	// Rules Interpreter.
	Execute(state State) error

	// ArrayExecute defines the execution behavior when this object is
	// encountered during array execution. For most objects, this is
	// the same as pushing to the data stack.
	ArrayExecute(state State) error

	// GetExecutable returns an executable version of this object.
	// The non-executable behavior of all objects is to push themselves
	// onto the data stack. If the executable behavior is the same as
	// the non-executable behavior, this method may return a non-executable object.
	GetExecutable() Object

	// GetNonExecutable returns the non-executable representation of this object.
	GetNonExecutable() Object

	// IsExecutable returns true if the object is executable.
	// When an array is executing each of its elements, executable objects
	// have their Execute() method called. Non-executable objects are simply
	// pushed to the data stack.
	IsExecutable() bool

	// Equals compares this Rules Engine Object with the given object.
	Equals(o Object) (bool, error)

	// Compare compares this Rules Engine Object with the given object.
	// Returns -1 if less than, 0 if equal, 1 if greater than.
	Compare(o Object) (int, error)

	// StringValue returns the string representation of the object's data.
	// If appending a String and a Date to create a new string,
	// use the results from StringValue().
	StringValue() string

	// PostFix returns the postfix representation of an object.
	// Used to build trace files and for debugging.
	PostFix() string

	// Clone creates a shallow copy of this object.
	// For immutable types (boolean, integer, strings), this may return
	// a pointer to the original object.
	Clone(session Session) (Object, error)

	// RClone returns a clone suitable for rule execution.
	// For many types, this returns the object itself.
	RClone() Object

	// Type conversion methods - return error if conversion not supported

	// IntValue returns the int representation of this object.
	IntValue() (int, error)

	// LongValue returns the int64 representation of this object.
	LongValue() (int64, error)

	// DoubleValue returns the float64 representation of this object.
	DoubleValue() (float64, error)

	// BooleanValue returns the boolean representation of this object.
	BooleanValue() (bool, error)

	// TimeValue returns the time.Time representation of this object.
	TimeValue() (time.Time, error)

	// ArrayValue returns the slice representation of this object.
	ArrayValue() ([]Object, error)

	// TableValue returns the map representation of this object.
	TableValue() (map[Object]Object, error)

	// RIntegerValue returns the RInteger representation of this object.
	RIntegerValue() (*RInteger, error)

	// RDoubleValue returns the RDouble representation of this object.
	RDoubleValue() (*RDouble, error)

	// RBooleanValue returns the RBoolean representation of this object.
	RBooleanValue() (*RBoolean, error)

	// RStringValue returns the RString representation of this object.
	RStringValue() *RString

	// RNameValue returns the RName representation of this object.
	RNameValue() (*RName, error)

	// RArrayValue returns the RArray representation of this object.
	RArrayValue() (*RArray, error)

	// RTableValue returns the RTable representation of this object.
	RTableValue() (*RTable, error)

	// RTimeValue returns the RDate representation of this object.
	RTimeValue() (*RDate, error)

	// REntityValue returns the Entity representation of this object.
	REntityValue() (Entity, error)
}

// State flags
const (
	StateDebug   = 1 << 0 // Debug output enabled
	StateTrace   = 1 << 1 // Trace output enabled
	StateVerbose = 1 << 2 // Verbose output enabled
)

// State represents the interpreter state with three stacks.
// This is a forward declaration - the full implementation is in interpreter/state.go
type State interface {
	// DataPush pushes an object onto the data stack
	// Returns error if stack overflow would occur
	DataPush(obj Object) error

	// DataPop pops and returns an object from the data stack
	DataPop() (Object, error)

	// DataPeek returns the top object without removing it
	DataPeek() (Object, error)

	// DataStackDepth returns the current depth of the data stack
	DataStackDepth() int

	// GetDataStack returns an element from the data stack by index (0 = bottom)
	GetDataStack(index int) (Object, error)

	// EntityPush pushes an entity onto the entity stack
	// Returns error if stack overflow would occur
	EntityPush(entity Entity) error

	// EntityPop pops and returns an entity from the entity stack
	EntityPop() (Entity, error)

	// EntityFetch returns entity at index from top (0 = top)
	EntityFetch(index int) (Entity, error)

	// EntityDepth returns the current depth of the entity stack
	EntityDepth() int

	// Find looks up a name on the entity stack
	Find(name *RName) (Object, error)

	// FindEntity finds the entity containing an attribute
	FindEntity(name *RName) (Entity, error)

	// Def assigns a value in the current context
	// Returns true if successful, false if name not found
	Def(name *RName, value Object, trace bool) (bool, error)

	// CtrlPush pushes an object onto the control stack
	// Returns error if stack overflow would occur
	CtrlPush(obj Object) error

	// CtrlPop pops and returns an object from the control stack
	CtrlPop() (Object, error)

	// CtrlStackDepth returns the current depth of the control stack
	CtrlStackDepth() int

	// GetCtrlStack returns an element from the control stack by index (0 = bottom)
	GetCtrlStack(index int) (Object, error)

	// GetFrameValue returns a value from the current frame
	GetFrameValue(index int) (Object, error)

	// SetFrameValue sets a value in the current frame
	SetFrameValue(index int, value Object) error

	// GetSession returns the associated session
	GetSession() Session

	// TestState checks if a state flag is set
	TestState(flag int) bool

	// SetState sets a state flag
	SetState(flag int)

	// ClearState clears a state flag
	ClearState(flag int)

	// Debug outputs a debug message
	Debug(msg string)

	// Print outputs a message
	Print(msg string)

	// PStack prints all stacks for debugging
	PStack()

	// TraceInfo outputs trace information
	TraceInfo(tag, attr, value, content string)

	// Evaluate executes code and verifies stack is balanced
	Evaluate(code Object) error

	// EvaluateCondition executes code and returns boolean result
	EvaluateCondition(code Object) (bool, error)

	// GetCurrentTableSection returns the current section name for tracing
	GetCurrentTableSection() string

	// GetNumberInSection returns the current number in section for tracing
	GetNumberInSection() int

	// SetCurrentTableSection sets the current section for tracing
	SetCurrentTableSection(section string, number int)
}

// Session represents an execution context.
// This is a forward declaration - the full implementation is in session/session.go
type Session interface {
	// GetState returns the interpreter state
	GetState() State

	// GetEntityFactory returns the entity factory
	GetEntityFactory() EntityFactory

	// GetUniqueID returns a unique ID for this session
	GetUniqueID() int

	// GetDateParser returns the date parser for this session
	GetDateParser() DateParser

	// GetRuleSet returns the rule set for this session
	GetRuleSet() RuleSet

	// CreateEntity creates a new entity instance by name
	CreateEntity(name *RName) (Entity, error)

	// Compile compiles a postfix expression string into executable code
	Compile(expr string) (Object, error)
}

// EntityFactory manages reference entities and creates instances.
// This is a forward declaration - the full implementation is in entity/factory.go
// DecisionTable interface for decision table operations
type DecisionTable interface {
	Object
	// ExecuteTable runs the decision table's initial actions and decision tree directly.
	// This bypasses context handling (which would cause recursion in contexts).
	ExecuteTable(state State) error
}

type EntityFactory interface {
	// GetUniqueID returns a unique ID
	GetUniqueID() int

	// GetDecisionTable returns a decision table by name
	GetDecisionTable(name *RName) (DecisionTable, error)

	// GetDecisionTablesEntity returns the entity containing all decision tables
	GetDecisionTablesEntity() Entity

	// GetReferenceEntity returns a reference entity by name
	GetReferenceEntity(name *RName) (Entity, error)

	// CreateEntity creates a new entity instance from a reference
	CreateEntity(session Session, name *RName) (Entity, error)
}

// Entity represents a typed attribute container.
// This is a forward declaration - the full implementation is in entity/entity.go
type Entity interface {
	Object

	// GetName returns the entity's name
	GetName() *RName

	// Get retrieves an attribute value
	Get(name *RName) (Object, error)

	// Put sets an attribute value
	Put(name *RName, value Object) error

	// ContainsAttribute checks if entity has an attribute
	ContainsAttribute(name *RName) bool

	// GetAttributeNames returns all attribute names
	GetAttributeNames() []*RName

	// GetID returns the unique instance ID
	GetID() int
}

// DateParser parses date strings.
// This is a forward declaration.
type DateParser interface {
	// GetDate parses a string into a time.Time
	GetDate(s string) (time.Time, error)

	// Parse is an alias for GetDate
	Parse(s string) (time.Time, error)
}

// RuleSet holds EDD, decision tables, and mappings.
// This is a forward declaration - the full implementation is in session/ruleset.go
type RuleSet interface {
	// NewSession creates a new session for this rule set
	NewSession() (Session, error)
}

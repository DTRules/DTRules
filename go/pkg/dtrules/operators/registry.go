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

// Package operators implements the DTRules operator framework and built-in operators.
package operators

import (
	"fmt"
	"sync"
	"time"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// OperatorFunc is the function signature for operator implementations.
type OperatorFunc func(state dtrules.State) error

// Operator represents an operator in the rules engine.
type Operator struct {
	dtrules.BaseObject
	name     *dtrules.RName
	fn       OperatorFunc
	priority int
	index    int // Index in operatorTable for fast lookup
}

var (
	operatorsMu sync.RWMutex
	operators   = make(map[*dtrules.RName]*Operator)
	// Indexed operator table for fast lookup (optimization)
	operatorTable    []*Operator
	operatorTableMu  sync.RWMutex
	operatorIndexMap = make(map[*dtrules.RName]int)
)

// Register registers an operator with the given name.
func Register(name string, fn OperatorFunc) *Operator {
	return RegisterWithPriority(name, fn, 0)
}

// RegisterWithPriority registers an operator with a specific priority.
func RegisterWithPriority(name string, fn OperatorFunc, priority int) *Operator {
	rname := dtrules.GetRName(name)

	operatorsMu.Lock()
	defer operatorsMu.Unlock()

	// Check for existing operator with higher priority
	if existing, ok := operators[rname]; ok {
		if priority <= existing.priority && existing.priority != 0 {
			return existing
		}
	}

	// Get or assign index for fast lookup
	operatorTableMu.Lock()
	idx, hasIdx := operatorIndexMap[rname]
	if !hasIdx {
		idx = len(operatorTable)
		operatorIndexMap[rname] = idx
		operatorTable = append(operatorTable, nil)
	}
	operatorTableMu.Unlock()

	op := &Operator{
		name:     rname,
		fn:       fn,
		priority: priority,
		index:    idx,
	}
	operators[rname] = op

	// Update the indexed table
	operatorTableMu.Lock()
	operatorTable[idx] = op
	operatorTableMu.Unlock()

	return op
}

// Alias creates an alias for an existing operator.
func Alias(existing, alias string) {
	existingName := dtrules.GetRName(existing)
	aliasName := dtrules.GetRName(alias)

	operatorsMu.Lock()
	defer operatorsMu.Unlock()

	if op, ok := operators[existingName]; ok {
		// Check for duplicate - allow if it's the same operator (idempotent)
		if existingOp, exists := operators[aliasName]; exists {
			if existingOp != op {
				panic(fmt.Sprintf("Duplicate Operators defined for %s", alias))
			}
			return // Already aliased to the same operator
		}
		operators[aliasName] = op
	}
}

// Get returns an operator by name.
func Get(name *dtrules.RName) (*Operator, bool) {
	operatorsMu.RLock()
	defer operatorsMu.RUnlock()

	op, ok := operators[name]
	return op, ok
}

// GetByIndex returns an operator by its index (fast path, lock-free).
// Safe because operators are only registered at init time.
func GetByIndex(idx int) *Operator {
	// Read table length first (safe since we only append)
	if idx >= 0 && idx < len(operatorTable) {
		return operatorTable[idx]
	}
	return nil
}

// GetIndex returns the index for an operator name.
func GetIndex(name *dtrules.RName) (int, bool) {
	operatorTableMu.RLock()
	defer operatorTableMu.RUnlock()

	idx, ok := operatorIndexMap[name]
	return idx, ok
}

// Index returns the operator's index.
func (o *Operator) Index() int {
	return o.index
}

// GetOperatorTable returns the operator table for bytecode execution.
// This returns a copy of the table as []dtrules.Object to avoid import cycles.
func GetOperatorTable() []dtrules.Object {
	operatorTableMu.RLock()
	defer operatorTableMu.RUnlock()

	result := make([]dtrules.Object, len(operatorTable))
	for i, op := range operatorTable {
		result[i] = op
	}
	return result
}

// GetByString returns an operator by string name.
func GetByString(name string) (*Operator, bool) {
	return Get(dtrules.GetRName(name))
}

// Type returns the operator type.
func (o *Operator) Type() *dtrules.RType {
	return dtrules.TypeOperator
}

// Execute runs this operator.
func (o *Operator) Execute(state dtrules.State) error {
	return o.fn(state)
}

// ArrayExecute runs this operator.
func (o *Operator) ArrayExecute(state dtrules.State) error {
	return o.fn(state)
}

// GetExecutable returns this operator.
func (o *Operator) GetExecutable() dtrules.Object {
	return o
}

// GetNonExecutable returns this operator.
func (o *Operator) GetNonExecutable() dtrules.Object {
	return o
}

// IsExecutable returns true for operators.
func (o *Operator) IsExecutable() bool {
	return true
}

// Equals compares operators.
func (o *Operator) Equals(obj dtrules.Object) (bool, error) {
	return o == obj, nil
}

// Compare is not supported for operators.
func (o *Operator) Compare(obj dtrules.Object) (int, error) {
	return 0, dtrules.UndefinedError("Compare", "Operator Objects do not support Compare")
}

// StringValue returns the operator name.
func (o *Operator) StringValue() string {
	return o.name.StringValue()
}

// PostFix returns the operator name.
func (o *Operator) PostFix() string {
	return o.name.StringValue()
}

// Clone returns this operator (operators are immutable).
func (o *Operator) Clone(session dtrules.Session) (dtrules.Object, error) {
	return o, nil
}

// RClone returns this operator.
func (o *Operator) RClone() dtrules.Object {
	return o
}

// RStringValue returns an RString for this operator.
func (o *Operator) RStringValue() *dtrules.RString {
	return dtrules.NewRString(o.StringValue())
}

// GetName returns the operator name.
func (o *Operator) GetName() *dtrules.RName {
	return o.name
}

// Priority returns the operator priority.
func (o *Operator) Priority() int {
	return o.priority
}

// String implements the Stringer interface.
func (o *Operator) String() string {
	return o.name.StringValue()
}

// Implement remaining Object interface methods

func (o *Operator) IntValue() (int, error) {
	return 0, dtrules.ConversionError("IntValue", "No Integer value exists for operator")
}

func (o *Operator) LongValue() (int64, error) {
	return 0, dtrules.ConversionError("LongValue", "No Long value exists for operator")
}

func (o *Operator) DoubleValue() (float64, error) {
	return 0, dtrules.ConversionError("DoubleValue", "No Double value exists for operator")
}

func (o *Operator) BooleanValue() (bool, error) {
	return false, dtrules.ConversionError("BooleanValue", "No Boolean value exists for operator")
}

func (o *Operator) TimeValue() (time.Time, error) {
	return time.Time{}, dtrules.ConversionError("TimeValue", "No Time value exists for operator")
}

func (o *Operator) ArrayValue() ([]dtrules.Object, error) {
	return nil, dtrules.ConversionError("ArrayValue", "No Array value exists for operator")
}

func (o *Operator) TableValue() (map[dtrules.Object]dtrules.Object, error) {
	return nil, dtrules.ConversionError("TableValue", "No Table value exists for operator")
}

func (o *Operator) RIntegerValue() (*dtrules.RInteger, error) {
	return nil, dtrules.ConversionError("RIntegerValue", "No Integer value exists for operator")
}

func (o *Operator) RDoubleValue() (*dtrules.RDouble, error) {
	return nil, dtrules.ConversionError("RDoubleValue", "No Double value exists for operator")
}

func (o *Operator) RBooleanValue() (*dtrules.RBoolean, error) {
	return nil, dtrules.ConversionError("RBooleanValue", "No Boolean value exists for operator")
}

func (o *Operator) RNameValue() (*dtrules.RName, error) {
	return nil, dtrules.ConversionError("RNameValue", "No Name value exists for operator")
}

func (o *Operator) RArrayValue() (*dtrules.RArray, error) {
	return nil, dtrules.ConversionError("RArrayValue", "No Array value exists for operator")
}

func (o *Operator) RTableValue() (*dtrules.RTable, error) {
	return nil, dtrules.ConversionError("RTableValue", "No Table value exists for operator")
}

func (o *Operator) RTimeValue() (*dtrules.RDate, error) {
	return nil, dtrules.ConversionError("RTimeValue", "No Time value exists for operator")
}

func (o *Operator) REntityValue() (dtrules.Entity, error) {
	return nil, dtrules.ConversionError("REntityValue", "No Entity value exists for operator")
}

// primitives is a singleton entity containing all operators
var primitives dtrules.Entity

// PrimitivesEntity wraps operators in an entity for the entity stack
type PrimitivesEntity struct {
	dtrules.BaseObject
	name *dtrules.RName
}

// NewPrimitivesEntity creates the primitives entity
func NewPrimitivesEntity() *PrimitivesEntity {
	return &PrimitivesEntity{
		name: dtrules.GetRName("primitives"),
	}
}

// GetPrimitives returns the primitives entity containing all operators.
func GetPrimitives() dtrules.Entity {
	if primitives == nil {
		primitives = NewPrimitivesEntity()
	}
	return primitives
}

// Type returns the entity type
func (p *PrimitivesEntity) Type() *dtrules.RType {
	return dtrules.TypeEntity
}

// Execute pushes this onto the data stack
func (p *PrimitivesEntity) Execute(state dtrules.State) error {
	return state.DataPush(p)
}

// ArrayExecute pushes this onto the data stack
func (p *PrimitivesEntity) ArrayExecute(state dtrules.State) error {
	return state.DataPush(p)
}

// GetExecutable returns this entity
func (p *PrimitivesEntity) GetExecutable() dtrules.Object {
	return p
}

// GetNonExecutable returns this entity
func (p *PrimitivesEntity) GetNonExecutable() dtrules.Object {
	return p
}

// IsExecutable returns false
func (p *PrimitivesEntity) IsExecutable() bool {
	return false
}

// Equals compares entities
func (p *PrimitivesEntity) Equals(obj dtrules.Object) (bool, error) {
	return p == obj, nil
}

// Compare is not supported
func (p *PrimitivesEntity) Compare(obj dtrules.Object) (int, error) {
	return 0, dtrules.UndefinedError("Compare", "Entity does not support Compare")
}

// StringValue returns the name
func (p *PrimitivesEntity) StringValue() string {
	return p.name.StringValue()
}

// PostFix returns the postfix representation
func (p *PrimitivesEntity) PostFix() string {
	return p.name.StringValue()
}

// Clone returns this entity
func (p *PrimitivesEntity) Clone(session dtrules.Session) (dtrules.Object, error) {
	return p, nil
}

// RClone returns this entity
func (p *PrimitivesEntity) RClone() dtrules.Object {
	return p
}

// RStringValue returns an RString
func (p *PrimitivesEntity) RStringValue() *dtrules.RString {
	return dtrules.NewRString(p.name.StringValue())
}

// Entity interface methods

// GetName returns the entity name
func (p *PrimitivesEntity) GetName() *dtrules.RName {
	return p.name
}

// Get retrieves an operator by name
func (p *PrimitivesEntity) Get(name *dtrules.RName) (dtrules.Object, error) {
	op, ok := Get(name)
	if ok {
		return op, nil
	}
	return nil, nil
}

// Put is not supported for primitives
func (p *PrimitivesEntity) Put(name *dtrules.RName, value dtrules.Object) error {
	return dtrules.UndefinedError("Put", "Cannot modify primitives entity")
}

// ContainsAttribute returns true if the operator exists
func (p *PrimitivesEntity) ContainsAttribute(name *dtrules.RName) bool {
	_, ok := Get(name)
	return ok
}

// GetID returns the entity ID
func (p *PrimitivesEntity) GetID() int {
	return 1 // Fixed ID for primitives
}

// GetAttributeNames returns all operator names
func (p *PrimitivesEntity) GetAttributeNames() []*dtrules.RName {
	operatorsMu.RLock()
	defer operatorsMu.RUnlock()
	names := make([]*dtrules.RName, 0, len(operators))
	for name := range operators {
		names = append(names, name)
	}
	return names
}

// REntityValue returns this entity
func (p *PrimitivesEntity) REntityValue() (dtrules.Entity, error) {
	return p, nil
}

// Remaining Object interface methods

func (p *PrimitivesEntity) IntValue() (int, error) {
	return 0, dtrules.ConversionError("IntValue", "No Integer value exists for entity")
}

func (p *PrimitivesEntity) LongValue() (int64, error) {
	return 0, dtrules.ConversionError("LongValue", "No Long value exists for entity")
}

func (p *PrimitivesEntity) DoubleValue() (float64, error) {
	return 0, dtrules.ConversionError("DoubleValue", "No Double value exists for entity")
}

func (p *PrimitivesEntity) BooleanValue() (bool, error) {
	return false, dtrules.ConversionError("BooleanValue", "No Boolean value exists for entity")
}

func (p *PrimitivesEntity) TimeValue() (time.Time, error) {
	return time.Time{}, dtrules.ConversionError("TimeValue", "No Time value exists for entity")
}

func (p *PrimitivesEntity) ArrayValue() ([]dtrules.Object, error) {
	return nil, dtrules.ConversionError("ArrayValue", "No Array value exists for entity")
}

func (p *PrimitivesEntity) TableValue() (map[dtrules.Object]dtrules.Object, error) {
	return nil, dtrules.ConversionError("TableValue", "No Table value exists for entity")
}

func (p *PrimitivesEntity) RIntegerValue() (*dtrules.RInteger, error) {
	return nil, dtrules.ConversionError("RIntegerValue", "No Integer value exists for entity")
}

func (p *PrimitivesEntity) RDoubleValue() (*dtrules.RDouble, error) {
	return nil, dtrules.ConversionError("RDoubleValue", "No Double value exists for entity")
}

func (p *PrimitivesEntity) RBooleanValue() (*dtrules.RBoolean, error) {
	return nil, dtrules.ConversionError("RBooleanValue", "No Boolean value exists for entity")
}

func (p *PrimitivesEntity) RNameValue() (*dtrules.RName, error) {
	return nil, dtrules.ConversionError("RNameValue", "No Name value exists for entity")
}

func (p *PrimitivesEntity) RArrayValue() (*dtrules.RArray, error) {
	return nil, dtrules.ConversionError("RArrayValue", "No Array value exists for entity")
}

func (p *PrimitivesEntity) RTableValue() (*dtrules.RTable, error) {
	return nil, dtrules.ConversionError("RTableValue", "No Table value exists for entity")
}

func (p *PrimitivesEntity) RTimeValue() (*dtrules.RDate, error) {
	return nil, dtrules.ConversionError("RTimeValue", "No Time value exists for entity")
}

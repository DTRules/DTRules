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
	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// Compile-time check that DTState implements dtrules.Runtime.
var _ dtrules.Runtime = (*DTState)(nil)

// =============================================================================
// RuntimeInit implementation — push state into the runtime before execution
// =============================================================================

// PushEntity pushes an entity onto the entity stack.
func (s *DTState) PushEntity(entity dtrules.Entity) error {
	return s.EntityPush(entity)
}

// PushValue pushes a value onto the value stack.
func (s *DTState) PushValue(v dtrules.Value) error {
	return s.ValuePush(v)
}

// PushData pushes an object onto the data stack.
func (s *DTState) PushData(obj dtrules.Object) error {
	return s.DataPush(obj)
}

// SetSession sets the session reference.
func (s *DTState) SetSession(session dtrules.Session) {
	s.session = session
}

// Note: SetOperatorTable is already implemented in state.go

// =============================================================================
// RuntimeQuery implementation — extract results after execution
// =============================================================================

// EntityByIndex returns the entity at the given depth from the top.
// 0 returns the top entity.
func (s *DTState) EntityByIndex(index int) (dtrules.Entity, error) {
	return s.EntityFetch(index)
}

// EntityStackDepth returns the current entity stack depth.
func (s *DTState) EntityStackDepth() int {
	return s.EntityDepth()
}

// PopValue pops and returns a value from the value stack.
func (s *DTState) PopValue() (dtrules.Value, error) {
	return s.ValuePop()
}

// PeekValue returns the top value without removing it.
func (s *DTState) PeekValue() (dtrules.Value, error) {
	return s.ValuePeek()
}

// ValueStackSize returns the current value stack depth.
func (s *DTState) ValueStackSize() int {
	return s.ValueStackDepth()
}

// PopData pops and returns an object from the data stack.
func (s *DTState) PopData() (dtrules.Object, error) {
	return s.DataPop()
}

// PeekData returns the top data stack object without removing it.
func (s *DTState) PeekData() (dtrules.Object, error) {
	return s.DataPeek()
}

// DataStackSize returns the current data stack depth.
func (s *DTState) DataStackSize() int {
	return s.DataStackDepth()
}

// FindValue looks up a name in the entity stack context and returns
// the value. Returns ValueNull if not found.
func (s *DTState) FindValue(name *dtrules.RName) (dtrules.Value, error) {
	obj, err := s.Find(name)
	if err != nil {
		return dtrules.ValueNull, err
	}
	if obj == nil {
		return dtrules.ValueNull, nil
	}
	return dtrules.ValueFromObject(obj), nil
}

// QueryEntity locates the entity containing the named attribute.
func (s *DTState) QueryEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return s.FindEntity(name)
}

// =============================================================================
// Runtime interface — execution
// =============================================================================

// Name returns "go" for the Go interpreter runtime.
func (s *DTState) Name() string {
	return "go"
}

// GetState returns this DTState as a dtrules.State interface.
func (s *DTState) GetState() dtrules.State {
	return s
}

// Note: ExecuteBytecode is already implemented in vm.go

// =============================================================================
// GoRuntimeFactory creates Go runtime instances.
// =============================================================================

// GoRuntimeFactory creates DTState-based runtimes for Go execution.
type GoRuntimeFactory struct{}

// Name returns "go".
func (f *GoRuntimeFactory) Name() string {
	return "go"
}

// CreateRuntime creates a new DTState runtime for the given session.
func (f *GoRuntimeFactory) CreateRuntime(session dtrules.Session) (dtrules.Runtime, error) {
	return NewDTState(session), nil
}

// Compile-time check that GoRuntimeFactory implements dtrules.RuntimeFactory.
var _ dtrules.RuntimeFactory = (*GoRuntimeFactory)(nil)

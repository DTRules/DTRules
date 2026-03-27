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
	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
)

// Factory creates DTState instances with the NativeASM bytecode executor.
type Factory struct{}

// NewFactory returns a new NativeASM runtime factory.
func NewFactory() *Factory { return &Factory{} }

// Name returns the runtime name.
func (f *Factory) Name() string { return "native-asm" }

// CreateState creates a new DTState with the NativeASM executor set.
func (f *Factory) CreateState(session dtrules.Session) (dtrules.State, error) {
	state := interpreter.NewDTState(session)
	state.SetBytecodeExecutor(NewExecutor())
	return state, nil
}

// Compile-time check
var _ dtrules.RuntimeFactory = (*Factory)(nil)

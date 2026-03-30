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

import "github.com/DTRules/DTRules/pkg/dtrules"

// GoRuntimeFactory creates DTState instances for the default Go interpreter runtime.
type GoRuntimeFactory struct{}

// NewGoRuntimeFactory returns a new GoRuntimeFactory.
func NewGoRuntimeFactory() *GoRuntimeFactory { return &GoRuntimeFactory{} }

// Name returns the runtime name.
func (f *GoRuntimeFactory) Name() string { return "go" }

// CreateState creates a new DTState for the given session.
func (f *GoRuntimeFactory) CreateState(session dtrules.Session) (dtrules.State, error) {
	return NewDTState(session), nil
}

// Compile-time check
var _ dtrules.RuntimeFactory = (*GoRuntimeFactory)(nil)

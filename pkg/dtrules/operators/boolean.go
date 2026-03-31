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

package operators

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	// Boolean operations
	Register("and", opAnd)
	Register("or", opOr)
	Register("not", opNot)
	Register("xor", opXor)

	// Comparison operations
	Register(">", opGt)
	Alias(">", "gt")
	Register("<", opLt)
	Alias("<", "lt")
	Register(">=", opGe)
	Alias(">=", "ge")
	Register("<=", opLe)
	Alias("<=", "le")
	Register("==", opEq)
	Alias("==", "eq")
	Register("!=", opNe)
	Alias("!=", "ne")
	Alias("!=", "neq")
	Register("beq", opBeq) // Boolean equals

	// Null check
	Register("isnull", opIsNull)
	Register("notnull", opNotNull)

	// Boolean constants
	RegisterConstant("true", dtrules.True)
	RegisterConstant("false", dtrules.False)
}

// RegisterConstant registers a constant value as an operator.
func RegisterConstant(name string, value dtrules.Object) {
	Register(name, func(state dtrules.State) error {
		return state.DataPush(value)
	})
}

// opAnd performs logical AND: ( a b -- a&&b )
func opAnd(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.BooleanValue()
	if err != nil {
		return err
	}
	bVal, err := b.BooleanValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(aVal && bVal))
}

// opOr performs logical OR: ( a b -- a||b )
func opOr(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.BooleanValue()
	if err != nil {
		return err
	}
	bVal, err := b.BooleanValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(aVal || bVal))
}

// opNot performs logical NOT: ( a -- !a )
func opNot(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.BooleanValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(!aVal))
}

// opXor performs logical XOR: ( a b -- a^b )
func opXor(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.BooleanValue()
	if err != nil {
		return err
	}
	bVal, err := b.BooleanValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(aVal != bVal))
}

// opGt performs greater than comparison: ( a b -- a>b )
func opGt(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	cmp, err := a.Compare(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(cmp > 0))
}

// opLt performs less than comparison: ( a b -- a<b )
func opLt(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	cmp, err := a.Compare(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(cmp < 0))
}

// opGe performs greater than or equal comparison: ( a b -- a>=b )
func opGe(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	cmp, err := a.Compare(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(cmp >= 0))
}

// opLe performs less than or equal comparison: ( a b -- a<=b )
func opLe(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	cmp, err := a.Compare(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(cmp <= 0))
}

// opEq performs equality comparison: ( a b -- a==b )
func opEq(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	eq, err := a.Equals(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(eq))
}

// opNe performs inequality comparison: ( a b -- a!=b )
func opNe(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	eq, err := a.Equals(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(!eq))
}

// opBeq performs boolean equality comparison: ( a b -- a==b )
func opBeq(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.BooleanValue()
	if err != nil {
		return err
	}
	bVal, err := b.BooleanValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(aVal == bVal))
}

// opIsNull checks if value is null: ( a -- isnull )
func opIsNull(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(a.Type() == dtrules.TypeNull))
}

// opNotNull checks if value is not null: ( a -- notnull )
func opNotNull(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(a.Type() != dtrules.TypeNull))
}

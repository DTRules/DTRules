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
	"math/big"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	// BigInt arithmetic operations
	Register("b+", opBigAdd)
	Alias("b+", "bigadd")

	Register("b-", opBigSub)
	Alias("b-", "bigsub")

	Register("b*", opBigMul)
	Alias("b*", "bigmul")

	Register("b/", opBigDiv)
	Alias("b/", "bigdiv")

	Register("bmod", opBigMod)

	Register("babs", opBigAbs)

	Register("bnegate", opBigNegate)

	// BigInt comparison operations
	Register("b==", opBigEqual)

	Register("b!=", opBigNotEqual)
	Alias("b!=", "b<>")

	Register("b>", opBigGreater)

	Register("b>=", opBigGreaterEqual)

	Register("b<", opBigLess)

	Register("b<=", opBigLessEqual)

	// Type conversion
	Register("cvbi", opCvBigInt)
}

// opBigAdd adds two BigInt values: ( a b -- a+b )
func opBigAdd(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := new(big.Int).Add(aVal.BigIntValue(), bVal.BigIntValue())
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigSub subtracts two BigInt values: ( a b -- a-b )
func opBigSub(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := new(big.Int).Sub(aVal.BigIntValue(), bVal.BigIntValue())
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigMul multiplies two BigInt values: ( a b -- a*b )
func opBigMul(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := new(big.Int).Mul(aVal.BigIntValue(), bVal.BigIntValue())
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigDiv divides two BigInt values (integer division): ( a b -- a/b )
func opBigDiv(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	bInt := bVal.BigIntValue()
	if bInt.Sign() == 0 {
		return dtrules.NewRulesError("Math Exception", "b/", "Division by zero")
	}
	result := new(big.Int).Div(aVal.BigIntValue(), bInt)
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigMod computes modulo of two BigInt values: ( a b -- a%b )
func opBigMod(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	bInt := bVal.BigIntValue()
	if bInt.Sign() == 0 {
		return dtrules.NewRulesError("Division By Zero", "bmod", "cannot mod by zero")
	}
	result := new(big.Int).Mod(aVal.BigIntValue(), bInt)
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigAbs returns absolute value of BigInt: ( a -- |a| )
func opBigAbs(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	result := new(big.Int).Abs(aVal.BigIntValue())
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigNegate negates a BigInt: ( a -- -a )
func opBigNegate(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	result := new(big.Int).Neg(aVal.BigIntValue())
	return state.DataPush(dtrules.GetRBigIntValue(result))
}

// opBigEqual compares two BigInt values for equality: ( a b -- bool )
func opBigEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) == 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opBigNotEqual compares two BigInt values for inequality: ( a b -- bool )
func opBigNotEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) != 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opBigGreater compares if a > b: ( a b -- bool )
func opBigGreater(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) > 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opBigGreaterEqual compares if a >= b: ( a b -- bool )
func opBigGreaterEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) >= 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opBigLess compares if a < b: ( a b -- bool )
func opBigLess(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) < 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opBigLessEqual compares if a <= b: ( a b -- bool )
func opBigLessEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBigIntValue()
	if err != nil {
		return err
	}
	result := aVal.BigIntValue().Cmp(bVal.BigIntValue()) <= 0
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opCvBigInt converts a value to BigInt: ( value -- bigint )
func opCvBigInt(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	return state.DataPush(aVal)
}

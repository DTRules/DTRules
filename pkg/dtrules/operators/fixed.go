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
	"math"
	"math/big"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	// Fixed-point arithmetic
	Register("fp+", opFixedAdd)
	Alias("fp+", "fpadd")

	Register("fp-", opFixedSub)
	Alias("fp-", "fpsub")

	Register("fp*", opFixedMul)
	Alias("fp*", "fpmul")

	Register("fp/", opFixedDiv)
	Alias("fp/", "fpdiv")

	Register("fpabs", opFixedAbs)
	Register("fpnegate", opFixedNegate)
	Register("fptrunc", opFixedTrunc)
	Register("fpmin", opFixedMin)
	Register("fpmax", opFixedMax)

	// Fixed-point comparison
	Register("fp==", opFixedEqual)

	Register("fp!=", opFixedNotEqual)
	Alias("fp!=", "fp<>")

	Register("fp>", opFixedGreater)
	Register("fp>=", opFixedGreaterEqual)
	Register("fp<", opFixedLess)
	Register("fp<=", opFixedLessEqual)

	// Explicit cast to fixed-point (the opt-in for double → fixed).
	Register("cvfp", opCvFixed)
}

// popFixedPair pops b then a from the data stack and promotes both to RFixed.
// Integer and bigint operands are auto-promoted; double operands return an
// error pointing to the cvfp cast.
func popFixedPair(state dtrules.State) (*dtrules.RFixed, *dtrules.RFixed, error) {
	b, err := state.DataPop()
	if err != nil {
		return nil, nil, err
	}
	a, err := state.DataPop()
	if err != nil {
		return nil, nil, err
	}
	aFp, err := dtrules.PromoteToRFixed(a)
	if err != nil {
		return nil, nil, err
	}
	bFp, err := dtrules.PromoteToRFixed(b)
	if err != nil {
		return nil, nil, err
	}
	return aFp, bFp, nil
}

func popFixedOne(state dtrules.State) (*dtrules.RFixed, error) {
	a, err := state.DataPop()
	if err != nil {
		return nil, err
	}
	return dtrules.PromoteToRFixed(a)
}

// opFixedAdd adds two fixed values: ( a b -- a+b )
func opFixedAdd(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	r, err := a.Add(b)
	if err != nil {
		return err
	}
	return state.DataPush(r)
}

// opFixedSub subtracts two fixed values: ( a b -- a-b )
func opFixedSub(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	r, err := a.Sub(b)
	if err != nil {
		return err
	}
	return state.DataPush(r)
}

// opFixedMul multiplies two fixed values with truncate-toward-zero: ( a b -- a*b )
func opFixedMul(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	r, err := a.Mul(b)
	if err != nil {
		return err
	}
	return state.DataPush(r)
}

// opFixedDiv divides two fixed values with truncate-toward-zero: ( a b -- a/b )
func opFixedDiv(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	r, err := a.Div(b)
	if err != nil {
		return err
	}
	return state.DataPush(r)
}

// opFixedAbs returns absolute value: ( a -- |a| )
func opFixedAbs(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Abs())
}

// opFixedNegate negates a fixed value: ( a -- -a )
func opFixedNegate(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Neg())
}

// opFixedTrunc truncates fractional part toward zero: ( a -- trunc(a) )
func opFixedTrunc(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Trunc())
}

// opFixedMin returns the lesser of two fp values: ( a b -- min(a,b) )
func opFixedMin(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	c, err := a.Compare(b)
	if err != nil {
		return err
	}
	if c <= 0 {
		return state.DataPush(a)
	}
	return state.DataPush(b)
}

// opFixedMax returns the greater of two fp values: ( a b -- max(a,b) )
func opFixedMax(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	c, err := a.Compare(b)
	if err != nil {
		return err
	}
	if c >= 0 {
		return state.DataPush(a)
	}
	return state.DataPush(b)
}

func opFixedCompare(state dtrules.State, want func(int) bool) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	c, err := a.Compare(b)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(want(c)))
}

func opFixedEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c == 0 })
}

func opFixedNotEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c != 0 })
}

func opFixedGreater(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c > 0 })
}

func opFixedGreaterEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c >= 0 })
}

func opFixedLess(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c < 0 })
}

func opFixedLessEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c <= 0 })
}

// opCvFixed converts the top stack value to RFixed: ( value -- fixed )
//
// Conversion rules:
//   - fixed: identity
//   - integer, bigint: exact (bigint range-checked against the 256-bit mantissa)
//   - double: truncated toward zero onto the 10^-8 grid, range-checked
//   - string: parsed as a decimal literal
//
// Anything else is a type error.
func opCvFixed(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	switch v := obj.(type) {
	case *dtrules.RFixed:
		return state.DataPush(v)
	case *dtrules.RInteger:
		fp, err := dtrules.GetRFixedFromInt64(v.Value())
		if err != nil {
			return err
		}
		return state.DataPush(fp)
	case *dtrules.RBigInt:
		fp, err := dtrules.GetRFixedFromBigInt(v.BigIntValue())
		if err != nil {
			return err
		}
		return state.DataPush(fp)
	case *dtrules.RDouble:
		fp, err := fixedFromDouble(v.Value())
		if err != nil {
			return err
		}
		return state.DataPush(fp)
	case *dtrules.RString:
		fp, err := dtrules.GetRFixedFromString(v.StringValue())
		if err != nil {
			return err
		}
		return state.DataPush(fp)
	default:
		return dtrules.NewRulesError("Math Exception", "cvfp",
			"cannot convert "+obj.Type().String()+" to fixed")
	}
}

// fixedFromDouble snaps a float64 onto the 10^-8 grid by multiplying in
// big.Float precision and truncating toward zero, then range-checking.
func fixedFromDouble(d float64) (*dtrules.RFixed, error) {
	if math.IsNaN(d) || math.IsInf(d, 0) {
		return nil, dtrules.NewRulesError("Math Exception", "cvfp",
			"cannot convert NaN or Inf to fixed")
	}
	f := new(big.Float).SetPrec(256).SetFloat64(d)
	scale := new(big.Float).SetPrec(256).SetInt64(100_000_000)
	f.Mul(f, scale)
	mantissa, _ := f.Int(nil) // big.Float.Int truncates toward zero
	return dtrules.GetRFixedFromMantissa(mantissa)
}


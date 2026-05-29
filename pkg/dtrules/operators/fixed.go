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

	// fphalfup/ : ( a b -- round_half_up(a/b) ). Round-half-away-from-zero
	// at the 10^-8 mantissa grid, distinct from fp/ which truncates toward
	// zero. The pure-fp idiom (a + b/2)/b adds half-the-grid-digit in
	// **value** units (0.5 of the result's units), not half of the
	// underlying mantissa, so it can't express a one-mantissa-unit rounding
	// bias. This operator does the rounding at mantissa precision.
	Register("fphalfup/", opFixedDivRoundHalfUp)
	Alias("fphalfup/", "fpdivhalfup")

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

// opFixedAdd: ( a b -- a+b ) adds two RFixed values exactly.
// Both operands must be fp; int/bigint are auto-promoted via popFixedPair.
// Returns a Math Exception if the resulting mantissa exceeds |m| < 2^255.
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

// opFixedSub: ( a b -- a-b ) subtracts two RFixed values exactly.
// Both operands must be fp; int/bigint are auto-promoted. Returns a
// Math Exception if the resulting mantissa exceeds |m| < 2^255.
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

// opFixedMul: ( a b -- a*b ) multiplies two RFixed values with truncate-
// toward-zero rescaling onto the 10^-8 grid. Both operands must be fp;
// int/bigint are auto-promoted. Returns a Math Exception if the final
// mantissa exceeds |m| < 2^255.
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

// opFixedDiv: ( a b -- a/b ) divides two RFixed values with truncate-
// toward-zero rescaling onto the 10^-8 grid. Both operands must be fp;
// int/bigint are auto-promoted. Returns a Math Exception on divide by
// zero or if the result exceeds |m| < 2^255.
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

// opFixedDivRoundHalfUp: ( a b -- round_half_up(a/b) ) divides on the
// 10^-8 grid with half rounding away from zero. Both operands must be
// fp; int/bigint are auto-promoted. Returns a Math Exception on divide
// by zero or if the result exceeds |m| < 2^255.
func opFixedDivRoundHalfUp(state dtrules.State) error {
	a, b, err := popFixedPair(state)
	if err != nil {
		return err
	}
	r, err := a.DivRoundHalfUp(b)
	if err != nil {
		return err
	}
	return state.DataPush(r)
}

// opFixedAbs: ( a -- |a| ) returns the absolute value of an RFixed.
// Because the mantissa range is symmetric (|m| < 2^255), abs is always
// representable and never overflows.
func opFixedAbs(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Abs())
}

// opFixedNegate: ( a -- -a ) negates an RFixed. The mantissa range is
// symmetric, so negation never overflows.
func opFixedNegate(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Neg())
}

// opFixedTrunc: ( a -- trunc(a) ) truncates the fractional part toward
// zero. Returns an RFixed whose mantissa is a multiple of 10^8 (i.e.
// the value has exactly .00000000 fractional digits).
func opFixedTrunc(state dtrules.State) error {
	a, err := popFixedOne(state)
	if err != nil {
		return err
	}
	return state.DataPush(a.Trunc())
}

// opFixedMin: ( a b -- min(a,b) ) returns the lesser of two RFixed values.
// Both operands must be fp; int/bigint are auto-promoted. When the two
// values compare equal, the first operand (a) is returned.
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

// opFixedMax: ( a b -- max(a,b) ) returns the greater of two RFixed values.
// Both operands must be fp; int/bigint are auto-promoted. When the two
// values compare equal, the first operand (a) is returned.
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

// opFixedEqual: ( a b -- bool ) exact mantissa equality. Both operands
// must be fp; int/bigint are auto-promoted.
func opFixedEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c == 0 })
}

// opFixedNotEqual: ( a b -- bool ) exact mantissa inequality.
func opFixedNotEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c != 0 })
}

// opFixedGreater: ( a b -- bool ) returns a > b.
func opFixedGreater(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c > 0 })
}

// opFixedGreaterEqual: ( a b -- bool ) returns a >= b.
func opFixedGreaterEqual(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c >= 0 })
}

// opFixedLess: ( a b -- bool ) returns a < b.
func opFixedLess(state dtrules.State) error {
	return opFixedCompare(state, func(c int) bool { return c < 0 })
}

// opFixedLessEqual: ( a b -- bool ) returns a <= b.
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


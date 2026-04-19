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
	"math"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	// Integer operations
	Register("+", opAdd)
	Alias("+", "ladd")

	Register("-", opSub)
	Alias("-", "lsub")

	Register("*", opMul)
	Alias("*", "lmul")

	Register("/", opDiv)
	Alias("/", "div")
	Alias("/", "ldiv")

	Register("mod", opMod)

	Register("abs", opAbs)
	Register("negate", opNegate)

	// Float operations
	Register("f+", opFAdd)
	Alias("f+", "fadd")

	Register("f-", opFSub)
	Alias("f-", "fsub")

	Register("f*", opFMul)
	Alias("f*", "fmul")

	Register("fdiv", opFDiv)
	Alias("fdiv", "f/")

	Register("fabs", opFAbs)
	Register("fnegate", opFNegate)

	Register("roundto", opRoundTo)
	Register("ceiling", opCeiling)
	Alias("ceiling", "ceil")
	Register("floor", opFloor)
	Register("fmax", opFMax)
	Register("fmin", opFMin)
	Register("max", opMax)
	Register("min", opMin)
}

// opAdd adds two integers: ( a b -- a+b )
// Returns error on overflow.
func opAdd(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.LongValue()
	if err != nil {
		return err
	}
	bVal, err := b.LongValue()
	if err != nil {
		return err
	}
	result := aVal + bVal
	// Check for overflow: if signs of inputs are same but result sign differs
	if (aVal > 0 && bVal > 0 && result < 0) || (aVal < 0 && bVal < 0 && result > 0) {
		return dtrules.NewRulesError("Math Exception", "+", "integer overflow")
	}
	return state.DataPush(dtrules.GetRIntegerValue(result))
}

// opSub subtracts two integers: ( a b -- a-b )
// Returns error on overflow.
func opSub(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.LongValue()
	if err != nil {
		return err
	}
	bVal, err := b.LongValue()
	if err != nil {
		return err
	}
	result := aVal - bVal
	// Check for overflow: subtracting negative is like adding positive, and vice versa
	if (bVal < 0 && aVal > 0 && result < 0) || (bVal > 0 && aVal < 0 && result > 0) {
		return dtrules.NewRulesError("Math Exception", "-", "integer overflow")
	}
	return state.DataPush(dtrules.GetRIntegerValue(result))
}

// opMul multiplies two integers: ( a b -- a*b )
// Returns error on overflow.
func opMul(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.LongValue()
	if err != nil {
		return err
	}
	bVal, err := b.LongValue()
	if err != nil {
		return err
	}
	result := aVal * bVal
	// Check for overflow: if b != 0 and result/b != a, overflow occurred
	if bVal != 0 && result/bVal != aVal {
		return dtrules.NewRulesError("Math Exception", "*", "integer overflow")
	}
	return state.DataPush(dtrules.GetRIntegerValue(result))
}

// opDiv divides two integers: ( a b -- a/b )
func opDiv(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.LongValue()
	if err != nil {
		return err
	}
	bVal, err := b.LongValue()
	if err != nil {
		return err
	}
	if bVal == 0 {
		return dtrules.NewRulesError("Math Exception", "/", "Division by zero")
	}
	return state.DataPush(dtrules.GetRIntegerValue(aVal / bVal))
}

// opMod computes modulo of two integers: ( a b -- a%b )
func opMod(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}

	bVal, err := b.LongValue()
	if err != nil {
		return err
	}
	if bVal == 0 {
		return dtrules.NewRulesError("Division By Zero", "mod", "cannot mod by zero")
	}

	aVal, err := a.LongValue()
	if err != nil {
		return err
	}

	return state.DataPush(dtrules.GetRIntegerValue(aVal % bVal))
}

// opAbs returns absolute value of integer: ( a -- |a| )
func opAbs(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.IntValue()
	if err != nil {
		return err
	}
	if aVal < 0 {
		aVal = -aVal
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(aVal))
}

// opNegate negates an integer: ( a -- -a )
func opNegate(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.IntValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(-aVal))
}

// opFAdd adds two doubles: ( a b -- a+b )
func opFAdd(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(aVal + bVal))
}

// opFSub subtracts two doubles: ( a b -- a-b )
func opFSub(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(aVal - bVal))
}

// opFMul multiplies two doubles: ( a b -- a*b )
func opFMul(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(aVal * bVal))
}

// opFDiv divides two doubles: ( a b -- a/b )
func opFDiv(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	if bVal == 0 {
		return dtrules.NewRulesError("Math Exception", "f/", "Division by zero")
	}
	return state.DataPush(dtrules.GetRDoubleValue(aVal / bVal))
}

// opFAbs returns absolute value of double: ( a -- |a| )
func opFAbs(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(math.Abs(aVal)))
}

// opFNegate negates a double: ( a -- -a )
func opFNegate(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(-aVal))
}

// opCeiling rounds up to nearest integer: ( a -- ceil(a) )
func opCeiling(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(math.Ceil(aVal)))
}

// opFloor rounds down to nearest integer: ( a -- floor(a) )
func opFloor(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(math.Floor(aVal)))
}

// opRoundTo rounds a number: ( number #places boundary -- number2 )
func opRoundTo(state dtrules.State) error {
	boundaryObj, err := state.DataPop()
	if err != nil {
		return err
	}
	placesObj, err := state.DataPop()
	if err != nil {
		return err
	}
	numberObj, err := state.DataPop()
	if err != nil {
		return err
	}

	boundary, err := boundaryObj.DoubleValue()
	if err != nil {
		return err
	}
	places, err := placesObj.IntValue()
	if err != nil {
		return err
	}
	number, err := numberObj.DoubleValue()
	if err != nil {
		return err
	}

	round := func(n, b float64) float64 {
		v := float64(int(n)) // Integer portion
		if b >= 1 {
			return v
		}
		r := math.Abs(n - v) // Fractional portion
		if b <= 0 {
			if r > 0 {
				v++
			}
			return v
		}
		if r >= b {
			v++
		}
		return v
	}

	// Scale is 10^|places|, not 10*|places| (the historic bug). For
	// places=2 we want to round to the nearest 0.01, which means scaling
	// by 100 before rounding to integer and scaling back down. The old
	// `10*places` gave 20 instead of 100, so `rounded to 2 decimal places`
	// actually rounded to the nearest 0.05 — wrong. The places=0 branch
	// also had a division-by-zero (scale was -10*0 = 0).
	//
	// places >= 0: multiply up by 10^places, round, divide back.
	// places  < 0: divide down by 10^|places| to round to the nearest
	//              10^|places|, then multiply back.
	if places < 0 {
		scale := math.Pow(10, float64(-places))
		number /= scale
		number = round(number, boundary)
		number *= scale
	} else {
		scale := math.Pow(10, float64(places))
		number *= scale
		number = round(number, boundary)
		number /= scale
	}

	return state.DataPush(dtrules.GetRDoubleValue(number))
}

// opFMax returns the maximum of two doubles: ( a b -- max(a,b) )
func opFMax(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(math.Max(aVal, bVal)))
}

// opFMin returns the minimum of two doubles: ( a b -- min(a,b) )
func opFMin(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.DoubleValue()
	if err != nil {
		return err
	}
	bVal, err := b.DoubleValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRDoubleValue(math.Min(aVal, bVal)))
}

// opMax returns the maximum of two integers: ( a b -- max(a,b) )
func opMax(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.IntValue()
	if err != nil {
		return err
	}
	bVal, err := b.IntValue()
	if err != nil {
		return err
	}
	if aVal > bVal {
		return state.DataPush(dtrules.GetRIntegerValueFromInt(aVal))
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(bVal))
}

// opMin returns the minimum of two integers: ( a b -- min(a,b) )
func opMin(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.IntValue()
	if err != nil {
		return err
	}
	bVal, err := b.IntValue()
	if err != nil {
		return err
	}
	if aVal < bVal {
		return state.DataPush(dtrules.GetRIntegerValueFromInt(aVal))
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(bVal))
}

// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package operators

import (
	"math/big"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// runCvb pushes v onto a fresh state, runs cvb, returns the popped result
// or the error.
func runCvb(t *testing.T, v dtrules.Object) (*dtrules.RBoolean, error) {
	t.Helper()
	state := newTestState()
	state.DataPush(v)
	op, _ := Get(dtrules.GetRName("cvb"))
	if err := op.Execute(state); err != nil {
		return nil, err
	}
	res, err := state.DataPop()
	if err != nil {
		return nil, err
	}
	return res.(*dtrules.RBoolean), nil
}

func TestCvbBoolPassthrough(t *testing.T) {
	b, err := runCvb(t, dtrules.True)
	if err != nil || b != dtrules.True {
		t.Errorf("cvb(true) = (%v, %v), want (true, nil)", b, err)
	}
	b, err = runCvb(t, dtrules.False)
	if err != nil || b != dtrules.False {
		t.Errorf("cvb(false) = (%v, %v), want (false, nil)", b, err)
	}
}

func TestCvbStringAccepted(t *testing.T) {
	trueStrings := []string{"true", "True", "TRUE", "yes", "YES", " y ", "t", "1"}
	for _, s := range trueStrings {
		b, err := runCvb(t, dtrules.NewRString(s))
		if err != nil || b != dtrules.True {
			t.Errorf("cvb(%q) = (%v, %v), want (true, nil)", s, b, err)
		}
	}
	falseStrings := []string{"false", "False", "FALSE", "no", "NO", "n", "f", "0"}
	for _, s := range falseStrings {
		b, err := runCvb(t, dtrules.NewRString(s))
		if err != nil || b != dtrules.False {
			t.Errorf("cvb(%q) = (%v, %v), want (false, nil)", s, b, err)
		}
	}
}

func TestCvbStringRejected(t *testing.T) {
	bad := []string{"", "maybe", "2", "truthy", "nope", " "}
	for _, s := range bad {
		_, err := runCvb(t, dtrules.NewRString(s))
		if err == nil {
			t.Errorf("cvb(%q) accepted; want error", s)
		}
	}
}

func TestCvbIntegerZeroIsFalse(t *testing.T) {
	b, err := runCvb(t, dtrules.GetRIntegerValueFromInt(0))
	if err != nil || b != dtrules.False {
		t.Errorf("cvb(int 0) = (%v, %v), want (false, nil)", b, err)
	}
}

func TestCvbIntegerNonzeroIsTrue(t *testing.T) {
	for _, n := range []int{1, -1, 42, -999} {
		b, err := runCvb(t, dtrules.GetRIntegerValueFromInt(n))
		if err != nil || b != dtrules.True {
			t.Errorf("cvb(int %d) = (%v, %v), want (true, nil)", n, b, err)
		}
	}
}

func TestCvbDoubleZeroIsFalse(t *testing.T) {
	b, err := runCvb(t, dtrules.GetRDoubleValue(0.0))
	if err != nil || b != dtrules.False {
		t.Errorf("cvb(double 0.0) = (%v, %v), want (false, nil)", b, err)
	}
}

func TestCvbDoubleNonzeroIsTrue(t *testing.T) {
	for _, f := range []float64{1.0, -1.0, 0.5, -0.0001} {
		b, err := runCvb(t, dtrules.GetRDoubleValue(f))
		if err != nil || b != dtrules.True {
			t.Errorf("cvb(double %v) = (%v, %v), want (true, nil)", f, b, err)
		}
	}
}

func TestCvbBigIntZeroIsFalse(t *testing.T) {
	b, err := runCvb(t, dtrules.GetRBigIntValue(big.NewInt(0)))
	if err != nil || b != dtrules.False {
		t.Errorf("cvb(bigint 0) = (%v, %v), want (false, nil)", b, err)
	}
}

func TestCvbBigIntNonzeroIsTrue(t *testing.T) {
	for _, n := range []int64{1, -1, 1 << 40} {
		b, err := runCvb(t, dtrules.GetRBigIntValue(big.NewInt(n)))
		if err != nil || b != dtrules.True {
			t.Errorf("cvb(bigint %d) = (%v, %v), want (true, nil)", n, b, err)
		}
	}
}

func TestCvbOtherTypesRejected(t *testing.T) {
	// null: not a boolean-coercible type. Any non-{bool, string, int, double,
	// bigint} should error.
	_, err := runCvb(t, dtrules.GetRNull())
	if err == nil {
		t.Error("cvb(null) accepted; want error")
	}
}

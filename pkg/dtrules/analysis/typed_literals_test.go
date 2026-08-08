// Copyright 2026 Paul Snow
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

package analysis

import (
	"fmt"
	"testing"
)

// TestTypedNumericLiteralsAreNotOperators pins #1006.
//
// `dtrules verify` reported the integer-form fixed-point literal `0fp` as an
// undefined operator while accepting `0.5fp` two lines below it in the same
// file. Both are valid — EL.g4's FP_LITERAL has three alternatives and
// `DIGIT+ 'fp'` is one of them — and the engine builds, loads and executes
// them correctly. On the Accumulate staking rules this was 44 uses across 8
// tables, which stopped verify being usable as the CI authoring-contract gate.
//
// `0.5fp` only appeared to work: it contains a `.`, so it fell through the
// dotted-field-reference branch. Right answer, wrong reason — which is why the
// decimal forms are pinned here too, and why the hex-bytes literal is, having
// had the identical defect without anyone having hit it yet.
func TestTypedNumericLiteralsAreNotOperators(t *testing.T) {
	literals := []string{
		// FP_LITERAL: DIGIT+ 'fp'
		"0fp", "12fp", "-3fp", "+3fp",
		// FP_LITERAL: DIGIT+ '.' DIGIT* 'fp'
		"0.5fp", "1.3fp", "2.fp",
		// FP_LITERAL: DIGIT* '.' DIGIT+ 'fp'
		".5fp",
		// EL is case-insensitive.
		"0FP", "0Fp",
		// HEX_BYTES_LITERAL: '0x' HEX_DIGIT*
		"0x1f", "0xdeadbeef", "0xDEADBEEF", "0x",
	}
	// Every single digit, because the reported symptom was specific to one
	// value (`0fp`) and "is it fixed for 5fp too?" is the obvious next
	// question. It is not digit-dependent, and this says so.
	for d := 0; d <= 9; d++ {
		literals = append(literals, fmt.Sprintf("%dfp", d))
	}
	literals = append(literals,
		"10fp", "100fp", "999fp", "123456789fp", "007fp", "00fp",
		"0.0fp", "5.0fp", "12.34fp", "0.00000001fp",
	)

	for _, tok := range literals {
		if operatorCandidate(tok) {
			t.Errorf("%q treated as an operator; it is a literal the compiler emits verbatim", tok)
		}
	}
	t.Logf("checked %d typed-literal forms", len(literals))
}

// TestTypedLiteralFixIsNotOverPermissive keeps the #1006 fix from swallowing
// real findings. An undefined operator is a hard gate failure, so widening the
// literal class must not turn identifiers into literals.
func TestTypedLiteralFixIsNotOverPermissive(t *testing.T) {
	operators := []string{
		"addto",  // a real operator
		"fp",     // the suffix alone is an identifier, not a literal
		"0fpx",   // suffix must end the token
		"x0fp",   // and must follow digits
		"0xg",    // g is not a hex digit
		"12fp34", // trailing junk
		"--3fp",  // one sign only
		"cvfp",   // the conversion operator, which is genuinely an operator
	}
	for _, tok := range operators {
		if !operatorCandidate(tok) {
			t.Errorf("%q exempted from the operator check; it is not a numeric literal", tok)
		}
	}
}

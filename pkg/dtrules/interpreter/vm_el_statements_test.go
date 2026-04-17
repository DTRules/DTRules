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
	"errors"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// TestOpError verifies that OpError pops the string and returns an ELStatementError
// with the correct message, distinguishable from other engine errors.
func TestOpError(t *testing.T) {
	state := NewDTState(&mockSession{})

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueString("STALE DATA"))
	bc.Emit(dtrules.OpError)

	err := state.ExecuteBytecode(bc)
	if err == nil {
		t.Fatal("expected error from OpError, got nil")
	}

	var elErr *dtrules.ELStatementError
	if !errors.As(err, &elErr) {
		t.Fatalf("expected ELStatementError, got %T: %v", err, err)
	}
	if elErr.Msg != "STALE DATA" {
		t.Errorf("expected message 'STALE DATA', got %q", elErr.Msg)
	}
}

// TestOpWarn verifies that OpWarn pops the string and returns nil (execution continues).
func TestOpWarn(t *testing.T) {
	state := NewDTState(&mockSession{})

	bc := dtrules.NewBytecodeChunk()
	bc.EmitPushConstant(dtrules.NewValueString("heads up"))
	bc.Emit(dtrules.OpWarn)

	err := state.ExecuteBytecode(bc)
	if err != nil {
		t.Fatalf("expected no error from OpWarn, got %v", err)
	}

	// Stack should be empty after warn pops the string
	if state.ValueStackDepth() != 0 {
		t.Errorf("expected empty stack after OpWarn, got depth %d", state.ValueStackDepth())
	}
}

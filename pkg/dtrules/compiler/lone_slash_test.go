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

package compiler

import (
	"bytes"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #1241: a lone "/" is the integer-division operator, not a literal
// name with an empty body. "/name" is still a literal name.

func TestCompile_LoneSlashIsDivision(t *testing.T) {
	c := newTestCompiler()
	result, err := c.Compile("6 2 / /q")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	arr, _ := result.ArrayValue()
	if len(arr) != 4 {
		t.Fatalf("got %d elements, want 4", len(arr))
	}
	if arr[2].Type() != dtrules.TypeOperator {
		t.Errorf(`"/" compiled to %v (%s), want the division operator`, arr[2].Type(), arr[2].StringValue())
	}
	if arr[3].Type() != dtrules.TypeName || arr[3].IsExecutable() || arr[3].StringValue() != "q" {
		t.Errorf(`"/q" compiled to %v %q (executable %v), want literal name q`,
			arr[3].Type(), arr[3].StringValue(), arr[3].IsExecutable())
	}
}

func TestCompileToBytecode_LoneSlashIsDivision(t *testing.T) {
	c := newTestCompiler()
	div, err := c.CompileToBytecode("6 2 /")
	if err != nil {
		t.Fatalf("CompileToBytecode: %v", err)
	}
	if n := len(div.Names()); n != 0 {
		t.Errorf(`"6 2 /" put %d names in the chunk, want 0 (the "/" became a name)`, n)
	}
	// Same chunk as "6 2 +" with the one opcode swapped.
	add, err := c.CompileToBytecode("6 2 +")
	if err != nil {
		t.Fatalf("CompileToBytecode: %v", err)
	}
	want := bytes.ReplaceAll(add.Code(), []byte{byte(dtrules.OpAdd)}, []byte{byte(dtrules.OpDiv)})
	if !bytes.Equal(div.Code(), want) {
		t.Errorf(`"6 2 /" code = %v, want %v`, div.Code(), want)
	}
}

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

package repository

import (
	"bytes"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Issue #1241: a lone "/" is the integer-division operator, not a literal
// name with an empty body. The repository's standalone bytecode reader must
// follow the same rule as the compiler package's readers.
func TestCompileExpression_LoneSlashIsDivision(t *testing.T) {
	div, err := compileExpression("6 2 /")
	if err != nil {
		t.Fatalf("compileExpression: %v", err)
	}
	if n := len(div.Names()); n != 0 {
		t.Errorf(`"6 2 /" put %d names in the chunk, want 0 (the "/" became a name)`, n)
	}
	add, err := compileExpression("6 2 +")
	if err != nil {
		t.Fatalf("compileExpression: %v", err)
	}
	want := bytes.ReplaceAll(add.Code(), []byte{byte(dtrules.OpAdd)}, []byte{byte(dtrules.OpDiv)})
	if !bytes.Equal(div.Code(), want) {
		t.Errorf(`"6 2 /" code = %v, want %v`, div.Code(), want)
	}

	lit, err := compileExpression("/q")
	if err != nil {
		t.Fatalf("compileExpression: %v", err)
	}
	if names := lit.Names(); len(names) != 1 || names[0].StringValue() != "q" {
		t.Errorf(`"/q" names = %v, want [q]`, names)
	}
}

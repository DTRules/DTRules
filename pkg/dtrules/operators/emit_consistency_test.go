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

package operators_test

import (
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// emitterSource is the EL postfix emitter, read relative to this package dir.
const emitterSource = "../compiler/el/postfix_emitter.go"

// numericLiteral matches an emitted value literal (integer or float), e.g.
// "0", "1.0", "0.5", "100.0", "-1".
var numericLiteral = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// nonOpEmits are emitter literals that are deliberately NOT operator-registry
// names: structural block markers and value literals pushed onto the stack, and
// decision-table control keywords consumed by the table interpreter rather than
// the operator table.
var nonOpEmits = map[string]bool{
	"{":  true, // block open
	"}":  true, // block close
	"{}": true, // empty block
	"true":  true, // boolean literals (values, not ops)
	"false": true,
	"null":  true,
	"otherwise":        true, // decision-table control markers, not registry ops
	"policystatements": true,
}

// knownUnimplementedEmits are operator names the emitter produces for EL
// features that have NO runtime implementation yet. They are listed explicitly
// — not silently skipped — so this test documents the gap and forces a
// deliberate decision when the feature lands. Entity relationships
// (`<s> of <entity>`, `<entity> has a <s>`) have no operator at all (#888).
var knownUnimplementedEmits = map[string]bool{
	"getrelationship": true, // #890: entity-relationship feature unimplemented
	"hasrelationship": true,
}

// TestEmittedOpsAreRegistered is the compiler↔runtime consistency guard: every
// operator literal the EL emitter can emit must resolve in the operator
// registry, otherwise any rule reaching that construct crashes at execute with
// "RName … was not defined" (the #878 / #888 class). Token-presence emitter
// tests cannot catch this because the postfix string is produced fine; only the
// runtime lookup fails. New emitted ops must be registered (or allowlisted here
// with a reason).
func TestEmittedOpsAreRegistered(t *testing.T) {
	data, err := os.ReadFile(emitterSource)
	if err != nil {
		t.Fatalf("read emitter source %s: %v", emitterSource, err)
	}
	re := regexp.MustCompile(`e\.emit\("([^"]+)"\)`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatalf("no e.emit(\"...\") literals found in %s — regex or path broken", emitterSource)
	}

	emitted := map[string]bool{}
	for _, m := range matches {
		emitted[m[1]] = true
	}

	var missing []string
	for op := range emitted {
		if numericLiteral.MatchString(op) || nonOpEmits[op] || knownUnimplementedEmits[op] {
			continue
		}
		if _, ok := operators.Get(dtrules.GetRName(op)); !ok {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	for _, op := range missing {
		t.Errorf("emitter emits %q but it is not registered — rules reaching it crash at runtime", op)
	}
}

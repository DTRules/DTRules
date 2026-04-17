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

package el

import (
	"strings"
	"testing"
)

func TestCompileErrorStatement(t *testing.T) {
	tests := []struct {
		name     string
		el       string
		expected string
	}{
		{
			name:     "error literal string",
			el:       `action error "bad";`,
			expected: `"bad" elstmterror`,
		},
		{
			name:     "warn literal string",
			el:       `action warn "heads up";`,
			expected: `"heads up" elstmtwarn`,
		},
		{
			name:     "error with string concatenation",
			el:       `action error "bad: " + field;`,
			expected: `"bad: " field strconcat elstmterror`,
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CompileAction(tt.el)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.el, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.el, result, expected)
			}
		})
	}
}

package authoring_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// TestCheckAction_MinMaxSyntax verifies all documented minimum/maximum syntax forms.
// Note: identifiers "a", "an", "the" are skipped as articles by the EL lexer,
// so we use "val1"/"val2" as representative identifiers.
func TestCheckAction_MinMaxSyntax(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		wantOp string
	}{
		// Natural language forms: the minimum/maximum of X and Y
		{"minimum of A and B (int)", "set result = the minimum of val1 and val2", "min"},
		{"maximum of A and B (int)", "set result = the maximum of val1 and val2", "max"},
		{"minimum of A and 0 (int)", "set result = the minimum of val1 and 0", "min"},
		{"maximum of 0 and A (int)", "set result = the maximum of 0 and val1", "max"},

		// Alias forms: smaller/larger of
		{"smaller of A and B (int)", "set result = smaller of val1 and val2", "min"},
		{"larger of A and B (int)", "set result = larger of val1 and val2", "max"},

		// Comma separator variants
		{"smaller of A, B (int)", "set result = smaller of val1, val2", "min"},
		{"larger of A, B (int)", "set result = larger of val1, val2", "max"},

		// Float variants (literal 0.0 forces float parse path)
		{"minimum of float and 0.0", "set result = the minimum of val1 and 0.0", "fmin"},
		{"maximum of 0.0 and float", "set result = the maximum of 0.0 and val1", "fmax"},
		{"smaller of float, float", "set result = smaller of val1, 0.0", "fmin"},
		{"larger of float, float", "set result = larger of val1, 0.0", "fmax"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postfix, err := authoring.CheckAction(tt.expr, nil)
			if err != nil {
				t.Errorf("CheckAction(%q) error: %v", tt.expr, err)
				return
			}
			if postfix == "" {
				t.Errorf("CheckAction(%q) returned empty postfix", tt.expr)
				return
			}
			if !strings.Contains(postfix, tt.wantOp) {
				t.Errorf("CheckAction(%q) postfix %q does not contain opcode %q", tt.expr, postfix, tt.wantOp)
			}
		})
	}
}

// TestCheckAction_MinMax_Issue623 is the exact reproducer from issue #623.
func TestCheckAction_MinMax_Issue623(t *testing.T) {
	_, err := authoring.CheckAction("set wp.ew = the minimum of wp.ew and wp.rc", nil)
	if err != nil {
		t.Errorf("issue #623 reproducer: unexpected error: %v", err)
	}
}

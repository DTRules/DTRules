package el

import (
	"strings"
	"testing"
)

// TestCreateEntityAs exercises the `create <type> as <local>` statement.
// Lowers to `/typeName createentity /localName xdef` so the local can then
// be the target of `set <local>.<field> = ...` and `add <local> to <coll>`
// statements that follow.
func TestCreateEntityAs(t *testing.T) {
	tests := []struct {
		name string
		el   string
		want string
	}{
		{
			name: "construct and bind",
			el:   `create state_tax_result as st_result;`,
			want: `/state_tax_result createentity /st_result xdef`,
		},
		{
			name: "construct then append",
			el: `create state_tax_result as st_result;` +
				`add st_result to job.state_tax_results;`,
			// `add X to <collection>` emits the same `addto` postfix the
			// hand-coded version used.
			want: `/state_tax_result createentity /st_result xdef ` +
				`st_result job.state_tax_results swap addto`,
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CompileAction(tt.el)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.el, err)
			}
			got = normalizeSpaces(got)
			want := normalizeSpaces(tt.want)
			if got != want {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.el, got, want)
			}
		})
	}
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

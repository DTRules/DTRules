package el

import (
	"strings"
	"testing"
)

// TestCreateEntityAs exercises the `create <type> as <local>` statement.
//
// As of #904 an UNDECLARED alias lowers to the local-slot machinery
// (`/typeName createentity allocate { <rest> } execute deallocate pop`),
// exactly like `local entity <name> = new <type> entity`. The old
// `/typeName createentity /localName xdef` lowering always failed at
// runtime for undeclared names — `xdef` refuses a name no entity in scope
// declares — and typed sets on the alias degraded to cvi because the alias
// was in no symbol table. An alias that IS a declared EDD attribute keeps
// the legacy xdef binding (see TestCreateEntityAs_DeclaredAliasKeepsXdef).
func TestCreateEntityAs(t *testing.T) {
	tests := []struct {
		name string
		el   string
		want string
	}{
		{
			name: "construct and bind",
			el:   `create state_tax_result as st_result;`,
			want: `/state_tax_result createentity allocate { } execute deallocate pop`,
		},
		{
			name: "construct then append",
			el: `create state_tax_result as st_result;` +
				`add st_result to job.state_tax_results;`,
			// The append references the alias through its local slot; the
			// bare-IDENT add path emits the single dest-owned addto.
			want: `/state_tax_result createentity allocate ` +
				`{ 0 local@ job.state_tax_results swap addto } execute deallocate pop`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh compiler per case: local slot indices are per-table
			// state (#814) and these cases each model one table.
			c := NewCompiler()
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

// TestCreateEntityAs_DeclaredAliasKeepsXdef: when the alias resolves in the
// EDD symbol table, create-as keeps the legacy attribute binding — this is
// the shipped pattern (bind to a declared scratch attribute, fill via a
// follow-up table) that must keep compiling unchanged (#904).
func TestCreateEntityAs_DeclaredAliasKeepsXdef(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"new_recipient": "entity",
	})
	got, err := c.CompileAction(`create token_recipient as new_recipient;`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := `/token_recipient createentity /new_recipient xdef`
	if normalizeSpaces(got) != normalizeSpaces(want) {
		t.Errorf("declared-alias lowering:\n  got:  %q\n  want: %q", got, want)
	}
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

package el

import (
	"strings"
	"testing"
)

// TestPerformDynamicTable exercises the `perform table named (<string-expression>)`
// syntax. The grammar lets a state-dispatch action build the table name from
// runtime values (e.g. job.state) and execute the matching XX_Tax table.
func TestPerformDynamicTable(t *testing.T) {
	tests := []struct {
		name string
		el   string
		want string
	}{
		{
			name: "literal name in expression",
			el:   `perform table named ("Calculate_CA_Tax");`,
			want: `"Calculate_CA_Tax" performtable`,
		},
		{
			name: "name from concatenation",
			el:   `perform table named ("Calculate_" + job.state + "_Tax");`,
			want: `"Calculate_" job.state strconcat "_Tax" strconcat performtable`,
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CompileAction(tt.el)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.el, err)
			}
			got = strings.TrimSpace(got)
			want := strings.TrimSpace(tt.want)
			if got != want {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.el, got, want)
			}
		})
	}
}

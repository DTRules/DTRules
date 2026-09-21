package el

import (
	"strings"
	"testing"
)

// #1249: `local <type> <name>` used to have a `...Defined` alternative per
// type, meant to catch a name that is already defined. The typed placeholder
// and undefinedIdent are both IDENT, so the parser always took `...Undef` and
// the check never ran: a redeclaration silently allocated a second slot. The
// decision now sits in the emitter, which has the symbol table and the locals
// in scope.

var localTypeKeywords = []struct{ kw, symType, undefLabel string }{
	{"entity", TypeEntity, "localEntityUndef"},
	{"long", TypeInteger, "localLongUndef"},
	{"int", TypeInteger, "localLongUndef"},
	{"double", TypeDouble, "localDoubleUndef"},
	{"boolean", TypeBoolean, "localBoolUndef"},
	{"date", TypeDate, "localDateUndef"},
	{"array", TypeArray, "localArrayUndef"},
	{"string", TypeString, "localStringUndef"},
	{"bigint", TypeBigInt, "localBigIntUndef"},
	{"fixed", TypeFixed, "localFixedUndef"},
	{"bytes", TypeBytes, "localBytesUndef"},
}

// An undefined name declares a local, through the Undef alternative, with the
// documented postfix (docs/el-reference.md, Local Variables).
func TestLocalDeclaration_UndefinedNameAllocates(t *testing.T) {
	for _, tc := range localTypeKeywords {
		c := NewCompiler()
		c.SetSymbols(map[string]string{"other": tc.symType})
		var tree IDoneContext
		c.parsed = func(d IDoneContext) { tree = d }
		got, err := c.CompileContext("local " + tc.kw + " x")
		if err != nil {
			t.Errorf("local %s x: %v", tc.kw, err)
			continue
		}
		if want := "null allocate execute deallocate pop"; got != want {
			t.Errorf("local %s x = %q, want %q", tc.kw, got, want)
		}
		if tree == nil || !treeHasLabel(tree, tc.undefLabel) {
			t.Errorf("local %s x did not parse through %s", tc.kw, tc.undefLabel)
		}
	}
}

// A name the EDD already defines cannot be declared as a local, in either
// form, whatever its type. This is the check the Java grammar made with its
// `LOCAL LONG RLONG` alternatives ("The variable 'x' is already defined").
func TestLocalDeclaration_EDDNameIsAlreadyDefined(t *testing.T) {
	for _, tc := range localTypeKeywords {
		for _, src := range []string{"local " + tc.kw + " total", "local " + tc.kw + " Total"} {
			c := NewCompiler()
			c.SetSymbols(map[string]string{"total": tc.symType, "account.total": tc.symType})
			got, err := c.CompileContext(src)
			if err == nil || !strings.Contains(err.Error(), "'total' is already defined") {
				t.Errorf("%s with total in the EDD: got %q, %v; want an already-defined error", src, got, err)
			}
		}
	}
	// A different type in the EDD is still a clash.
	c := NewCompiler()
	c.SetSymbols(map[string]string{"total": TypeDouble})
	if _, err := c.CompileContext("local long total = 5"); err == nil {
		t.Errorf("local long total = 5 with total a double in the EDD compiled")
	}
}

// A name already declared as a local in scope cannot be declared again: the
// second declaration used to allocate a new slot and silently rebind the name.
func TestLocalDeclaration_LocalInScopeIsAlreadyDefined(t *testing.T) {
	c := NewCompiler()
	if _, err := c.CompileContext("local long n"); err != nil {
		t.Fatal(err)
	}
	c.MarkLocalScope()
	c.ResetToLocalScope()
	if _, err := c.CompileAction("local double n = 1.0"); err == nil || !strings.Contains(err.Error(), "'n' is already defined") {
		t.Errorf("redeclaring context local n in an action: err = %v, want already defined", err)
	}

	// Sibling actions each start from the context's scope, so a local one
	// action declares does not block the next.
	c.ResetToLocalScope()
	if _, err := c.CompileAction("local long k = 1"); err != nil {
		t.Fatal(err)
	}
	c.ResetToLocalScope()
	if _, err := c.CompileAction("local long k = 2"); err != nil {
		t.Errorf("sibling action declaring k again: %v", err)
	}

	// Within one action, a second declaration of the same name is refused.
	c.ResetToLocalScope()
	if _, err := c.CompileAction("local long j = 1; local long j = 2"); err == nil || !strings.Contains(err.Error(), "'j' is already defined") {
		t.Errorf("declaring j twice in one action: err = %v, want already defined", err)
	}

	// A new table starts with no locals.
	c.ResetLocals()
	if _, err := c.CompileContext("local long n"); err != nil {
		t.Errorf("local long n in a new table: %v", err)
	}
}

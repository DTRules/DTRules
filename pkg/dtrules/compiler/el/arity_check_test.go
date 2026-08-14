package el_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

func compileWith(t *testing.T, src string) (string, error) {
	t.Helper()
	c := el.NewCompiler()
	c.SetOperatorChecker(func(n string) bool { _, ok := operators.GetByString(n); return ok })
	c.SetOperatorArity(func(n string) int {
		op, ok := operators.GetByString(n)
		if !ok {
			return 0
		}
		return op.Arity()
	})
	return c.CompileAction(src)
}

func TestShortCallIsRefused(t *testing.T) {
	if out, err := compileWith(t, `subsets(hand.cards)`); err == nil {
		t.Fatalf("a one-argument subsets compiled clean: %q", out)
	} else if !strings.Contains(err.Error(), "4") || !strings.Contains(err.Error(), "subsets") {
		t.Errorf("error should name the operator and the counts: %v", err)
	}
}

func TestOverlongCallIsRefused(t *testing.T) {
	if out, err := compileWith(t, `subsets(hand.cards, "combo", "value", hand.combos, 7)`); err == nil {
		t.Fatalf("a five-argument subsets compiled clean: %q", out)
	}
}

func TestCorrectCallStillCompiles(t *testing.T) {
	out, err := compileWith(t, `subsets(hand.cards, "combo", "value", hand.combos)`)
	if err != nil {
		t.Fatalf("the correct four-argument form was refused: %v", err)
	}
	if !strings.Contains(out, "subsets") {
		t.Errorf("postfix lost the operator: %q", out)
	}
}

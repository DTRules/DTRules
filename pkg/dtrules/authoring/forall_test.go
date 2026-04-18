package authoring_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

func forallTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "forall")
}

func forallSetup(t *testing.T) *session.RuleSet {
	t.Helper()
	xmlDir := filepath.Join(forallTestdataDir(t), "xml")

	rs := session.NewRuleSet("forall-test")

	eddPath := filepath.Join(xmlDir, "forall_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}

	dtPath := filepath.Join(xmlDir, "forall_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("open DT: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Fatalf("LoadDecisionTables: %v", err)
	}

	return rs
}

func forallSession(t *testing.T, rs *session.RuleSet, inputFile string) (*session.RSession, *interpreter.DTState) {
	t.Helper()
	xmlDir := filepath.Join(forallTestdataDir(t), "xml")

	sess, err := session.NewSession(rs)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("unexpected state type %T", sess.GetState())
	}
	state.SetOperatorTable(operators.GetOperatorTable())

	mapPath := filepath.Join(xmlDir, "forall_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()

	dataFile, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("open data: %v", err)
	}
	defer dataFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if err := m.LoadDataAndPushSingletons(dataFile); err != nil {
		t.Fatalf("LoadDataAndPushSingletons: %v", err)
	}

	return sess, state
}

func getCount(t *testing.T, state *interpreter.DTState) int64 {
	t.Helper()
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		ent, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		if ent.GetName().StringValue() == "calculation_context" {
			val, err := ent.Get(dtrules.GetRName("processed_count"))
			if err != nil || val == nil {
				return 0
			}
			ri, err := val.RIntegerValue()
			if err != nil {
				return 0
			}
			n, err := ri.LongValue()
			if err != nil {
				return 0
			}
			return n
		}
	}
	return 0
}

// runForm executes the named entry table against the named input fixture and
// returns calculation_context.processed_count. An assertion failure flagging
// the #626 regression signature ("not defined by any Entity on the Entity
// Stack") fails the test with a distinct message — every form should have
// pushed the element entity before running the body.
func runForm(t *testing.T, table, inputFile string) int64 {
	t.Helper()
	rs := forallSetup(t)
	inputPath := filepath.Join(forallTestdataDir(t), "inputs", inputFile)
	sess, state := forallSession(t, rs, inputPath)

	if err := sess.Execute(table); err != nil {
		if strings.Contains(err.Error(), "was not defined by any Entity on the Entity Stack") {
			t.Fatalf("REGRESSION (%s): forall context did not push element onto entity stack: %v", table, err)
		}
		t.Fatalf("Execute %s: %v", table, err)
	}
	return getCount(t, state)
}

// Each form gets its own subtests. The three input fixtures exercise:
//   two_auth_one_not: 3 accounts, 2 authorized
//   empty_accounts  : 0 accounts
//   three_auth      : 3 accounts, all authorized
// Simple / InEntity / AllowRemove forms filter inside the body (condition
// column Y/N), so processed_count == authorized count.
// Where / InEntityWhere forms filter in the context; their body is
// unconditional, so processed_count == authorized count as well (the filter
// predicate is equivalent).

func TestForallSimple(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_Simple", input); got != want {
				t.Errorf("FA_Simple on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallAllowRemove(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_AllowRemove", input); got != want {
				t.Errorf("FA_AllowRemove on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallWhere(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_Where", input); got != want {
				t.Errorf("FA_Where on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallWhereAllowRemove(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_WhereAllowRemove", input); got != want {
				t.Errorf("FA_WhereAllowRemove on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallInEntity(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_InEntity", input); got != want {
				t.Errorf("FA_InEntity on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallInEntityAllowRemove(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_InEntityAllowRemove", input); got != want {
				t.Errorf("FA_InEntityAllowRemove on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

func TestForallInEntityWhere(t *testing.T) {
	cases := map[string]int64{
		"two_auth_one_not.xml": 2,
		"empty_accounts.xml":   0,
		"three_auth.xml":       3,
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := runForm(t, "FA_InEntityWhere", input); got != want {
				t.Errorf("FA_InEntityWhere on %s: got %d, want %d", input, got, want)
			}
		})
	}
}

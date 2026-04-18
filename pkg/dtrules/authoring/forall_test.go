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

// forallSetup loads EDD + DT and returns a configured RuleSet.
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

// forallSession creates a session, wires operators, and loads test data via map.
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

// getCount returns calculation_context.processed_count from the entity stack.
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

// TestForAll_ArrayElementsOnStack verifies that with 2 authorized and 1
// unauthorized staking_account, the forall context iterates all three and
// processed_count ends up at 2.
func TestForAll_ArrayElementsOnStack(t *testing.T) {
	rs := forallSetup(t)
	inputPath := filepath.Join(forallTestdataDir(t), "inputs", "two_auth_one_not.xml")
	sess, state := forallSession(t, rs, inputPath)

	if err := sess.Execute("Process_Accounts"); err != nil {
		// Fail only if it's the original "not defined" bug — not a real execution error
		if strings.Contains(err.Error(), "was not defined by any Entity on the Entity Stack") {
			t.Fatalf("REGRESSION: forall context does not push entity onto stack: %v", err)
		}
		t.Fatalf("Execute: %v", err)
	}

	// 2 authorized × 1 increment + 1 unauthorized × 0 = 2
	count := getCount(t, state)
	if count != 2 {
		t.Errorf("expected processed_count = 2 (two authorized accounts), got %d", count)
	}
}

// TestForAll_EmptyArray verifies that with no staking_accounts, the forall
// context does not iterate and processed_count stays at 0.
func TestForAll_EmptyArray(t *testing.T) {
	rs := forallSetup(t)
	inputPath := filepath.Join(forallTestdataDir(t), "inputs", "empty_accounts.xml")
	sess, state := forallSession(t, rs, inputPath)

	if err := sess.Execute("Process_Accounts"); err != nil {
		if strings.Contains(err.Error(), "was not defined by any Entity on the Entity Stack") {
			t.Fatalf("REGRESSION: forall context does not push entity onto stack: %v", err)
		}
		t.Fatalf("Execute: %v", err)
	}

	count := getCount(t, state)
	if count != 0 {
		t.Errorf("expected processed_count = 0 (empty array), got %d", count)
	}
}

// TestForAll_AllAuthorized verifies that with 3 authorized accounts, the forall
// context iterates all three and processed_count ends up at 3.
func TestForAll_AllAuthorized(t *testing.T) {
	rs := forallSetup(t)
	inputPath := filepath.Join(forallTestdataDir(t), "inputs", "three_auth.xml")
	sess, state := forallSession(t, rs, inputPath)

	if err := sess.Execute("Process_Accounts"); err != nil {
		if strings.Contains(err.Error(), "was not defined by any Entity on the Entity Stack") {
			t.Fatalf("REGRESSION: forall context does not push entity onto stack: %v", err)
		}
		t.Fatalf("Execute: %v", err)
	}

	// 3 authorized × 1 increment each = 3
	count := getCount(t, state)
	if count != 3 {
		t.Errorf("expected processed_count = 3 (three authorized accounts), got %d", count)
	}
}

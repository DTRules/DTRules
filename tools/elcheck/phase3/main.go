// Command phase3 standardizes CorporateTax's table naming and adds the
// dispatch layer (PLAN.md Phase 3).
//
// 36 states use Determine_XX_Filing_Requirement / Calculate_XX_Income_
// Adjustments / Calculate_XX_State_Tax; 15 (one authoring batch: CO IA ID IN
// KS KY LA MA MD ME MI MN MO MS MT) spell the same three tables with
// _Corporate_. Dynamic dispatch needs one convention, so the 15 are renamed
// to the majority. Four gross-receipts states (OH TN TX WA) keep their real
// tables and gain thin standard-named wrappers that perform them, so
// `perform table named ("..." + apportionment.state_code + "...")` resolves
// for every state.
//
// Everything goes through the authoring SDK; one Save at the end.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

var corporateStates = []string{"CO", "IA", "ID", "IN", "KS", "KY", "LA", "MA",
	"MD", "ME", "MI", "MN", "MO", "MS", "MT"}

// wrappers: state -> standard table name -> the real tables it performs, in order.
var wrappers = map[string]map[string][]string{
	"OH": {
		"Calculate_OH_Income_Adjustments": {"Calculate_OH_Taxable_Gross_Receipts"},
		"Calculate_OH_State_Tax":          {"Calculate_OH_CAT"},
	},
	"TN": {
		"Calculate_TN_State_Tax": {"Calculate_TN_Excise_Tax", "Calculate_TN_Franchise_Tax"},
	},
	"TX": {
		"Calculate_TX_Income_Adjustments": {"Determine_TX_Deduction_Method", "Calculate_TX_Margin"},
		"Calculate_TX_State_Tax":          {"Calculate_TX_Franchise_Tax"},
	},
	"WA": {
		"Calculate_WA_Income_Adjustments": {"Determine_WA_Business_Classification", "Calculate_WA_Gross_Receipts"},
		"Calculate_WA_State_Tax":          {"Calculate_WA_BO_Tax", "Calculate_WA_Small_Business_Credit"},
	},
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "phase3: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	p, err := authoring.OpenProject("sampleprojects/CorporateTax")
	if err != nil {
		fatal("open: %v", err)
	}

	// 1. Renames: the same three-table trio, majority spelling.
	renamed := 0
	for _, st := range corporateStates {
		for old, new_ := range map[string]string{
			"Determine_" + st + "_Corporate_Filing_Requirement": "Determine_" + st + "_Filing_Requirement",
			"Calculate_" + st + "_Corporate_Income_Adjustments": "Calculate_" + st + "_Income_Adjustments",
			"Calculate_" + st + "_Corporate_Tax":                "Calculate_" + st + "_State_Tax",
		} {
			if p.Table(old) == nil {
				continue // already renamed (idempotent re-run)
			}
			if err := p.RenameTable(old, new_); err != nil {
				fatal("rename %s -> %s: %v", old, new_, err)
			}
			renamed++
		}
	}

	// 2. Wrapper tables for the states whose real computation has its own
	// names. Condition: the wrapper only fires for its own state, which also
	// documents what it is. Policy FIRST, single column.
	added := 0
	for st, tables := range wrappers {
		file := "states/" + st + "_corp_dt.xml"
		for name, performs := range tables {
			if p.Table(name) != nil {
				continue // idempotent re-run
			}
			t, err := p.AddTable(name, file,
				"standard-name wrapper so state dispatch resolves (#948 Phase 3)")
			if err != nil {
				fatal("add %s: %v", name, err)
			}
			t.Policy = "FIRST"
			if err := t.AddCondition(authoring.Condition{
				Number:  1,
				Comment: "wrapper: dispatch arrives here for " + st,
				DSL:     `apportionment.state_code == "` + st + `"`,
				Columns: map[int]string{1: "Y"},
			}); err != nil {
				fatal("%s condition: %v", name, err)
			}
			for i, target := range performs {
				if err := t.AddAction(authoring.Action{
					Number:  i + 1,
					Comment: "performs the state's real computation",
					DSL:     "perform " + target,
					Columns: map[int]bool{1: true},
				}); err != nil {
					fatal("%s action %s: %v", name, target, err)
				}
			}
			added++
		}
	}

	// 3. The orchestrator: dispatch on apportionment.state_code.
	if p.Table("Run_Corporate_Tax") == nil {
		if !p.HasFile("orchestrator_dt.xml") {
			if err := p.CreateFile("orchestrator_dt.xml", 100, 199,
				"entry/orchestration tables live above the per-state ranges"); err != nil {
				fatal("create orchestrator file: %v", err)
			}
		}
		t, err := p.AddTable("Run_Corporate_Tax", "orchestrator_dt.xml",
			"entry table: dispatches to the state trio by apportionment.state_code (#948 Phase 3)")
		if err != nil {
			fatal("orchestrator: %v", err)
		}
		t.Policy = "FIRST"
		if err := t.AddCondition(authoring.Condition{
			Number:  1,
			Comment: "a state must be selected",
			DSL:     "apportionment.state_code is not null",
			Columns: map[int]string{1: "Y"},
		}); err != nil {
			fatal("orchestrator condition: %v", err)
		}
		actions := []string{
			`perform table named ("Determine_" + apportionment.state_code + "_Filing_Requirement");`,
			`perform table named ("Calculate_" + apportionment.state_code + "_Income_Adjustments");`,
			`perform table named ("Calculate_" + apportionment.state_code + "_State_Tax");`,
		}
		comments := []string{
			"nexus: does this corporation file in the state at all?",
			"state additions and subtractions to federal taxable income",
			"the state's tax, credits, and refund-or-owed",
		}
		for i, dsl := range actions {
			if err := t.AddAction(authoring.Action{
				Number:  i + 1,
				Comment: comments[i],
				DSL:     dsl,
				Columns: map[int]bool{1: true},
			}); err != nil {
				fatal("orchestrator action %d: %v", i+1, err)
			}
		}
	}

	if err := p.Save(); err != nil {
		fatal("save: %v", err)
	}
	fmt.Printf("renamed %d tables, added %d wrappers + orchestrator\n", renamed, added)

	// sanity: every state resolves the full trio
	var missing []string
	for _, st := range allStates(p) {
		for _, pat := range []string{"Determine_%s_Filing_Requirement",
			"Calculate_%s_Income_Adjustments", "Calculate_%s_State_Tax"} {
			name := fmt.Sprintf(pat, st)
			if p.Table(name) == nil {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		fatal("dispatch cannot resolve: %s", strings.Join(missing, ", "))
	}
	fmt.Println("dispatch trio resolves for every state")
}

func allStates(p *authoring.Project) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range p.Tables() {
		if strings.HasPrefix(name, "Determine_") && strings.HasSuffix(name, "_Filing_Requirement") {
			st := strings.TrimSuffix(strings.TrimPrefix(name, "Determine_"), "_Filing_Requirement")
			if len(st) == 2 && !seen[st] {
				seen[st] = true
				out = append(out, st)
			}
		}
	}
	return out
}

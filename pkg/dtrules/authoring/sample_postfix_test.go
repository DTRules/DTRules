package authoring

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
)

// TestSamplePostfixIsCompiledFromDSL is the gate for SPEC invariant 2 over
// every sample project: postfix is compiled from DSL, never written (#1300).
//
// `dtrules verify` already forces this for every file a workbook produces,
// because the workbook carries no postfix and the rebuild compiles it. What it
// cannot see is a file no workbook produces. The #1300 audit found two such
// files: KidAid_Application's copy of KidAid, compiled before #1287 and never
// rebuildable, and TaxReturn's TEMPLATE_dt.xml, whose hand-written
// <action_postfix> is what the loader would have executed. So this checks the
// XML itself, not the pipeline: every stored postfix must be exactly what the
// authoring compiler makes of its DSL today, and there is no postfix without
// DSL.
//
// It walks every *_dt.xml, including files the loader skips, because a
// skipped file is still a file someone copies.
func TestSamplePostfixIsCompiledFromDSL(t *testing.T) {
	root := filepath.Join("..", "..", "..", "sampleprojects")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("sampleprojects not found at %s: %v", root, err)
	}

	symbolsFor := map[string]map[string]string{}
	var files, rows int
	var problems []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_dt.xml") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		// Symbols come from the project's whole xml/ tree, as for `table put`:
		// TaxReturn's states/XX_dt.xml types against TaxReturn_edd.xml.
		xmlRoot := filepath.Dir(path)
		for dir := xmlRoot; dir != root && dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
			if filepath.Base(dir) == "xml" {
				xmlRoot = dir
				break
			}
		}
		syms, ok := symbolsFor[xmlRoot]
		if !ok {
			syms = LoadEDDSymbols(xmlRoot)
			symbolsFor[xmlRoot] = syms
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var dt excel.DecisionTablesXML
		if err := xml.Unmarshal(data, &dt); err != nil {
			problems = append(problems, fmt.Sprintf("%s: does not parse: %v", rel, err))
			return nil
		}
		files++

		for _, tbl := range dt.Tables {
			// One compiler per table, in the order authoring compiles, so local
			// slot numbering matches (table.go).
			tc := newTableCompiler(syms)
			check := func(kind, item, dsl, stored string) {
				if strings.TrimSpace(dsl) == "" && strings.TrimSpace(stored) == "" {
					return
				}
				rows++
				where := fmt.Sprintf("%s: %s: %s", rel, tbl.TableName, item)
				if strings.TrimSpace(dsl) == "" {
					problems = append(problems, fmt.Sprintf("%s: postfix with no DSL (hand-written): %q",
						where, collapse(stored)))
					return
				}
				got := tc.compile(dsl, kind)
				if got == "" && strings.TrimSpace(stored) != "" {
					problems = append(problems, fmt.Sprintf("%s: DSL does not compile, yet postfix is stored: dsl=%q postfix=%q",
						where, collapse(dsl), collapse(stored)))
					return
				}
				if collapse(got) != collapse(stored) {
					problems = append(problems, fmt.Sprintf("%s: stored postfix is not what the DSL compiles to\n    dsl:      %q\n    stored:   %q\n    compiles: %q",
						where, collapse(dsl), collapse(stored), collapse(got)))
				}
			}
			for i, c := range tbl.Contexts.Details {
				check("context", fmt.Sprintf("context %d", i+1), c.DSL, c.Postfix)
			}
			inits := append(append([]excel.InitialActionXML{}, tbl.InitialActions...), tbl.InitialActionsLegacy...)
			for i, ia := range inits {
				check("action", fmt.Sprintf("initial action %d", i+1), ia.DSL, ia.Postfix)
				// The legacy pair: the loader prefers <action_postfix> when it
				// is set, so it must be compiled from <action_dsl> too.
				check("action", fmt.Sprintf("initial action %d (action_dsl)", i+1), ia.ActionDSL, ia.ActionPostfix)
			}
			for _, c := range tbl.Conditions {
				check("condition", "condition "+c.Number, c.DSL, c.Postfix)
			}
			for _, a := range tbl.Actions {
				check("action", "action "+a.Number, a.DSL, a.Postfix)
			}
			for _, ps := range tbl.PolicyStatements {
				rows++
				if want := excel.CompilePolicyStatement(ps.Description); collapse(want) != collapse(ps.Postfix) {
					problems = append(problems, fmt.Sprintf("%s: %s: policy statement column %s: stored %q, compiles to %q",
						rel, tbl.TableName, ps.Column, collapse(ps.Postfix), collapse(want)))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 || rows == 0 {
		t.Fatalf("checked %d files and %d rows: the walk found nothing, so this gate proves nothing", files, rows)
	}
	for _, p := range problems {
		t.Error(p)
	}
	t.Logf("%d decision-table files, %d compiled rows checked", files, rows)
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// The loader and verify skip template files; the authoring API listed them as
// tables of the project, so `table get`/`patch` would edit a file the engine
// never loads (#1300).
func TestOpenProjectSkipsTemplateFiles(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(filepath.Join(xmlDir, "states"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(xmlDir, "P_edd.xml"), declaredEDD)
	write(filepath.Join(xmlDir, "P_dt.xml"), declaredDT)
	write(filepath.Join(xmlDir, "states", "TEMPLATE_dt.xml"),
		strings.Replace(declaredDT, "<table_name>Only</table_name>", "<table_name>Calculate_XX_Tax</table_name>", 1))
	write(filepath.Join(xmlDir, "states", "TEMPLATE_edd.xml"),
		strings.Replace(declaredEDD, `name="age"`, `name="xx_placeholder"`, 1))
	if !loader.SkipRuleFile(filepath.Join(xmlDir, "states", "TEMPLATE_dt.xml")) {
		t.Fatal("precondition: the loader no longer skips TEMPLATE files; this test's premise is gone")
	}

	p, err := OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range p.Tables() {
		if strings.EqualFold(name, "Calculate_XX_Tax") {
			t.Errorf("the template's table is listed as a table of the project: %v", p.Tables())
		}
	}
	for _, f := range p.EDDFiles() {
		if strings.Contains(strings.ToUpper(filepath.Base(f)), "TEMPLATE") {
			t.Errorf("the template EDD is offered as a project EDD: %v", p.EDDFiles())
		}
	}
	if _, ok := LoadEDDSymbols(xmlDir)["job.xx_placeholder"]; ok {
		t.Error("the template EDD's placeholder field is in the project's symbol table")
	}
}

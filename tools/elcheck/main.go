// Command elcheck reports, for every row of every decision table in a project,
// whether its stored postfix is what its EL DSL compiles to today.
//
// A project is safe to edit through the authoring API only when `hand` and
// `err` are both zero — `syncToXML` regenerates every postfix from its DSL, so
// a row carrying postfix without DSL, or DSL that no longer compiles, is
// emptied by the next `table put`. See tools/elcheck/README.md.
//
// Reads a project's DT XML, compiles every row's DSL through one EL compiler
// per table (contexts, initial actions, conditions, actions — the exact order
// syncToXML uses, so context locals are in scope), and diffs the compiled
// postfix against what is stored on disk.
//
// With -overrides, candidate DSL is substituted for named rows before
// compiling. That is how a hand-coded row gets authored: propose EL, compile
// it in the table's real local scope, and apply only on an exact postfix match.
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
)

type dtFile struct {
	Tables []dtTable `xml:"decision_table"`
}

type dtTable struct {
	Name     string `xml:"table_name"`
	Contexts struct {
		Details []row `xml:"context_details"`
	} `xml:"contexts"`
	// Both element spellings; the canonical one wins. Reading only
	// initial_action_details made the probe silently blind to 312 rows the
	// moment the authoring API normalised them.
	InitialActions struct {
		Modern []row `xml:"initial_action"`
		Legacy []row `xml:"initial_action_details"`
	} `xml:"initial_actions"`
	Conditions struct {
		Details []row `xml:"condition_details"`
	} `xml:"conditions"`
	Actions struct {
		Details []row `xml:"action_details"`
	} `xml:"actions"`
}

// row covers all four kinds; only the matching element names populate.
type row struct {
	CtxNum  string `xml:"context_number"`
	CondNum string `xml:"condition_number"`
	ActNum  string `xml:"action_number"`
	CtxDSL  string `xml:"context_dsl"`
	CondDSL string `xml:"condition_dsl"`
	ActDSL  string `xml:"action_dsl"`
	InitDSL string `xml:"initial_action_dsl"`
	CtxPF   string `xml:"context_postfix"`
	CondPF  string `xml:"condition_postfix"`
	ActPF   string `xml:"action_postfix"`
	InitPF  string `xml:"initial_action_postfix"`
	CtxCmt  string `xml:"context_comment"`
	CondCmt string `xml:"condition_comment"`
	ActCmt  string `xml:"action_comment"`
	InitCmt string `xml:"initial_action_comment"`
}

func (r row) dsl(kind string) string {
	switch kind {
	case "context":
		return r.CtxDSL
	case "initial":
		return r.InitDSL
	case "condition":
		return r.CondDSL
	default:
		return r.ActDSL
	}
}

func (r row) pf(kind string) string {
	switch kind {
	case "context":
		return r.CtxPF
	case "initial":
		return r.InitPF
	case "condition":
		return r.CondPF
	default:
		return r.ActPF
	}
}

func (r row) comment(kind string) string {
	switch kind {
	case "context":
		return r.CtxCmt
	case "initial":
		return r.InitCmt
	case "condition":
		return r.CondCmt
	default:
		return r.ActCmt
	}
}

// key names a row the way HandCodedRows does: contexts and initial actions by
// 1-based position, conditions and actions by their declared number.
func (r row) key(kind string, i int) string {
	switch kind {
	case "context":
		return fmt.Sprintf("context %d", i+1)
	case "initial":
		return fmt.Sprintf("initial action %d", i+1)
	case "condition":
		// Position, not the declared number: `table get` renumbers rows to
		// their position on load, so a number key points at a different row
		// on the far side of the authoring API.
		return fmt.Sprintf("condition@%d", i+1)
	default:
		return fmt.Sprintf("action@%d", i+1)
	}
}

func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

// initialRows returns the initial-action rows from whichever spelling the
// file uses, canonical first.
func initialRows(t dtTable) []row {
	if len(t.InitialActions.Modern) > 0 {
		return t.InitialActions.Modern
	}
	return t.InitialActions.Legacy
}

func main() {
	project := flag.String("project", ".", "project root")
	only := flag.String("table", "", "limit to one table")
	overrides := flag.String("overrides", "", "JSON file: {\"Table\":{\"condition 3\":\"<el>\"}}")
	show := flag.String("show", "problem", "which rows to print: problem|all|hand")
	exclude := flag.String("exclude", "", "skip files whose path contains this substring (e.g. a generated merge output)")
	flag.Parse()

	symbols := authoring.LoadEDDSymbols(*project)

	var ov map[string]map[string]string
	if *overrides != "" {
		b, err := os.ReadFile(*overrides)
		if err != nil {
			fatal(err)
		}
		if err := json.Unmarshal(b, &ov); err != nil {
			fatal(err)
		}
	}

	// Recurse: CorporateTax keeps its real content in xml/states/, not xml/.
	var files []string
	err := filepath.WalkDir(*project+"/xml", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), "_dt.xml") || strings.Contains(d.Name(), "TEMPLATE") {
			return nil
		}
		// A project that merges per-state files into one artifact (CorporateTax)
		// would otherwise have every row counted twice.
		if *exclude != "" && strings.Contains(p, *exclude) {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		fatal(err)
	}
	sort.Strings(files)

	var nHand, nDiff, nErr, nMatch, nResolved, nProse int
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			fatal(err)
		}
		var doc dtFile
		if err := xml.Unmarshal(b, &doc); err != nil {
			fmt.Printf("== %s\n  SKIPPED (does not parse): %v\n", f, err)
			continue
		}
		for _, t := range doc.Tables {
			if *only != "" && !strings.EqualFold(*only, t.Name) {
				continue
			}
			// One compiler per table; locals persist across rows (#965).
			c := el.NewCompiler()
			c.SetSymbols(symbols)
			c.ResetLocals()

			var out []string
			kinds := []struct {
				kind string
				rows []row
			}{
				{"context", t.Contexts.Details},
				{"initial", initialRows(t)},
				{"condition", t.Conditions.Details},
				{"action", t.Actions.Details},
			}
			for _, kd := range kinds {
				for i, r := range kd.rows {
					k := r.key(kd.kind, i)
					stored := r.pf(kd.kind)
					dsl := r.dsl(kd.kind)
					proposed, isOverride := ov[t.Name][k]
					if isOverride {
						dsl = proposed
					}

					status, got, cerr := compileRow(c, kd.kind, dsl)
					line := ""
					switch {
					case strings.TrimSpace(dsl) == "" && strings.TrimSpace(stored) == "":
						continue // empty row, nothing to author
					case strings.TrimSpace(dsl) == "":
						nHand++
						line = fmt.Sprintf("  HAND    %-16s stored: %s%s", k, norm(stored), cmt(r.comment(kd.kind)))
					case isProse(dsl) && strings.TrimSpace(stored) == "":
						// Commented-out documentation row: compiles to
						// nothing, stores nothing. Nothing to author.
						nProse++
						if *show == "all" {
							line = fmt.Sprintf("  prose   %-16s %s", k, oneline(dsl))
						}
					case cerr != nil:
						nErr++
						line = fmt.Sprintf("  ERR     %-16s %v\n            dsl: %s", k, cerr, oneline(dsl))
					case norm(got) == norm(stored):
						if isOverride {
							nResolved++
							line = fmt.Sprintf("  RESOLVED %-15s %s\n            -> %s", k, oneline(dsl), norm(got))
						} else {
							nMatch++
							if *show == "all" {
								line = fmt.Sprintf("  ok      %-16s %s", k, oneline(dsl))
							}
						}
					default:
						nDiff++
						line = fmt.Sprintf("  DIFF    %-16s dsl:    %s\n            stored: %s\n            got:    %s",
							k, oneline(dsl), norm(stored), norm(got))
					}
					_ = status
					if line != "" {
						out = append(out, line)
					}
				}
			}
			if len(out) > 0 {
				fmt.Printf("== %s\n%s\n", t.Name, strings.Join(out, "\n"))
			}
		}
	}
	fmt.Printf("\nTOTAL ok=%d prose=%d resolved=%d hand=%d diff=%d err=%d\n", nMatch, nProse, nResolved, nHand, nDiff, nErr)
}

func compileRow(c *el.Compiler, kind, dsl string) (string, string, error) {
	dsl = strings.TrimSpace(dsl)
	if dsl == "" {
		return "empty", "", nil
	}
	var (
		pf  string
		err error
	)
	switch kind {
	case "context":
		pf, err = c.CompileContext(dsl)
	case "condition":
		pf, err = c.CompileCondition(dsl)
	default: // initial action + action
		pf, err = c.CompileAction(dsl)
	}
	return "compiled", pf, err
}

func oneline(s string) string { return norm(s) }

// isProse reports a commented-out documentation row.
func isProse(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#")
}

func cmt(s string) string {
	s = norm(s)
	if s == "" {
		return ""
	}
	return "\n            // " + s
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "elc:", err)
	os.Exit(1)
}

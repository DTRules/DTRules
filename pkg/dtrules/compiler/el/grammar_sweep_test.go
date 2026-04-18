// Grammar sweep: drive every labeled alternative in EL.g4 through the
// compiler. Two assertions per entry: compile succeeds, postfix is non-empty.
// The empty-postfix check is what makes the silent fall-through that caused
// #626 impossible — a labeled alternative without a PostfixEmitter override
// emits nothing, and this sweep fails.
//
// The corpus is generated from EL.g4 itself (see tools/gen_corpus.py) so
// adding a new labeled alternative forces a corpus entry: the coverage guard
// below fails if any grammar label has no corpus row.

package el

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

type corpusRow struct {
	rule  string
	label string
	entry string // condition | action | context | raw
	dsl   string
}

func loadTSV(t *testing.T, path string) []corpusRow {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	var rows []corpusRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue // skip header
		}
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		// Accept >=4 columns (overrides file has an optional note column 5).
		if len(parts) < 4 {
			t.Fatalf("bad tsv line in %s: %q", path, line)
		}
		rows = append(rows, corpusRow{rule: parts[0], label: parts[1], entry: parts[2], dsl: parts[3]})
	}
	return rows
}

// loadCorpus merges generated corpus with manual overrides. Overrides replace
// generated rows by (rule, label) key.
func loadCorpus(t *testing.T) []corpusRow {
	t.Helper()
	base := loadTSV(t, "testdata/grammar_corpus.tsv")
	overrides := loadTSV(t, "testdata/grammar_overrides.tsv")
	keyed := make(map[string]corpusRow, len(base))
	for _, r := range base {
		keyed[r.rule+"/"+r.label] = r
	}
	for _, r := range overrides {
		keyed[r.rule+"/"+r.label] = r
	}
	// Stable order by rule then label for reproducible test output.
	keys := make([]string, 0, len(keyed))
	for k := range keyed {
		keys = append(keys, k)
	}
	// Simple sort via repeated pass — avoids import bloat.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	out := make([]corpusRow, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyed[k])
	}
	return out
}

func compileRow(c *Compiler, r corpusRow) (string, error) {
	switch r.entry {
	case "condition":
		return c.CompileCondition(r.dsl)
	case "action":
		return c.CompileAction(r.dsl)
	case "context":
		return c.CompileContext(r.dsl)
	case "raw":
		return c.Compile(r.dsl)
	default:
		return "", nil
	}
}

// Labels that legitimately emit no postfix at compile time (empty statements,
// no-op markers). These are the ONLY labels allowed to produce empty output;
// every other label's emit must be non-empty, or the fall-through bug that
// caused #626 has a new victim.
var emptyPostfixAllowed = map[string]bool{
	"emptyAction":          true,
	"emptyCondition":       true,
	"emptyContext":         true,
	"emptyPolicyStatement": true,
}

// loadKnownFails reads the inventory of labels that are known to fail the
// sweep today. Each entry is either a DSL sample the generator can't
// construct correctly yet, or a real emitter fall-through bug tracked for
// follow-up. New entries must NOT be added casually — the correct response
// to a new failure is to either fix the emitter or refine the DSL sample.
func loadKnownFails(t *testing.T) map[string]string {
	return loadTSV3(t, "testdata/grammar_known_fails.tsv")
}

// loadHelpers reads the grammar helper-rule exemption list. Helper labels
// are grammar fragments (e.g. blist, arrayList, includeSearch) that can't
// be compiled directly from a top-level DSL entry point — they live inside
// other rules. The sweep skips direct compile for helpers, and the coverage
// guard treats them as covered so adding them to the grammar doesn't force
// a bogus corpus entry.
func loadHelpers(t *testing.T) map[string]string {
	return loadTSV3(t, "testdata/grammar_helpers.tsv")
}

// loadTSV3 is a shared reader for the 3-column TSV files (rule, label, notes)
// plus an optional 4th column that's ignored here.
func loadTSV3(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			t.Fatalf("bad line in %s: %q", path, line)
		}
		// Concatenate anything past column 2 as the reason.
		reason := parts[2]
		if len(parts) == 4 {
			reason = parts[2] + " — " + parts[3]
		}
		out[parts[0]+"/"+parts[1]] = reason
	}
	return out
}

func TestGrammarSweep_CompilesAndEmitsPostfix(t *testing.T) {
	rows := loadCorpus(t)
	if len(rows) == 0 {
		t.Fatal("corpus is empty")
	}
	known := loadKnownFails(t)
	helpers := loadHelpers(t)

	c := NewCompiler()
	var fails []string
	var unexpectedPasses []string
	for _, r := range rows {
		name := r.rule + "/" + r.label
		// Helpers can't be top-level compiled; their coverage is via the
		// parent-rule labels that wrap them. Skip direct compile assertion.
		if _, isHelper := helpers[name]; isHelper {
			continue
		}
		postfix, err := compileRow(c, r)
		passed := err == nil && (strings.TrimSpace(postfix) != "" || emptyPostfixAllowed[r.label])
		if _, isKnown := known[name]; isKnown {
			if passed {
				unexpectedPasses = append(unexpectedPasses, name)
			}
			continue
		}
		if err != nil {
			fails = append(fails, name+" COMPILE_ERR: "+err.Error()+"  dsl="+r.dsl)
			continue
		}
		if strings.TrimSpace(postfix) == "" && !emptyPostfixAllowed[r.label] {
			fails = append(fails, name+" EMPTY_POSTFIX (silent fall-through?)  dsl="+r.dsl)
			continue
		}
	}
	if len(unexpectedPasses) > 0 {
		t.Logf("%d labels passed that are listed in known_fails — remove them from testdata/grammar_known_fails.tsv:", len(unexpectedPasses))
		for _, p := range unexpectedPasses {
			t.Logf("  %s", p)
		}
		t.Fatalf("known_fails must shrink, not silently pass")
	}
	if len(fails) > 0 {
		t.Logf("%d grammar-sweep failures (of %d labels, with %d known-fails excluded):", len(fails), len(rows), len(known))
		for _, f := range fails {
			t.Logf("  %s", f)
		}
		t.Fatalf("grammar sweep failed")
	}
}

// TestGrammarSweep_CoverageGuard reads EL.g4 and asserts every labeled
// alternative has at least one corpus row. Adding a new `# label` to the
// grammar without a corpus entry fails this test.
func TestGrammarSweep_CoverageGuard(t *testing.T) {
	grammarLabels := extractGrammarLabels(t)
	rows := loadCorpus(t)
	covered := map[string]bool{}
	for _, r := range rows {
		covered[r.rule+"/"+r.label] = true
	}
	var missing []string
	for _, gl := range grammarLabels {
		if !covered[gl] {
			missing = append(missing, gl)
		}
	}
	if len(missing) > 0 {
		t.Logf("%d grammar labels missing from corpus:", len(missing))
		for i, m := range missing {
			if i >= 40 {
				t.Logf("  ... and %d more", len(missing)-i)
				break
			}
			t.Logf("  %s", m)
		}
		t.Fatalf("grammar coverage gap: %d labels lack corpus entries", len(missing))
	}
}

var labelRE = regexp.MustCompile(`#\s*([A-Za-z_][A-Za-z0-9_]*)\s*$`)
var ruleStartRE = regexp.MustCompile(`^([a-z][A-Za-z0-9_]*)\s*(?:\[[^\]]*\])?\s*:`)

func extractGrammarLabels(t *testing.T) []string {
	t.Helper()
	f, err := os.Open("EL.g4")
	if err != nil {
		t.Fatalf("open EL.g4: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var labels []string
	var rule string
	inBlockComment := false
	for sc.Scan() {
		line := sc.Text()
		// Strip block comments /* */ crudely.
		if inBlockComment {
			if idx := strings.Index(line, "*/"); idx >= 0 {
				line = line[idx+2:]
				inBlockComment = false
			} else {
				continue
			}
		}
		if idx := strings.Index(line, "/*"); idx >= 0 {
			if end := strings.Index(line[idx:], "*/"); end >= 0 {
				line = line[:idx] + line[idx+end+2:]
			} else {
				line = line[:idx]
				inBlockComment = true
			}
		}
		// Strip line comments.
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if m := ruleStartRE.FindStringSubmatch(trimmed); m != nil {
			rule = m[1]
		}
		if strings.HasPrefix(trimmed, ";") || trimmed == ";" {
			rule = ""
		}
		if m := labelRE.FindStringSubmatch(line); m != nil && rule != "" {
			labels = append(labels, rule+"/"+m[1])
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan EL.g4: %v", err)
	}
	return labels
}

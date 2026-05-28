// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// runCompile is the `dtrules compile <dir>` handler. It walks *_dt.xml under
// dir and, for every DSL element with non-comment content, runs the EL
// compiler and writes the postfix back into the file in place. No Excel
// round-trip — bytes outside the targeted <X_postfix> elements are untouched.
//
// This is the canonical way to backfill postfix without paying the
// XML→Excel→XML canonicalization cost of `dtrules build --from-xml`, which
// rewrites whitespace, attribute order, and any postfix that doesn't survive
// the lossy Excel layer. Use compile when:
//
//   - You have authored EL DSL and want to ship pre-compiled XML.
//   - You're migrating a project off the loader's recompile fallback.
//   - You're verifying every DSL element parses without touching Excel.
//
// Exit code 0 only if every non-comment DSL element compiles. Any failure
// prints `<file>: <kind> N ('<dsl>'): <parse error>` and exits 1. By default
// successful fills are written to disk alongside the failures so the file
// improves incrementally; pass --strict for atomic-or-nothing behavior
// (used by CI gates where partial state would be misleading).
func (c *CLI) runCompile(args []string) int {
	target := "."
	dryRun := false
	verbose := false
	strict := false
	noAnalyze := false
	force := false
	forceOverwriteExcel := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--strict":
			strict = true
		case "--no-analyze":
			noAnalyze = true
		case "--force":
			force = true
		case "--force-overwrite-excel":
			forceOverwriteExcel = true
		case "-v", "--verbose":
			verbose = true
		case "-h", "--help":
			c.printCompileUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				c.printCompileUsage()
				return 2
			}
			target = args[i]
		}
	}

	// Walk *_dt.xml under target. If target is itself a file, run on that file.
	var files []string
	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		return 1
	}
	if info.IsDir() {
		_ = filepath.WalkDir(target, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(name, "_dt.xml") {
				return nil
			}
			// TEMPLATE_*.xml files are author-facing scaffolds with
			// placeholder content ([STATE], [Filing Status], etc.) that
			// is intentionally not valid EL — they get copied into a
			// new state file and edited. Skip them so the compile pass
			// doesn't flag template placeholders as parse errors.
			if strings.HasPrefix(strings.ToUpper(name), "TEMPLATE_") {
				return nil
			}
			files = append(files, p)
			return nil
		})
	} else if strings.HasSuffix(info.Name(), "_dt.xml") {
		files = []string{target}
	} else {
		fmt.Fprintf(os.Stderr, "compile: %s is not a *_dt.xml file or directory\n", target)
		return 1
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "compile: no *_dt.xml files under %s\n", target)
		return 1
	}

	// Excel-sync guard (v1.14.5): refuse the write if a covered Excel
	// file has been touched since the last export. `dtrules compile`
	// writes XML in place; without this guard an AI's compile could
	// silently override a human's open Excel edits. --dry-run skips
	// the guard since nothing is actually written.
	if !dryRun {
		// Decide the directory the guard searches in. For dir targets
		// it's the target itself; for single-file targets it's the
		// file's parent directory (where any sibling `.sync-manifest.json`
		// would live).
		guardDir := target
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			guardDir = filepath.Dir(target)
		}
		if err := authoring.GuardExcelInDir(guardDir, forceOverwriteExcel); err != nil {
			fmt.Fprintf(os.Stderr, "compile: %v\n", err)
			fmt.Fprintln(os.Stderr, "\nTo proceed, either:")
			fmt.Fprintln(os.Stderr, "  1) Run `dtrules build --from-excel` to import the Excel changes first, OR")
			fmt.Fprintln(os.Stderr, "  2) Re-run with --force-overwrite-excel to overwrite the human Excel edits.")
			return 1
		}
	}

	cmp := el.NewCompiler()

	// Wire up the EDD-derived symbol table so the EL compiler picks the
	// right arithmetic dispatch (fp- / fpmin for fixed operands, f- / fmin
	// for double, - / min for integer). Without this, every operand
	// defaults to integer and the emitted postfix uses int ops + `cvi`
	// conversions that crash at runtime when the actual value is RFixed or
	// RDouble (#790). The build pipeline's workbook importer already does
	// this; v1.12.0's in-loader compiler did this; the standalone compile
	// path needs to do it too.
	//
	// EDDs are discovered next to (and recursively under) the target. If
	// none are found, the compiler proceeds with no symbol table — fine
	// for projects with no typed operands, but a heads-up is printed so
	// the user knows type-aware dispatch was skipped.
	symbols := loadEDDSymbols(target)
	if len(symbols) > 0 {
		cmp.SetSymbols(symbols)
		if verbose {
			fmt.Printf("loaded %d symbols from EDD\n", len(symbols))
		}
	} else if verbose {
		fmt.Println("no EDD found near target — compile will assume integer-typed operands")
	}

	totalCompiled := 0
	totalSkipped := 0
	totalErrors := 0
	var errLines []string

	for _, f := range files {
		filled, skipped, errs := compileFile(cmp, f, dryRun, strict, force)
		totalCompiled += filled
		totalSkipped += skipped
		totalErrors += len(errs)
		for _, e := range errs {
			errLines = append(errLines, fmt.Sprintf("%s: %s", f, e))
		}
		if verbose && (filled > 0 || len(errs) > 0) {
			fmt.Printf("  %s: compiled=%d skipped=%d errors=%d\n", f, filled, skipped, len(errs))
		}
	}

	fmt.Printf("compile: %d file(s), compiled %d element(s), skipped %d (already had postfix or comment-only), %d error(s)\n",
		len(files), totalCompiled, totalSkipped, totalErrors)
	if dryRun {
		fmt.Println("(dry run — no files modified)")
	}

	// Excel refresh (v1.14.5): now that XML postfix has been written,
	// re-export Excel from the new state so the two formats stay
	// paired on disk. No-op on projects without a `.sync-manifest.json`
	// (legacy / flat layouts). Skipped on --dry-run.
	if !dryRun && totalCompiled > 0 {
		refreshDir := target
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			refreshDir = filepath.Dir(target)
		}
		if err := authoring.RefreshExcelInDir(refreshDir); err != nil {
			fmt.Fprintf(os.Stderr, "compile: Excel refresh failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "  XML was written; Excel may be stale. Run `dtrules build` to recover.")
			// Don't fail the command — XML write succeeded. Exit code
			// reflects compile success/failure only.
		}
	}

	// Advisory pass: run decisiontable.Analyze on every table in every file.
	// Same call the build pipeline and `dtrules table warnings` use, so the
	// warning set is identical across surfaces. This is what makes
	// `dtrules compile <dir>` a one-stop check on layouts the rest of the
	// authoring CLI can't reach (no xml/ subdir required) — e.g. library
	// consumers whose rules live at `pkg/.../rules/` flat.
	totalWarnings := 0
	if !noAnalyze {
		for _, f := range files {
			ws := analyzeFile(f)
			if verbose && len(ws) > 0 {
				fmt.Printf("  %s: warnings=%d\n", f, len(ws))
			}
			for _, w := range ws {
				fmt.Fprintln(os.Stderr, w.String())
			}
			totalWarnings += len(ws)
		}
		fmt.Printf("advisory: %d warning(s)\n", totalWarnings)
	}

	if totalErrors > 0 {
		fmt.Fprintln(os.Stderr, "\nCompile errors:")
		for _, e := range errLines {
			fmt.Fprintln(os.Stderr, "  "+e)
		}
		return 1
	}
	return 0
}

// loadEDDSymbols discovers every *_edd.xml file under root (or alongside
// root if root is a file) and parses out a flat map of field → type that
// the EL compiler's SetSymbols expects. Both `<field>` and
// `<entity>.<field>` keys are emitted so bare-name references and
// qualified references resolve identically — matching what
// session.RuleSet.buildSymbolTable used to produce before v1.14.0 made
// that method unused.
//
// Schema is the minimal subset the EL compiler cares about: entity name
// and the field's `type` attribute. Anything else in the EDD is ignored.
// Parse errors on individual files are silently skipped — a malformed EDD
// is the project's problem, not the compile pass's, and the compile will
// surface it through a downstream parse failure if the DT depends on it.
func loadEDDSymbols(root string) map[string]string {
	type eddField struct {
		Name string `xml:"name,attr"`
		Type string `xml:"type,attr"`
	}
	type eddEntity struct {
		Name   string     `xml:"name,attr"`
		Fields []eddField `xml:"field"`
	}
	type eddFile struct {
		Entities []eddEntity `xml:"entity"`
	}

	// Decide which directory tree to walk. If root is a *_dt.xml file we
	// look in its containing directory; otherwise we walk root itself.
	walkDir := root
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		walkDir = filepath.Dir(root)
	}

	symbols := make(map[string]string)
	_ = filepath.WalkDir(walkDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_edd.xml") {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		var f eddFile
		if xml.Unmarshal(data, &f) != nil {
			return nil
		}
		for _, ent := range f.Entities {
			for _, fld := range ent.Fields {
				if fld.Name == "" || fld.Type == "" {
					continue
				}
				symbols[fld.Name] = fld.Type
				if ent.Name != "" {
					symbols[ent.Name+"."+fld.Name] = fld.Type
				}
			}
		}
		return nil
	})
	return symbols
}

// analyzeFile parses a *_dt.xml file and runs the advisory pass on every
// table within it. Returns the aggregated warning slice. Errors during
// parse short-circuit to an empty slice — the compile step already
// reported any structural problems, so we don't double-print here.
func analyzeFile(path string) []decisiontable.Warning {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	dt, err := excel.UnmarshalDecisionTablesXML(data)
	if err != nil {
		return nil
	}
	var warnings []decisiontable.Warning
	for i := range dt.Tables {
		// buildAnalysisInputs (defined in build.go, same package) is the
		// canonical XML→Inputs conversion. Reusing it keeps the advisory
		// surface for `dtrules build` and `dtrules compile` bit-for-bit
		// identical — anything one reports, the other reports.
		warnings = append(warnings, decisiontable.Analyze(buildAnalysisInputs(&dt.Tables[i]))...)
	}
	return warnings
}

func (c *CLI) printCompileUsage() {
	fmt.Println(`Usage: dtrules compile [dir-or-file] [options]

Compile EL DSL to postfix in place across every decision table under the
given directory. Surgical: only fills empty <*_postfix> elements when the
matching <*_dsl> is non-empty and non-comment. No Excel round-trip; bytes
outside the targeted postfix elements are untouched.

Options:
  --dry-run     Report what would change without writing files.
  --strict      Refuse to write any file that has at least one compile
                error (atomic-or-nothing per file). Default writes the
                successful fills and reports the errors.
  --force       Overwrite existing postfix with a fresh compile. Use
                after a compiler bug fix (e.g. v1.14.2 #790) to refresh
                postfix produced by an earlier buggy version. Default
                only fills empty postfix.
  --force-overwrite-excel
                Bypass the Excel-mtime guard. By default 'compile'
                refuses to run when a covered Excel file is newer than
                its last export (would clobber human edits). Pass this
                flag only after deciding the XML version should win.
  --no-analyze  Skip the advisory pass after compile. Default runs it.
  -v, --verbose Per-file compiled/skipped/error/warning counts.

By default, after filling postfix this command also runs the advisory
pass (decisiontable.Analyze) on every table and prints warnings to
stderr — same warnings the build pipeline and 'dtrules table warnings'
emit. This makes 'dtrules compile <dir>' a one-stop authoring check
on layouts the rest of the CLI can't reach (no xml/ subdir required).

Exit codes:
  0  every DSL element compiled (or was already populated/comment-only).
  1  at least one DSL element failed to compile, or an I/O error occurred.

This command is the canonical backfill path when XML authoring outpaces
the build pipeline, or when migrating a project off the loader's
recompile fallback. For the full build pipeline (sync with Excel, run
advisory pass), use 'dtrules build'.`)
}

// kindPair holds one regex per element kind that locates a
// <X_dsl>DSL</X_dsl> immediately followed by an *empty* <X_postfix>...
// </X_postfix> pair (where "empty" means whitespace-only between the
// open and close tags). Each kind gets its own regex so the closing tag
// is forced to match the opening kind (Go's regexp has no
// backreferences; alternation + non-greedy `.*?` was prone to spanning
// across non-empty postfix elements).
//
// `[^<]*` constrains the DSL capture to the text node — `<` is always
// entity-encoded as `&lt;` inside DSL bodies, so this is safe and
// avoids any cross-element backtracking. Newlines pass through fine
// because the character class only excludes `<`.
type kindPair struct {
	kind    string
	reEmpty *regexp.Regexp // matches only empty <X_postfix> (default mode)
	reAny   *regexp.Regexp // matches any <X_postfix> body (--force mode, for refreshing stale postfix)
}

// Each regex is anchored to a single element kind. `[^<]` inside both DSL
// and postfix captures keeps us inside the text node — `<` is always
// entity-encoded inside body content, so this avoids cross-element
// backtracking.
var dslPostfixPairs = []kindPair{
	{
		"context",
		regexp.MustCompile(`<context_dsl>([^<]*)</context_dsl>\s*<context_postfix>\s*</context_postfix>`),
		regexp.MustCompile(`<context_dsl>([^<]*)</context_dsl>\s*<context_postfix>[^<]*</context_postfix>`),
	},
	{
		"initial_action",
		regexp.MustCompile(`<initial_action_dsl>([^<]*)</initial_action_dsl>\s*<initial_action_postfix>\s*</initial_action_postfix>`),
		regexp.MustCompile(`<initial_action_dsl>([^<]*)</initial_action_dsl>\s*<initial_action_postfix>[^<]*</initial_action_postfix>`),
	},
	{
		"action",
		regexp.MustCompile(`<action_dsl>([^<]*)</action_dsl>\s*<action_postfix>\s*</action_postfix>`),
		regexp.MustCompile(`<action_dsl>([^<]*)</action_dsl>\s*<action_postfix>[^<]*</action_postfix>`),
	},
	{
		"condition",
		regexp.MustCompile(`<condition_dsl>([^<]*)</condition_dsl>\s*<condition_postfix>\s*</condition_postfix>`),
		regexp.MustCompile(`<condition_dsl>([^<]*)</condition_dsl>\s*<condition_postfix>[^<]*</condition_postfix>`),
	},
}

// compileFile reads f, compiles each DSL→postfix pair, and writes f
// back. Returns (filled, skipped, errors).
//
// In the default mode any successful fills are written even when other
// elements in the same file failed to compile — the file is improved
// incrementally and the errors surface as a deliberate to-do list.
//
// With strict=true the file is written only if every non-comment DSL
// element compiles. Use --strict when you need an atomic guarantee
// that every element in the file is build-clean (CI gate behavior).
//
// With force=true any existing postfix is overwritten with a fresh
// compile result. Use --force after a compiler fix (e.g. v1.14.2's
// #790 SetSymbols repair) to refresh stored postfix that was produced
// by an earlier buggy version. Default behavior only fills empty
// postfix; existing content is preserved as the authoritative form.
func compileFile(cmp *el.Compiler, f string, dryRun, strict, force bool) (int, int, []string) {
	data, err := os.ReadFile(f)
	if err != nil {
		return 0, 0, []string{fmt.Sprintf("read failed: %v", err)}
	}

	cmp.ResetLocals()

	var compileErrs []string
	filled := 0
	skipped := 0

	// Run one pass per kind. Each pass walks the (already-rewritten) byte
	// buffer end-to-end. Order matters for diagnostic numbering only: we
	// process kinds in the same order they appear in dslPostfixPairs so
	// `condition N` in an error message lines up with the loader's view.
	newData := data
	for _, kp := range dslPostfixPairs {
		kind := kp.kind
		// Pick the regex that matches the operating mode: --force rewrites
		// any existing postfix; default only fills empty slots.
		re := kp.reEmpty
		if force {
			re = kp.reAny
		}
		n := 0
		newData = re.ReplaceAllFunc(newData, func(match []byte) []byte {
			n++
			m := re.FindSubmatch(match)
			if len(m) < 2 {
				return match
			}
			dsl := decodeXMLText(string(m[1]))
			dslTrimmed := strings.TrimSpace(dsl)

			if dslTrimmed == "" || isCommentOnly(dslTrimmed) {
				skipped++
				return match
			}

			compiled, err := compileKind(cmp, kind, dsl)
			if err != nil {
				compileErrs = append(compileErrs, fmt.Sprintf("%s %d ('%s'): %v",
					kind, n, snippet(dslTrimmed), err))
				return match
			}

			filled++
			// Reconstruct the pair with the postfix slot populated.
			// Preserve the original DSL bytes verbatim by indexing the
			// match: everything up to the postfix open tag stays as-is.
			postfixOpen := []byte(fmt.Sprintf("<%s_postfix>", kind))
			openIdx := indexOfLast(match, postfixOpen)
			if openIdx < 0 {
				return match
			}
			out := make([]byte, 0, len(match)+len(compiled))
			out = append(out, match[:openIdx]...)
			out = append(out, postfixOpen...)
			out = append(out, []byte(encodeXMLText(compiled))...)
			out = append(out, []byte(fmt.Sprintf("</%s_postfix>", kind))...)
			return out
		})
	}

	// Strict mode: any error blocks the write so the file stays in a
	// fully-known state. Non-strict (default): write the successful
	// fills and report the errors — the file is no worse than before
	// for the failing elements, and the successful ones land.
	if strict && len(compileErrs) > 0 {
		return filled, skipped, compileErrs
	}
	if filled > 0 && !dryRun {
		if err := os.WriteFile(f, newData, 0o644); err != nil {
			return filled, skipped, append(compileErrs, fmt.Sprintf("write failed: %v", err))
		}
	}
	return filled, skipped, compileErrs
}

// compileKind dispatches to the right EL compiler entry point for each
// table-element kind. The loader uses the same trio (CompileContext /
// CompileAction / CompileCondition) so postfix produced here is
// bit-for-bit what `dtrules build` would emit on the import step.
//
// initial_action and action both go to CompileAction — the EL grammar
// doesn't distinguish them; the loader treats both as action statements.
func compileKind(cmp *el.Compiler, kind, dsl string) (string, error) {
	switch kind {
	case "context":
		return cmp.CompileContext(dsl)
	case "condition":
		return cmp.CompileCondition(dsl)
	case "action", "initial_action":
		return cmp.CompileAction(dsl)
	default:
		return "", fmt.Errorf("unknown element kind %q", kind)
	}
}

// isCommentOnly mirrors loader/dt.go's commentary handling: a DSL whose
// trimmed body is purely a `//`, `#`, or `/* ... */` comment is treated
// as a no-op and gets an empty postfix.
func isCommentOnly(dsl string) bool {
	for _, line := range strings.Split(dsl, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "/*") && strings.HasSuffix(t, "*/") {
			continue
		}
		return false
	}
	return true
}

// indexOfLast returns the byte offset of the last occurrence of needle
// in haystack, or -1 if absent. Used to find the postfix open tag
// inside a matched pair so we can splice the compiled body in at the
// correct position without re-running the regex on the slice.
func indexOfLast(haystack, needle []byte) int {
	last := -1
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			last = i
		}
	}
	return last
}

// snippet trims long DSL strings for the error message. The full body is
// always available in the XML for follow-up; the message just needs to
// help the reader locate the offending element.
func snippet(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 80 {
		return s[:77] + "..."
	}
	return s
}

// decodeXMLText reverses the limited XML entity encoding the loader
// expects in DSL bodies: &amp; &lt; &gt; &quot; &apos;. Anything else
// passes through unchanged.
func decodeXMLText(s string) string {
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// encodeXMLText escapes the same five entities so the postfix body we
// write back into the XML survives a subsequent XML parse.
func encodeXMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

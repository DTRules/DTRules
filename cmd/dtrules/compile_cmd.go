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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
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
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--strict":
			strict = true
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

	cmp := el.NewCompiler()
	totalCompiled := 0
	totalSkipped := 0
	totalErrors := 0
	var errLines []string

	for _, f := range files {
		filled, skipped, errs := compileFile(cmp, f, dryRun, strict)
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

	if totalErrors > 0 {
		fmt.Fprintln(os.Stderr, "\nCompile errors:")
		for _, e := range errLines {
			fmt.Fprintln(os.Stderr, "  "+e)
		}
		return 1
	}
	return 0
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
  -v, --verbose Per-file compiled/skipped/error counts.

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
	kind string
	re   *regexp.Regexp
}

var dslPostfixPairs = []kindPair{
	{"context", regexp.MustCompile(`<context_dsl>([^<]*)</context_dsl>\s*<context_postfix>\s*</context_postfix>`)},
	{"initial_action", regexp.MustCompile(`<initial_action_dsl>([^<]*)</initial_action_dsl>\s*<initial_action_postfix>\s*</initial_action_postfix>`)},
	{"action", regexp.MustCompile(`<action_dsl>([^<]*)</action_dsl>\s*<action_postfix>\s*</action_postfix>`)},
	{"condition", regexp.MustCompile(`<condition_dsl>([^<]*)</condition_dsl>\s*<condition_postfix>\s*</condition_postfix>`)},
}

// compileFile reads f, compiles each DSL→postfix pair with an empty
// postfix slot, and writes f back. Returns (filled, skipped, errors).
//
// In the default mode any successful fills are written even when other
// elements in the same file failed to compile — the file is improved
// incrementally and the errors surface as a deliberate to-do list.
//
// With strict=true the file is written only if every non-comment DSL
// element compiles. Use --strict when you need an atomic guarantee
// that every element in the file is build-clean (CI gate behavior).
func compileFile(cmp *el.Compiler, f string, dryRun, strict bool) (int, int, []string) {
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
		n := 0
		newData = kp.re.ReplaceAllFunc(newData, func(match []byte) []byte {
			n++
			m := kp.re.FindSubmatch(match)
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

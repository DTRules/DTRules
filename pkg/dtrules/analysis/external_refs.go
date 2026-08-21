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

package analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ExternalRefKind classifies a way a decision table can depend on logic
// that isn't defined inside the project — the failure mode the authoring
// contract gate exists to reject. Each kind is a hard error for
// `dtrules verify`, not advisory: a table that leans on an undefined
// table, field, or operator is not self-contained and must not commit.
type ExternalRefKind string

const (
	// ExternalRefTable — a `perform <Name>` whose callee table isn't
	// defined anywhere in the project's *_dt.xml set.
	ExternalRefTable ExternalRefKind = "undefined_table"
	// ExternalRefUnboundedDispatch — a `perform table named (<expr>)` whose
	// expression has no literal segments, so no bound can be derived and no
	// target enumerated. The analyzer cannot reason about the rule set past
	// this point and says so (#776).
	ExternalRefUnboundedDispatch ExternalRefKind = "unbounded_dispatch"

	// ExternalRefField — a dotted `entity.attr` reference where `entity`
	// is a declared EDD entity but `attr` is not one of its declared
	// fields. The entity exists, so the attribute is a real typo or a
	// dependency on data the EDD never declares.
	ExternalRefField ExternalRefKind = "undefined_field"

	// ExternalRefOperator — a compiled postfix operator token that isn't
	// registered in the operator registry. The DSL named a function the
	// engine doesn't implement.
	ExternalRefOperator ExternalRefKind = "undefined_operator"
)

// ExternalRefFinding is one table → undefined-symbol dependency.
type ExternalRefFinding struct {
	Kind   ExternalRefKind
	Table  string // referencing decision table
	Symbol string // the undefined symbol (table name, entity.attr, or operator)
	DTFile string // *_dt.xml the reference was found in
}

// String renders a finding in a stable, human-readable form.
func (f ExternalRefFinding) String() string {
	switch f.Kind {
	case ExternalRefTable:
		return fmt.Sprintf("%s performs undefined table %s (%s)", f.Table, f.Symbol, f.DTFile)
	case ExternalRefUnboundedDispatch:
		return fmt.Sprintf("%s dispatches on an expression with no literal parts (%s) — no bound can be derived; "+
			"anchor the name with a literal prefix or suffix (%s)", f.Table, f.Symbol, f.DTFile)
	case ExternalRefField:
		return fmt.Sprintf("%s references undefined field %s (%s)", f.Table, f.Symbol, f.DTFile)
	case ExternalRefOperator:
		return fmt.Sprintf("%s uses undefined operator %q (%s)", f.Table, f.Symbol, f.DTFile)
	default:
		return fmt.Sprintf("%s references undefined %s %s (%s)", f.Table, f.Kind, f.Symbol, f.DTFile)
	}
}

// OperatorChecker reports whether tok is a registered operator. The
// operator registry lives in a package whose dependency weight we don't
// want to pull into this leaf static-analysis package, so the caller
// (cmd/dtrules, which already wires the registry) injects the lookup —
// typically `func(t string) bool { _, ok := operators.GetByString(t); return ok }`.
// A nil checker skips the operator check entirely.
type OperatorChecker func(tok string) bool

// AnalyzeExternalRefs walks every *_dt.xml under xmlDir and reports every
// way a table depends on a symbol the project doesn't define: undefined
// `perform` targets, undefined EDD field references, and (when ops is
// non-nil) postfix operators absent from the registry.
//
// Findings are sorted deterministically (kind, table, symbol) so callers
// and golden tests get stable output.
func AnalyzeExternalRefs(xmlDir string, isOperator OperatorChecker) ([]ExternalRefFinding, error) {
	var findings []ExternalRefFinding

	// (a) Undefined `perform` targets — reuse the call-graph orphan pass.
	graph, err := AnalyzeTableCallGraph(xmlDir)
	if err != nil {
		return nil, err
	}
	for _, o := range graph.OrphanCalls {
		findings = append(findings, ExternalRefFinding{
			Kind:   ExternalRefTable,
			Table:  o.Caller,
			Symbol: o.Callee,
			DTFile: o.DTFile,
		})
	}
	for _, d := range graph.UnboundedDispatch {
		findings = append(findings, ExternalRefFinding{
			Kind:   ExternalRefUnboundedDispatch,
			Table:  d.Caller,
			Symbol: d.Expr,
			DTFile: d.DTFile,
		})
	}

	// (b) Undefined EDD field references, and (c) undefined operators.
	schema, err := loadEDDSchema(xmlDir)
	if err != nil {
		return nil, err
	}
	symbols, err := buildProjectSymbols(xmlDir, schema, graph, isOperator)
	if err != nil {
		return nil, err
	}
	refFindings, err := collectUndefinedSymbols(xmlDir, symbols)
	if err != nil {
		return nil, err
	}
	findings = append(findings, refFindings...)

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Table != findings[j].Table {
			return findings[i].Table < findings[j].Table
		}
		return findings[i].Symbol < findings[j].Symbol
	})
	return findings, nil
}

// projectSymbols is the complete name space a reference can resolve
// against. A name "exists" if it is any one of these — that is the whole
// point: a postfix token like `client` is not an undefined operator, it is
// a declared entity; resolving against the operator table alone is what
// produced the false positives. Existence is the union, not any single
// table.
type projectSymbols struct {
	schema     *eddSchema      // declared entities and their fields
	entities   map[string]bool // declared entity names (lowercase)
	fieldAttrs map[string]bool // every declared field's attribute name (lowercase), any entity
	tables     map[string]bool // defined decision-table names (lowercase)
	locals     map[string]bool // DSL-declared locals and iteration aliases (lowercase)
	isOperator OperatorChecker // registry lookup; nil ⇒ skip the operator check
}

// localDeclPattern captures names introduced by `local <type> <name>`
// declarations. The element types mirror the EL grammar's LOCAL forms.
var localDeclPattern = regexp.MustCompile(
	`(?i)\blocal\s+(?:entity|long|double|boolean|date|array|string|bigint|fixed|integer)\s+([a-z_][a-z0-9_]*)`)

// aliasPattern captures iteration/binding aliases: `for each <v> in ...`
// and the `as <alias>` tail of `for all ... as` and `create ... as`.
var aliasPattern = regexp.MustCompile(
	`(?i)\bfor\s+each\s+([a-z_][a-z0-9_]*)\s+in\b|\bas\s+([a-z_][a-z0-9_]*)`)

// slashNamePattern captures postfix name literals — `/ira_max allocate`,
// `/result.foo xdef` — that introduce a name into the program's dictionary.
// The leading `/` must follow whitespace, line start, or `(` so XML closing
// tags (`</field>`) don't match. A name pushed via `/name` provably exists,
// so its later bare use is not an undefined operator.
var slashNamePattern = regexp.MustCompile(`(?:^|[\s(])/([a-z_][a-z0-9_]*)`)

// buildProjectSymbols loads the full name space: EDD entities/fields, the
// defined table names from the call graph, and every local/alias declared
// in any DSL fragment. Locals are collected project-wide (not per-table) on
// purpose — over-inclusion only ever suppresses a finding, and for a hard
// gate a missed finding is far cheaper than a false positive that blocks a
// legitimate commit.
func buildProjectSymbols(xmlDir string, schema *eddSchema, graph *TableCallGraph, isOperator OperatorChecker) (*projectSymbols, error) {
	s := &projectSymbols{
		schema:     schema,
		entities:   make(map[string]bool),
		fieldAttrs: make(map[string]bool),
		tables:     make(map[string]bool),
		locals:     make(map[string]bool),
		isOperator: isOperator,
	}
	for entity, attrs := range schema.FieldsByEntity {
		s.entities[entity] = true
		for attr := range attrs {
			s.fieldAttrs[attr] = true
		}
	}
	for table := range graph.Tables {
		s.tables[strings.ToLower(table)] = true
	}

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), "_dt.xml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", d.Name(), err)
		}
		for _, m := range localDeclPattern.FindAllStringSubmatch(strings.ToLower(string(data)), -1) {
			s.locals[m[1]] = true
		}
		for _, m := range aliasPattern.FindAllStringSubmatch(strings.ToLower(string(data)), -1) {
			if m[1] != "" {
				s.locals[m[1]] = true
			}
			if m[2] != "" {
				s.locals[m[2]] = true
			}
		}
		for _, m := range slashNamePattern.FindAllStringSubmatch(strings.ToLower(string(data)), -1) {
			s.locals[m[1]] = true
		}
		return nil
	})
	return s, err
}

// knownBareName reports whether a bare (non-dotted) token resolves to any
// declared symbol: a registered operator, an EL keyword, a declared entity,
// any declared field name, a defined table, or a local/alias.
func (s *projectSymbols) knownBareName(tok string) bool {
	if s.isOperator != nil && s.isOperator(tok) {
		return true // operators may be case- or symbol-sensitive: check as-is
	}
	lc := strings.ToLower(tok)
	return elKeywords[lc] ||
		s.entities[lc] ||
		s.fieldAttrs[lc] ||
		s.tables[lc] ||
		s.locals[lc]
}

// collectUndefinedSymbols scans each table's DSL for dotted references whose
// entity is declared but whose field isn't, and scans each table's compiled
// postfix for bare tokens that resolve to no declared symbol at all.
func collectUndefinedSymbols(xmlDir string, symbols *projectSymbols) ([]ExternalRefFinding, error) {
	var findings []ExternalRefFinding

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_dt.xml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		var doc struct {
			Tables []struct {
				Name           string         `xml:"table_name"`
				Contexts       []contextEntry `xml:"contexts>context_details"`
				InitialActions []struct {
					DSL     string `xml:"initial_action_dsl"`
					Postfix string `xml:"action_postfix"`
				} `xml:"initial_actions>initial_action"`
				Conditions []struct {
					DSL     string `xml:"condition_dsl"`
					Postfix string `xml:"condition_postfix"`
				} `xml:"conditions>condition_details"`
				Actions []struct {
					DSL     string `xml:"action_dsl"`
					Postfix string `xml:"action_postfix"`
				} `xml:"actions>action_details"`
			} `xml:"decision_table"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil // skip unparseable, mirroring the other analyzers
		}

		for _, t := range doc.Tables {
			if t.Name == "" {
				continue
			}

			// Gather every DSL fragment and every postfix fragment for
			// this table.
			var dslFrags, postfixFrags []string
			for _, c := range t.Contexts {
				dslFrags = append(dslFrags, c.DSL)
			}
			for _, c := range t.Conditions {
				dslFrags = append(dslFrags, c.DSL)
				postfixFrags = append(postfixFrags, c.Postfix)
			}
			for _, a := range t.InitialActions {
				dslFrags = append(dslFrags, a.DSL)
				postfixFrags = append(postfixFrags, a.Postfix)
			}
			for _, a := range t.Actions {
				dslFrags = append(dslFrags, a.DSL)
				postfixFrags = append(postfixFrags, a.Postfix)
			}

			// (b) Undefined field references: entity declared, field not.
			for _, ref := range undefinedFieldRefs(dslFrags, symbols) {
				findings = append(findings, ExternalRefFinding{
					Kind:   ExternalRefField,
					Table:  t.Name,
					Symbol: ref,
					DTFile: name,
				})
			}

			// (c) Undefined operators: a postfix token that resolves to no
			// declared symbol at all. Needs the operator registry to tell
			// real operators from typos.
			if symbols.isOperator != nil {
				for _, op := range undefinedOperators(postfixFrags, symbols) {
					findings = append(findings, ExternalRefFinding{
						Kind:   ExternalRefOperator,
						Table:  t.Name,
						Symbol: op,
						DTFile: name,
					})
				}
			}
		}
		return nil
	})
	return findings, err
}

// undefinedFieldRefs returns the sorted, unique set of dotted `entity.attr`
// references in dslFrags where `entity` is a declared EDD entity but `attr`
// is not a declared field of it.
//
// The gate is deliberately one-directional: it fires only when the entity
// root is a declared entity. A dotted reference whose root is NOT a declared
// entity (a local entity, an alias from `for each x in ...`, a runtime-
// reserved name like `mapping.key`) is left alone — the analyzer has no
// authority to call those undefined, and flagging them would produce false
// positives that block legitimate commits. A root that is also a declared
// local/alias is skipped for the same reason. This is a necessary-condition
// check: it catches fields that are provably absent from the entity's
// declaration, while accepting that runtime-materialized fields can't be
// ruled out statically.
func undefinedFieldRefs(dslFrags []string, symbols *projectSymbols) []string {
	seen := make(map[string]bool)
	var out []string
	for _, frag := range dslFrags {
		if frag == "" {
			continue
		}
		for _, m := range identifierPattern.FindAllStringSubmatch(strings.ToLower(frag), -1) {
			entity, attr := m[1], m[2]
			key := entity + "." + attr
			if seen[key] || isRuntimeReserved(key) {
				continue
			}
			if symbols.locals[entity] {
				continue // root is a local/alias — can't resolve its fields
			}
			attrs, known := symbols.schema.FieldsByEntity[entity]
			if !known {
				continue // root isn't a declared entity — not our call to make
			}
			if attrs[attr] {
				continue // declared field — fine
			}
			seen[key] = true
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// undefinedOperators returns the sorted, unique set of postfix tokens in
// postfixFrags that resolve to no declared symbol at all.
//
// The token classification mirrors the runtime's: a postfix stream is
// whitespace-delimited with quoted strings preserved. A token is exempt
// from the check when it's a block delimiter, a string/name/var literal
// (`"..."`, `'...'`, `/Name`, `$var`), numeric, a dotted field reference,
// or an uppercase-initial identifier (a typed entity/table name resolved
// via the dictionary). Everything else — a bare lowercase word or a symbol
// like `s==i` — must resolve to *something* the project declares: a
// registered operator, an EL keyword, a declared entity or field, a defined
// table, or a local. Resolving against the operator table alone is exactly
// what produced false positives like `client` (a declared entity); only a
// token that exists in none of those name spaces is genuinely undefined.
func undefinedOperators(postfixFrags []string, symbols *projectSymbols) []string {
	seen := make(map[string]bool)
	var out []string
	for _, frag := range postfixFrags {
		for _, tok := range tokenizePostfix(frag) {
			if !operatorCandidate(tok) {
				continue
			}
			if symbols.knownBareName(tok) {
				continue
			}
			if seen[tok] {
				continue
			}
			seen[tok] = true
			out = append(out, tok)
		}
	}
	sort.Strings(out)
	return out
}

// operatorCandidate reports whether tok is in the class of postfix tokens
// that are required to be registered operators. See undefinedOperators for
// the exemptions and why each is dictionary-resolved rather than an
// operator.
func operatorCandidate(tok string) bool {
	// Block delimiters, including the empty block the ifelse emitter writes
	// for an `if` with no else branch. The compiler builds these into
	// executable arrays; they never reach the operator registry, which is
	// why operators.nonOpEmits lists "{}" too. Missing this one made every
	// table containing a bare `if` fail the external-reference check.
	if tok == "" || tok == "{" || tok == "}" || tok == "{}" {
		return false
	}
	switch tok[0] {
	case '"', '\'': // string literal (double- or single-quoted)
		return false
	case '/': // name literal (table/entity reference pushed by name)
		return false
	case '$': // variable/local reference
		return false
	}
	if isNumericLiteral(tok) {
		return false
	}
	if isTypedNumericLiteral(tok) {
		return false
	}
	if strings.Contains(tok, ".") { // dotted field reference
		return false
	}
	if tok[0] >= 'A' && tok[0] <= 'Z' { // typed entity / table name
		return false
	}
	return true
}

// isTypedNumericLiteral reports whether tok is a numeric literal carrying a
// type marker: a fixed-point literal (`0fp`, `0.5fp`, `.5fp`) or a hex-bytes
// literal (`0x1f`). These are literals the compiler emits verbatim into
// postfix, not operators.
//
// isNumericLiteral cannot see them because it rejects any token containing a
// letter, so `0fp` was reported as an undefined operator and `dtrules verify`
// failed on a rule set that builds, loads and executes correctly (#1006). On
// the Accumulate staking rules that was 44 uses across 8 tables — enough to
// stop verify being usable as the authoring-contract gate in CI, which is the
// point of the check.
//
// `0.5fp` appeared to work, but only by accident: it contains a `.`, so the
// dotted-field-reference branch below let it through. Both forms are now
// recognised for the right reason, and the hex form — which had the identical
// defect and simply had not been hit yet — with them.
//
// The accepted shapes mirror the grammar exactly (EL.g4, FP_LITERAL and
// HEX_BYTES_LITERAL):
//
//	FP_LITERAL        : DIGIT+ '.' DIGIT* 'fp' | DIGIT* '.' DIGIT+ 'fp' | DIGIT+ 'fp'
//	HEX_BYTES_LITERAL : '0x' HEX_DIGIT*
func isTypedNumericLiteral(tok string) bool {
	if tok == "" {
		return false
	}
	// Hex bytes: 0x followed by hex digits (the grammar permits none). No
	// sign — it is a bytes literal, not an arithmetic one.
	if len(tok) >= 2 && tok[0] == '0' && (tok[1] == 'x' || tok[1] == 'X') {
		for _, r := range tok[2:] {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
		return true
	}

	// Fixed point: a numeric body followed by the fp suffix, matched
	// case-insensitively because EL names and keywords are. The sign is left
	// to isNumericLiteral, which permits exactly one; stripping it here as
	// well would have accepted "--3fp".
	if len(tok) < 3 {
		return false
	}
	suffix := tok[len(tok)-2:]
	if suffix != "fp" && suffix != "FP" && suffix != "Fp" && suffix != "fP" {
		return false
	}
	return isNumericLiteral(tok[:len(tok)-2])
}

// isNumericLiteral reports whether tok is an integer or decimal numeric
// literal (optionally signed). Operator symbols like `-` or `f>` are not
// numeric and remain operator candidates.
func isNumericLiteral(tok string) bool {
	if tok == "" {
		return false
	}
	hasDigit := false
	for i, r := range tok {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '.' || r == '_':
			// thousands/decimal separators within a number
		case (r == '-' || r == '+') && i == 0:
			// leading sign
		default:
			return false
		}
	}
	return hasDigit
}

// tokenizePostfix splits a postfix string into whitespace-delimited tokens,
// preserving single- or double-quoted string literals (which may contain
// spaces) as a single token. There are no escape sequences in the postfix
// the compiler emits, so a quote simply runs to the next matching quote.
//
// Line comments are dropped: a `%`, `//`, or `#` that begins a token runs
// to the end of the line. The compiler preserves such comments into the
// stored postfix, and their prose ("% For 2025: max enlisted pay ...") must
// not be mistaken for operator tokens.
func tokenizePostfix(postfix string) []string {
	var tokens []string
	var cur strings.Builder
	var quote rune // 0 when not inside a string; otherwise the opening quote

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	runes := []rune(postfix)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote != 0:
			cur.WriteRune(r)
			if r == quote {
				flush()
				quote = 0
			}
		case r == '"' || r == '\'':
			cur.WriteRune(r)
			quote = r
		case cur.Len() == 0 && (r == '%' || r == '#' ||
			(r == '/' && i+1 < len(runes) && runes[i+1] == '/')):
			// Line comment at a token boundary — skip to end of line.
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

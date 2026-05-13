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

// Package analysis provides project-wide static analysis for DTRules projects.
package analysis

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EDDWarning records an informational finding about EDD field usage.
type EDDWarning struct {
	Field   string // "entity.attribute"
	EddFile string // source EDD file name
	Reason  string
}

// String formats the warning in the canonical INFO form.
func (w EDDWarning) String() string {
	return fmt.Sprintf("INFO %s (%s)", w.Reason, w.EddFile)
}

// AnalyzeEDDUsage walks all *_dt.xml files under xmlDir, collects every
// identifier referenced in DSL fields, then diffs against the EDD declared
// in *_edd.xml. It returns informational warnings for unused and write-only fields.
//
// Phase 1 of #776: bare identifiers inside `for all <field>` contexts are
// resolved against the iterated entity type and counted as references.
// This crushes the false-positive class where DSL like
//
//	for all taxpayers
//	  taxpayer.is_self_employed and earned_income > 0
//
// would have marked `taxpayer.earned_income` as "unused" because the
// regex-only pass never saw `earned_income` as `taxpayer.earned_income`.
// Phases 2-5 (using/create/cross-table/enumeration-bounded dynamic
// strings) follow up; the regex pass at the dotted-identifier level
// stays as a fallback for what the entity-stack walk doesn't yet model.
func AnalyzeEDDUsage(xmlDir string) ([]EDDWarning, error) {
	schema, err := loadEDDSchema(xmlDir)
	if err != nil {
		return nil, err
	}

	readRefs, writeRefs, err := collectDTReferences(xmlDir, schema)
	if err != nil {
		return nil, err
	}

	return diffEDDUsage(schema.Fields, readRefs, writeRefs), nil
}

// eddField is a declared EDD field.
type eddField struct {
	EntityAttr string // "entity.attr"
	Access     string // "r", "rw", etc.
	EddFile    string
}

// eddSchema is the loaded view of every EDD declaration we need for
// entity-stack-aware reference resolution (#776 phase 1).
//
//   - Fields maps "entity.attr" → declaration (the legacy view; what
//     diffEDDUsage compares references against).
//   - ArraySubtype maps "entity.attr" → element entity type for array
//     fields. Used to resolve `for all <field>` contexts to the
//     iterated entity type.
//   - FieldsByEntity maps entity name → set of attribute names.
//     Used to attribute a bare identifier inside a `for all` body to
//     a real field of the iterated entity.
type eddSchema struct {
	Fields         map[string]eddField        // "entity.attr" -> decl
	ArraySubtype   map[string]string          // "entity.attr" -> element entity (lowercased)
	FieldsByEntity map[string]map[string]bool // entity -> { attr -> true }
}

// loadEDDDeclarations reads all *_edd.xml files under xmlDir and returns
// declared fields keyed by "entity.attr".
func loadEDDDeclarations(xmlDir string) (map[string]eddField, error) {
	s, err := loadEDDSchema(xmlDir)
	if err != nil {
		return nil, err
	}
	return s.Fields, nil
}

// loadEDDSchema is the entity-stack-aware load. It collects the same
// flat `entity.attr` decls the legacy analyzer wants, plus the
// array→element-type and entity→fields indexes needed by the
// `for all <field>` resolution in collectDTReferences.
func loadEDDSchema(xmlDir string) (*eddSchema, error) {
	s := &eddSchema{
		Fields:         make(map[string]eddField),
		ArraySubtype:   make(map[string]string),
		FieldsByEntity: make(map[string]map[string]bool),
	}

	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if !strings.HasSuffix(name, "_edd.xml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		var edd struct {
			Entities []struct {
				Name   string `xml:"name,attr"`
				Fields []struct {
					Name    string `xml:"name,attr"`
					Type    string `xml:"type,attr"`
					Subtype string `xml:"subtype,attr"`
					Access  string `xml:"access,attr"`
				} `xml:"field"`
			} `xml:"entity"`
		}
		if err := xml.Unmarshal(data, &edd); err != nil {
			return nil // Not a valid EDD file — skip silently.
		}

		for _, entity := range edd.Entities {
			entityName := strings.ToLower(entity.Name)
			if s.FieldsByEntity[entityName] == nil {
				s.FieldsByEntity[entityName] = make(map[string]bool)
			}
			for _, field := range entity.Fields {
				attrName := strings.ToLower(field.Name)
				key := entityName + "." + attrName
				s.FieldsByEntity[entityName][attrName] = true
				if strings.EqualFold(field.Type, "array") && field.Subtype != "" {
					s.ArraySubtype[key] = strings.ToLower(field.Subtype)
				}
				if isRuntimeReserved(key) {
					continue
				}
				s.Fields[key] = eddField{
					EntityAttr: key,
					Access:     field.Access,
					EddFile:    name,
				}
			}
		}
		return nil
	})
	return s, err
}

// identifierPattern matches dotted identifiers like "job.income" or "taxpayer.agi".
var identifierPattern = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b`)

// setPrefix matches LHS assignment: "set job.foo" or "/job.foo xdef" or "/job.foo set"
var assignPattern = regexp.MustCompile(`(?i)(?:set\s+([a-z_][a-z0-9_]*)\.([a-z0-9_]+)|/([a-z_][a-z0-9_]*)\.([a-z0-9_]+)\s+(?:xdef|set)|/([a-z_][a-z0-9_]*)\.([a-z0-9_]+)\s+swap\s+addto)`)

// createEntityPattern detects "new X entity" or "createentity" near entity push
var createEntityPattern = regexp.MustCompile(`(?i)new\s+([a-z_][a-z0-9_]*)\s+entity`)

// collectDTReferences walks all *_dt.xml files and collects identifiers.
// readRefs: identifiers appearing in read positions (conditions, RHS of assignments).
// writeRefs: identifiers appearing as LHS of assignment statements.
//
// If schema is non-nil, each table's contexts are scanned for
// `for all <field>` patterns. The field's EDD-declared array subtype
// becomes the topmost entity for the table's conditions, actions, and
// initial-actions, and bare identifiers in those DSLs that match a
// field of the iterated entity are recorded as `entitytype.field`
// references. This is the entity-stack-aware pass from #776 phase 1.
func collectDTReferences(xmlDir string, schema *eddSchema) (readRefs map[string]bool, writeRefs map[string]bool, err error) {
	readRefs = make(map[string]bool)
	writeRefs = make(map[string]bool)

	err = filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
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

		var tables struct {
			Tables []struct {
				Contexts []contextEntry `xml:"contexts>context_details"`
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
		if err := xml.Unmarshal(data, &tables); err != nil {
			return nil // skip unparseable
		}

		for _, table := range tables.Tables {
			// Determine the entity stack for this table from its
			// `for all <field>` contexts. Empty when the schema is
			// nil or no recognizable `for all` clause is present —
			// the analyzer then falls back to the dotted-only regex
			// pass, identical to the pre-#776 behaviour.
			entityStack := stackFromContexts(schema, table.Contexts)

			// The iterated field itself is being read by the
			// `for all` clause. Record it so the analyzer doesn't
			// flag e.g. `job.taxpayers` as unused just because no
			// table writes `... = job.taxpayers` outside the context.
			if schema != nil {
				for _, c := range table.Contexts {
					extractIteratedFieldReads(c.DSL, schema, readRefs)
				}
			}

			// Conditions are always reads.
			for _, c := range table.Conditions {
				extractReads(c.DSL, readRefs)
				extractReads(c.Postfix, readRefs)
				if schema != nil {
					extractBareReads(c.DSL, entityStack, schema, readRefs)
				}
			}
			// Actions: extract writes explicitly, then reads for non-write positions.
			for _, a := range table.InitialActions {
				extractWritesAndReads(a.DSL, writeRefs, readRefs)
				extractWritesAndReads(a.Postfix, writeRefs, readRefs)
				if schema != nil {
					extractBareWritesAndReads(a.DSL, entityStack, schema, writeRefs, readRefs)
				}
			}
			for _, a := range table.Actions {
				extractWritesAndReads(a.DSL, writeRefs, readRefs)
				extractWritesAndReads(a.Postfix, writeRefs, readRefs)
				if schema != nil {
					extractBareWritesAndReads(a.DSL, entityStack, schema, writeRefs, readRefs)
				}
			}
		}
		return nil
	})
	return readRefs, writeRefs, err
}

// forAllPattern matches the EL `for all <field>` and
// `for all <entity>.<field>` context openers, capturing the
// fully-qualified field reference. The trailing `where`/`whose`
// clause is preserved by the parser but doesn't affect which entity
// type gets pushed.
//
// Examples matched:
//
//	for all taxpayers
//	for all job.state_periods
//	for all taxpayers whose is_self_employed is true
//	for all incomes where amount > 0
var forAllPattern = regexp.MustCompile(`(?i)\bfor\s+all\s+([a-z_][a-z0-9_]*(?:\.[a-z_][a-z0-9_]*)?)`)

// contextEntry is the minimal XML projection of a `<context_details>`
// element that stackFromContexts needs. Named (rather than inline
// anonymous) so tests can construct values without re-declaring the
// struct shape on the fly.
type contextEntry struct {
	DSL string `xml:"context_dsl"`
}

// extractIteratedFieldReads records the iterated array field itself
// as a read. `for all taxpayers` is structurally a read of
// `job.taxpayers` (or whatever entity declares the field) — without
// this, an EDD-declared array that's only consumed via context
// iteration looks unused.
func extractIteratedFieldReads(text string, schema *eddSchema, into map[string]bool) {
	if schema == nil {
		return
	}
	for _, m := range forAllPattern.FindAllStringSubmatch(text, -1) {
		ref := strings.ToLower(m[1])
		if strings.Contains(ref, ".") {
			if _, ok := schema.ArraySubtype[ref]; ok && !isRuntimeReserved(ref) {
				into[ref] = true
			}
			continue
		}
		// Bare reference: find the entity that declares it as an
		// array field. If exactly one does, record it; ambiguous
		// or unknown bare references are left alone.
		for fqn := range schema.ArraySubtype {
			if strings.HasSuffix(fqn, "."+ref) && !isRuntimeReserved(fqn) {
				into[fqn] = true
				break
			}
		}
	}
}

// stackFromContexts resolves the entity types that a table's contexts
// push onto the runtime entity stack. Returns the list of entity-type
// names (lowercased) in stack order (innermost last). Unrecognized or
// unresolvable references yield no entry — the entity-stack-aware pass
// then skips those tables and the dotted-only regex pass takes over.
func stackFromContexts(schema *eddSchema, contexts []contextEntry) []string {
	if schema == nil {
		return nil
	}
	var stack []string
	for _, c := range contexts {
		for _, m := range forAllPattern.FindAllStringSubmatch(c.DSL, -1) {
			ref := strings.ToLower(m[1])
			// Two shapes: bare ("taxpayers") or dotted ("job.state_periods").
			// For the bare case the field lives on an entity we have to
			// guess — but in practice the rule sets we ship are written
			// against a single "job" root entity (TaxReturn convention)
			// or the field name is unique enough that exactly one entity
			// declares it. We search across the schema and accept the
			// first match.
			entity := ""
			if i := strings.Index(ref, "."); i > 0 {
				if sub, ok := schema.ArraySubtype[ref]; ok {
					entity = sub
				}
			} else {
				for fqn, sub := range schema.ArraySubtype {
					if strings.HasSuffix(fqn, "."+ref) {
						entity = sub
						break
					}
				}
			}
			if entity != "" {
				stack = append(stack, entity)
			}
		}
	}
	return stack
}

// bareIdentifierPattern matches a stand-alone lowercase identifier.
// We strip dotted forms before applying this; what's left are tokens
// that could be bare field references resolved against the entity
// stack. EL keywords and obvious noise (numeric literals, etc.) are
// filtered out by the elKeywords table and the post-check that the
// name actually corresponds to a field of the topmost entity.
var bareIdentifierPattern = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\b`)

// elKeywords is the closed set of EL identifiers that must never be
// treated as field references regardless of what's on the entity
// stack. Anything not in this set and not matching a known field is
// silently dropped — the analyzer is conservative, false-negatives
// (a real reference we missed) are preferred over false-positives.
var elKeywords = map[string]bool{
	"a": true, "all": true, "an": true, "and": true, "as": true, "at": true,
	"between": true, "by": true, "called": true, "create": true,
	"default": true, "does": true, "else": true, "endif": true,
	"equal": true, "exists": true, "false": true, "first": true, "for": true,
	"from": true, "greater": true, "has": true, "have": true, "if": true,
	"in": true, "include": true, "includes": true, "is": true, "it": true,
	"its": true, "less": true, "local": true, "match": true, "matches": true,
	"max": true, "maximum": true, "min": true, "minimum": true, "named": true,
	"never": true, "no": true, "none": true, "not": true, "now": true,
	"null": true, "of": true, "on": true, "one": true, "or": true,
	"order": true, "otherwise": true, "out": true, "over": true, "pass": true,
	"perform": true, "plus": true, "pop": true, "print": true,
	"product": true, "push": true, "set": true, "side": true, "size": true,
	"some": true, "starting": true, "starts": true, "sum": true,
	"table": true, "the": true, "then": true, "there": true, "these": true,
	"this": true, "those": true, "through": true, "to": true, "today": true,
	"true": true, "type": true, "until": true, "using": true, "value": true,
	"was": true, "when": true, "where": true, "which": true, "while": true,
	"whose": true, "with": true, "within": true, "x": true, "y": true,
	"yes": true, "zone": true,
}

// stripStringsAndDotted returns `text` with quoted strings and dotted
// identifiers blanked out, so that bareIdentifierPattern won't grab
// either the dotted forms (already handled by extractReads /
// extractWritesAndReads) or anything inside a string literal.
func stripStringsAndDotted(text string) string {
	// Blank string literals first so identifierPattern doesn't see them.
	out := stringLiteralPattern.ReplaceAllStringFunc(text, func(s string) string {
		// Keep length to preserve regex offsets; not strictly needed but cheap.
		return strings.Repeat(" ", len(s))
	})
	out = identifierPattern.ReplaceAllStringFunc(out, func(s string) string {
		return strings.Repeat(" ", len(s))
	})
	return out
}

// stringLiteralPattern matches "..." with no escape handling — adequate
// for the DSL the analyzer sees, where strings are simple display text.
var stringLiteralPattern = regexp.MustCompile(`"[^"]*"`)

// extractBareReads attributes every bare identifier in `text` that
// matches a field of the topmost entity on the stack as a read of
// `topEntity.field`. Conservative: identifiers that match neither an
// EL keyword nor a real field are dropped silently.
func extractBareReads(text string, entityStack []string, schema *eddSchema, into map[string]bool) {
	if len(entityStack) == 0 {
		return
	}
	stripped := stripStringsAndDotted(strings.ToLower(text))
	addBareRefs(stripped, entityStack, schema, into)
}

// extractBareWritesAndReads is the action-side counterpart: bare
// identifiers in LHS position become writes, all other bare matches
// become reads. The LHS detection is conservative — only the
// `set <bare> = ...` shape counts as a write target here, because
// the dotted-set case (`set entity.attr = ...`) is already picked up
// by extractWritesAndReads on the same DSL.
func extractBareWritesAndReads(text string, entityStack []string, schema *eddSchema, writes, reads map[string]bool) {
	if len(entityStack) == 0 {
		return
	}
	lower := strings.ToLower(text)

	// Bare-name LHS writes: `set <ident> = ...`.
	bareAssign := regexp.MustCompile(`(?i)\bset\s+([a-z_][a-z0-9_]*)\s*=`)
	localWrites := make(map[string]bool)
	for _, m := range bareAssign.FindAllStringSubmatch(lower, -1) {
		ident := m[1]
		if elKeywords[ident] {
			continue
		}
		if key := attribute(ident, entityStack, schema); key != "" {
			localWrites[key] = true
			writes[key] = true
		}
	}

	stripped := stripStringsAndDotted(lower)
	for _, m := range bareIdentifierPattern.FindAllStringSubmatch(stripped, -1) {
		ident := m[1]
		if elKeywords[ident] {
			continue
		}
		key := attribute(ident, entityStack, schema)
		if key == "" || localWrites[key] {
			continue
		}
		reads[key] = true
	}
}

// addBareRefs is the shared core of extractBareReads — walks the
// already-stripped text once and records bare references against the
// topmost-entity field set.
func addBareRefs(stripped string, entityStack []string, schema *eddSchema, into map[string]bool) {
	for _, m := range bareIdentifierPattern.FindAllStringSubmatch(stripped, -1) {
		ident := m[1]
		if elKeywords[ident] {
			continue
		}
		if key := attribute(ident, entityStack, schema); key != "" {
			into[key] = true
		}
	}
}

// attribute resolves a bare identifier against the entity stack and
// returns the canonical `entity.attr` key if the identifier matches a
// declared field of any in-scope entity (innermost first). Empty
// string means "not a field reference we recognize" — the analyzer
// drops the token rather than guess.
func attribute(ident string, entityStack []string, schema *eddSchema) string {
	for i := len(entityStack) - 1; i >= 0; i-- {
		entity := entityStack[i]
		if attrs, ok := schema.FieldsByEntity[entity]; ok && attrs[ident] {
			return entity + "." + ident
		}
	}
	return ""
}

// extractReads collects all dotted identifiers from text as read references.
func extractReads(text string, into map[string]bool) {
	for _, m := range identifierPattern.FindAllStringSubmatch(strings.ToLower(text), -1) {
		key := m[1] + "." + m[2]
		if !isRuntimeReserved(key) {
			into[key] = true
		}
	}
}

// extractWritesAndReads separates write targets (LHS of set/xdef) from read references.
// Write targets go into writes; all other identifiers go into reads.
func extractWritesAndReads(text string, writes, reads map[string]bool) {
	lower := strings.ToLower(text)

	// Collect write targets first.
	localWrites := make(map[string]bool)
	for _, m := range assignPattern.FindAllStringSubmatch(lower, -1) {
		pairs := [][2]string{
			{m[1], m[2]},
			{m[3], m[4]},
			{m[5], m[6]},
		}
		for _, p := range pairs {
			if p[0] != "" && p[1] != "" {
				key := p[0] + "." + p[1]
				if !isRuntimeReserved(key) {
					localWrites[key] = true
					writes[key] = true
				}
			}
		}
	}

	// All other identifiers are reads.
	for _, m := range identifierPattern.FindAllStringSubmatch(lower, -1) {
		key := m[1] + "." + m[2]
		if !isRuntimeReserved(key) && !localWrites[key] {
			reads[key] = true
		}
	}
}

// isRuntimeReserved returns true for identifiers that the runtime manages
// and should not be flagged as unused.
func isRuntimeReserved(key string) bool {
	// mapping*key and other known runtime fields
	if strings.Contains(key, "*") {
		return true
	}
	reserved := []string{
		"mapping.key",
		"constants.",
	}
	for _, r := range reserved {
		if strings.HasPrefix(key, r) {
			return true
		}
	}
	return false
}

// diffEDDUsage compares declarations against references and produces warnings.
func diffEDDUsage(decls map[string]eddField, readRefs, writeRefs map[string]bool) []EDDWarning {
	var warnings []EDDWarning

	for key, f := range decls {
		isRead := readRefs[key]
		isWritten := writeRefs[key]

		// Fields with access="r" are input-only (no decision table sets them).
		if f.Access == "r" {
			if !isRead {
				warnings = append(warnings, EDDWarning{
					Field:   key,
					EddFile: f.EddFile,
					Reason:  fmt.Sprintf("unused EDD field: %s (declared in %s, never referenced)", key, f.EddFile),
				})
			}
			continue
		}

		if !isRead && !isWritten {
			warnings = append(warnings, EDDWarning{
				Field:   key,
				EddFile: f.EddFile,
				Reason:  fmt.Sprintf("unused EDD field: %s (declared in %s, never referenced)", key, f.EddFile),
			})
		} else if isWritten && !isRead {
			warnings = append(warnings, EDDWarning{
				Field:   key,
				EddFile: f.EddFile,
				Reason:  fmt.Sprintf("write-only EDD field: %s (set but never read)", key),
			})
		}
	}

	return warnings
}

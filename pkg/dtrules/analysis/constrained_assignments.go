// Copyright 2026 Paul Snow
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

	"github.com/DTRules/DTRules/pkg/dtrules/loader"
)

// ConstraintAdvisory is one action that assigns a string literal outside the
// declared vocabulary of the field it writes (#1209).
type ConstraintAdvisory struct {
	Table        string   `json:"table"`
	ActionNumber string   `json:"action_number"`
	Field        string   `json:"field"` // entity.field, authored spelling
	Literal      string   `json:"literal"`
	Allowed      []string `json:"allowed_values"`
	DTFile       string   `json:"dt_file"`
}

func (a ConstraintAdvisory) String() string {
	return fmt.Sprintf("INFO constrained assignment: %s action %s sets %s = %q, which is not one of [%s] (%s)",
		a.Table, a.ActionNumber, a.Field, a.Literal, strings.Join(a.Allowed, ", "), a.DTFile)
}

// setLiteral matches `set <ref> = "<literal>"` — the only assignment shape a
// vocabulary can be checked against statically. Anything computed is the
// rules' business and is not reported.
var setLiteral = regexp.MustCompile(`(?i)\bset\s+([A-Za-z_][A-Za-z0-9_.*]*)\s*=\s*"([^"]*)"`)

// AnalyzeConstrainedAssignments reports actions whose DSL assigns a literal
// that the target field's declared vocabulary does not contain (#1209).
//
// This is advisory and never an error. Enforcement stops at the boundary:
// a value handed in from outside the rules is refused, but a table may set
// whatever it computes — a rule set is allowed to know something the EDD's
// vocabulary hasn't been told yet. What a literal out of the set almost
// always is, though, is a typo ("Bannana"), and a typo is worth a line in a
// review.
func AnalyzeConstrainedAssignments(xmlDir string) ([]ConstraintAdvisory, error) {
	vocab := collectVocabularies(xmlDir)
	if len(vocab) == 0 {
		return nil, nil
	}

	var out []ConstraintAdvisory
	err := filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_dt.xml") || loader.SkipRuleFile(p) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		var doc dtDoc
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, t := range doc.Tables {
			for _, a := range t.Actions {
				out = append(out, checkAssignmentDSL(vocab, t.Name, a.Number, a.DSL, filepath.Base(p))...)
			}
			for _, a := range t.InitialActions {
				out = append(out, checkAssignmentDSL(vocab, t.Name, "initial", a.DSL, filepath.Base(p))...)
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Table != out[j].Table {
			return out[i].Table < out[j].Table
		}
		return out[i].ActionNumber < out[j].ActionNumber
	})
	return out, err
}

// checkAssignmentDSL reports every `set field = "literal"` in one action's DSL
// whose literal is outside the field's vocabulary.
func checkAssignmentDSL(vocab map[string]*vocabEntry, table, actionNumber, dsl, dtFile string) []ConstraintAdvisory {
	var out []ConstraintAdvisory
	for _, m := range setLiteral.FindAllStringSubmatch(dsl, -1) {
		ref, literal := m[1], m[2]
		v := vocab[strings.ToLower(ref)]
		if v == nil || v.ambiguous {
			continue
		}
		if matchesVocabulary(v.values, literal) {
			continue
		}
		out = append(out, ConstraintAdvisory{
			Table:        table,
			ActionNumber: actionNumber,
			Field:        v.qualified,
			Literal:      literal,
			Allowed:      v.values,
			DTFile:       dtFile,
		})
	}
	return out
}

// matchesVocabulary reports whether literal is in values. Case-insensitive,
// the same rule EL uses for names and the runtime gate uses for values.
func matchesVocabulary(values []string, literal string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(literal)) {
			return true
		}
	}
	return false
}

// vocabEntry is one field's declared vocabulary, keyed both by its qualified
// name and (when only one entity declares it) by the bare field name, because
// a table with the entity on its context writes the field unqualified.
type vocabEntry struct {
	qualified string // entity.field, authored spelling
	values    []string
	ambiguous bool // bare name declared by more than one entity: can't resolve
}

// collectVocabularies indexes every field in the project that declares an
// allowed_value, by qualified name and by bare field name.
func collectVocabularies(xmlDir string) map[string]*vocabEntry {
	vocab := map[string]*vocabEntry{}
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_edd.xml") || loader.SkipRuleFile(p) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		var doc constraintEDDDoc
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, ent := range doc.Entities {
			for _, f := range ent.Fields {
				var values []string
				for _, v := range f.AllowedValues {
					if s := strings.TrimSpace(v.Value); s != "" {
						values = append(values, s)
					}
				}
				if len(values) == 0 {
					continue
				}
				e := &vocabEntry{qualified: ent.Name + "." + f.Name, values: values}
				vocab[strings.ToLower(e.qualified)] = e
				bare := strings.ToLower(f.Name)
				if prior, seen := vocab[bare]; seen && prior.qualified != e.qualified {
					prior.ambiguous = true
				} else {
					vocab[bare] = e
				}
			}
		}
		return nil
	})
	return vocab
}

// constraintEDDDoc is the slice of an EDD this pass reads: the fields that
// declare a vocabulary.
type constraintEDDDoc struct {
	Entities []struct {
		Name   string `xml:"name,attr"`
		Fields []struct {
			Name          string `xml:"name,attr"`
			AllowedValues []struct {
				Value string `xml:"value,attr"`
			} `xml:"allowed_value"`
		} `xml:"field"`
	} `xml:"entity"`
}

// dtDoc is the slice of a decision-table file this pass reads: each table's
// name and the DSL of its actions.
type dtDoc struct {
	Tables []struct {
		Name    string `xml:"table_name"`
		Actions []struct {
			Number string `xml:"action_number"`
			DSL    string `xml:"action_dsl"`
		} `xml:"actions>action_details"`
		InitialActions []struct {
			DSL string `xml:"initial_action_dsl"`
		} `xml:"initial_actions>initial_action"`
	} `xml:"decision_table"`
}

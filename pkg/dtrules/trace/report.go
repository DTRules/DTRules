// Copyright 2026 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// ReportSpec is the user-composed description of a trace report: which
// entities to show, which of their EDD fields, filtered and sorted how.
// Specs are plain JSON, saveable in the project (reports/*.report.json),
// and run identically against a baseline or a speculative trace — which is
// what makes outcome diffs possible.
type ReportSpec struct {
	Name     string          `json:"name"`
	Sections []ReportSection `json:"sections"`
}

// ReportSection selects one set of rows.
//
// Instance source, one of:
//   - Entity: every instance of that entity the run created or touched
//     (e.g. "token_recipient" lists all recipients built during the run).
//   - Source "entity.attr": the elements of that array attribute on the
//     final state (e.g. "staking_report.payouts").
//
// Fields are EDD attribute names (empty = every attribute, sorted). Where
// filters rows; Key names the field used to align rows in diffs (defaults
// to the first field).
type ReportSection struct {
	Title  string         `json:"title,omitempty"`
	Entity string         `json:"entity,omitempty"`
	Source string         `json:"source,omitempty"`
	Fields []string       `json:"fields,omitempty"`
	Where  []ReportFilter `json:"where,omitempty"`
	Sort   string         `json:"sort,omitempty"`
	Key    string         `json:"key,omitempty"`
}

// ReportFilter is one predicate on a field. Ops: == != > >= < <= contains.
// Comparison is numeric when both sides parse as numbers, else
// case-insensitive text (EL semantics).
type ReportFilter struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

// Report is the result of running a spec against a trace.
type Report struct {
	Name     string                `json:"name"`
	Sections []ReportSectionResult `json:"sections"`
}

// ReportSectionResult is one rendered section: column order in Fields,
// row values keyed by field name. Total counts rows before filtering.
type ReportSectionResult struct {
	Title  string              `json:"title"`
	Entity string              `json:"entity"`
	Key    string              `json:"key"`
	Fields []string            `json:"fields"`
	Rows   []map[string]string `json:"rows"`
	Total  int                 `json:"total"`
	Error  string              `json:"error,omitempty"`
}

// GenerateReport runs spec against a trace that has been replayed to its
// end (SetState to the finalState node): instances and final values are
// read from the replay session.
func (t *Trace) GenerateReport(sess dtrules.Session, spec ReportSpec) *Report {
	rep := &Report{Name: spec.Name}
	for _, sec := range spec.Sections {
		rep.Sections = append(rep.Sections, t.generateSection(sess, sec))
	}
	return rep
}

func (t *Trace) generateSection(sess dtrules.Session, sec ReportSection) ReportSectionResult {
	res := ReportSectionResult{Title: sec.Title, Entity: sec.Entity}
	if res.Title == "" {
		if sec.Source != "" {
			res.Title = sec.Source
		} else {
			res.Title = sec.Entity
		}
	}

	instances, err := t.sectionInstances(sess, sec)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Total = len(instances)

	// Field list: explicit, or every attribute of the first instance.
	fields := sec.Fields
	if len(fields) == 0 && len(instances) > 0 {
		for _, n := range instances[0].GetAttributeNames() {
			name := n.StringValue()
			if name == "" || strings.Contains(name, "*") {
				continue // internal attrs like mapping*key
			}
			fields = append(fields, name)
		}
		sort.Strings(fields)
	}
	res.Fields = fields
	res.Key = sec.Key
	if res.Key == "" && len(fields) > 0 {
		res.Key = fields[0]
	}

	for _, e := range instances {
		row := map[string]string{"#id": strconv.Itoa(e.GetID())}
		for _, f := range fields {
			row[f] = entityFieldValue(e, f)
		}
		if rowMatches(e, sec.Where) {
			res.Rows = append(res.Rows, row)
		}
	}

	if sec.Sort != "" {
		sortRows(res.Rows, sec.Sort)
	}
	return res
}

// sectionInstances resolves the section's instance source.
func (t *Trace) sectionInstances(sess dtrules.Session, sec ReportSection) ([]dtrules.Entity, error) {
	if sec.Source != "" {
		parts := strings.SplitN(sec.Source, ".", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("source %q must be entity.attribute", sec.Source)
		}
		state := sess.GetState()
		holder, err := state.FindEntity(dtrules.GetRName(parts[1]))
		if err != nil || holder == nil {
			// The attribute may not be on the stack path — search instances
			// of the named entity instead (EL case-insensitive).
			for _, e := range t.InstancesOf(parts[0]) {
				holder = e
				break
			}
		}
		if holder == nil {
			return nil, fmt.Errorf("no %s on the final state", parts[0])
		}
		v, err := holder.Get(dtrules.GetRName(parts[1]))
		if err != nil || v == nil {
			return nil, fmt.Errorf("%s has no attribute %s", parts[0], parts[1])
		}
		arr, err := v.RArrayValue()
		if err != nil {
			return nil, fmt.Errorf("%s is not an array", sec.Source)
		}
		var out []dtrules.Entity
		for i := 0; i < arr.Size(); i++ {
			el, gerr := arr.Get(i)
			if gerr != nil {
				continue
			}
			if e, ok := el.(dtrules.Entity); ok {
				out = append(out, e)
			}
		}
		return out, nil
	}
	if sec.Entity == "" {
		return nil, fmt.Errorf("section needs an entity or a source")
	}
	return t.InstancesOf(sec.Entity), nil
}

// entityFieldValue reads one attribute as display text (EL name matching).
func entityFieldValue(e dtrules.Entity, field string) string {
	for _, n := range e.GetAttributeNames() {
		if strings.EqualFold(n.StringValue(), field) {
			if v, err := e.Get(n); err == nil && v != nil {
				return v.StringValue()
			}
			return ""
		}
	}
	return ""
}

// rowMatches applies every filter (AND semantics).
func rowMatches(e dtrules.Entity, where []ReportFilter) bool {
	for _, f := range where {
		val := entityFieldValue(e, f.Field)
		if !filterMatch(val, f.Op, f.Value) {
			return false
		}
	}
	return true
}

func filterMatch(val, op, target string) bool {
	nv, nerr := strconv.ParseFloat(strings.TrimSpace(val), 64)
	nt, terr := strconv.ParseFloat(strings.TrimSpace(target), 64)
	numeric := nerr == nil && terr == nil
	switch op {
	case "==", "=", "":
		if numeric {
			return nv == nt
		}
		return strings.EqualFold(strings.TrimSpace(val), strings.TrimSpace(target))
	case "!=":
		return !filterMatch(val, "==", target)
	case ">":
		return numeric && nv > nt
	case ">=":
		return numeric && nv >= nt
	case "<":
		return numeric && nv < nt
	case "<=":
		return numeric && nv <= nt
	case "contains":
		return strings.Contains(strings.ToLower(val), strings.ToLower(target))
	}
	return false
}

// sortRows orders rows by a field, numerically when values parse.
func sortRows(rows []map[string]string, field string) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i][field], rows[j][field]
		na, aerr := strconv.ParseFloat(a, 64)
		nb, berr := strconv.ParseFloat(b, 64)
		if aerr == nil && berr == nil {
			return na < nb
		}
		return a < b
	})
}

// ── diffs ────────────────────────────────────────────────────────────

// ReportDiff compares two runs of the SAME spec (baseline vs modified):
// per section, rows added, removed, and field-level changes, aligned by
// the section's key field.
type ReportDiff struct {
	Name     string            `json:"name"`
	Sections []SectionDiffData `json:"sections"`
}

// SectionDiffData is the row-level delta of one section.
type SectionDiffData struct {
	Title   string              `json:"title"`
	Fields  []string            `json:"fields"`
	Added   []map[string]string `json:"added"`
	Removed []map[string]string `json:"removed"`
	Changed []RowChange         `json:"changed"`
}

// RowChange is one row present in both runs with differing fields.
type RowChange struct {
	Key    string            `json:"key"`
	Before map[string]string `json:"before"`
	After  map[string]string `json:"after"`
	Fields []string          `json:"fields"` // which fields differ
}

// DiffReports aligns rows by each section's key field and reports adds,
// removals, and changed fields.
func DiffReports(baseline, modified *Report) *ReportDiff {
	diff := &ReportDiff{Name: modified.Name}
	for i := range modified.Sections {
		ms := modified.Sections[i]
		var bs *ReportSectionResult
		if i < len(baseline.Sections) && baseline.Sections[i].Title == ms.Title {
			bs = &baseline.Sections[i]
		} else {
			for j := range baseline.Sections {
				if baseline.Sections[j].Title == ms.Title {
					bs = &baseline.Sections[j]
					break
				}
			}
		}
		sd := SectionDiffData{Title: ms.Title, Fields: ms.Fields}
		if bs == nil {
			sd.Added = ms.Rows
			diff.Sections = append(diff.Sections, sd)
			continue
		}

		key := ms.Key
		rowKey := func(r map[string]string) string {
			if key != "" && r[key] != "" {
				return r[key]
			}
			return r["#id"]
		}
		before := map[string]map[string]string{}
		for _, r := range bs.Rows {
			before[rowKey(r)] = r
		}
		seen := map[string]bool{}
		for _, r := range ms.Rows {
			k := rowKey(r)
			seen[k] = true
			b, ok := before[k]
			if !ok {
				sd.Added = append(sd.Added, r)
				continue
			}
			var changed []string
			for _, f := range ms.Fields {
				if b[f] != r[f] {
					changed = append(changed, f)
				}
			}
			if len(changed) > 0 {
				sd.Changed = append(sd.Changed, RowChange{Key: k, Before: b, After: r, Fields: changed})
			}
		}
		for _, r := range bs.Rows {
			if !seen[rowKey(r)] {
				sd.Removed = append(sd.Removed, r)
			}
		}
		diff.Sections = append(diff.Sections, sd)
	}
	return diff
}

// ── rendering ────────────────────────────────────────────────────────

// Markdown renders the report as markdown tables (the CLI surface).
func (r *Report) Markdown() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", r.Name)
	for _, s := range r.Sections {
		fmt.Fprintf(&sb, "\n## %s\n\n", s.Title)
		if s.Error != "" {
			fmt.Fprintf(&sb, "_error: %s_\n", s.Error)
			continue
		}
		if len(s.Rows) == 0 {
			fmt.Fprintf(&sb, "_no rows (%d before filters)_\n", s.Total)
			continue
		}
		fmt.Fprintf(&sb, "| %s |\n", strings.Join(s.Fields, " | "))
		fmt.Fprintf(&sb, "|%s\n", strings.Repeat(" --- |", len(s.Fields)))
		for _, row := range s.Rows {
			vals := make([]string, len(s.Fields))
			for i, f := range s.Fields {
				vals[i] = row[f]
			}
			fmt.Fprintf(&sb, "| %s |\n", strings.Join(vals, " | "))
		}
		if len(s.Rows) < s.Total {
			fmt.Fprintf(&sb, "\n_%d of %d rows (filtered)_\n", len(s.Rows), s.Total)
		}
	}
	return sb.String()
}

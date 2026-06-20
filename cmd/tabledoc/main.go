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

// Command tabledoc renders a DTRules project to a self-contained HTML document:
// an overview, the Entity Data Dictionary (the data model), and every decision
// table as a classic condition/action grid. Convert it to PDF with headless
// Chrome (flags must precede the project dir — Go's flag parsing stops at
// the first positional argument):
//
//	go run ./cmd/tabledoc -o tables.html <project-dir>
//	google-chrome --headless --print-to-pdf=tables.pdf tables.html
//
// The HTML is print-styled: section breaks per page, repeated grid headers,
// fit-to-width columns.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func main() {
	out := flag.String("o", "tables.html", "output HTML file")
	title := flag.String("title", "", "document title (default: project dir name)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: tabledoc [-o out.html] [-title \"...\"] <project-dir>")
		fmt.Fprintln(os.Stderr, "  (flags must come before the project dir)")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *out, *title); err != nil {
		fmt.Fprintln(os.Stderr, "tabledoc:", err)
		os.Exit(1)
	}
}

func run(projectDir, outPath, title string) error {
	p, err := authoring.OpenProject(projectDir)
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	if title == "" {
		title = projectName(projectDir)
	}

	// Decision tables, ordered by TABLE_NUMBER then name. p.Tables() can
	// list a name more than once on a cross-file name collision, and
	// p.Table(name) resolves the first match every time — so dedup by name
	// to avoid rendering one table twice and dropping the other silently.
	names := p.Tables()
	seen := make(map[string]bool, len(names))
	tables := make([]*authoring.Table, 0, len(names))
	for _, n := range names {
		if seen[n] {
			fmt.Fprintf(os.Stderr, "tabledoc: warning: duplicate table name %q — rendering once\n", n)
			continue
		}
		seen[n] = true
		if t := p.Table(n); t != nil {
			tables = append(tables, t)
		} else {
			fmt.Fprintf(os.Stderr, "tabledoc: warning: table %q listed but did not resolve\n", n)
		}
	}
	sort.SliceStable(tables, func(i, j int) bool {
		if tables[i].Number != tables[j].Number {
			return tables[i].Number < tables[j].Number
		}
		return tables[i].Name < tables[j].Name
	})
	tviews := make([]tableView, 0, len(tables))
	for _, t := range tables {
		tviews = append(tviews, makeTableView(t))
	}

	// EDD entities (only those with attributes), name-sorted.
	entities := makeEDDViews(p.EDD().Entities())

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return pageTmpl.Execute(f, pageData{
		Title:      title,
		TableCount: len(tviews),
		Entities:   entities,
		Tables:     tviews,
	})
}

func projectName(dir string) string {
	dir = strings.TrimRight(dir, "/")
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[i+1:]
	}
	if dir == "" || dir == "." {
		return "DTRules Project"
	}
	return dir
}

// --- view model -------------------------------------------------------------

type pageData struct {
	Title      string
	TableCount int
	Entities   []entityView
	Tables     []tableView
}

// EDD views ------------------------------------------------------------------

type entityView struct {
	Name  string
	Attrs []attrView
}

type attrView struct {
	Name     string
	Type     string // type, with "of <subtype>" appended for arrays/entity refs
	Default  string
	IO       string // input / output / in·out / —  (from access)
	Comment  string
	Collect  bool
	Question string // one-line summary of how the field is asked
}

func makeEDDViews(entities []*authoring.Entity) []entityView {
	views := make([]entityView, 0, len(entities))
	for _, e := range entities {
		if len(e.Attributes) == 0 {
			continue
		}
		ev := entityView{Name: e.Name}
		for _, a := range e.Attributes {
			ev.Attrs = append(ev.Attrs, attrView{
				Name:     a.Name,
				Type:     typeLabel(a.Type, a.Subtype),
				Default:  strings.TrimSpace(a.Default),
				IO:       ioLabel(a.Access),
				Comment:  strings.TrimSpace(a.Comment),
				Collect:  strings.EqualFold(strings.TrimSpace(a.Collect), "true"),
				Question: questionSummary(a),
			})
		}
		views = append(views, ev)
	}
	sort.SliceStable(views, func(i, j int) bool {
		return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
	})
	return views
}

func typeLabel(t, subtype string) string {
	t = strings.TrimSpace(t)
	subtype = strings.TrimSpace(subtype)
	if subtype != "" {
		return t + " of " + subtype
	}
	return t
}

func ioLabel(access string) string {
	switch strings.ToLower(strings.TrimSpace(access)) {
	case "r":
		return "input"
	case "w":
		return "output"
	case "rw":
		return "in·out"
	default:
		return "—"
	}
}

// questionSummary describes how a collected field is asked in the interview.
func questionSummary(a authoring.Attribute) string {
	if !strings.EqualFold(strings.TrimSpace(a.Collect), "true") {
		return ""
	}
	parts := []string{}
	if qt := strings.TrimSpace(a.QuestionType); qt != "" {
		parts = append(parts, qt)
	}
	if len(a.Options) > 0 {
		labels := make([]string, 0, len(a.Options))
		for _, o := range a.Options {
			l := strings.TrimSpace(o.Label)
			if l == "" {
				l = strings.TrimSpace(o.Value)
			}
			labels = append(labels, l)
		}
		parts = append(parts, "("+strings.Join(labels, ", ")+")")
	}
	// Reference range. Either bound may be empty (one-sided): render those
	// as ≥ / ≤ rather than collapsing to a misleading point value.
	low := strings.TrimSpace(a.QuestionRefLow)
	high := strings.TrimSpace(a.QuestionRefHigh)
	units := strings.TrimSpace(a.QuestionUnits)
	var rng string
	switch {
	case low != "" && high != "":
		rng = "normal " + low + "–" + high
	case low != "":
		rng = "normal ≥ " + low
	case high != "":
		rng = "normal ≤ " + high
	}
	switch {
	case rng != "":
		if units != "" {
			rng += " " + units
		}
		parts = append(parts, rng)
	case units != "":
		// Units only — no range to call "normal".
		parts = append(parts, "in "+units)
	}
	return strings.Join(parts, " ")
}

// Decision-table views -------------------------------------------------------

type tableView struct {
	Number     int
	Name       string
	Policy     string
	Cols       []int
	Contexts   []string
	Initial    []string
	Conditions []rowView
	Actions    []rowView
}

type rowView struct {
	Label   string
	DSL     string
	Comment string
	Cells   []string
}

func makeTableView(t *authoring.Table) tableView {
	n := t.Columns()
	cols := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		cols = append(cols, c)
	}
	tv := tableView{Number: t.Number, Name: t.Name, Policy: policyLabel(t.Policy), Cols: cols}
	for _, c := range t.Contexts {
		if s := strings.TrimSpace(c.DSL); s != "" {
			tv.Contexts = append(tv.Contexts, s)
		}
	}
	for _, ia := range t.InitialActions {
		if s := strings.TrimSpace(ia.DSL); s != "" {
			tv.Initial = append(tv.Initial, s)
		}
	}
	for _, c := range t.Conditions {
		cells := make([]string, 0, n)
		for col := 1; col <= n; col++ {
			v := strings.TrimSpace(c.Columns[col])
			if v == "" {
				v = "-"
			}
			cells = append(cells, v)
		}
		tv.Conditions = append(tv.Conditions, rowView{
			Label: fmt.Sprintf("%d", c.Number), DSL: strings.TrimSpace(c.DSL),
			Comment: strings.TrimSpace(c.Comment), Cells: cells,
		})
	}
	for _, a := range t.Actions {
		cells := make([]string, 0, n)
		for col := 1; col <= n; col++ {
			if a.Columns[col] {
				cells = append(cells, "X")
			} else {
				cells = append(cells, "")
			}
		}
		tv.Actions = append(tv.Actions, rowView{
			Label: fmt.Sprintf("%d", a.Number), DSL: strings.TrimSpace(a.DSL),
			Comment: strings.TrimSpace(a.Comment), Cells: cells,
		})
	}
	return tv
}

func policyLabel(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "BALANCED":
		return "BALANCED"
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}

// --- template ---------------------------------------------------------------

var pageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}} — Rules Reference</title>
<style>
  :root { --ink:#1a1a2e; --muted:#5a5a72; --line:#d4d4e0; --head:#2d2d44; --accent:#6d28d9;
          --teal:#0f766e; --y:#047857; --n:#b91c1c; }
  * { box-sizing: border-box; }
  body { font-family: -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: var(--ink); margin: 0; line-height: 1.45; }
  h1,h2,h3 { line-height: 1.2; }
  code, .mono { font-family: "SF Mono", ui-monospace, "Cascadia Code", Consolas, monospace; }

  .cover { padding: 30mm 18mm 0; }
  .cover h1 { font-size: 30pt; margin: 0 0 6pt; }
  .cover .sub { color: var(--muted); font-size: 13pt; }

  .section { padding: 12mm 14mm; page-break-before: always; }
  .section > h2.shead { font-size: 19pt; margin: 0 0 2pt; }
  .section > .lead { color: var(--muted); font-size: 10.5pt; max-width: 46em; margin: 0 0 12pt; }
  .explainer { font-size: 10pt; }
  .explainer h3 { font-size: 11.5pt; margin: 14pt 0 3pt; color: var(--head); }
  .explainer p { margin: 0 0 7pt; max-width: 46em; }
  .explainer ul { margin: 0 0 7pt; padding-left: 18px; max-width: 46em; }
  .explainer li { margin: 0 0 3pt; }
  .legend { display:flex; gap: 16px; flex-wrap: wrap; font-size: 9pt; color: var(--muted); margin: 4pt 0 0; }
  .chip { font-weight:700; } .chip.y{color:var(--y)} .chip.n{color:var(--n)} .chip.x{color:var(--accent)} .chip.d{color:#b8b8c8}

  /* EDD */
  .entity { margin: 0 0 14pt; break-inside: avoid; }
  .entity h3 { font-size: 12.5pt; margin: 0 0 4pt; }
  .entity h3 .mono { color: var(--teal); }
  table.edd { border-collapse: collapse; width: 100%; font-size: 8.5pt; }
  table.edd th, table.edd td { border: 1px solid var(--line); padding: 4px 6px; vertical-align: top; text-align: left; }
  table.edd thead th { background: var(--head); color: #fff; font-weight: 600; }
  table.edd td.f { font-family: "SF Mono", ui-monospace, Consolas, monospace; white-space: nowrap; }
  table.edd td.ty { color: var(--muted); white-space: nowrap; }
  table.edd td.io { text-transform: lowercase; color: var(--muted); white-space: nowrap; }
  table.edd .ask { color: var(--accent); }
  table.edd .cmt { color: var(--muted); }

  /* decision-table grid */
  .table { margin: 0 0 6mm; break-inside: avoid; }
  .table h3 { font-size: 14pt; margin: 0 0 2pt; }
  .table h3 .num { color: var(--accent); font-variant-numeric: tabular-nums; }
  .policy { display: inline-block; font-size: 8pt; font-weight: 600; letter-spacing: .04em;
            color: #fff; background: var(--head); border-radius: 4px; padding: 2px 7px; margin-bottom: 7pt; }
  .pre { margin: 4pt 0 7pt; padding: 6pt 8pt; background: #f5f5fb; border-left: 3px solid var(--accent);
         font-family: "SF Mono", ui-monospace, Consolas, monospace; font-size: 8.5pt; white-space: pre-wrap; }
  .pre .k { color: var(--muted); font-weight: 600; }
  table.grid { border-collapse: collapse; width: 100%; font-size: 8.5pt; }
  table.grid th, table.grid td { border: 1px solid var(--line); padding: 4px 6px; vertical-align: top; }
  table.grid thead th { background: var(--head); color: #fff; font-weight: 600; }
  table.grid .num { width: 5%; text-align: center; color: var(--muted); font-variant-numeric: tabular-nums; }
  table.grid .expr code { font-family: "SF Mono", ui-monospace, Consolas, monospace; }
  table.grid .expr .cmt { display:block; color: var(--muted); font-size: 7.5pt; margin-top: 1px; }
  table.grid .col { width: 26px; text-align: center; font-weight: 700; font-variant-numeric: tabular-nums; }
  table.grid td.col.y { color: var(--y); } table.grid td.col.n { color: var(--n); }
  table.grid td.col.x { color: var(--accent); } table.grid td.col.dash { color: #b8b8c8; }
  .band th { background: #ece9fb !important; color: var(--head) !important; text-transform: uppercase;
             font-size: 7.5pt; letter-spacing: .06em; }
  @page { size: A4; margin: 12mm; }
</style>
</head>
<body>
  <section class="cover">
    <h1>{{.Title}}</h1>
    <div class="sub">DTRules rules reference — data model &amp; decision tables</div>
  </section>

  <!-- OVERVIEW -->
  <section class="section explainer">
    <h2 class="shead">How this rule set is built</h2>
    <p class="lead">A DTRules application is two artifacts authored in spreadsheets: an <strong>Entity Data
      Dictionary</strong> that defines the data, and a set of <strong>decision tables</strong> that define the
      logic. This document lists both, plus how they drive an interactive interview.</p>

    <h3>The Entity Data Dictionary (EDD) — what it contains</h3>
    <p>The EDD is the data model. It declares the <strong>entities</strong> (things the rules reason about, e.g.
      a patient) and, for each, its <strong>attributes</strong>: a name, a type (string, integer, decimal,
      boolean, date, an entity reference, or an array), an optional default, and whether the field is an
      <em>input</em> the rules read or an <em>output</em> they produce. The decision tables may only reference
      fields declared here — the EDD is the shared vocabulary.</p>
    <p>An attribute can also be marked <strong>collected</strong>: rather than arriving as bulk input, the engine
      asks the user for it. Collected fields carry question metadata — a prompt, a type
      (<span class="mono">multiple_choice</span>, <span class="mono">ascii</span>, <span class="mono">number</span>,
      <span class="mono">date</span>), choice options, and for numbers a lab-style normal range.</p>

    <h3>Decision tables — what they contain</h3>
    <p>Each decision table is a grid. The top rows are <strong>conditions</strong> (when something is true) and the
      bottom rows are <strong>actions</strong> (what to do); every numbered <strong>column is one rule</strong>.
      A condition cell is <span class="chip y">Y</span> (must be true), <span class="chip n">N</span> (must be
      false), or <span class="chip d">–</span> (don't care). An action cell is marked when that action fires for
      the column. Each condition and action is written in DTRules' Expression Language (EL), shown in the grid.</p>
    <ul>
      <li><strong>Policy</strong> — <span class="mono">FIRST</span> fires the first matching column,
        <span class="mono">ALL</span> fires every matching column in order, <span class="mono">BALANCED</span>
        expects exactly one column to match.</li>
      <li><strong>Initial actions</strong> run once before the columns; <strong>contexts</strong> push an entity
        (e.g. "for all active medications") so the table evaluates against each.</li>
      <li>Tables call other tables with <span class="mono">perform</span>, composing larger logic from small,
        readable pieces.</li>
    </ul>

    <h3>The interview UI is automatic</h3>
    <p>There is no hand-built form. When the engine <em>executes the decision tables</em> and a rule reads a
      collected field that hasn't been supplied, it pauses and asks that field's question — on the command line
      or in the browser — then resumes. The screens a user sees are a by-product of which fields the tables
      actually reach: change the tables or the EDD questions and the interview changes with them, with no UI code
      to update.</p>
  </section>

  <!-- EDD -->
  {{if .Entities}}
  <section class="section">
    <h2 class="shead">Entity Data Dictionary</h2>
    <p class="lead">The data model the decision tables read and write. <span class="ask mono">●</span> marks a
      field the interview collects from the user.</p>
    {{range .Entities}}
    <div class="entity">
      <h3><span class="mono">{{.Name}}</span></h3>
      <table class="edd">
        <thead>
          <tr><th>Field</th><th>Type</th><th>Default</th><th>I/O</th><th>Asked as</th><th>Description</th></tr>
        </thead>
        <tbody>
          {{range .Attrs}}
          <tr>
            <td class="f">{{if .Collect}}<span class="ask">●</span> {{end}}{{.Name}}</td>
            <td class="ty">{{.Type}}</td>
            <td class="ty">{{if .Default}}<code>{{.Default}}</code>{{else}}—{{end}}</td>
            <td class="io">{{.IO}}</td>
            <td class="ask">{{if .Question}}{{.Question}}{{else}}—{{end}}</td>
            <td class="cmt">{{.Comment}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>
    {{end}}
  </section>
  {{end}}

  <!-- DECISION TABLES -->
  <section class="section">
    <h2 class="shead">Decision Tables</h2>
    <p class="lead">{{.TableCount}} tables. Conditions (top) decide which rule columns apply; actions (bottom) run
      for the marked columns.</p>
    <div class="legend">
      <span><span class="chip y">Y</span> condition must be true</span>
      <span><span class="chip n">N</span> must be false</span>
      <span><span class="chip d">–</span> don't care</span>
      <span><span class="chip x">✓</span> action fires</span>
    </div>

    {{range .Tables}}
    <div class="table">
      <h3><span class="num">{{if .Number}}{{.Number}} · {{end}}</span>{{.Name}}</h3>
      <div class="policy">{{.Policy}}</div>
      {{if .Contexts}}<div class="pre"><span class="k">contexts:</span>
{{range .Contexts}}  {{.}}
{{end}}</div>{{end}}
      {{if .Initial}}<div class="pre"><span class="k">initial actions:</span>
{{range .Initial}}  {{.}}
{{end}}</div>{{end}}
      <table class="grid">
        <thead>
          <tr><th class="num">#</th><th class="expr">Condition</th>{{range .Cols}}<th class="col">{{.}}</th>{{end}}</tr>
        </thead>
        <tbody>
          {{range .Conditions}}
          <tr>
            <td class="num">{{.Label}}</td>
            <td class="expr"><code>{{.DSL}}</code>{{if .Comment}}<span class="cmt">{{.Comment}}</span>{{end}}</td>
            {{range .Cells}}<td class="col {{if eq . "Y"}}y{{else if eq . "N"}}n{{else}}dash{{end}}">{{if eq . "-"}}–{{else}}{{.}}{{end}}</td>{{end}}
          </tr>
          {{end}}
          <tr class="band"><th class="num"></th><th class="expr">Action</th>{{range .Cols}}<th class="col">{{.}}</th>{{end}}</tr>
          {{range .Actions}}
          <tr>
            <td class="num">{{.Label}}</td>
            <td class="expr"><code>{{.DSL}}</code>{{if .Comment}}<span class="cmt">{{.Comment}}</span>{{end}}</td>
            {{range .Cells}}<td class="col {{if eq . "X"}}x{{else}}dash{{end}}">{{if eq . "X"}}✓{{end}}</td>{{end}}
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>
    {{end}}
  </section>
</body>
</html>
`))

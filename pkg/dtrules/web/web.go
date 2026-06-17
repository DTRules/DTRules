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

// Package web serves a DTRules rule set as an interactive web interview
// (#850/#855). It drives one execution per browser session in its own
// goroutine; the read-point collector blocks on a channel, and each HTTP form
// POST supplies the next answer and resumes execution. When execution
// finishes, the result is rendered. The goroutine *is* the continuation — no
// re-execution.
package web

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

// Result is the rendered outcome of one run.
type Result struct {
	Title    string
	Fields   []Field           // scalar outputs
	Lists    []List            // array outputs (warnings, rationale, ...)
	Readings []collect.Reading // collected range-bearing values, lab-report style
	DataXML  string            // canonical data collected this run (for review/edit)
	Error    string
}

// Field is one scalar output value.
type Field struct{ Name, Value string }

// List is one array output.
type List struct {
	Name  string
	Items []string
}

// RunFunc executes one complete interview: it creates a fresh session, runs
// the entry table with the given asker (which blocks for each collect field),
// and returns the rendered result. reviewData, when non-empty, is canonical
// data XML from a prior run loaded in Review mode — so every collect field is
// re-asked pre-filled with the prior answer (the review/modify loop, #853).
type RunFunc func(asker collect.Asker, reviewData string) (*Result, error)

type answer struct {
	value string
	ok    bool
}

// interview is one in-flight execution goroutine.
type interview struct {
	reqCh    chan collect.Request
	ansCh    chan answer
	doneCh   chan *Result
	finished *Result
}

func startInterview(run RunFunc, reviewData string) *interview {
	iv := &interview{
		reqCh:  make(chan collect.Request),
		ansCh:  make(chan answer),
		doneCh: make(chan *Result, 1),
	}
	asker := collect.AskerFunc(func(req collect.Request) (dtrules.Object, bool, error) {
		iv.reqCh <- req
		a := <-iv.ansCh
		if !a.ok {
			return nil, false, nil
		}
		return dtrules.GetRString(a.value), true, nil
	})
	go func() {
		res, err := run(asker, reviewData)
		if err != nil {
			res = &Result{Error: err.Error()}
		}
		if res == nil {
			res = &Result{}
		}
		iv.doneCh <- res
	}()
	return iv
}

// next blocks for the next question, or returns the final result when the run
// has finished.
func (iv *interview) next() (*collect.Request, *Result) {
	if iv.finished != nil {
		return nil, iv.finished
	}
	select {
	case req := <-iv.reqCh:
		return &req, nil
	case res := <-iv.doneCh:
		iv.finished = res
		return nil, res
	}
}

func (iv *interview) answer(value string, ok bool) { iv.ansCh <- answer{value, ok} }

// qa is one answered question, shown as a stacked one-liner in the transcript.
type qa struct {
	Prompt  string
	Display string // the answer as shown
	Entity  string
	Field   string
	Raw     string // the data value, for rebuilding canonical data on review
	Units   string
	Flag    string // "", "H", "L"
	Ref     string // reference range text
}

// uiSession is the per-browser state: the running interview, the question
// awaiting an answer, and the transcript of answers so far.
type uiSession struct {
	iv       *interview
	pending  *collect.Request
	answered []qa
}

// Server is an http.Handler that runs one interview per browser session.
type Server struct {
	run      RunFunc
	title    string // page/tab title (the project name)
	mu       sync.Mutex
	sessions map[string]*uiSession
	seq      int
}

// NewServer returns a Server that starts a fresh interview (via run) for each
// new browser session. title is shown as the browser tab title; "" falls back
// to "DTRules".
func NewServer(run RunFunc, title string) *Server {
	if title == "" {
		title = "DTRules"
	}
	return &Server{run: run, title: title, sessions: map[string]*uiSession{}}
}

// faviconSVG is a small brand mark: a rounded square with a checkmark,
// representing a decided rule outcome. Served at /favicon.svg.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">` +
	`<rect width="64" height="64" rx="13" fill="#0a7d5a"/>` +
	`<path d="M17 33l9 10 21-23" fill="none" stroke="#fff" stroke-width="7" stroke-linecap="round" stroke-linejoin="round"/>` +
	`</svg>`

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/favicon.svg":
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "max-age=86400")
		_, _ = w.Write([]byte(faviconSVG))
		return
	case "/favicon.ico":
		w.WriteHeader(http.StatusNoContent) // browsers fall back to the SVG link
		return
	case "/", "/answer", "/review":
		// handled below
	default:
		http.NotFound(w, r)
		return
	}

	// /review restarts the interview pre-filled with the answers given so far
	// (Review load mode) so the user can change any value and re-run.
	if r.URL.Path == "/review" {
		reviewData := ""
		if cur := s.lookup(r); cur != nil {
			reviewData = buildCanonical(cur.answered)
		}
		us := s.newSession(w, reviewData)
		s.render(w, us)
		return
	}

	us := s.session(w, r)
	if r.Method == http.MethodPost && us.pending != nil {
		_ = r.ParseForm()
		val := r.FormValue("answer")
		// Record the answer for the transcript, then resume execution.
		// An empty submission keeps the default (ok=false).
		us.answered = append(us.answered, qaFrom(us.pending, val))
		us.pending = nil
		us.iv.answer(val, val != "")
	}
	s.render(w, us)
}

// render advances the interview one step: the next question (with the running
// transcript), or the final result.
func (s *Server) render(w http.ResponseWriter, us *uiSession) {
	req, res := us.iv.next()
	if res != nil {
		us.pending = nil
		s.writeResult(w, us.answered, res, len(us.answered) > 0)
		return
	}
	us.pending = req
	s.writeQuestion(w, us.answered, req)
}

// lookup returns the uiSession for the request's cookie, or nil.
func (s *Server) lookup(r *http.Request) *uiSession {
	c, err := r.Cookie("dtrsid")
	if err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[c.Value]
}

// session returns the existing session for a POST cookie, or a fresh one.
func (s *Server) session(w http.ResponseWriter, r *http.Request) *uiSession {
	if r.Method == http.MethodPost {
		if us := s.lookup(r); us != nil {
			return us
		}
	}
	return s.newSession(w, "")
}

// newSession allocates a session id, starts an interview (optionally seeded
// with review data), and sets the session cookie.
func (s *Server) newSession(w http.ResponseWriter, reviewData string) *uiSession {
	s.mu.Lock()
	s.seq++
	sid := strconv.Itoa(s.seq)
	us := &uiSession{iv: startInterview(s.run, reviewData)}
	s.sessions[sid] = us
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "dtrsid", Value: sid, Path: "/"})
	return us
}

// qaFrom builds a transcript entry pairing a question with its answer.
func qaFrom(req *collect.Request, val string) qa {
	cur := ""
	if req.Current != nil {
		cur = req.Current.StringValue()
	}
	raw := val
	if raw == "" { // empty submission keeps the default
		raw = cur
	}
	display := raw
	if req.QType == "multiple_choice" {
		for _, o := range req.Options {
			if o.Value == raw {
				display = o.Label
				if display == "" {
					display = o.Value
				}
				break
			}
		}
	}
	q := qa{Prompt: promptOf(req), Display: display, Entity: req.Entity, Field: req.Field, Raw: raw, Units: req.Units}
	if req.QType == "number" {
		q.Flag = collect.Flag(raw, req.RefLow, req.RefHigh)
		q.Ref = collect.RangeText(req.RefLow, req.RefHigh, "")
	}
	return q
}

func promptOf(req *collect.Request) string {
	if req.Text != "" {
		return req.Text
	}
	return req.Entity + "." + req.Field
}

// buildCanonical rebuilds canonical data XML from a transcript, grouped by
// entity — the seed for a Review-mode re-run.
func buildCanonical(answered []qa) string {
	if len(answered) == 0 {
		return ""
	}
	var order []string
	byEntity := map[string][]qa{}
	for _, a := range answered {
		if _, ok := byEntity[a.Entity]; !ok {
			order = append(order, a.Entity)
		}
		byEntity[a.Entity] = append(byEntity[a.Entity], a)
	}
	var b strings.Builder
	b.WriteString("<dtrules-data>")
	for _, e := range order {
		b.WriteString("<" + e + ">")
		for _, a := range byEntity[e] {
			b.WriteString("<" + a.Field + ">" + html.EscapeString(a.Raw) + "</" + a.Field + ">")
		}
		b.WriteString("</" + e + ">")
	}
	b.WriteString("</dtrules-data>")
	return b.String()
}

// transcript renders the answered questions as stacked one-liners.
func transcript(answered []qa) string {
	if len(answered) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="log">`)
	for _, a := range answered {
		badge := ""
		switch a.Flag {
		case "H":
			badge = ` <span class="flag hi">H</span>`
		case "L":
			badge = ` <span class="flag lo">L</span>`
		}
		unit := ""
		if a.Units != "" {
			unit = " " + html.EscapeString(a.Units)
		}
		b.WriteString(fmt.Sprintf(
			`<div class="qa"><span class="chk">&#10003;</span><span class="q">%s</span><span class="a">%s%s%s</span></div>`,
			html.EscapeString(a.Prompt), html.EscapeString(a.Display), unit, badge))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// reviewButton renders the "Review / edit answers" action, shown once at least
// one answer exists.
func reviewButton(answered []qa) string {
	if len(answered) == 0 {
		return ""
	}
	return `<form method="post" action="/review" class="inline">` +
		`<button type="submit" class="ghost">Review / edit answers</button></form>`
}

func (s *Server) writeQuestion(w http.ResponseWriter, answered []qa, req *collect.Request) {
	cur := ""
	if req.Current != nil {
		cur = req.Current.StringValue()
	}
	var control string
	switch req.QType {
	case "multiple_choice":
		var b strings.Builder
		for _, o := range req.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			sel := ""
			if o.Value == cur {
				sel = " selected"
			}
			b.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, html.EscapeString(o.Value), sel, html.EscapeString(label)))
		}
		control = fmt.Sprintf(`<select name="answer" autofocus>%s</select>`, b.String())
	case "number":
		// Lab convention: show the reference range as guidance, but do NOT set
		// min/max on the input — out-of-range values must be accepted.
		hint := ""
		if rng := collect.RangeText(req.RefLow, req.RefHigh, req.Units); rng != "" {
			hint = fmt.Sprintf(`<div class="hint">normal range: %s</div>`, html.EscapeString(rng))
		}
		unit := ""
		if req.Units != "" {
			unit = `<span class="unit">` + html.EscapeString(req.Units) + `</span>`
		}
		control = fmt.Sprintf(`%s<input name="answer" type="number" step="any" placeholder="%s" autofocus>%s`,
			hint, html.EscapeString(cur), unit)
	case "date":
		control = `<input name="answer" type="date" autofocus>`
	default:
		control = fmt.Sprintf(`<input name="answer" type="text" placeholder="%s" autofocus>`, html.EscapeString(cur))
	}
	card := fmt.Sprintf(`<div class="card"><h2>%s</h2>
<form method="post" action="/answer">%s
<div class="actions"><button type="submit" class="primary">Next &rarr;</button>
<button type="submit" name="answer" value="" class="ghost">Use default (%s)</button></div></form></div>`,
		html.EscapeString(promptOf(req)), control, html.EscapeString(cur))
	s.page(w, transcript(answered)+card+`<div class="actions">`+reviewButton(answered)+`</div>`)
}

func (s *Server) writeResult(w http.ResponseWriter, answered []qa, res *Result, canReview bool) {
	if res.Error != "" {
		s.page(w, fmt.Sprintf(`<div class="card"><h2>Error</h2><pre>%s</pre></div><div class="actions"><a class="ghost" href="/">Start over</a></div>`, html.EscapeString(res.Error)))
		return
	}
	var b strings.Builder
	b.WriteString(transcript(answered))
	b.WriteString(`<div class="card">`)
	if len(res.Readings) > 0 {
		b.WriteString(`<h2>Collected values</h2><table>`)
		for _, r := range res.Readings {
			badge, cls := "", ""
			switch r.Flag {
			case "H":
				badge, cls = ` <span class="flag hi">H</span>`, ` class="hi"`
			case "L":
				badge, cls = ` <span class="flag lo">L</span>`, ` class="lo"`
			}
			ref := collect.RangeText(r.Low, r.High, "")
			b.WriteString(fmt.Sprintf(`<tr><th>%s</th><td%s>%s %s%s</td><td class="ref">ref %s</td></tr>`,
				html.EscapeString(r.Field), cls, html.EscapeString(r.Value), html.EscapeString(r.Units), badge, html.EscapeString(ref)))
		}
		b.WriteString(`</table>`)
	}
	b.WriteString(`<h2>Result</h2><table>`)
	for _, f := range res.Fields {
		b.WriteString(fmt.Sprintf(`<tr><th>%s</th><td>%s</td></tr>`, html.EscapeString(f.Name), html.EscapeString(f.Value)))
	}
	b.WriteString(`</table>`)
	for _, l := range res.Lists {
		b.WriteString(fmt.Sprintf(`<h3>%s</h3><ul>`, html.EscapeString(l.Name)))
		for _, it := range l.Items {
			b.WriteString("<li>" + html.EscapeString(it) + "</li>")
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</div><div class="actions">`)
	if canReview {
		b.WriteString(reviewButton(answered))
	}
	b.WriteString(`<a class="ghost" href="/">Start over</a></div>`)
	s.page(w, b.String())
}

func (s *Server) page(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := html.EscapeString(s.title)
	fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title><link rel="icon" type="image/svg+xml" href="/favicon.svg">
<style>%s</style></head><body>
<header><span class="mark">%s</span><h1>%s</h1></header>
<main>%s</main></body></html>`, title, pageCSS, faviconSVG, title, body)
}

const pageCSS = `
:root{--brand:#0a7d5a;--brand-d:#086b4d;--ink:#1f2933;--muted:#6b7785;--line:#e3e8ee;--bg:#eef2f6}
*{box-sizing:border-box}
body{font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:var(--ink);
background:linear-gradient(180deg,#f6f9fb,var(--bg));margin:0;min-height:100vh;padding:0 1rem 3rem}
header{display:flex;align-items:center;gap:.6rem;max-width:560px;margin:0 auto;padding:1.4rem 0 .6rem}
header .mark svg{width:30px;height:30px;display:block}
h1{font-size:1.2rem;margin:0;color:var(--brand)}
main{max-width:560px;margin:0 auto}
.card{background:#fff;border:1px solid var(--line);border-radius:12px;padding:1.1rem 1.2rem;
box-shadow:0 1px 3px rgba(20,40,60,.06);margin:.8rem 0}
.card h2{font-size:1.15rem;margin:.1rem 0 .8rem}
.log{margin:.4rem 0}
.qa{display:flex;align-items:baseline;gap:.5rem;padding:.4rem .7rem;border-left:3px solid var(--brand);
background:#fff;border-radius:7px;margin:.35rem 0;box-shadow:0 1px 2px rgba(20,40,60,.05)}
.qa .chk{color:var(--brand);font-weight:700}
.qa .q{color:var(--muted);flex:1}
.qa .a{font-weight:600}
.flag{font-weight:700;border-radius:5px;padding:0 .35rem;margin-left:.25rem;font-size:.8rem}
.flag.hi{background:#fdecec;color:#b00}
.flag.lo{background:#e9f1fc;color:#06c}
td.hi{color:#b00}td.lo{color:#06c}
input,select{font-size:1rem;padding:.5rem .6rem;border:1px solid #c7d0da;border-radius:8px;min-width:14rem}
input:focus,select:focus{outline:none;border-color:var(--brand);box-shadow:0 0 0 3px rgba(10,125,90,.15)}
.unit{margin-left:.5rem;color:var(--muted)}
.hint{color:var(--muted);font-size:.9rem;margin:0 0 .5rem}
.actions{display:flex;gap:.5rem;align-items:center;flex-wrap:wrap;margin-top:.9rem}
button,a.ghost{font-size:.95rem;padding:.5rem .9rem;border-radius:8px;cursor:pointer;text-decoration:none;border:1px solid transparent}
button.primary{background:var(--brand);color:#fff;border-color:var(--brand)}
button.primary:hover{background:var(--brand-d)}
button.ghost,a.ghost{background:#fff;color:var(--brand);border-color:#cfe0d8}
button.ghost:hover,a.ghost:hover{background:#f1f8f5}
.inline{display:inline}
table{border-collapse:collapse;width:100%%}
th{text-align:left;padding:.3rem 1rem .3rem 0;color:var(--muted);font-weight:600;vertical-align:top}
td{padding:.3rem 0}
td.ref{color:var(--muted);padding-left:1rem}
h3{margin:.9rem 0 .3rem;font-size:1rem}
ul{margin:.2rem 0 .6rem;padding-left:1.1rem}
pre{white-space:pre-wrap;color:#b00}
`

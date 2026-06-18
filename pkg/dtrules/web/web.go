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
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/interview"
)

// errAborted unwinds a parked interview goroutine when its session is reaped.
var errAborted = errors.New("interview session abandoned")

const (
	sessionTTL  = 30 * time.Minute // idle sessions are reaped after this
	reapEvery   = 5 * time.Minute
	maxUpload   = 4 << 20 // 4 MB cap on an uploaded session file
	maxSessions = 5000     // hard cap; oldest idle sessions evicted past this
)

// Result, Field, List, and RunFunc are re-exported from the interview package
// — the UI-neutral contract this server renders. A custom UI or engine can use
// interview directly without importing the HTTP server.
type (
	Result  = interview.Result
	Field   = interview.Field
	List    = interview.List
	RunFunc = interview.RunnerFunc // convenience adapter for an inline Runner
)

type answer struct {
	value string
	ok    bool
}

// runJob is one in-flight execution goroutine driving a Runner.
type runJob struct {
	reqCh     chan collect.Request
	ansCh     chan answer
	doneCh    chan *Result
	cancel    chan struct{} // closed by abort() to unwind a parked goroutine
	cancelOne sync.Once
	finished  *Result
}

// abort signals a parked interview goroutine to unwind (idempotent).
func (j *runJob) abort() { j.cancelOne.Do(func() { close(j.cancel) }) }

func startJob(run interview.Runner, reviewData string) *runJob {
	j := &runJob{
		reqCh:  make(chan collect.Request),
		ansCh:  make(chan answer),
		doneCh: make(chan *Result, 1),
		cancel: make(chan struct{}),
	}
	// The asker blocks for an answer, but unwinds if the session is reaped
	// (browser gone), so a goroutine never parks forever (H2).
	asker := collect.AskerFunc(func(req collect.Request) (dtrules.Object, bool, error) {
		select {
		case j.reqCh <- req:
		case <-j.cancel:
			return nil, false, errAborted
		}
		select {
		case a := <-j.ansCh:
			if !a.ok {
				return nil, false, nil
			}
			return dtrules.GetRString(a.value), true, nil
		case <-j.cancel:
			return nil, false, errAborted
		}
	})
	go func() {
		res, err := run.Run(asker, reviewData)
		if errors.Is(err, errAborted) {
			return // reaped — drop silently
		}
		if err != nil {
			res = &Result{Error: err.Error()}
		}
		if res == nil {
			res = &Result{}
		}
		j.doneCh <- res
	}()
	return j
}

// next blocks for the next question, or returns the final result when the run
// has finished.
func (j *runJob) next() (*collect.Request, *Result) {
	if j.finished != nil {
		return nil, j.finished
	}
	select {
	case req := <-j.reqCh:
		return &req, nil
	case res := <-j.doneCh:
		j.finished = res
		return nil, res
	}
}

func (j *runJob) answer(value string, ok bool) { j.ansCh <- answer{value, ok} }

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
// awaiting an answer, the transcript of answers so far, and the last completed
// run's full canonical data (inputs + results) for download.
type uiSession struct {
	mu       sync.Mutex // serializes concurrent requests for the same session (H3)
	iv       *runJob
	pending  *collect.Request
	answered []qa
	dataXML  string
	lastSeen time.Time
}

// Server is an http.Handler that drives one interview per browser session
// against a Runner (any decision-table engine).
type Server struct {
	run      interview.Runner
	title    string // page/tab title (the project name)
	mu       sync.Mutex
	sessions map[string]*uiSession
	seq      int
}

// NewServer returns a Server that starts a fresh interview (via run) for each
// new browser session. run is any interview.Runner — the default engine
// (interview.Project) or a custom one. title is shown as the browser tab
// title; "" falls back to "DTRules".
func NewServer(run interview.Runner, title string) *Server {
	if title == "" {
		title = "DTRules"
	}
	s := &Server{run: run, title: title, sessions: map[string]*uiSession{}}
	go s.reap()
	return s
}

// reap periodically drops idle sessions, unwinding their goroutines (H1/H2).
// It runs for the Server's lifetime (one goroutine, not per-session).
func (s *Server) reap() {
	t := time.NewTicker(reapEvery)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-sessionTTL)
		s.mu.Lock()
		for sid, us := range s.sessions {
			us.mu.Lock()
			idle := us.lastSeen.Before(cutoff)
			iv := us.iv
			us.mu.Unlock()
			if idle {
				if iv != nil {
					iv.abort()
				}
				delete(s.sessions, sid)
			}
		}
		s.mu.Unlock()
	}
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
	case "/data":
		s.serveData(w, r)
		return
	case "/upload":
		s.handleUpload(w, r)
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
			cur.mu.Lock()
			reviewData = buildCanonical(cur.answered)
			cur.mu.Unlock()
		}
		us := s.newSession(w, reviewData)
		us.mu.Lock()
		defer us.mu.Unlock()
		s.render(w, us)
		return
	}

	us := s.session(w, r)
	us.mu.Lock() // serialize concurrent requests for the same session (H3)
	defer us.mu.Unlock()
	us.lastSeen = time.Now()
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
		us.dataXML = res.DataXML
		s.writeResult(w, us.answered, res, len(us.answered) > 0, res.DataXML != "")
		return
	}
	us.pending = req
	s.writeQuestion(w, us.answered, req)
}

// serveData streams the current session's collected data (inputs + results) as
// a downloadable canonical XML file.
func (s *Server) serveData(w http.ResponseWriter, r *http.Request) {
	us := s.lookup(r)
	if us == nil {
		http.NotFound(w, r)
		return
	}
	us.mu.Lock()
	data := us.dataXML
	us.lastSeen = time.Now()
	us.mu.Unlock()
	if data == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, s.downloadName()))
	_, _ = io.WriteString(w, data)
}

// downloadName is the default filename offered when saving a session.
func (s *Server) downloadName() string {
	name := strings.ToLower(strings.ReplaceAll(s.title, " ", "-"))
	if name == "" {
		name = "dtrules"
	}
	return name + "-session.xml"
}

// handleUpload accepts a previously downloaded data file and starts a fresh
// interview pre-filled from it (Review mode), so the user can review and modify.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload) // bound form parsing (L1)
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll() // clean any temp files
		}
	}()
	f, _, err := r.FormFile("file")
	if err != nil {
		s.page(w, `<div class="card"><h2>No file</h2><p>Choose a saved session file to review.</p></div>`+
			`<div class="actions"><a class="ghost" href="/">Back</a></div>`)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxUpload))
	if err != nil {
		http.Error(w, "could not read upload", http.StatusBadRequest)
		return
	}
	us := s.newSession(w, string(data))
	us.mu.Lock()
	defer us.mu.Unlock()
	s.render(w, us)
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
	us := &uiSession{iv: startJob(s.run, reviewData), lastSeen: time.Now()}
	// Hard cap: if a burst outpaces the reaper, evict the oldest sessions
	// (and unwind their goroutines) so the map can't grow without bound (H1).
	if len(s.sessions) >= maxSessions {
		s.evictOldestLocked(len(s.sessions) - maxSessions + 1)
	}
	s.sessions[sid] = us
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "dtrsid", Value: sid, Path: "/"})
	return us
}

// evictOldestLocked removes the n least-recently-seen sessions. Caller holds s.mu.
func (s *Server) evictOldestLocked(n int) {
	type aged struct {
		sid  string
		seen time.Time
	}
	all := make([]aged, 0, len(s.sessions))
	for sid, us := range s.sessions {
		us.mu.Lock()
		all = append(all, aged{sid, us.lastSeen})
		us.mu.Unlock()
	}
	sort.Slice(all, func(i, j int) bool { return all[i].seen.Before(all[j].seen) })
	for i := 0; i < n && i < len(all); i++ {
		if us := s.sessions[all[i].sid]; us != nil && us.iv != nil {
			us.iv.abort()
		}
		delete(s.sessions, all[i].sid)
	}
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
		// Entity/field become XML tags — only safe identifiers may, so an
		// unusual rule-set name can't inject markup that's re-parsed on
		// review or reflected into HTML (H4). Values are always escaped.
		if !safeTag(a.Entity) || !safeTag(a.Field) {
			continue
		}
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

// safeTag reports whether name is a safe XML element name (a conservative
// identifier subset: letter/underscore start, then letters/digits/_/-/. ).
func safeTag(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case i > 0 && (r >= '0' && r <= '9' || r == '-' || r == '.'):
		default:
			return false
		}
	}
	return true
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

// uploadForm renders a single button that opens a file dialog and submits as
// soon as a file is chosen — so "upload" is one click, then pick a file.
func uploadForm() string {
	return `<form method="post" action="/upload" enctype="multipart/form-data" class="inline">` +
		`<label class="ghost filebtn" title="Open a saved session to review and modify">` +
		`Upload session` +
		`<input type="file" name="file" accept=".xml,application/xml" onchange="this.form.submit()" hidden></label>` +
		`<noscript> <button type="submit" class="ghost">Upload</button></noscript>` +
		`</form>`
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
	footer := `<div class="actions">` + reviewButton(answered) + `</div>`
	if len(answered) == 0 {
		footer += uploadForm() // offer to resume a saved session before any answers
	}
	s.page(w, transcript(answered)+card+footer)
}

func (s *Server) writeResult(w http.ResponseWriter, answered []qa, res *Result, canReview, canDownload bool) {
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
	b.WriteString(`</div>`)
	// Row 1: session file actions — upload and download together.
	b.WriteString(`<div class="actions">`)
	b.WriteString(uploadForm())
	if canDownload {
		b.WriteString(fmt.Sprintf(`<a class="ghost" href="/data" download="%s" onclick="return dlSession(event,this)">Download session</a>`, s.downloadName()))
	}
	b.WriteString(`</div>`)
	// Row 2: navigation — review and start over.
	b.WriteString(`<div class="actions">`)
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
<main>%s</main><script>%s</script></body></html>`, title, pageCSS, faviconSVG, title, body, pageJS)
}

// pageJS intercepts the Download link to offer a Save dialog where the user can
// rename the file. Chromium browsers get a native save picker (location + name)
// via showSaveFilePicker; others fall back to a rename prompt; with no JS the
// link is a plain download. No '%' or backticks (embedded in a raw template).
const pageJS = `
function dlSession(ev,a){
 if(!window.fetch){return true;}
 ev.preventDefault();
 fetch(a.href).then(function(r){return r.blob();}).then(function(blob){
  var name=a.getAttribute('download')||'session.xml';
  if(window.showSaveFilePicker){
   window.showSaveFilePicker({suggestedName:name,types:[{description:'DTRules session',accept:{'application/xml':['.xml']}}]})
    .then(function(h){return h.createWritable();})
    .then(function(w){return w.write(blob).then(function(){return w.close();});})
    .catch(function(e){if(e&&e.name==='AbortError'){return;}dlFallback(blob,name);});
  }else{dlFallback(blob,name);}
 });
 return false;
}
function dlFallback(blob,suggested){
 var name=window.prompt('Save session as:',suggested);
 if(name===null){return;}
 if(!name){name=suggested;}
 var u=URL.createObjectURL(blob);
 var el=document.createElement('a');el.href=u;el.download=name;
 document.body.appendChild(el);el.click();document.body.removeChild(el);URL.revokeObjectURL(u);
}
`

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
button,a.ghost,label.ghost{font-size:.95rem;padding:.5rem .9rem;border-radius:8px;cursor:pointer;
text-decoration:none;border:1px solid transparent;display:inline-block;line-height:1.2}
button.primary{background:var(--brand);color:#fff;border-color:var(--brand)}
button.primary:hover{background:var(--brand-d)}
button.ghost,a.ghost,label.ghost{background:#fff;color:var(--brand);border-color:#cfe0d8}
button.ghost:hover,a.ghost:hover,label.ghost:hover{background:#f1f8f5}
.inline{display:inline}
table{border-collapse:collapse;width:100%%}
th{text-align:left;padding:.3rem 1rem .3rem 0;color:var(--muted);font-weight:600;vertical-align:top}
td{padding:.3rem 0}
td.ref{color:var(--muted);padding-left:1rem}
h3{margin:.9rem 0 .3rem;font-size:1rem}
ul{margin:.2rem 0 .6rem;padding-left:1.1rem}
pre{white-space:pre-wrap;color:#b00}
`

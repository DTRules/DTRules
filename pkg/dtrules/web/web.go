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
// and returns the rendered result.
type RunFunc func(asker collect.Asker) (*Result, error)

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

func startInterview(run RunFunc) *interview {
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
		res, err := run(asker)
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

// Server is an http.Handler that runs one interview per browser session.
type Server struct {
	run   RunFunc
	title string // page/tab title (the project name)
	mu    sync.Mutex
	ivs   map[string]*interview
	seq   int
}

// NewServer returns a Server that starts a fresh interview (via run) for each
// new browser session. title is shown as the browser tab title; "" falls back
// to "DTRules".
func NewServer(run RunFunc, title string) *Server {
	if title == "" {
		title = "DTRules"
	}
	return &Server{run: run, title: title, ivs: map[string]*interview{}}
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
	case "/", "/answer":
		// handled below
	default:
		http.NotFound(w, r)
		return
	}
	iv, sid := s.session(w, r)

	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		val := r.FormValue("answer")
		// An empty submission keeps the default (ok=false).
		iv.answer(val, val != "")
	}

	req, res := iv.next()
	if res != nil {
		s.clear(sid)
		s.writeResult(w, res)
		return
	}
	s.writeQuestion(w, req)
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) (*interview, string) {
	if r.Method == http.MethodPost {
		if c, err := r.Cookie("dtrsid"); err == nil {
			s.mu.Lock()
			iv := s.ivs[c.Value]
			s.mu.Unlock()
			if iv != nil {
				return iv, c.Value
			}
		}
	}
	// New session.
	s.mu.Lock()
	s.seq++
	sid := strconv.Itoa(s.seq)
	iv := startInterview(s.run)
	s.ivs[sid] = iv
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "dtrsid", Value: sid, Path: "/"})
	return iv, sid
}

func (s *Server) clear(sid string) {
	s.mu.Lock()
	delete(s.ivs, sid)
	s.mu.Unlock()
}

func (s *Server) writeQuestion(w http.ResponseWriter, req *collect.Request) {
	prompt := req.Text
	if prompt == "" {
		prompt = req.Entity + "." + req.Field
	}
	cur := ""
	if req.Current != nil {
		cur = req.Current.StringValue()
	}
	var control string
	switch req.QType {
	case "multiple_choice":
		var b string
		for _, o := range req.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			sel := ""
			if o.Value == cur {
				sel = " selected"
			}
			b += fmt.Sprintf(`<option value="%s"%s>%s</option>`, html.EscapeString(o.Value), sel, html.EscapeString(label))
		}
		control = fmt.Sprintf(`<select name="answer">%s</select>`, b)
	case "number":
		// Lab convention: show the reference range as guidance, but do NOT set
		// min/max on the input — out-of-range values must be accepted.
		hint := ""
		if rng := collect.RangeText(req.RefLow, req.RefHigh, req.Units); rng != "" {
			hint = fmt.Sprintf(`<p style="color:#555">normal range: %s</p>`, html.EscapeString(rng))
		}
		unit := ""
		if req.Units != "" {
			unit = " " + html.EscapeString(req.Units)
		}
		control = fmt.Sprintf(`%s<input name="answer" type="number" step="any" placeholder="%s">%s`,
			hint, html.EscapeString(cur), unit)
	case "date":
		control = `<input name="answer" type="date">`
	default:
		control = fmt.Sprintf(`<input name="answer" type="text" placeholder="%s">`, html.EscapeString(cur))
	}
	s.page(w, fmt.Sprintf(`<h2>%s</h2><form method="post" action="/answer">%s
<p><button type="submit">Next</button>
<button type="submit" name="answer" value="">Use default (%s)</button></p></form>`,
		html.EscapeString(prompt), control, html.EscapeString(cur)))
}

func (s *Server) writeResult(w http.ResponseWriter, res *Result) {
	if res.Error != "" {
		s.page(w, fmt.Sprintf(`<h2>Error</h2><pre>%s</pre><p><a href="/">Start over</a></p>`, html.EscapeString(res.Error)))
		return
	}
	body := ""
	if len(res.Readings) > 0 {
		body += "<h2>Collected values</h2><table>"
		for _, r := range res.Readings {
			flag, color := "", "#111"
			switch r.Flag {
			case "H":
				flag, color = " <b>H</b>", "#b00"
			case "L":
				flag, color = " <b>L</b>", "#06c"
			}
			ref := collect.RangeText(r.Low, r.High, "")
			body += fmt.Sprintf(`<tr><th align=left>%s</th><td style="color:%s">%s %s%s</td><td style="color:#777">ref %s</td></tr>`,
				html.EscapeString(r.Field), color, html.EscapeString(r.Value), html.EscapeString(r.Units), flag, html.EscapeString(ref))
		}
		body += "</table>"
	}
	body += "<h2>Result</h2><table>"
	for _, f := range res.Fields {
		body += fmt.Sprintf("<tr><th align=left>%s</th><td>%s</td></tr>", html.EscapeString(f.Name), html.EscapeString(f.Value))
	}
	body += "</table>"
	for _, l := range res.Lists {
		body += fmt.Sprintf("<h3>%s</h3><ul>", html.EscapeString(l.Name))
		for _, it := range l.Items {
			body += "<li>" + html.EscapeString(it) + "</li>"
		}
		body += "</ul>"
	}
	body += `<p><a href="/">Start over</a></p>`
	s.page(w, body)
}

func (s *Server) page(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := html.EscapeString(s.title)
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title>
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
<style>body{font-family:sans-serif;max-width:640px;margin:2rem auto;padding:0 1rem}
h1{font-size:1.3rem;color:#0a7d5a}
select,input{font-size:1rem;padding:.3rem}button{font-size:1rem;padding:.4rem .8rem;margin-right:.5rem}
th{padding-right:1rem}</style></head><body><h1>%s</h1>%s</body></html>`, title, title, body)
}

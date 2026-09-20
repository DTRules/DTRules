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

package web

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const webConstrainedEDD = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
  <entity name="patient" access="rw">
    <field name="diagnosis" type="string" access="rw" default_value="Acute Sinusitis" collect="true">
      <question text="Working diagnosis?" type="ascii"></question>
      <allowed_value value="Acute Sinusitis"></allowed_value>
      <allowed_value value="Chronic Sinusitis"></allowed_value>
    </field>
  </entity>
</entity_data_dictionary>`

// constrainedRunner drives the real collect resolver against a constrained
// EDD, so a browser's answer travels the whole write path the UI uses.
func constrainedRunner(t *testing.T) interviewRun {
	t.Helper()
	return func(a collect.Asker, _ string) (*Result, error) {
		rs := session.NewRuleSet("web-constraint-test")
		if err := rs.LoadEDD(strings.NewReader(webConstrainedEDD)); err != nil {
			return nil, err
		}
		pe := rs.GetEntityFactory().FindRefEntityByString("patient")
		diagnosis := dtrules.GetRName("diagnosis")
		if err := collect.New(a).MaybeCollect(pe, diagnosis); err != nil {
			return nil, err
		}
		v, _ := pe.Get(diagnosis)
		return &Result{Fields: []Field{{Name: "diagnosis", Value: v.StringValue()}}}, nil
	}
}

// interviewRun is the signature RunFunc adapts.
type interviewRun = func(collect.Asker, string) (*Result, error)

// browser is a cookie-carrying client for one interview session.
type browser struct {
	t   *testing.T
	ts  *httptest.Server
	cli *http.Client
}

func newBrowser(t *testing.T, run interviewRun) *browser {
	t.Helper()
	ts := httptest.NewServer(NewServer(RunFunc(run), "Test"))
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, ts: ts, cli: &http.Client{Jar: jar}}
}

func (b *browser) get(path string) string {
	b.t.Helper()
	r, err := b.cli.Get(b.ts.URL + path)
	if err != nil {
		b.t.Fatal(err)
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return string(body)
}

func (b *browser) answer(v string) string {
	b.t.Helper()
	r, err := b.cli.PostForm(b.ts.URL+"/answer", url.Values{"answer": {v}})
	if err != nil {
		b.t.Fatal(err)
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return string(body)
}

// TestWebInterview_RefusesValueOutsideVocabulary: the web interview is an
// outside write path like any other. An answer the field does not admit ends
// the run with an error naming the field, the value and the set — and no
// result is rendered (#1209).
func TestWebInterview_RefusesValueOutsideVocabulary(t *testing.T) {
	b := newBrowser(t, constrainedRunner(t))
	if page := b.get("/"); !strings.Contains(page, "Working diagnosis?") {
		t.Fatalf("first question not shown:\n%s", page)
	}
	page := b.answer("Banana")
	for _, want := range []string{"Error", "patient.diagnosis", "Banana", "Chronic Sinusitis"} {
		if !strings.Contains(page, want) {
			t.Errorf("refusal page missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "<h2>Result</h2>") {
		t.Errorf("a refused answer must not produce a result page:\n%s", page)
	}
}

// TestWebInterview_AcceptsValueInVocabulary is the same path with a legal
// answer: it runs through to a result. Case is irrelevant to the match.
func TestWebInterview_AcceptsValueInVocabulary(t *testing.T) {
	b := newBrowser(t, constrainedRunner(t))
	b.get("/")
	page := b.answer("chronic sinusitis")
	if !strings.Contains(page, "chronic sinusitis") {
		t.Errorf("accepted answer not in the result page:\n%s", page)
	}
	if strings.Contains(page, "<h2>Error</h2>") {
		t.Errorf("a legal answer must not produce an error page:\n%s", page)
	}
}

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

package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

// TestServer_Interview drives a two-question interview over real HTTP and
// checks that the blocking goroutine asks each question in turn and renders
// the collected answers.
func TestServer_Interview(t *testing.T) {
	run := func(a collect.Asker, _ string) (*Result, error) {
		v1, ok1, _ := a.Ask(collect.Request{Field: "pcr", QType: "number", Text: "How old?",
			RefLow: "0.7", RefHigh: "1.3", Units: "mg/dL"})
		v2, ok2, _ := a.Ask(collect.Request{Field: "allergic", QType: "multiple_choice", Text: "Allergic?",
			Options: []collect.Option{{Value: "true", Label: "Yes"}, {Value: "false", Label: "No"}}})
		age, al := "0", "false"
		if ok1 {
			age = v1.StringValue()
		}
		if ok2 {
			al = v2.StringValue()
		}
		return &Result{
			Fields:   []Field{{Name: "age", Value: age}, {Name: "allergic", Value: al}},
			Lists:    []List{{Name: "notes", Items: []string{"done"}}},
			Readings: []collect.Reading{{Field: "pcr", Value: age, Units: "mg/dL", Low: "0.7", High: "1.3", Flag: collect.Flag(age, "0.7", "1.3")}},
		}, nil
	}

	ts := httptest.NewServer(NewServer(run, "Test"))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	get := func(path string) string {
		r, err := client.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}
	post := func(answer string) string {
		r, err := client.PostForm(ts.URL+"/answer", url.Values{"answer": {answer}})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}

	// First page: the number question shows its reference range as guidance.
	if body := get("/"); !strings.Contains(body, "How old?") || !strings.Contains(body, `type="number"`) ||
		!strings.Contains(body, "normal range: 0.7–1.3 mg/dL") {
		t.Fatalf("first question / range not shown:\n%s", body)
	}
	// Answer with an out-of-range value -> the second question (with options).
	if body := post("1.85"); !strings.Contains(body, "Allergic?") || !strings.Contains(body, "<select") || !strings.Contains(body, ">Yes<") {
		t.Fatalf("second question not shown:\n%s", body)
	}
	// Answer it -> the result, with a lab-style Collected-values section that
	// flags the out-of-range reading H.
	body := post("true")
	for _, want := range []string{"Result", "allergic", "true", "notes", "done", "Collected values", "pcr", `>H</span>`, "ref 0.7–1.3"} {
		if !strings.Contains(body, want) {
			t.Errorf("result missing %q:\n%s", want, body)
		}
	}
}

// TestServer_TitleAndFavicon checks the tab title is the given project name
// and that a favicon is served.
func TestServer_TitleAndFavicon(t *testing.T) {
	run := func(a collect.Asker, _ string) (*Result, error) { return &Result{}, nil }
	ts := httptest.NewServer(NewServer(run, "My Project"))
	defer ts.Close()

	r, err := ts.Client().Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r.Body)
	r.Body.Close()
	page := string(b)
	if !strings.Contains(page, "<title>My Project</title>") {
		t.Errorf("tab title not the project name:\n%s", page)
	}
	if !strings.Contains(page, `href="/favicon.svg"`) {
		t.Errorf("favicon link missing:\n%s", page)
	}

	f, err := ts.Client().Get(ts.URL + "/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Body.Close()
	if ct := f.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("favicon content-type = %q, want image/svg+xml", ct)
	}
}

// TestServer_Review checks that answers accumulate as a transcript, the
// result offers a review action, and POSTing /review starts a new run fed the
// answers given so far (rebuilt as canonical data).
func TestServer_Review(t *testing.T) {
	run := func(a collect.Asker, reviewData string) (*Result, error) {
		if reviewData != "" {
			return &Result{Fields: []Field{{Name: "loaded", Value: reviewData}}}, nil
		}
		v, _, _ := a.Ask(collect.Request{Entity: "patient", Field: "age", Text: "Age?", QType: "number"})
		return &Result{Fields: []Field{{Name: "age", Value: v.StringValue()}}}, nil
	}
	ts := httptest.NewServer(NewServer(run, "T"))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	if _, err := client.Get(ts.URL + "/"); err != nil { // -> Age? question
		t.Fatal(err)
	}
	r1, err := client.PostForm(ts.URL+"/answer", url.Values{"answer": {"58"}}) // -> result
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := io.ReadAll(r1.Body)
	r1.Body.Close()
	page := string(b1)
	// The result shows the transcript (the answered question) and a review action.
	if !strings.Contains(page, "Age?") || !strings.Contains(page, "58") || !strings.Contains(page, "Review / edit answers") {
		t.Fatalf("result missing transcript/review action:\n%s", page)
	}

	// /review re-runs, seeded with the prior answer rebuilt as canonical data.
	r2, err := client.Post(ts.URL+"/review", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(r2.Body)
	r2.Body.Close()
	if !strings.Contains(string(b2), "&lt;age&gt;58&lt;/age&gt;") {
		t.Errorf("review run was not fed the prior answer:\n%s", string(b2))
	}
}

// TestServer_DataDownloadUpload checks that a finished run can be downloaded
// as canonical data and re-uploaded to start a review.
func TestServer_DataDownloadUpload(t *testing.T) {
	const saved = "<dtrules-data><patient><age>58</age></patient></dtrules-data>"
	run := func(a collect.Asker, reviewData string) (*Result, error) {
		if reviewData != "" {
			return &Result{Fields: []Field{{Name: "loaded", Value: reviewData}}}, nil
		}
		_, _, _ = a.Ask(collect.Request{Entity: "patient", Field: "age", Text: "Age?", QType: "number"})
		return &Result{Fields: []Field{{Name: "drug", Value: "X"}}, DataXML: saved}, nil
	}
	ts := httptest.NewServer(NewServer(run, "Demo"))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	client.Get(ts.URL + "/")                                                  // -> Age?
	r, _ := client.PostForm(ts.URL+"/answer", url.Values{"answer": {"58"}})   // -> result
	b, _ := io.ReadAll(r.Body)
	r.Body.Close()
	if !strings.Contains(string(b), "Download data (XML)") {
		t.Fatalf("result missing download link:\n%s", string(b))
	}

	// Download serves the canonical data as an attachment.
	d, _ := client.Get(ts.URL + "/data")
	body, _ := io.ReadAll(d.Body)
	d.Body.Close()
	if got := string(body); got != saved {
		t.Errorf("download body = %q, want %q", got, saved)
	}
	if cd := d.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("download not an attachment: %q", cd)
	}

	// Upload the file in a fresh browser -> a review run fed that data.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "session.xml")
	fw.Write(body)
	mw.Close()
	jar2, _ := cookiejar.New(nil)
	client2 := &http.Client{Jar: jar2}
	u, err := client2.Post(ts.URL+"/upload", mw.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	ub, _ := io.ReadAll(u.Body)
	u.Body.Close()
	if !strings.Contains(string(ub), "&lt;age&gt;58&lt;/age&gt;") {
		t.Errorf("upload did not feed the saved data to a review run:\n%s", string(ub))
	}
}

// TestServer_UseDefault checks that an empty submission keeps the default
// (ok=false at the asker).
func TestServer_UseDefault(t *testing.T) {
	run := func(a collect.Asker, _ string) (*Result, error) {
		_, ok, _ := a.Ask(collect.Request{Field: "age", QType: "number", Text: "Age?", Current: dtrules.GetRString("40")})
		used := "answered"
		if !ok {
			used = "default"
		}
		return &Result{Fields: []Field{{Name: "path", Value: used}}}, nil
	}
	ts := httptest.NewServer(NewServer(run, "Test"))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if _, err := client.Get(ts.URL + "/"); err != nil {
		t.Fatal(err)
	}
	r, err := client.PostForm(ts.URL+"/answer", url.Values{"answer": {""}})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(b), "default") {
		t.Errorf("empty submission should keep default:\n%s", string(b))
	}
}

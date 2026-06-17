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
	"io"
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
	run := func(a collect.Asker) (*Result, error) {
		v1, ok1, _ := a.Ask(collect.Request{Field: "age", QType: "number", Text: "How old?"})
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
			Fields: []Field{{Name: "age", Value: age}, {Name: "allergic", Value: al}},
			Lists:  []List{{Name: "notes", Items: []string{"done"}}},
		}, nil
	}

	ts := httptest.NewServer(NewServer(run))
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

	// First page: the first question.
	if body := get("/"); !strings.Contains(body, "How old?") || !strings.Contains(body, `type="number"`) {
		t.Fatalf("first question not shown:\n%s", body)
	}
	// Answer it -> the second question (with options).
	if body := post("58"); !strings.Contains(body, "Allergic?") || !strings.Contains(body, "<select") || !strings.Contains(body, ">Yes<") {
		t.Fatalf("second question not shown:\n%s", body)
	}
	// Answer it -> the result, reflecting both answers.
	body := post("true")
	for _, want := range []string{"Result", "age", "58", "allergic", "true", "notes", "done"} {
		if !strings.Contains(body, want) {
			t.Errorf("result missing %q:\n%s", want, body)
		}
	}
}

// TestServer_UseDefault checks that an empty submission keeps the default
// (ok=false at the asker).
func TestServer_UseDefault(t *testing.T) {
	run := func(a collect.Asker) (*Result, error) {
		_, ok, _ := a.Ask(collect.Request{Field: "age", QType: "number", Text: "Age?", Current: dtrules.GetRString("40")})
		used := "answered"
		if !ok {
			used = "default"
		}
		return &Result{Fields: []Field{{Name: "path", Value: used}}}, nil
	}
	ts := httptest.NewServer(NewServer(run))
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

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

package authoring

import (
	"strings"
	"testing"
)

// TestValidateAttribute_Constraints covers the declaration rules for #1209,
// vector 9 among them: a default outside the field's own vocabulary is an
// error, because no input can ever correct it.
func TestValidateAttribute_Constraints(t *testing.T) {
	base := Attribute{Name: "diagnosis", Type: "string", Access: "rw"}
	with := func(f func(*Attribute)) Attribute {
		a := base
		f(&a)
		return a
	}
	vocab := []string{"Acute Sinusitis", "Chronic Sinusitis"}

	cases := []struct {
		name    string
		attr    Attribute
		wantErr string // substring; "" means the attribute must validate
	}{
		{"a default inside the vocabulary", with(func(a *Attribute) {
			a.AllowedValues = vocab
			a.Default = "Acute Sinusitis"
		}), ""},
		{"a default matching in another case", with(func(a *Attribute) {
			a.AllowedValues = vocab
			a.Default = "acute sinusitis"
		}), ""},
		{"a default outside the vocabulary", with(func(a *Attribute) {
			a.AllowedValues = vocab
			a.Default = "Banana"
		}), "not one of the allowed values"},
		{"a default longer than max_length", with(func(a *Attribute) {
			a.MaxLength = "5"
			a.Default = "123456"
		}), "max_length is 5"},
		{"a default with more words than max_words", with(func(a *Attribute) {
			a.MaxWords = "2"
			a.Default = "one two three"
		}), "max_words is 2"},
		{"a limit that is not a number", with(func(a *Attribute) {
			a.MaxLength = "forty"
		}), "not a non-negative whole number"},
		{"a length limit on a double", with(func(a *Attribute) {
			a.Type = "double"
			a.MaxLength = "10"
		}), "apply to a string field"},
		{"a vocabulary on a boolean", with(func(a *Attribute) {
			a.Type = "boolean"
			a.AllowedValues = []string{"yes", "no"}
		}), "string or integer field"},
		{"a vocabulary on an integer", with(func(a *Attribute) {
			a.Type = "integer"
			a.AllowedValues = []string{"1", "2"}
			a.Default = "2"
		}), ""},
		{"one value listed twice", with(func(a *Attribute) {
			a.AllowedValues = []string{"Yes", "yes"}
		}), "listed twice"},
		{"an empty value in the vocabulary", with(func(a *Attribute) {
			a.AllowedValues = []string{"Yes", ""}
		}), "is empty"},
		{"no constraints at all", base, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAttribute(tc.attr)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("validateAttribute: unexpected error %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("validateAttribute accepted %+v; want error containing %q", tc.attr, tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
			if tc.wantErr != "" && err != nil && !strings.Contains(err.Error(), "diagnosis") {
				t.Errorf("error %q does not name the field", err)
			}
		})
	}
}

// TestConstraintsSurviveTheAttributeView pins the typed view: constraints
// must survive Attribute → XML → Attribute, or `edd get` after `edd put`
// returns something the author did not write (#1209).
func TestConstraintsSurviveTheAttributeView(t *testing.T) {
	a := Attribute{
		Name: "diagnosis", Type: "string", Access: "rw",
		AllowedValues: []string{"Acute Sinusitis", "Chronic Sinusitis"},
		MaxLength:     "40", MaxWords: "5",
	}
	got := attributeFromXML(attributeToXML(a))
	if len(got.AllowedValues) != 2 || got.AllowedValues[0] != "Acute Sinusitis" {
		t.Errorf("allowed_values lost: %+v", got.AllowedValues)
	}
	if got.MaxLength != "40" || got.MaxWords != "5" {
		t.Errorf("limits lost: max_length=%q max_words=%q", got.MaxLength, got.MaxWords)
	}
}

// TestPatchConstraints covers the patch conventions: an omitted constraint
// keeps what the field has, and the explicit clears ([] and "0") remove it.
func TestPatchConstraints(t *testing.T) {
	base := Attribute{
		Name: "diagnosis", Type: "string", Access: "rw",
		AllowedValues: []string{"Acute Sinusitis"}, MaxLength: "40", MaxWords: "5",
	}

	kept := mergeAttribute(base, Attribute{Comment: "a comment"})
	if len(kept.AllowedValues) != 1 || kept.MaxLength != "40" || kept.MaxWords != "5" {
		t.Errorf("an unrelated patch dropped the constraints: %+v", kept)
	}

	cleared := mergeAttribute(base, Attribute{AllowedValues: []string{}, MaxLength: "0", MaxWords: "0"})
	if len(cleared.AllowedValues) != 0 || cleared.MaxLength != "" || cleared.MaxWords != "" {
		t.Errorf("the explicit clear left constraints behind: %+v", cleared)
	}

	replaced := mergeAttribute(base, Attribute{AllowedValues: []string{"Chronic Sinusitis"}, MaxLength: "80"})
	if len(replaced.AllowedValues) != 1 || replaced.AllowedValues[0] != "Chronic Sinusitis" ||
		replaced.MaxLength != "80" || replaced.MaxWords != "5" {
		t.Errorf("patch did not replace exactly what it named: %+v", replaced)
	}
}

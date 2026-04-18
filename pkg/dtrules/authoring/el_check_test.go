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

package authoring_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func TestCheckCondition_WithSymbols(t *testing.T) {
	symbols := map[string]string{"age": "long"}
	postfix, err := authoring.CheckCondition("age is greater than 18", symbols)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postfix == "" {
		t.Fatal("expected non-empty postfix")
	}
}

func TestCheckCondition_NilSymbols(t *testing.T) {
	postfix, err := authoring.CheckCondition("x is equal to 5", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postfix == "" {
		t.Fatal("expected non-empty postfix")
	}
}

func TestCheckCondition_InvalidExpression(t *testing.T) {
	_, err := authoring.CheckCondition("age ~ 18", nil)
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	// Error should contain position info (line N:M)
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("expected position info in error, got: %v", err)
	}
}

func TestCheckAction_Valid(t *testing.T) {
	postfix, err := authoring.CheckAction("set x = 5;", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postfix == "" {
		t.Fatal("expected non-empty postfix")
	}
}

func TestCheckAction_Invalid(t *testing.T) {
	_, err := authoring.CheckAction("set = = =;", nil)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestCheckContext_Valid(t *testing.T) {
	postfix, err := authoring.CheckContext("local bigint x = (bigint) 10;", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if postfix == "" {
		t.Fatal("expected non-empty postfix")
	}
}

func TestCheckCondition_EmptyString(t *testing.T) {
	postfix, err := authoring.CheckCondition("", nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if postfix != "" {
		t.Errorf("expected empty postfix for empty input, got: %q", postfix)
	}
}

func TestCheckAction_EmptyString(t *testing.T) {
	postfix, err := authoring.CheckAction("", nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if postfix != "" {
		t.Errorf("expected empty postfix for empty input, got: %q", postfix)
	}
}

func TestCheckContext_EmptyString(t *testing.T) {
	postfix, err := authoring.CheckContext("", nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if postfix != "" {
		t.Errorf("expected empty postfix for empty input, got: %q", postfix)
	}
}

// TestAuthoringHasNoCmdImports ensures the authoring package has no cmd/ dependencies.
func TestAuthoringHasNoCmdImports(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse authoring package: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if strings.Contains(path, "cmd/") {
					t.Errorf("authoring package must not import cmd/ packages, found: %s", path)
				}
			}
		}
	}
}

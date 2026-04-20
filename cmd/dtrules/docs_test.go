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

package main

import (
	"strings"
	"testing"
)

func TestDocumentation_MappingTopic(t *testing.T) {
	doc, ok := docTopics["mapping"]
	if !ok {
		t.Fatal("mapping topic not registered in docTopics")
	}

	required := []string{"setattribute", "MAP:", "RAttribute", "enclosure"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("mapping doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_ProjectLayoutTopic(t *testing.T) {
	doc, ok := docTopics["project-layout"]
	if !ok {
		t.Fatal("project-layout topic not registered in docTopics")
	}
	required := []string{"_dt.xml", "_edd.xml", "_map.xml", ".sync-manifest.json", "A1"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("project-layout doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_DatabaseTopic(t *testing.T) {
	doc, ok := docTopics["database"]
	if !ok {
		t.Fatal("database topic not registered in docTopics")
	}
	required := []string{"key", "mapping*key", "EDD"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("database doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_ArchitectureTopic(t *testing.T) {
	doc, ok := docTopics["architecture"]
	if !ok {
		t.Fatal("architecture topic not registered in docTopics")
	}
	required := []string{"dev-time", "deploy-time", "go:embed"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("architecture doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_EmbeddingTopic(t *testing.T) {
	doc, ok := docTopics["embedding"]
	if !ok {
		t.Fatal("embedding topic not registered in docTopics")
	}
	required := []string{"//go:embed", "embed.FS"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("embedding doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_EmbeddingContainsExtractExcel(t *testing.T) {
	doc, ok := docTopics["embedding"]
	if !ok {
		t.Fatal("embedding topic not registered in docTopics")
	}
	if !strings.Contains(doc, "ExtractExcel") {
		t.Error("embedding doc should document ExtractExcel for dumping embedded rules")
	}
}

func TestDocumentation_EmbeddingUsesLoadRulesFromFS(t *testing.T) {
	doc, ok := docTopics["embedding"]
	if !ok {
		t.Fatal("embedding topic not registered in docTopics")
	}
	if !strings.Contains(doc, "LoadRulesFromFS") {
		t.Error("embedding doc should show LoadRulesFromFS (not tempdir workaround)")
	}
	if strings.Contains(doc, "MkdirTemp") || strings.Contains(doc, "copyEmbedToDir") {
		t.Error("embedding doc should not reference the old tempdir workaround")
	}
}

func TestDocumentation_AuthoringTopic(t *testing.T) {
	doc, ok := docTopics["authoring"]
	if !ok {
		t.Fatal("authoring topic not registered in docTopics")
	}

	required := []string{
		"OpenProject",
		"ExecuteEntry",
		"ResumeAt",
		"SetAttribute",
		"ResetState",
		"AddCondition",
		"UpdateCondition",
		"AddColumn",
		"UpdateColumn",
		"EntityStack",
		"Resolve",
		"Step",
		"Continue",
		"CheckCondition",
		"CheckAction",
		"CheckContext",
		"pkg/dtrules/authoring",
		"go doc",
	}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("authoring doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_AuthoringInIndex(t *testing.T) {
	_, ok := docTopics["authoring"]
	if !ok {
		t.Fatal("authoring topic not found in docTopics map")
	}
}

func TestDocumentation_FixedTopic(t *testing.T) {
	doc, ok := docTopics["fixed"]
	if !ok {
		t.Fatal("fixed topic not registered in docTopics")
	}
	if doc == "" {
		t.Fatal("fixed topic is empty")
	}
	required := []string{"8-decimal", "1.5fp", "fp+", "cvfp", "(fixed)", "local fixed"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("fixed doc missing required term: %q", term)
		}
	}
}

func TestDocumentation_OperatorsHasFixedPointSection(t *testing.T) {
	doc, ok := docTopics["operators"]
	if !ok {
		t.Fatal("operators topic not registered in docTopics")
	}
	required := []string{"Fixed-Point Operators", "fp+", "fpmin", "fpmax", "cvfp"}
	for _, term := range required {
		if !strings.Contains(doc, term) {
			t.Errorf("operators doc missing required fixed-point term: %q", term)
		}
	}
}

func TestDocumentation_ELLiteralsMentionsFixed(t *testing.T) {
	doc, ok := docTopics["el"]
	if !ok {
		t.Fatal("el topic not registered in docTopics")
	}
	if !strings.Contains(doc, "fp") {
		t.Error("el doc literals section should mention fp suffix")
	}
	if !strings.Contains(doc, "1.5fp") {
		t.Error("el doc should include a 1.5fp literal example")
	}
}

func TestDocumentation_EDDTypeTableHasFixed(t *testing.T) {
	doc, ok := docTopics["edd"]
	if !ok {
		t.Fatal("edd topic not registered in docTopics")
	}
	if !strings.Contains(doc, "fixed") {
		t.Error("edd type table should list 'fixed'")
	}
	if !strings.Contains(doc, "0.0fp") {
		t.Error("edd type table should show 0.0fp default")
	}
}

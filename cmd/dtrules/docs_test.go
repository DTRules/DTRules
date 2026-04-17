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

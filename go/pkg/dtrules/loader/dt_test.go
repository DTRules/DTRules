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

package loader

import (
	"strings"
	"testing"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
)

func TestDTLoaderMalformedXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0"?>
<decision_tables>
  <decision_table>
    <table_name>Broken
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err == nil {
		t.Fatal("Expected error for malformed XML")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestDTLoaderEmptyXML(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
</decision_tables>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Empty DT file should load without error: %v", err)
	}
}

func TestDTLoaderReadError(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	err := loader.Load(&errorReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

func TestDTLoaderGetErrors(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	// Initially no errors
	if len(loader.GetErrors()) != 0 {
		t.Error("Expected no errors initially")
	}
}

func TestNewDTLoader(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewDTLoader(nil, factory)

	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}
	if loader.factory != factory {
		t.Error("Expected factory to be set")
	}
}

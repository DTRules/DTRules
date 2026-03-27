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

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
)

// TestEDDLoaderWithFileMetadata tests EDD loading when file_metadata is present
func TestEDDLoaderWithFileMetadata(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/CA_edd.xml</file_path>
  </file_metadata>
  <entity name="california_entity" access="rw">
    <field name="state_tax" type="double" access="rw"/>
    <field name="state_code" type="string" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with file_metadata: %v", err)
	}

	// Verify entity was created
	entityName := dtrules.GetRName("california_entity")
	refEntity, err := factory.GetReferenceEntity(entityName)
	if err != nil {
		t.Fatalf("Failed to get california_entity: %v", err)
	}
	if refEntity == nil {
		t.Fatal("Expected california_entity to be created")
	}

	// Cast to concrete type to access GetFilePath
	ent, ok := refEntity.(*entity.REntity)
	if !ok {
		t.Fatalf("Entity is not *REntity, got type: %T", refEntity)
	}

	// Verify file path is set correctly from file_metadata
	filePath := ent.GetFilePath()
	if filePath != "states/CA_edd.xml" {
		t.Errorf("Expected file_path='states/CA_edd.xml', got: %s", filePath)
	}
}

// TestEDDLoaderLegacyXlsFile tests EDD loading with legacy xls_file attribute (no file_metadata)
func TestEDDLoaderLegacyXlsFile(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="legacy_entity" access="rw" xls_file="LegacyEDD.xls">
    <field name="value" type="integer" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with xls_file: %v", err)
	}

	// Verify entity was created
	entityName := dtrules.GetRName("legacy_entity")
	refEntity, err := factory.GetReferenceEntity(entityName)
	if err != nil {
		t.Fatalf("Failed to get legacy_entity: %v", err)
	}
	if refEntity == nil {
		t.Fatal("Expected legacy_entity to be created")
	}

	// Cast to concrete type to access GetFilePath
	ent, ok := refEntity.(*entity.REntity)
	if !ok {
		t.Fatalf("Entity is not *REntity, got type: %T", refEntity)
	}

	// GetFilePath should fall back to xls_file value
	filePath := ent.GetFilePath()
	if filePath != "LegacyEDD.xls" {
		t.Errorf("Expected GetFilePath fallback to 'LegacyEDD.xls', got: %s", filePath)
	}
}

// TestEDDGetFilePathPriority tests that file_metadata takes precedence over xls_file
func TestEDDGetFilePathPriority(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/NY_edd.xml</file_path>
  </file_metadata>
  <entity name="priority_entity" access="rw" xls_file="OldPath.xls">
    <field name="value" type="integer" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with both file_metadata and xls_file: %v", err)
	}

	// Verify entity was created
	entityName := dtrules.GetRName("priority_entity")
	refEntity, err := factory.GetReferenceEntity(entityName)
	if err != nil {
		t.Fatalf("Failed to get priority_entity: %v", err)
	}
	if refEntity == nil {
		t.Fatal("Expected priority_entity to be created")
	}

	// Cast to concrete type to access GetFilePath
	ent, ok := refEntity.(*entity.REntity)
	if !ok {
		t.Fatalf("Entity is not *REntity, got type: %T", refEntity)
	}

	// file_metadata should take precedence over xls_file
	filePath := ent.GetFilePath()
	if filePath != "states/NY_edd.xml" {
		t.Errorf("Expected file_metadata to take precedence: 'states/NY_edd.xml', got: %s", filePath)
	}

	// Verify old xls_file value is not used
	if filePath == "OldPath.xls" {
		t.Error("file_metadata should override xls_file, but got xls_file value instead")
	}
}

// TestEDDGetFilePathEmpty tests behavior when neither file_metadata nor xls_file is present
func TestEDDGetFilePathEmpty(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="no_path_entity" access="rw">
    <field name="value" type="integer" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD without file_metadata or xls_file: %v", err)
	}

	// Verify entity was created
	entityName := dtrules.GetRName("no_path_entity")
	refEntity, err := factory.GetReferenceEntity(entityName)
	if err != nil {
		t.Fatalf("Failed to get no_path_entity: %v", err)
	}
	if refEntity == nil {
		t.Fatal("Expected no_path_entity to be created")
	}

	// Cast to concrete type to access GetFilePath
	ent, ok := refEntity.(*entity.REntity)
	if !ok {
		t.Fatalf("Entity is not *REntity, got type: %T", refEntity)
	}

	// GetFilePath should return empty string
	filePath := ent.GetFilePath()
	if filePath != "" {
		t.Errorf("Expected empty file path, got: %s", filePath)
	}
}

// TestEDDLoaderMultipleEntitiesWithFileMetadata tests that file_metadata applies to all entities
func TestEDDLoaderMultipleEntitiesWithFileMetadata(t *testing.T) {
	factory := entity.NewFactory(nil)
	loader := NewEDDLoader(nil, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <file_metadata>
    <file_path>states/CO_edd.xml</file_path>
  </file_metadata>
  <entity name="entity_one" access="rw">
    <field name="value1" type="integer" access="rw"/>
  </entity>
  <entity name="entity_two" access="rw">
    <field name="value2" type="double" access="rw"/>
  </entity>
</entity_data_dictionary>`

	err := loader.Load(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to load EDD with multiple entities: %v", err)
	}

	// Verify both entities have the same file path from file_metadata
	entity1Obj, _ := factory.GetReferenceEntity(dtrules.GetRName("entity_one"))
	if entity1Obj == nil {
		t.Fatal("Expected entity_one to be created")
	}
	entity1, ok := entity1Obj.(*entity.REntity)
	if !ok {
		t.Fatalf("entity_one is not *REntity, got type: %T", entity1Obj)
	}

	entity2Obj, _ := factory.GetReferenceEntity(dtrules.GetRName("entity_two"))
	if entity2Obj == nil {
		t.Fatal("Expected entity_two to be created")
	}
	entity2, ok := entity2Obj.(*entity.REntity)
	if !ok {
		t.Fatalf("entity_two is not *REntity, got type: %T", entity2Obj)
	}

	path1 := entity1.GetFilePath()
	path2 := entity2.GetFilePath()

	if path1 != "states/CO_edd.xml" {
		t.Errorf("Expected entity_one file_path='states/CO_edd.xml', got: %s", path1)
	}

	if path2 != "states/CO_edd.xml" {
		t.Errorf("Expected entity_two file_path='states/CO_edd.xml', got: %s", path2)
	}
}

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

package authoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenameTablePersists guards the fix for a silent no-op: assigning to a
// Table view's Name field changes nothing durable, because Project.Table
// builds a fresh view per call and only mutator methods sync back. The CLI's
// set-name patch did exactly that and reported success while the XML kept the
// old name. RenameTable mutates the underlying record.
func TestRenameTablePersists(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Old_Name</table_name>
<xls_file>t.xls</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS />
<TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts>
<initial_actions></initial_actions>
<conditions>
<condition_details>
<condition_number>1</condition_number>
<condition_comment>always</condition_comment>
<condition_dsl></condition_dsl>
<condition_postfix></condition_postfix>
<column_value><column_number>1</column_number><column_value>Y</column_value></column_value>
</condition_details>
</conditions>
<actions></actions>
</decision_table>
</decision_tables>
`
	if err := os.WriteFile(filepath.Join(xmlDir, "t_dt.xml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The trap this API exists to close: a view assignment does not persist.
	view := p.Table("Old_Name")
	if view == nil {
		t.Fatal("table not found")
	}
	view.Name = "Assigned_Only"

	if err := p.RenameTable("Old_Name", "New_Name"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if p.Table("Old_Name") != nil {
		t.Error("old name still resolves after rename")
	}
	if p.Table("New_Name") == nil {
		t.Error("new name does not resolve after rename")
	}
	if err := p.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(xmlDir, "t_dt.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<table_name>New_Name</table_name>") {
		t.Error("rename did not persist to XML")
	}
	if strings.Contains(string(data), "Assigned_Only") {
		t.Error("a bare view assignment leaked to XML — that path must stay inert")
	}

	// Collision guard.
	if err := p.RenameTable("New_Name", "New_Name"); err != nil {
		t.Errorf("same-name rename should be a no-op, got %v", err)
	}
	if _, err := p.AddTable("Other", "t_dt.xml", ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := p.RenameTable("Other", "New_Name"); err == nil {
		t.Error("rename onto an existing name must fail")
	}
}

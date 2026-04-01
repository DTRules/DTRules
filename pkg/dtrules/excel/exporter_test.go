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

package excel

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
)

func TestSortTablesByTableNumber(t *testing.T) {
	tests := []struct {
		name     string
		tables   func() []*decisiontable.RDecisionTable
		expected []string // expected order of table names
	}{
		{
			name: "numeric ordering",
			tables: func() []*decisiontable.RDecisionTable {
				// Create tables with different TABLE_NUMBER values
				tables := []*decisiontable.RDecisionTable{
					createTestTable("Table_C", "30"),
					createTestTable("Table_A", "10"),
					createTestTable("Table_B", "20"),
				}
				return tables
			},
			expected: []string{"Table_A", "Table_B", "Table_C"},
		},
		{
			name: "tables without numbers sort alphabetically after numbered",
			tables: func() []*decisiontable.RDecisionTable {
				tables := []*decisiontable.RDecisionTable{
					createTestTable("Zebra", ""),
					createTestTable("Alpha", ""),
					createTestTable("Numbered", "5"),
				}
				return tables
			},
			expected: []string{"Numbered", "Alpha", "Zebra"},
		},
		{
			name: "numeric vs string sorting",
			tables: func() []*decisiontable.RDecisionTable {
				// Ensure "10" comes after "2" numerically (not before like string sort)
				tables := []*decisiontable.RDecisionTable{
					createTestTable("Table_10", "10"),
					createTestTable("Table_2", "2"),
					createTestTable("Table_1", "1"),
				}
				return tables
			},
			expected: []string{"Table_1", "Table_2", "Table_10"},
		},
		{
			name: "mixed numbered and unnumbered",
			tables: func() []*decisiontable.RDecisionTable {
				tables := []*decisiontable.RDecisionTable{
					createTestTable("Z_Table", ""),
					createTestTable("A_Table", "100"),
					createTestTable("M_Table", "50"),
					createTestTable("B_Table", ""),
				}
				return tables
			},
			expected: []string{"M_Table", "A_Table", "B_Table", "Z_Table"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := tt.tables()
			sortTablesByTableNumber(tables)

			for i, expected := range tt.expected {
				if tables[i].GetName() != expected {
					t.Errorf("position %d: got %s, want %s", i, tables[i].GetName(), expected)
				}
			}
		})
	}
}

func createTestTable(name, tableNumber string) *decisiontable.RDecisionTable {
	rname := dtrules.GetRName(name)
	// Pass nil for session - it's not needed for sorting tests
	dt := decisiontable.NewRDecisionTable(rname, nil)
	if tableNumber != "" {
		dt.SetField("TABLE_NUMBER", tableNumber)
	}
	return dt
}

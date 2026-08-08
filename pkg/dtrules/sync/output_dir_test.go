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

package sync

import "testing"

// A workbook produces a DT file, an EDD file, or both, and the unused path is
// left empty. Deriving the output directory from DTXMLPath alone answers "."
// for an EDD-only workbook, so the import landed in the process's working
// directory instead of the project (#1056).

func TestOutputDirUsesWhicheverPathIsSet(t *testing.T) {
	cases := []struct {
		name     string
		wb       CombinedWorkbook
		fallback string
		want     string
	}{
		{
			name: "mixed workbook uses the DT path",
			wb:   CombinedWorkbook{DTXMLPath: "/p/xml/Foo_dt.xml", EDDXMLPath: "/p/xml/Foo_edd.xml"},
			want: "/p/xml",
		},
		{
			name: "DT-only workbook",
			wb:   CombinedWorkbook{DTXMLPath: "/p/xml/Foo_dt.xml"},
			want: "/p/xml",
		},
		{
			// The regression. Before the fix this answered "." and the EDD
			// was written wherever the process happened to be running.
			name: "EDD-only workbook falls through to the EDD path",
			wb:   CombinedWorkbook{EDDXMLPath: "/p/xml/Foo_edd.xml"},
			want: "/p/xml",
		},
		{
			name: "nested workbook keeps its subdirectory",
			wb:   CombinedWorkbook{EDDXMLPath: "/p/xml/states/CO_edd.xml"},
			want: "/p/xml/states",
		},
		{
			name:     "neither path set falls back to the syncer's xml dir",
			wb:       CombinedWorkbook{},
			fallback: "/p/xml",
			want:     "/p/xml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.wb.outputDir(tc.fallback); got != tc.want {
				t.Errorf("outputDir(%q) = %q, want %q", tc.fallback, got, tc.want)
			}
		})
	}
}

// The specific shape that bit us: never answer the working directory when the
// workbook names a real destination.
func TestOutputDirNeverAnswersWorkingDirectory(t *testing.T) {
	wb := CombinedWorkbook{EDDXMLPath: "/project/xml/TaxReturn_edd.xml"}
	if got := wb.outputDir("/project/xml"); got == "." {
		t.Fatal(`outputDir answered "." — the import would write into the caller's cwd`)
	}
}

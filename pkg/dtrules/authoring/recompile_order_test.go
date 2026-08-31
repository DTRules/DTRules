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

import "testing"

// The order the contract states: export Excel, then compile it to produce XML.
//
// Writing the XML from the model and then exporting Excel from that XML leaves
// two artifacts that agree because two code paths agreed — and `verify`'s
// [build] check existed to keep discovering when they did not. Recompiling the
// workbook makes the XML literally the output of compiling the Excel, so
// agreement is a property of how the file was produced rather than something
// to test for (#1091).

func TestRecompileIsSkippedWhenNothingChanged(t *testing.T) {
	// No workbooks changed means no recompile: an edit to one table in a
	// 58-workbook project must not regenerate 58 XML files.
	if err := recompileWorkbooks(t.TempDir(), t.TempDir(), nil); err != nil {
		t.Errorf("recompiling an empty set should be a no-op, got %v", err)
	}
}

// A workbook that cannot be read is reported against its own name rather than
// failing anonymously somewhere inside the importer.
func TestRecompileNamesTheWorkbookItFailedOn(t *testing.T) {
	err := recompileWorkbooks(t.TempDir(), t.TempDir(), []string{"/nonexistent/Missing.xlsx"})
	if err == nil {
		t.Fatal("recompiling a workbook that does not exist should fail")
	}
	if !contains(err.Error(), "Missing.xlsx") {
		t.Errorf("error should name the workbook: %v", err)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(needle) == 0 || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

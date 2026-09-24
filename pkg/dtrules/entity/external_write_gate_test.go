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

package entity

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEveryDecodingWriterIsGated is the grep-grade acceptance of #1220.
//
// CheckExternalWrite is the one gate on a value arriving from outside (#1209):
// allowed_values, max_length and max_words hold only if every path that
// decodes outside data into a field goes through it. pkg/dtrules/loader's
// JSONDataLoader decoded JSON and called Put directly, so a value outside a
// field's constraints was accepted there; it was unused and has been deleted.
//
// The rule checked: a non-test file under pkg/ that decodes JSON or XML and
// calls .Put( must also call CheckExternalWrite. It is a heuristic, as a
// grep is -- a file that decodes rules rather than data and writes entities
// would need a line here saying why it is exempt.
func TestEveryDecodingWriterIsGated(t *testing.T) {
	decodes := regexp.MustCompile(`json\.Unmarshal|json\.NewDecoder|xml\.NewDecoder|xml\.Unmarshal`)
	puts := regexp.MustCompile(`\.Put\(`)
	root := ".." // pkg/dtrules
	checked := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !decodes.Match(src) || !puts.Match(src) {
			return nil
		}
		checked++
		if !strings.Contains(string(src), "CheckExternalWrite") {
			t.Errorf("%s decodes outside data and calls Put without entity.CheckExternalWrite", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no decoding writer found under pkg/dtrules: the walk is not seeing the gated loaders, so it proves nothing")
	}
}

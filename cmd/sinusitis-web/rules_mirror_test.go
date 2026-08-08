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
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedRulesMirrorTheSample keeps rules/xml byte-identical to
// sampleprojects/SinusitisTherapy/xml.
//
// This copy exists only because //go:embed cannot reach outside the package
// directory. The sample project is the source of truth: it is what
// `dtrules build` maintains against the workbook, what `dtrules verify`
// checks, and what the campaign repairs. A drifting copy is how the web demo
// ends up serving different rules from the ones anyone edits — and the two
// had already drifted, the sample missing the collect metadata that makes the
// interview work at all.
//
// To update: edit the sample project, run `dtrules build` there, then
//
//	cp sampleprojects/SinusitisTherapy/xml/*.xml cmd/sinusitis-web/rules/xml/
func TestEmbeddedRulesMirrorTheSample(t *testing.T) {
	const sampleDir = "../../sampleprojects/SinusitisTherapy/xml"
	const mirrorDir = "rules/xml"

	sampleFiles, err := os.ReadDir(sampleDir)
	if err != nil {
		t.Skipf("sample project not available: %v", err)
	}
	mirrorFiles, err := os.ReadDir(mirrorDir)
	if err != nil {
		t.Fatalf("read %s: %v", mirrorDir, err)
	}

	inMirror := map[string]bool{}
	for _, f := range mirrorFiles {
		inMirror[f.Name()] = true
	}

	for _, f := range sampleFiles {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !inMirror[name] {
			t.Errorf("%s is in the sample project but not embedded in %s", name, mirrorDir)
			continue
		}
		delete(inMirror, name)

		want, err := os.ReadFile(filepath.Join(sampleDir, name))
		if err != nil {
			t.Fatalf("read sample %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join(mirrorDir, name))
		if err != nil {
			t.Fatalf("read mirror %s: %v", name, err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s has drifted from the sample project — re-copy it from %s", name, sampleDir)
		}
	}

	for name := range inMirror {
		t.Errorf("%s is embedded in %s but no longer exists in the sample project", name, mirrorDir)
	}
}

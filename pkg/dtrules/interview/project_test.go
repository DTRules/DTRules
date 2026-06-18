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

package interview

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

// declineAsker never supplies a value (used when full input is provided).
var declineAsker = collect.AskerFunc(func(collect.Request) (dtrules.Object, bool, error) {
	return nil, false, nil
})

func TestProject_RunSinusitis(t *testing.T) {
	root := filepath.Join("..", "..", "..", "sampleprojects", "SinusitisTherapy")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("sample project not present: %v", err)
	}
	xmlDir := filepath.Join(root, "xml")
	input := filepath.Join(root, "testfiles", "TestScenarios", "ChallengeExample", "input.xml")

	runner := Project(xmlDir, Options{Entry: "Determine_Therapy", Input: input})
	res, err := runner.Run(declineAsker, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// buildResult + datafile.Write succeeded (H6) and produced output.
	if res.DataXML == "" {
		t.Error("expected canonical data to be produced")
	}
	var drug string
	for _, f := range res.Fields {
		if f.Name == "recommended_drug" {
			drug = f.Value
		}
	}
	if drug == "" {
		t.Errorf("expected a recommended_drug in the result; got fields %+v", res.Fields)
	}
}

func TestProject_MissingEntryTableErrors(t *testing.T) {
	root := filepath.Join("..", "..", "..", "sampleprojects", "SinusitisTherapy")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("sample project not present: %v", err)
	}
	runner := Project(filepath.Join(root, "xml"), Options{Entry: "NoSuchTable"})
	if _, err := runner.Run(declineAsker, ""); err == nil {
		t.Error("expected an error for a missing entry table, got nil (H5)")
	}
}

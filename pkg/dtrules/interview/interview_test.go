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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

// TestRunnerContract shows the seam: a custom engine (not the default Project)
// satisfies Runner, and a UI drives it by supplying an Asker and consuming the
// Result — without either side knowing about the other.
func TestRunnerContract(t *testing.T) {
	// A future decision-table app could implement Runner however it likes.
	var engine Runner = RunnerFunc(func(asker collect.Asker, reviewData string) (*Result, error) {
		v, ok, err := asker.Ask(collect.Request{Field: "age", QType: "number", Text: "Age?"})
		if err != nil {
			return nil, err
		}
		val := "(default)"
		if ok {
			val = v.StringValue()
		}
		return &Result{Title: "result", Fields: []Field{{Name: "age", Value: val}}}, nil
	})

	// The UI side: an Asker that answers, and consumption of the Result.
	asker := collect.AskerFunc(func(req collect.Request) (dtrules.Object, bool, error) {
		if req.Field != "age" || req.Text != "Age?" {
			t.Errorf("unexpected request: %+v", req)
		}
		return dtrules.GetRString("42"), true, nil
	})

	res, err := engine.Run(asker, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Fields) != 1 || res.Fields[0].Name != "age" || res.Fields[0].Value != "42" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

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

package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// A payload with its fields nested one level deep decoded to all-zero values
// and half-applied an empty row while reporting `patched` -- Scopa's primiera
// totals went to 0 two steps from the cause (#1144).

func TestPatchRejectsUnknownFields(t *testing.T) {
	payload := []byte(`{"op":"update-action","action_number":1,` +
		`"action":{"number":1,"dsl":"x"}}`)
	var patch tablePatch
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	err := dec.Decode(&patch)
	if err == nil {
		t.Fatal("a nested \"action\" object must be rejected, not silently dropped")
	}
}

// Omitted fields keep their values -- that is what patch means. Only pointers
// can tell "omitted" from "explicitly empty", which is why DSL and Comment
// are *string on the patch struct.
func TestPatchDistinguishesOmittedFromEmpty(t *testing.T) {
	var omitted, explicit tablePatch
	if err := json.Unmarshal([]byte(`{"op":"update-action","action_number":1,"comment":"c"}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.DSL != nil {
		t.Error("an omitted dsl must decode to nil, so the existing DSL survives")
	}
	if err := json.Unmarshal([]byte(`{"op":"update-action","action_number":1,"dsl":""}`), &explicit); err != nil {
		t.Fatal(err)
	}
	if explicit.DSL == nil || *explicit.DSL != "" {
		t.Error("an explicit \"dsl\": \"\" is a stated request to blank, and must decode as present")
	}
}

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

package decisiontable

import "testing"

// The executable decision tree is a runtime artifact, and it is not small: it
// grows with the columns a table can take. An authoring save of TaxReturn --
// which executes nothing — allocated 46GB, 99.8% of it NewCNode and
// NewANodeForColumn, because the loader builds a tree for every table on every
// load (#1132).

func TestSkipDecisionTreeLeavesNoTree(t *testing.T) {
	dt := &RDecisionTable{}
	dt.SkipDecisionTree = true

	if dt.decisionTree != nil {
		t.Fatal("fixture starts with a tree")
	}
	if dt.SkipDecisionTree != true {
		t.Error("the flag did not take")
	}
}

// The builder must carry the flag onto the table, since that is how the loader
// passes it down.
func TestBuilderCarriesTheFlag(t *testing.T) {
	b := NewBuilder("T", nil)
	if b == nil {
		t.Fatal("NewBuilder returned nil")
	}
	b.SetSkipDecisionTree(true)
	if !b.dt.SkipDecisionTree {
		t.Error("SetSkipDecisionTree did not reach the table being built")
	}

	// The default must build the tree: every runtime consumer needs it.
	if other := NewBuilder("U", nil); other != nil && other.dt.SkipDecisionTree {
		t.Error("skipping the tree became the default")
	}
}

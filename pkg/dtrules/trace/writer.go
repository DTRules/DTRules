// Copyright 2025 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Provenance identifies what produced a trace: the DTRules build and a
// fingerprint of the rules that ran. Loading a trace under a different
// DTRules version or different rules warrants a warning — replayed state
// may diverge from what actually happened.
type Provenance struct {
	DTRulesVersion   string
	RulesFingerprint string
}

// FingerprintRules hashes the project's XML rule files (sorted by relative
// path, names included) so a trace can record exactly which rules ran.
// The result is "sha256:<hex>".
func FingerprintRules(xmlDir string) (string, error) {
	var files []string
	err := filepath.Walk(xmlDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasSuffix(strings.ToLower(path), ".xml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)

	h := sha256.New()
	for _, f := range files {
		rel, _ := filepath.Rel(xmlDir, f)
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\n", rel)
		h.Write(data)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// StackState is the slice of session state the trace writer reads: the
// entity stack at the end of execution.
type StackState interface {
	EntityDepth() int
	EntityFetch(i int) (dtrules.Entity, error)
}

// WriteHeader opens a DTRulesTrace document, recording provenance as root
// attributes. Empty fields are omitted so old-style consumers see the same
// bare <DTRulesTrace> they always did.
func WriteHeader(w io.Writer, p Provenance) {
	fmt.Fprint(w, "<DTRulesTrace")
	if p.DTRulesVersion != "" {
		fmt.Fprintf(w, " dtrules_version=\"%s\"", escapeXML(p.DTRulesVersion))
	}
	if p.RulesFingerprint != "" {
		fmt.Fprintf(w, " rules_fingerprint=\"%s\"", escapeXML(p.RulesFingerprint))
	}
	fmt.Fprintln(w, ">")
}

// WriteFinalState records the resulting entity stack — every entity with
// every attribute value — so a loaded trace can be verified against the
// state its replay reconstructs.
func WriteFinalState(w io.Writer, state StackState) {
	fmt.Fprintln(w, "<finalState>")
	depth := state.EntityDepth()
	// Bottom of stack first, matching the order entities were pushed.
	// EntityFetch(0) is the bottom of the stack.
	for i := 0; i < depth; i++ {
		e, err := state.EntityFetch(i)
		if err != nil || e == nil {
			continue
		}
		fmt.Fprintf(w, "\t<entity name=\"%s\" id=\"%d\">\n",
			escapeXML(e.GetName().StringValue()), e.GetID())
		names := e.GetAttributeNames()
		sorted := make([]string, 0, len(names))
		byName := make(map[string]*dtrules.RName, len(names))
		for _, n := range names {
			s := n.StringValue()
			sorted = append(sorted, s)
			byName[s] = n
		}
		sort.Strings(sorted)
		for _, s := range sorted {
			if s == "" {
				continue
			}
			v, err := e.Get(byName[s])
			val := ""
			if err == nil && v != nil {
				val = v.StringValue()
			}
			fmt.Fprintf(w, "\t\t<attr name=\"%s\">%s</attr>\n", escapeXML(s), escapeXML(val))
		}
		fmt.Fprintln(w, "\t</entity>")
	}
	fmt.Fprintln(w, "</finalState>")
}

// WriteFooter closes the DTRulesTrace document.
func WriteFooter(w io.Writer) {
	fmt.Fprintln(w, "</DTRulesTrace>")
}

// Provenance returns the provenance recorded in a loaded trace's root
// element. Fields are empty for traces that predate provenance recording.
func (t *Trace) Provenance() Provenance {
	if t.root == nil {
		return Provenance{}
	}
	return Provenance{
		DTRulesVersion:   t.root.Attributes["dtrules_version"],
		RulesFingerprint: t.root.Attributes["rules_fingerprint"],
	}
}

// FinalState returns the trace's recorded <finalState> node, or nil for
// traces that did not record one.
func (t *Trace) FinalState() *TraceNode {
	if t.root == nil {
		return nil
	}
	for _, c := range t.root.Children {
		if c.Name == "finalState" {
			return c
		}
	}
	return nil
}

// VerifyFinalState compares a state (typically one reconstructed by replay)
// against the trace's recorded final state, entity by entity and attribute
// by attribute. It returns a description of every mismatch; an empty result
// means the replay faithfully reproduced the recorded end state. The
// debugger runs this on every loaded trace.
func (t *Trace) VerifyFinalState(state StackState) []string {
	fs := t.FinalState()
	if fs == nil {
		return []string{"trace records no finalState"}
	}

	var mismatches []string
	depth := state.EntityDepth()
	if depth != len(fs.Children) {
		mismatches = append(mismatches, fmt.Sprintf(
			"entity stack depth: replay has %d, trace recorded %d", depth, len(fs.Children)))
	}

	n := depth
	if len(fs.Children) < n {
		n = len(fs.Children)
	}
	for i := 0; i < n; i++ {
		recorded := fs.Children[i]
		e, err := state.EntityFetch(i)
		if err != nil || e == nil {
			mismatches = append(mismatches, fmt.Sprintf("stack[%d]: missing in replay", i))
			continue
		}
		name := recorded.Attributes["name"]
		if got := e.GetName().StringValue(); got != name {
			mismatches = append(mismatches, fmt.Sprintf(
				"stack[%d]: entity %q in replay, %q recorded", i, got, name))
			continue
		}
		for _, attr := range recorded.Children {
			if attr.Name != "attr" {
				continue
			}
			attrName := attr.Attributes["name"]
			if attrName == "" {
				continue
			}
			v, err := e.Get(dtrules.GetRName(attrName))
			got := ""
			if err == nil && v != nil {
				got = v.StringValue()
			}
			if got != attr.Body {
				mismatches = append(mismatches, fmt.Sprintf(
					"%s.%s: replay %q, trace recorded %q", name, attrName, got, attr.Body))
			}
		}
	}
	return mismatches
}

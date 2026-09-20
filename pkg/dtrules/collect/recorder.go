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

package collect

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// Recorder is a non-blocking Collector (#1210). Instead of asking a human for
// a reached, uncollected `collect` field, it records the question and lets the
// field's current (default) value stand, so execution runs to completion
// without prompting.
//
// The run's result is therefore *provisional*: it was computed from defaults
// nobody confirmed. A caller answers the recorded questions, writes them into
// a canonical data file, and re-runs with that data. Because defaults steer
// the branches taken, the set of questions reached can change once real
// answers arrive — so a caller loops until Pending() comes back empty. That is
// inherent to a path-dependent interview, not a defect.
//
// Recorder implements dtrules.Collector directly, so it attaches with
// SetCollector(collect.NewRecorder()). It also implements Asker, so it can be
// handed to New if a caller wants the plain Collector wrapper.
type Recorder struct {
	inner   *Collector
	pending []Pending
	seen    map[string]bool
	// instance is the id of the entity whose field is currently being
	// collected, captured in MaybeCollect and read back in Ask.
	instance int
}

// NewRecorder returns a Recorder ready to attach via SetCollector.
func NewRecorder() *Recorder {
	r := &Recorder{seen: map[string]bool{}}
	r.inner = New(r)
	return r
}

// PendingOption is one choice of a multiple_choice question, as published.
type PendingOption struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// Pending is one unanswered collect field the run reached — the publishable
// form of a Request, plus the default that was substituted for it.
type Pending struct {
	Entity       string          `json:"entity"`
	Instance     int             `json:"instance"`
	Field        string          `json:"field"`
	QuestionText string          `json:"question_text,omitempty"`
	QuestionType string          `json:"question_type,omitempty"`
	Options      []PendingOption `json:"options,omitempty"`
	RefLow       string          `json:"ref_low,omitempty"`
	RefHigh      string          `json:"ref_high,omitempty"`
	Units        string          `json:"units,omitempty"`
	Default      string          `json:"default"`
}

// MaybeCollect implements dtrules.Collector. It delegates the collect-field
// test, the collected-marking and the tracking to the ordinary Collector; all
// it adds is the identity of the instance being read, which a Request does not
// carry and two instances of one entity type need to be told apart (#1210).
func (r *Recorder) MaybeCollect(e dtrules.Entity, attr *dtrules.RName) error {
	prev := r.instance
	if re, ok := e.(*entity.REntity); ok && re != nil {
		r.instance = re.GetID()
	}
	err := r.inner.MaybeCollect(e, attr)
	r.instance = prev
	return err
}

// Ask implements Asker. It never blocks: it records the question and answers
// "no answer — keep the default", which leaves the field marked collected so
// it is recorded once however often it is read.
func (r *Recorder) Ask(req Request) (dtrules.Object, bool, error) {
	def := ""
	if req.Current != nil {
		def = req.Current.StringValue()
	}
	key := strings.ToLower(req.Entity) + "\x00" + strings.ToLower(req.Field)
	key += "\x00" + strconv.Itoa(r.instance)
	if r.seen[key] {
		return nil, false, nil
	}
	r.seen[key] = true

	p := Pending{
		Entity:       req.Entity,
		Instance:     r.instance,
		Field:        req.Field,
		QuestionText: req.Text,
		QuestionType: req.QType,
		RefLow:       req.RefLow,
		RefHigh:      req.RefHigh,
		Units:        req.Units,
		Default:      def,
	}
	for _, o := range req.Options {
		p.Options = append(p.Options, PendingOption{Value: o.Value, Label: o.Label})
	}
	r.pending = append(r.pending, p)
	return nil, false, nil
}

// Pending returns the questions recorded so far, in the order reached. The
// slice is never nil, so it marshals as `[]` rather than `null`.
func (r *Recorder) Pending() []Pending {
	if r.pending == nil {
		return []Pending{}
	}
	return r.pending
}

// WritePending writes the recorded questions to w as a JSON array. An empty
// recording writes `[]` — the file is the answer "nothing is pending", not an
// absence.
func (r *Recorder) WritePending(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r.Pending())
}

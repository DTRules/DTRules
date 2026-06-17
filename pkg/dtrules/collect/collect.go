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

// Package collect implements interactive data collection for DTRules (#850/
// #852). It bridges the runtime's dtrules.Collector seam to a pluggable Asker
// — the CLI prompt, the web form, or a test stub — so the interpreter stays
// free of any UI dependency.
//
// Attach a Collector to a DTState (via SetCollector) to turn on interactive
// mode: when execution reads a `collect` field whose value has not yet been
// collected on that entity instance, the Collector asks the Asker for it,
// writes it, and marks it collected. With no Collector attached, execution is
// pure batch and behaves exactly as today.
package collect

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// Option is one choice presented for a multiple_choice question.
type Option struct {
	Value string
	Label string
}

// Request is everything the Asker needs to ask for one field's value. It is
// assembled at the read point, so it carries the live entity/field identity
// and the field's current (default or prior) value as the pre-fill.
type Request struct {
	Entity  string         // entity instance's type name (e.g. "patient")
	Field   string         // attribute name (e.g. "age")
	Text    string         // question prompt
	QType   string         // multiple_choice | ascii | number | date
	Options []Option       // choices, for multiple_choice
	Current dtrules.Object // current/default value (the pre-fill)

	// Reference range for a number question (lab-report style, #850): the
	// expected ("normal") band and a unit label. Shown as guidance; the
	// entered value is flagged High/Low but never rejected. Either bound may
	// be empty (one-sided).
	RefLow  string
	RefHigh string
	Units   string
}

// HasRange reports whether the request carries a reference range to display.
func (r Request) HasRange() bool { return r.RefLow != "" || r.RefHigh != "" }

// Asker obtains a value for one collect field. ok=false means "no answer —
// keep the current/default value" (the field is still marked collected so it
// is not asked again). A non-nil error aborts execution.
//
// The returned value may be any dtrules.Object the field's type accepts;
// entity.Put coerces strings (RString) to the declared type, so an Asker can
// simply return the raw text as an RString.
type Asker interface {
	Ask(req Request) (value dtrules.Object, ok bool, err error)
}

// AskerFunc adapts a function to the Asker interface.
type AskerFunc func(req Request) (dtrules.Object, bool, error)

// Ask calls f.
func (f AskerFunc) Ask(req Request) (dtrules.Object, bool, error) { return f(req) }

// Collector implements dtrules.Collector by delegating each collect field to
// an Asker.
type Collector struct {
	asker Asker
}

// New returns a Collector that asks a for each reached, uncollected collect
// field.
func New(a Asker) *Collector { return &Collector{asker: a} }

// MaybeCollect implements dtrules.Collector. It is a no-op unless e is a
// reference entity whose attribute attr is a `collect` field that has not yet
// been collected on this instance.
func (c *Collector) MaybeCollect(e dtrules.Entity, attr *dtrules.RName) error {
	re, ok := e.(*entity.REntity)
	if !ok || re == nil || c.asker == nil {
		return nil
	}
	entry := re.GetEntry(attr)
	if entry == nil || !entry.Collect {
		return nil
	}
	// First touch of a collect field on this instance turns on tracking.
	re.EnableCollectTracking()
	if re.IsCollected(attr) {
		return nil
	}

	cur, _ := re.Get(attr)
	req := Request{
		Entity:  re.GetName().GetName(),
		Field:   attr.GetName(),
		Current: cur,
	}
	if entry.Question != nil {
		req.Text = entry.Question.Text
		req.QType = entry.Question.Type
		req.RefLow = entry.Question.RefLow
		req.RefHigh = entry.Question.RefHigh
		req.Units = entry.Question.Units
		for _, o := range entry.Question.Options {
			req.Options = append(req.Options, Option{Value: o.Value, Label: o.Label})
		}
	}

	value, answered, err := c.asker.Ask(req)
	if err != nil {
		return err
	}
	if answered && value != nil {
		if err := re.Put(attr, value); err != nil {
			return err
		}
	}
	// Mark collected even when the Asker declined (kept the default), so the
	// field is never asked twice.
	re.MarkCollected(attr)
	return nil
}

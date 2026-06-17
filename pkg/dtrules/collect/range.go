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
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// Flag compares a numeric value against a reference range and returns "H" when
// it is above the high bound, "L" when below the low bound, or "" when it is
// within range, the bounds are open, or the value/bounds are not numeric.
// Following lab convention, an out-of-range value is flagged, never rejected.
func Flag(value, low, high string) string {
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return ""
	}
	if high != "" {
		if h, err := strconv.ParseFloat(strings.TrimSpace(high), 64); err == nil && v > h {
			return "H"
		}
	}
	if low != "" {
		if l, err := strconv.ParseFloat(strings.TrimSpace(low), 64); err == nil && v < l {
			return "L"
		}
	}
	return ""
}

// RangeText renders a reference range for display, e.g. "0.7–1.3 mg/dL",
// "≤ 200 mg/dL", or "≥ 60". Returns "" when no bounds are set.
func RangeText(low, high, units string) string {
	u := ""
	if units != "" {
		u = " " + units
	}
	switch {
	case low != "" && high != "":
		return low + "–" + high + u
	case high != "":
		return "≤ " + high + u
	case low != "":
		return "≥ " + low + u
	default:
		return ""
	}
}

// Reading is one collected, range-bearing value with its lab-style flag.
type Reading struct {
	Entity string
	Field  string
	Value  string
	Units  string
	Low    string
	High   string
	Flag   string // "", "H", or "L"
}

// RangedReadings scans the given entities for collect fields that carry a
// reference range and have been collected, returning each as a Reading with
// its High/Low flag — the data a front-end needs to report results lab-style.
func RangedReadings(entities []*entity.REntity) []Reading {
	var out []Reading
	for _, e := range entities {
		if e == nil {
			continue
		}
		for _, attr := range e.GetAttributeNames() {
			entry := e.GetEntry(attr)
			if entry == nil || !entry.Collect || entry.Question == nil {
				continue
			}
			q := entry.Question
			if q.RefLow == "" && q.RefHigh == "" {
				continue
			}
			if !e.IsCollected(attr) {
				continue
			}
			val := ""
			if v, _ := e.Get(attr); v != nil {
				val = v.StringValue()
			}
			out = append(out, Reading{
				Entity: e.GetName().GetName(),
				Field:  attr.GetName(),
				Value:  val,
				Units:  q.Units,
				Low:    q.RefLow,
				High:   q.RefHigh,
				Flag:   Flag(val, q.RefLow, q.RefHigh),
			})
		}
	}
	return out
}

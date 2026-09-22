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

package dtrules_test

import (
	"testing"
	"time"
)

// `current time` is the current instant, in UTC (#1266). `current date` is today at
// midnight UTC; before this there was no EL word for the instant except the
// `current date in zone "UTC"` idiom, where `in zone` changed the meaning
// and not just the zone.
func TestCurrentTimeIsTheInstant(t *testing.T) {
	before := time.Now().UTC()
	r := newTimeRig(t)
	r.exec("set out = now")
	after := time.Now().UTC()

	got := r.date("out")
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("current time = %s, want between %s and %s", got, before, after)
	}
	if name, offset := got.Zone(); name != "UTC" || offset != 0 {
		t.Errorf("current time carries zone %s (offset %d), want UTC: a rule must not answer differently on a server in another zone", name, offset)
	}
}

// The scheduler case from #1266: how long since something happened. Before
// `current time` this had to be written `current date in zone "UTC"`, and writing
// `current date` instead silently measured from midnight.
func TestCurrentTimeMeasuresElapsed(t *testing.T) {
	r := newTimeRig(t)
	r.put("d1", time.Now().UTC().Add(-90*time.Second))
	r.exec("set n = seconds from d1 to current time")
	if got := r.int("n"); got < 85 || got > 120 {
		t.Errorf("seconds from 90s ago to now = %d, want about 90", got)
	}

	// `current date` is midnight, so it is not a stand-in for the instant.
	r.exec("set out = current date")
	if h, m, s := r.date("out").Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("current date = %s, want midnight", r.date("out"))
	}
}

// `now in zone <s>` is the same instant stamped with that zone: the generic
// rewrap, unlike `today in zone <s>`, which is midnight in that zone (#1273).
func TestCurrentTimeInZoneKeepsTheInstant(t *testing.T) {
	r := newTimeRig(t)
	r.exec("set d1 = now")
	r.exec(`set out = current time in zone "America/Chicago"`)

	utc, chicago := r.date("d1"), r.date("out")
	if d := chicago.Sub(utc); d < -5*time.Second || d > 5*time.Second {
		t.Errorf("current time in zone differs by %s, want the same instant", d)
	}
	if name, _ := chicago.Zone(); name != "CST" && name != "CDT" {
		t.Errorf("now in zone \"America/Chicago\" carries zone %s, want CST or CDT", name)
	}
}

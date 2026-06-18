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

// Package interview is the seam between a user interface and a set of decision
// tables (#850). A UI drives a Runner: it supplies an Asker (which answers each
// reached collect field) and receives a Result. The two halves are decoupled —
//
//   - UI side:    implement collect.Asker (ask the user) and render a Result.
//   - Engine side: implement Runner (run the rules, ask, return a Result).
//
// The default engine is Project, which loads a rule-set directory and runs an
// entry table. The default UI is pkg/dtrules/web. Future projects can supply
// their own Runner, and future UIs can consume any Runner — neither needs to
// know about the other, and neither changes the engine.
package interview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/datafile"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Result is the outcome of one interview run, in UI-neutral form.
type Result struct {
	Title    string            // the output entity's name
	Fields   []Field           // scalar outputs
	Lists    []List            // array outputs (warnings, rationale, ...)
	Readings []collect.Reading // collected range-bearing values, lab-report style
	DataXML  string            // canonical data (inputs + results) for save/review
	Error    string            // non-empty if the run failed
}

// Field is one scalar output value.
type Field struct{ Name, Value string }

// List is one array output.
type List struct {
	Name  string
	Items []string
}

// Runner runs one interview to completion. It executes the decision tables,
// calling asker for each reached collect field that has not been supplied, and
// returns the rendered Result. reviewData, when non-empty, is canonical data
// XML from a prior run, loaded in Review mode — so every collect field is
// re-asked pre-filled with the prior answer.
type Runner interface {
	Run(asker collect.Asker, reviewData string) (*Result, error)
}

// RunnerFunc adapts a function to the Runner interface.
type RunnerFunc func(asker collect.Asker, reviewData string) (*Result, error)

// Run calls f.
func (f RunnerFunc) Run(asker collect.Asker, reviewData string) (*Result, error) {
	return f(asker, reviewData)
}

// Options configures the default Project runner.
type Options struct {
	Entry        string // decision table to run (required)
	Input        string // optional input data file (loaded via the mapping)
	ResultEntity string // output entity to render (default "result")
}

// Project returns the default Runner: it loads the rule set in xmlDir, runs the
// entry table (optionally pre-loaded with input or review data), and renders
// the result entity. Each Run uses a fresh session, so runs are independent.
func Project(xmlDir string, opts Options) Runner {
	if opts.ResultEntity == "" {
		opts.ResultEntity = "result"
	}
	return RunnerFunc(func(asker collect.Asker, reviewData string) (*Result, error) {
		return runDir(xmlDir, opts, asker, reviewData)
	})
}

func runDir(xmlDir string, opts Options, asker collect.Asker, reviewData string) (*Result, error) {
	rs := session.NewRuleSet(filepath.Base(xmlDir))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		return nil, err
	}
	sess, err := rs.NewSession()
	if err != nil {
		return nil, err
	}
	if err := initMapping(sess, xmlDir, opts.Input); err != nil {
		return nil, err
	}
	state := sess.GetState()
	// Review mode: pre-load the prior run's answers as defaulted (not
	// collected), so each collect field is re-asked pre-filled (#853).
	if reviewData != "" {
		if err := loadReview(state, sess, reviewData); err != nil {
			return nil, err
		}
	}
	if dts, ok := state.(*interpreter.DTState); ok {
		dts.SetCollector(collect.New(asker))
	}
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(opts.Entry))
	if err != nil {
		return nil, err
	}
	if dt == nil {
		return nil, fmt.Errorf("entry table %q not found", opts.Entry)
	}
	if err := dt.Execute(state); err != nil {
		return nil, err
	}
	res := buildResult(state, opts.ResultEntity)
	entities := stackEntities(state)
	res.Readings = collect.RangedReadings(entities)
	var data strings.Builder
	if err := datafile.Write(&data, entities); err != nil {
		return nil, fmt.Errorf("serializing collected data: %w", err)
	}
	res.DataXML = data.String()
	return res, nil
}

// loadReview loads canonical data into the live stack instances in Review mode
// (values set, but left defaulted so they are re-asked pre-filled).
func loadReview(state dtrules.State, sess dtrules.Session, data string) error {
	find := func(name string) *entity.REntity {
		e, err := state.FindEntity(dtrules.GetRName(name))
		if err != nil || e == nil {
			return nil
		}
		re, _ := e.(*entity.REntity)
		return re
	}
	create := func(subtype string) (*entity.REntity, error) {
		e, err := sess.CreateEntity(dtrules.GetRName(subtype))
		if err != nil {
			return nil, err
		}
		re, _ := e.(*entity.REntity)
		return re, nil
	}
	return datafile.Read(strings.NewReader(data), find, create, datafile.Review)
}

// stackEntities returns the distinct data entities on the state's entity stack
// (the executed instances), excluding the operator "primitives" entity.
func stackEntities(state dtrules.State) []*entity.REntity {
	dts, ok := state.(*interpreter.DTState)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []*entity.REntity
	for i := 0; i < dts.EntityDepth(); i++ {
		e, err := dts.GetEntityStack(i)
		if err != nil {
			continue
		}
		re, ok := e.(*entity.REntity)
		if !ok {
			continue
		}
		name := re.GetName().GetName()
		if name == "primitives" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, re)
	}
	return out
}

func initMapping(sess dtrules.Session, xmlDir, input string) error {
	maps, _ := filepath.Glob(filepath.Join(xmlDir, "*_map.xml"))
	if len(maps) == 0 {
		return nil
	}
	mf, err := os.Open(maps[0])
	if err != nil {
		return err
	}
	defer mf.Close()
	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mf); err != nil {
		return err
	}
	if err := m.Initialize(); err != nil {
		return err
	}
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := m.LoadData(f); err != nil {
			return err
		}
	}
	return nil
}

// buildResult collects the output entity's fields into a renderable Result.
func buildResult(state dtrules.State, entityName string) *Result {
	res := &Result{Title: entityName}
	e, err := state.FindEntity(dtrules.GetRName(entityName))
	if err != nil || e == nil {
		return res
	}
	re, ok := e.(*entity.REntity)
	if !ok {
		return res
	}
	for _, attr := range re.GetAttributeNames() {
		name := attr.GetName()
		if name == entityName || name == "mapping*key" {
			continue
		}
		v, err := re.Get(attr)
		if err != nil || v == nil || v.Type() == dtrules.TypeNull {
			continue
		}
		if arr, err := v.ArrayValue(); err == nil {
			if len(arr) == 0 {
				continue
			}
			items := make([]string, 0, len(arr))
			for _, el := range arr {
				items = append(items, el.StringValue())
			}
			res.Lists = append(res.Lists, List{Name: name, Items: items})
			continue
		}
		res.Fields = append(res.Fields, Field{Name: name, Value: v.StringValue()})
	}
	return res
}

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

package web

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Options configures ServeDir.
type Options struct {
	Entry        string // decision table to run (required)
	Input        string // optional input data file (loaded via the mapping)
	ResultEntity string // output entity to render (default "result")
	NoOpen       bool   // suppress auto-opening the browser
}

// ServeDir serves the rule set in xmlDir as an interactive web interview on
// addr. Each browser session loads a fresh copy and runs the entry table; any
// reached collect field that isn't supplied is asked via the web form. Unless
// NoOpen is set, the user's browser is opened once the listener is bound.
func ServeDir(addr, xmlDir string, opts Options) error {
	if opts.ResultEntity == "" {
		opts.ResultEntity = "result"
	}
	run := func(asker collect.Asker) (*Result, error) {
		return runDir(xmlDir, opts, asker)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := browserURL(ln.Addr())
	fmt.Printf("\n  DTRules web interview ready:  %s\n  (Ctrl-C to stop)\n\n", url)
	if !opts.NoOpen {
		go func() {
			if err := openBrowser(url); err != nil {
				fmt.Printf("  Could not auto-open a browser (%v). Open the URL above manually.\n", err)
			}
		}()
	}
	return http.Serve(ln, NewServer(run))
}

func runDir(xmlDir string, opts Options, asker collect.Asker) (*Result, error) {
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
	if dts, ok := state.(*interpreter.DTState); ok {
		dts.SetCollector(collect.New(asker))
	}
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(opts.Entry))
	if err != nil || dt == nil {
		return nil, err
	}
	if err := dt.Execute(state); err != nil {
		return nil, err
	}
	res := buildResult(state, opts.ResultEntity)
	res.Readings = collect.RangedReadings(stackEntities(state))
	return res, nil
}

// stackEntities returns the distinct data entities on the state's entity
// stack (the executed instances), excluding the operator "primitives" entity.
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

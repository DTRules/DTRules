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

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/datafile"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
	webpkg "github.com/DTRules/DTRules/pkg/dtrules/web"
)

// runRun handles `dtrules run [path] --entry <table> [--input f] [--interactive]`.
// It loads a project, runs a decision table, and prints the result entity.
// With --interactive, a CLI prompt collects any reached `collect` field whose
// value hasn't been provided (#850/#854).
func (c *CLI) runRun(args []string) int {
	path, entry, input, resultEntity := ".", "", "", "result"
	var save, data, review string
	interactive, web, noOpen := false, false, false
	port := "0" // 0 = let the OS pick a free port
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--entry":
			if i+1 < len(args) {
				entry = args[i+1]
				i++
			}
		case "--input":
			if i+1 < len(args) {
				input = args[i+1]
				i++
			}
		case "--save":
			if i+1 < len(args) {
				save = args[i+1]
				i++
			}
		case "--data":
			if i+1 < len(args) {
				data = args[i+1]
				i++
			}
		case "--review":
			if i+1 < len(args) {
				review = args[i+1]
				i++
			}
		case "--result-entity":
			if i+1 < len(args) {
				resultEntity = args[i+1]
				i++
			}
		case "--web":
			web = true
		case "--no-open":
			noOpen = true
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--interactive", "-i":
			interactive = true
		case "-h", "--help":
			c.printRunUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				c.printRunUsage()
				return 1
			}
			path = args[i]
		}
	}
	if entry == "" {
		fmt.Fprintln(os.Stderr, "Error: --entry <table> is required")
		c.printRunUsage()
		return 1
	}

	xmlDir, _, err := resolveDirs(mustAbs(path), "", "")
	if err != nil || !dirExists(xmlDir) {
		fmt.Fprintf(os.Stderr, "Error: could not find xml/ directory under %s\n", path)
		return 1
	}

	if web {
		addr := ":" + port
		if err := webpkg.ServeDir(addr, xmlDir, webpkg.Options{Entry: entry, Input: input, ResultEntity: resultEntity, NoOpen: noOpen}); err != nil {
			fmt.Fprintf(os.Stderr, "Error serving: %v\n", err)
			return 1
		}
		return 0
	}

	rs := session.NewRuleSet(filepath.Base(mustAbs(path)))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		return 1
	}
	sess, err := rs.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		return 1
	}

	// Initialize entities (and optionally load input data) via the mapping.
	if err := initMapping(sess, xmlDir, input); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing data: %v\n", err)
		return 1
	}

	state := sess.GetState()

	// Canonical data load (mapping-free): --data is authoritative (batch),
	// --review pre-fills for re-interview. Loaded into the live stack
	// instances so execution sees the values.
	if data != "" || review != "" {
		mode := datafile.Authoritative
		file := data
		if review != "" {
			mode, file = datafile.Review, review
		}
		if err := loadCanonical(state, sess, file, mode); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading data %q: %v\n", file, err)
			return 1
		}
	}

	if interactive {
		if dts, ok := state.(*interpreter.DTState); ok {
			dts.SetCollector(collect.New(newCLIAsker()))
		} else {
			fmt.Fprintln(os.Stderr, "Warning: interactive collection unavailable for this state")
		}
	}

	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))
	if err != nil || dt == nil {
		fmt.Fprintf(os.Stderr, "Error: decision table %q not found\n", entry)
		return 1
	}
	if err := dt.Execute(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %q: %v\n", entry, err)
		return 1
	}

	// Save the collected dataset as canonical, mapping-free data XML — an
	// audit record and a replay/review fixture.
	if save != "" {
		f, err := os.Create(save)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %q: %v\n", save, err)
			return 1
		}
		defer f.Close()
		if err := datafile.Write(f, dataEntities(state, rs)); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving data: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "saved data to %s\n", save)
	}

	renderResult(state, resultEntity)
	return 0
}

// initMapping loads the project's *_map.xml (if any), initializes the entity
// stack, and loads input data when a file is given.
func initMapping(sess dtrules.Session, xmlDir, input string) error {
	maps, _ := filepath.Glob(filepath.Join(xmlDir, "*_map.xml"))
	if len(maps) == 0 {
		return nil // no mapping; rely on whatever the session set up
	}
	mapFile, err := os.Open(maps[0])
	if err != nil {
		return err
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
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

// dataEntities returns the live, executed data entities (the instances on the
// state's entity stack), not the factory's templates — so a save captures
// collected/computed values. The operator "primitives" entity is excluded.
func dataEntities(state dtrules.State, rs *session.RuleSet) []*entity.REntity {
	var out []*entity.REntity
	seen := map[string]bool{}
	for _, ref := range rs.GetEntityFactory().GetRefEntities() {
		name := ref.GetName().GetName()
		if name == "primitives" || seen[name] {
			continue
		}
		inst, err := state.FindEntity(dtrules.GetRName(name))
		if err != nil || inst == nil {
			continue // not on the stack (e.g. an array-element template)
		}
		if re, ok := inst.(*entity.REntity); ok {
			seen[name] = true
			out = append(out, re)
		}
	}
	return out
}

// loadCanonical loads a canonical (mapping-free) data file into the live
// entity-stack instances (so execution sees the values), in the given mode.
func loadCanonical(state dtrules.State, sess dtrules.Session, file string, mode datafile.LoadMode) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
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
	return datafile.Read(f, find, create, mode)
}

// renderResult prints the named output entity's fields after execution.
func renderResult(state dtrules.State, entityName string) {
	result, err := state.FindEntity(dtrules.GetRName(entityName))
	if err != nil || result == nil {
		fmt.Printf("\n(no %q entity to display)\n", entityName)
		return
	}
	fmt.Printf("\n=== %s ===\n", entityName)
	for _, attr := range result.GetAttributeNames() {
		name := attr.GetName()
		if name == entityName || name == "mapping*key" {
			continue
		}
		v, err := result.Get(attr)
		if err != nil || v == nil {
			continue
		}
		if arr, err := v.ArrayValue(); err == nil {
			if len(arr) == 0 {
				continue
			}
			fmt.Printf("  %s:\n", name)
			for _, e := range arr {
				fmt.Printf("    - %s\n", e.StringValue())
			}
			continue
		}
		fmt.Printf("  %-22s %s\n", name+":", v.StringValue())
	}
}

func (c *CLI) printRunUsage() {
	fmt.Println(`Usage: dtrules run [path] --entry <table> [options]

Loads a project, runs a decision table, and prints the result entity.

Options:
  --entry <table>        Decision table to run (required)
  --input <file.xml>     Input data to load via the project mapping
  --data <file.xml>      Load canonical (mapping-free) data, authoritative
  --review <file.xml>    Load canonical data for re-interview (pre-filled, asked)
  --save <file.xml>      Save the collected data as canonical XML after the run
  --interactive, -i      Prompt for any reached collect field not supplied
  --web                  Serve an interactive web interview instead of a CLI run
  --port <n>             Port for --web (default: an unused port chosen by the OS)
  --no-open              Do not auto-open the browser with --web
  --result-entity <name> Output entity to print (default: result)

The closed loop: collect with --interactive --save, replay with --data, edit
with --review --interactive.

Examples:
  dtrules run ./sampleprojects/SinusitisTherapy --entry Determine_Therapy --interactive --save case.xml
  dtrules run . --entry Determine_Therapy --data case.xml
  dtrules run . --entry Determine_Therapy --review case.xml --interactive`)
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// cliAsker prompts for collect-field values on the terminal.
type cliAsker struct {
	in *bufio.Reader
}

func newCLIAsker() *cliAsker { return &cliAsker{in: bufio.NewReader(os.Stdin)} }

// Ask implements collect.Asker. An empty line keeps the current default.
func (a *cliAsker) Ask(req collect.Request) (dtrules.Object, bool, error) {
	cur := ""
	if req.Current != nil {
		cur = req.Current.StringValue()
	}
	prompt := req.Text
	if prompt == "" {
		prompt = fmt.Sprintf("%s.%s", req.Entity, req.Field)
	}
	fmt.Printf("\n%s\n", prompt)

	if req.QType == "multiple_choice" {
		for i, o := range req.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			fmt.Printf("  %d) %s\n", i+1, label)
		}
		fmt.Printf("  [default %s] > ", cur)
		line, _ := a.in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return nil, false, nil
		}
		if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(req.Options) {
			return dtrules.GetRString(req.Options[n-1].Value), true, nil
		}
		return dtrules.GetRString(line), true, nil // accept a literal value too
	}

	fmt.Printf("  (%s) [default %s] > ", req.QType, cur)
	line, _ := a.in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false, nil
	}
	return dtrules.GetRString(line), true, nil
}

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/project"
)

// `dtrules map` is the authoring surface for mapping files.
//
// It was the one authoring surface with no write API. A mapping decides which
// entity each external tag lands in, and which entities exist on the stack at
// all -- an entity absent from the <entities> block is never created, so every
// rule that reads it fails at run time. The map file in CorporateTax carries a
// comment admitting the gap: "Added by hand because the mapping authoring API
// covers only setattribute entries." Hand-edited, because there was no other
// way (#1103).

// mapJSON is the wire shape: the parts of a mapping an author changes.
type mapJSON struct {
	Name           string             `json:"name,omitempty"`
	Entities       []mapEntityJSON    `json:"entities"`
	Initialization []string           `json:"initialization"`
	CreateEntities []mapCreateJSON    `json:"createEntities,omitempty"`
	Attributes     []mapAttributeJSON `json:"attributes"`
}

type mapEntityJSON struct {
	Name string `json:"name"`
	// Number is cardinality: "1" for a singleton, "*" for many.
	Number string `json:"number"`
}

type mapCreateJSON struct {
	Entity string `json:"entity"`
	Tag    string `json:"tag"`
	ID     string `json:"id,omitempty"`
	List   string `json:"list,omitempty"`
}

type mapAttributeJSON struct {
	Tag        string `json:"tag"`
	RAttribute string `json:"rattribute,omitempty"`
	Enclosure  string `json:"enclosure"`
	Type       string `json:"type,omitempty"`
}

// mapPatchOp is one change. Deliberately one op per invocation, like `edd
// patch`, so a failure names exactly what failed.
type mapPatchOp struct {
	Op        string            `json:"op"`
	Entity    string            `json:"entity,omitempty"`
	Number    string            `json:"number,omitempty"`
	Attribute *mapAttributeJSON `json:"attribute,omitempty"`
	Tag       string            `json:"tag,omitempty"`
}

func (c *CLI) runMap(args []string) int {
	ctx := &tableCmdCtx{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	if len(args) == 0 {
		c.printMapUsage()
		return 1
	}
	sub := args[0]
	rest := args[1:]
	var mapFile string
	ctx.projectPath, mapFile, ctx.forceOverwriteExcel, rest = parseMapFlags(rest)

	switch sub {
	case "get":
		return ctx.mapGet(mapFile)
	case "patch":
		return ctx.mapPatch(mapFile)
	case "help", "-h", "--help":
		c.printMapUsage()
		return 0
	default:
		return emitErr(ctx.stderr, 1, "invalid_command", "", "known: get|patch",
			fmt.Sprintf("unknown map subcommand %q", sub))
	}
}

// parseMapFlags is parseProjectFlags with --map-file instead of --edd-file.
func parseMapFlags(args []string) (projectPath, mapFile string, force bool, rest []string) {
	out := make([]string, 0, len(args))
	projectPath = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--map-file":
			if i+1 < len(args) {
				mapFile = args[i+1]
				i++
				continue
			}
		case "--project", "-p":
			if i+1 < len(args) {
				projectPath = args[i+1]
				i++
				continue
			}
		case "--force-overwrite-excel":
			force = true
			continue
		}
		out = append(out, args[i])
	}
	return projectPath, mapFile, force, out
}

// resolveMapFile finds the mapping to work on. Same rule as EDD files and as
// loading data (#1098): one needs no argument, several must be chosen between.
func (ctx *tableCmdCtx) resolveMapFile(mapFile string) (string, error) {
	xmlDir := projectXMLDirFor(ctx.projectPath)
	if mapFile != "" {
		for _, cand := range []string{mapFile,
			filepath.Join(xmlDir, mapFile), filepath.Join(ctx.projectPath, mapFile)} {
			if info, err := os.Stat(cand); err == nil && !info.IsDir() {
				return cand, nil
			}
		}
		return "", fmt.Errorf("map file %q not found", mapFile)
	}
	var found []string
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_map.xml") {
			found = append(found, p)
		}
		return nil
	})
	sort.Strings(found)
	switch len(found) {
	case 0:
		return "", fmt.Errorf("no *_map.xml file in %s", xmlDir)
	case 1:
		return found[0], nil
	}
	names := make([]string, 0, len(found))
	for _, f := range found {
		names = append(names, filepath.Base(f))
	}
	return "", fmt.Errorf("this project has %d mapping files (%s); say which with --map-file <name>",
		len(found), strings.Join(names, ", "))
}

func (ctx *tableCmdCtx) mapGet(mapFile string) int {
	path, err := ctx.resolveMapFile(mapFile)
	if err != nil {
		return emitErr(ctx.stderr, 1, "ambiguous_map", "", "pass --map-file <name>", err.Error())
	}
	m, err := excel.LoadMapXMLFromFile(path)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return writeJSONOr(ctx, mapToJSON(m))
}

func (ctx *tableCmdCtx) mapPatch(mapFile string) int {
	path, err := ctx.resolveMapFile(mapFile)
	if err != nil {
		return emitErr(ctx.stderr, 1, "ambiguous_map", "", "pass --map-file <name>", err.Error())
	}
	var op mapPatchOp
	if err := json.NewDecoder(ctx.stdin).Decode(&op); err != nil {
		return emitErr(ctx.stderr, 1, "parse_error", "", "patch input must be a JSON object", err.Error())
	}
	m, err := excel.LoadMapXMLFromFile(path)
	if err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	if err := applyMapOp(m, op); err != nil {
		return emitErr(ctx.stderr, 1, "invalid_patch", "", "known ops: add-entity|delete-entity|add-attribute|delete-attribute", err.Error())
	}
	if err := excel.WriteMapXML(m, path); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "save failed", err.Error())
	}
	// Excel is the system of record, so the paired workbook catches up in the
	// same operation -- the same contract every other authoring write obeys.
	xmlDir := projectXMLDirFor(ctx.projectPath)
	if err := refreshMapWorkbook(xmlDir, path); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "the XML was written but its workbook was not", err.Error())
	}
	return writeJSONOr(ctx, map[string]string{"op": op.Op, "status": "patched",
		"file": filepath.Base(path)})
}

// applyMapOp is the whole edit vocabulary.
//
// add-entity does two things, because "this entity exists" is two declarations
// in a mapping: a cardinality entry, and a push onto the initialization stack.
// An entity with only the first is declared and never created, which is a rule
// that fails at run time with "not defined by any Entity on the Entity Stack".
// Making a caller remember both would be an invitation to that bug.
func applyMapOp(m *excel.MapXML, op mapPatchOp) error {
	switch op.Op {
	case "add-entity":
		if op.Entity == "" {
			return fmt.Errorf("add-entity needs an entity name")
		}
		number := op.Number
		if number == "" {
			number = "1"
		}
		for _, e := range m.EntityDecls {
			if strings.EqualFold(e.Name, op.Entity) {
				return fmt.Errorf("entity %q is already declared", op.Entity)
			}
		}
		m.EntityDecls = append(m.EntityDecls, excel.MapEntityDecl{Name: op.Entity, Number: number})
		if number == "1" {
			m.InitialEntities = append(m.InitialEntities,
				excel.MapInitialEntity{Entity: op.Entity, EPush: true})
		}
		return nil

	case "delete-entity":
		found := false
		decls := m.EntityDecls[:0]
		for _, e := range m.EntityDecls {
			if strings.EqualFold(e.Name, op.Entity) {
				found = true
				continue
			}
			decls = append(decls, e)
		}
		if !found {
			return fmt.Errorf("entity %q is not declared", op.Entity)
		}
		m.EntityDecls = decls
		inits := m.InitialEntities[:0]
		for _, e := range m.InitialEntities {
			if !strings.EqualFold(e.Entity, op.Entity) {
				inits = append(inits, e)
			}
		}
		m.InitialEntities = inits
		return nil

	case "add-attribute":
		a := op.Attribute
		if a == nil || a.Tag == "" || a.Enclosure == "" {
			return fmt.Errorf("add-attribute needs an attribute with a tag and an enclosure")
		}
		ra := a.RAttribute
		if ra == "" {
			ra = a.Tag
		}
		m.Entries = append(m.Entries, excel.MapEntry{
			Tag: a.Tag, RAttribute: ra, Enclosure: a.Enclosure, Type: a.Type})
		return nil

	case "delete-attribute":
		found := false
		out := m.Entries[:0]
		for _, e := range m.Entries {
			if !e.IsSection && strings.EqualFold(e.Tag, op.Tag) &&
				(op.Entity == "" || strings.EqualFold(e.Enclosure, op.Entity)) {
				found = true
				continue
			}
			out = append(out, e)
		}
		if !found {
			return fmt.Errorf("no attribute with tag %q", op.Tag)
		}
		m.Entries = out
		return nil
	}
	return fmt.Errorf("unknown op %q", op.Op)
}

func mapToJSON(m *excel.MapXML) mapJSON {
	out := mapJSON{Name: m.MapName}
	for _, e := range m.EntityDecls {
		out.Entities = append(out.Entities, mapEntityJSON{Name: e.Name, Number: e.Number})
	}
	for _, e := range m.InitialEntities {
		out.Initialization = append(out.Initialization, e.Entity)
	}
	for _, c := range m.CreateEntities {
		out.CreateEntities = append(out.CreateEntities,
			mapCreateJSON{Entity: c.Entity, Tag: c.Tag, ID: c.ID, List: c.List})
	}
	for _, e := range m.Entries {
		if e.IsSection {
			continue
		}
		out.Attributes = append(out.Attributes, mapAttributeJSON{
			Tag: e.Tag, RAttribute: e.RAttribute, Enclosure: e.Enclosure, Type: e.Type})
	}
	return out
}

func writeJSONOr(ctx *tableCmdCtx, v any) int {
	if err := writeJSON(ctx.stdout, v); err != nil {
		return emitErr(ctx.stderr, 1, "io_error", "", "", err.Error())
	}
	return 0
}

func (c *CLI) printMapUsage() {
	fmt.Println(`Usage: dtrules map <command> [--project <path>] [--map-file <name>]

Commands:
  get     Print the mapping as JSON.
  patch   Apply one change from JSON on stdin.

Ops:
  {"op":"add-entity","entity":"nv_result","number":"1"}
  {"op":"delete-entity","entity":"nv_result"}
  {"op":"add-attribute","attribute":{"tag":"gross_revenue","enclosure":"nv_result","type":"double"}}
  {"op":"delete-attribute","tag":"gross_revenue","entity":"nv_result"}

A mapping decides which entity each external tag lands in, and which entities
exist on the stack at all. add-entity with number "1" also pushes the entity in
the initialization stack, because an entity that is declared and never pushed
fails at run time.

Writes the XML and refreshes the paired _map.xlsx, like every authoring write.`)
}

// projectXMLDirFor resolves a project's rules directory the same way every
// other surface does.
func projectXMLDirFor(projectPath string) string {
	if cfg := project.Load(projectPath); dirExists(cfg.XMLDir) {
		return cfg.XMLDir
	}
	if xd := filepath.Join(projectPath, "xml"); dirExists(xd) {
		return xd
	}
	return projectPath
}

// refreshMapWorkbook regenerates the _map.xlsx paired with a _map.xml.
func refreshMapWorkbook(xmlDir, mapXMLPath string) error {
	m, err := excel.LoadMapXMLFromFile(mapXMLPath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(xmlDir, mapXMLPath)
	if err != nil {
		rel = filepath.Base(mapXMLPath)
	}
	excelDir := project.Load(filepath.Dir(xmlDir)).ExcelDir
	if !dirExists(excelDir) {
		excelDir = filepath.Join(filepath.Dir(xmlDir), "excel")
	}
	if !dirExists(excelDir) {
		return nil // no workbooks in this project; nothing to keep in step
	}
	out := filepath.Join(excelDir, strings.TrimSuffix(rel, ".xml")+".xlsx")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return excel.NewMapExporter().ExportToFile(m, out)
}

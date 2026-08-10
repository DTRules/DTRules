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

// Package excel provides Excel import/export functionality for DTRules.
package excel

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// DTImporter imports decision tables from Excel files.
type DTImporter struct {
	verbose    bool
	compileEL  bool              // If true, compile EL expressions to postfix
	elCompiler ELCompiler        // Optional EL compiler
	symbols    map[string]string // Symbol table for EL compilation
	stats      *ImportStats      // Accumulated import statistics (nil = not collecting)
}

// ELCompiler defines the interface for compiling EL expressions to postfix.
// This is implemented by the el.Compiler in pkg/dtrules/compiler/el.
type ELCompiler interface {
	// SetSymbols sets the symbol table for type resolution.
	SetSymbols(symbols map[string]string)
	// CompileCondition compiles a condition expression to postfix.
	CompileCondition(el string) (string, error)
	// CompileAction compiles an action statement to postfix.
	CompileAction(el string) (string, error)
	// CompileContext compiles a context statement to postfix.
	CompileContext(el string) (string, error)
}

// LocalResetter is implemented by compilers that carry per-table local
// variable state. Local slot indices are numbered per table, so a compiler
// reused across tables must be reset between them or the numbering keeps
// climbing — see compileTableEL.
type LocalResetter interface {
	ResetLocals()
}

// NewDTImporter creates a new decision table importer.
func NewDTImporter() *DTImporter {
	return &DTImporter{}
}

// SetVerbose enables verbose output during import.
func (i *DTImporter) SetVerbose(v bool) {
	i.verbose = v
}

// SetELCompiler sets the EL compiler for compiling expressions to postfix.
// When set, imported EL descriptions are compiled to postfix automatically.
func (i *DTImporter) SetELCompiler(compiler ELCompiler) {
	i.elCompiler = compiler
	i.compileEL = compiler != nil
}

// SetSymbols sets the symbol table used for EL compilation.
// The symbol table maps identifier names to their types (entity, long, double, etc.)
func (i *DTImporter) SetSymbols(symbols map[string]string) {
	i.symbols = symbols
	if i.elCompiler != nil {
		i.elCompiler.SetSymbols(symbols)
	}
}

// SetStats attaches an ImportStats collector. Pass nil to disable collection.
func (i *DTImporter) SetStats(s *ImportStats) {
	i.stats = s
}

// SourceXML records the Excel workbook and sheet from which an artifact was imported.
// It enables the exporter to route the artifact back to the exact workbook and sheet.
type SourceXML struct {
	RelativePath string `xml:"relative_path"`
	FileName     string `xml:"file_name"`
	SheetNumber  int    `xml:"sheet_number"`
}

// DecisionTablesXML represents the root XML element for decision tables.
type DecisionTablesXML struct {
	XMLName xml.Name           `xml:"decision_tables"`
	Tables  []DecisionTableXML `xml:"decision_table"`
}

// DecisionTableXML represents a single decision table in XML format.
type DecisionTableXML struct {
	Source           *SourceXML           `xml:"source,omitempty"`
	TableName        string               `xml:"table_name"`
	TableDescription string               `xml:"table_description,omitempty"`
	XLSFile          string               `xml:"xls_file"`
	AttributeFields  AttributeFieldsXML   `xml:"attribute_fields"`
	Contexts         ContextsField        `xml:"contexts"`
	InitialActions   []InitialActionXML   `xml:"initial_actions>initial_action"`
	// InitialActionsLegacy is the `<initial_action_details>` spelling, which
	// every other section of the file uses (`<condition_details>`,
	// `<action_details>`, `<context_details>`) and which SyntaxTests uses
	// throughout. Only `<initial_action>` was ever read, so those rows were
	// invisible: not loaded, not compiled, not executable, and not reachable
	// from the authoring API — 312 rows of one sample that had never run.
	// Read both, normalise to the canonical spelling on write-out.
	InitialActionsLegacy []InitialActionXML `xml:"initial_actions>initial_action_details,omitempty"`
	Conditions       []ConditionXML       `xml:"conditions>condition_details"`
	Actions          []ActionXML          `xml:"actions>action_details"`
	PolicyStatements []PolicyStatementXML `xml:"policy_statements>policy_statement"`
	ELCompiled       bool                 `xml:"el_compiled,attr"` // True if postfix was generated from EL
}

// ContextsField is the in-memory shape of a table's <contexts> element. It
// preserves every nested element on a complete round-trip:
//
//	<contexts>
//	  <context_entity>state_period</context_entity>     <!-- legacy iterator hint -->
//	  <context_details>
//	    <context_number>1</context_number>
//	    <context_comment>Iterate ...</context_comment>
//	    <context_name>...</context_name>                <!-- optional -->
//	    <context_description>...</context_description>  <!-- optional -->
//	    <context_dsl>for all accounts</context_dsl>
//	    <context_postfix>{ ... } job.accounts forall</context_postfix>
//	  </context_details>
//	</contexts>
//
// Unmarshalling also tolerates the legacy raw-text form
// (`<contexts>for all accounts</contexts>`) by promoting each non-empty line
// to a `ContextDetailXML` with a sequential number. Marshalling emits
// `<context_entity>` elements first, then a `<context_details>` block per
// entry.
type ContextsField struct {
	// Entities are the legacy `<context_entity>X</context_entity>` directives
	// that hint the table iterates over an entity collection. The loader
	// doesn't actively compile these (it expects a structured `<context_details>`
	// with `for all X` DSL), but they're preserved so existing tables don't
	// silently lose data on a round-trip.
	Entities []string
	// Details are the structured per-context entries.
	Details []ContextDetailXML
}

// ContextDetailXML represents one structured `<context_details>` block.
type ContextDetailXML struct {
	Number      int    `xml:"context_number,omitempty"`
	Comment     string `xml:"context_comment"`
	Name        string `xml:"context_name,omitempty"`
	Description string `xml:"context_description,omitempty"`
	DSL         string `xml:"context_dsl"`
	Postfix     string `xml:"context_postfix"`
}

// IsEmpty reports whether the contexts element has no entities and no details.
func (c ContextsField) IsEmpty() bool {
	return len(c.Entities) == 0 && len(c.Details) == 0
}

// DSLLines returns the DSL strings for each context detail in order. Used by
// the Excel exporter (xml_exporter.go) to lay out the contexts grid.
func (c ContextsField) DSLLines() []string {
	out := make([]string, 0, len(c.Details))
	for _, d := range c.Details {
		if dsl := strings.TrimSpace(d.DSL); dsl != "" {
			out = append(out, dsl)
		}
	}
	return out
}

// AppendDSL adds a new context detail with the given DSL string and
// sequential number. Used by Excel-import code paths that read contexts as
// raw cell values.
func (c *ContextsField) AppendDSL(dsl string) {
	c.Details = append(c.Details, ContextDetailXML{
		Number: len(c.Details) + 1,
		DSL:    dsl,
	})
}

// UnmarshalXML accepts the modern structured form, the legacy
// raw-text-inside-<contexts> form, and the loose `<context_entity>` directive
// form. All three round-trip losslessly through subsequent marshals.
func (c *ContextsField) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var tmp struct {
		Entities []string           `xml:"context_entity"`
		Details  []ContextDetailXML `xml:"context_details"`
		InnerXML string             `xml:",innerxml"`
	}
	if err := d.DecodeElement(&tmp, &start); err != nil {
		return err
	}
	c.Entities = tmp.Entities
	c.Details = tmp.Details
	// Legacy raw-text fallback: if neither entities nor details were parsed
	// but the element has content, treat each non-empty line as a single-DSL
	// context detail. Strips XML comments / surrounding whitespace.
	if len(c.Entities) == 0 && len(c.Details) == 0 {
		text := strings.TrimSpace(tmp.InnerXML)
		// If inner XML contains tags we didn't recognize, don't treat as DSL.
		if text != "" && !strings.ContainsAny(text, "<>") {
			for i, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					c.Details = append(c.Details, ContextDetailXML{
						Number: i + 1,
						DSL:    line,
					})
				}
			}
		}
	}
	// Normalize: ensure every detail has a Number assigned for write-back
	// (even if the source XML didn't include one).
	for i := range c.Details {
		if c.Details[i].Number == 0 {
			c.Details[i].Number = i + 1
		}
	}
	return nil
}

// AttributeFieldsXML holds metadata fields for the table.
//
// The table type has three spellings in real DTRules XML — <Type>, <TYPE>
// and <type> — and the runtime loader accepts all three. This model used to
// read only <Type>, so a table written in the legacy uppercase form loaded
// with an empty type and, on the next authoring write, was SAVED with an
// empty <Type>: the policy that decides whether a table is FIRST, ALL or
// BALANCED, silently erased. KidAid's tables are all <TYPE>.
//
// EffectiveType resolves the three; the marshaller writes <Type>.
type AttributeFieldsXML struct {
	Type          string `xml:"Type"`
	TypeUppercase string `xml:"TYPE,omitempty"`
	TypeLowercase string `xml:"type,omitempty"`
	Comments      string `xml:"COMMENTS"`
	TableNumber   string `xml:"TABLE_NUMBER"`
	FilePath      string `xml:"FILE_PATH,omitempty"`
}

// EffectiveType returns the table type from whichever spelling carries it.
func (a AttributeFieldsXML) EffectiveType() string {
	if a.Type != "" {
		return a.Type
	}
	if a.TypeUppercase != "" {
		return a.TypeUppercase
	}
	return a.TypeLowercase
}

// EffectiveInitialActions returns the table's initial actions from whichever
// element spelling carries them. Writers must assign to InitialActions and
// clear InitialActionsLegacy so a table cannot end up carrying two lists.
func (d DecisionTableXML) EffectiveInitialActions() []InitialActionXML {
	if len(d.InitialActions) > 0 {
		return d.InitialActions
	}
	return d.InitialActionsLegacy
}

// valueAfterColon returns everything after the first colon in a "Label: value"
// attribute cell, or "" when the cell carries only the label (e.g. an empty
// "TABLE_NUMBER:" cell). The older TrimPrefix("LABEL: ") approach assumed a
// trailing space and a value; a value-less cell slipped through and the label
// itself was stored as the value, producing invalid XML on the next load.
func valueAfterColon(cell string) string {
	if i := strings.Index(cell, ":"); i >= 0 {
		return cell[i+1:]
	}
	return cell
}

// InitialActionXML represents an initial action.
//
// Two tag conventions coexist in real DTRules XML:
//
//   - Modern (EL-aware authoring): <initial_action_dsl> / <initial_action_postfix>
//   - Legacy (Excel-import emitted): <action_dsl> / <action_postfix>
//
// On read, either form is accepted. On write, the modern form is emitted.
// The Comment field maps to <action_comment>, which both conventions share.
type InitialActionXML struct {
	Comment        string `xml:"action_comment"`
	InitialComment string `xml:"initial_action_comment,omitempty"` // legacy alternate of Comment
	DSL            string `xml:"initial_action_dsl"`
	Postfix        string `xml:"initial_action_postfix"`
	ActionDSL      string `xml:"action_dsl"`     // legacy alternate of DSL
	ActionPostfix  string `xml:"action_postfix"` // legacy alternate of Postfix
}

// EffectiveComment returns the comment from whichever spelling carries it.
func (a InitialActionXML) EffectiveComment() string {
	if a.Comment != "" {
		return a.Comment
	}
	return a.InitialComment
}

// EffectiveDSL returns the DSL with precedence: modern tag > legacy tag.
func (a InitialActionXML) EffectiveDSL() string {
	if a.DSL != "" {
		return a.DSL
	}
	return a.ActionDSL
}

// EffectivePostfix returns the postfix with precedence: modern > legacy.
func (a InitialActionXML) EffectivePostfix() string {
	if a.Postfix != "" {
		return a.Postfix
	}
	return a.ActionPostfix
}

// ConditionXML represents a condition row with its column values.
type ConditionXML struct {
	Number  string           `xml:"condition_number"`
	Comment string           `xml:"condition_comment"`
	DSL     string           `xml:"condition_dsl"`
	Postfix string           `xml:"condition_postfix"`
	Columns []ColumnValueXML `xml:"condition_column"`
}

// ActionXML represents an action row with its column values.
type ActionXML struct {
	Number  string           `xml:"action_number"`
	Comment string           `xml:"action_comment"`
	DSL     string           `xml:"action_dsl"`
	Postfix string           `xml:"action_postfix"`
	Columns []ColumnValueXML `xml:"action_column"`
}

// ColumnValueXML represents a column value in conditions or actions.
type ColumnValueXML struct {
	Number int    `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

// PolicyStatementXML represents a policy statement for a column.
type PolicyStatementXML struct {
	Column      string `xml:"column,attr"`
	Description string `xml:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix"`
}

// ImportDecisionTables reads an Excel file and returns decision table XML.
func (i *DTImporter) ImportDecisionTables(filename string) (*DecisionTablesXML, error) {
	return i.importDecisionTablesWithSource(filename, filepath.Base(filename), filepath.Base(filename))
}

// importDecisionTablesWithSource reads an Excel file and returns decision table XML,
// setting xlsFile in XLSFile and relPath in the <source> element.
func (i *DTImporter) importDecisionTablesWithSource(filename, xlsFile, relPath string) (*DecisionTablesXML, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	tables := &DecisionTablesXML{}

	// Process each sheet as a decision table, tracking 1-based sheet index
	sheets := f.GetSheetList()
	for sheetIdx, sheetName := range sheets {
		if i.verbose {
			fmt.Printf("Processing sheet: %s\n", sheetName)
		}

		table, err := i.parseSheet(f, sheetName, xlsFile)
		if err != nil {
			if i.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
		}
		if table != nil {
			// Compile EL expressions if compiler is set
			if err := i.compileTableEL(table); err != nil {
				if i.verbose {
					fmt.Printf("  EL compilation warning: %v\n", err)
				}
			}
			table.Source = &SourceXML{
				RelativePath: relPath,
				FileName:     filepath.Base(filename),
				SheetNumber:  sheetIdx + 1, // 1-based
			}
			tables.Tables = append(tables.Tables, *table)
			if i.stats != nil {
				i.stats.Tables++
				i.stats.Actions += len(table.Actions)
				i.stats.Conditions += len(table.Conditions)
			}
		}
	}

	return tables, nil
}

// ImportDecisionTablesFromDir reads all Excel files in a directory recursively.
// Files are processed in sorted order within each directory level.
// The xls_file metadata is set to the relative path from the base directory.
func (i *DTImporter) ImportDecisionTablesFromDir(dir string) (*DecisionTablesXML, error) {
	tables := &DecisionTablesXML{}

	// Collect all xlsx files with their paths
	type fileEntry struct {
		path    string
		relPath string
	}
	var xlsxFiles []fileEntry

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
			return nil
		}
		// Skip temp files (Excel creates these while files are open)
		if strings.HasPrefix(name, "~$") {
			return nil
		}

		// Calculate relative path for xls_file metadata
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = name
		}

		xlsxFiles = append(xlsxFiles, fileEntry{path: path, relPath: relPath})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort by relative path for consistent ordering
	sort.Slice(xlsxFiles, func(i, j int) bool {
		return xlsxFiles[i].relPath < xlsxFiles[j].relPath
	})

	// Process each file
	for _, entry := range xlsxFiles {
		if i.verbose {
			fmt.Printf("Processing file: %s\n", entry.relPath)
		}

		fileTables, err := i.importDecisionTablesWithSource(entry.path, entry.relPath, entry.relPath)
		if err != nil {
			if i.verbose {
				fmt.Printf("  Warning: %v\n", err)
			}
			continue
		}

		tables.Tables = append(tables.Tables, fileTables.Tables...)
	}

	return tables, nil
}

// WriteXML writes the decision tables to an XML file.
// normalizeProvenance backfills workbook provenance — XLSFile and the
// <source> element — for tables that have none, before the XML is written.
// Emitted XML is then a complete record an editor or sync tool can use to
// locate every table's workbook, every compile.
//
// Tables created through the authoring SDK (Project.AddTable) historically
// carried an empty <xls_file> and no <source> at all, while Excel-imported
// tables carried both. Tooling that locates a table's workbook/sheet through
// this metadata then sees two classes of table — and the SDK-authored ones
// look like they don't exist even though the engine resolves and runs them
// (observed in the staking rules: 19 of 34 tables were provenance-less).
// The emitter owns this metadata: DTRules produces the XML on every compile,
// so every compile must emit it for every table.
//
// Existing provenance is PRESERVED, never rewritten — <source> records where
// a table was imported from, and the round-trip contract keeps original
// sheet positions intact (see TestRoundTripSourceMetadata). Backfill rules
// for tables without provenance:
//   - Workbook: inherited from the first table in the file that has one;
//     failing that, derived from the XML filename ("staking_dt.xml" ->
//     "staking.xlsx"). RelativePath inherits the same way.
//   - Sheet numbers: appended after the file's highest existing sheet
//     number, in ascending numeric TABLE_NUMBER order (non-numeric numbers
//     after numeric, ties by name — the exporter's ordering), so assignment
//     is deterministic across compiles and reads as "these tables land as
//     appended sheets".
func normalizeProvenance(tables *DecisionTablesXML, xmlPath string) {
	if tables == nil || len(tables.Tables) == 0 {
		return
	}

	workbook, rel, maxSheet := "", "", 0
	var missing []int
	for i := range tables.Tables {
		t := &tables.Tables[i]
		if workbook == "" && strings.TrimSpace(t.XLSFile) != "" {
			workbook = strings.TrimSpace(t.XLSFile)
		}
		if t.Source != nil {
			if rel == "" && strings.TrimSpace(t.Source.RelativePath) != "" {
				rel = strings.TrimSpace(t.Source.RelativePath)
			}
			if t.Source.SheetNumber > maxSheet {
				maxSheet = t.Source.SheetNumber
			}
		} else {
			missing = append(missing, i)
		}
	}
	if workbook == "" {
		base := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
		base = strings.TrimSuffix(base, "_dt")
		workbook = base + ".xlsx"
	}
	if rel == "" {
		rel = workbook
	}
	fileName := filepath.Base(workbook)

	// Deterministic append order for the provenance-less tables.
	sort.SliceStable(missing, func(a, b int) bool {
		ta, tb := &tables.Tables[missing[a]], &tables.Tables[missing[b]]
		na, errA := strconv.Atoi(strings.TrimSpace(ta.AttributeFields.TableNumber))
		nb, errB := strconv.Atoi(strings.TrimSpace(tb.AttributeFields.TableNumber))
		if errA == nil && errB == nil {
			return na < nb
		}
		if errA == nil {
			return true
		}
		if errB == nil {
			return false
		}
		return ta.TableName < tb.TableName
	})
	for _, i := range missing {
		t := &tables.Tables[i]
		maxSheet++
		t.Source = &SourceXML{RelativePath: rel, FileName: fileName, SheetNumber: maxSheet}
	}
	for i := range tables.Tables {
		if strings.TrimSpace(tables.Tables[i].XLSFile) == "" {
			tables.Tables[i].XLSFile = workbook
		}
	}
}

// normalizeTableNumbers backfills missing TABLE_NUMBERs. Tables that already
// carry a numeric number keep it; the rest are assigned numbers in document
// order, in increments of 100, continuing above the highest existing number
// (starting at 100 when none are numbered).
func normalizeTableNumbers(tables *DecisionTablesXML) {
	max := 0
	for i := range tables.Tables {
		if n, err := strconv.Atoi(strings.TrimSpace(tables.Tables[i].AttributeFields.TableNumber)); err == nil && n > max {
			max = n
		}
	}
	next := (max/100)*100 + 100
	for i := range tables.Tables {
		t := &tables.Tables[i]
		if _, err := strconv.Atoi(strings.TrimSpace(t.AttributeFields.TableNumber)); err != nil {
			t.AttributeFields.TableNumber = strconv.Itoa(next)
			next += 100
		}
	}
}

// normalizeSectionNumbers renumbers every per-table section sequentially
// (1..N in document order): contexts, conditions, actions. Section numbers
// are display labels with no semantic value — the engine executes by
// position — so a user-specified sequence buys nothing, and drifted
// numbering (1,2,4,7,...) made the editor, the debugger, and engine error
// messages disagree about which action was which.
func normalizeSectionNumbers(tables *DecisionTablesXML) {
	for i := range tables.Tables {
		t := &tables.Tables[i]
		for j := range t.Contexts.Details {
			t.Contexts.Details[j].Number = j + 1
		}
		for j := range t.Conditions {
			t.Conditions[j].Number = strconv.Itoa(j + 1)
		}
		for j := range t.Actions {
			t.Actions[j].Number = strconv.Itoa(j + 1)
		}
	}
}

func (i *DTImporter) WriteXML(tables *DecisionTablesXML, filename string) error {
	normalizeTableNumbers(tables)
	normalizeSectionNumbers(tables)
	normalizeProvenance(tables, filename)

	// Open file for writing
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create XML file: %w", err)
	}
	defer f.Close()

	// Write XML header
	f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	// Write opening tag
	f.WriteString("<decision_tables>\n")

	// Write each table
	for _, table := range tables.Tables {
		if err := i.writeTable(f, &table); err != nil {
			return err
		}
	}

	// Write closing tag
	f.WriteString("</decision_tables>\n")

	return nil
}

// writeTable writes a single decision table to the file with proper formatting.
//
// The emitted XML preserves every field that the in-memory model represents.
// Empty optional elements (DSL, postfix, comment) are emitted as
// self-closing tags when empty, matching what the loader expects and what
// hand-authored XML in the project looks like.
func (i *DTImporter) writeTable(f *os.File, table *DecisionTableXML) error {
	tableNum := table.AttributeFields.TableNumber
	f.WriteString(fmt.Sprintf("\n<!-- TABLE %s: %s -->\n", tableNum, table.TableName))

	if table.ELCompiled {
		f.WriteString("<decision_table el_compiled=\"true\">\n")
	} else {
		f.WriteString("<decision_table>\n")
	}

	if table.Source != nil {
		f.WriteString("<source>\n")
		f.WriteString(fmt.Sprintf("<relative_path>%s</relative_path>\n", xmlEscapeText(table.Source.RelativePath)))
		f.WriteString(fmt.Sprintf("<file_name>%s</file_name>\n", xmlEscapeText(table.Source.FileName)))
		f.WriteString(fmt.Sprintf("<sheet_number>%d</sheet_number>\n", table.Source.SheetNumber))
		f.WriteString("</source>\n")
	}

	f.WriteString(fmt.Sprintf("<table_name>%s</table_name>\n", xmlEscapeText(table.TableName)))
	if table.TableDescription != "" {
		f.WriteString(fmt.Sprintf("<table_description>%s</table_description>\n", xmlEscapeText(table.TableDescription)))
	}
	f.WriteString(fmt.Sprintf("<xls_file>%s</xls_file>\n", xmlEscapeText(table.XLSFile)))

	// Attribute fields
	f.WriteString("<attribute_fields>\n")
	f.WriteString(fmt.Sprintf("<Type>%s</Type>\n", xmlEscapeText(table.AttributeFields.EffectiveType())))
	f.WriteString(fmt.Sprintf("<COMMENTS>%s</COMMENTS>\n", xmlEscapeText(table.AttributeFields.Comments)))
	f.WriteString(fmt.Sprintf("<TABLE_NUMBER>%s</TABLE_NUMBER>\n", xmlEscapeText(table.AttributeFields.TableNumber)))
	if table.AttributeFields.FilePath != "" {
		f.WriteString(fmt.Sprintf("<FILE_PATH>%s</FILE_PATH>\n", xmlEscapeText(table.AttributeFields.FilePath)))
	}
	f.WriteString("</attribute_fields>\n")

	writeContextsXML(f, table.Contexts)

	// Initial actions. Both element spellings and both DSL/postfix tag
	// conventions are accepted on read; on write we emit the canonical
	// <initial_action> element and the modern <initial_action_dsl> /
	// <initial_action_postfix> tags for new content, preserving whichever
	// form was present for entries originally read from legacy XML.
	//
	// Read through EffectiveInitialActions, not the canonical field alone: a
	// table that arrived spelled <initial_action_details> has an empty
	// canonical list, and writing that emitted <initial_actions></initial_actions>
	// — silently deleting every initial action in the table on the first save.
	initialActions := table.EffectiveInitialActions()
	if len(initialActions) == 0 {
		f.WriteString("<initial_actions></initial_actions>\n")
	} else {
		f.WriteString("<initial_actions>\n")
		for _, action := range initialActions {
			f.WriteString("<initial_action>\n")
			if c := action.EffectiveComment(); c != "" {
				f.WriteString(fmt.Sprintf("<action_comment>%s</action_comment>\n", xmlEscapeText(c)))
			}
			writeDSLOrPostfix(f, "initial_action_dsl", action.DSL)
			writeBlockPostfix(f, "initial_action_postfix", action.Postfix)
			// Preserve legacy alternate tags if they were the originals
			// (i.e., the modern fields are empty but the legacy fields have
			// content).
			if action.DSL == "" && action.ActionDSL != "" {
				writeDSLOrPostfix(f, "action_dsl", action.ActionDSL)
			}
			if action.Postfix == "" && action.ActionPostfix != "" {
				writeBlockPostfix(f, "action_postfix", action.ActionPostfix)
			}
			f.WriteString("</initial_action>\n")
		}
		f.WriteString("</initial_actions>\n")
	}

	// Conditions
	if len(table.Conditions) == 0 {
		f.WriteString("<conditions></conditions>\n")
	} else {
		f.WriteString("<conditions>\n")
		for _, cond := range table.Conditions {
			f.WriteString("<condition_details>\n")
			f.WriteString(fmt.Sprintf("<condition_number>%s</condition_number>\n", xmlEscapeText(cond.Number)))
			f.WriteString(fmt.Sprintf("<condition_comment>%s</condition_comment>\n", xmlEscapeText(cond.Comment)))
			writeDSLOrPostfix(f, "condition_dsl", cond.DSL)
			writeBlockPostfix(f, "condition_postfix", cond.Postfix)
			for _, col := range sortedColumns(cond.Columns) {
				f.WriteString(fmt.Sprintf("<condition_column column_number=\"%d\" column_value=\"%s\" />\n",
					col.Number, xmlEscapeAttr(col.Value)))
			}
			f.WriteString("</condition_details>\n")
		}
		f.WriteString("</conditions>\n")
	}

	// Actions
	if len(table.Actions) == 0 {
		f.WriteString("<actions></actions>\n")
	} else {
		f.WriteString("<actions>\n")
		for _, action := range table.Actions {
			f.WriteString("<action_details>\n")
			f.WriteString(fmt.Sprintf("<action_number>%s</action_number>\n", xmlEscapeText(action.Number)))
			f.WriteString(fmt.Sprintf("<action_comment>%s</action_comment>\n", xmlEscapeText(action.Comment)))
			writeDSLOrPostfix(f, "action_dsl", action.DSL)
			writeBlockPostfix(f, "action_postfix", action.Postfix)
			for _, col := range sortedColumns(action.Columns) {
				f.WriteString(fmt.Sprintf("<action_column column_number=\"%d\" column_value=\"%s\" />\n",
					col.Number, xmlEscapeAttr(col.Value)))
			}
			f.WriteString("</action_details>\n")
		}
		f.WriteString("</actions>\n")
	}

	// Policy statements
	if len(table.PolicyStatements) == 0 {
		f.WriteString("<policy_statements></policy_statements>\n")
	} else {
		f.WriteString("<policy_statements>\n")
		for _, policy := range table.PolicyStatements {
			f.WriteString(fmt.Sprintf("<policy_statement column=\"%s\">\n", xmlEscapeAttr(policy.Column)))
			f.WriteString(fmt.Sprintf("<policy_description>%s</policy_description>\n", xmlEscapeText(policy.Description)))
			f.WriteString(fmt.Sprintf("<policy_statement_postfix>%s</policy_statement_postfix>\n", xmlEscapeText(policy.Postfix)))
			f.WriteString("</policy_statement>\n")
		}
		f.WriteString("</policy_statements>\n")
	}

	f.WriteString("</decision_table>\n")
	return nil
}

// writeDSLOrPostfix emits a one-line DSL element. Empty content produces a
// self-closing tag (e.g. `<condition_dsl />`) which is what hand-authored
// XML in the project looks like and what the loader treats as "no DSL".
func writeDSLOrPostfix(f *os.File, tag, content string) {
	if strings.TrimSpace(content) == "" {
		f.WriteString(fmt.Sprintf("<%s />\n", tag))
		return
	}
	f.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, xmlEscapeText(content), tag))
}

// writeBlockPostfix emits a postfix-style block. Empty content produces an
// empty open/close pair, matching the loader-tolerated form. Non-empty
// content is wrapped with newlines for readability.
func writeBlockPostfix(f *os.File, tag, content string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		f.WriteString(fmt.Sprintf("<%s></%s>\n", tag, tag))
		return
	}
	f.WriteString(fmt.Sprintf("<%s>\n%s\n</%s>\n", tag, xmlEscapeText(trimmed), tag))
}

// writeContextsXML emits the <contexts>…</contexts> block, preserving the
// full structure: legacy `<context_entity>` directives first, then one
// `<context_details>` block per detail with all of its sub-elements.
func writeContextsXML(f *os.File, contexts ContextsField) {
	if contexts.IsEmpty() {
		f.WriteString("<contexts></contexts>\n")
		return
	}
	f.WriteString("<contexts>\n")
	for _, ent := range contexts.Entities {
		f.WriteString(fmt.Sprintf("<context_entity>%s</context_entity>\n", xmlEscapeText(ent)))
	}
	for _, d := range contexts.Details {
		f.WriteString("<context_details>\n")
		num := d.Number
		if num == 0 {
			num = 1
		}
		f.WriteString(fmt.Sprintf("<context_number>%d</context_number>\n", num))
		f.WriteString(fmt.Sprintf("<context_comment>%s</context_comment>\n", xmlEscapeText(d.Comment)))
		if d.Name != "" {
			f.WriteString(fmt.Sprintf("<context_name>%s</context_name>\n", xmlEscapeText(d.Name)))
		}
		if d.Description != "" {
			f.WriteString(fmt.Sprintf("<context_description>%s</context_description>\n", xmlEscapeText(d.Description)))
		}
		if d.DSL == "" {
			f.WriteString("<context_dsl />\n")
		} else {
			f.WriteString(fmt.Sprintf("<context_dsl>%s</context_dsl>\n", xmlEscapeText(d.DSL)))
		}
		if d.Postfix == "" {
			f.WriteString("<context_postfix></context_postfix>\n")
		} else {
			f.WriteString(fmt.Sprintf("<context_postfix>\n%s\n</context_postfix>\n", xmlEscapeText(strings.TrimSpace(d.Postfix))))
		}
		f.WriteString("</context_details>\n")
	}
	f.WriteString("</contexts>\n")
}

// xmlEscapeText escapes special XML characters for use in element text
// content. Double-quote characters are left as-is since XML only requires
// them escaped inside attribute values.
func xmlEscapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// xmlEscapeAttr escapes special XML characters for use in attribute values.
// Includes double-quote escaping since attributes are delimited by ".
func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// xmlEscape is retained for backward compatibility with callers that emit
// quote-delimited attribute values via fmt.Sprintf into the same template.
// Prefer xmlEscapeText for element text and xmlEscapeAttr for attributes.
func xmlEscape(s string) string { return xmlEscapeAttr(s) }

// parseSheet parses a single Excel sheet into a decision table.
func (i *DTImporter) parseSheet(f *excelize.File, sheetName, xlsFile string) (*DecisionTableXML, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty sheet")
	}

	table := &DecisionTableXML{
		XLSFile: xlsFile,
	}

	// Try to detect format (dt2excel format vs exporter.go format)
	format := i.detectFormat(rows)

	switch format {
	case "dt2excel":
		return i.parseDT2ExcelFormat(rows, table)
	case "exporter":
		return i.parseExporterFormat(rows, sheetName, table)
	default:
		return nil, fmt.Errorf("unrecognized Excel format")
	}
}

// detectFormat detects which Excel format the sheet uses.
func (i *DTImporter) detectFormat(rows [][]string) string {
	if len(rows) == 0 {
		return "unknown"
	}

	// Check first row for format indicators
	if len(rows[0]) > 0 {
		firstCell := strings.ToLower(strings.TrimSpace(rows[0][0]))

		// dt2excel format starts with "Decision Table"
		if firstCell == "decision table" {
			return "dt2excel"
		}

		// exporter.go format starts with "Name: <table_name>"
		if strings.HasPrefix(firstCell, "name:") {
			return "exporter"
		}
	}

	// Try to detect by looking for other markers
	for _, row := range rows {
		if len(row) > 0 {
			cell := strings.ToLower(strings.TrimSpace(row[0]))
			if strings.HasPrefix(cell, "conditions:") {
				return "exporter"
			}
			if cell == "conditions" && len(row) > 1 {
				return "dt2excel"
			}
		}
	}

	return "unknown"
}

// parseDT2ExcelFormat parses the format generated by cmd/dt2excel.
func (i *DTImporter) parseDT2ExcelFormat(rows [][]string, table *DecisionTableXML) (*DecisionTableXML, error) {
	var currentSection string
	_ = 0 // maxCol not needed since we process all non-empty cells

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		firstCell := strings.TrimSpace(row[0])
		firstCellLower := strings.ToLower(firstCell)

		switch {
		case firstCellLower == "decision table":
			if len(row) > 1 {
				table.TableName = strings.TrimSpace(row[1])
			}

		case firstCellLower == "table number":
			if len(row) > 1 {
				table.AttributeFields.TableNumber = strings.TrimSpace(row[1])
			}

		case firstCellLower == "type":
			if len(row) > 1 {
				table.AttributeFields.Type = strings.TrimSpace(row[1])
			}

		case firstCellLower == "comments":
			if len(row) > 1 {
				table.AttributeFields.Comments = strings.TrimSpace(row[1])
			}

		case firstCellLower == "file_path":
			if len(row) > 1 {
				table.AttributeFields.FilePath = strings.TrimSpace(row[1])
			}

		case firstCellLower == "initial actions":
			currentSection = "initial_actions"

		case firstCellLower == "conditions":
			currentSection = "conditions"

		case firstCellLower == "actions":
			currentSection = "actions"
			// Keep maxCol from conditions section

		case firstCellLower == "policy statements":
			currentSection = "policy"

		case firstCellLower == "#":
			// Header row, skip
			continue

		default:
			switch currentSection {
			case "initial_actions":
				if len(row) > 1 && firstCell != "" {
					action := InitialActionXML{
						DSL:     firstCell,
						Postfix: strings.TrimSpace(row[1]),
					}
					table.InitialActions = append(table.InitialActions, action)
				}

			case "conditions":
				// Parse condition row: #, DSL, Col1, Col2, ...
				// Only process rows that start with a number
				if len(row) > 1 && firstCell != "" {
					if _, err := strconv.Atoi(firstCell); err == nil {
						cond := ConditionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)), // Use comment as DSL
						}
						// Parse column values - columns start at index 2
						for col := 2; col < len(row); col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								cond.Columns = append(cond.Columns, ColumnValueXML{
									Number: col - 1, // column 1 is at index 2, so col-1
									Value:  val,
								})
							}
						}
						table.Conditions = append(table.Conditions, cond)
					}
				}

			case "actions":
				// Parse action row: #, DSL, Col1, Col2, ...
				// Only process rows that start with a number
				if len(row) > 1 && firstCell != "" {
					if _, err := strconv.Atoi(firstCell); err == nil {
						action := ActionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 1)), // Use comment as DSL
						}
						// Parse column values - columns start at index 2
						for col := 2; col < len(row); col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								action.Columns = append(action.Columns, ColumnValueXML{
									Number: col - 1, // column 1 is at index 2, so col-1
									Value:  val,
								})
							}
						}
						table.Actions = append(table.Actions, action)
					}
				}

			case "policy":
				// Parse policy row: "Column N", Description
				if strings.HasPrefix(firstCellLower, "column ") {
					colStr := strings.TrimPrefix(firstCellLower, "column ")
					policy := PolicyStatementXML{
						Column:      colStr,
						Description: strings.TrimSpace(safeGet(row, 1)),
					}
					table.PolicyStatements = append(table.PolicyStatements, policy)
				}
			}
		}

		_ = rowIdx // Used in conditions section
	}

	if table.TableName == "" {
		return nil, fmt.Errorf("no table name found")
	}

	return table, nil
}

// parseExporterFormat parses the format generated by exporter.go.
func (i *DTImporter) parseExporterFormat(rows [][]string, sheetName string, table *DecisionTableXML) (*DecisionTableXML, error) {
	var currentSection string
	var numCols int

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		firstCell := strings.TrimSpace(row[0])
		firstCellLower := strings.ToLower(firstCell)

		switch {
		case strings.HasPrefix(firstCellLower, "dt:"):
			// Type-marked name row: "DT: Table_Name" (new combined-workbook format)
			name := strings.TrimPrefix(firstCell, "DT: ")
			name = strings.TrimPrefix(name, "dt: ")
			name = strings.TrimPrefix(name, "DT:")
			name = strings.TrimPrefix(name, "dt:")
			table.TableName = strings.ReplaceAll(strings.TrimSpace(name), " ", "_")

		case strings.HasPrefix(firstCellLower, "name:"):
			// Legacy name row: "Name: Table_Name"
			name := strings.TrimPrefix(firstCell, "Name: ")
			name = strings.TrimPrefix(name, "name: ")
			// Convert display name back to underscore format
			table.TableName = strings.ReplaceAll(strings.TrimSpace(name), " ", "_")

		case strings.HasPrefix(firstCellLower, "type:"):
			// Type row: "Type: FIRST"
			table.AttributeFields.Type = strings.TrimSpace(valueAfterColon(firstCell))

		case strings.HasPrefix(firstCellLower, "comments:"):
			// Comments field
			table.AttributeFields.Comments = strings.TrimSpace(valueAfterColon(firstCell))

		case strings.HasPrefix(firstCellLower, "table_number:"):
			// Table number field
			table.AttributeFields.TableNumber = strings.TrimSpace(valueAfterColon(firstCell))

		case strings.HasPrefix(firstCellLower, "file_path:"):
			// File path field
			table.AttributeFields.FilePath = strings.TrimSpace(valueAfterColon(firstCell))

		case strings.HasPrefix(firstCellLower, "contexts"):
			// Exporter writes the title as "CONTEXTS: COMMENTS", so an exact
			// "contexts:" match silently dropped every table's context block
			// (e.g. a "for all ..." iterator) on round-trip.
			currentSection = "contexts"

		// The exporter writes each section's title, DSL label, and rule-column
		// numbers on ONE row (see exporter.writeConditions), then data rows
		// follow. So parse numCols from THIS title row and go straight to the
		// data section — exactly as the "contexts" case above already does.
		// (The old *_header states consumed the *next* row as a separate
		// header, which silently ate the first condition/action of every
		// table on round-trip.) Robust to a legacy two-row layout too: a
		// stray column-number row has no positive integer in column A, so the
		// data parser skips it.
		case strings.HasPrefix(firstCellLower, "initial actions"):
			currentSection = "initial_actions"

		case strings.HasPrefix(firstCellLower, "conditions"):
			numCols = i.countHeaderColumns(row)
			currentSection = "conditions"

		case strings.HasPrefix(firstCellLower, "actions"):
			numCols = i.countHeaderColumns(row)
			currentSection = "actions"

		// Same one-row layout as conditions/actions above: the exporter writes
		// "Policy:", the "Policy Statements" label, and the rule-column numbers
		// on the title row, then one data row per statement. Routing through a
		// separate *_header state ate the first statement of every table on
		// round-trip (column 1's policy vanished).
		case strings.HasPrefix(firstCellLower, "policy"):
			currentSection = "policy"

		default:
			switch currentSection {
			case "contexts":
				// Context row: number, comment, expression (merged at col C+).
				// Each row becomes a structured ContextDetailXML so non-DSL
				// fields (comment, postfix) round-trip cleanly.
				if len(row) > 2 && firstCell != "" {
					table.Contexts.Details = append(table.Contexts.Details, ContextDetailXML{
						Number:  len(table.Contexts.Details) + 1,
						Comment: strings.TrimSpace(safeGet(row, 1)),
						DSL:     strings.TrimSpace(row[2]),
					})
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "initial_actions_header":
				// Skip the "Initial Actions" label row, next row is header with column numbers
				currentSection = "initial_actions"

			case "initial_actions":
				// Initial action row: number, comment, DSL
				// Col A=number, col B=comment, col C=DSL
				if len(row) > 2 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						action := InitialActionXML{
							DSL: strings.TrimSpace(row[2]),
						}
						table.InitialActions = append(table.InitialActions, action)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "conditions_header":
				// Header row with column numbers
				numCols = i.countHeaderColumns(row)
				currentSection = "conditions"

			case "conditions":
				// Condition row: col A=number, col B=comment, col C=DSL, col D+=decision columns
				// Postfix is not stored in Excel (stripped on export); EL compiler regenerates it.
				if len(row) > 2 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						cond := ConditionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 2)),
						}
						// Parse column values (columns start at index 3).
						//
						// An explicit "-" is kept. It means the same thing to
						// the runtime as an absent entry (Stepper.processColumn:
						// `if !hasVal || colVal == "-" { continue }`), but
						// discarding it made the round trip lossy — the XML's
						// "-" came back as nothing, so an Excel-authored rebuild
						// always differed from the committed XML and the #1010
						// gate reported a difference no author could act on
						// (#1017). An empty cell still yields no entry, so a
						// project that never used "-" is unaffected.
						for col := 3; col < len(row) && col < numCols+3; col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								cond.Columns = append(cond.Columns, ColumnValueXML{
									Number: col - 2, // 1-indexed
									Value:  val,
								})
							}
						}
						table.Conditions = append(table.Conditions, cond)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "actions_header":
				// Header row with column numbers
				numCols = i.countHeaderColumns(row)
				currentSection = "actions"

			case "actions":
				// Action row: col A=number, col B=comment, col C=DSL, col D+=decision columns
				// Postfix is not stored in Excel (stripped on export); EL compiler regenerates it.
				if len(row) > 2 && firstCell != "" {
					num, err := strconv.Atoi(firstCell)
					if err == nil && num > 0 {
						action := ActionXML{
							Number:  firstCell,
							Comment: strings.TrimSpace(safeGet(row, 1)),
							DSL:     strings.TrimSpace(safeGet(row, 2)),
						}
						// "-" kept, as for conditions above (#1017).
						for col := 3; col < len(row) && col < numCols+3; col++ {
							val := strings.TrimSpace(row[col])
							if val != "" {
								action.Columns = append(action.Columns, ColumnValueXML{
									Number: col - 2, // 1-indexed
									Value:  val,
								})
							}
						}
						table.Actions = append(table.Actions, action)
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}

			case "policy":
				// New format: column number in A, policy text in B (one row per policy)
				if firstCell != "" {
					if _, err := strconv.Atoi(firstCell); err == nil && len(row) > 1 {
						desc := strings.TrimSpace(safeGet(row, 1))
						if desc != "" {
							policy := PolicyStatementXML{
								Column:      firstCell,
								Description: desc,
								Postfix:     CompilePolicyStatement(desc),
							}
							table.PolicyStatements = append(table.PolicyStatements, policy)
						}
						continue
					}
				}
				// A legacy two-row layout puts the rule-column numbers on their
				// own row below the title. Skip it: those numbers are headers,
				// not statements.
				if isColumnNumberRow(row) {
					continue
				}
				// Old format: empty A, "Column Policy" in B, statements in D, E, F...
				if len(row) > 3 {
					for col := 3; col < len(row); col++ {
						val := strings.TrimSpace(row[col])
						if val != "" {
							policy := PolicyStatementXML{
								Column:      strconv.Itoa(col - 2), // 1-indexed
								Description: val,
								Postfix:     CompilePolicyStatement(val),
							}
							table.PolicyStatements = append(table.PolicyStatements, policy)
						}
					}
				}
				if len(row) == 0 || firstCell == "" {
					currentSection = ""
				}
			}
		}

		_ = rowIdx // Suppress unused warning
	}

	// Fallback: use sheet name if no table name found
	if table.TableName == "" {
		table.TableName = strings.ReplaceAll(sheetName, " ", "_")
	}

	if table.TableName == "" {
		return nil, fmt.Errorf("no table name found")
	}

	return table, nil
}

// countColumns counts the number of decision columns from a header row.
func (i *DTImporter) countColumns(row []string) int {
	count := 0
	colPattern := regexp.MustCompile(`(?i)^col\s*\d+$`)
	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if colPattern.MatchString(cell) {
			count++
		}
	}
	return count
}

// countHeaderColumns counts columns from the exporter format header (1, 2, 3, ...).
func (i *DTImporter) countHeaderColumns(row []string) int {
	count := 0
	for j := 3; j < len(row); j++ {
		cell := strings.TrimSpace(row[j])
		if _, err := strconv.Atoi(cell); err == nil {
			count++
		}
	}
	if count == 0 {
		// Fallback: count non-empty cells after column 3
		for j := 3; j < len(row); j++ {
			cell := strings.TrimSpace(row[j])
			if cell != "" {
				count++
			}
		}
	}
	return count
}

// sortedColumns returns the column entries in column order without disturbing
// the caller's slice. Written XML is a file under version control, so its
// element order has to be a function of the content — not of whatever order a
// Go map happened to yield upstream.
func sortedColumns(cols []ColumnValueXML) []ColumnValueXML {
	out := make([]ColumnValueXML, len(cols))
	copy(out, cols)
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// isColumnNumberRow reports whether a row is a bare rule-column header —
// columns A and B empty and every remaining populated cell an integer, e.g.
// the "1 2 3 4" row a legacy two-row section layout puts under its title.
// Such a row carries no authored content and must not be parsed as data.
func isColumnNumberRow(row []string) bool {
	if strings.TrimSpace(safeGet(row, 0)) != "" || strings.TrimSpace(safeGet(row, 1)) != "" {
		return false
	}
	numbers := 0
	for j := 2; j < len(row); j++ {
		cell := strings.TrimSpace(row[j])
		if cell == "" {
			continue
		}
		if _, err := strconv.Atoi(cell); err != nil {
			return false
		}
		numbers++
	}
	return numbers > 0
}

// safeGet safely gets a string from a slice, returning empty string if out of bounds.
func safeGet(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// compileTableEL compiles EL expressions in a table to postfix.
// This is called after parsing to generate postfix from EL descriptions.
// If no EL compiler is set, this method does nothing.
func (i *DTImporter) compileTableEL(table *DecisionTableXML) error {
	if !i.compileEL || i.elCompiler == nil {
		// Skipping compilation is only harmless when there is no DSL to
		// compile. With DSL present it means every postfix element in this
		// table is about to be written empty, producing a rule set that loads
		// and decides nothing.
		//
		// This was silent, and it is exactly how `dtrules sync import` shipped
		// a 490-line degradation of KidAid while reporting success — two
		// import pipelines, only one of which wired a compiler (#929). A
		// caller that legitimately wants no compilation passes no DSL or
		// collects no stats, so this costs them nothing.
		if n := countDSLRows(table); n > 0 && i.stats != nil {
			i.stats.AddDrop(table.TableName, 0, "postfix",
				fmt.Sprintf("no EL compiler wired: %d DSL row(s) would be written with empty postfix — "+
					"construct the importer with SetELCompiler (see newWorkbookImporter in cmd/dtrules)", n))
		}
		return nil
	}

	// Compiling with no symbol table is not a degraded mode, it is a wrong
	// one: with no types, every field reference compiles as an integer. A
	// `fixed` subtraction becomes `-` instead of `fp-` and the assignment
	// stores `cvi` instead of `cvfp`, so a money calculation silently changes
	// its arithmetic — and the build reports "no drops" and exits 0.
	//
	// This is reachable whenever the workbook set carries no EDD sheet and the
	// output directory has no EDD either, which is exactly what
	// `build --from-excel` into a fresh directory does: symbols come from the
	// *output* dir (newWorkbookImporter → LoadEDDSymbols(xmlDir)). The
	// Accumulate staking rules hit it — 35 tables, entities=0, every
	// fixed-point operator downgraded (#1029).
	//
	// A table with no DSL has nothing to type, so it is unaffected.
	if n := countDSLRows(table); len(i.symbols) == 0 && n > 0 {
		if i.stats != nil {
			i.stats.AddDrop(table.TableName, 0, "postfix",
				fmt.Sprintf("no EDD in the Excel source: %d DSL row(s) would compile untyped, "+
					"turning fp- into - and cvfp into cvi. Export your EDD into the workbook "+
					"(dtrules sync export --force), then rebuild", n))
		}
		return fmt.Errorf("%s: refusing to compile with no EDD symbols — "+
			"every field would be typed as integer", table.TableName)
	}

	// Local slot indices are numbered per table, and this importer reuses one
	// compiler for every table in a workbook. Without a reset the counter
	// keeps climbing, so the second table's `local@` indices come out one
	// higher than the frame the runtime allocates, the third two higher, and
	// so on. The rules still build — the postfix is well formed — and fail at
	// execution with "[OutOfBounds] GetFrameValue" (#1047).
	//
	// Compiler.ResetLocals documents the rule and authoring already obeys it;
	// this path never did, which is why it only showed up when a project was
	// rebuilt from Excel rather than authored through the API.
	if r, ok := i.elCompiler.(LocalResetter); ok {
		r.ResetLocals()
	}

	// Set symbols if available
	if i.symbols != nil {
		i.elCompiler.SetSymbols(i.symbols)
	}

	var errors []string

	// recordCompileFailure decides whether a compile error is a fatal drop
	// (real broken EL) or a non-fatal warning (legacy prose-DSL where the DSL
	// field equals the comment text, pre-#504). Prose flows through the
	// round-trip unchanged; the build stays green but the consumer sees the
	// warning and can migrate at their pace. Returns true if the failure
	// was a real drop (fatal), false if classified as a warning.
	hadDrop := false
	recordCompileFailure := func(tableName, item, dsl, comment string, err error) {
		isWarning := strings.TrimSpace(dsl) == strings.TrimSpace(comment) && comment != ""
		if i.stats != nil {
			if isWarning {
				i.stats.AddWarning(tableName, 0, item,
					"legacy prose-DSL (DSL field equals comment text); not valid EL, preserved verbatim — migrate per #504: "+err.Error())
			} else {
				i.stats.AddDrop(tableName, 0, item, err.Error())
			}
		}
		if !isWarning {
			hadDrop = true
		}
	}

	// Compile initial actions. Initial actions don't carry a comment field
	// so the legacy-prose warning path doesn't apply — they're always drops.
	for idx := range table.InitialActions {
		action := &table.InitialActions[idx]
		if action.DSL != "" {
			postfix, err := i.elCompiler.CompileAction(action.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("initial action %d: %v", idx+1, err))
				if i.stats != nil {
					i.stats.AddDrop(table.TableName, 0,
						fmt.Sprintf("initial action %d", idx+1), err.Error())
				}
				continue
			}
			i.warnNumericDowngrade(table.TableName,
				fmt.Sprintf("initial action %d", idx+1), action.Postfix, postfix)
			action.Postfix = postfix
			if i.stats != nil {
				i.stats.Compiled++
			}
		}
	}

	// Compile contexts (iterators, locals, and entity pushes that run before
	// the conditions). These were previously left uncompiled, so a table with
	// a "for all ..." context round-tripped to an empty <context_postfix> and
	// the strict loader rejected it.
	for idx := range table.Contexts.Details {
		ctx := &table.Contexts.Details[idx]
		if ctx.DSL != "" {
			postfix, err := i.elCompiler.CompileContext(ctx.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("context %d: %v", idx+1, err))
				recordCompileFailure(table.TableName,
					fmt.Sprintf("context %d", idx+1),
					ctx.DSL, ctx.Comment, err)
				continue
			}
			i.warnNumericDowngrade(table.TableName,
				fmt.Sprintf("context %d", idx+1), ctx.Postfix, postfix)
			ctx.Postfix = postfix
			if i.stats != nil {
				i.stats.Compiled++
			}
		}
	}

	// Compile conditions
	for idx := range table.Conditions {
		cond := &table.Conditions[idx]
		if cond.DSL != "" {
			postfix, err := i.elCompiler.CompileCondition(cond.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("condition %s: %v", cond.Number, err))
				recordCompileFailure(table.TableName,
					fmt.Sprintf("condition %s", cond.Number),
					cond.DSL, cond.Comment, err)
				continue
			}
			i.warnNumericDowngrade(table.TableName,
				"condition "+cond.Number, cond.Postfix, postfix)
			cond.Postfix = postfix
			if i.stats != nil {
				i.stats.Compiled++
			}
		}
	}

	// Compile actions
	for idx := range table.Actions {
		action := &table.Actions[idx]
		if action.DSL != "" {
			postfix, err := i.elCompiler.CompileAction(action.DSL)
			if err != nil {
				errors = append(errors, fmt.Sprintf("action %s: %v", action.Number, err))
				recordCompileFailure(table.TableName,
					fmt.Sprintf("action %s", action.Number),
					action.DSL, action.Comment, err)
				continue
			}
			i.warnNumericDowngrade(table.TableName,
				"action "+action.Number, action.Postfix, postfix)
			action.Postfix = postfix
			if i.stats != nil {
				i.stats.Compiled++
			}
		}
	}

	// Only propagate as an error if at least one failure was a real drop
	// (not a legacy-prose warning). Warning-only compile failures don't
	// fail the build — they're tracked in stats.Warnings and printed in
	// the summary.
	if len(errors) > 0 && hadDrop {
		return fmt.Errorf("EL compilation errors in table %s: %s",
			table.TableName, strings.Join(errors, "; "))
	}

	// Mark the table as EL-compiled
	table.ELCompiled = true

	return nil
}

// MergeTables merges multiple DecisionTablesXML into one.
func MergeTables(tables ...*DecisionTablesXML) *DecisionTablesXML {
	result := &DecisionTablesXML{}
	for _, t := range tables {
		if t != nil {
			result.Tables = append(result.Tables, t.Tables...)
		}
	}
	return result
}

// SortTablesByNumber sorts tables by their TABLE_NUMBER field.
func SortTablesByNumber(tables *DecisionTablesXML) {
	sort.Slice(tables.Tables, func(i, j int) bool {
		numI, errI := strconv.Atoi(tables.Tables[i].AttributeFields.TableNumber)
		numJ, errJ := strconv.Atoi(tables.Tables[j].AttributeFields.TableNumber)
		if errI != nil || errJ != nil {
			return tables.Tables[i].TableName < tables.Tables[j].TableName
		}
		return numI < numJ
	})
}

// countDSLRows reports how many rows of a table carry EL DSL that would need
// compiling. Used to tell "nothing to compile" apart from "a compiler was
// needed and none was wired" — the two cases look identical at the point
// compilation is skipped, and conflating them is what made #929 silent.
func countDSLRows(table *DecisionTableXML) int {
	n := 0
	count := func(dsl string) {
		if strings.TrimSpace(dsl) != "" {
			n++
		}
	}
	for _, c := range table.Contexts.Details {
		count(c.DSL)
	}
	for _, a := range table.EffectiveInitialActions() {
		count(a.DSL)
	}
	for _, c := range table.Conditions {
		count(c.DSL)
	}
	for _, a := range table.Actions {
		count(a.DSL)
	}
	return n
}

// numericDowngrades are the operator substitutions that quietly change what a
// number means. Key is what the committed postfix had; value is what the
// recompile produced and why it matters.
//
// Only losses are listed. Compiling `fp/` where the old artifact had `/` is an
// upgrade and says nothing worrying, so it is not here.
var numericDowngrades = map[string]struct{ to, meaning string }{
	"fphalfup/": {"fp/", "round-half-up division became truncating"},
	"fp/":       {"/", "fixed-point division became integer"},
	"fp*":       {"*", "fixed-point multiply became integer"},
	"fp+":       {"+", "fixed-point add became integer"},
	"fp-":       {"-", "fixed-point subtract became integer"},
	"cvfp":      {"cvi", "value stored as integer instead of fixed-point"},
}

// warnNumericDowngrade reports a recompile that silently weakens arithmetic.
//
// A committed artifact can carry postfix an older compiler produced from DSL
// the current one reads differently. The staking rules are the case in point
// (#1019): the DSL says `weighted_balance * staker_budget / total_weighted`,
// plain `/`, which compiles to truncating `fp/` — but the committed postfix
// says `fphalfup/`, because an older compiler conflated the two forms. The
// spec requires round-half-up and their own test asserts it, so the correct
// behaviour was being held up by a stale artifact. The first Excel-authored
// rebuild would have switched that division to truncation with no drops
// reported and exit 0.
//
// This is deliberately a warning, not a drop. Recompiling legitimately changes
// postfix — #1015's associativity fix rewrote multiply chains across the
// samples — so refusing would block real work. What must not happen is the
// change going unmentioned.
//
// Only whole tokens count: postfix is space-separated, and matching on
// substrings would see `fp/` inside `fphalfup/` and report every correct
// round-half-up row as a downgrade.
func (i *DTImporter) warnNumericDowngrade(tableName, item, before, after string) {
	if i.stats == nil || before == "" || before == after {
		return
	}
	old, recompiled := countOps(before), countOps(after)
	for lost, d := range numericDowngrades {
		// The operator has to actually disappear, and its weaker form has to
		// actually appear, before this is a downgrade rather than an edit.
		if old[lost] > recompiled[lost] && recompiled[d.to] > old[d.to] {
			i.stats.AddWarning(tableName, 0, item,
				fmt.Sprintf("recompiling changed %s to %s: %s. The DSL and the committed "+
					"postfix disagree — say what you mean in the DSL (for round-half-up, "+
					"`divide X by Y rounding by 0.5fp`) and rebuild", lost, d.to, d.meaning))
		}
	}
}

// countOps tallies whole-token occurrences in a postfix string.
func countOps(postfix string) map[string]int {
	n := make(map[string]int)
	for _, tok := range strings.Fields(postfix) {
		n[tok]++
	}
	return n
}

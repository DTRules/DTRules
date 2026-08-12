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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const (
	maxCol         = 16
	narrowColWidth = 5.0 // Decision columns (Y/N/X values)
	wideColWidth   = 40.0

	colorEDDEntityHeader = "D0D8E8"
	colorEDDInput        = "FFFFC0"
)

// Exporter exports decision tables and EDD to Excel format.
type Exporter struct {
	ruleSet *session.RuleSet
	// scopedTo is the workbook key an export is currently confined to, or ""
	// for a whole-project export. Set for the duration of one workbook's
	// export so the EDD sheet carries that workbook's own declarations and
	// nobody else's (#1109).
	scopedTo string
}

// NewExporter creates a new Excel exporter.
func NewExporter(rs *session.RuleSet) *Exporter {
	return &Exporter{ruleSet: rs}
}

// ExportDecisionTables exports all decision tables to a single Excel file.
// Tables are ordered by TABLE_NUMBER to match execution order.
func (e *Exporter) ExportDecisionTables(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)
	tables := e.getAllDecisionTables()
	sortTablesByTableNumber(tables)

	styler, err := NewStyler(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	for _, dt := range tables {
		if err := e.writeDecisionTable(f, dt, styler); err != nil {
			return fmt.Errorf("failed to write table %s: %w", dt.GetName(), err)
		}
	}

	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	return f.SaveAs(filename)
}

// ExportDecisionTablesOwnedBy writes only the tables that belong in filename,
// judged by each table's xls_file, and reports how many it wrote.
//
// ExportDecisionTables writes every table in the rule set, which is right for
// a caller producing one workbook from a whole project and destructive for one
// refreshing a project's existing workbooks in place: run over N workbooks it
// writes the rule set N times and every workbook ends up holding every table.
// A no-op `dtrules table put` on SinusitisTherapy took service1_medication.xlsx
// from 3 sheets to 6 and therapy.xlsx from 1 to 6, turning a verified project
// red without any rule being edited (#1077).
//
// Matching is on base name with the extension dropped, because the recorded
// xls_file is historically inconsistent about it -- the same table may be
// recorded as Foo.xls, Foo.xlsx or a path -- and case-insensitively, because
// nothing about a name in DTRules is case-sensitive.
//
// Returns 0 when nothing claims the workbook -- neither a table nor an entity.
// Callers should treat that as "leave the file alone": overwriting it with
// nothing would empty a workbook whose contents merely record a different
// spelling.
//
// A workbook may hold entities and no tables, and that case has to be written
// too. Returning early on "no tables" meant an EDD-only workbook was never
// refreshed at all, so every `dtrules edd` edit updated the XML and left the
// workbook stale -- and the next `build --from-excel` restored the XML from
// that stale workbook, silently reverting the edit. Folding 401 fields into
// CorporateTax's master EDD survived until the next build and then vanished
// (#1094).
func (e *Exporter) ExportDecisionTablesOwnedBy(filename string) (int, error) {
	want := workbookKey(filename)

	var owned []*decisiontable.RDecisionTable
	for _, dt := range e.getAllDecisionTables() {
		if workbookKey(dt.GetFilePath()) == want {
			owned = append(owned, dt)
		}
	}
	ents := e.entitiesOwnedBy(filename)
	if len(owned) == 0 && len(ents) == 0 {
		return 0, nil
	}
	// Confine the EDD sheet to this workbook's own field declarations.
	e.scopedTo = want
	defer func() { e.scopedTo = "" }()
	sortTablesByTableNumber(owned)

	f := excelize.NewFile()
	defer f.Close()
	defaultSheet := f.GetSheetName(0)

	styler, err := NewStyler(f)
	if err != nil {
		return 0, fmt.Errorf("failed to create styles: %w", err)
	}
	for _, dt := range owned {
		if err := e.writeDecisionTable(f, dt, styler); err != nil {
			return 0, fmt.Errorf("failed to write table %s: %w", dt.GetName(), err)
		}
	}

	// The workbook's own entities go back with its tables. Writing decision
	// tables alone deleted the EDD sheet from every workbook that carried one,
	// on every authoring write -- and Excel is the system of record, so that
	// deleted the dictionary from the record. The next build then imported a
	// workbook with no types and compiled the DSL untyped, turning `f<` into
	// `<`: two fixed-point amounts compared as integers (#1094).
	if len(ents) > 0 {
		if err := e.writeEDDSheet(f, styler, "EDD", ents); err != nil {
			return 0, fmt.Errorf("failed to write EDD sheet: %w", err)
		}
	} else if err := refuseToDropEDDSheet(filename); err != nil {
		return 0, err
	}

	if len(f.GetSheetList()) > 1 {
		f.DeleteSheet(defaultSheet)
	}
	// Count both: the caller uses this to decide whether anything was
	// written, and an EDD-only workbook writes no tables.
	return len(owned) + len(ents), f.SaveAs(filename)
}

// refuseToDropEDDSheet blocks an export that would replace a workbook holding
// an EDD sheet with one holding none.
//
// Losing a sheet is losing rules. Excel is the system of record, so a refresh
// that drops the dictionary deletes it from the record, and the next build
// imports a workbook with no types and compiles the DSL untyped -- `f<` becomes
// `<`, two fixed-point amounts compared as integers (#1094).
//
// The condition is still narrower than "a refresh must not reduce a workbook's
// sheet count": `dtrules table delete` legitimately removes a sheet, so that
// rule would refuse real work. What has no legitimate form is a workbook that
// keeps its tables and loses its dictionary.
//
// This fires when no entity claims the workbook. That was too broad while
// ownership followed the merged entity -- TaxReturn's 50 state EDDs each
// declare the shared `result` naming their own workbook, and only one survived
// the merge, so 49 workbooks would have been refused for a bug in the
// ownership test rather than in the project. Ownership now follows the field
// declaration (#1109), so an unclaimed workbook means what it says.
func refuseToDropEDDSheet(filename string) error {
	existing, err := excelize.OpenFile(filename)
	if err != nil {
		// No file yet, or unreadable: nothing is being dropped.
		return nil
	}
	defer existing.Close()
	for _, sheet := range existing.GetSheetList() {
		if !strings.EqualFold(strings.TrimSpace(sheet), "EDD") {
			continue
		}
		return fmt.Errorf("refusing to export %s: it has an EDD sheet and no entity "+
			"claims this workbook, so the export would delete the dictionary. "+
			"The entities come from the loaded rule set — load the project's EDD "+
			"files before exporting", filepath.Base(filename))
	}
	return nil
}

// workbookKey reduces a workbook reference to something comparable: base name,
// no extension, lowercased.
func workbookKey(ref string) string {
	base := filepath.Base(strings.TrimSpace(ref))
	return strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
}

// getAllDecisionTables returns all decision tables from the ruleset.
func (e *Exporter) getAllDecisionTables() []*decisiontable.RDecisionTable {
	tableNames := e.ruleSet.GetDecisionTableNames()
	tables := make([]*decisiontable.RDecisionTable, 0, len(tableNames))

	for _, rname := range tableNames {
		dtObj := e.ruleSet.GetEntityFactory().FindDecisionTable(rname)
		if dtObj == nil {
			continue
		}
		dt, ok := dtObj.(*decisiontable.RDecisionTable)
		if !ok {
			continue
		}
		tables = append(tables, dt)
	}
	return tables
}

// sortTablesByTableNumber sorts decision tables by their TABLE_NUMBER field.
// Tables without a TABLE_NUMBER are sorted alphabetically after numbered tables.
func sortTablesByTableNumber(tables []*decisiontable.RDecisionTable) {
	sort.Slice(tables, func(i, j int) bool {
		numI := tables[i].GetField("TABLE_NUMBER")
		numJ := tables[j].GetField("TABLE_NUMBER")

		intI, errI := strconv.Atoi(numI)
		intJ, errJ := strconv.Atoi(numJ)

		if errI == nil && errJ == nil {
			return intI < intJ
		}
		if errI == nil {
			return true
		}
		if errJ == nil {
			return false
		}
		return tables[i].GetName() < tables[j].GetName()
	})
}

// ExportDecisionTablesToDir exports decision tables grouped by xls_file to a directory.
func (e *Exporter) ExportDecisionTablesToDir(dir string) error {
	groups := make(map[string][]*decisiontable.RDecisionTable)
	tableNames := e.ruleSet.GetDecisionTableNames()

	for _, rname := range tableNames {
		dtObj := e.ruleSet.GetEntityFactory().FindDecisionTable(rname)
		if dtObj == nil {
			continue
		}
		dt, ok := dtObj.(*decisiontable.RDecisionTable)
		if !ok {
			continue
		}

		filePath := dt.GetFilePath()
		if filePath == "" {
			filePath = "Other"
		}
		xlsFile := strings.TrimSuffix(strings.TrimSuffix(filePath, ".xls"), ".xlsx") + ".xlsx"
		groups[xlsFile] = append(groups[xlsFile], dt)
	}

	for _, tables := range groups {
		sortTablesByTableNumber(tables)
	}

	for xlsFile, tables := range groups {
		if err := e.writeDecisionTableGroup(dir, xlsFile, tables); err != nil {
			return err
		}
	}

	return e.writeDecisionTableIndex(dir, groups)
}

func (e *Exporter) writeDecisionTableGroup(dir, xlsFile string, tables []*decisiontable.RDecisionTable) error {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)

	styler, err := NewStyler(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	for _, dt := range tables {
		if err := e.writeDecisionTable(f, dt, styler); err != nil {
			return fmt.Errorf("failed to write table %s: %w", dt.GetName(), err)
		}
	}

	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	return f.SaveAs(dir + "/" + xlsFile)
}

func (e *Exporter) writeDecisionTableIndex(dir string, groups map[string][]*decisiontable.RDecisionTable) error {
	var sb strings.Builder

	sb.WriteString("# Decision Tables Index\n\n")
	sb.WriteString("This folder contains decision tables organized by functional area.\n\n")

	groupNames := make([]string, 0, len(groups))
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| File | Tables | Description |\n")
	sb.WriteString("|------|--------|-------------|\n")

	totalTables := 0
	for _, name := range groupNames {
		tables := groups[name]
		totalTables += len(tables)
		desc := getGroupDescription(name)
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %d | %s |\n", name, name, len(tables), desc))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** | |\n\n", totalTables))

	sb.WriteString("## Tables by File\n\n")

	for _, name := range groupNames {
		tables := groups[name]
		sb.WriteString(fmt.Sprintf("### %s\n\n", name))
		sb.WriteString("| Table | Description |\n")
		sb.WriteString("|-------|-------------|\n")

		for _, dt := range tables {
			fields := dt.GetFields()
			comment := fields["COMMENTS"]
			if len(comment) > 80 {
				comment = comment[:80] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", dt.GetName(), comment))
		}
		sb.WriteString("\n")
	}

	return writeFile(dir+"/index.md", sb.String())
}

func getGroupDescription(filename string) string {
	name := strings.TrimSuffix(filename, ".xlsx")
	descriptions := map[string]string{
		"MainFlow":       "Entry point and main tax calculation flow",
		"Income":         "Income processing tables",
		"Deductions":     "Deduction calculations",
		"TaxCalculation": "Tax brackets and liability calculation",
		"Credits":        "Tax credit calculations",
		"SpecialForms":   "Special IRS forms (K-1, foreign, etc.)",
		"Validation":     "Test validation tables",
		"Helpers":        "Helper and utility tables",
		"Other":          "Uncategorized tables",
	}
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return ""
}

func writeFile(filepath, content string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

// ExportEDD exports the Entity Data Dictionary to an Excel file.
func (e *Exporter) ExportEDD(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	styler, err := NewStyler(f)
	if err != nil {
		return err
	}
	eddStyles, err := e.newEDDStyles(f)
	if err != nil {
		return err
	}

	e.setEDDColumnWidths(f, sheet)
	e.writeEDDHeaders(f, sheet, styler, 1)
	FreezePaneAtRow2(f, sheet)

	entities := e.ruleSet.GetEntityFactory().GetRefEntities()
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].GetName().StringValue() < entities[j].GetName().StringValue()
	})

	e.writeEDDEntities(f, sheet, entities, styler, eddStyles, 2)

	return f.SaveAs(filename)
}

// ExportCombinedWorkbook exports both decision tables and EDD to a single Excel file.
func (e *Exporter) ExportCombinedWorkbook(filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)

	styler, err := NewStyler(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	tables := e.getAllDecisionTables()
	sortTablesByTableNumber(tables)

	for _, dt := range tables {
		if err := e.writeDecisionTable(f, dt, styler); err != nil {
			return fmt.Errorf("failed to write table %s: %w", dt.GetName(), err)
		}
	}

	if entities := e.allEntities(); len(entities) > 0 {
		if err := e.writeEDDSheet(f, styler, "EDD", entities); err != nil {
			return fmt.Errorf("failed to write EDD sheet: %w", err)
		}
	}

	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	return f.SaveAs(filename)
}

// writeEDDSheet writes the EDD data to a sheet in an existing workbook.
// Row 1 carries the "EDD: EDD" type marker so mixed-workbook importers can
// detect sheet type from A1 without relying on the sheet name.
// writeEDDSheet adds an EDD sheet holding the given entities. Callers that
// want the whole dictionary pass e.allEntities().
func (e *Exporter) writeEDDSheet(f *excelize.File, styler *Styler, sheetName string, entities []*entity.REntity) error {
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	eddStyles, err := e.newEDDStyles(f)
	if err != nil {
		return err
	}

	e.setEDDColumnWidths(f, sheetName)

	// Row 1: type marker for mixed-workbook sheet-type detection
	f.SetCellValue(sheetName, "A1", "EDD: EDD")
	f.SetCellStyle(sheetName, "A1", "A1", styler.HeaderStyle)

	e.writeEDDHeaders(f, sheetName, styler, 2)
	FreezePaneAtRow3(f, sheetName)

	e.writeEDDEntities(f, sheetName, entities, styler, eddStyles, 3)

	return nil
}

// allEntities is every entity in the rule set, in name order.
func (e *Exporter) allEntities() []*entity.REntity {
	entities := e.ruleSet.GetEntityFactory().GetRefEntities()
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].GetName().StringValue() < entities[j].GetName().StringValue()
	})
	return entities
}

// entitiesOwnedBy is the entities whose workbook is filename, in name order.
//
// Deliberately GetXlsFile and not GetFilePath: the latter prefers the EDD's
// own file identity ("CHIP_edd", from <file_metadata><file_path>) and falls
// back to the workbook only when that is absent. That is the right answer for
// ExportEDDToDir, which names output files after the EDD, and the wrong one
// here, where the question is which workbook an entity belongs in. Matching on
// it silently found nothing and the EDD sheet went on being dropped.
func (e *Exporter) entitiesOwnedBy(filename string) []*entity.REntity {
	want := workbookKey(filename)
	var owned []*entity.REntity
	for _, ent := range e.allEntities() {
		if workbookKey(ent.GetXlsFile()) == want || e.declaresFieldsIn(ent, want) {
			owned = append(owned, ent)
		}
	}
	return owned
}

// declaresFieldsIn reports whether any of the entity's fields were declared by
// the named workbook's EDD.
//
// An entity is one thing at run time and may be declared in many files. The
// merged REntity keeps one xls_file, so matching on that alone claimed only
// the workbook that happened to win: TaxReturn's 50 state EDDs each add fields
// to the shared `result` naming their own workbook, and 49 of them were
// claimed by nothing and lost their EDD sheet on every refresh (#1109).
func (e *Exporter) declaresFieldsIn(ent *entity.REntity, want string) bool {
	for _, entry := range ent.GetEntries() {
		if entry != nil && entry.SourceXlsFile != "" && workbookKey(entry.SourceXlsFile) == want {
			return true
		}
	}
	return false
}

// includes decides whether one field belongs in the EDD sheet being written.
//
// When an export is scoped to a workbook, a field goes in only if that
// workbook's EDD declared it -- otherwise every state workbook would receive
// the whole merged entity, all 50 states' fields, and the next build would
// write each of those back into every state's EDD file. Fields with no
// recorded source (synthesized entries, EDDs that declare no xls_file) stay
// with the entity's own workbook, which is where they were before.
func (e *Exporter) includes(entry *entity.EntityEntry) bool {
	if e.scopedTo == "" {
		return true
	}
	if entry.SourceXlsFile == "" {
		return workbookKey(e.ownerOf(entry)) == e.scopedTo
	}
	return workbookKey(entry.SourceXlsFile) == e.scopedTo
}

// ownerOf is the workbook an unsourced field falls back to: its entity's.
func (e *Exporter) ownerOf(entry *entity.EntityEntry) string {
	if entry.Entity == nil {
		return ""
	}
	return entry.Entity.GetXlsFile()
}

// ExportEDDToDir exports entities grouped by xls_file to a directory.
func (e *Exporter) ExportEDDToDir(dir string) error {
	groups := make(map[string][]*entity.REntity)
	entities := e.ruleSet.GetEntityFactory().GetRefEntities()

	for _, ent := range entities {
		filePath := ent.GetFilePath()
		if filePath == "" {
			filePath = "Other"
		}
		xlsFile := strings.TrimSuffix(strings.TrimSuffix(filePath, ".xls"), ".xlsx") + ".xlsx"
		groups[xlsFile] = append(groups[xlsFile], ent)
	}

	for _, ents := range groups {
		sort.Slice(ents, func(i, j int) bool {
			return ents[i].GetName().StringValue() < ents[j].GetName().StringValue()
		})
	}

	for xlsFile, ents := range groups {
		if err := e.writeEDDGroup(dir, xlsFile, ents); err != nil {
			return err
		}
	}

	return e.writeEDDIndex(dir, groups)
}

func (e *Exporter) writeEDDGroup(dir, xlsFile string, entities []*entity.REntity) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	styler, err := NewStyler(f)
	if err != nil {
		return err
	}
	eddStyles, err := e.newEDDStyles(f)
	if err != nil {
		return err
	}

	e.setEDDColumnWidths(f, sheet)
	e.writeEDDHeaders(f, sheet, styler, 1)
	FreezePaneAtRow2(f, sheet)
	e.writeEDDEntities(f, sheet, entities, styler, eddStyles, 2)

	return f.SaveAs(dir + "/" + xlsFile)
}

func (e *Exporter) writeEDDIndex(dir string, groups map[string][]*entity.REntity) error {
	var sb strings.Builder

	sb.WriteString("# Entity Data Dictionary Index\n\n")
	sb.WriteString("This folder contains entity definitions organized by functional area.\n\n")

	groupNames := make([]string, 0, len(groups))
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| File | Entities | Attributes |\n")
	sb.WriteString("|------|----------|------------|\n")

	totalEntities := 0
	totalAttrs := 0
	for _, name := range groupNames {
		ents := groups[name]
		entCount := len(ents)
		attrCount := 0
		for _, ent := range ents {
			attrCount += len(ent.GetAttributeNames()) - 2
		}
		totalEntities += entCount
		totalAttrs += attrCount
		sb.WriteString(fmt.Sprintf("| [%s](%s) | %d | %d |\n", name, name, entCount, attrCount))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** | **%d** |\n\n", totalEntities, totalAttrs))

	sb.WriteString("## Entities by File\n\n")

	for _, name := range groupNames {
		ents := groups[name]
		sb.WriteString(fmt.Sprintf("### %s\n\n", name))
		sb.WriteString("| Entity | Attributes | Description |\n")
		sb.WriteString("|--------|------------|-------------|\n")

		for _, ent := range ents {
			attrCount := len(ent.GetAttributeNames()) - 2
			comment := ent.GetComment()
			if len(comment) > 60 {
				comment = comment[:60] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", ent.GetName().StringValue(), attrCount, comment))
		}
		sb.WriteString("\n")
	}

	return writeFile(dir+"/index.md", sb.String())
}

// eddExtraStyles holds EDD-specific styles beyond the shared Styler.
type eddExtraStyles struct {
	entityHeader int
	rowEven      int
	rowOdd       int
	attribute    int
	typeString   int
	typeDouble   int
	typeInteger  int
	typeBoolean  int
	typeDate     int
	typeArray    int
	typeDefault  int
	input        int
	comment      int
}

func (e *Exporter) newEDDStyles(f *excelize.File) (*eddExtraStyles, error) {
	s := &eddExtraStyles{}
	var err error

	thin := thinBorder()
	centered := &excelize.Alignment{Horizontal: "center", Vertical: "center"}

	s.entityHeader, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: fontSansSerif, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorEDDEntityHeader}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.rowEven, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.rowOdd, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F2F2F2"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.attribute, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: fontSansSerif, Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.typeString, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "0070C0"}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeDouble, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "00B050"}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeInteger, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "00B050"}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeBoolean, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "ED7D31"}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeDate, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "7030A0"}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeArray, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10, Color: "C00000", Italic: true}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}
	s.typeDefault, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: fontSansSerif, Size: 10}, Alignment: centered, Border: thin})
	if err != nil {
		return nil, err
	}

	s.input, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorEDDInput}},
		Alignment: centered,
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.comment, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 9, Color: "666666"},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (e *Exporter) getTypeStyle(s *eddExtraStyles, typeName string) int {
	switch strings.ToLower(typeName) {
	case "string":
		return s.typeString
	case "double":
		return s.typeDouble
	case "integer":
		return s.typeInteger
	case "boolean":
		return s.typeBoolean
	case "date":
		return s.typeDate
	case "array":
		return s.typeArray
	default:
		return s.typeDefault
	}
}

func (e *Exporter) setEDDColumnWidths(f *excelize.File, sheet string) {
	AutoWidth(f, sheet, "A", 18)
	AutoWidth(f, sheet, "B", 25)
	AutoWidth(f, sheet, "C", 10)
	AutoWidth(f, sheet, "D", 12)
	AutoWidth(f, sheet, "E", 18)
	AutoWidth(f, sheet, "F", 8)
	AutoWidth(f, sheet, "G", 8)
	AutoWidth(f, sheet, "H", 55)
	AutoWidth(f, sheet, "I", 8)
	AutoWidth(f, sheet, "J", 30)
	AutoWidth(f, sheet, "K", 16)
	AutoWidth(f, sheet, "L", 30)
	AutoWidth(f, sheet, "M", 18)
}

func (e *Exporter) writeEDDHeaders(f *excelize.File, sheet string, styler *Styler, startRow int) {
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description", "Collect", "Question", "Q Type", "Options", "Reference"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, startRow)
		styler.ApplyHeader(f, sheet, cell, cell, cell, header)
	}
}

func (e *Exporter) writeEDDEntities(f *excelize.File, sheet string, entities []*entity.REntity, styler *Styler, s *eddExtraStyles, startRow int) {
	row := startRow
	for _, ent := range entities {
		entityName := ent.GetName().StringValue()

		attrNames := ent.GetAttributeNames()
		sort.Slice(attrNames, func(i, j int) bool {
			return attrNames[i].StringValue() < attrNames[j].StringValue()
		})

		attrCount := 0
		for _, attrName := range attrNames {
			if attrName.StringValue() != entityName && attrName.StringValue() != "mapping*key" {
				if entry := ent.GetEntry(attrName); entry != nil && e.includes(entry) {
					attrCount++
				}
			}
		}

		if attrCount == 0 {
			continue
		}

		f.SetCellValue(sheet, cellName(1, row), entityName)
		f.SetCellValue(sheet, cellName(2, row), fmt.Sprintf("(%d attributes)", attrCount))
		for col := 1; col <= eddColumnCount; col++ {
			f.SetCellStyle(sheet, cellName(col, row), cellName(col, row), s.entityHeader)
		}
		row++

		attrRow := 0
		for _, attrName := range attrNames {
			if attrName.StringValue() == entityName || attrName.StringValue() == "mapping*key" {
				continue
			}

			entry := ent.GetEntry(attrName)
			if entry == nil || !e.includes(entry) {
				continue
			}

			access := ""
			if entry.Readable {
				access += "r"
			}
			if entry.Writable {
				access += "w"
			}

			rowStyle := s.rowEven
			if attrRow%2 == 1 {
				rowStyle = s.rowOdd
			}

			f.SetCellValue(sheet, cellName(1, row), "")
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), rowStyle)

			// The authored spelling, not the interned one — see
			// EntityEntry.AuthoredName (#1040).
			fieldName := attrName.StringValue()
			if entry.AuthoredName != "" {
				fieldName = entry.AuthoredName
			}
			f.SetCellValue(sheet, cellName(2, row), fieldName)
			f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), s.attribute)

			typeStyle := e.getTypeStyle(s, entry.Type.String())
			f.SetCellValue(sheet, cellName(3, row), entry.Type.String())
			f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), typeStyle)

			f.SetCellValue(sheet, cellName(4, row), entry.SubType)
			f.SetCellStyle(sheet, cellName(4, row), cellName(4, row), rowStyle)

			f.SetCellValue(sheet, cellName(5, row), entry.DefaultTxt)
			f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), rowStyle)

			inputStyle := rowStyle
			if entry.Input != "" {
				inputStyle = s.input
			}
			f.SetCellValue(sheet, cellName(6, row), entry.Input)
			f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), inputStyle)

			f.SetCellValue(sheet, cellName(7, row), access)
			f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), rowStyle)

			f.SetCellValue(sheet, cellName(8, row), entry.Comment)
			f.SetCellStyle(sheet, cellName(8, row), cellName(8, row), s.comment)

			// Collect + question metadata (#850), columns I–M.
			collectTxt, qText, qType, qOpts, qRef := "", "", "", "", ""
			if entry.Collect {
				collectTxt = "true"
				if entry.Question != nil {
					qText = entry.Question.Text
					qType = entry.Question.Type
					xo := make([]EDDXMLOption, 0, len(entry.Question.Options))
					for _, o := range entry.Question.Options {
						xo = append(xo, EDDXMLOption{Value: o.Value, Label: o.Label})
					}
					qOpts = encodeEDDOptions(xo)
					qRef = encodeEDDRef(entry.Question.RefLow, entry.Question.RefHigh, entry.Question.Units)
				}
			}
			f.SetCellValue(sheet, cellName(9, row), collectTxt)
			f.SetCellStyle(sheet, cellName(9, row), cellName(9, row), rowStyle)
			f.SetCellValue(sheet, cellName(10, row), qText)
			f.SetCellStyle(sheet, cellName(10, row), cellName(10, row), rowStyle)
			f.SetCellValue(sheet, cellName(11, row), qType)
			f.SetCellStyle(sheet, cellName(11, row), cellName(11, row), rowStyle)
			f.SetCellValue(sheet, cellName(12, row), qOpts)
			f.SetCellStyle(sheet, cellName(12, row), cellName(12, row), rowStyle)
			f.SetCellValue(sheet, cellName(13, row), qRef)
			f.SetCellStyle(sheet, cellName(13, row), cellName(13, row), rowStyle)

			row++
			attrRow++
		}
	}
}

func (e *Exporter) writeDecisionTable(f *excelize.File, dt *decisiontable.RDecisionTable, styler *Styler) error {
	// AuthoredName, not GetName: the interned name carries whatever casing was
	// seen first anywhere in the project, so writing it back renames the
	// author's table (#1040).
	name := dt.AuthoredName()
	sheetName := sanitizeSheetName(name)

	for i := 1; ; i++ {
		idx, _ := f.GetSheetIndex(sheetName)
		if idx == -1 {
			break
		}
		sheetName = fmt.Sprintf("%s_%d", sanitizeSheetName(name)[:min(28, len(name))], i)
	}

	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	_ = idx

	AutoWidth(f, sheetName, "A", 20)
	AutoWidth(f, sheetName, "B", wideColWidth)
	AutoWidth(f, sheetName, "C", wideColWidth)

	numCols := dt.GetMaxCol()
	if numCols < maxCol {
		numCols = maxCol
	}

	for i := 0; i < numCols; i++ {
		col, _ := excelize.ColumnNumberToName(4 + i)
		f.SetColWidth(sheetName, col, col, narrowColWidth)
	}

	row := 1
	row = e.writeName(f, sheetName, dt, row, numCols, styler)
	row = e.writeType(f, sheetName, dt, row, numCols, styler)
	row = e.writeFields(f, sheetName, dt, row, numCols, styler)
	row++
	row = e.writeContexts(f, sheetName, dt, row, numCols, styler)
	row = e.writeInitialActions(f, sheetName, dt, row, numCols, styler)
	row = e.writeConditions(f, sheetName, dt, row, numCols, styler)
	row = e.writeActions(f, sheetName, dt, row, numCols, styler)
	e.writePolicyStatements(f, sheetName, dt, row, numCols, styler)

	return nil
}

func (e *Exporter) writeName(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	displayName := strings.ReplaceAll(dt.AuthoredName(), "_", " ")
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "DT: "+displayName)
	f.SetCellStyle(sheet, startCell, endCell, styler.BodyStyle)

	return row + 1
}

func (e *Exporter) writeType(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "Type: "+dt.GetTableType().String())
	f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
	return row + 1
}

func (e *Exporter) writeFields(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	fields := dt.GetFields()

	names := make([]string, 0, len(fields))
	for name := range fields {
		if strings.ToUpper(name) != "TYPE" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		value := fields[name]
		startCell := cellName(1, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		f.SetCellValue(sheet, startCell, name+": "+value)
		f.SetCellStyle(sheet, startCell, endCell, styler.BodyStyle)
		row++
	}

	return row
}

func (e *Exporter) writeContexts(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "CONTEXTS: COMMENTS")
	f.MergeCell(sheet, cellName(3, row), cellName(3+numCols, row))
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3+numCols, row), "DSL: CONTEXTS")
	row++

	contexts := dt.GetContexts()
	comments := dt.GetContextsComment()

	for i, ctx := range contexts {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)

		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		styler.ApplyBody(f, sheet, cellName(2, row), comment)

		startCell := cellName(3, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		styler.ApplyDSL(f, sheet, startCell, ctx)
		f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
		row++
	}

	row++
	return row
}

func (e *Exporter) writeInitialActions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "INITIAL ACTIONS: COMMENTS")
	f.MergeCell(sheet, cellName(3, row), cellName(3+numCols, row))
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3+numCols, row), "DSL: INITIAL ACTIONS")
	row++

	actions := dt.GetInitialActions()
	comments := dt.GetInitialActionsComment()

	for i, action := range actions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)

		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		styler.ApplyBody(f, sheet, cellName(2, row), comment)

		startCell := cellName(3, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		styler.ApplyDSL(f, sheet, startCell, action)
		f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
		row++
	}

	row++
	return row
}

func (e *Exporter) writeConditions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "CONDITIONS: COMMENTS")
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3, row), "DSL: CONDITIONS")

	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styler.HeaderStyle)
	}
	row++

	conditions := dt.GetConditions()
	comments := dt.GetConditionsComment()
	condTable := dt.GetConditionTable()

	for i, cond := range conditions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)

		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		styler.ApplyBody(f, sheet, cellName(2, row), comment)
		styler.ApplyDSL(f, sheet, cellName(3, row), cond)

		if i < len(condTable) {
			for j := 0; j < numCols; j++ {
				val := ""
				// "-" is written through, not blanked. It and an absent entry
				// mean the same thing to the runtime (see Stepper.processColumn:
				// `if !hasVal || colVal == "-" { continue }`), but blanking it
				// here made the round trip lossy: the XML's "-" came back as
				// nothing, so an Excel-authored rebuild always differed from the
				// committed XML and the #1010 verify gate reported a difference
				// no author could act on. Ten of them on the staking rules
				// (#1017). "-" is also the conventional don't-care notation, so
				// showing it distinguishes "considered, irrelevant" from "not
				// filled in yet".
				if j < len(condTable[i]) {
					val = condTable[i][j]
				}
				cell := cellName(4+j, row)
				f.SetCellValue(sheet, cell, val)
				f.SetCellStyle(sheet, cell, cell, styler.BodyStyle)
			}
		}
		row++
	}

	row++
	return row
}

func (e *Exporter) writeActions(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	// Use SeparatorStyle on action column numbers to visually mark the condition/action boundary.
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "ACTIONS: COMMENTS")
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3, row), "DSL: ACTIONS")

	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styler.SeparatorStyle)
	}
	row++

	actions := dt.GetActions()
	comments := dt.GetActionsComment()
	actionTable := dt.GetActionTable()

	for i, action := range actions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)

		comment := ""
		if i < len(comments) {
			comment = comments[i]
		}
		styler.ApplyBody(f, sheet, cellName(2, row), comment)
		styler.ApplyDSL(f, sheet, cellName(3, row), action)

		if i < len(actionTable) {
			for j := 0; j < numCols; j++ {
				val := ""
				// Same as conditions above: "-" round-trips rather than being
				// blanked (#1017).
				if j < len(actionTable[i]) {
					val = actionTable[i][j]
				}
				cell := cellName(4+j, row)
				f.SetCellValue(sheet, cell, val)
				f.SetCellStyle(sheet, cell, cell, styler.BodyStyle)
			}
		}
		row++
	}

	row++
	return row
}

func (e *Exporter) writePolicyStatements(f *excelize.File, sheet string, dt *decisiontable.RDecisionTable, row, numCols int, styler *Styler) int {
	policyStatements := dt.GetPolicyStatements()

	hasStatements := false
	for i := 1; i < len(policyStatements); i++ {
		if policyStatements[i] != "" {
			hasStatements = true
			break
		}
	}
	if !hasStatements {
		return row
	}

	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "Policy:")
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3, row), "Policy Statements")

	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styler.HeaderStyle)
	}
	row++

	// One row per policy statement: column number in A, policy text in B
	for i := 1; i < len(policyStatements); i++ {
		if policyStatements[i] == "" {
			continue
		}

		f.SetCellValue(sheet, cellName(1, row), i)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)

		styler.ApplyBody(f, sheet, cellName(2, row), policyStatements[i])
		styler.ApplyDSL(f, sheet, cellName(3, row), "")

		for j := 0; j < numCols; j++ {
			cell := cellName(4+j, row)
			f.SetCellValue(sheet, cell, "")
			f.SetCellStyle(sheet, cell, cell, styler.BodyStyle)
		}

		row++
	}

	return row
}

// Helper functions

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func sanitizeSheetName(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 31 {
		name = name[:31]
	}
	for _, c := range []string{":", "\\", "/", "?", "*", "[", "]"} {
		name = strings.ReplaceAll(name, c, "")
	}
	return name
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

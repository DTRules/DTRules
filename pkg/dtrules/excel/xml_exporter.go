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

package excel

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// UnmarshalDecisionTablesXML parses raw XML bytes into a DecisionTablesXML model.
func UnmarshalDecisionTablesXML(data []byte) (*DecisionTablesXML, error) {
	var dt DecisionTablesXML
	if err := xml.Unmarshal(data, &dt); err != nil {
		return nil, err
	}
	return &dt, nil
}

// UnmarshalEDDXML parses raw XML bytes into an EDDXML model.
func UnmarshalEDDXML(data []byte) (*EDDXML, error) {
	var edd EDDXML
	if err := xml.Unmarshal(data, &edd); err != nil {
		return nil, err
	}
	return &edd, nil
}

// WriteDTXMLToExcel writes a DecisionTablesXML model to an xlsx file.
// It is a pure XML→xlsx conversion; no RuleSet or session is required.
func WriteDTXMLToExcel(dt *DecisionTablesXML, filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)

	styler, err := NewStyler(f)
	if err != nil {
		return fmt.Errorf("failed to create styles: %w", err)
	}

	tables := make([]DecisionTableXML, len(dt.Tables))
	copy(tables, dt.Tables)
	sortDecisionTableXMLByNumber(tables)

	for i := range tables {
		if err := writeDTXMLSheet(f, &tables[i], styler); err != nil {
			return fmt.Errorf("failed to write table %s: %w", tables[i].TableName, err)
		}
	}

	sheetList := f.GetSheetList()
	if len(sheetList) > 1 {
		f.DeleteSheet(defaultSheet)
	}

	return f.SaveAs(filename)
}

func sortDecisionTableXMLByNumber(tables []DecisionTableXML) {
	sort.SliceStable(tables, func(i, j int) bool {
		ni, erri := strconv.Atoi(tables[i].AttributeFields.TableNumber)
		nj, errj := strconv.Atoi(tables[j].AttributeFields.TableNumber)
		if erri == nil && errj == nil {
			return ni < nj
		}
		if erri == nil {
			return true
		}
		if errj == nil {
			return false
		}
		return tables[i].TableName < tables[j].TableName
	})
}

func writeDTXMLSheet(f *excelize.File, dt *DecisionTableXML, styler *Styler) error {
	name := dt.TableName
	sheetName := sanitizeSheetName(name)
	for i := 1; ; i++ {
		idx, _ := f.GetSheetIndex(sheetName)
		if idx == -1 {
			break
		}
		sheetName = fmt.Sprintf("%s_%d", sanitizeSheetName(name)[:min(28, len(name))], i)
	}

	if _, err := f.NewSheet(sheetName); err != nil {
		return err
	}

	AutoWidth(f, sheetName, "A", 20)
	AutoWidth(f, sheetName, "B", wideColWidth)
	AutoWidth(f, sheetName, "C", wideColWidth)

	numCols := maxColFromDTXML(dt)
	if numCols < maxCol {
		numCols = maxCol
	}
	for i := 0; i < numCols; i++ {
		col, _ := excelize.ColumnNumberToName(4 + i)
		f.SetColWidth(sheetName, col, col, narrowColWidth)
	}

	row := 1
	row = writeDTXMLName(f, sheetName, dt, row, numCols, styler)
	row = writeDTXMLType(f, sheetName, dt, row, numCols, styler)
	row = writeDTXMLFields(f, sheetName, dt, row, numCols, styler)
	row++
	row = writeDTXMLContexts(f, sheetName, dt, row, numCols, styler)
	row = writeDTXMLInitialActions(f, sheetName, dt, row, numCols, styler)
	row = writeDTXMLConditions(f, sheetName, dt, row, numCols, styler)
	row = writeDTXMLActions(f, sheetName, dt, row, numCols, styler)
	writeDTXMLPolicyStatements(f, sheetName, dt, row, numCols, styler)
	return nil
}

func maxColFromDTXML(dt *DecisionTableXML) int {
	n := 0
	for _, c := range dt.Conditions {
		for _, col := range c.Columns {
			if col.Number > n {
				n = col.Number
			}
		}
	}
	for _, a := range dt.Actions {
		for _, col := range a.Columns {
			if col.Number > n {
				n = col.Number
			}
		}
	}
	return n
}

func writeDTXMLName(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	displayName := strings.ReplaceAll(dt.TableName, "_", " ")
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "DT: "+displayName)
	f.SetCellStyle(sheet, startCell, endCell, styler.BodyStyle)
	return row + 1
}

func writeDTXMLType(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	startCell := cellName(1, row)
	endCell := cellName(3+numCols, row)
	f.MergeCell(sheet, startCell, endCell)
	f.SetCellValue(sheet, startCell, "Type: "+dt.AttributeFields.Type)
	f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
	return row + 1
}

func writeDTXMLFields(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	type kv struct{ k, v string }
	var fields []kv
	if dt.AttributeFields.Comments != "" {
		fields = append(fields, kv{"COMMENTS", dt.AttributeFields.Comments})
	}
	if dt.AttributeFields.TableNumber != "" {
		fields = append(fields, kv{"TABLE_NUMBER", dt.AttributeFields.TableNumber})
	}
	if dt.AttributeFields.FilePath != "" {
		fields = append(fields, kv{"FILE_PATH", dt.AttributeFields.FilePath})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].k < fields[j].k })

	for _, f2 := range fields {
		startCell := cellName(1, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		f.SetCellValue(sheet, startCell, f2.k+": "+f2.v)
		f.SetCellStyle(sheet, startCell, endCell, styler.BodyStyle)
		row++
	}
	return row
}

func writeDTXMLContexts(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "CONTEXTS: COMMENTS")
	f.MergeCell(sheet, cellName(3, row), cellName(3+numCols, row))
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3+numCols, row), "DSL: CONTEXTS")
	row++

	if !dt.Contexts.IsEmpty() {
		// Emit one Excel row per structured context detail, plus one row per
		// legacy <context_entity> directive so the user can see / edit both
		// forms.
		i := 0
		for _, ent := range dt.Contexts.Entities {
			i++
			f.SetCellValue(sheet, cellName(1, row), i)
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
			styler.ApplyBody(f, sheet, cellName(2, row), "context_entity")
			startCell := cellName(3, row)
			endCell := cellName(3+numCols, row)
			f.MergeCell(sheet, startCell, endCell)
			styler.ApplyDSL(f, sheet, startCell, ent)
			f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
			row++
		}
		for _, ctx := range dt.Contexts.Details {
			i++
			f.SetCellValue(sheet, cellName(1, row), i)
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
			styler.ApplyBody(f, sheet, cellName(2, row), ctx.Comment)
			startCell := cellName(3, row)
			endCell := cellName(3+numCols, row)
			f.MergeCell(sheet, startCell, endCell)
			styler.ApplyDSL(f, sheet, startCell, ctx.DSL)
			f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
			row++
		}
	}

	row++
	return row
}

func writeDTXMLInitialActions(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "INITIAL ACTIONS: COMMENTS")
	f.MergeCell(sheet, cellName(3, row), cellName(3+numCols, row))
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3+numCols, row), "DSL: INITIAL ACTIONS")
	row++

	for i, action := range dt.InitialActions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
		styler.ApplyBody(f, sheet, cellName(2, row), "")
		startCell := cellName(3, row)
		endCell := cellName(3+numCols, row)
		f.MergeCell(sheet, startCell, endCell)
		styler.ApplyDSL(f, sheet, startCell, action.DSL)
		f.SetCellStyle(sheet, startCell, endCell, styler.DSLStyle)
		row++
	}

	row++
	return row
}

func writeDTXMLConditions(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "CONDITIONS: COMMENTS")
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3, row), "DSL: CONDITIONS")
	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styler.HeaderStyle)
	}
	row++

	for i, cond := range dt.Conditions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
		styler.ApplyBody(f, sheet, cellName(2, row), cond.Comment)
		styler.ApplyDSL(f, sheet, cellName(3, row), cond.DSL)

		colVals := make(map[int]string, len(cond.Columns))
		for _, cv := range cond.Columns {
			colVals[cv.Number] = cv.Value
		}
		for j := 0; j < numCols; j++ {
			val := colVals[j+1]
			cell := cellName(4+j, row)
			f.SetCellValue(sheet, cell, val)
			f.SetCellStyle(sheet, cell, cell, styler.BodyStyle)
		}
		row++
	}

	row++
	return row
}

func writeDTXMLActions(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	f.MergeCell(sheet, cellName(1, row), cellName(2, row))
	styler.ApplyHeader(f, sheet, cellName(1, row), cellName(1, row), cellName(2, row), "ACTIONS: COMMENTS")
	styler.ApplyHeader(f, sheet, cellName(3, row), cellName(3, row), cellName(3, row), "DSL: ACTIONS")
	for i := 0; i < numCols; i++ {
		cell := cellName(4+i, row)
		f.SetCellValue(sheet, cell, i+1)
		f.SetCellStyle(sheet, cell, cell, styler.SeparatorStyle)
	}
	row++

	for i, action := range dt.Actions {
		f.SetCellValue(sheet, cellName(1, row), i+1)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
		styler.ApplyBody(f, sheet, cellName(2, row), action.Comment)
		styler.ApplyDSL(f, sheet, cellName(3, row), action.DSL)

		colVals := make(map[int]string, len(action.Columns))
		for _, cv := range action.Columns {
			colVals[cv.Number] = cv.Value
		}
		for j := 0; j < numCols; j++ {
			val := colVals[j+1]
			cell := cellName(4+j, row)
			f.SetCellValue(sheet, cell, val)
			f.SetCellStyle(sheet, cell, cell, styler.BodyStyle)
		}
		row++
	}

	row++
	return row
}

func writeDTXMLPolicyStatements(f *excelize.File, sheet string, dt *DecisionTableXML, row, numCols int, styler *Styler) int {
	if len(dt.PolicyStatements) == 0 {
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

	for _, ps := range dt.PolicyStatements {
		colNum, _ := strconv.Atoi(ps.Column)
		f.SetCellValue(sheet, cellName(1, row), colNum)
		f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), styler.NumberStyle)
		styler.ApplyBody(f, sheet, cellName(2, row), ps.Description)
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

// WriteEDDXMLToExcel writes an EDDXML model to an xlsx file.
// It is a pure XML→xlsx conversion; no RuleSet or session is required.
func WriteEDDXMLToExcel(edd *EDDXML, filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "EDD"
	f.SetSheetName("Sheet1", sheet)

	styler, err := NewStyler(f)
	if err != nil {
		return err
	}
	eddStyles, err := newEDDStylesForFile(f)
	if err != nil {
		return err
	}

	setEDDColumnWidthsForFile(f, sheet)
	writeEDDHeadersForFile(f, sheet, styler, 1)
	FreezePaneAtRow2(f, sheet)
	writeEDDXMLEntities(f, sheet, edd.Entities, styler, eddStyles, 2)

	return f.SaveAs(filename)
}

func newEDDStylesForFile(f *excelize.File) (*eddExtraStyles, error) {
	// Delegate to the Exporter method via a throw-away Exporter (no ruleSet needed).
	e := &Exporter{}
	return e.newEDDStyles(f)
}

func setEDDColumnWidthsForFile(f *excelize.File, sheet string) {
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

func writeEDDHeadersForFile(f *excelize.File, sheet string, styler *Styler, startRow int) {
	headers := []string{"Entity", "Attribute", "Type", "SubType", "Default", "Input", "Access", "Description", "Collect", "Question", "Q Type", "Options", "Reference"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, startRow)
		styler.ApplyHeader(f, sheet, cell, cell, cell, header)
	}
}

func writeEDDXMLEntities(f *excelize.File, sheet string, entities []*EDDXMLEntity, styler *Styler, s *eddExtraStyles, startRow int) {
	row := startRow
	for _, ent := range entities {
		attrCount := len(ent.Fields)
		if attrCount == 0 {
			continue
		}

		f.SetCellValue(sheet, cellName(1, row), ent.Name)
		f.SetCellValue(sheet, cellName(2, row), fmt.Sprintf("(%d attributes)", attrCount))
		for col := 1; col <= eddColumnCount; col++ {
			f.SetCellStyle(sheet, cellName(col, row), cellName(col, row), s.entityHeader)
		}
		row++

		for attrRow, field := range ent.Fields {
			rowStyle := s.rowEven
			if attrRow%2 == 1 {
				rowStyle = s.rowOdd
			}

			f.SetCellValue(sheet, cellName(1, row), "")
			f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), rowStyle)

			f.SetCellValue(sheet, cellName(2, row), field.Name)
			f.SetCellStyle(sheet, cellName(2, row), cellName(2, row), s.attribute)

			typeStyle := getEDDTypeStyle(s, field.Type)
			f.SetCellValue(sheet, cellName(3, row), field.Type)
			f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), typeStyle)

			f.SetCellValue(sheet, cellName(4, row), field.SubType)
			f.SetCellStyle(sheet, cellName(4, row), cellName(4, row), rowStyle)

			f.SetCellValue(sheet, cellName(5, row), field.DefaultValue)
			f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), rowStyle)

			inputStyle := rowStyle
			if field.Input != "" {
				inputStyle = s.input
			}
			f.SetCellValue(sheet, cellName(6, row), field.Input)
			f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), inputStyle)

			f.SetCellValue(sheet, cellName(7, row), field.Access)
			f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), rowStyle)

			f.SetCellValue(sheet, cellName(8, row), field.Comment)
			f.SetCellStyle(sheet, cellName(8, row), cellName(8, row), s.comment)

			// Collect + question metadata (#850), columns I–M.
			collectTxt, qText, qType, qOpts, qRef := "", "", "", "", ""
			if strings.EqualFold(field.Collect, "true") {
				collectTxt = "true"
				if field.Question != nil {
					qText = field.Question.Text
					qType = field.Question.Type
					qOpts = encodeEDDOptions(derefOptions(field.Question.Options))
					qRef = encodeEDDRef(field.Question.RefLow, field.Question.RefHigh, field.Question.Units)
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
		}
	}
}

// derefOptions converts []*EDDXMLOption to []EDDXMLOption for encoding.
func derefOptions(opts []*EDDXMLOption) []EDDXMLOption {
	out := make([]EDDXMLOption, 0, len(opts))
	for _, o := range opts {
		if o != nil {
			out = append(out, *o)
		}
	}
	return out
}

func getEDDTypeStyle(s *eddExtraStyles, typeName string) int {
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

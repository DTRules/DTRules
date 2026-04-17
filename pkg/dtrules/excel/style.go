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

import "github.com/xuri/excelize/v2"

const (
	maxColWidth = 60.0

	colorHeader    = "E8E8E8"
	colorSeparator = "D0D0D0"
	fontSansSerif  = "Calibri"
	fontMono       = "Consolas"
)

// Styler holds named style indices for a single excelize.File.
// Create one per file via NewStyler.
type Styler struct {
	// HeaderStyle: bold, #E8E8E8 fill, thin border — for section header rows.
	HeaderStyle int
	// BodyStyle: thin border, Calibri — for regular metadata cells.
	BodyStyle int
	// DSLStyle: thin border, Consolas monospace — for EL/DSL expression cells.
	DSLStyle int
	// SeparatorStyle: slightly darker fill — for visual section separators.
	SeparatorStyle int
	// NumberStyle: centered, thin border — for row/column number cells.
	NumberStyle int
}

// NewStyler registers all named styles with f and returns a populated Styler.
func NewStyler(f *excelize.File) (*Styler, error) {
	s := &Styler{}
	var err error

	thin := thinBorder()

	s.HeaderStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: fontSansSerif, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorHeader}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.BodyStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.DSLStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontMono, Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.SeparatorStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: fontSansSerif, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{colorSeparator}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	s.NumberStyle, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: fontSansSerif, Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thin,
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

// ApplyHeader writes value to cell and applies HeaderStyle across the range start:end.
func (s *Styler) ApplyHeader(f *excelize.File, sheet, cell, start, end string, value interface{}) {
	f.SetCellValue(sheet, cell, value)
	f.SetCellStyle(sheet, start, end, s.HeaderStyle)
}

// ApplyBody writes value to cell and applies BodyStyle.
func (s *Styler) ApplyBody(f *excelize.File, sheet, cell string, value interface{}) {
	f.SetCellValue(sheet, cell, value)
	f.SetCellStyle(sheet, cell, cell, s.BodyStyle)
}

// ApplyDSL writes value to cell and applies DSLStyle.
func (s *Styler) ApplyDSL(f *excelize.File, sheet, cell string, value interface{}) {
	f.SetCellValue(sheet, cell, value)
	f.SetCellStyle(sheet, cell, cell, s.DSLStyle)
}

// FreezePaneAtRow2 sets the standard freeze pane below row 1.
func FreezePaneAtRow2(f *excelize.File, sheet string) {
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
}

func FreezePaneAtRow3(f *excelize.File, sheet string) {
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      2,
		TopLeftCell: "A3",
		ActivePane:  "bottomLeft",
	})
}

// AutoWidth sets a column width clamped to [min, maxColWidth] based on a hint.
// hint is the expected content width in characters.
func AutoWidth(f *excelize.File, sheet, col string, hint float64) {
	w := hint
	if w < 8 {
		w = 8
	}
	if w > maxColWidth {
		w = maxColWidth
	}
	f.SetColWidth(sheet, col, col, w)
}

func thinBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "CCCCCC", Style: 1},
		{Type: "top", Color: "CCCCCC", Style: 1},
		{Type: "bottom", Color: "CCCCCC", Style: 1},
		{Type: "right", Color: "CCCCCC", Style: 1},
	}
}

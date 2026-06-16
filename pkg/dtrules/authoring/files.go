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

package authoring

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// FileInfo describes one decision-table file in a project: its path (relative
// to the xml dir), its declared TABLE_NUMBER range (Lo/Hi == 0 when unranged),
// a human/LLM-facing purpose, and the tables it holds.
type FileInfo struct {
	Path    string   `json:"path"`
	Lo      int      `json:"lo,omitempty"`
	Hi      int      `json:"hi,omitempty"`
	Purpose string   `json:"purpose,omitempty"`
	Tables  []string `json:"tables"`
}

// ranged reports whether a file entry has a declared, enforceable number range.
func (e *dtFileEntry) ranged() bool { return e.lo > 0 && e.hi >= e.lo }

// ---- path helpers ---------------------------------------------------------

func (p *Project) projectRoot() string {
	if filepath.Base(p.xmlDir) == "xml" {
		return filepath.Dir(p.xmlDir)
	}
	return p.xmlDir
}

func (p *Project) notesFile() string {
	return filepath.Join(p.projectRoot(), "authoring-notes.md")
}

// relPathOf returns the slash-separated path of an absolute DT file relative to
// the project's xml dir.
func (p *Project) relPathOf(abs string) string {
	rel, err := filepath.Rel(p.xmlDir, abs)
	if err != nil {
		rel = filepath.Base(abs)
	}
	return filepath.ToSlash(rel)
}

// normFile normalizes a caller-supplied file reference to (relPath, absPath).
// It is slash-normalized and auto-suffixed to end in _dt.xml.
func (p *Project) normFile(file string) (rel, abs string) {
	f := filepath.ToSlash(strings.TrimSpace(file))
	if !strings.HasSuffix(f, "_dt.xml") {
		f = strings.TrimSuffix(f, ".xml") + "_dt.xml"
	}
	return f, filepath.Join(p.xmlDir, filepath.FromSlash(f))
}

func (p *Project) fileIndex(rel string) int {
	for i := range p.dtFiles {
		if p.relPathOf(p.dtFiles[i].path) == rel {
			return i
		}
	}
	return -1
}

// HasFile reports whether a DT file (by relative or bare path) is known to the
// project.
func (p *Project) HasFile(file string) bool {
	rel, _ := p.normFile(file)
	return p.fileIndex(rel) >= 0
}

// FileRel returns the normalized (slash, _dt.xml-suffixed) relative path for a
// caller-supplied file reference, without requiring the file to exist.
func (p *Project) FileRel(file string) string {
	rel, _ := p.normFile(file)
	return rel
}

// RangeOf returns the declared range of a file (exists=false if unknown,
// lo/hi==0 if the file is unranged).
func (p *Project) RangeOf(file string) (lo, hi int, exists bool) {
	rel, _ := p.normFile(file)
	idx := p.fileIndex(rel)
	if idx < 0 {
		return 0, 0, false
	}
	return p.dtFiles[idx].lo, p.dtFiles[idx].hi, true
}

// FileOf returns the relative path of the file holding the named table, or ""
// if the table is not found.
func (p *Project) FileOf(table string) string {
	for i := range p.dtFiles {
		for _, t := range p.dtFiles[i].tables.Tables {
			if t.TableName == table {
				return p.relPathOf(p.dtFiles[i].path)
			}
		}
	}
	return ""
}

// ---- change-log accumulation ---------------------------------------------

func (p *Project) now() time.Time {
	if p.clock != nil {
		return p.clock()
	}
	return time.Now()
}

func (p *Project) logChange(format string, args ...interface{}) {
	line := fmt.Sprintf("- %s — %s", p.now().Format("2006-01-02"), fmt.Sprintf(format, args...))
	p.pendingLog = append(p.pendingLog, line)
}

// AppendNote records a free-form, dated entry in the project's change log for
// design discussion the next session should see.
func (p *Project) AppendNote(text string) {
	p.logChange("note — %q", strings.TrimSpace(text))
}

// ---- range validation -----------------------------------------------------

func (p *Project) overlappingRange(skipRel string, lo, hi int) (string, bool) {
	for i := range p.dtFiles {
		e := &p.dtFiles[i]
		rel := p.relPathOf(e.path)
		if rel == skipRel || !e.ranged() {
			continue
		}
		if lo <= e.hi && e.lo <= hi {
			return rel, true
		}
	}
	return "", false
}

func validRange(lo, hi int) error {
	if lo <= 0 || hi < lo {
		return fmt.Errorf("invalid range [%d-%d]: need 1 <= lo <= hi", lo, hi)
	}
	return nil
}

// globalNextNumber returns max(all table numbers)+10, or 1000 if none. Used for
// unranged (legacy) files.
func (p *Project) globalNextNumber() int {
	max := 0
	for i := range p.dtFiles {
		for _, t := range p.dtFiles[i].tables.Tables {
			if n, err := strconv.Atoi(strings.TrimSpace(t.AttributeFields.TableNumber)); err == nil && n > max {
				max = n
			}
		}
	}
	if max == 0 {
		return 1000
	}
	return max + 10
}

// nextNumberInFile computes the next TABLE_NUMBER for a file: within its range
// for ranged files (lo for empty, max+10 otherwise; error if it exceeds hi),
// or the global max+10 for unranged files.
func (p *Project) nextNumberInFile(idx int) (int, error) {
	e := &p.dtFiles[idx]
	if !e.ranged() {
		return p.globalNextNumber(), nil
	}
	max := 0
	for _, t := range e.tables.Tables {
		if n, err := strconv.Atoi(strings.TrimSpace(t.AttributeFields.TableNumber)); err == nil && n > max {
			max = n
		}
	}
	next := e.lo
	if max > 0 {
		next = max + 10
	}
	if next > e.hi {
		return 0, fmt.Errorf("file %q range [%d-%d] is full; widen it with set-range or use another file",
			p.relPathOf(e.path), e.lo, e.hi)
	}
	return next, nil
}

// validateNumberFor checks that n is a legal explicit TABLE_NUMBER for the named
// table: inside its file's range (if ranged) and unique across the project.
func (p *Project) validateNumberFor(table string, n int) error {
	if n < 1 {
		return fmt.Errorf("table number must be >= 1")
	}
	fileRel := p.FileOf(table)
	if idx := p.fileIndex(fileRel); idx >= 0 {
		e := &p.dtFiles[idx]
		if e.ranged() && (n < e.lo || n > e.hi) {
			return fmt.Errorf("number %d is outside file %q range [%d-%d]", n, fileRel, e.lo, e.hi)
		}
	}
	for i := range p.dtFiles {
		for _, t := range p.dtFiles[i].tables.Tables {
			if t.TableName == table {
				continue
			}
			if cur, err := strconv.Atoi(strings.TrimSpace(t.AttributeFields.TableNumber)); err == nil && cur == n {
				return fmt.Errorf("number %d already used by table %q", n, t.TableName)
			}
		}
	}
	return nil
}

// ---- file / table structural operations -----------------------------------

// CreateFile registers a new, empty decision-table file with a declared,
// non-overlapping number range. reason is required and recorded in the change
// log (and used as the file's purpose). The file is materialized on Save.
func (p *Project) CreateFile(file string, lo, hi int, reason string) error {
	rel, abs := p.normFile(file)
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("creating file %q requires a reason", rel)
	}
	if p.fileIndex(rel) >= 0 {
		return fmt.Errorf("file %q already exists", rel)
	}
	if err := validRange(lo, hi); err != nil {
		return err
	}
	if other, ok := p.overlappingRange(rel, lo, hi); ok {
		return fmt.Errorf("range [%d-%d] overlaps file %q", lo, hi, other)
	}
	p.dtFiles = append(p.dtFiles, dtFileEntry{
		path:    abs,
		tables:  &excel.DecisionTablesXML{},
		lo:      lo,
		hi:      hi,
		purpose: strings.TrimSpace(reason),
	})
	p.logChange("add file `%s` [%d-%d] — %q", rel, lo, hi, strings.TrimSpace(reason))
	return nil
}

// SetFileRange changes a file's number range. It rejects overlaps with other
// ranged files and rejects a range that would exclude a number already in use.
func (p *Project) SetFileRange(file string, lo, hi int, reason string) error {
	rel, _ := p.normFile(file)
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("set-range on %q requires a reason", rel)
	}
	idx := p.fileIndex(rel)
	if idx < 0 {
		return fmt.Errorf("file %q does not exist", rel)
	}
	if err := validRange(lo, hi); err != nil {
		return err
	}
	if other, ok := p.overlappingRange(rel, lo, hi); ok {
		return fmt.Errorf("range [%d-%d] overlaps file %q", lo, hi, other)
	}
	for _, t := range p.dtFiles[idx].tables.Tables {
		if n, err := strconv.Atoi(strings.TrimSpace(t.AttributeFields.TableNumber)); err == nil {
			if n < lo || n > hi {
				return fmt.Errorf("table %q has number %d outside the requested range [%d-%d]", t.TableName, n, lo, hi)
			}
		}
	}
	p.dtFiles[idx].lo = lo
	p.dtFiles[idx].hi = hi
	p.logChange("set-range `%s` [%d-%d] — %q", rel, lo, hi, strings.TrimSpace(reason))
	return nil
}

// MoveTable relocates a table to another existing file, renumbering it into the
// target file's range. reason is required. If the source file is left empty it
// is scheduled for deletion on Save.
func (p *Project) MoveTable(name, target, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("moving table %q requires a reason", name)
	}
	srcIdx, tabIdx := p.locateTable(name)
	if srcIdx < 0 {
		return fmt.Errorf("table %q not found", name)
	}
	rel, _ := p.normFile(target)
	dstIdx := p.fileIndex(rel)
	if dstIdx < 0 {
		return fmt.Errorf("target file %q does not exist; create it first (with a range and reason)", rel)
	}
	srcRel := p.relPathOf(p.dtFiles[srcIdx].path)
	if srcIdx == dstIdx {
		return fmt.Errorf("table %q is already in %q", name, rel)
	}
	num, err := p.nextNumberInFile(dstIdx)
	if err != nil {
		return err
	}
	tbl := p.dtFiles[srcIdx].tables.Tables[tabIdx]
	tbl.AttributeFields.TableNumber = strconv.Itoa(num)
	// remove from source
	src := p.dtFiles[srcIdx].tables.Tables
	p.dtFiles[srcIdx].tables.Tables = append(src[:tabIdx], src[tabIdx+1:]...)
	// append to target
	p.dtFiles[dstIdx].tables.Tables = append(p.dtFiles[dstIdx].tables.Tables, tbl)
	p.logChange("move `%s`: `%s` → `%s` — %q", name, srcRel, rel, strings.TrimSpace(reason))
	p.dropIfEmpty(srcIdx, fmt.Sprintf("last table moved out: %s", strings.TrimSpace(reason)))
	return nil
}

// locateTable returns (fileIndex, tableIndex) for a table name, or (-1,-1).
func (p *Project) locateTable(name string) (int, int) {
	for fi := range p.dtFiles {
		for ti, t := range p.dtFiles[fi].tables.Tables {
			if t.TableName == name {
				return fi, ti
			}
		}
	}
	return -1, -1
}

// dropIfEmpty removes a now-empty file entry, scheduling its on-disk artifacts
// for deletion on Save and recording it in the change log.
func (p *Project) dropIfEmpty(idx int, why string) {
	if idx < 0 || idx >= len(p.dtFiles) {
		return
	}
	if len(p.dtFiles[idx].tables.Tables) > 0 {
		return
	}
	rel := p.relPathOf(p.dtFiles[idx].path)
	p.orphans = append(p.orphans, p.dtFiles[idx].path)
	p.dtFiles = append(p.dtFiles[:idx], p.dtFiles[idx+1:]...)
	p.logChange("delete empty file `%s` (%s)", rel, why)
}

// ---- Files() view ---------------------------------------------------------

// Files returns the project's decision-table files with their ranges, purposes,
// and member tables, ordered by range (ranged files first, by lo) then path.
func (p *Project) Files() []FileInfo {
	out := make([]FileInfo, 0, len(p.dtFiles))
	for i := range p.dtFiles {
		e := &p.dtFiles[i]
		fi := FileInfo{Path: p.relPathOf(e.path), Purpose: e.purpose, Tables: []string{}}
		if e.ranged() {
			fi.Lo, fi.Hi = e.lo, e.hi
		}
		for _, t := range e.tables.Tables {
			fi.Tables = append(fi.Tables, t.TableName)
		}
		out = append(out, fi)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := out[i].Lo > 0, out[j].Lo > 0
		if ri != rj {
			return ri // ranged files first
		}
		if ri && out[i].Lo != out[j].Lo {
			return out[i].Lo < out[j].Lo
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// ---- authoring-notes.md I/O ----------------------------------------------

var notesFileLineRE = regexp.MustCompile("^- `([^`]+)`(?:\\s+\\[(\\d+)-(\\d+)\\])?(?:\\s+—\\s+(.*))?\\s*$")

// loadNotes reads authoring-notes.md (if present) and applies declared ranges
// and purposes to the matching loaded DT files.
func (p *Project) loadNotes() {
	data, err := os.ReadFile(p.notesFile())
	if err != nil {
		return
	}
	for _, raw := range filesSection(string(data)) {
		m := notesFileLineRE.FindStringSubmatch(strings.TrimRight(raw, " \t"))
		if m == nil {
			continue
		}
		rel := filepath.ToSlash(strings.TrimSpace(m[1]))
		idx := p.fileIndex(rel)
		if idx < 0 {
			continue
		}
		if m[2] != "" && m[3] != "" {
			p.dtFiles[idx].lo, _ = strconv.Atoi(m[2])
			p.dtFiles[idx].hi, _ = strconv.Atoi(m[3])
		}
		if m[4] != "" {
			p.dtFiles[idx].purpose = strings.TrimSpace(m[4])
		}
	}
}

// filesSection returns the body lines under the "## Files" header.
func filesSection(content string) []string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "## Files" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	var body []string
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			break
		}
		body = append(body, lines[i])
	}
	return body
}

// writeNotes rewrites the API-managed Files section, appends pending change-log
// entries, and preserves all other content. It is a no-op unless the project
// uses the feature (an existing notes file, a declared range, or pending log
// entries), so plain in-file edits never spawn the doc.
func (p *Project) writeNotes() error {
	_, statErr := os.Stat(p.notesFile())
	exists := statErr == nil
	ranged := false
	for i := range p.dtFiles {
		if p.dtFiles[i].ranged() {
			ranged = true
			break
		}
	}
	if !exists && len(p.pendingLog) == 0 && !ranged {
		return nil
	}

	content := defaultNotesSkeleton
	if exists {
		b, err := os.ReadFile(p.notesFile())
		if err != nil {
			return err
		}
		content = string(b)
	}

	// Regenerate the Files section body.
	var fileLines []string
	for _, fi := range p.Files() {
		line := "- `" + fi.Path + "`"
		if fi.Lo > 0 {
			line += fmt.Sprintf(" [%d-%d]", fi.Lo, fi.Hi)
		}
		if fi.Purpose != "" {
			line += " — " + fi.Purpose
		}
		fileLines = append(fileLines, line)
	}
	content = setMarkdownSection(content, "Files", fileLines)
	content = appendMarkdownSection(content, "Change log", p.pendingLog)

	if err := os.MkdirAll(filepath.Dir(p.notesFile()), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(p.notesFile(), []byte(content), 0644); err != nil {
		return err
	}
	p.pendingLog = nil
	return nil
}

const defaultNotesSkeleton = `# Authoring Notes

## Files

## Conventions

## Change log
`

// setMarkdownSection replaces the body under "## <header>" with body lines,
// preserving every other line. The section is appended if absent.
func setMarkdownSection(content, header string, body []string) string {
	lines := strings.Split(content, "\n")
	start, end := sectionSpan(lines, header)
	if start < 0 {
		return appendSection(content, header, body)
	}
	out := append([]string{}, lines[:start]...)
	out = append(out, body...)
	out = append(out, "")
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n")
}

// appendMarkdownSection appends lines to the end of "## <header>"'s body
// (before the next section), preserving everything else. No-op if lines empty.
func appendMarkdownSection(content, header string, lines []string) string {
	if len(lines) == 0 {
		return content
	}
	all := strings.Split(content, "\n")
	start, end := sectionSpan(all, header)
	if start < 0 {
		return appendSection(content, header, lines)
	}
	// Trim trailing blank lines inside the section body, then append.
	insertAt := end
	for insertAt > start && strings.TrimSpace(all[insertAt-1]) == "" {
		insertAt--
	}
	out := append([]string{}, all[:insertAt]...)
	out = append(out, lines...)
	out = append(out, "")
	out = append(out, all[insertAt:]...)
	return strings.Join(out, "\n")
}

// sectionSpan returns [bodyStart, bodyEnd) line indices for "## <header>"'s
// body (excluding the header line), or (-1,-1) if absent. bodyEnd is the index
// of the next "## " header or len(lines).
func sectionSpan(lines []string, header string) (int, int) {
	target := "## " + header
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == target {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return -1, -1
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return start, end
}

func appendSection(content, header string, body []string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(content, "\n"))
	b.WriteString("\n\n## ")
	b.WriteString(header)
	b.WriteString("\n")
	for _, l := range body {
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}

// removeOrphans deletes the on-disk .xml (and best-effort combined .xlsx) for
// files that were emptied this session.
func (p *Project) removeOrphans() {
	for _, abs := range p.orphans {
		_ = os.Remove(abs)
		// Best-effort: remove the combined Excel workbook (CO_dt.xml -> CO.xlsx).
		rel := p.relPathOf(abs)
		stem := strings.TrimSuffix(rel, "_dt.xml")
		excelDir := filepath.Join(p.projectRoot(), "excel")
		_ = os.Remove(filepath.Join(excelDir, filepath.FromSlash(stem)+".xlsx"))
	}
	p.orphans = nil
}

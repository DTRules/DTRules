package docs_test

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

const elRefFile = "el-reference.md"

const elRefBanner = "> **NOT FOR RULE AUTHORS.** Rule authors MUST use `dtrules docs el`.\n" +
	"> This file is a reference for engineers debugging the compiler or runtime.\n" +
	"> Never copy postfix from this file into a rule artifact — the EL compiler\n" +
	"> generates postfix automatically."

func readELRef(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(elRefFile)
	if err != nil {
		t.Fatalf("cannot read %s: %v", elRefFile, err)
	}
	return string(data)
}

func TestELReference_BannerPresent(t *testing.T) {
	content := readELRef(t)
	if !strings.HasPrefix(content, elRefBanner) {
		t.Errorf("%s does not start with required banner.\ngot prefix:\n%s", elRefFile, content[:min(len(content), 300)])
	}
}

func TestELReference_NoTODO(t *testing.T) {
	content := readELRef(t)
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "TODO") || strings.Contains(upper, "FIXME") {
			t.Errorf("%s line %d: contains TODO/FIXME: %s", elRefFile, lineNum, line)
		}
	}
}

func TestELReference_GrammarProductionsCovered(t *testing.T) {
	content := readELRef(t)

	// Key production names from EL.g4 that must appear in the reference document.
	required := []string{
		"bexpr",
		"iexpr",
		"fexpr",
		"bigexpr",
		"strexpr",
		"dexpr",
		"eexpr",
		"nexpr",
		"arrayExpr",
		"texpr",
		"setstatement",
		"ifstatement",
		"ifblock",
		"ifcontinue",
		"forallctl",
		"forallblock",
		"foreachblock",
		"firstblock",
		"forfirstctl",
		"localvariables",
		"addtostatement",
		"xmlvaluestatements",
		"performstatement",
		"debugstatement",
		"contextForTable",
		"contextstatement",
		"statementList",
		"block",
		"statement",
		"done",
		"optSemi",
		"number",
		"addtodest",
		"includeSearch",
		"blist",
		"thereis",
		"forctl",
		"datestatement",
		"randomstatements",
		"operatorstatements",
		"arrayLit",
		"arrayList",
		"indxExpr",
		"tablelist",
		"leftIexpr",
		"leftFexpr",
		"leftBexpr",
		"leftEexpr",
		"leftStrexpr",
		"leftDexpr",
		"leftArrayRef",
		"leftBigexpr",
		"separator",
		"usingblock",
		"usingstatement",
		"possessiveRef",
		"colonRef",
		"inthe",
		"subtodest",
		"operatorlist",
	}

	for _, prod := range required {
		if !strings.Contains(content, prod) {
			t.Errorf("%s: missing grammar production %q", elRefFile, prod)
		}
	}
}

func TestELReference_RequiredSections(t *testing.T) {
	content := readELRef(t)

	sections := []string{
		"## Overview",
		"## Literals",
		"## Types",
		"## Operators",
		"### Arithmetic",
		"### Comparison",
		"### Logical",
		"### String",
		"### Date",
		"### Array",
		"### Entity",
		"## Built-in Functions",
		"## Control Flow",
		"## Statements",
		"### set",
		"### increment / decrement",
		"### perform",
		"### xml set attribute",
		"## Map / Filter / Sum / There-is",
		"## Local Variables",
		"## Grammar Appendix",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("%s: missing required section %q", elRefFile, section)
		}
	}
}

func TestELReference_HasPostfixExamples(t *testing.T) {
	content := readELRef(t)
	if !strings.Contains(content, "**Compiled postfix**:") {
		t.Errorf("%s: no 'Compiled postfix' examples found", elRefFile)
	}
	// Count postfix examples
	count := strings.Count(content, "**Compiled postfix**:")
	if count < 30 {
		t.Errorf("%s: only %d postfix examples, expected at least 30", elRefFile, count)
	}
}

func TestELReference_HasTaxExamples(t *testing.T) {
	content := readELRef(t)
	if !strings.Contains(content, "**Tax example**:") {
		t.Errorf("%s: no Tax examples found", elRefFile)
	}
	count := strings.Count(content, "**Tax example**:")
	if count < 10 {
		t.Errorf("%s: only %d tax examples, expected at least 10", elRefFile, count)
	}
}

func TestELReference_HasEligibilityExamples(t *testing.T) {
	content := readELRef(t)
	if !strings.Contains(content, "Eligibility example") {
		t.Errorf("%s: no Eligibility examples found", elRefFile)
	}
	// Count all eligibility example markers (including "(real postfix from DTEligibility)" variants)
	count := strings.Count(content, "Eligibility example")
	if count < 10 {
		t.Errorf("%s: only %d eligibility examples, expected at least 10", elRefFile, count)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

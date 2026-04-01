package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
)

type dtXMLFile struct {
	XMLName xml.Name  `xml:"decision_tables"`
	Tables  []dtTable `xml:"decision_table"`
}

type dtTable struct {
	TableName  string       `xml:"table_name"`
	Conditions dtConditions `xml:"conditions"`
	Actions    dtActions    `xml:"actions"`
}

type dtConditions struct {
	Conditions []dtCondition `xml:"condition_details"`
}

type dtCondition struct {
	Description string `xml:"condition_description"`
	Postfix     string `xml:"condition_postfix"`
}

type dtActions struct {
	Actions []dtAction `xml:"action_details"`
}

type dtAction struct {
	Description string `xml:"action_description"`
	Postfix     string `xml:"action_postfix"`
}

func main() {
	compiler := el.NewCompiler()

	// Test TaxReturn
	data, err := os.ReadFile("sampleprojects/TaxReturn/xml/TaxReturn_dt.xml")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var dt dtXMLFile
	if err := xml.Unmarshal(data, &dt); err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		return
	}

	fmt.Println("=== CHIP EL Compilation Test ===")
	for _, table := range dt.Tables {
		fmt.Printf("\nTable: %s\n", table.TableName)
		for i, cond := range table.Conditions.Conditions {
			desc := strings.TrimSpace(cond.Description)
			if desc == "" {
				continue
			}
			_, err := compiler.CompileCondition(desc)
			if err != nil {
				fmt.Printf("  COND %d FAIL: %s\n    Error: %v\n", i+1, desc, err)
			} else {
				fmt.Printf("  COND %d OK: %s\n", i+1, desc)
			}
		}
		for i, action := range table.Actions.Actions {
			desc := strings.TrimSpace(action.Description)
			if desc == "" {
				continue
			}
			_, err := compiler.CompileAction(desc)
			if err != nil {
				fmt.Printf("  ACTION %d FAIL: %s\n    Error: %v\n", i+1, desc, err)
			} else {
				fmt.Printf("  ACTION %d OK: %s\n", i+1, desc)
			}
		}
	}
}

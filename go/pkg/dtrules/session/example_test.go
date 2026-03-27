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

package session_test

import (
	"fmt"

	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

// Example_loadFromDirectory demonstrates loading all XML files from a directory.
// Note: This example requires the TaxReturn sample project to be present.
func Example_loadFromDirectory() {
	// Create a new rule set
	rs := session.NewRuleSet("TaxReturn")

	// In a real application, you would use an absolute path or a path
	// relative to your project root:
	// err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")

	// For this example, we'll demonstrate the API without requiring the files
	fmt.Printf("API Usage:\n")
	fmt.Printf("  rs := session.NewRuleSet(\"TaxReturn\")\n")
	fmt.Printf("  err := rs.LoadFromDirectory(\"./xml\")\n")
	fmt.Printf("  session, _ := rs.NewSession()\n")

	// Just to satisfy the compiler
	_ = rs

	// Output:
	// API Usage:
	//   rs := session.NewRuleSet("TaxReturn")
	//   err := rs.LoadFromDirectory("./xml")
	//   session, _ := rs.NewSession()
}

// Example_loadRulesFromDirectory demonstrates the convenience function.
func Example_loadRulesFromDirectory() {
	// Demonstrate the API
	fmt.Printf("Convenience Function API:\n")
	fmt.Printf("  rs, err := session.LoadRulesFromDirectory(\"TaxReturn\", \"./xml\")\n")
	fmt.Printf("  if err != nil {\n")
	fmt.Printf("    log.Fatal(err)\n")
	fmt.Printf("  }\n")
	fmt.Printf("  session, _ := rs.NewSession()\n")

	// Output:
	// Convenience Function API:
	//   rs, err := session.LoadRulesFromDirectory("TaxReturn", "./xml")
	//   if err != nil {
	//     log.Fatal(err)
	//   }
	//   session, _ := rs.NewSession()
}

// Example_loadRulesFromFiles demonstrates loading from individual files.
func Example_loadRulesFromFiles() {
	// Demonstrate loading from specific files
	fmt.Printf("File-Based Loading API:\n")
	fmt.Printf("  rs, err := session.LoadRulesFromFiles(\"TaxReturn\",\n")
	fmt.Printf("    \"./xml/TaxReturn_edd.xml\",\n")
	fmt.Printf("    \"./xml/TaxReturn_dt.xml\")\n")
	fmt.Printf("  session, _ := rs.NewSession()\n")

	// Output:
	// File-Based Loading API:
	//   rs, err := session.LoadRulesFromFiles("TaxReturn",
	//     "./xml/TaxReturn_edd.xml",
	//     "./xml/TaxReturn_dt.xml")
	//   session, _ := rs.NewSession()
}

// Example_loadEDDFile demonstrates loading individual files.
func Example_loadEDDFile() {
	// Demonstrate incremental loading
	fmt.Println("Incremental Loading API:")
	fmt.Println("  rs := session.NewRuleSet(\"MyRuleSet\")")
	fmt.Println("")
	fmt.Println("  // Load EDD")
	fmt.Println("  err := rs.LoadEDDFile(\"./xml/TaxReturn_edd.xml\")")
	fmt.Println("")
	fmt.Println("  // Load decision tables")
	fmt.Println("  err = rs.LoadDecisionTablesFile(\"./xml/TaxReturn_dt.xml\")")
	fmt.Println("")
	fmt.Println("  // Check what was loaded")
	fmt.Println("  entities := rs.GetEntityNames()")
	fmt.Println("  tables := rs.GetDecisionTableNames()")

	// Output:
	// Incremental Loading API:
	//   rs := session.NewRuleSet("MyRuleSet")
	//
	//   // Load EDD
	//   err := rs.LoadEDDFile("./xml/TaxReturn_edd.xml")
	//
	//   // Load decision tables
	//   err = rs.LoadDecisionTablesFile("./xml/TaxReturn_dt.xml")
	//
	//   // Check what was loaded
	//   entities := rs.GetEntityNames()
	//   tables := rs.GetDecisionTableNames()
}

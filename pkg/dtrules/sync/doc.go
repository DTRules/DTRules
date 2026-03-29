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

/*
Package sync provides bidirectional Excel <-> XML synchronization for DTRules.

This package enables a workflow where business analysts can edit rules in Excel
while developers can edit the same rules in XML. The sync package detects which
files have been modified based on timestamps and synchronizes them appropriately.

# File Pairing Convention

XML and Excel files are paired based on their relative paths and naming:

	xml/TaxReturn_dt.xml    <->  excel/TaxReturn_dt.xlsx
	xml/TaxReturn_edd.xml   <->  excel/TaxReturn_edd.xlsx
	xml/states/CO_dt.xml    <->  excel/states/CO_dt.xlsx

# Sync Directions

The package detects four possible sync states:

  - NoSync: Files are in sync (timestamps within grace period)
  - ExcelToXML: Excel is newer, should be imported to XML
  - XMLToExcel: XML is newer, should be exported to Excel
  - Conflict: Both files modified since last sync

# Basic Usage

Create a syncer and check the sync state:

	syncer := sync.NewSyncer(xmlDir, excelDir)
	direction, pairs, err := syncer.CheckSync()

Perform synchronization:

	// Sync all files based on timestamps
	result, err := syncer.SyncAll()

	// Or sync in a specific direction
	err := syncer.SyncExcelToXML()
	err := syncer.SyncXMLToExcel()

# Compilation Integration

Use CompileWithSync for integrated workflow:

	result, err := sync.CompileWithSync("TaxReturn", xmlDir, excelDir)
	if err != nil {
	    log.Fatal(err)
	}
	ruleSet := result.RuleSet

Or use the SyncCompiler for more control:

	opts := &sync.CompilerOptions{
	    XMLDir:   xmlDir,
	    ExcelDir: excelDir,
	    SyncOptions: &sync.SyncOptions{
	        ConflictResolution: "prefer-excel",
	        DryRun:             false,
	    },
	}
	compiler := sync.NewSyncCompiler(opts)

	// Set importer/exporter when available
	compiler.SetImporter(excelImporter)
	compiler.SetExporter(excelExporter)

	result, err := compiler.Compile("TaxReturn")

# Conflict Resolution

When both files have been modified, the behavior is controlled by
the ConflictResolution option:

  - "error" (default): Return an error and stop
  - "prefer-excel": Use the Excel version
  - "prefer-xml": Use the XML version
  - "skip": Skip conflicting files

# Grace Period

To handle filesystem timestamp precision issues, timestamps within
a configurable grace period (default 2 seconds) are considered equal.

# Skipped Files

The following files are automatically skipped during sync:

  - Files in testfiles/ directories
  - Files in schemas/ directories
  - Template files (containing "TEMPLATE" in name)
  - Mapping files (*_map.xml)

# Dependency Note

This package defines interfaces for ExcelImporter and ExcelExporter.
The actual implementations are in separate packages:

  - ExcelExporter: Already implemented in github.com/DTRules/DTRules/pkg/dtrules/excel
  - ExcelImporter: To be implemented (stub calls return descriptive errors)

Set the importer/exporter using SetImporter() and SetExporter() methods
once the implementations are available.
*/
package sync

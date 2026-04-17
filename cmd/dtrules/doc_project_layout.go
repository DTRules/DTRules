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

package main

const docProjectLayout = `Project Layout
==============

A DTRules project is a directory that contains all the files needed to define
and execute a rule set. This topic describes the standard folder layout, the
file naming conventions introduced in v1.5.0, and the purpose of every
special file.


Standard Folder Layout
----------------------
Every project follows this structure:

    MyProject/
    ├── DTRules.xml                  # Project root descriptor
    ├── excel/                       # Source of truth — edit here
    │   ├── .sync-manifest.json      # Sync state (do NOT commit)
    │   ├── federal/
    │   │   ├── Federal_dt.xlsx      # Federal decision tables workbook
    │   │   └── Federal_edd.xlsx     # Federal entity definitions workbook
    │   └── states/
    │       ├── CO_dt.xlsx           # Colorado decision tables
    │       ├── CO_edd.xlsx          # Colorado entity definitions
    │       ├── CA_dt.xlsx
    │       └── CA_edd.xlsx
    ├── xml/                         # Compiled from Excel — edit only *_dsl tags
    │   ├── federal/
    │   │   ├── Federal_dt.xml
    │   │   └── Federal_edd.xml
    │   └── states/
    │       ├── CO_dt.xml
    │       ├── CO_edd.xml
    │       ├── CA_dt.xml
    │       └── CA_edd.xml
    └── testfiles/                   # Test scenarios
        └── TestScenarios/
            ├── Scenario1/
            │   ├── input.xml
            │   └── expected.xml
            └── Scenario2/
                ├── input.xml
                └── expected.xml


DTRules.xml — Project Root Descriptor
--------------------------------------
DTRules.xml lives at the project root and tells the runtime which XML files to
load, and in what order. A typical file looks like this:

    <DTRules>
      <compileralias name="EL">com.dtrules.compiler.el.EL</compileralias>
      <compiler>EL</compiler>

      <RuleSet name="MyProject" source="file">
        <RuleSetFilePath>/xml</RuleSetFilePath>
        <WorkingDirectory>/temp</WorkingDirectory>

        <Entities       name="MyProject_edd.xml" />
        <Decisiontables name="MyProject_dt.xml"  />
        <Map            name="MyProject_map.xml" />
      </RuleSet>
    </DTRules>

Key elements:
  - RuleSet name      Identifier used when loading the rule set in code
  - RuleSetFilePath   Path prefix for XML file loading (relative to project root)
  - Entities          EDD file (entity definitions)
  - Decisiontables    DT file (business rules)
  - Map               Map file (value mappings, optional)

Multiple <Entities> and <Decisiontables> elements are allowed — the engine
loads them in the order listed.


Custom Directory Overrides in DTRules.xml
------------------------------------------
Projects that store rules in non-standard directories can declare overrides
directly in DTRules.xml. This removes the need to pass flags on every command:

    <dtrules>
      <xml_dir>pkg/dtrules/rules</xml_dir>
      <excel_dir>pkg/dtrules/excel</excel_dir>
      ...
    </dtrules>

Both elements are optional. Paths may be relative (resolved against the project
root) or absolute.

When these elements are present, verify, validate, and build all use them
automatically. CLI flags still take the highest priority if supplied:

    dtrules verify --xml-dir override/path          # flag wins
    dtrules verify                                   # DTRules.xml wins
    dtrules verify /project/without/dtrules-xml     # defaults (xml/, excel/)

Directory resolution order (highest to lowest priority):
  1. --xml-dir / --excel-dir CLI flags
  2. <xml_dir> / <excel_dir> in DTRules.xml
  3. Default: xml/ and excel/ relative to the project root

Better error messages:
  When a custom directory cannot be found, the error message names the
  flag you can use to correct it:

      ERROR: could not find xml directory
        Tried: /home/user/project/nonexistent
        Use --xml-dir <path> or declare <xml_dir> in DTRules.xml.


File Naming Convention (v1.5.0+)
----------------------------------
Since v1.5.0, each file's suffix signals what artifact type it contains.
The sync system enforces these names and uses them to determine which
Excel sheet(s) are relevant.

    Suffix      Artifact        Example Excel         Example XML
    -------     ------------    ------------------    ------------------
    _dt         Decision tables Federal_dt.xlsx       Federal_dt.xml
    _edd        Entity defs     Federal_edd.xlsx      Federal_edd.xml
    _map        Value mappings  Federal_map.xlsx      Federal_map.xml

The base name before the suffix is free-form but should be descriptive.
State files conventionally use the two-letter state code as the base name:
CO_dt.xlsx, CO_edd.xlsx.


Single-Artifact Workbooks vs. Mixed-Artifact Workbooks
--------------------------------------------------------
Single-artifact workbook (suffix present):
  - The file name carries the suffix (_dt, _edd, or _map).
  - Every sheet in the workbook is treated as the same artifact type.
  - Example: CO_dt.xlsx — all sheets are decision table sheets.

Mixed-artifact workbook (no suffix, routed by A1 marker):
  - The file name has no artifact suffix (e.g., CO.xlsx or TaxReturn.xlsx).
  - Each sheet declares its type by placing a marker in cell A1:
      DT:    — sheet contains a decision table
      EDD:   — sheet contains entity definitions
      MAP:   — sheet contains value mappings
  - The sync system routes each sheet to the correct XML file based on the
    A1 marker.

Use single-artifact workbooks when different artifact types are maintained by
different people or teams. Use mixed-artifact workbooks when it is more
convenient to keep everything for one domain in a single file.


Subdirectory Support
---------------------
The excel/ and xml/ directories may contain arbitrary subdirectories. The sync
system mirrors the directory tree: a file at excel/states/CO_dt.xlsx maps to
xml/states/CO_dt.xml. There is no depth limit.

Subfolders are useful for:
  - Separating federal rules from state rules (federal/, states/)
  - Grouping rules by functional domain (income/, deductions/, credits/)
  - Isolating rules authored by different teams

The only requirement is that the structure under excel/ mirrors the structure
under xml/ for every managed file.


excel/ — Source of Truth
-------------------------
All human editing happens in excel/. When a business analyst changes a rule,
they open the .xlsx file in this directory and save it.

IMPORTANT: The sync system will refuse to export XML -> Excel if it detects
that an .xlsx file was modified after the last export. This prevents
accidentally overwriting a user's work. If an export is rejected, run
'dtrules sync import' first, then re-apply your XML changes, then export.

See 'dtrules docs workflow' for the full import/export cycle.


xml/ — Compiled Output
------------------------
The xml/ directory holds XML files that were generated by importing from Excel.
Two rules govern editing XML directly:

1. Only edit tags whose names end in _dsl:
       condition_dsl, action_dsl, context_dsl, initial_action_dsl

   These tags hold EL (Expression Language) source. The compiler reads them
   and regenerates the corresponding compiled tags automatically during the
   next build.

2. Never hand-edit compiled tags (condition, action, context, initial_action
   without the _dsl suffix). Those are overwritten on every build.

See 'dtrules docs el' for EL syntax and 'dtrules docs xml-format' for the
full XML structure.


.sync-manifest.json — Sync State File
---------------------------------------
The .sync-manifest.json file lives inside excel/ and records the last time
each .xlsx file was exported from XML. The sync system uses this to detect
whether a user has edited Excel since the last export.

Structure (for reference only — do not edit by hand):

    {
      "version": 1,
      "lastUpdated": "2024-06-01T12:00:00Z",
      "files": {
        "states/CO_dt.xlsx": {
          "lastExportTime": "2024-06-01T11:55:00Z",
          "excelModTimeAtExport": "2024-06-01T11:55:01Z",
          "xmlFiles": ["states/CO_dt.xml"]
        }
      }
    }

Fields:
  - version              Manifest format version (currently 1)
  - lastUpdated          When the manifest was last written
  - files                Map of Excel paths (relative to manifest) to their sync state
  - lastExportTime       When 'dtrules sync export' last wrote this .xlsx file
  - excelModTimeAtExport Excel file's modtime at export — used to detect later edits
  - xmlFiles             Which XML files were exported to produce this .xlsx

DO NOT COMMIT .sync-manifest.json to version control. Add it to .gitignore.
It is a local machine artifact; it is meaningless on another developer's
machine where the file timestamps are different.


testfiles/ — Test Scenarios
-----------------------------
The testfiles/ directory holds test inputs and expected outputs that verify
the rule set behaves correctly. Each scenario is a subdirectory under
testfiles/TestScenarios/:

    testfiles/
    └── TestScenarios/
        └── SingleFiler_Standard/
            ├── input.xml       # Entity values fed to the rule engine
            └── expected.xml    # Expected entity values after execution

A minimal input.xml:

    <?xml version="1.0" encoding="UTF-8"?>
    <RuleSet>
      <entity name="taxpayer">
        <field name="filing_status">SINGLE</field>
        <field name="income">55000.0</field>
      </entity>
    </RuleSet>

A minimal expected.xml:

    <?xml version="1.0" encoding="UTF-8"?>
    <RuleSet>
      <entity name="result">
        <field name="federal_tax">9900.0</field>
        <field name="state_tax">2420.0</field>
      </entity>
    </RuleSet>

Create at least three scenarios per rule set:
  - A typical/happy-path case
  - A boundary case (zero income, minimum values, etc.)
  - An edge case or alternate path (different filing status, etc.)


Putting It All Together — Complete Example
-------------------------------------------
Here is the full layout for a two-subdirectory project (federal + states):

    TaxReturn/
    ├── DTRules.xml
    ├── excel/
    │   ├── .sync-manifest.json          # gitignored
    │   ├── federal/
    │   │   ├── Federal_edd.xlsx         # taxpayer, result entity definitions
    │   │   └── Federal_dt.xlsx          # federal tax bracket tables
    │   └── states/
    │       ├── CO_edd.xlsx              # Colorado-specific entities/constants
    │       ├── CO_dt.xlsx               # Colorado tax calculation tables
    │       ├── CA_edd.xlsx
    │       └── CA_dt.xlsx
    ├── xml/
    │   ├── federal/
    │   │   ├── Federal_edd.xml
    │   │   └── Federal_dt.xml
    │   └── states/
    │       ├── CO_edd.xml
    │       ├── CO_dt.xml
    │       ├── CA_edd.xml
    │       └── CA_dt.xml
    └── testfiles/
        └── TestScenarios/
            ├── CO_SingleFiler/
            │   ├── input.xml
            │   └── expected.xml
            ├── CA_JointFiler/
            │   ├── input.xml
            │   └── expected.xml
            └── Federal_Only/
                ├── input.xml
                └── expected.xml


See Also
---------
  dtrules docs workflow        Import/export cycle and build commands
  dtrules docs xml-format      Full XML file format specification
  dtrules docs edd             Entity Data Dictionary reference
  dtrules docs decision-tables Decision table authoring guide
  dtrules docs el              Expression Language syntax reference
`

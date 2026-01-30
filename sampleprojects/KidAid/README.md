# KidAid - Child Assistance Program

A sample DTRules project demonstrating child assistance eligibility determination.

## Overview

This project implements eligibility rules for a fictional child assistance program. Similar to CHIP, it evaluates applicants for various assistance programs based on income, age, and other factors, with specific rules for covered counties.

## What This Demonstrates

- Eligibility determination logic
- County-based program availability
- Income group processing
- Citizenship validation
- Multi-program determination (KidAid, Medicaid, Food Stamps)

## Project Structure

```
KidAid/
├── DTRules.xml                    # Project configuration
├── DecisionTables/
│   └── KidAid_dt.xls              # Decision tables with eligibility rules
├── edd/
│   └── KidAid_edd.xls             # Entity Definition Document
├── xml/                           # Compiled XML (generated)
├── repository/                    # Packaged rules for deployment
├── testfiles/
│   └── TestScenarios/             # Test input files
│       ├── TestCase_001.xml
│       └── TestCase_002.xml
└── src/main/java/
    └── com/dtrules/samples/sampleproject2/
        ├── CompileKidAid.java     # Compiles Excel → XML
        └── TestKidAid.java        # Runs test cases
```

## Entities

| Entity | Description |
|--------|-------------|
| `case` | Container with case ID, county code, and clients |
| `client` | Applicant with age, eligibility, income groups |
| `income` | Income record with amount and type |
| `job` | Processing job with results |
| `relationship` | Client relationships (parent/child, spouse) |
| `result` | Eligibility determination output |
| `constants` | Program types and covered counties list |

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

```bash
cd sampleprojects/KidAid
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.CompileKidAid"
```

### Run Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.TestKidAid"
```

## Configuration

`DTRules.xml`:
```xml
<DTRules>
  <compiler>EL</compiler>
  <RuleSet name="KidAid" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="kidaid_edd.xml" />
    <Decisiontables  name="kidaid_dt.xml"  />
    <Map             name="kidaid_map.xml" />
  </RuleSet>
</DTRules>
```

## Related Projects

- **KidAid_Application** - Standalone application wrapper
- **CHIP** - Similar eligibility determination project

## DSL

Uses **EL (Expression Language)** for conditions and actions.

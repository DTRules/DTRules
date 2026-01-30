# CHIP - Children's Health Insurance Program

A sample DTRules project demonstrating health insurance eligibility determination.

## Overview

This project implements eligibility rules for a fictional health insurance program similar to CHIP (Children's Health Insurance Program). It determines whether applicants qualify for various assistance programs based on income, age, citizenship, and other factors.

## What This Demonstrates

- Complex eligibility decision logic
- Multiple entity relationships (case → clients → income)
- Federal Poverty Level (FPL) percentage calculations
- Program-level determination (CHIP, Medicaid, Food Stamps)
- Test case validation with expected outcomes

## Project Structure

```
CHIP/
├── DTRules.xml                    # Project configuration
├── DecisionTables/
│   └── ChipEligibility_dt.xls     # Decision tables with eligibility rules
├── edd/
│   └── ChipEligibility_edd.xls    # Entity Definition Document
├── xml/                           # Compiled XML (generated)
├── repository/                    # Packaged rules for deployment
├── testfiles/
│   └── TestScenarios/             # Test input files
│       ├── TestCase_001.xml
│       └── TestCase_002.xml
└── src/main/java/
    └── com/dtrules/samples/chipeligibility/
        ├── CompileChip.java       # Compiles Excel → XML
        ├── TestChip.java          # Runs test cases
        └── TestCaseGen.java       # Generates test cases
```

## Entities

| Entity | Description |
|--------|-------------|
| `case` | Container for case ID, county, and clients |
| `client` | Applicant with age, income, eligibility status |
| `income` | Income record with amount and type |
| `job` | Processing job with results |
| `relationship` | Links between clients (parent/child, spouse) |
| `result` | Eligibility determination output |
| `constants` | Program types and configuration values |

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

Convert Excel files to XML:
```bash
cd sampleprojects/CHIP
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.CompileChip"
```

This generates:
- `xml/CHIP_edd.xml` - Compiled entity definitions
- `xml/CHIP_dt.xml` - Compiled decision tables
- `xml/CHIP_map.xml` - Data mapping configuration

### Run Tests

Execute test cases:
```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
```

Output is written to `testfiles/output/`.

### Generate Test Cases

Create additional test scenarios:
```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestCaseGen"
```

## Configuration

`DTRules.xml`:
```xml
<DTRules>
  <compiler>EL</compiler>
  <RuleSet name="CHIP" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="CHIP_edd.xml" />
    <Decisiontables  name="CHIP_dt.xml"  />
    <Map             name="CHIP_map.xml" />
  </RuleSet>
</DTRules>
```

## Key Decision Tables

The main decision table `Compute_Eligibility` evaluates:
- Age requirements (under 19 for CHIP)
- Income thresholds relative to Federal Poverty Level
- Citizenship/immigration status
- Current insurance coverage
- Household composition

## Related Projects

- **ChipApp** - Standalone application wrapper for running CHIP rules
- **KidAid** - Similar eligibility determination project

## DSL

Uses **EL (Expression Language)** for conditions and actions.

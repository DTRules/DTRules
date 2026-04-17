# StateTax - State Income Tax Computation

A DTRules sample project implementing state income tax computation for all 50 US states plus the District of Columbia, using 2025 tax year data.

## Overview

This project demonstrates how DTRules can implement a complete, data-driven tax computation engine. Rather than hardcoding tax rules for each state, the engine uses a generic set of decision tables that operate on state-specific configuration data (brackets, deductions, exemptions). Any state's tax can be computed by providing the appropriate `state_config` entity in the input XML — no code changes required.

The project covers three tax paradigms used across US states:
- **No income tax** (9 states: AK, FL, NV, NH, SD, TN, TX, WA, WY)
- **Flat tax** (13 states: AZ, CO, GA, IA, ID, IL, IN, KY, LA, MI, NC, PA, UT)
- **Progressive brackets** (29 states + DC, ranging from 2 to 12 brackets)

## What This Demonstrates

- Data-driven decision table design (generic logic + configurable data)
- Array iteration with guard-flag pattern for early termination
- Three-way routing based on tax type (none / flat / progressive)
- Entity-driven tax bracket computation with pre-computed base tax values
- Filing status dispatch (Single, MFJ, Head of Household)
- Standard deduction and personal exemption calculations
- Self-contained test cases with embedded state configuration data
- Both Java and Go implementations

## Project Structure

```
StateTax/
├── DTRules.xml                    # Project configuration
├── pom.xml                        # Maven build configuration
├── DecisionTables/                # Excel source (for future use)
├── edd/                           # Excel source (for future use)
├── xml/                           # Rule definition XML files
│   ├── StateTax_edd.xml           # Entity definitions (7 entities)
│   ├── StateTax_dt.xml            # Decision tables (9 tables)
│   └── StateTax_map.xml           # XML-to-entity mapping
├── repository/                    # Packaged rules for deployment
│   └── xml/                       # Same as xml/ (for Go integration)
├── testfiles/
│   └── TestScenarios/             # Test cases (51 files, one per state + DC)
│       ├── TestCase_AK_none.xml
│       ├── TestCase_AL_progressive.xml
│       ├── TestCase_CA_progressive.xml
│       ├── TestCase_IL_flat.xml
│       ├── TestCase_NY_progressive.xml
│       ├── TestCase_TX_none.xml
│       └── ... (51 total)
├── temp/                          # Working directory
└── src/main/java/
    └── com/dtrules/samples/statetax/
        ├── CompileStateTax.java   # Compiles Excel → XML
        └── TestStateTax.java      # Runs test cases with formatted output
```

## Entities

| Entity | Description |
|--------|-------------|
| `state_config` | State-specific tax parameters: tax type, flat rate, standard deductions (single/MFJ/HOH), exemption amount, and bracket arrays for each filing status |
| `bracket` | A single tax bracket: floor, ceiling, marginal rate in basis points, and pre-computed cumulative base tax |
| `job` | Tax processing job: state code, tax year, results array |
| `taxpayer` | Individual filer: filing status, dependents, income sources, adjustments, and all computed intermediate values (gross income, AGI, taxable income, tax owed) |
| `income` | Income source: type (WA/INT/DIV/CG/OTH) and annual amount |
| `adjustment` | Above-the-line adjustment: type (SLI/IRA/HSA) and amount |
| `result` | Tax computation output: all computed values, state code, and processing notes |

### Key Entity Fields

**state_config:**
| Field | Type | Description |
|-------|------|-------------|
| `state_code` | string | Two-letter state abbreviation |
| `state_name` | string | Full state name |
| `tax_type` | string | `none`, `flat`, or `progressive` |
| `flat_rate_bps` | integer | Flat tax rate in basis points (e.g., 495 = 4.95%) |
| `standard_deduction_single` | integer | Standard deduction for Single filers |
| `standard_deduction_mfj` | integer | Standard deduction for Married Filing Jointly |
| `standard_deduction_hoh` | integer | Standard deduction for Head of Household |
| `exemption_amount` | integer | Per-exemption deduction amount |
| `brackets_single` | array(bracket) | Tax brackets for Single filers |
| `brackets_mfj` | array(bracket) | Tax brackets for MFJ filers |
| `brackets_hoh` | array(bracket) | Tax brackets for HOH filers |

**bracket:**
| Field | Type | Description |
|-------|------|-------------|
| `floor` | integer | Bottom of bracket range (inclusive) |
| `ceiling` | integer | Top of bracket range (999999999 for highest bracket) |
| `rate_bps` | integer | Marginal rate in basis points (e.g., 600 = 6%) |
| `base_tax` | integer | Pre-computed cumulative tax owed at the floor of this bracket |

## Decision Tables

The engine uses 9 decision tables with a clean separation between routing, computation, and result assembly:

| # | Table | Type | Purpose |
|---|-------|------|---------|
| 1 | `Compute_Tax` | FIRST | Main entry point — routes by `state_config.tax_type` |
| 2 | `Calculate_Gross_Income` | FIRST | Iterates income sources, sums into `taxpayer.grossIncome` |
| 3 | `Apply_Adjustments` | FIRST | Iterates adjustments, subtracts from AGI |
| 4 | `Determine_Filing_Details` | FIRST | Sets standard deduction and exemption count from `state_config` based on filing status |
| 5 | `Calculate_Taxable_Income` | FIRST | Computes `AGI - deduction - (exemptions × exemption_amount)`, floors at $0 |
| 6 | `Apply_Flat_Tax` | FIRST | `taxOwed = taxableIncome × flat_rate_bps / 10000` |
| 7 | `Select_And_Apply_Brackets` | FIRST | Selects bracket array by filing status, calls `Apply_Bracket_Iteration` |
| 8 | `Apply_Bracket_Iteration` | FIRST | Iterates ordered brackets with guard flag; applies `base_tax + (income - floor) × rate_bps / 10000` |
| 9 | `Evaluate_Results` | FIRST | Creates result entity with all computed values |

### Compute_Tax Routing Logic

```
┌──────────────────────────────────────────────────────────────────┐
│                        Compute_Tax                               │
│                                                                  │
│  tax_type == "none"  ──→  Skip to Evaluate_Results (tax = $0)   │
│                                                                  │
│  tax_type == "flat"  ──→  Calculate_Gross_Income                │
│                           Set AGI = grossIncome                  │
│                           Apply_Adjustments                      │
│                           Determine_Filing_Details               │
│                           Calculate_Taxable_Income               │
│                           Apply_Flat_Tax                         │
│                           Evaluate_Results                       │
│                                                                  │
│  tax_type == "progressive" ──→ (same pipeline but with)         │
│                           Select_And_Apply_Brackets              │
│                           Evaluate_Results                       │
└──────────────────────────────────────────────────────────────────┘
```

### Bracket Iteration Pattern

Progressive tax brackets are applied using an iteration with a guard flag:

1. `Select_And_Apply_Brackets` copies the correct bracket array (single/MFJ/HOH) to `taxpayer.active_brackets` and resets the `bracket_applied` flag
2. `Apply_Bracket_Iteration` iterates through brackets in order (lowest to highest)
3. For each bracket, two conditions are checked:
   - `bracket_applied == false` (guard — ensures only one bracket fires)
   - `taxableIncome <= bracket.ceiling` (income falls within this bracket)
4. When both conditions match, the tax is computed: `base_tax + (taxableIncome - floor) × rate_bps / 10000`
5. The guard flag is set to `true`, preventing subsequent brackets from firing

### Base Tax Pre-computation

Each bracket stores a pre-computed `base_tax` value — the cumulative tax owed at the bracket's floor. This avoids summing across all lower brackets at runtime:

```
Bracket 1: floor=0,     ceiling=10000, rate=200bps, base_tax=0
Bracket 2: floor=10000, ceiling=40000, rate=400bps, base_tax=200
            (base_tax = 0 + 10000×200/10000 = 200)
Bracket 3: floor=40000, ceiling=80000, rate=600bps, base_tax=1400
            (base_tax = 200 + 30000×400/10000 = 1400)
```

Formula: `base_tax[i] = base_tax[i-1] + (ceiling[i-1] - floor[i-1]) × rate_bps[i-1] / 10000`

All arithmetic uses integer division (truncation).

## Test Case Format

Each test case is a self-contained XML file that includes the state's tax configuration and a taxpayer scenario. This design means adding a new state requires only a new XML file — no code changes.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!-- Virginia - Progressive brackets (4 brackets)
     Single, $65,000 wages, 0 dependents
     Taxable = 65000 - 8750 - (1 * 930) = 55320
     Bracket [17000, 999999999, 575, 720]:
       tax = 720 + (55320 - 17000) * 575 / 10000 = 720 + 2203 = 2923
     Expected: taxOwed = $2,923
-->
<job>
  <tax_year>2025</tax_year>
  <state>VA</state>

  <state_config>
    <state_code>VA</state_code>
    <state_name>Virginia</state_name>
    <tax_type>progressive</tax_type>
    <flat_rate_bps>0</flat_rate_bps>
    <standard_deduction_single>8750</standard_deduction_single>
    <standard_deduction_mfj>17500</standard_deduction_mfj>
    <standard_deduction_hoh>8750</standard_deduction_hoh>
    <exemption_amount>930</exemption_amount>

    <brackets_single>
      <bracket_s><floor>0</floor><ceiling>3000</ceiling>
        <rate_bps>200</rate_bps><base_tax>0</base_tax></bracket_s>
      <bracket_s><floor>3000</floor><ceiling>5000</ceiling>
        <rate_bps>300</rate_bps><base_tax>60</base_tax></bracket_s>
      <bracket_s><floor>5000</floor><ceiling>17000</ceiling>
        <rate_bps>500</rate_bps><base_tax>120</base_tax></bracket_s>
      <bracket_s><floor>17000</floor><ceiling>999999999</ceiling>
        <rate_bps>575</rate_bps><base_tax>720</base_tax></bracket_s>
    </brackets_single>
    <brackets_mfj></brackets_mfj>
    <brackets_hoh></brackets_hoh>
  </state_config>

  <taxpayer>
    <taxpayer_ID>VA01</taxpayer_ID>
    <filing_status>SINGLE</filing_status>
    <num_dependents>0</num_dependents>
    <incomes>
      <income><type>WA</type><amount>65000</amount></income>
    </incomes>
    <adjustments></adjustments>
  </taxpayer>
</job>
```

**XML tag conventions:**
- `<bracket_s>` — bracket for Single filing (maps to `brackets_single` array)
- `<bracket_m>` — bracket for MFJ filing (maps to `brackets_mfj` array)
- `<bracket_h>` — bracket for HOH filing (maps to `brackets_hoh` array)

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

Convert Excel files to XML (if modifying Excel sources):
```bash
cd sampleprojects/StateTax
mvn exec:java -Dexec.mainClass="com.dtrules.samples.statetax.CompileStateTax"
```

### Run Tests (Java)

Execute test cases with formatted output:
```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.statetax.TestStateTax"
```

Output includes state code, filing status, gross income, AGI, deductions, exemptions, taxable income, tax owed, effective rate, and processing notes.

### Run Tests (Go)

From the `go/` directory:
```bash
go test -v -run "TestStateTax" ./pkg/dtrules/
```

This runs three tests:
- `TestStateTaxIntegration` — full pipeline with Illinois flat tax
- `TestStateTaxEDDLoad` — validates all 7 entities and their attributes
- `TestStateTaxDTLoad` — validates all 9 decision tables load correctly

## Configuration

`DTRules.xml`:
```xml
<DTRules>
  <compiler>EL</compiler>
  <RuleSet name="StateTax" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="StateTax_edd.xml" />
    <Decisiontables  name="StateTax_dt.xml"  />
    <Map             name="StateTax_map.xml" />
  </RuleSet>
</DTRules>
```

## Mapping

The mapping configuration (`StateTax_map.xml`) uses explicit `list` attributes on `createentity` entries to route bracket entities into the correct filing-status arrays:

```xml
<createentity entity='bracket' tag='bracket_s' id='id' list='brackets_single'/>
<createentity entity='bracket' tag='bracket_m' id='id' list='brackets_mfj'/>
<createentity entity='bracket' tag='bracket_h' id='id' list='brackets_hoh'/>
```

The `state_config`, `job`, and `taxpayer` entities are created from the input XML by the data loader. In the Go implementation, `LoadDataAndPush` is used to push these singleton entities onto the entity stack after loading.

## Tax Data Sources

All bracket and deduction data uses **2025 tax year** rates from:
- [Tax Foundation — 2025 State Individual Income Tax Rates and Brackets](https://taxfoundation.org/data/all/state/state-income-tax-rates/)
- State revenue department publications and official rate schedules
- Legislative updates effective January 1, 2025 (GA 5.19%, LA flat 3%, ID 5.3%, OH 3.125%, WV new brackets)

## Verified Results (All 50 States + DC)

Every test case has been verified to produce the exact expected tax value using integer arithmetic:

| State | Type | Income | Taxable | Tax | | State | Type | Income | Taxable | Tax |
|-------|------|--------|---------|-----|---|-------|------|--------|---------|-----|
| AK | none | $80k | — | $0 | | MT | prog | $71k | $56,000 | $3,050 |
| AL | prog | $55k | $50,500 | $2,485 | | NC | flat | $140k | $114,500 | $4,866 |
| AR | prog | $48k | $45,590 | $1,692 | | ND | prog | $83k | $68,000 | $380 |
| AZ | flat | $72k | $57,000 | $1,425 | | NE | prog | $64k | $55,400 | $2,401 |
| CA | prog | $85k | $79,294 | $3,913 | | NH | none | $65k | — | $0 |
| CO | flat | $90k | $75,000 | $3,300 | | NJ | prog | $105k | $104,000 | $4,499 |
| CT | prog | $120k | $105,000 | $5,050 | | NM | prog | $76k | $61,000 | $2,457 |
| DC | prog | $95k | $80,000 | $5,200 | | NV | none | $70k | — | $0 |
| DE | prog | $62k | $58,750 | $2,874 | | NY | prog | $120k | $112,000 | $6,151 |
| FL | none | $95k | — | $0 | | OH | prog | $60k | $57,600 | $867 |
| GA | flat | $80k | $68,000 | $3,529 | | OK | prog | $77k | $69,650 | $3,117 |
| HI | prog | $95k | $89,456 | $5,688 | | OR | prog | $82k | $79,165 | $6,617 |
| IA | flat | $82k | $82,000 | $3,116 | | PA | flat | $85k | $85,000 | $2,609 |
| ID | flat | $68k | $52,250 | $2,769 | | RI | prog | $74k | $58,000 | $2,175 |
| IL | flat | $75k | $72,150 | $3,571 | | SC | prog | $69k | $54,000 | $2,670 |
| IN | flat | $58k | $57,000 | $1,710 | | SD | none | $55k | — | $0 |
| KS | prog | $70k | $57,235 | $3,106 | | TN | none | $88k | — | $0 |
| KY | flat | $73k | $69,730 | $2,789 | | TX | none | $100k | — | $0 |
| LA | flat | $66k | $53,500 | $1,605 | | UT | flat | $91k | $91,000 | $4,140 |
| MA | prog | $100k | $95,600 | $4,780 | | VA | prog | $65k | $55,320 | $2,923 |
| MD | prog | $85k | $78,450 | $3,673 | | VT | prog | $77k | $64,250 | $2,683 |
| ME | prog | $75k | $54,850 | $3,447 | | WA | none | $110k | — | $0 |
| MI | flat | $78k | $72,200 | $3,068 | | WI | prog | $81k | $66,040 | $3,102 |
| MN | prog | $88k | $73,050 | $4,494 | | WV | prog | $56k | $54,000 | $1,786 |
| MO | prog | $67k | $52,000 | $2,265 | | WY | none | $62k | — | $0 |
| MS | prog | $52k | $43,700 | $1,482 | | | | | | |

## Adding a New State or Updating Tax Data

1. Create a new test case XML file in `testfiles/TestScenarios/`
2. Set the `state_config` with the state's current rates, deductions, and brackets
3. Pre-compute `base_tax` for each bracket: `base_tax[i] = base_tax[i-1] + (ceiling[i-1] - floor[i-1]) × rate_bps[i-1] / 10000`
4. Set the last bracket's `ceiling` to `999999999`
5. Use `<bracket_s>` tags for Single, `<bracket_m>` for MFJ, `<bracket_h>` for HOH
6. No code changes needed — the engine is fully data-driven

## Limitations

- Tax brackets are provided for Single filing status in most test cases; MFJ and HOH bracket arrays are included where tested
- Special state rules not modeled: surtaxes (MA 4% over $1M), phase-outs, tax benefit recapture (NY), local taxes (NYC, IN counties), credits (as opposed to deductions)
- Deductions are standard only (no itemized deduction support)
- States that use credits instead of exemption deductions (CA, DE, IA, NE, OR) are modeled with `exemption_amount=0` for simplicity

## Related Projects

- **TestProject** — Minimal DTRules template to start from
- **CHIP** — Health insurance eligibility determination
- **KidAid** — Child assistance program eligibility

## DSL

Uses **EL (Expression Language)** for conditions and actions.

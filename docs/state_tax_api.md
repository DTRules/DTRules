# State Tax System API Documentation

This document provides comprehensive guidance for implementing state income tax calculations in the DTRules TaxReturn system.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Entity Structure](#entity-structure)
- [Decision Table Structure](#decision-table-structure)
- [Adding a New State](#adding-a-new-state)
- [Implementation Examples](#implementation-examples)
  - [Flat Tax: Illinois](#flat-tax-illinois)
  - [Progressive Tax: New Hampshire](#progressive-tax-new-hampshire)
  - [Progressive Tax: Montana](#progressive-tax-montana)
  - [Tax Credits](#tax-credits)
  - [Multi-State Allocation](#multi-state-allocation)
- [Postfix Notation Guide](#postfix-notation-guide)
- [Test Format](#test-format)
- [Constants Management](#constants-management)
- [Architecture Diagrams](#architecture-diagrams)

---

## Overview

The DTRules TaxReturn system implements state income tax calculations through a modular architecture using decision tables and entities. The system supports:

- Flat tax states (e.g., Illinois at 4.95%)
- Progressive tax states (e.g., New Hampshire, Montana)
- Multi-state returns for part-year residents and non-residents
- State-specific credits and deductions
- States with no income tax (TX, FL, WA, NV, etc.)

**Key Files:**
- **Entity Definitions**: `/sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- **Decision Tables**: `/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`
- **Test Cases**: `/sampleprojects/TaxReturn/testfiles/TestScenarios/`
- **Multi-State Docs**: `/sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md`

---

## Architecture

### Execution Flow

```
TABLE 1: Compute_Tax_Return (Entry Point)
├─> TABLE 2: Calculate_Gross_Income
├─> TABLE 3: Calculate_Deductions
├─> TABLE 4: Calculate_Taxable_Income
├─> TABLE 5: Calculate_Tax_Liability (Federal)
├─> TABLE 6: Calculate_Credits
│
├─> TABLE 40000: Dispatch_State_Tax *** STATE TAX ENTRY POINT ***
│   │
│   ├─> TABLE 45000: Allocate_Income_By_State (if multi-state)
│   │
│   ├─> TABLE 41100: Calculate_IL_Tax (Illinois)
│   ├─> TABLE 42700: Calculate_NH_Tax (New Hampshire)
│   ├─> TABLE 44900: Calculate_MT_Tax (Montana)
│   └─> [New state tables go here]
│
├─> TABLE 7: Calculate_Final_Balance
└─> TABLE 8: Validate_Results
```

### Table Numbering Convention

- **1-99**: Core federal tax calculation flow
- **6000s**: Federal tax credits (CTC, Education Credits, etc.)
- **19000-26999**: Special tax forms (K-1, IRA, Farm, Foreign, etc.)
- **40000-45000**: State tax tables
  - **TABLE 40000**: Dispatch_State_Tax (dispatcher/router)
  - **TABLE 41xxx**: Illinois state tax
  - **TABLE 42xxx**: New Hampshire state tax
  - **TABLE 44xxx**: Montana state tax
  - **TABLE 45000**: Allocate_Income_By_State (multi-state allocation)
  - **TABLE 4xxxx**: Reserved for future state implementations

**Recommendation**: When adding a new state, use the next available 41xxx, 43xxx, 46xxx, etc. range.

---

## Entity Structure

### Core State Tax Entities

#### `state_period` Entity

Represents a period of time during the tax year when a taxpayer resided or worked in a specific state. Used for multi-state returns.

**Location**: `TaxReturn_edd.xml`, referenced in `Core.xlsx`

**Fields:**
```xml
<entity_name>state_period</entity_name>
<fields>
  <field name="id" type="integer" access="rw"
         comment="Unique identifier for the state period"/>

  <field name="state_code" type="string" access="rw"
         comment="Two-letter state code (e.g., NY, FL, CA, TX)"/>

  <field name="start_date" type="date" access="rw"
         comment="Start date of residence/work in this state"/>

  <field name="end_date" type="date" access="rw"
         comment="End date of residence/work in this state"/>

  <field name="resident_status" type="string" access="rw"
         comment="full_year, part_year, or nonresident"/>

  <field name="days_in_state" type="integer" access="rc"
         comment="Number of days in this state (computed)"/>

  <field name="allocation_percentage" type="double" access="rc"
         comment="Percentage of income allocated (computed)"/>

  <field name="allocated_income" type="double" access="rc"
         comment="Income allocated to this state (computed)"/>

  <field name="allocated_withholding" type="double" access="rc"
         comment="Withholding allocated to this state (computed)"/>

  <field name="notes" type="array" access="rw"
         comment="Notes about this state period"/>
</fields>
```

**Access Modifiers:**
- `rw` = read/write (can be set externally)
- `rc` = read/computed (calculated by decision tables)

#### `state_tax_result` Entity

Stores the results of state tax calculations for a specific state.

**Location**: `TaxReturn_edd.xml`, referenced in `Core.xlsx`

**Fields:**
```xml
<entity_name>state_tax_result</entity_name>
<fields>
  <field name="state_code" type="string" access="rw"
         comment="Two-letter state code"/>

  <field name="state_agi" type="double" access="rc"
         comment="State adjusted gross income"/>

  <field name="state_taxable_income" type="double" access="rc"
         comment="State taxable income after deductions"/>

  <field name="state_tax_before_credits" type="double" access="rc"
         comment="State tax before credits"/>

  <field name="state_credits" type="double" access="rc"
         comment="Total state tax credits"/>

  <field name="state_tax_liability" type="double" access="rc"
         comment="State tax liability after credits"/>

  <field name="state_withholding" type="double" access="rc"
         comment="State tax withheld"/>

  <field name="state_refund_or_owed" type="double" access="rc"
         comment="Refund (positive) or owed (negative)"/>
</fields>
```

#### `job` Entity (Modified)

The root entity for a tax return calculation.

**State-Related Fields:**
```xml
<entity_name>job</entity_name>
<fields>
  <!-- Single-state field (backward compatible) -->
  <field name="state" type="string" access="rw"
         comment="Single state code for simple returns"/>

  <!-- Multi-state support -->
  <field name="state_periods" type="array" subtype="state_period" access="rw"
         comment="Array of periods in different states"/>

  <field name="state_tax_results" type="array" subtype="state_tax_result" access="rw"
         comment="Array of state tax results"/>
</fields>
```

#### `income` Entity (Modified)

**State-Related Fields:**
```xml
<entity_name>income</entity_name>
<fields>
  <field name="state_code" type="string" access="rw"
         comment="State where income was earned (for multi-state allocation)"/>

  <field name="state_period_id" type="integer" access="rw"
         comment="Reference to state_period for date-based allocation"/>
</fields>
```

---

## Decision Table Structure

### TABLE 40000: Dispatch_State_Tax

This is the main dispatcher that routes to state-specific calculation tables.

**Table Type**: ITERATIVE (processes each state_period)

**XML Structure**:
```xml
<decision_table>
  <table_name>Dispatch_State_Tax</table_name>
  <xls_file>State.xls</xls_file>
  <attribute_fields>
    <Type>ITERATIVE</Type>
    <COMMENTS>Dispatches to appropriate state tax calculation based on state code</COMMENTS>
    <TABLE_NUMBER>40000</TABLE_NUMBER>
  </attribute_fields>

  <contexts>
    <context>state_period</context>
  </contexts>

  <initial_actions>
    <initial_action>
      <action_description>Check for multi-state and allocate income</action_description>
      <action_postfix>
        job.state_periods arraysize 0 > if
          Allocate_Income_By_State execute
        then
      </action_postfix>
    </initial_action>
  </initial_actions>

  <conditions>
    <condition_details>
      <condition_number>1</condition_number>
      <condition_comment>Check state code</condition_comment>
      <condition_description>Is this Illinois?</condition_description>
      <condition_postfix>
        state_period.state_code "IL" streq
      </condition_postfix>
    </condition_details>

    <condition_details>
      <condition_number>2</condition_number>
      <condition_comment>Check state code</condition_comment>
      <condition_description>Is this New Hampshire?</condition_description>
      <condition_postfix>
        state_period.state_code "NH" streq
      </condition_postfix>
    </condition_details>

    <!-- More state conditions... -->
  </conditions>

  <columns>
    <column>
      <column_number>1</column_number>
      <conditions>Y - - - -</conditions>
      <actions>
        <action>
          <action_description>Calculate Illinois tax</action_description>
          <action_postfix>Calculate_IL_Tax execute</action_postfix>
        </action>
      </actions>
    </column>

    <column>
      <column_number>2</column_number>
      <conditions>- Y - - -</conditions>
      <actions>
        <action>
          <action_description>Calculate New Hampshire tax</action_description>
          <action_postfix>Calculate_NH_Tax execute</action_postfix>
        </action>
      </actions>
    </column>

    <!-- More state columns... -->
  </columns>
</decision_table>
```

---

## Adding a New State

Follow these steps to add a new state income tax calculation:

### Step 1: Choose a Table Number

Select an unused table number in the 40000-45000 range:
- **41xxx**: If following Illinois pattern
- **43xxx**: Next available range
- **46xxx-49xxx**: Additional ranges

Example: For California, use **TABLE 43100** (next available after 42xxx)

### Step 2: Add State Constants to EDD

Edit `TaxReturn_edd.xml` and add constants to the `constants` entity (around line 960).

**Example for California:**
```xml
<!-- California 2025 Constants -->
<attribute name="ca_tax_rate_1" type="double" default="0.0100"
           comment="California tax bracket 1: 1%"/>
<attribute name="ca_tax_rate_2" type="double" default="0.0200"
           comment="California tax bracket 2: 2%"/>
<attribute name="ca_tax_rate_3" type="double" default="0.0400"
           comment="California tax bracket 3: 4%"/>
<attribute name="ca_tax_rate_4" type="double" default="0.0600"
           comment="California tax bracket 4: 6%"/>
<attribute name="ca_tax_rate_5" type="double" default="0.0800"
           comment="California tax bracket 5: 8%"/>
<attribute name="ca_tax_rate_6" type="double" default="0.0930"
           comment="California tax bracket 6: 9.3%"/>
<attribute name="ca_tax_rate_7" type="double" default="0.1030"
           comment="California tax bracket 7: 10.3%"/>
<attribute name="ca_tax_rate_8" type="double" default="0.1130"
           comment="California tax bracket 8: 11.3%"/>
<attribute name="ca_tax_rate_9" type="double" default="0.1230"
           comment="California tax bracket 9: 12.3%"/>

<!-- California bracket thresholds for Single filers -->
<attribute name="ca_bracket_2_threshold_single" type="double" default="10099"
           comment="CA bracket 2 starts at $10,099"/>
<attribute name="ca_bracket_3_threshold_single" type="double" default="23942"
           comment="CA bracket 3 starts at $23,942"/>
<!-- ... more thresholds ... -->

<!-- California standard deduction -->
<attribute name="ca_standard_deduction_single" type="double" default="5202"
           comment="CA standard deduction for single filers"/>
<attribute name="ca_standard_deduction_mfj" type="double" default="10404"
           comment="CA standard deduction for MFJ"/>
```

**Naming Convention:**
- Use lowercase state abbreviation: `ca_`, `ny_`, `tx_`
- Rate constants: `{state}_tax_rate_{n}` or `{state}_rate_{n}`
- Thresholds: `{state}_bracket_{n}_threshold_{filing_status}`
- Deductions: `{state}_standard_deduction_{filing_status}`

### Step 3: Create State Tax Calculation Table

Add a new decision table to `TaxReturn_dt.xml`.

**Template:**
```xml
<!-- ====================================================================== -->
<!-- TABLE 43100: Calculate_CA_Tax - California State Income Tax          -->
<!-- Reference: California Form 540 (2025)                                 -->
<!-- ====================================================================== -->
<decision_table>
<table_name>Calculate_CA_Tax</table_name>
<xls_file>State.xls</xls_file>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>Calculates California state income tax per Form 540. [Brief description of tax structure]</COMMENTS>
  <TABLE_NUMBER>43100</TABLE_NUMBER>
</attribute_fields>

<contexts></contexts>

<initial_actions>
  <initial_action>
    <action_description>Initialize CA tax calculation</action_description>
    <action_postfix>
      "CALIFORNIA FORM 540 - State Income Tax Calculation" job.audit_trail swap addto
      result.agi /result.ca_agi xdef
    </action_postfix>
  </initial_action>
</initial_actions>

<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>Filing Status</condition_comment>
    <condition_description>Married Filing Jointly</condition_description>
    <condition_postfix>
      job.filing_status "married_filing_jointly" streq
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>2</condition_number>
    <condition_comment>Filing Status</condition_comment>
    <condition_description>Single</condition_description>
    <condition_postfix>
      job.filing_status "single" streq
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>3</condition_number>
    <condition_comment>Filing Status</condition_comment>
    <condition_description>Head of Household</condition_description>
    <condition_postfix>
      job.filing_status "head_of_household" streq
    </condition_postfix>
  </condition_details>
</conditions>

<columns>
  <!-- Column 1: Married Filing Jointly -->
  <column>
    <column_number>1</column_number>
    <conditions>Y - -</conditions>
    <actions>
      <action>
        <action_description>Apply MFJ standard deduction</action_description>
        <action_postfix>
          ca_standard_deduction_mfj /result.ca_standard_deduction xdef
          result.ca_agi result.ca_standard_deduction f- 0 fmax /result.ca_taxable_income xdef
        </action_postfix>
      </action>

      <action>
        <action_description>Calculate CA tax with progressive brackets (MFJ)</action_description>
        <action_postfix>
          <!-- Progressive bracket calculation logic here -->
          <!-- See examples below for flat vs progressive -->
        </action_postfix>
      </action>

      <action>
        <action_description>Log calculation to audit trail</action_description>
        <action_postfix>
          "  CA AGI: $" result.ca_agi cvs strconcat job.audit_trail swap addto
          "  CA Standard Deduction: $" result.ca_standard_deduction cvs strconcat job.audit_trail swap addto
          "  CA Taxable Income: $" result.ca_taxable_income cvs strconcat job.audit_trail swap addto
          "  CA Tax: $" result.ca_tax cvs strconcat job.audit_trail swap addto
        </action_postfix>
      </action>
    </actions>
  </column>

  <!-- Column 2: Single -->
  <column>
    <column_number>2</column_number>
    <conditions>- Y -</conditions>
    <actions>
      <!-- Similar structure for Single filing status -->
    </actions>
  </column>

  <!-- Column 3: Head of Household -->
  <column>
    <column_number>3</column_number>
    <conditions>- - Y</conditions>
    <actions>
      <!-- Similar structure for HOH filing status -->
    </actions>
  </column>
</columns>

</decision_table>
```

### Step 4: Add State to Dispatcher

Edit TABLE 40000 in `TaxReturn_dt.xml` to add the new state.

**Add a condition:**
```xml
<condition_details>
  <condition_number>4</condition_number>
  <condition_comment>Check state code</condition_comment>
  <condition_description>Is this California?</condition_description>
  <condition_postfix>
    state_period.state_code "CA" streq
  </condition_postfix>
</condition_details>
```

**Add a column:**
```xml
<column>
  <column_number>4</column_number>
  <conditions>- - - Y -</conditions>
  <actions>
    <action>
      <action_description>Calculate California tax</action_description>
      <action_postfix>Calculate_CA_Tax execute</action_postfix>
    </action>
  </actions>
</column>
```

**Update all other columns** to add a dash (-) in the new condition position.

### Step 5: Add Result Fields (Optional)

If your state needs custom intermediate calculation fields beyond the standard `state_tax_result` entity, add them to the `result` entity in `TaxReturn_edd.xml`.

**Example:**
```xml
<!-- California-specific intermediate fields -->
<attribute name="ca_agi" type="double" access="rc"
           comment="California adjusted gross income"/>
<attribute name="ca_standard_deduction" type="double" access="rc"
           comment="California standard deduction"/>
<attribute name="ca_taxable_income" type="double" access="rc"
           comment="California taxable income"/>
<attribute name="ca_bracket_1_tax" type="double" access="rc"
           comment="Tax from bracket 1"/>
<!-- ... more brackets ... -->
<attribute name="ca_tax" type="double" access="rc"
           comment="Total California tax"/>
```

### Step 6: Create Test Cases

Create test files in `/sampleprojects/TaxReturn/testfiles/TestScenarios/`.

**Naming Convention**: `TestCase_CA_01_Description.xml`, `TestCase_CA_02_Description.xml`, etc.

See [Test Format](#test-format) section for details.

### Step 7: Validate

1. Build the project to ensure XML is valid
2. Run tests to verify calculations
3. Check audit trail output for correctness

---

## Implementation Examples

### Flat Tax: Illinois

**Tax Structure:**
- Flat rate: 4.95%
- Personal exemption: $2,775 per person (taxpayers + dependents)

**TABLE 41100 Implementation:**

```xml
<decision_table>
<table_name>Calculate_IL_Tax</table_name>
<xls_file>State.xls</xls_file>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>Illinois flat tax at 4.95% with personal exemptions</COMMENTS>
  <TABLE_NUMBER>41100</TABLE_NUMBER>
</attribute_fields>

<initial_actions>
  <initial_action>
    <action_description>Initialize IL calculation</action_description>
    <action_postfix>
      "ILLINOIS FORM IL-1040 - State Income Tax Calculation" job.audit_trail swap addto
      0 local@ num_people >
      result.agi /result.il_agi xdef
    </action_postfix>
  </initial_action>
</initial_actions>

<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>Calculate number of people</condition_comment>
    <condition_description>Count taxpayers and dependents</condition_description>
    <condition_postfix>
      job.taxpayers arraysize job.dependents arraysize + local@ num_people >
      num_people 1 >=
    </condition_postfix>
  </condition_details>
</conditions>

<columns>
  <column>
    <column_number>1</column_number>
    <conditions>Y</conditions>
    <actions>
      <action>
        <action_description>Calculate total exemptions</action_description>
        <action_postfix>
          num_people il_personal_exemption f* /result.il_exemption_total xdef
          "  Number of people: " num_people cvs strconcat job.audit_trail swap addto
          "  IL Personal Exemption ($2,775 x " num_people cvs strconcat
            "): $" strconcat result.il_exemption_total cvs strconcat
            job.audit_trail swap addto
        </action_postfix>
      </action>

      <action>
        <action_description>Calculate IL taxable income</action_description>
        <action_postfix>
          result.il_agi result.il_exemption_total f- 0 fmax /result.il_taxable_income xdef
          "  IL Taxable Income: $" result.il_taxable_income cvs strconcat
            job.audit_trail swap addto
        </action_postfix>
      </action>

      <action>
        <action_description>Apply flat 4.95% rate</action_description>
        <action_postfix>
          result.il_taxable_income il_tax_rate f* /result.il_tax xdef
          "  IL Tax (4.95%): $" result.il_tax cvs strconcat
            job.audit_trail swap addto
        </action_postfix>
      </action>
    </actions>
  </column>
</columns>

</decision_table>
```

**Key Postfix Patterns:**

1. **Count array items:**
   ```
   job.taxpayers arraysize job.dependents arraysize + local@ num_people >
   ```

2. **Multiplication:**
   ```
   num_people il_personal_exemption f* /result.il_exemption_total xdef
   ```

3. **Subtraction with floor at zero:**
   ```
   result.il_agi result.il_exemption_total f- 0 fmax /result.il_taxable_income xdef
   ```
   Translation: `max(AGI - exemptions, 0)` → prevents negative taxable income

4. **Apply percentage:**
   ```
   result.il_taxable_income il_tax_rate f* /result.il_tax xdef
   ```

### Progressive Tax: New Hampshire

**Tax Structure:**
- Bracket 1: 3% on first $75,000
- Bracket 2: 5% on $75,001-$150,000
- Bracket 3: 7.5% on $150,001+
- Standard deduction: $8,000 (Single), $16,000 (MFJ)

**TABLE 42700 Implementation (Progressive Bracket Logic):**

```xml
<action>
  <action_description>Calculate NH tax with progressive brackets</action_description>
  <action_postfix>
    0 /result.nh_bracket_1_tax xdef
    0 /result.nh_bracket_2_tax xdef
    0 /result.nh_bracket_3_tax xdef

    <!-- Check if income exceeds bracket 3 threshold -->
    result.nh_taxable_income nh_bracket_3_threshold f> if
      <!-- All three brackets apply -->
      nh_bracket_2_threshold nh_bracket_1_rate f* /result.nh_bracket_1_tax xdef
      nh_bracket_3_threshold nh_bracket_2_threshold f- nh_bracket_2_rate f* /result.nh_bracket_2_tax xdef
      result.nh_taxable_income nh_bracket_3_threshold f- nh_bracket_3_rate f* /result.nh_bracket_3_tax xdef
    else
      <!-- Check if income exceeds bracket 2 threshold -->
      result.nh_taxable_income nh_bracket_2_threshold f> if
        <!-- Two brackets apply -->
        nh_bracket_2_threshold nh_bracket_1_rate f* /result.nh_bracket_1_tax xdef
        result.nh_taxable_income nh_bracket_2_threshold f- nh_bracket_2_rate f* /result.nh_bracket_2_tax xdef
      else
        <!-- Only first bracket applies -->
        result.nh_taxable_income nh_bracket_1_rate f* /result.nh_bracket_1_tax xdef
      endif
    endif

    <!-- Sum all brackets -->
    result.nh_bracket_1_tax result.nh_bracket_2_tax f+ result.nh_bracket_3_tax f+ /result.nh_tax xdef

    <!-- Audit trail -->
    "  NH Tax Calculation (Progressive Brackets):" job.audit_trail swap addto
    "    Bracket 1 (3% on $0-$75,000): $" result.nh_bracket_1_tax cvs strconcat
      job.audit_trail swap addto
    "    Bracket 2 (5% on $75,001-$150,000): $" result.nh_bracket_2_tax cvs strconcat
      job.audit_trail swap addto
    "    Bracket 3 (7.5% on $150,001+): $" result.nh_bracket_3_tax cvs strconcat
      job.audit_trail swap addto
    "  Total NH Tax: $" result.nh_tax cvs strconcat job.audit_trail swap addto
  </action_postfix>
</action>
```

**Key Progressive Tax Pattern:**

```
if (taxable_income > bracket_3_threshold) {
  bracket_1_tax = bracket_2_threshold * rate_1
  bracket_2_tax = (bracket_3_threshold - bracket_2_threshold) * rate_2
  bracket_3_tax = (taxable_income - bracket_3_threshold) * rate_3
} else if (taxable_income > bracket_2_threshold) {
  bracket_1_tax = bracket_2_threshold * rate_1
  bracket_2_tax = (taxable_income - bracket_2_threshold) * rate_2
} else {
  bracket_1_tax = taxable_income * rate_1
}
total_tax = bracket_1_tax + bracket_2_tax + bracket_3_tax
```

### Progressive Tax: Montana

**Tax Structure:**
- Bracket 1: 4.7% on income up to threshold
- Bracket 2: 5.9% on income above threshold
- Thresholds vary by filing status:
  - Single/MFS: $21,100
  - MFJ: $42,200
  - HOH: $31,700
- Standard deductions vary by filing status

**TABLE 44900 Implementation (Filing Status Conditions):**

```xml
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_description>Married Filing Jointly</condition_description>
    <condition_postfix>
      job.filing_status "married_filing_jointly" streq
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>2</condition_number>
    <condition_description>Single</condition_description>
    <condition_postfix>
      job.filing_status "single" streq
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>3</condition_number>
    <condition_description>Head of Household</condition_description>
    <condition_postfix>
      job.filing_status "head_of_household" streq
    </condition_postfix>
  </condition_details>
</conditions>

<columns>
  <!-- Column 1: MFJ -->
  <column>
    <column_number>1</column_number>
    <conditions>Y - -</conditions>
    <actions>
      <action>
        <action_description>Apply MFJ standard deduction and threshold</action_description>
        <action_postfix>
          mt_standard_deduction_mfj /result.mt_standard_deduction xdef
          mt_bracket_2_threshold_mfj local@ mt_threshold >

          result.mt_agi result.mt_standard_deduction f- 0 fmax /result.mt_taxable_income xdef

          <!-- Calculate progressive brackets -->
          result.mt_taxable_income mt_threshold f> if
            mt_threshold mt_bracket_1_rate f* /result.mt_bracket_1_tax xdef
            result.mt_taxable_income mt_threshold f- mt_bracket_2_rate f* /result.mt_bracket_2_tax xdef
          else
            result.mt_taxable_income mt_bracket_1_rate f* /result.mt_bracket_1_tax xdef
          endif

          result.mt_bracket_1_tax result.mt_bracket_2_tax f+ /result.mt_tax xdef
        </action_postfix>
      </action>
    </actions>
  </column>

  <!-- Column 2: Single -->
  <column>
    <column_number>2</column_number>
    <conditions>- Y -</conditions>
    <actions>
      <action>
        <action_description>Apply Single standard deduction and threshold</action_description>
        <action_postfix>
          mt_standard_deduction_single /result.mt_standard_deduction xdef
          mt_bracket_2_threshold_single local@ mt_threshold >

          <!-- Rest similar to MFJ column with different threshold -->
        </action_postfix>
      </action>
    </actions>
  </column>

  <!-- Column 3: HOH -->
  <!-- Similar structure -->
</columns>
```

**Key Pattern: Filing Status-Specific Values**

Use conditions to check filing status, then apply appropriate thresholds/deductions in each column:

```
Column 1 (MFJ):  mt_bracket_2_threshold_mfj, mt_standard_deduction_mfj
Column 2 (Single): mt_bracket_2_threshold_single, mt_standard_deduction_single
Column 3 (HOH): mt_bracket_2_threshold_hoh, mt_standard_deduction_hoh
```

### Tax Credits

Credits are typically implemented using the **ALL** table type, which evaluates all rows for each item in an array (e.g., each dependent).

**Example: Child Tax Credit (TABLE 6100)**

```xml
<decision_table>
<table_name>Calculate_Child_Tax_Credit</table_name>
<attribute_fields>
  <Type>ALL</Type>
  <COMMENTS>Evaluates CTC for all dependents</COMMENTS>
  <TABLE_NUMBER>6100</TABLE_NUMBER>
</attribute_fields>

<contexts>
  <context>dependent</context>
</contexts>

<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_description>Is this a child?</condition_description>
    <condition_postfix>
      dependent.relationship "child" streq
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>2</condition_number>
    <condition_description>Under age 17?</condition_description>
    <condition_postfix>
      dependent.age constants.ctc_age_limit <
    </condition_postfix>
  </condition_details>

  <condition_details>
    <condition_number>3</condition_number>
    <condition_description>Has SSN?</condition_description>
    <condition_postfix>
      dependent.has_ssn true beq
    </condition_postfix>
  </condition_details>
</conditions>

<columns>
  <!-- Column 1: Child under 17 with SSN → $2,000 CTC -->
  <column>
    <column_number>1</column_number>
    <conditions>Y Y Y</conditions>
    <actions>
      <action>
        <action_description>Award Child Tax Credit</action_description>
        <action_postfix>
          true /dependent.qualifies_for_ctc xdef
          constants.ctc_amount /dependent.ctc_amount xdef
          dependent.ctc_amount result.total_ctc f+ /result.total_ctc xdef
          "  Child Tax Credit for " dependent.name strconcat ": $" strconcat
            dependent.ctc_amount cvs strconcat job.audit_trail swap addto
        </action_postfix>
      </action>
    </actions>
  </column>

  <!-- Column 2: Child age 17+ with SSN → $500 ODC -->
  <column>
    <column_number>2</column_number>
    <conditions>Y N Y</conditions>
    <actions>
      <action>
        <action_description>Award Other Dependent Credit</action_description>
        <action_postfix>
          true /dependent.qualifies_for_odc xdef
          constants.odc_amount /dependent.odc_amount xdef
          dependent.odc_amount result.total_odc f+ /result.total_odc xdef
          "  Other Dependent Credit for " dependent.name strconcat ": $" strconcat
            dependent.odc_amount cvs strconcat job.audit_trail swap addto
        </action_postfix>
      </action>
    </actions>
  </column>

  <!-- Column 3: Non-child dependent with SSN → $500 ODC -->
  <column>
    <column_number>3</column_number>
    <conditions>N - Y</conditions>
    <actions>
      <action>
        <action_description>Award Other Dependent Credit</action_description>
        <action_postfix>
          <!-- Similar to column 2 -->
        </action_postfix>
      </action>
    </actions>
  </column>
</columns>

</decision_table>
```

**Key Pattern: Accumulating Credits**

```
credit_amount result.total_credit f+ /result.total_credit xdef
```

This adds the credit amount to the running total.

### Multi-State Allocation

**TABLE 45000: Allocate_Income_By_State**

For taxpayers who moved between states or worked in multiple states during the year.

```xml
<decision_table>
<table_name>Allocate_Income_By_State</table_name>
<xls_file>State.xls</xls_file>
<attribute_fields>
  <Type>ITERATIVE</Type>
  <COMMENTS>Allocates income across states for multi-state returns</COMMENTS>
  <TABLE_NUMBER>45000</TABLE_NUMBER>
</attribute_fields>

<contexts>
  <context>state_period</context>
</contexts>

<initial_actions>
  <initial_action>
    <action_description>Start allocation</action_description>
    <action_postfix>
      "Multi-State Income Allocation" job.audit_trail swap addto
    </action_postfix>
  </initial_action>
</initial_actions>

<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_description>State period exists</condition_description>
    <condition_postfix>
      state_period.state_code null:s ne
    </condition_postfix>
  </condition_details>
</conditions>

<columns>
  <column>
    <column_number>1</column_number>
    <conditions>Y</conditions>
    <actions>
      <action>
        <action_description>Calculate days in state</action_description>
        <action_postfix>
          state_period.end_date state_period.start_date daysbetween
            /state_period.days_in_state xdef

          state_period.days_in_state cvt:d 365.0 div
            /state_period.allocation_percentage xdef

          "  " state_period.state_code strconcat ": " strconcat
            state_period.days_in_state cvs strconcat " days (" strconcat
            state_period.allocation_percentage 100.0 f* cvs strconcat "%)" strconcat
            job.audit_trail swap addto
        </action_postfix>
      </action>

      <action>
        <action_description>Allocate income to state</action_description>
        <action_postfix>
          0.0 /state_period.allocated_income xdef
          0.0 /state_period.allocated_withholding xdef

          job.incomes foreach income do
            <!-- If income is state-specific, allocate only to matching state -->
            income.state_code state_period.state_code streq
            income.state_code null:s streq or
            if
              state_period.allocated_income
                income.gross_amount state_period.allocation_percentage f* f+
                /state_period.allocated_income xdef

              state_period.allocated_withholding
                income.tax_withheld state_period.allocation_percentage f* f+
                /state_period.allocated_withholding xdef
            then
          done

          "    Allocated Income: $" state_period.allocated_income cvs strconcat
            job.audit_trail swap addto
          "    Allocated Withholding: $" state_period.allocated_withholding cvs strconcat
            job.audit_trail swap addto
        </action_postfix>
      </action>
    </actions>
  </column>
</columns>

</decision_table>
```

**Key Pattern: Foreach Loop**

```
job.incomes foreach income do
  <!-- Condition check -->
  income.state_code state_period.state_code streq if
    <!-- Actions -->
  then
done
```

**Key Pattern: Date Arithmetic**

```
state_period.end_date state_period.start_date daysbetween /state_period.days_in_state xdef
state_period.days_in_state cvt:d 365.0 div /state_period.allocation_percentage xdef
```

---

## Postfix Notation Guide

DTRules uses **Reverse Polish Notation (RPN)** for all calculations. Operands are pushed onto a stack, and operators consume items from the stack.

### Basic Operators

#### Arithmetic

| Operator | Description | Example | Equivalent |
|----------|-------------|---------|------------|
| `f+` | Float addition | `a b f+` | `a + b` |
| `f-` | Float subtraction | `a b f-` | `a - b` |
| `f*` | Float multiplication | `a b f*` | `a * b` |
| `div` | Division | `a b div` | `a / b` |
| `fmax` | Maximum of two floats | `a b fmax` | `max(a, b)` |
| `fmin` | Minimum of two floats | `a b fmin` | `min(a, b)` |

#### Comparison

| Operator | Description | Example | Equivalent |
|----------|-------------|---------|------------|
| `f>` | Float greater than | `a b f>` | `a > b` |
| `f<` | Float less than | `a b f<` | `a < b` |
| `f>=` | Float greater or equal | `a b f>=` | `a >= b` |
| `f<=` | Float less or equal | `a b f<=` | `a <= b` |
| `feq` | Float equals | `a b feq` | `a == b` |
| `streq` | String equals | `a b streq` | `a.equals(b)` |
| `beq` | Boolean equals | `a b beq` | `a == b` |
| `ne` | Not equals | `a b ne` | `a != b` |

#### Logical

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `cond1 cond2 and` |
| `or` | Logical OR | `cond1 cond2 or` |
| `not` | Logical NOT | `cond not` |

#### Stack Operations

| Operator | Description | Stack Effect |
|----------|-------------|--------------|
| `dup` | Duplicate top | `a` → `a a` |
| `swap` | Swap top two | `a b` → `b a` |
| `pop` | Remove top | `a b` → `a` |

#### Variable Operations

| Operator | Description | Example |
|----------|-------------|---------|
| `!` | Store to variable (pop) | `value variable !` |
| `xdef` | Define variable | `value /variable_name xdef` |
| `>` | Store to local variable | `value local@ name >` |
| `local@` | Local variable marker | `0 local@ counter >` |

#### Type Conversion

| Operator | Description | Example |
|----------|-------------|---------|
| `cvt:d` | Convert to double | `integer cvt:d` |
| `cvs` or `cvt:s` | Convert to string | `number cvs` |

#### Control Flow

| Construct | Syntax | Description |
|-----------|--------|-------------|
| If-Then | `condition if ... then` | Execute if true |
| If-Else | `condition if ... else ... then` | Conditional branch |
| If-ElseIf | `cond1 if ... cond2 elseif ... else ... then` | Multiple conditions |

#### Array Operations

| Operator | Description | Example |
|----------|-------------|---------|
| `arraysize` or `length` | Get array length | `job.dependents arraysize` |
| `addto` | Add to array | `item array addto` |
| `foreach ... do ... done` | Iterate array | `array foreach item do ... done` |

#### String Operations

| Operator | Description | Example |
|----------|-------------|---------|
| `strconcat` | Concatenate strings | `"Hello " name strconcat` |

#### Date Operations

| Operator | Description | Example |
|----------|-------------|---------|
| `daysbetween` | Days between dates | `end_date start_date daysbetween` |

#### Execution

| Operator | Description | Example |
|----------|-------------|---------|
| `execute` | Execute decision table | `Calculate_Tax execute` |

### Common Patterns

#### 1. Simple Assignment

**Infix**: `result.agi = 50000`

**Postfix**:
```
50000 /result.agi xdef
```

#### 2. Arithmetic Assignment

**Infix**: `result.taxable = result.agi - result.deduction`

**Postfix**:
```
result.agi result.deduction f- /result.taxable xdef
```

#### 3. Maximum (Floor at Zero)

**Infix**: `result.taxable = max(result.agi - result.deduction, 0)`

**Postfix**:
```
result.agi result.deduction f- 0 fmax /result.taxable xdef
```

#### 4. Percentage Calculation

**Infix**: `result.tax = result.taxable * 0.0495`

**Postfix**:
```
result.taxable il_tax_rate f* /result.tax xdef
```

or inline:
```
result.taxable 0.0495 f* /result.tax xdef
```

#### 5. Conditional Assignment

**Infix**:
```java
if (job.filing_status.equals("single")) {
  result.deduction = 8000;
} else {
  result.deduction = 16000;
}
```

**Postfix**:
```
job.filing_status "single" streq if
  8000 /result.deduction xdef
else
  16000 /result.deduction xdef
then
```

#### 6. String Concatenation

**Infix**: `message = "Tax: $" + result.tax`

**Postfix**:
```
"Tax: $" result.tax cvs strconcat
```

#### 7. Multi-Part String Building

**Infix**: `message = "Bracket 1 (" + rate + "%): $" + tax`

**Postfix**:
```
"Bracket 1 (" rate cvs strconcat "%): $" strconcat tax cvs strconcat
```

#### 8. Add to Audit Trail

**Infix**: `job.audit_trail.add("Message: " + value)`

**Postfix**:
```
"Message: " value cvs strconcat job.audit_trail swap addto
```

**Why swap?** Because `addto` expects `array item addto`, but our stack has `item array`, so we swap them.

#### 9. Array Length Check

**Infix**: `if (job.dependents.size() > 0)`

**Postfix**:
```
job.dependents arraysize 0 >
```

#### 10. Sum Array Sizes

**Infix**: `count = job.taxpayers.size() + job.dependents.size()`

**Postfix**:
```
job.taxpayers arraysize job.dependents arraysize + /count xdef
```

#### 11. Local Variable

**Infix**:
```java
int threshold = 75000;
// use threshold
```

**Postfix**:
```
75000 local@ threshold >
threshold // later reference
```

#### 12. Foreach Loop

**Infix**:
```java
for (Income income : job.incomes) {
  if (income.state_code.equals("CA")) {
    total += income.amount;
  }
}
```

**Postfix**:
```
job.incomes foreach income do
  income.state_code "CA" streq if
    total income.amount f+ /total xdef
  then
done
```

#### 13. Progressive Bracket Logic

**Infix**:
```java
if (taxable > bracket2_threshold) {
  bracket1_tax = bracket2_threshold * rate1;
  bracket2_tax = (taxable - bracket2_threshold) * rate2;
} else {
  bracket1_tax = taxable * rate1;
  bracket2_tax = 0;
}
total_tax = bracket1_tax + bracket2_tax;
```

**Postfix**:
```
taxable bracket2_threshold f> if
  bracket2_threshold rate1 f* /bracket1_tax xdef
  taxable bracket2_threshold f- rate2 f* /bracket2_tax xdef
else
  taxable rate1 f* /bracket1_tax xdef
  0 /bracket2_tax xdef
then
bracket1_tax bracket2_tax f+ /total_tax xdef
```

---

## Test Format

Test files are XML documents in `/sampleprojects/TaxReturn/testfiles/TestScenarios/`.

### Naming Convention

- **Single-state**: `TestCase_{STATE}_{NN}_{Description}.xml`
  - Example: `TestCase_NH_01_Single_W2.xml`
  - Example: `TestCase_IL_02_Retirement_Income.xml`

- **Multi-state**: `TestCase_MultiState_{NN}_{Description}.xml`
  - Example: `TestCase_MultiState_01_NY_FL_Move.xml`
  - Example: `TestCase_MultiState_03_Traveling_Consultant.xml`

### Test Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<job>
  <!-- Test identification -->
  <id>NH-01</id>
  <tax_year>2025</tax_year>
  <filing_status>single</filing_status>
  <state>NH</state>

  <!-- Expected values for validation -->
  <expected_agi>65000</expected_agi>
  <expected_taxable_income>50000</expected_taxable_income>
  <expected_tax>5000</expected_tax>
  <expected_nh_agi>65000</expected_nh_agi>
  <expected_nh_taxable_income>57000</expected_nh_taxable_income>
  <expected_nh_tax>1710</expected_nh_tax>
  <expected_total_tax>6710</expected_total_tax>

  <!-- Taxpayer information -->
  <taxpayers>
    <taxpayer id="1">
      <name>John Smith</name>
      <ssn>123-45-6789</ssn>
      <date_of_birth>1/15/1980</date_of_birth>
      <w2_wages>65000</w2_wages>
      <w2_withholding>9500</w2_withholding>
    </taxpayer>
  </taxpayers>

  <!-- Dependents (if applicable) -->
  <dependents>
    <dependent id="1">
      <name>Jane Smith</name>
      <relationship>child</relationship>
      <age>12</age>
      <has_ssn>true</has_ssn>
    </dependent>
  </dependents>

  <!-- Multi-state information (if applicable) -->
  <state_periods>
    <state_period id="1">
      <state_code>NH</state_code>
      <start_date>1/1/2025</start_date>
      <end_date>4/30/2025</end_date>
      <resident_status>nonresident</resident_status>
    </state_period>
  </state_periods>

  <!-- Income details (for multi-state allocation) -->
  <incomes>
    <income id="1">
      <type>w2_wages</type>
      <gross_amount>65000</gross_amount>
      <tax_withheld>9500</tax_withheld>
      <state_code>NH</state_code>
      <state_period_id>1</state_period_id>
    </income>
  </incomes>
</job>
```

### Test Case Examples

#### Single-State Test: Illinois Single Filer

```xml
<?xml version="1.0" encoding="UTF-8"?>
<job>
  <id>IL-01</id>
  <tax_year>2025</tax_year>
  <filing_status>single</filing_status>
  <state>IL</state>

  <expected_agi>75000</expected_agi>
  <expected_il_agi>75000</expected_il_agi>
  <expected_il_taxable_income>72225</expected_il_taxable_income>
  <expected_il_tax>3575</expected_il_tax>

  <taxpayers>
    <taxpayer id="1">
      <name>Jane Doe</name>
      <w2_wages>75000</w2_wages>
      <w2_withholding>3600</w2_withholding>
    </taxpayer>
  </taxpayers>
</job>
```

**Calculation**:
- AGI: $75,000
- Exemptions: 1 person × $2,775 = $2,775
- Taxable Income: $75,000 - $2,775 = $72,225
- Tax: $72,225 × 4.95% = $3,575.14 ≈ $3,575

#### Multi-State Test: NY to FL Move

```xml
<?xml version="1.0" encoding="UTF-8"?>
<job>
  <id>MultiState-01</id>
  <tax_year>2025</tax_year>
  <filing_status>single</filing_status>
  <state>FL</state>

  <expected_agi>90000</expected_agi>

  <taxpayers>
    <taxpayer id="1">
      <name>Bob Johnson</name>
      <w2_wages>90000</w2_wages>
      <w2_withholding>12000</w2_withholding>
    </taxpayer>
  </taxpayers>

  <state_periods>
    <state_period id="1">
      <state_code>NY</state_code>
      <start_date>1/1/2025</start_date>
      <end_date>7/31/2025</end_date>
      <resident_status>part_year</resident_status>
      <!-- Expected: 212 days = 58.1% -->
    </state_period>
    <state_period id="2">
      <state_code>FL</state_code>
      <start_date>8/1/2025</start_date>
      <end_date>12/31/2025</end_date>
      <resident_status>part_year</resident_status>
      <!-- Expected: 153 days = 41.9% -->
    </state_period>
  </state_periods>
</job>
```

**Calculation**:
- Total income: $90,000
- NY period: 212 days = 58.1% → $52,290
- FL period: 153 days = 41.9% → $37,710
- NY tax calculated on $52,290
- FL has no state income tax

### Running Tests

Tests are executed through the DTRules test framework. The system:

1. Loads the test XML
2. Executes the decision tables
3. Compares computed values to expected values
4. Reports pass/fail for each assertion

---

## Constants Management

All state tax constants are defined in the `constants` entity in `TaxReturn_edd.xml` (around line 960).

### Naming Convention

```xml
<!-- {state}_{constant_type}_{details} -->
<attribute name="il_tax_rate" type="double" default="0.0495"/>
<attribute name="il_personal_exemption" type="double" default="2775"/>

<attribute name="nh_bracket_1_rate" type="double" default="0.0300"/>
<attribute name="nh_bracket_2_threshold" type="double" default="75000"/>
<attribute name="nh_standard_deduction_single" type="double" default="8000"/>

<attribute name="ca_bracket_1_rate" type="double" default="0.0100"/>
<attribute name="ca_bracket_2_threshold_single" type="double" default="10099"/>
<attribute name="ca_standard_deduction_mfj" type="double" default="10404"/>
```

### Guidelines

1. **State prefix**: Always use lowercase two-letter state code (e.g., `ca_`, `ny_`)
2. **Rate constants**: `{state}_tax_rate` or `{state}_bracket_{n}_rate`
3. **Thresholds**: `{state}_bracket_{n}_threshold` or `{state}_bracket_{n}_threshold_{filing_status}`
4. **Deductions**: `{state}_standard_deduction_{filing_status}`
5. **Credits**: `{state}_{credit_name}_amount`
6. **Type**: Use `double` for all monetary and percentage values
7. **Percentages**: Express as decimals (4.95% = 0.0495)

### Example: Adding California Constants

```xml
<!-- California 2025 Tax Constants -->
<!-- Progressive tax rates -->
<attribute name="ca_bracket_1_rate" type="double" default="0.0100"
           comment="CA bracket 1: 1%"/>
<attribute name="ca_bracket_2_rate" type="double" default="0.0200"
           comment="CA bracket 2: 2%"/>
<attribute name="ca_bracket_3_rate" type="double" default="0.0400"
           comment="CA bracket 3: 4%"/>
<attribute name="ca_bracket_4_rate" type="double" default="0.0600"
           comment="CA bracket 4: 6%"/>
<attribute name="ca_bracket_5_rate" type="double" default="0.0800"
           comment="CA bracket 5: 8%"/>
<attribute name="ca_bracket_6_rate" type="double" default="0.0930"
           comment="CA bracket 6: 9.3%"/>

<!-- Thresholds for Single filers -->
<attribute name="ca_bracket_2_threshold_single" type="double" default="10099"/>
<attribute name="ca_bracket_3_threshold_single" type="double" default="23942"/>
<attribute name="ca_bracket_4_threshold_single" type="double" default="37788"/>
<attribute name="ca_bracket_5_threshold_single" type="double" default="52455"/>
<attribute name="ca_bracket_6_threshold_single" type="double" default="66295"/>

<!-- Thresholds for MFJ -->
<attribute name="ca_bracket_2_threshold_mfj" type="double" default="20198"/>
<attribute name="ca_bracket_3_threshold_mfj" type="double" default="47884"/>
<!-- ... etc ... -->

<!-- Standard deductions -->
<attribute name="ca_standard_deduction_single" type="double" default="5202"/>
<attribute name="ca_standard_deduction_mfj" type="double" default="10404"/>
<attribute name="ca_standard_deduction_hoh" type="double" default="10412"/>
```

---

## Architecture Diagrams

### State Tax Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    TABLE 1: Compute_Tax_Return                  │
│                         (Main Entry Point)                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ├─→ Federal Tax Calculation (Tables 2-6)
                             │   ├─ Gross Income
                             │   ├─ Deductions
                             │   ├─ Taxable Income
                             │   ├─ Tax Liability
                             │   └─ Credits
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│              TABLE 40000: Dispatch_State_Tax                    │
│                    (State Tax Dispatcher)                       │
│                                                                 │
│  Type: ITERATIVE (processes each state_period)                 │
│  Context: state_period                                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ├─→ Check if multi-state
                             │   (job.state_periods.size() > 0?)
                             │
                             ├─→ If multi-state:
                             │   Execute TABLE 45000: Allocate_Income_By_State
                             │
                             ├─→ Route to state-specific tables:
                             │
        ┌────────────────────┼────────────────────┬───────────────┐
        │                    │                    │               │
        ▼                    ▼                    ▼               ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐    ┌──────────┐
│  TABLE 41100  │    │  TABLE 42700  │    │  TABLE 44900  │    │  Future  │
│   Illinois    │    │New Hampshire  │    │   Montana     │    │  States  │
│   Flat Tax    │    │  Progressive  │    │  Progressive  │    │          │
│    4.95%      │    │   3/5/7.5%    │    │   4.7/5.9%    │    │   ...    │
└───────────────┘    └───────────────┘    └───────────────┘    └──────────┘
        │                    │                    │               │
        └────────────────────┴────────────────────┴───────────────┘
                             │
                             ▼
                   Store results in:
                   state_tax_result entity
```

### Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                            job                                  │
│  (Root entity - represents a complete tax return)               │
├─────────────────────────────────────────────────────────────────┤
│ - id: string                                                    │
│ - tax_year: integer                                             │
│ - filing_status: string (single, mfj, hoh, mfs)                 │
│ - state: string (backward compatible, single state)             │
│ - state_periods: array<state_period>  ← MULTI-STATE            │
│ - state_tax_results: array<state_tax_result>                    │
│ - taxpayers: array<taxpayer>                                    │
│ - dependents: array<dependent>                                  │
│ - incomes: array<income>                                        │
│ - audit_trail: array<string>                                    │
└───────────┬─────────────────────────────────┬──────────────────┘
            │                                 │
            │ Contains multiple               │ Contains multiple
            ▼                                 ▼
┌───────────────────────────┐    ┌──────────────────────────────┐
│      state_period         │    │     state_tax_result         │
├───────────────────────────┤    ├──────────────────────────────┤
│ - id: integer             │    │ - state_code: string         │
│ - state_code: string      │    │ - state_agi: double          │
│ - start_date: date        │    │ - state_taxable_income       │
│ - end_date: date          │    │ - state_tax_before_credits   │
│ - resident_status         │    │ - state_credits: double      │
│ - days_in_state (calc)    │    │ - state_tax_liability        │
│ - allocation_% (calc)     │    │ - state_withholding          │
│ - allocated_income (calc) │    │ - state_refund_or_owed       │
│ - allocated_withhold(calc)│    └──────────────────────────────┘
└───────────────────────────┘
            │
            │ Referenced by
            ▼
┌───────────────────────────┐
│        income             │
├───────────────────────────┤
│ - id: integer             │
│ - type: string            │
│ - gross_amount: double    │
│ - tax_withheld: double    │
│ - state_code: string      │  ← Links income to state
│ - state_period_id: int    │  ← Links to specific period
└───────────────────────────┘
```

### Multi-State Allocation Flow

```
┌─────────────────────────────────────────────────────────────────┐
│           TABLE 45000: Allocate_Income_By_State                 │
│                  (ITERATIVE on state_period)                    │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
            For each state_period in job.state_periods:
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  NY Period   │    │  FL Period   │    │  CA Period   │
│  212 days    │    │  153 days    │    │  (example)   │
│  58.1%       │    │  41.9%       │    │              │
└──────┬───────┘    └──────┬───────┘    └──────┬───────┘
       │                   │                   │
       ├─→ Calculate days_in_state              │
       │   (end_date - start_date)              │
       │                                        │
       ├─→ Calculate allocation_percentage      │
       │   (days_in_state / 365)                │
       │                                        │
       └─→ Allocate income:                     │
           For each income in job.incomes:      │
             If income.state_code == "NY"       │
               OR income.state_code == null:    │
                 allocated_income +=            │
                   income.amount * 58.1%        │
                                                │
                    Results stored in:          │
                 state_period.allocated_income  │
              state_period.allocated_withholding│
```

### Decision Table Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                       DECISION TABLE                            │
├─────────────────────────────────────────────────────────────────┤
│  TABLE NUMBER: 41100                                            │
│  TABLE NAME: Calculate_IL_Tax                                   │
│  TYPE: FIRST (first matching column executes)                   │
│  CONTEXT: (none - operates on job/result entities)              │
└─────────────────────────────────────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   INITIAL    │    │  CONDITIONS  │    │   COLUMNS    │
│   ACTIONS    │    │              │    │              │
├──────────────┤    ├──────────────┤    ├──────────────┤
│ Execute once │    │ Evaluated    │    │ Actions      │
│ at table     │    │ for each     │    │ execute if   │
│ start        │    │ column       │    │ conditions   │
│              │    │              │    │ match        │
│ - Initialize │    │ Cond 1: Y/-  │    │ Col 1:       │
│ - Set        │    │ Cond 2: Y/-  │    │   Cond: Y    │
│   defaults   │    │ Cond 3: Y/-  │    │   Actions:   │
│ - Audit log  │    │              │    │   - Calc AGI │
│              │    │ Each returns │    │   - Calc Tax │
│              │    │ true/false   │    │   - Log      │
└──────────────┘    └──────────────┘    └──────────────┘
```

### Data Flow: Single-State Return

```
┌──────────────┐
│  Test XML    │
│  Input Data  │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────┐
│  job entity populated:              │
│  - filing_status = "single"         │
│  - state = "IL"                     │
│  - taxpayers[0].w2_wages = 75000    │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  TABLE 1: Compute_Tax_Return        │
│  Calculates federal tax             │
│  → result.agi = 75000               │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  TABLE 40000: Dispatch_State_Tax    │
│  Checks: job.state == "IL"          │
│  Routes to: Calculate_IL_Tax        │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  TABLE 41100: Calculate_IL_Tax      │
│  - result.il_agi = 75000            │
│  - exemptions = 1 × 2775 = 2775     │
│  - taxable = 75000 - 2775 = 72225   │
│  - tax = 72225 × 4.95% = 3575       │
│  → result.il_tax = 3575             │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  result entity contains:            │
│  - agi = 75000                      │
│  - il_agi = 75000                   │
│  - il_taxable_income = 72225        │
│  - il_tax = 3575                    │
└─────────────────────────────────────┘
```

---

## Summary Checklist

When adding a new state, ensure you complete all these steps:

- [ ] Choose an unused table number (e.g., 43100 for California)
- [ ] Add state constants to `constants` entity in `TaxReturn_edd.xml`
  - [ ] Tax rates
  - [ ] Bracket thresholds (if progressive)
  - [ ] Standard deductions (by filing status)
  - [ ] Credits (if applicable)
- [ ] Create state tax calculation table in `TaxReturn_dt.xml`
  - [ ] Table metadata (name, number, type, comments)
  - [ ] Initial actions (initialize, audit log header)
  - [ ] Conditions (filing status checks, income thresholds, etc.)
  - [ ] Columns with actions (one per filing status or scenario)
- [ ] Add state condition to TABLE 40000 (Dispatch_State_Tax)
- [ ] Add state column to TABLE 40000 that calls your new table
- [ ] (Optional) Add result fields to `result` entity for intermediate values
- [ ] Create test cases
  - [ ] At least one test per filing status
  - [ ] Edge cases (low income, high income, multiple brackets)
  - [ ] Include expected values for validation
- [ ] Build project and verify XML is valid
- [ ] Run tests and verify calculations
- [ ] Review audit trail output for correctness

---

## Related Documentation

- **Multi-State Allocation**: `/sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md`
- **General API Guide**: `/docs/API-GUIDE.md`
- **Architecture Overview**: `/docs/ARCHITECTURE.md`
- **Entity Definitions**: `/sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- **Decision Tables**: `/sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`

---

## Contact and Support

For questions or issues with state tax implementations, please refer to:

- **DTRules Documentation**: `/docs/`
- **GitHub Issues**: https://github.com/DTRules/DTRules
- **Decision Table Reference**: See `/docs/ARCHITECTURE.md` for decision table concepts

---

**Document Version**: 1.0
**Last Updated**: 2026-03-23
**Covers**: Illinois (IL), New Hampshire (NH), Montana (MT) state tax implementations

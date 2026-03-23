# State Tax API Documentation

Comprehensive guide for implementing state income tax calculations in DTRules.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Entity Structure](#entity-structure)
- [Table Naming Conventions](#table-naming-conventions)
- [How to Add a New State](#how-to-add-a-new-state)
- [Implementation Examples](#implementation-examples)
  - [Flat Tax (Illinois)](#flat-tax-illinois)
  - [Progressive Tax (New Hampshire)](#progressive-tax-new-hampshire)
  - [Tax Credits (Montana)](#tax-credits-montana)
  - [Multi-State Allocation](#multi-state-allocation)
- [Postfix Notation](#postfix-notation)
- [Test Format](#test-format)
- [Architecture Diagrams](#architecture-diagrams)
- [Best Practices](#best-practices)

---

## Overview

The DTRules state tax system provides a flexible framework for calculating state income taxes across multiple jurisdictions. It supports:

- **Flat tax rates** (e.g., Illinois)
- **Progressive bracket systems** (e.g., New Hampshire, Montana)
- **Tax credits and deductions** (e.g., Montana)
- **Multi-state resident allocation** (part-year residents, nonresidents)
- **States with no income tax** (TX, FL, WA, NV, WY, SD, AK, TN)

**Location:** `sampleprojects/TaxReturn/`

**Key Components:**
- Entity definitions: `xml/TaxReturn_edd.xml`
- Decision tables: `xml/TaxReturn_dt.xml`
- Test cases: `testfiles/TestScenarios/`
- Documentation: `docs/MULTI_STATE_ALLOCATION.md`

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    State Tax Calculation Flow                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  TABLE 40000: Dispatch_State_Tax                                │
│  - Routes to appropriate state calculation                      │
│  - Handles multi-state allocation                               │
│  - Falls back to single-state (job.state)                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┴──────────────┐
                ▼                            ▼
┌───────────────────────────┐  ┌───────────────────────────┐
│ Multi-State?              │  │ Single-State              │
│                           │  │                           │
│ TABLE 45000:              │  │ Direct to state table:    │
│ Allocate_Income_By_State  │  │ - Calculate_IL_Tax        │
│                           │  │ - Calculate_NH_Tax        │
│ Allocates income based on │  │ - Calculate_MT_Tax        │
│ days in each state        │  │                           │
└───────────────────────────┘  └───────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────────┐
│  For each state_period:                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Calculate_IL_Tax│  │ Calculate_NH_Tax│  │ Calculate_MT_Tax│ │
│  │ (TABLE 41100)   │  │ (TABLE 42700)   │  │ (TABLE 44900)   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Result Entity                                                  │
│  - il_tax, nh_tax, mt_tax                                       │
│  - Detailed calculations stored for audit trail                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Entity Structure

### Core Entities

#### `job` Entity
The main input entity containing taxpayer information and configuration.

**Key Fields:**
```
- state (string): Primary state of residence
- filing_status (string): "single", "married_filing_jointly", "head_of_household", etc.
- taxpayers (array): Array of taxpayer entities
- dependents (array): Array of dependent entities
- incomes (array): Array of income entities
- state_periods (array): Array of state_period entities (for multi-state)
- audit_trail (array): Audit log of calculations
```

**State-Specific Constants (Illinois example):**
```
- il_tax_rate (double): 0.0495 (4.95%)
- il_personal_exemption (double): 2775
```

**State-Specific Constants (New Hampshire example):**
```
- nh_bracket_1_rate (double): 0.03 (3%)
- nh_bracket_1_threshold (integer): 75000
- nh_bracket_2_rate (double): 0.05 (5%)
- nh_bracket_2_threshold (integer): 150000
- nh_bracket_3_rate (double): 0.075 (7.5%)
- nh_standard_deduction_single (integer): 8000
- nh_standard_deduction_mfj (integer): 16000
```

#### `state_period` Entity
Represents a period of time in a specific state (for multi-state returns).

**Fields:**
```xml
<entity name='state_period'>
  <field name='id' type='integer' comment='State period ID'/>
  <field name='state_code' type='string' comment='Two-letter state code'/>
  <field name='start_date' type='date' comment='Start date'/>
  <field name='end_date' type='date' comment='End date'/>
  <field name='resident_status' type='string' comment='full_year, part_year, or nonresident'/>
  <field name='days_in_state' type='integer' comment='Computed'/>
  <field name='allocation_percentage' type='double' comment='Computed'/>
  <field name='allocated_income' type='double' comment='Computed'/>
  <field name='allocated_withholding' type='double' comment='Computed'/>
  <field name='notes' type='array'/>
</entity>
```

**Resident Status Values:**
- `full_year`: Resident for entire tax year
- `part_year`: Moved into or out of state during tax year
- `nonresident`: Worked in state but not a resident

#### `income` Entity
Represents a single income source.

**Key Fields for State Tax:**
```
- type (string): "w2_wages", "self_employment", "pension", etc.
- gross_amount (double): Total income amount
- state_code (string): State where income was earned (for multi-state)
- state_period_id (integer): Reference to state_period
- state_withholding (double): State tax withheld
```

#### `result` Entity
Stores calculated tax results.

**Illinois Fields:**
```
- il_agi (double): Illinois Adjusted Gross Income
- il_exemption_total (double): Total personal exemptions
- il_taxable_income (double): Taxable income after exemptions
- il_tax (double): Calculated Illinois tax
```

**New Hampshire Fields:**
```
- nh_agi (double): New Hampshire AGI
- nh_standard_deduction (double): Standard deduction based on filing status
- nh_taxable_income (double): Taxable income after deduction
- nh_bracket_1_tax (double): Tax from bracket 1
- nh_bracket_2_tax (double): Tax from bracket 2
- nh_bracket_3_tax (double): Tax from bracket 3
- nh_tax (double): Total NH tax
```

**Montana Fields:**
```
- mt_agi (double): Montana AGI
- mt_standard_deduction (double): Standard deduction
- mt_taxable_income (double): Taxable income
- mt_bracket_1_tax (double): Tax from bracket 1
- mt_bracket_2_tax (double): Tax from bracket 2
- mt_tax (double): Total MT tax
```

---

## Table Naming Conventions

### Table Number Ranges

The state tax system uses a structured numbering convention:

| Range | Purpose | Example |
|-------|---------|---------|
| **40000-40999** | State dispatchers and routing | 40000: Dispatch_State_Tax |
| **41000-41999** | States A-I | 41100: Calculate_IL_Tax (Illinois) |
| **42000-42999** | States J-N | 42700: Calculate_NH_Tax (New Hampshire) |
| **43000-43999** | States O-T | (Reserved for future states) |
| **44000-44999** | States U-Z | 44900: Calculate_MT_Tax (Montana) |
| **45000-45999** | Helper/utility tables | 45000: Allocate_Income_By_State |

### Naming Pattern

Decision tables follow this naming pattern:
```
Calculate_[STATE_CODE]_Tax
```

Examples:
- `Calculate_IL_Tax` - Illinois
- `Calculate_NH_Tax` - New Hampshire
- `Calculate_MT_Tax` - Montana
- `Calculate_CA_Tax` - California (future)

### Table Types

| Type | Description | Use Case |
|------|-------------|----------|
| **FIRST** | Executes first matching column | Single tax calculation |
| **ITERATIVE** | Processes array elements | Multi-state, multiple income sources |
| **UNIQUE** | All matching columns execute | Multiple conditions apply |

---

## How to Add a New State

Follow these steps to implement a new state tax calculation:

### Step 1: Research State Tax Rules

Gather the following information:
- Tax rate(s) and bracket structure
- Standard deduction amounts
- Personal exemptions
- Credits and special adjustments
- Filing status variations
- Official tax forms for reference

### Step 2: Define Constants in Entity

Add state-specific constants to the `job` entity in `TaxReturn_edd.xml`:

```xml
<!-- California example -->
<field name='ca_bracket_1_rate' type='double' default='0.01' comment='1%'/>
<field name='ca_bracket_1_threshold' type='integer' default='10000'/>
<field name='ca_bracket_2_rate' type='double' default='0.02' comment='2%'/>
<field name='ca_bracket_2_threshold' type='integer' default='23000'/>
<!-- ... additional brackets ... -->
<field name='ca_standard_deduction_single' type='integer' default='5363'/>
<field name='ca_standard_deduction_mfj' type='integer' default='10726'/>
```

### Step 3: Add Result Fields

Add calculation result fields to the `result` entity:

```xml
<!-- California result fields -->
<field name='ca_agi' type='double' default='0' comment='California AGI'/>
<field name='ca_standard_deduction' type='double' default='0'/>
<field name='ca_taxable_income' type='double' default='0'/>
<field name='ca_bracket_1_tax' type='double' default='0'/>
<field name='ca_bracket_2_tax' type='double' default='0'/>
<!-- ... additional bracket tax fields ... -->
<field name='ca_tax' type='double' default='0' comment='Total CA tax'/>
```

### Step 4: Create Decision Table

Create a new decision table following the naming convention. Choose appropriate table number from the ranges above.

**Example structure:**

```xml
<decision_table>
<table_name>Calculate_CA_Tax</table_name>
<xls_file>State.xls</xls_file>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>Calculates California state income tax per Form 540...</COMMENTS>
  <TABLE_NUMBER>41200</TABLE_NUMBER>
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
    <condition_comment>Determine filing status</condition_comment>
    <condition_description>Check if single filer</condition_description>
    <condition_postfix>
      job.filing_status "single" streq
    </condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="2" column_value="N"></condition_column>
  </condition_details>
</conditions>
<actions>
  <!-- Action details here -->
</actions>
</decision_table>
```

### Step 5: Add to Dispatcher

Update `Dispatch_State_Tax` (TABLE 40000) to route to your new state:

Add a condition:
```xml
<condition_details>
  <condition_number>X</condition_number>
  <condition_comment>California</condition_comment>
  <condition_description>State is CA</condition_description>
  <condition_postfix>
    state_period.state_code "CA" streq
  </condition_postfix>
  <condition_column column_number="X" column_value="Y"></condition_column>
</condition_details>
```

Add an action:
```xml
<action_details>
  <action_number>X</action_number>
  <action_comment>Calculate California tax</action_comment>
  <action_description>Call CA calculation table</action_description>
  <action_postfix>
    "Calculating California state tax..." job.audit_trail swap addto
    Calculate_CA_Tax dtexecute
  </action_postfix>
  <action_column column_number="X" column_value="X"></action_column>
</action_details>
```

### Step 6: Create Test Cases

Create XML test files in `testfiles/TestScenarios/`:

**Single-State Test (`TestCase_CA_01_Single.xml`):**
```xml
<test_case name="CA Single Filer">
  <job>
    <state>CA</state>
    <filing_status>single</filing_status>
    <incomes>
      <income id="1">
        <type>w2_wages</type>
        <gross_amount>75000</gross_amount>
      </income>
    </incomes>
  </job>
  <expected_results>
    <ca_tax>XXXX.XX</ca_tax>
  </expected_results>
</test_case>
```

**Multi-State Test (`TestCase_MultiState_CA_NV.xml`):**
```xml
<test_case name="CA to NV Move">
  <job>
    <state>NV</state>
    <state_periods>
      <state_period id="1">
        <state_code>CA</state_code>
        <start_date>1/1/2025</start_date>
        <end_date>6/30/2025</end_date>
        <resident_status>part_year</resident_status>
      </state_period>
      <state_period id="2">
        <state_code>NV</state_code>
        <start_date>7/1/2025</start_date>
        <end_date>12/31/2025</end_date>
        <resident_status>part_year</resident_status>
      </state_period>
    </state_periods>
    <!-- ... income details ... -->
  </job>
</test_case>
```

### Step 7: Test and Validate

1. Compile the decision tables: `mvn exec:java -Dexec.mainClass="CompileClass"`
2. Run test cases: `mvn exec:java -Dexec.mainClass="TestClass"`
3. Verify calculations against official tax forms
4. Check audit trail output for correctness

---

## Implementation Examples

### Flat Tax (Illinois)

Illinois uses a simple flat tax rate with personal exemptions.

**Tax Formula:**
```
Illinois AGI = Federal AGI - Retirement Income
Taxable Income = Illinois AGI - (Number of People × Personal Exemption)
Illinois Tax = Taxable Income × 4.95%
```

**Implementation (TABLE 41100: Calculate_IL_Tax):**

```xml
<decision_table>
<table_name>Calculate_IL_Tax</table_name>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>Illinois flat tax at 4.95% with $2,775 personal exemptions</COMMENTS>
  <TABLE_NUMBER>41100</TABLE_NUMBER>
</attribute_fields>
<initial_actions>
  <initial_action>
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
    <condition_postfix>
      job.taxpayers arraysize job.dependents arraysize + local@ num_people >
      num_people 1 >=
    </condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>Calculate IL tax with exemptions</action_comment>
    <action_postfix>
      result.agi local@ il_retirement_subtraction >
      0 local@ state_bond_addition >

      { dup type income_type >
        income_type "pension" streq
        income_type "ira_distribution" streq or
        income_type "social_security" streq or
        if
          dup gross_amount il_retirement_subtraction f+
          local@ il_retirement_subtraction >
        endif
      } job.incomes forall

      il_retirement_subtraction 0 f> if
        result.il_agi il_retirement_subtraction f- /result.il_agi xdef
      endif

      num_people il_personal_exemption f* /result.il_exemption_total xdef
      result.il_agi result.il_exemption_total f- 0 fmax /result.il_taxable_income xdef
      result.il_taxable_income il_tax_rate f* /result.il_tax xdef
    </action_postfix>
  </action_details>
</actions>
</decision_table>
```

**Key Points:**
- Subtracts all retirement income (pension, IRA, Social Security)
- Calculates personal exemptions based on number of people
- Applies flat 4.95% rate
- Uses `fmax` to ensure taxable income doesn't go negative

---

### Progressive Tax (New Hampshire)

New Hampshire uses progressive tax brackets with different rates.

**Tax Formula:**
```
NH AGI = Federal AGI (simplified for this example)
Taxable Income = NH AGI - Standard Deduction
Bracket 1 Tax = min(Taxable Income, $75,000) × 3%
Bracket 2 Tax = min(max(Taxable Income - $75,000, 0), $75,000) × 5%
Bracket 3 Tax = max(Taxable Income - $150,000, 0) × 7.5%
Total Tax = Bracket 1 + Bracket 2 + Bracket 3
```

**Implementation (TABLE 42700: Calculate_NH_Tax):**

```xml
<decision_table>
<table_name>Calculate_NH_Tax</table_name>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>NH progressive brackets: 3% on first $75k, 5% on $75k-$150k, 7.5% above $150k</COMMENTS>
  <TABLE_NUMBER>42700</TABLE_NUMBER>
</attribute_fields>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>Filing status is single</condition_comment>
    <condition_postfix>
      job.filing_status "single" streq
    </condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
    <condition_column column_number="2" column_value="N"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>Calculate NH tax - Single</action_comment>
    <action_postfix>
      nh_standard_deduction_single /result.nh_standard_deduction xdef
      result.nh_agi result.nh_standard_deduction f- 0 fmax /result.nh_taxable_income xdef

      <!-- Bracket 1: 3% on first $75,000 -->
      result.nh_taxable_income nh_bracket_1_threshold imin
      nh_bracket_1_rate f* /result.nh_bracket_1_tax xdef

      <!-- Bracket 2: 5% on $75,000 to $150,000 -->
      result.nh_taxable_income nh_bracket_1_threshold isub 0 imax
      nh_bracket_1_threshold imin
      nh_bracket_2_rate f* /result.nh_bracket_2_tax xdef

      <!-- Bracket 3: 7.5% above $150,000 -->
      result.nh_taxable_income nh_bracket_2_threshold isub 0 imax
      nh_bracket_3_rate f* /result.nh_bracket_3_tax xdef

      result.nh_bracket_1_tax result.nh_bracket_2_tax f+
      result.nh_bracket_3_tax f+ /result.nh_tax xdef
    </action_postfix>
  </action_details>
  <action_details>
    <action_number>2</action_number>
    <action_comment>Calculate NH tax - MFJ</action_comment>
    <action_postfix>
      nh_standard_deduction_mfj /result.nh_standard_deduction xdef
      <!-- Similar bracket calculations with MFJ deduction -->
    </action_postfix>
  </action_details>
</actions>
</decision_table>
```

**Key Points:**
- Uses `imin` (integer minimum) and `imax` (integer maximum) for bracket calculations
- Separates tax calculation by bracket for transparency
- Different standard deductions by filing status
- Each bracket amount is calculated and stored separately

---

### Tax Credits (Montana)

Montana supports both progressive brackets and standard deductions.

**Tax Formula:**
```
MT AGI = Federal AGI
Taxable Income = MT AGI - Standard Deduction
Bracket 1 Tax = min(Taxable Income, Threshold) × 4.7%
Bracket 2 Tax = max(Taxable Income - Threshold, 0) × 5.9%
Total Tax = Bracket 1 + Bracket 2
```

**Implementation (TABLE 44900: Calculate_MT_Tax):**

```xml
<decision_table>
<table_name>Calculate_MT_Tax</table_name>
<attribute_fields>
  <Type>FIRST</Type>
  <COMMENTS>Montana progressive brackets with standard deductions</COMMENTS>
  <TABLE_NUMBER>44900</TABLE_NUMBER>
</attribute_fields>
<conditions>
  <condition_details>
    <condition_number>1</condition_number>
    <condition_comment>Single filer</condition_comment>
    <condition_postfix>
      job.filing_status "single" streq
    </condition_postfix>
    <condition_column column_number="1" column_value="Y"></condition_column>
  </condition_details>
  <condition_details>
    <condition_number>2</condition_number>
    <condition_comment>MFJ filer</condition_comment>
    <condition_postfix>
      job.filing_status "married_filing_jointly" streq
    </condition_postfix>
    <condition_column column_number="2" column_value="Y"></condition_column>
  </condition_details>
</conditions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>MT tax - Single</action_comment>
    <action_postfix>
      mt_standard_deduction_single /result.mt_standard_deduction xdef
      mt_bracket_2_threshold_single local@ bracket_2_threshold >

      result.mt_agi result.mt_standard_deduction f- 0 fmax /result.mt_taxable_income xdef

      <!-- Bracket 1: 4.7% up to threshold -->
      result.mt_taxable_income bracket_2_threshold imin
      mt_bracket_1_rate f* /result.mt_bracket_1_tax xdef

      <!-- Bracket 2: 5.9% above threshold -->
      result.mt_taxable_income bracket_2_threshold isub 0 imax
      mt_bracket_2_rate f* /result.mt_bracket_2_tax xdef

      result.mt_bracket_1_tax result.mt_bracket_2_tax f+ /result.mt_tax xdef
    </action_postfix>
  </action_details>
  <action_details>
    <action_number>2</action_number>
    <action_comment>MT tax - MFJ</action_comment>
    <action_postfix>
      mt_standard_deduction_mfj /result.mt_standard_deduction xdef
      mt_bracket_2_threshold_mfj local@ bracket_2_threshold >
      <!-- Similar calculations with MFJ threshold -->
    </action_postfix>
  </action_details>
</actions>
</decision_table>
```

**Key Points:**
- Bracket thresholds vary by filing status
- Uses local variables for cleaner code
- Standard deductions reduce AGI before bracket calculations
- Two-bracket structure is simpler than NH's three brackets

---

### Multi-State Allocation

For taxpayers who moved between states or worked in multiple states.

**Allocation Formula:**
```
Days in State = End Date - Start Date
Allocation Percentage = Days in State / 365
Allocated Income = Total Income × Allocation Percentage
```

**Implementation (TABLE 45000: Allocate_Income_By_State):**

```xml
<decision_table>
<table_name>Allocate_Income_By_State</table_name>
<attribute_fields>
  <Type>ITERATIVE</Type>
  <COMMENTS>Allocates income across states based on days worked/lived</COMMENTS>
  <TABLE_NUMBER>45000</TABLE_NUMBER>
</attribute_fields>
<contexts>state_period</contexts>
<initial_actions>
  <initial_action>
    <action_postfix>
      "=== Allocating income by state ===" job.audit_trail swap addto
      state_period.start_date state_period.end_date daysbetween
      /state_period.days_in_state xdef

      state_period.days_in_state 365 div /state_period.allocation_percentage xdef

      0.0 /state_period.allocated_income xdef
      0.0 /state_period.allocated_withholding xdef
    </action_postfix>
  </initial_action>
</initial_actions>
<actions>
  <action_details>
    <action_number>1</action_number>
    <action_comment>Allocate income to this state</action_comment>
    <action_postfix>
      { dup state_code local@ income_state >
        dup gross_amount local@ income_amount >
        dup state_withholding local@ withholding_amount >

        income_state state_period.state_code streq
        income_state null eq or
        if
          state_period.allocated_income
          income_amount state_period.allocation_percentage f* f+
          /state_period.allocated_income xdef

          state_period.allocated_withholding
          withholding_amount state_period.allocation_percentage f* f+
          /state_period.allocated_withholding xdef
        endif
      } job.incomes forall
    </action_postfix>
  </action_details>
</actions>
</decision_table>
```

**Key Points:**
- Uses ITERATIVE type to process each `state_period`
- Calculates days using `daysbetween` operator
- Allocates income if `state_code` matches or is null
- Proportionally allocates withholding with income

**Multi-State Dispatcher Updates:**

The dispatcher (TABLE 40000) must handle multi-state scenarios:

```xml
<initial_actions>
  <initial_action>
    <action_postfix>
      job.state_periods arraysize 0 > if
        <!-- Multi-state: Call allocation first -->
        Allocate_Income_By_State dtexecute
      endif
    </action_postfix>
  </initial_action>
</initial_actions>
```

---

## Postfix Notation

DTRules uses a **stack-based postfix notation** (Reverse Polish Notation) for all expressions and actions in decision tables. Understanding this notation is crucial for implementing state tax calculations.

### Stack-Based Execution

All operations use a stack:
1. Push operands onto stack
2. Execute operator (pops operands, pushes result)
3. Repeat until expression complete

### Basic Operations

#### Arithmetic

**Infix:** `result.agi - 12000`
**Postfix:** `result.agi 12000 f-`

**Infix:** `income * tax_rate`
**Postfix:** `income tax_rate f*`

**Infix:** `(agi - deduction) * 0.05`
**Postfix:** `agi deduction f- 0.05 f*`

#### Operators

| Operation | Integer | Float | Description |
|-----------|---------|-------|-------------|
| Add | `iadd` or `+` | `f+` or `fadd` | Addition |
| Subtract | `isub` or `-` | `f-` or `fsub` | Subtraction |
| Multiply | `imul` or `*` | `f*` or `fmul` | Multiplication |
| Divide | `idiv` or `/` or `div` | `f/` or `fdiv` | Division |
| Maximum | `imax` | `fmax` | Maximum of two values |
| Minimum | `imin` | `fmin` | Minimum of two values |

### Variable Operations

#### Reading Variables
Simply reference the variable name:
```
result.agi          # Pushes value onto stack
job.filing_status   # Pushes value onto stack
```

#### Writing Variables
Use `xdef` (define) or `>` operators:
```
# Set result.il_agi to result.agi value
result.agi /result.il_agi xdef

# Equivalent using > operator
result.agi local@ temp_agi >
```

#### Local Variables
Declare and use local variables:
```
# Declare local variable
0 local@ num_people >

# Use local variable
num_people il_personal_exemption f*
```

### String Operations

```
# String concatenation
"Tax: $" result.tax cvs strconcat

# String comparison
job.filing_status "single" streq    # String equal
state_period.state_code "CA" streq
```

### Type Conversions

| Operator | Converts To | Example |
|----------|-------------|---------|
| `cvs` | String | `result.tax cvs` |
| `cvi` | Integer | `"123" cvi` |
| `cvr` | Float/Real | `"123.45" cvr` |
| `cvb` | Boolean | `"true" cvb` |
| `cvd` | Date | `"1/1/2025" cvd` |

### Control Flow

#### If/Then/Else
```
condition if
  # True branch
  value1 /result xdef
else
  # False branch
  value2 /result xdef
endif
```

**Example:**
```
result.il_agi result.il_exemption_total f- 0 fmax /result.il_taxable_income xdef
```
Equivalent to: `max(agi - exemption, 0)`

#### Loops (ForAll)

Process array elements:
```
{
  # Code block executed for each element
  # 'dup' duplicates top stack element
  dup gross_amount total f+ local@ total >
} job.incomes forall
```

### Complex Example: Progressive Tax Bracket

**Goal:** Calculate tax for NH Bracket 2 (5% on $75k-$150k)

**Formula:** `min(max(taxable_income - 75000, 0), 75000) * 0.05`

**Postfix:**
```
result.nh_taxable_income        # Push taxable income
nh_bracket_1_threshold          # Push 75000
isub                            # Subtract: income - 75000
0                               # Push 0
imax                            # Max: max(income - 75000, 0)
nh_bracket_1_threshold          # Push 75000
imin                            # Min: min(result, 75000)
nh_bracket_2_rate               # Push 0.05
f*                              # Multiply: result * 0.05
/result.nh_bracket_2_tax xdef   # Store in result.nh_bracket_2_tax
```

### Audit Trail Logging

Add messages to audit trail:
```
"Starting Illinois tax calculation" job.audit_trail swap addto

"  Illinois AGI: $" result.il_agi cvs strconcat
" (Form IL-1040)" strconcat
job.audit_trail swap addto
```

**Breakdown:**
1. `"text"` - Push string literal
2. `result.il_agi` - Push value
3. `cvs` - Convert to string
4. `strconcat` - Concatenate strings
5. `job.audit_trail` - Push audit trail array
6. `swap` - Swap top two stack elements
7. `addto` - Add string to array

### Common Patterns

#### Maximum of Zero (Non-Negative)
```
value 0 fmax    # Ensures result >= 0
```

#### Clamping to Range
```
value min_val fmax max_val fmin    # Clamp between min and max
```

#### Conditional Assignment
```
condition if
  value1
else
  value2
endif
/result xdef
```

#### Array Sum
```
0.0 local@ total >
{ dup amount total f+ local@ total > } items forall
total /result xdef
```

---

## Test Format

Test cases are XML files that define inputs and expected outputs.

### Test File Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<test_suite>
  <test_case name="Descriptive Test Name">
    <!-- Input Data -->
    <job>
      <state>XX</state>
      <filing_status>single</filing_status>
      <taxpayers>
        <taxpayer id="1">
          <first_name>John</first_name>
          <last_name>Doe</last_name>
          <ssn>123-45-6789</ssn>
          <date_of_birth>1/1/1980</date_of_birth>
        </taxpayer>
      </taxpayers>

      <incomes>
        <income id="1">
          <type>w2_wages</type>
          <gross_amount>75000</gross_amount>
          <federal_withholding>8500</federal_withholding>
          <state_withholding>3500</state_withholding>
        </income>
      </incomes>
    </job>

    <!-- Expected Results -->
    <expected_results>
      <il_agi>75000.00</il_agi>
      <il_exemption_total>2775.00</il_exemption_total>
      <il_taxable_income>72225.00</il_taxable_income>
      <il_tax>3575.14</il_tax>
    </expected_results>
  </test_case>
</test_suite>
```

### Single-State Test Example

**TestCase_IL_01_Single_W2.xml:**
```xml
<test_case name="IL Single Filer with W-2 Income">
  <job>
    <state>IL</state>
    <filing_status>single</filing_status>

    <taxpayers>
      <taxpayer id="1">
        <first_name>Alice</first_name>
        <last_name>Johnson</last_name>
      </taxpayer>
    </taxpayers>

    <incomes>
      <income id="1">
        <type>w2_wages</type>
        <gross_amount>65000</gross_amount>
        <employer_name>Tech Corp</employer_name>
      </income>
    </incomes>
  </job>

  <expected_results>
    <il_agi>65000.00</il_agi>
    <il_exemption_total>2775.00</il_exemption_total>
    <il_taxable_income>62225.00</il_taxable_income>
    <il_tax>3080.14</il_tax>
  </expected_results>
</test_case>
```

### Multi-State Test Example

**TestCase_MultiState_NH_IL.xml:**
```xml
<test_case name="Multi-State: NH to IL Move">
  <job>
    <state>IL</state>  <!-- Final state -->
    <filing_status>single</filing_status>

    <state_periods>
      <state_period id="1">
        <state_code>NH</state_code>
        <start_date>1/1/2025</start_date>
        <end_date>6/30/2025</end_date>
        <resident_status>part_year</resident_status>
      </state_period>
      <state_period id="2">
        <state_code>IL</state_code>
        <start_date>7/1/2025</start_date>
        <end_date>12/31/2025</end_date>
        <resident_status>part_year</resident_status>
      </state_period>
    </state_periods>

    <taxpayers>
      <taxpayer id="1">
        <first_name>Bob</first_name>
        <last_name>Smith</last_name>
      </taxpayer>
    </taxpayers>

    <incomes>
      <income id="1">
        <type>w2_wages</type>
        <gross_amount>100000</gross_amount>
        <!-- No state_code: allocate proportionally -->
      </income>
    </incomes>
  </job>

  <expected_results>
    <!-- NH allocation: 181 days / 365 = 49.6% -->
    <nh_allocated_income>49600.00</nh_allocated_income>
    <nh_tax>1248.00</nh_tax>

    <!-- IL allocation: 184 days / 365 = 50.4% -->
    <il_allocated_income>50400.00</il_allocated_income>
    <il_tax>2357.14</il_tax>
  </expected_results>
</test_case>
```

### Progressive Tax Test Example

**TestCase_NH_03_High_Income_All_Brackets.xml:**
```xml
<test_case name="NH High Income - All Three Brackets">
  <job>
    <state>NH</state>
    <filing_status>single</filing_status>

    <taxpayers>
      <taxpayer id="1">
        <first_name>Carol</first_name>
        <last_name>Williams</last_name>
      </taxpayer>
    </taxpayers>

    <incomes>
      <income id="1">
        <type>w2_wages</type>
        <gross_amount>200000</gross_amount>
      </income>
    </incomes>
  </job>

  <expected_results>
    <nh_agi>200000.00</nh_agi>
    <nh_standard_deduction>8000.00</nh_standard_deduction>
    <nh_taxable_income>192000.00</nh_taxable_income>

    <!-- Bracket 1: $75,000 × 3% = $2,250 -->
    <nh_bracket_1_tax>2250.00</nh_bracket_1_tax>

    <!-- Bracket 2: $75,000 × 5% = $3,750 -->
    <nh_bracket_2_tax>3750.00</nh_bracket_2_tax>

    <!-- Bracket 3: ($192,000 - $150,000) × 7.5% = $3,150 -->
    <nh_bracket_3_tax>3150.00</nh_bracket_3_tax>

    <!-- Total: $2,250 + $3,750 + $3,150 = $9,150 -->
    <nh_tax>9150.00</nh_tax>
  </expected_results>
</test_case>
```

### Running Tests

```bash
cd sampleprojects/TaxReturn

# Compile decision tables
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.CompileTaxReturn"

# Run all tests
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.TestTaxReturn"

# Run specific test
mvn exec:java -Dexec.mainClass="com.dtrules.samples.taxreturn.TestTaxReturn" \
  -Dexec.args="TestCase_IL_01_Single_W2.xml"
```

---

## Architecture Diagrams

### Overall System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                           │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │ Web Service  │   │ Batch Jobs   │   │ CLI Tools    │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DTRules Engine                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  RulesDirectory                                           │  │
│  │  - Manages rule sets                                      │  │
│  │  - Configuration (DTRules.xml)                            │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │                                     │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  RuleSet                                                  │  │
│  │  - Entity definitions (*_edd.xml)                         │  │
│  │  - Decision tables (*_dt.xml)                             │  │
│  │  - Mappings (*_map.xml)                                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │                                     │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  IRSession (per evaluation)                               │  │
│  │  - Entity instances                                       │  │
│  │  - Execution state                                        │  │
│  │  - Stack-based interpreter                                │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Data Layer                                  │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │ XML Files    │   │ Java Objects │   │ Databases    │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### State Tax Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  Application calls session.execute("Dispatch_State_Tax")       │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│  TABLE 40000: Dispatch_State_Tax                                │
│                                                                 │
│  Check: job.state_periods array empty?                          │
└─────────────────────────────────────────────────────────────────┘
            │                                    │
            │ Empty                              │ Has entries
            ▼                                    ▼
  ┌──────────────────────┐         ┌────────────────────────────┐
  │ Single-State Mode    │         │ Multi-State Mode           │
  │                      │         │                            │
  │ Use job.state        │         │ Execute:                   │
  │ Call Calculate_XX_Tax│         │ Allocate_Income_By_State   │
  └──────────────────────┘         └────────────────────────────┘
            │                                    │
            │                                    ▼
            │              ┌────────────────────────────────────┐
            │              │ For each state_period:             │
            │              │ - Calculate days_in_state          │
            │              │ - Calculate allocation_percentage  │
            │              │ - Allocate income and withholding  │
            │              └────────────────────────────────────┘
            │                                    │
            │                                    ▼
            │              ┌────────────────────────────────────┐
            │              │ Iterate state_periods array:       │
            │              │ Dispatch to state calculation      │
            │              └────────────────────────────────────┘
            │                                    │
            └──────────────┬─────────────────────┘
                           ▼
         ┌────────────────────────────────────────────┐
         │  Route by state_code:                      │
         │  - IL → Calculate_IL_Tax (41100)           │
         │  - NH → Calculate_NH_Tax (42700)           │
         │  - MT → Calculate_MT_Tax (44900)           │
         │  - TX/FL/WA/NV → Log "No income tax"       │
         └────────────────────────────────────────────┘
                           │
                           ▼
         ┌────────────────────────────────────────────┐
         │  State-Specific Calculation                │
         │  1. Initialize (set AGI)                   │
         │  2. Apply state adjustments                │
         │  3. Calculate deductions/exemptions        │
         │  4. Determine taxable income               │
         │  5. Apply tax rates/brackets               │
         │  6. Store results in result entity         │
         │  7. Log to audit trail                     │
         └────────────────────────────────────────────┘
                           │
                           ▼
         ┌────────────────────────────────────────────┐
         │  Return control to application             │
         │  Results available in result entity        │
         └────────────────────────────────────────────┘
```

### Entity Relationship Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                            job                               │
├──────────────────────────────────────────────────────────────┤
│ - state: string                                              │
│ - filing_status: string                                      │
│ - taxpayers: array <taxpayer>                                │
│ - dependents: array <dependent>                              │
│ - incomes: array <income>                                    │
│ - state_periods: array <state_period>                        │
│ - audit_trail: array <string>                                │
│ - il_tax_rate, nh_bracket_1_rate, etc. (constants)          │
└──────────────────────────────────────────────────────────────┘
                    │          │          │
        ┌───────────┘          │          └───────────┐
        ▼                      ▼                      ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────────┐
│  taxpayer   │       │   income    │       │  state_period   │
├─────────────┤       ├─────────────┤       ├─────────────────┤
│ - id        │       │ - id        │       │ - id            │
│ - name      │       │ - type      │       │ - state_code    │
│ - ssn       │       │ - amount    │       │ - start_date    │
│ - dob       │       │ - state_code│◀──────│ - end_date      │
└─────────────┘       │ - state_id  │       │ - days_in_state │
                      └─────────────┘       │ - alloc_pct     │
                                            │ - alloc_income  │
                                            └─────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                          result                              │
├──────────────────────────────────────────────────────────────┤
│ Federal Results:                                             │
│ - agi, taxable_income, federal_tax                           │
│                                                              │
│ Illinois Results:                                            │
│ - il_agi, il_exemption_total, il_taxable_income, il_tax     │
│                                                              │
│ New Hampshire Results:                                       │
│ - nh_agi, nh_standard_deduction, nh_taxable_income          │
│ - nh_bracket_1_tax, nh_bracket_2_tax, nh_bracket_3_tax      │
│ - nh_tax                                                     │
│                                                              │
│ Montana Results:                                             │
│ - mt_agi, mt_standard_deduction, mt_taxable_income          │
│ - mt_bracket_1_tax, mt_bracket_2_tax, mt_tax                │
└──────────────────────────────────────────────────────────────┘
```

### Decision Table Processing Model

```
┌─────────────────────────────────────────────────────────────┐
│              Decision Table: Calculate_IL_Tax               │
├─────────────────────────────────────────────────────────────┤
│  Type: FIRST (execute first matching column)                │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Initial Actions (always executed)                          │
│  - Set up local variables                                   │
│  - Initialize result fields                                 │
│  - Log start of calculation                                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Conditions (evaluated left to right)                       │
│                                                             │
│     Condition 1    Condition 2    Condition 3              │
│  ┌──────────────┬──────────────┬──────────────┐           │
│  │ Column 1     │ Column 2     │ Column 3     │           │
│  ├──────────────┼──────────────┼──────────────┤           │
│  │ Y            │ N            │ -            │           │
│  │ num_people   │ num_people   │ (don't care) │           │
│  │ >= 1         │ = 0          │              │           │
│  └──────────────┴──────────────┴──────────────┘           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Find First Matching Column                                 │
│  - Evaluate all conditions for Column 1                     │
│  - If all match, execute Column 1 actions                   │
│  - Otherwise, try Column 2, then Column 3, etc.             │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Actions (execute for matching column)                      │
│                                                             │
│     Action 1       Action 2       Action 3                 │
│  ┌──────────────┬──────────────┬──────────────┐           │
│  │ Column 1     │ Column 2     │ Column 3     │           │
│  ├──────────────┼──────────────┼──────────────┤           │
│  │ X            │              │ X            │           │
│  │ Calculate    │              │ Special      │           │
│  │ with exemp.  │              │ handling     │           │
│  └──────────────┴──────────────┴──────────────┘           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Results stored in result entity                            │
│  - result.il_agi                                            │
│  - result.il_taxable_income                                 │
│  - result.il_tax                                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Best Practices

### 1. Entity Design

**DO:**
- Use descriptive field names (`il_taxable_income` not `il_ti`)
- Store intermediate calculations for audit trail
- Use appropriate data types (integer for counts, double for money)
- Add comments to all entity fields
- Group related constants in the `job` entity

**DON'T:**
- Overload single field with multiple meanings
- Use generic field names like `temp1`, `value2`
- Mix different state results in same fields
- Forget to initialize result fields to 0

### 2. Decision Table Design

**DO:**
- Use FIRST type for single-column selection (tax calculations)
- Use ITERATIVE type for processing arrays (multi-state, multiple incomes)
- Keep conditions simple and testable
- Log all major calculation steps to audit trail
- Use meaningful column descriptions

**DON'T:**
- Create overly complex conditions
- Nest decision tables more than 2-3 levels deep
- Forget to handle edge cases (zero income, negative values)
- Skip initial actions for setup

### 3. Postfix Notation

**DO:**
- Use local variables for intermediate results
- Comment complex postfix expressions in XML
- Test postfix expressions incrementally
- Use `fmax` to enforce non-negative results
- Maintain consistent operator usage (f+ for floats, + for integers)

**DON'T:**
- Leave orphaned values on the stack
- Use wrong operator types (mixing integer and float ops)
- Create excessively long postfix expressions (break into steps)
- Forget type conversions (cvs, cvi, cvr)

### 4. Testing

**DO:**
- Test boundary conditions (zero income, threshold amounts)
- Test all bracket combinations for progressive taxes
- Create multi-state test cases
- Verify calculations against official tax forms
- Include both typical and edge cases

**DON'T:**
- Test only "happy path" scenarios
- Skip multi-state scenarios
- Use unrealistic test data
- Forget to test filing status variations

### 5. Documentation

**DO:**
- Reference official state tax forms in comments
- Document tax formulas in table comments
- Maintain audit trail with descriptive messages
- Update documentation when rates change
- Include effective tax year in documentation

**DON'T:**
- Assume formulas are self-explanatory
- Skip documentation of special cases
- Use ambiguous terminology
- Forget to document assumptions

### 6. State Tax Implementation

**DO:**
- Follow naming conventions consistently
- Reuse patterns from existing state implementations
- Store all tax constants in entity definitions (not hard-coded)
- Handle states with no income tax explicitly
- Implement backward compatibility for single-state returns

**DON'T:**
- Hard-code tax rates in decision tables
- Forget to update dispatcher when adding new states
- Skip validation of state codes
- Assume all states have income tax

### 7. Multi-State Handling

**DO:**
- Allocate income based on actual days in state
- Support both proportional and source-based allocation
- Fall back to single-state gracefully
- Document resident status values
- Test part-year and nonresident scenarios

**DON'T:**
- Assume equal allocation across states
- Forget to handle states with no income tax
- Skip withholding allocation
- Ignore resident status differences

### 8. Performance

**DO:**
- Use integer operations where possible (faster than float)
- Minimize array iterations
- Reuse local variables
- Cache calculated values
- Use FIRST type when only one column will match

**DON'T:**
- Recalculate same values multiple times
- Use UNIQUE type unnecessarily (slower)
- Create deeply nested loops
- Forget to clear large arrays after use

### 9. Maintenance

**DO:**
- Version control all XML files
- Track tax rate changes by year
- Use constants for values that change annually
- Document assumptions and limitations
- Create regression tests for bug fixes

**DON'T:**
- Modify production rules without testing
- Remove old test cases
- Change table numbers after deployment
- Skip impact analysis for changes

### 10. Error Handling

**DO:**
- Validate input data (state codes, dates, amounts)
- Handle missing or null values gracefully
- Log errors to audit trail
- Use sensible defaults
- Test with invalid inputs

**DON'T:**
- Assume all input is valid
- Allow negative tax calculations
- Skip null checks
- Fail silently on errors

---

## Summary

The DTRules state tax system provides a robust, extensible framework for implementing state income tax calculations. Key takeaways:

1. **Consistent Structure**: All state implementations follow the same entity/table/test pattern
2. **Stack-Based Execution**: Postfix notation provides precise control over calculations
3. **Multi-State Support**: Built-in allocation for part-year residents and multi-state workers
4. **Audit Trail**: All calculations are logged for transparency and debugging
5. **Extensibility**: Adding new states follows a clear, repeatable process

**Reference Implementations:**
- **Flat Tax**: Illinois (TABLE 41100)
- **Progressive Tax**: New Hampshire (TABLE 42700), Montana (TABLE 44900)
- **Multi-State**: Allocation (TABLE 45000), Dispatcher (TABLE 40000)

**Next Steps:**
1. Review existing state implementations
2. Study postfix notation examples
3. Create test cases for your state
4. Implement decision table
5. Validate against official tax forms

For questions or additional examples, refer to:
- `docs/EL-REFERENCE.md` - Expression Language reference
- `docs/ARCHITECTURE.md` - System architecture
- `sampleprojects/TaxReturn/docs/MULTI_STATE_ALLOCATION.md` - Multi-state details

---

**Document Version:** 1.0
**Last Updated:** 2026-03-23
**Compatible with:** DTRules 5.0-SNAPSHOT

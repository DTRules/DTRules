# Decision Table XML Format

This document specifies the required XML format for decision tables in DTRules.

## Format Requirements

**CRITICAL: All new projects MUST use the EL (Expression Language) format.**

The EL format is the required standard for all new decision tables. The legacy postfix format is deprecated and should not be used for new work.

---

## Quick Comparison

| Feature | EL Format (REQUIRED) | Legacy Format (DEPRECATED) |
|---------|---------------------|---------------------------|
| Table name | `<decision_table name="...">` attribute | `<table_name>...</table_name>` child element |
| Conditions | `<expression>...</expression>` | `<condition_postfix>...</condition_postfix>` |
| Actions | Inline EL in `<action>` tags | `<action_postfix>...</action_postfix>` |
| Validation | Compile-time by EL compiler | Runtime only |
| Readability | Human-readable, business-friendly | Stack-based, developer-only |
| Maintenance | Easy to modify | Error-prone |

---

## EL Format (REQUIRED)

The EL format uses the `name` attribute on the `<decision_table>` element and `<expression>` tags for conditions. Actions use inline EL syntax.

### Structure

```xml
<decision_tables name="Project_Tables">

  <decision_table name="Table_Name" number="1000">
    <description>
      Human-readable description of what this table does.
    </description>

    <conditions>
      <condition name="condition_name">
        <expression>taxpayer.income > 50000</expression>
        <comment>Income exceeds threshold</comment>
      </condition>
    </conditions>

    <rules>
      <rule number="1">
        <conditions>
          <condition_name>Y</condition_name>
        </conditions>
        <actions>
          <action>
            result.eligible = true;
            result.tax = income * rate;
          </action>
        </actions>
        <policy>
          Business rationale for this rule.
        </policy>
      </rule>
    </rules>
  </decision_table>

</decision_tables>
```

### Complete Example

```xml
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables name="TaxCalculation_Tables">

  <decision_table name="Calculate_Net_Receipts" number="2000">
    <description>
      Calculate net receipts from gross receipts minus returns and allowances.
      Form 1120 line 1c = 1a - 1b
    </description>

    <conditions>
      <condition name="has_gross_receipts">
        <expression>revenue.gross_receipts > 0</expression>
        <comment>Has gross receipts from sales</comment>
      </condition>
    </conditions>

    <rules>
      <rule number="1">
        <conditions>
          <has_gross_receipts>Y</has_gross_receipts>
        </conditions>
        <actions>
          <action>
            revenue.net_receipts = revenue.gross_receipts - revenue.returns_and_allowances;
          </action>
        </actions>
        <policy>
          Net receipts = Gross receipts - Returns and allowances.
          Reference: Form 1120 Instructions, Line 1c
        </policy>
      </rule>

      <rule number="2">
        <conditions>
          <has_gross_receipts>N</has_gross_receipts>
        </conditions>
        <actions>
          <action>
            revenue.net_receipts = 0.0;
          </action>
        </actions>
        <policy>
          No gross receipts = zero net receipts.
        </policy>
      </rule>
    </rules>
  </decision_table>

</decision_tables>
```

### Key Elements

| Element | Description |
|---------|-------------|
| `<decision_table name="..." number="...">` | Table with name attribute and optional number |
| `<description>` | Human-readable description of table purpose |
| `<conditions>` | Container for condition definitions |
| `<condition name="...">` | Named condition with expression and comment |
| `<expression>` | EL condition expression (human-readable) |
| `<rules>` | Container for rule columns |
| `<rule number="...">` | Individual rule (column in the decision table) |
| `<actions>` | Container for action statements |
| `<action>` | EL action statement (uses assignment syntax) |
| `<policy>` | Business rationale or policy reference |

---

## Legacy Format (DEPRECATED)

**DO NOT USE THIS FORMAT FOR NEW PROJECTS.**

The legacy format uses `<table_name>` as a child element and requires hand-coded postfix notation. This format is deprecated because:

- **Error-prone**: Stack-based postfix notation is difficult to write correctly
- **Hard to maintain**: Business users cannot read or modify postfix
- **No compile-time validation**: Syntax errors only discovered at runtime
- **Inconsistent**: Different authors use different patterns

### Legacy Structure (for reference only)

```xml
<!-- DEPRECATED - DO NOT USE FOR NEW PROJECTS -->
<decision_tables>

<decision_table>
<table_name>Calculate_Tax</table_name>
<xls_file>Tax_dt.xls</xls_file>
<attribute_fields>
<Type>FIRST</Type>
<COMMENTS>Description here</COMMENTS>
<TABLE_NUMBER>1000</TABLE_NUMBER>
</attribute_fields>

<contexts></contexts>
<initial_actions></initial_actions>

<conditions>
<condition_details>
<condition_number>1</condition_number>
<condition_comment>Income above threshold</condition_comment>
<condition_description>taxpayer.income > 50000</condition_description>
<condition_postfix>
taxpayer.income 50000 f>
</condition_postfix>
<condition_column column_number="1" column_value="Y"></condition_column>
</condition_details>
</conditions>

<actions>
<action_details>
<action_number>1</action_number>
<action_comment>Calculate tax</action_comment>
<action_postfix>
income rate fmul /result.tax xdef
</action_postfix>
<action_column column_number="1" column_value="X"></action_column>
</action_details>
</actions>

</decision_table>

</decision_tables>
```

---

## The EL Compiler

The EL compiler converts human-readable EL descriptions to postfix notation automatically. This is how the workflow works:

1. **Authors write EL** - Conditions and actions use natural syntax
2. **EL compiler processes** - Generates correct postfix for runtime
3. **Runtime executes postfix** - Engine uses compiled postfix

### Example Transformation

**EL Input (what you write):**
```
taxpayer.filing_status == "SINGLE" AND taxpayer.income > 50000.0
```

**Postfix Output (generated automatically):**
```
taxpayer.filing_status "SINGLE" streq taxpayer.income 50000.0 f> and
```

### Using the Compiler

```go
import "github.com/DTRules/DTRules/pkg/dtrules/compiler/el"

compiler := el.NewCompiler()
postfix, err := compiler.CompileCondition("taxpayer.income > 50000.0")
// postfix: "taxpayer.income 50000.0 f>"
```

---

## Why EL is Required

### Problem: Hand-Coded Postfix

Hand-coding postfix notation leads to:

1. **Stack errors** - Easy to push/pop incorrectly
2. **Type mismatches** - Using wrong operators (e.g., `eq` vs `streq`)
3. **Maintenance burden** - Only developers can read postfix
4. **No early validation** - Errors found at runtime, not compile time

### Solution: EL Descriptions

EL provides:

1. **Natural syntax** - Reads like English/math: `income > 50000`
2. **Compile-time validation** - Errors caught before runtime
3. **Business-friendly** - Analysts can read and verify
4. **Consistent output** - Compiler generates correct postfix every time

---

## Migration from Legacy to EL

To migrate existing legacy decision tables to EL format:

### Step 1: Identify the condition descriptions

In legacy format, look for `<condition_description>`:
```xml
<condition_description>taxpayer.filing_status == "SINGLE"</condition_description>
```

### Step 2: Convert to EL expression

Create the EL format with `<expression>`:
```xml
<condition name="is_single">
  <expression>taxpayer.filing_status == "SINGLE"</expression>
</condition>
```

### Step 3: Convert actions

Legacy action postfix:
```xml
<action_postfix>
income rate fmul /result.tax xdef
</action_postfix>
```

EL action:
```xml
<action>
  result.tax = income * rate;
</action>
```

### Step 4: Update table structure

Change from child element to attribute:
```xml
<!-- Before (legacy) -->
<decision_table>
<table_name>Calculate_Tax</table_name>
...

<!-- After (EL) -->
<decision_table name="Calculate_Tax" number="1000">
...
```

---

## Best Practices

1. **Always use EL format** - Never hand-code postfix for new tables
2. **Use meaningful names** - Condition and table names should be self-documenting
3. **Include descriptions** - Every table should have a `<description>`
4. **Add comments** - Conditions should have `<comment>` explaining intent
5. **Include policy** - Rules should have `<policy>` with business rationale
6. **Number tables** - Use the `number` attribute for ordering and reference

---

## EL Syntax Reference

For complete EL syntax documentation, see:
- [EL Reference](el-reference.md) - Complete syntax, operators, and functions
- [EL Compiler README](../pkg/dtrules/compiler/el/README.md) - Compiler details and workflow

### Quick Syntax Reference

| Operation | EL Syntax | Notes |
|-----------|-----------|-------|
| Equality | `a == b` or `a = b` | Works for strings and numbers |
| Comparison | `a > b`, `a < b`, `a >= b`, `a <= b` | Numeric comparison |
| Logical AND | `a AND b` or `a and b` | Boolean logic |
| Logical OR | `a OR b` or `a or b` | Boolean logic |
| Assignment | `result.field = value` | Set entity field |
| Arithmetic | `a + b`, `a - b`, `a * b`, `a / b` | Standard operators |
| Call table | `perform TableName` | Execute another table |

---

## File Organization

For projects with multiple decision tables, organize files by domain:

```
project/
  xml/
    Project_dt.xml          # Main/core tables
    Project_dt_core.xml     # Core tables (merged)
    states/
      TEMPLATE_dt.xml       # Template for state tables
      CO_dt.xml             # Colorado-specific
      CA_dt.xml             # California-specific
```

Use merge scripts to combine files for testing while keeping source files modular.

---

## See Also

- [EL Reference](el-reference.md) - Complete EL syntax documentation
- [EL Compiler](../pkg/dtrules/compiler/el/README.md) - Compiler implementation
- [Architecture Guide](architecture.md) - System design overview
- [Sample Projects](../sampleprojects/README.md) - Working examples

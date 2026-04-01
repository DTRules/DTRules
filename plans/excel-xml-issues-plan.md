# Plan: Addressing Excel/XML Issues #395-399

## Summary

Issues #395-399 reveal a **documentation and discoverability gap**, not missing features. DTRules already has the execution model, sync commands, and context capabilities these users need - they just can't find them.

## Issue Classification

| Issue | Title | Type | Action |
|-------|-------|------|--------|
| #395 | Control table for execution order | User Error | Document table-calling patterns |
| #396 | Context iteration for entity arrays | User Error | Document context statements |
| #397 | Metadata preservation for round-trip | Partial Gap | Investigate + document what exists |
| #398 | CLI sync commands | User Error | Already exists - improve discoverability |
| #399 | Sheet order match TABLE_NUMBER | Enhancement | Implement in exporter |

---

## Issue #395: Control Table for Execution Order

**User Request:** Add a control table to define decision table execution order.

**Reality:** DTRules tables call other tables directly. There is no need for external orchestration.

### Response to User

```markdown
Thanks for the detailed issue. DTRules already supports this through table-to-table calls.

**How execution order works in DTRules:**

1. **Entry point table**: Your application calls one table (e.g., `Compute_Tax_Return`)
2. **Tables call tables**: That table uses `perform` to call other tables
3. **Order is defined by the calling table**: The sequence is in your rule logic, not metadata

**Example from TaxReturn:**
```
// In Compute_Tax_Return actions:
perform Calculate_Gross_Income;
perform Calculate_Deductions;
perform Calculate_Tax_Liability;
```

**File/directory ordering:**
- Files use numeric prefixes: `001_Compute_Tax_Return.xlsx`, `010_Calculate_Deductions.xlsx`
- TABLE_NUMBER orders tables within a single file
- This is for human readability, not runtime execution

**What you may actually need:**
If you want to see execution flow, run with tracing enabled - it shows which tables called which.

See `dtrules docs decision-tables` for more on table calls.
```

### Documentation to Add

Add to `dtrules docs decision-tables`:

```
## Table Execution Flow

DTRules uses a **call graph** model, not a sequential list:

1. Your application calls an entry point table
2. That table's actions can call other tables using `perform`
3. Called tables can call more tables
4. Execution follows the call graph

### Example: Tax Return Flow

Entry point: `Compute_Tax_Return`

```
Compute_Tax_Return
├── Calculate_Gross_Income
│   ├── Process_W2_Income
│   └── Process_Self_Employment
├── Calculate_Deductions
│   ├── Calculate_Standard_Deduction
│   └── Calculate_Itemized_Deductions
└── Calculate_Tax_Liability
    └── Apply_Tax_Brackets
```

### File Organization Convention

Use numeric prefixes for human readability:
- `001_Compute_Tax_Return.xlsx` (entry point)
- `010_Calculate_Gross_Income.xlsx`
- `020_Calculate_Deductions.xlsx`

TABLE_NUMBER within files follows the same pattern.
```

**Close issue as:** Won't Fix (Working as Designed) - with helpful explanation

---

## Issue #396: Context Iteration for Entity Arrays

**User Request:** Add support for iterating over entity arrays in decision tables.

**Reality:** DTRules has context statements for this. User needs documentation.

### Response to User

```markdown
DTRules already supports this through **context statements**.

**How to iterate over entities:**

In your decision table's Contexts row (row 6 in Excel), specify:
```
Contexts: staking_account
```

This means: "Execute this table once for each `staking_account` entity."

**Setting up the iteration:**

1. Define an array field in your EDD:
   ```xml
   <field name="staking_accounts" type="array" subtype="staking_account"/>
   ```

2. In your calling table, use `forall` to iterate:
   ```
   forall staking_accounts perform Calculate_Rewards
   ```

3. Inside `Calculate_Rewards`, use context to access current item:
   ```
   Contexts: staking_account
   ```
   Now `staking_account.balance` refers to the current iteration's account.

**Example pattern:**
```
// Parent table action:
forall recipient.accounts perform Process_Account

// Process_Account contexts row:
Contexts: account

// Process_Account can now use:
account.balance > 0    // current account in iteration
```

See `dtrules docs el` for `forall` syntax.
See `dtrules docs decision-tables` for context statements.
```

### Documentation to Add

Add to `dtrules docs decision-tables`:

```
## Context Statements and Iteration

### Processing Entity Arrays

To process each item in an array, use `forall` with `perform`:

**Step 1: Define the array in EDD**
```xml
<entity name="portfolio">
    <field name="accounts" type="array" subtype="account"/>
</entity>
```

**Step 2: Iterate in calling table**
```
// Action in parent table:
forall portfolio.accounts perform Process_Account
```

**Step 3: Set context in called table**
```
Contexts: account
```

Now `Process_Account` executes once per account, with `account` referring to the current item.

### Context Row in Excel

Row 6 of decision tables specifies contexts:

| Row | Content |
|-----|---------|
| 1 | Decision Table |
| 2 | Name: Process_Account |
| 3 | Type: ALL |
| 4 | Number: 3100 |
| 5 | Purpose: Calculate rewards for single account |
| 6 | Contexts: account |
| 7 | Conditions: |

### Multiple Contexts

You can specify multiple contexts:
```
Contexts: account, period
```

This is useful when iterating over nested structures.
```

Add to `dtrules docs el`:

```
## Iteration Operators

### forall

Iterates over an array, executing an action for each element:

```
forall <array> <action>
```

**Examples:**
```
forall accounts perform Validate_Account
forall line_items set total = total + item.amount
forall dependents increment dependent_count
```

### The Context Connection

When using `forall` with `perform`, the called table should declare
a context matching the array element type. This binds the current iteration
item to that context name.
```

**Close issue as:** Won't Fix (Working as Designed) - with helpful explanation

---

## Issue #397: Metadata Preservation for Round-Trip

**User Request:** Preserve metadata during Excel ↔ XML round-trips.

**Status:** Needs investigation - may have legitimate gaps.

### Investigation Needed

1. What metadata IS preserved today?
   - `xls_file` attribute on entities and tables
   - TABLE_NUMBER
   - Comments in XML

2. What metadata is LOST?
   - Excel cell formatting (colors, fonts)
   - Excel comments/notes
   - Column widths
   - Hidden columns

3. What SHOULD be preserved?
   - Structural metadata (required for correct round-trip)
   - vs. Presentation metadata (nice-to-have)

### Action Items

1. **Audit current preservation:**
   ```bash
   # Export XML to Excel, then import back
   dtrules sync export
   dtrules sync import
   # Compare original XML to re-imported XML
   diff original.xml reimported.xml
   ```

2. **Document what's preserved** in `dtrules docs workflow`

3. **If gaps exist:** Create focused issues for specific metadata types

### Response to User (Preliminary)

```markdown
Thanks for raising this. Let me clarify what's currently preserved:

**Preserved in round-trip:**
- Table names, numbers, types
- All conditions and actions (EL expressions)
- Entity definitions and field metadata
- `xls_file` source tracking
- XML comments (in XML → Excel direction)

**Not preserved (by design):**
- Excel cell formatting (colors, fonts, borders)
- Excel comments/notes
- Column widths and row heights

**Why formatting isn't preserved:**
DTRules treats Excel as a data entry format, not a presentation format. The authoritative content is the logic, not the styling.

**What you may actually need:**
If you're losing actual rule content (not just formatting), that's a bug. Can you provide a specific example where content is lost?
```

**Keep issue open** pending investigation

---

## Issue #398: CLI Sync Commands

**User Request:** Add `dtrules export`, `dtrules import`, `dtrules sync` commands.

**Reality:** These already exist.

### Response to User

```markdown
Great news - these commands already exist!

```bash
# Show sync status of all files
dtrules sync status

# Check for pending user edits (useful in scripts)
dtrules sync check

# Import Excel files to XML
dtrules sync import

# Export XML files to Excel
dtrules sync export

# Auto-detect direction and sync
dtrules sync auto
```

**Full documentation:**
```bash
dtrules help
dtrules docs workflow
```

**Example workflow:**
```bash
# After editing Excel files:
dtrules sync import

# After AI/developer edits XML:
dtrules sync check    # verify no Excel conflicts
dtrules sync export   # update Excel
dtrules sync import   # recompile EL to postfix
```

I'll improve discoverability so future users find these more easily.
```

### Discoverability Improvements

1. **Add to README.md:**
   ```markdown
   ## Quick Start

   ```bash
   # See available commands
   dtrules help

   # Sync Excel and XML files
   dtrules sync status
   dtrules sync import
   dtrules sync export
   ```
   ```

2. **Improve `dtrules help` output** - make sync commands more prominent

3. **Add `dtrules` (no args) friendly output** - show quick command summary

**Close issue as:** Duplicate/Already Exists - with helpful pointer to docs

---

## Issue #399: Sheet Order Match TABLE_NUMBER

**User Request:** Excel sheet tabs should be ordered by TABLE_NUMBER.

**Status:** Legitimate enhancement for the exporter.

### Analysis

When exporting XML to Excel (multi-table workbook), sheets are currently created in:
- Alphabetical order?
- XML parse order?
- Arbitrary order?

User wants sheets ordered by TABLE_NUMBER so visual navigation matches execution flow.

### Implementation Plan

1. **Modify exporter** (`pkg/dtrules/excel/` or wherever export lives):
   - After generating sheets, reorder by TABLE_NUMBER
   - Use excelize's sheet ordering API

2. **Add validation warning** (optional):
   - When importing, warn if sheet order doesn't match TABLE_NUMBER order
   - "Warning: Sheet 'Calculate_Tax' (TABLE_NUMBER 2000) appears before 'Calculate_Income' (TABLE_NUMBER 1000)"

### Code Location

Check where Excel export happens:
```
pkg/dtrules/excel/  - likely has exporter
cmd/dtrules/        - CLI commands
```

### Response to User

```markdown
Good suggestion. When exporting to Excel, sheets should be ordered by TABLE_NUMBER for easier navigation.

I'll implement this in the exporter so sheets appear in logical order.

As a workaround until this is implemented, you can manually reorder sheets in Excel (drag tabs).
```

**Keep issue open** - implement enhancement

---

## Implementation Order

### Phase 1: Documentation (Immediate)

1. Add table-calling patterns to `dtrules docs decision-tables`
2. Add context/iteration docs to `dtrules docs el`
3. Add sync quick-start to README
4. Improve `dtrules help` output

### Phase 2: Issue Responses (After docs updated)

1. Close #395 with explanation + link to new docs
2. Close #396 with explanation + link to new docs
3. Close #398 with pointer to existing commands
4. Comment on #397 requesting specific examples
5. Acknowledge #399 as valid enhancement

### Phase 3: Implementation

1. Implement #399 (sheet ordering by TABLE_NUMBER)
2. Investigate #397 (metadata preservation audit)

---

## Documentation Locations

| Topic | File | Command |
|-------|------|---------|
| Table execution flow | `cmd/dtrules/docs/decision-tables.go` | `dtrules docs decision-tables` |
| Context statements | `cmd/dtrules/docs/decision-tables.go` | `dtrules docs decision-tables` |
| forall operator | `cmd/dtrules/docs/el.go` | `dtrules docs el` |
| Sync workflow | `cmd/dtrules/docs/workflow.go` | `dtrules docs workflow` |
| Quick start | `README.md` | - |

---

## Success Criteria

1. Running `dtrules help` clearly shows sync commands exist
2. `dtrules docs decision-tables` explains table-to-table calls
3. `dtrules docs el` documents `forall` and iteration patterns
4. Issues #395, #396, #398 closed with helpful explanations
5. Issue #399 implemented (sheets ordered by TABLE_NUMBER)
6. Issue #397 investigated with clear documentation of what's preserved

# State Tax Implementation Files

This directory contains **separate XML files for each state's tax implementation**.

## Purpose

Each state gets 2 files to avoid merge conflicts during parallel development:
- `XX_edd.xml` - State-specific constants (tax rates, deductions, brackets, etc.)
- `XX_dt.xml` - State-specific decision tables (tax calculation logic)

Where `XX` is the 2-letter state code (CO, CA, NY, TX, etc.)

## How a state is authored

These are generated rule files, like every file under `xml/`. Do not edit
them by hand and do not copy the templates: the authoring API writes the XML,
compiles the postfix and updates the paired workbook in one operation, and
`dtrules verify` rejects XML that has no Excel behind it. See
[docs/authoring-contract.md](../../../../docs/authoring-contract.md).

```bash
cd sampleprojects/TaxReturn

# constants
echo '{"op":"add-field","entity":"result","field":{"name":"co_tax_rate","type":"double","default":"0.044","comment":"CO flat rate 4.4% (2025)"}}' \
  | dtrules edd patch --edd-file states/CO_edd.xml --project .

# the table (--range and --reason are required when the file is new)
dtrules table put CO_Tax --file states/CO_dt.xml --range 40600-40699 \
  --reason "Colorado tax; own file to avoid merge conflicts" --project . < co_tax.json

dtrules verify .
```

There is no merge step. The loader reads every `*_dt.xml` and `*_edd.xml`
under `xml/` directly, so a state file is part of the project the moment it
exists, and `TaxReturn_dt.xml` never carries a copy of it.

Every `XX_Tax` table reads `result.state_calc_agi` — the AGI of the roster
entry being computed — and writes `result.computed_state_taxable_income` and
`result.computed_state_tax`, which `Compute_Roster_State_Tax` harvests onto
the `state_tax_result`. A table that writes only `result.xx_state_tax` leaves
the roster at zero.

## File Naming Convention

- **State codes**: Use 2-letter postal codes (USPS standard)
  - Colorado: `CO_edd.xml`, `CO_dt.xml`
  - California: `CA_edd.xml`, `CA_dt.xml`
  - New York: `NY_edd.xml`, `NY_dt.xml`

- **Templates**: `TEMPLATE_edd.xml`, `TEMPLATE_dt.xml`
  - Copy these to create new state implementations
  - Never modify the templates directly

## Table Numbering Convention

State tax decision tables use table numbers in the 40000-49999 range:

- **Format**: `4[state_number][00-99]`
- **state_number**: Alphabetical order of state (01=AL, 02=AK, etc.)

Examples:
- Alabama (AL, #1): 40100-40199
- Alaska (AK, #2): 40200-40299
- California (CA, #5): 40500-40599
- Colorado (CO, #6): 40600-40699
- New York (NY, #33): 43300-43399

See `TEMPLATE_dt.xml` for details.

## Benefits of Separate Files

1. **Zero merge conflicts**: States don't modify the same files
2. **Parallel development**: 41 states can be implemented simultaneously
3. **Modularity**: Each state is self-contained
4. **Easier review**: PRs show only one state's changes
5. **Simpler debugging**: State-specific issues isolated to state files

## See Also

- Authoring contract: `../../../../docs/authoring-contract.md`
- Claude Code instructions: `../../../../.claude/CLAUDE.md`

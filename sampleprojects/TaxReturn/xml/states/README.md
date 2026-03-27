# State Tax Implementation Files

This directory contains **separate XML files for each state's tax implementation**.

## Purpose

Each state gets 2 files to avoid merge conflicts during parallel development:
- `XX_edd.xml` - State-specific constants (tax rates, deductions, brackets, etc.)
- `XX_dt.xml` - State-specific decision tables (tax calculation logic)

Where `XX` is the 2-letter state code (CO, CA, NY, TX, etc.)

## Quick Start

### 1. Copy Templates

```bash
cd sampleprojects/TaxReturn/xml/states
cp TEMPLATE_edd.xml CO_edd.xml
cp TEMPLATE_dt.xml CO_dt.xml
```

### 2. Edit Your Files

Edit `CO_edd.xml` to add Colorado-specific constants:
```xml
<entity name="result">
  <field name="co_tax_rate" type="double" default_value="0.044"
         comment="CO flat rate 4.4% (2025)"/>
</entity>
```

Edit `CO_dt.xml` to implement Colorado tax calculation logic.

### 3. Merge and Test

The merge script combines all state files into the main XML files:

```bash
cd ../..  # Back to TaxReturn directory
./scripts/merge-states.sh
cd ../../go
go test ./pkg/dtrules/... -run TestTaxReturn
```

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

## Build Process

The build process works as follows:

1. **Development**: Edit separate state files (`XX_edd.xml`, `XX_dt.xml`)
2. **Merge**: Run `scripts/merge-states.sh` to combine all files
3. **Generated**: Creates `TaxReturn_edd.xml` and `TaxReturn_dt.xml`
4. **Testing**: Tests run against the merged files
5. **Git**: Commit ONLY your state files, NOT the merged files

## Benefits of Separate Files

1. **Zero merge conflicts**: States don't modify the same files
2. **Parallel development**: 41 states can be implemented simultaneously
3. **Modularity**: Each state is self-contained
4. **Easier review**: PRs show only one state's changes
5. **Simpler debugging**: State-specific issues isolated to state files

## See Also

- Main documentation: `../../ARCHITECTURE_REFACTOR.md`
- Merge script: `../../scripts/merge-states.sh`
- Templates: `TEMPLATE_edd.xml`, `TEMPLATE_dt.xml`
- Claude Code instructions: `../../../../.claude/CLAUDE.md`

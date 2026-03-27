# State Tax Architecture Refactor

## Problem

Current architecture causes merge conflicts when implementing multiple states in parallel:
- All 41 states add constants to single file: `TaxReturn_edd.xml`
- All 41 states add decision tables to single file: `TaxReturn_dt.xml`
- **Result**: Parallel development impossible, 30+ issues blocked by merge conflicts

## Design Principle

**States are independent.** Updates to Colorado should not conflict with updates to California.

## Solution: Separate Files Per State

### Directory Structure

```
sampleprojects/TaxReturn/xml/
├── TaxReturn_edd.xml              # Core entities (job, taxpayer, result, etc.)
├── TaxReturn_dt.xml               # Core decision tables (federal tax, dispatcher)
├── states/
│   ├── CO_edd.xml                 # Colorado-specific constants
│   ├── CO_dt.xml                  # Colorado decision tables
│   ├── CA_edd.xml                 # California-specific constants
│   ├── CA_dt.xml                  # California decision tables
│   ├── IL_edd.xml                 # Illinois-specific constants
│   ├── IL_dt.xml                  # Illinois decision tables
│   └── ...                        # One pair per state
```

### File Contents

#### Core File: `TaxReturn_edd.xml`
```xml
<entity_data_dictionary version="1.0">
  <!-- Core entities used by all states -->
  <entity name="job">
    <field name="state" type="string" .../>
    <field name="filing_status" type="string" .../>
    <!-- Federal fields, no state-specific constants -->
  </entity>

  <entity name="result">
    <field name="agi" type="double" .../>
    <field name="federal_tax" type="double" .../>
    <!-- Federal results, no state-specific fields -->
  </entity>

  <entity name="state_tax_result">
    <!-- Shared state result structure -->
    <field name="state_code" type="string" .../>
    <field name="state_agi" type="double" .../>
    <field name="state_tax_liability" type="double" .../>
  </entity>
</entity_data_dictionary>
```

#### State File: `states/CO_edd.xml`
```xml
<entity_data_dictionary version="1.0">
  <!-- Colorado-specific constants only -->
  <entity name="result">
    <field name="co_tax_rate" type="double" default_value="0.044"
           comment="CO flat rate 4.4% (2025)"/>
    <field name="co_standard_deduction_single" type="double" default_value="15000"
           comment="CO standard deduction single (2025)"/>
    <field name="co_standard_deduction_joint" type="double" default_value="30000"
           comment="CO standard deduction married filing jointly (2025)"/>
  </entity>
</entity_data_dictionary>
```

#### State File: `states/CO_dt.xml`
```xml
<decision_tables>
  <decision_table name="Calculate_CO_Tax" table_number="41000">
    <!-- Colorado tax calculation logic -->
  </decision_table>
</decision_tables>
```

#### State File: `states/CA_edd.xml`
```xml
<entity_data_dictionary version="1.0">
  <!-- California-specific constants only -->
  <entity name="result">
    <field name="ca_conformity_date" type="string" default_value="2025-01-01"
           comment="CA IRC conformity date per SB 711"/>
    <field name="ca_military_retirement_exclusion" type="double" default_value="20000"
           comment="CA military retirement exclusion (AB 1786, 2025-2029)"/>
    <!-- CA tax brackets -->
    <field name="ca_bracket_1_limit" type="double" default_value="10412" .../>
    <field name="ca_bracket_1_rate" type="double" default_value="0.01" .../>
    <!-- ... -->
  </entity>
</entity_data_dictionary>
```

### Loader Implementation

#### Option A: Multi-file Loader (New Feature)

Enhance `EDDLoader` and `DTLoader` in Go to support directory loading:

```go
// Load EDD from directory (merges all files)
func LoadEDDDirectory(dir string) (*EDDFile, error) {
    // 1. Load core TaxReturn_edd.xml
    core, err := LoadEDD(filepath.Join(dir, "TaxReturn_edd.xml"))

    // 2. Load all states/*_edd.xml
    stateFiles, _ := filepath.Glob(filepath.Join(dir, "states", "*_edd.xml"))
    for _, stateFile := range stateFiles {
        state, err := LoadEDD(stateFile)
        // 3. Merge state entities into core
        core.MergeEntities(state)
    }

    return core, nil
}

// MergeEntities adds fields from source entities into target
func (edd *EDDFile) MergeEntities(source *EDDFile) error {
    for _, srcEntity := range source.Entities {
        // Find matching entity in target
        targetEntity := edd.FindEntity(srcEntity.Name)
        if targetEntity == nil {
            return fmt.Errorf("entity %s not found in core EDD", srcEntity.Name)
        }

        // Add all fields from source to target
        for _, field := range srcEntity.Fields {
            // Check for duplicates
            if targetEntity.HasField(field.Name) {
                return fmt.Errorf("duplicate field %s in entity %s", field.Name, srcEntity.Name)
            }
            targetEntity.Fields = append(targetEntity.Fields, field)
        }
    }
    return nil
}
```

Similar logic for Decision Tables.

#### Option B: Build-time Merge (Simpler)

Create a preprocessor script that merges files before loading:

```bash
#!/bin/bash
# merge-states.sh

# Merge all state EDD files into TaxReturn_edd.xml
cat TaxReturn_edd_core.xml > TaxReturn_edd.xml
for state in states/*_edd.xml; do
    # Extract entity content and append
    xmlstarlet sel -t -c "//entity" "$state" >> TaxReturn_edd.xml
done
echo "</entity_data_dictionary>" >> TaxReturn_edd.xml

# Merge all state DT files
cat TaxReturn_dt_core.xml > TaxReturn_dt.xml
for state in states/*_dt.xml; do
    xmlstarlet sel -t -c "//decision_table" "$state" >> TaxReturn_dt.xml
done
echo "</decision_tables>" >> TaxReturn_dt.xml
```

Run before tests: `make merge-states && go test ./...`

## Implementation Plan

### Phase 1: Create State Directory Structure (Foundation Issue)

1. Create `sampleprojects/TaxReturn/xml/states/` directory
2. Extract existing state constants from `TaxReturn_edd.xml` into state files
3. Extract existing state tables from `TaxReturn_dt.xml` into state files
4. Keep original files as backup during transition

**States already implemented** (extract these first):
- CO, IL, IN, MI, NC, PA (flat tax states)
- NY, MA, CT, NJ, VT, ME, RI, NH, OR, HI, ID, WI, IA, KS, MO, ND, KY, OH, MT, GA (progressive states)

### Phase 2: Implement Loader Support

**Option A**: Implement multi-file loading in Go loaders
- Modify `go/pkg/dtrules/loader/edd.go`
- Modify `go/pkg/dtrules/loader/dt.go`
- Add directory scanning and merging logic
- Update test harness to use directory loading

**Option B**: Implement build-time merge script
- Create `sampleprojects/TaxReturn/scripts/merge-states.sh`
- Update `go/pkg/dtrules/Makefile` to run merge before tests
- Keep separate files in git, generate merged files on build

### Phase 3: Update Remaining Issues

Once refactor is complete:
1. Resume orchestrator on remaining ~32 issues
2. Each new state creates just 2 files: `XX_edd.xml` and `XX_dt.xml`
3. **Zero merge conflicts** between states
4. Truly parallel development

## Benefits

1. **Zero conflicts**: States don't modify same files
2. **Parallel development**: 41 states can be developed simultaneously
3. **Modularity**: Each state is self-contained
4. **Easier review**: PR shows only one state's changes
5. **Simpler debugging**: State-specific issues isolated to state files
6. **Scalability**: Adding new states doesn't touch existing states

## Migration Path

### Backward Compatibility

Keep merged `TaxReturn_edd.xml` and `TaxReturn_dt.xml` generated as build artifacts:
- Git ignores merged files
- Separate state files are source of truth
- Build process merges them
- Tests run against merged files
- Old code paths still work

### Testing

1. Extract existing states to separate files
2. Merge them back
3. Run all existing tests
4. Verify output identical to before refactor

## Estimated Effort

- **Phase 1** (Extract existing states): 4-6 hours
  - Script to extract state constants/tables
  - Create 30 state file pairs
  - Validate XML well-formed

- **Phase 2** (Loader support): 3-5 hours
  - Option A: Implement multi-file loading
  - Option B: Build-time merge script
  - Update test harness
  - Validate all tests pass

- **Phase 3** (Resume development): Immediate
  - No changes needed to remaining issues
  - Just create separate files instead of editing shared files

**Total**: 1-2 days to unblock 32 issues and enable parallel development

## Decision

**Recommend Option B (Build-time merge)** for fastest implementation:
- Simpler to implement (shell script vs Go code changes)
- No loader changes required
- Works with existing test infrastructure
- Can migrate to Option A later if needed

## Next Steps

1. Create foundation issue for this refactor
2. Implement extraction script for existing 30 states
3. Implement merge script for build process
4. Validate all tests pass
5. Resume orchestrator on remaining 32 issues

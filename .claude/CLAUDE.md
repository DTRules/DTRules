# DTRules Project - Claude Code Instructions

## CRITICAL: Output Redirection

**ALWAYS redirect command output to log files**. Long-running commands and verbose output will crash the AI context.

### Required for ALL commands:

```bash
# Maven builds
mvn clean install > /tmp/maven-build.log 2>&1

# Go builds
cd go && go build ./... > /tmp/go-build.log 2>&1

# Tests
go test ./... > /tmp/go-test.log 2>&1

# Git operations (these are OK without redirect)
git status
git add <files>
git commit -m "message"
git push origin <branch> > /tmp/git-push.log 2>&1
```

### Check logs with tail:
```bash
tail -50 /tmp/maven-build.log
tail -50 /tmp/go-test.log
```

**NEVER run builds or tests without redirecting output.**

## State Tax Implementation

### CRITICAL: Use Separate Files Per State

**Each state gets 2 files** to avoid merge conflicts:
- `sampleprojects/TaxReturn/xml/states/XX_edd.xml` (state constants)
- `sampleprojects/TaxReturn/xml/states/XX_dt.xml` (state decision tables)

Where XX is the 2-letter state code (CO, CA, NY, etc.)

**Do NOT edit** the main `TaxReturn_edd.xml` or `TaxReturn_dt.xml` files directly!

### File Templates

Copy these templates to create your state files:
- `sampleprojects/TaxReturn/xml/states/TEMPLATE_edd.xml`
- `sampleprojects/TaxReturn/xml/states/TEMPLATE_dt.xml`

Example:
```bash
cp sampleprojects/TaxReturn/xml/states/TEMPLATE_edd.xml sampleprojects/TaxReturn/xml/states/CO_edd.xml
cp sampleprojects/TaxReturn/xml/states/TEMPLATE_dt.xml sampleprojects/TaxReturn/xml/states/CO_dt.xml
```

### State Implementation Pattern

1. **Research ONLY what you need**:
   - Tax rate(s)
   - Standard deduction amounts
   - Exemption amounts
   - 2-3 key state-specific rules
   - Don't research everything upfront!

2. **Create state files from templates**:
   ```bash
   cp sampleprojects/TaxReturn/xml/states/TEMPLATE_edd.xml sampleprojects/TaxReturn/xml/states/XX_edd.xml
   cp sampleprojects/TaxReturn/xml/states/TEMPLATE_dt.xml sampleprojects/TaxReturn/xml/states/XX_dt.xml
   ```

3. **Add constants to `XX_edd.xml`**:
   ```xml
   <entity name="result">
     <field name='co_tax_rate' type='double' default_value='0.044'
            comment='CO flat rate 4.4% (2025)'/>
     <field name='co_standard_deduction_single' type='double' default_value='15000'
            comment='CO standard deduction single (2025)'/>
   </entity>
   ```

4. **Create decision table in `XX_dt.xml`**:
   - Use TEMPLATE_dt.xml as starting point
   - Replace XX with your state code
   - Assign unique table number (see template for numbering convention)
   - Keep it simple initially, add complexity incrementally

5. **Merge files before testing**:
   ```bash
   cd sampleprojects/TaxReturn
   ./scripts/merge-states.sh
   ```

6. **Build and test**:
   ```bash
   cd go
   go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
   tail -50 /tmp/test.log
   ```

7. **Create 3 test cases** in `testfiles/TestScenarios/State/XX/`

### Git Workflow for States

```bash
# Create your state files
cp xml/states/TEMPLATE_edd.xml xml/states/CO_edd.xml
cp xml/states/TEMPLATE_dt.xml xml/states/CO_dt.xml

# Edit the files (add your state's logic)
# ... edit CO_edd.xml and CO_dt.xml ...

# Merge and test
./scripts/merge-states.sh
cd go && go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
tail -50 /tmp/test.log

# Stage ONLY your state files (not the merged files!)
git add xml/states/CO_edd.xml xml/states/CO_dt.xml

# Commit
git commit -m "feat: implement CO state tax (#180)"

# Push
git push origin feature/issue-180 > /tmp/git-push.log 2>&1
```

### Why Separate Files?

**Problem**: All 41 states modifying `TaxReturn_edd.xml` and `TaxReturn_dt.xml` causes guaranteed merge conflicts.

**Solution**: Each state gets its own files. The merge script combines them for testing. States are truly independent - Colorado changes don't conflict with California changes.

### Avoiding Context Overflow

**Do NOT**:
- Research entire state tax code upfront
- Read all IRS publications
- Try to understand everything before starting
- Run builds/tests without redirecting output

**DO**:
- Research incrementally (rate → deductions → special rules)
- Start simple, add complexity
- Test frequently with redirected output
- Ask questions if stuck

### Complex States (CA, NY, MA)

These have extensive regulatory detail. **Work incrementally**:

1. First pass: Basic rate and standard deduction only
2. Second pass: Add one special rule
3. Third pass: Add more complexity
4. Don't try to do everything at once!

## Build Commands

```bash
# Full build (REDIRECT OUTPUT!)
mvn clean install > /tmp/maven-build.log 2>&1

# Go build only
cd go && go build ./... > /tmp/go-build.log 2>&1

# Go tests
cd go && go test ./... > /tmp/go-test.log 2>&1

# Specific test
cd go && go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
```

## Commit Convention

```bash
git commit -m "feat: implement <State> state income tax (#<issue>)"
git commit -m "fix: correct <State> tax bracket calculation (#<issue>)"
git commit -m "test: add test cases for <State> (#<issue>)"
```

## Git Workflow

```bash
# Stage ONLY your state-specific files
git add sampleprojects/TaxReturn/xml/states/XX_edd.xml
git add sampleprojects/TaxReturn/xml/states/XX_dt.xml
git add sampleprojects/TaxReturn/testfiles/TestScenarios/State/XX/

# Commit
git commit -m "feat: implement XX state tax (#<issue>)"

# Push (redirect for verbose output)
git push origin feature/issue-<N> > /tmp/git-push.log 2>&1
```

**IMPORTANT**: Do NOT commit the merged `TaxReturn_edd.xml` or `TaxReturn_dt.xml` files! These are generated files created by the merge script for testing only.

## When to Ask for Help

- State has unique approach you don't understand
- Can't find official tax rate information
- Tests fail with unclear errors (after checking logs)
- Context filling up despite redirecting output

## Performance

Check logs frequently rather than keeping output in memory:
```bash
tail -30 /tmp/build.log
tail -30 /tmp/test.log
grep -i error /tmp/build.log
```

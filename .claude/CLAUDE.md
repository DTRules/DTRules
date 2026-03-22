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

### File Locations
- **Decision tables**: `sampleprojects/TaxReturn/xml/TaxReturn_dt.xml`
- **Entity definitions**: `sampleprojects/TaxReturn/xml/TaxReturn_edd.xml`
- **Test cases**: `sampleprojects/TaxReturn/testfiles/TestScenarios/State/<STATE>/`
- **Test harness**: `go/pkg/dtrules/taxreturn_results_test.go`

### State Implementation Pattern

1. **Research ONLY what you need**:
   - Tax rate(s)
   - Standard deduction amounts
   - Exemption amounts
   - 2-3 key state-specific rules
   - Don't research everything upfront!

2. **Add constants to EDD** (`TaxReturn_edd.xml`):
   ```xml
   <field name='co_tax_rate' type='double' default_value='0.044' comment='CO flat rate 4.4%'/>
   ```

3. **Create decision table** (`TaxReturn_dt.xml`):
   - Use existing federal tables as examples
   - Keep it simple initially
   - Add complexity incrementally

4. **Add state branch to dispatcher**:
   ```xml
   job.state "CO" streq { Calculate_CO_Tax } if
   ```

5. **Create 3 test cases** minimum

6. **Build and test**:
   ```bash
   cd go
   go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
   tail -50 /tmp/test.log
   ```

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
# Stage specific files (preferred over git add .)
git add sampleprojects/TaxReturn/xml/TaxReturn_dt.xml
git add sampleprojects/TaxReturn/xml/TaxReturn_edd.xml

# Commit
git commit -m "feat: implement CO state tax (#180)"

# Push (redirect for verbose output)
git push origin feature/issue-<N> > /tmp/git-push.log 2>&1
```

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

# DTRules Project - Claude Code Instructions

## Project Structure (Go Primary)

DTRules is now a Go-first project. The structure is:

```
DTRules/
├── cmd/                    # CLI commands
│   └── dtrules/           # Main CLI tool
├── pkg/dtrules/           # Core library
│   ├── sdk/               # Embeddable SDK
│   ├── sync/              # Excel/XML sync
│   └── ...
├── examples/              # Example applications
├── sampleprojects/        # Rule sets
├── legacy/                # Archived code
│   ├── java/              # Original Java implementation
│   └── go/                # ASM-dependent Go code
└── go.mod                 # Module: github.com/DTRules/DTRules
```

## CRITICAL: Output Redirection

**ALWAYS redirect command output to log files**. Long-running commands and verbose output will crash the AI context.

### Required for ALL commands:

```bash
# Go builds (from repo root)
go build ./... > /tmp/go-build.log 2>&1

# Tests
go test ./... > /tmp/go-test.log 2>&1

# Make build
make build > /tmp/make-build.log 2>&1

# Git operations (these are OK without redirect)
git status
git add <files>
git commit -m "message"
git push origin <branch> > /tmp/git-push.log 2>&1
```

### Check logs with tail:
```bash
tail -50 /tmp/go-build.log
tail -50 /tmp/go-test.log
```

**NEVER run builds or tests without redirecting output.**

## Embedded Documentation

The `dtrules` binary includes comprehensive documentation for AI and developers:

```bash
dtrules docs                     # List all topics
dtrules docs xml-format          # XML file format specification
dtrules docs decision-tables     # How to write decision tables
dtrules docs operators           # All operators with examples
dtrules docs sdk                 # Embedding in applications
dtrules docs examples            # Complete working examples
dtrules docs workflow            # Development workflow
```

**Use `dtrules docs` when you need to understand DTRules concepts.**

## Excel/XML Synchronization (CRITICAL)

**Excel is the source of truth.** Run `dtrules build` after every edit — whether you changed Excel or XML. The build command auto-detects which files changed and runs the full pipeline (normalize + compile) so steps cannot be skipped.

```bash
dtrules build                    # Auto-detect and run the full pipeline
dtrules build --from-excel       # Force Excel-authored path
dtrules build --from-xml         # Force XML-authored path
dtrules build --dry-run          # Show what would change without writing
```

See `dtrules docs workflow` for details on both the Excel-authored and XML-authored paths.

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

2. **Create state files from templates**

3. **Add constants to `XX_edd.xml`**

4. **Create decision table in `XX_dt.xml`**

5. **Merge files before testing**:
   ```bash
   cd sampleprojects/TaxReturn
   ./scripts/merge-states.sh
   ```

6. **Build and test**:
   ```bash
   go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
   tail -50 /tmp/test.log
   ```

7. **Create 3 test cases** in `testfiles/TestScenarios/State/XX/`

### Git Workflow for States

```bash
# Create your state files
cp sampleprojects/TaxReturn/xml/states/TEMPLATE_edd.xml sampleprojects/TaxReturn/xml/states/CO_edd.xml
cp sampleprojects/TaxReturn/xml/states/TEMPLATE_dt.xml sampleprojects/TaxReturn/xml/states/CO_dt.xml

# Edit the files (add your state's logic)
# ... edit CO_edd.xml and CO_dt.xml ...

# Merge and test
cd sampleprojects/TaxReturn && ./scripts/merge-states.sh
go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1
tail -50 /tmp/test.log

# Stage ONLY your state files (not the merged files!)
git add sampleprojects/TaxReturn/xml/states/CO_edd.xml sampleprojects/TaxReturn/xml/states/CO_dt.xml

# Commit
git commit -m "feat: implement CO state tax (#180)"

# Push
git push origin feature/issue-180 > /tmp/git-push.log 2>&1
```

**IMPORTANT**: Do NOT commit the merged `TaxReturn_edd.xml` or `TaxReturn_dt.xml` files!

## Build Commands

```bash
# Build CLI (from repo root)
make build > /tmp/make-build.log 2>&1

# Or directly
go build -o build/dtrules ./cmd/dtrules/ > /tmp/go-build.log 2>&1

# Run tests
go test ./... > /tmp/go-test.log 2>&1

# Specific test
go test ./pkg/dtrules/... -run TestTaxReturn > /tmp/test.log 2>&1

# Install
make install > /tmp/make-install.log 2>&1
```

## Commit Convention

```bash
git commit -m "feat: implement <State> state income tax (#<issue>)"
git commit -m "fix: correct <State> tax bracket calculation (#<issue>)"
git commit -m "test: add test cases for <State> (#<issue>)"
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

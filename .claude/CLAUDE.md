# DTRules Project - Claude Code Instructions

## Project Structure (Go Primary)

DTRules is now a Go-first project. The structure is:

```
DTRules/
├── cmd/                    # CLI commands
│   ├── api/                # HTTP API server
│   └── dtrules/            # Main CLI tool
├── pkg/dtrules/            # Core library
│   ├── authoring/          # Typed authoring view + Project API
│   ├── compiler/el/        # ANTLR-based EL → postfix compiler
│   ├── decisiontable/      # Decision-table model + advisory pass
│   ├── interpreter/        # Stack-based VM (Go + amd64 ASM)
│   ├── operators/          # Operator registry
│   ├── runtime/            # Bytecode executors (Go + nativeasm)
│   ├── session/            # Execution context
│   ├── sync/               # Excel/XML sync + validation
│   └── ...
├── sampleprojects/         # Rule sets (TaxReturn, CHIP, ...)
├── scripts/                # Project-level scripts (merge-pr, ...)
├── ui/                     # TypeScript UI surface
├── legacy/
│   └── go/                 # ASM-dependent Go code (archived)
└── go.mod                  # Module: github.com/DTRules/DTRules
```

A `pkg/dtrules/sdk` package for embeddable engine wiring is being
extracted (#757). Until it lands, both CLI binaries (`cmd/dtrules`,
`cmd/api`) glue the engine pipeline together independently.

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

## Authoring Contract (CRITICAL)

**Excel is the system of record for DSL. `postfix` is a compiled artifact, never authored. Every tool that writes XML must write the same DSL back to Excel in the same operation.** Full spec: [docs/authoring-contract.md](../docs/authoring-contract.md).

There are exactly two ways to change a rule:

- **Edit Excel**, then `dtrules build` — extracts DSL to XML and compiles DSL→postfix. Excel is the input; XML is generated.
- **Call the authoring API** (`dtrules table`/`dtrules edd`, MCP write tools) — it writes the XML DSL, compiles postfix, **and** updates Excel in the same operation. If the project has no Excel yet, the API bootstraps it from the XML.

```bash
dtrules build                    # Excel → XML (+ compile); the human path
dtrules build --dry-run          # Show what would change without writing
dtrules table put <name>         # Programmatic edit; updates XML AND Excel
dtrules edd put                  # Programmatic EDD edit; updates XML AND Excel
```

Hard rules:

- **Never hand-edit XML.** It is generated. Edits go through Excel or the authoring API.
- **Never hand-write `postfix`.** It is the compiled output of DSL.
- **`dtrules compile` does not exist** — it was a writer that bypassed Excel and has been removed. DSL→postfix is an internal step of `build` and the API only.

See `dtrules docs workflow` for the build pipeline.

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
# BEFORE DECLARING A TASK COMPLETE -- run this:
make check > /tmp/make-check.log 2>&1 && tail -20 /tmp/make-check.log

# Scoped tests are NOT sufficient. `make check` runs go build ./... (full module)
# followed by go vet and the scoped test suite. A task is only done when check passes.

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

# DTRules Project - Claude Code Instructions

**The system specification is normative: [docs/SPEC.md](../docs/SPEC.md).**

- ALL work is done **against the spec**. Before building or changing
  anything, read the relevant section; if the change contradicts the spec,
  resolve that first (change the spec deliberately, or change the plan).
- **New work requires updating the spec.** A change is not finished until
  SPEC.md describes the system as it now is — same commit or same change
  set, not "later".

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
│   ├── interpreter/        # Stack-based VM (Go)
│   ├── operators/          # Operator registry
│   ├── runtime/            # Bytecode executor interface (goruntime)
│   ├── session/            # Execution context
│   ├── sync/               # Excel/XML sync + validation
│   └── ...
├── sampleprojects/         # Rule sets (TaxReturn, CHIP, ...)
├── scripts/                # merge-pr.sh (tracked); other scripts are local-only
├── ui/                     # TypeScript UI surface
├── legacy/
│   └── go/                 # ASM-dependent Go code (archived)
└── go.mod                  # Module: github.com/DTRules/DTRules
```

There is no SDK package and none is planned. An embedder wires the engine
from `session`, `mapping`/`datafile` and the entry table directly; that is
the supported path, and it is what both CLI binaries (`cmd/dtrules`,
`cmd/api`) and `pkg/dtrules/web` do. A `pkg/dtrules/sdk` wrapper was written
and removed (`69774f70`) because values already arrive through the EDD as
XML — do not propose it again. The path is spelled out in README.md
("Embedding in a Go application"), `dtrules docs embedding`, and §2.11 of
[docs/SPEC.md](../docs/SPEC.md); `pkg/dtrules/embedding_example_test.go`
compiles and runs it.

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

## EL is case-insensitive (CRITICAL)

`Status` and `status` are **one name**, not two that clash. There is no
ambiguity to resolve and nothing to disambiguate — "collision" is a
case-sensitive-language idea and does not apply here. If you catch yourself
reasoning about the *risk* of two spellings meeting, the model is wrong: they
were always the same thing.

Case is for people — readability and project conventions (`Compute_Tax_Return`
for a table, `taxpayer.gross_income` for a field). It means nothing to
execution, and no rule ever behaves differently because of it.

It must still be **preserved**, for two reasons, neither of them correctness:
the author's chosen spelling is theirs to keep, and the XML must round-trip
byte-identically because that is what `dtrules verify` compares.

- Matching a name — regex, map key, comparison — must normalise
  (`strings.EqualFold`, `/i`, lowercased keys).
- Writing a name back out — Excel, XML, a report — must use the **authored**
  spelling, never the interned one. `RDecisionTable.AuthoredName()` and
  `EntityEntry.AuthoredName` exist for exactly this (#1040).
- Do not describe differing case as a conflict in issues, commits or docs. It
  teaches the wrong model to whoever reads next.


## State Tax Implementation

Each state's rules live in their own pair of files under
`sampleprojects/TaxReturn/xml/states/` so that parallel state work never
touches the same file:

- `states/XX_edd.xml` — the state's constants (rates, brackets, deductions)
- `states/XX_dt.xml` — the state's decision tables (`XX_Tax` and helpers)

The loader reads every `*_dt.xml` and `*_edd.xml` under `xml/` directly.
There is no merge step: a state file is part of the project the moment it
exists, and `TaxReturn_dt.xml` never contains a copy of it.

### Authoring a state (through the API — never by hand)

The state files are generated XML like every other rule file. **Do not edit
them by hand and do not copy another state's file.** Author through the API, which
writes the XML, compiles the postfix and updates the paired workbook in one
operation (#1169 made this work for `states/*`):

```bash
cd sampleprojects/TaxReturn

# Constants: add fields to the state's EDD file (creates it if new)
echo '{"op":"add-field","entity":"result","field":{"name":"co_tax_rate","type":"double","default":"0.044","comment":"CO flat rate 4.4% (2025)"}}' \
  | dtrules edd patch --edd-file states/CO_edd.xml --project .

# Rules: create or replace the state's table (--range and --reason when the file is new)
dtrules table put CO_Tax --file states/CO_dt.xml --range 40600-40699 \
  --reason "Colorado tax; own file to avoid merge conflicts" --project . < co_tax.json

# Later edits: patch one cell or one row at a time
echo '{"op":"update-action-dsl","action_number":1,"dsl":"..."}' | dtrules table patch CO_Tax --project .

dtrules table schema        # the JSON shape `table put` expects
dtrules docs decision-tables
```

Table numbers: `4[state_number][00-99]` where state_number is the state's
alphabetical position (AL=01 … WY=50); see `states/README.md`.

Every `XX_Tax` table computes against `result.state_calc_agi` (the AGI of the
roster entry being computed, set by `Compute_Roster_State_Tax`) and writes
its answer to `result.computed_state_taxable_income` and
`result.computed_state_tax`. The dispatcher harvests those onto the
`state_tax_result` roster entry; a table that writes only `result.xx_state_tax`
leaves the roster at zero.

### What to research

Tax rate(s), standard deduction, exemptions, and two or three key
state-specific rules. Cite the state revenue department's publication in the
table or field comment. IRS and state publications first — do not ask the
user tax questions (see the memory on this).

### Test and verify

```bash
go test ./pkg/dtrules/... -run TestTaxReturnScenarioCoverage > /tmp/test.log 2>&1
tail -20 /tmp/test.log
dtrules verify sampleprojects/TaxReturn     # XML must match its Excel source
```

Create three scenarios under `testfiles/TestScenarios/XX/` with the
expected figures derived from the state's own worksheet, never from what the
rules happen to compute.

### Git

Stage the state's XML files **and** the workbook the API updated for them.
`dtrules verify` in CI rejects XML with no Excel behind it.

```bash
git add sampleprojects/TaxReturn/xml/states/CO_edd.xml sampleprojects/TaxReturn/xml/states/CO_dt.xml \
        sampleprojects/TaxReturn/excel/states/CO.xlsx
git commit -m "feat: implement CO state tax (#180)"
```

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

## Merging PRs

**Never run `gh pr merge` directly.** Merge with `scripts/merge-pr.sh`, which
tests the PR squashed onto the current `main` (not the PR alone) and merges
only that tested commit. See §2.10 of [docs/SPEC.md](../docs/SPEC.md).

```bash
scripts/merge-pr.sh 1301 1302 1303      # in order; stops at the first failure
scripts/merge-pr.sh --check 1301        # run every check, push and merge nothing
```

It prints one line per PR; logs are in `/tmp/dtrules-merge-pr/pr-<N>.log`.
Merge stacked PRs parent-first. If it stops on a conflict, resolve it on the
PR branch, push, and rerun it from that PR.

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

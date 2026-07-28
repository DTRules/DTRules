# DTRules

A high-performance **Decision Table Rules Engine** written in **Go**.

DTRules lets business analysts and policy experts express complex logic as
**decision tables** — a tabular form of business rules that is readable by
non-programmers and executable by the engine. Rules are authored in Excel
(or through a programmatic API), compiled to a compact postfix bytecode, and
run on a fast stack-based virtual machine. The same rule set can run as a
batch job, an interactive command-line interview, or a self-contained web app.

- **Author in Excel or via an API** — a spreadsheet is the human-friendly
  surface; a JSON authoring API is the programmatic one.
- **One compiled artifact, many front-ends** — batch, CLI interview, or web.
- **Self-contained, embeddable** — compile a rule set into a single Go binary
  with no external files at runtime.
- **Fixed-point decimals** — a 256-bit `fixed` type for token, staking, and
  blockchain math with no float drift (`dtrules docs fixed`).
- **Embedded documentation** — `dtrules docs` ships a full manual in the binary
  for humans and AI agents alike.

---

## Table of contents

- [Install](#install)
- [Quick start](#quick-start)
- [Core concepts](#core-concepts)
- [The authoring contract](#the-authoring-contract)
- [The `dtrules` CLI](#the-dtrules-cli)
- [Interactive data collection](#interactive-data-collection)
- [Embedding in a Go application](#embedding-in-a-go-application)
- [Project structure](#project-structure)
- [Sample projects](#sample-projects)
- [Expression Language (EL)](#expression-language-el)
- [Development](#development)
- [Documentation](#documentation)
- [Performance](#performance)
- [Requirements & license](#requirements--license)

---

## Install

**Option 1 — `go install`** (requires Go 1.24+):

```bash
go install github.com/DTRules/DTRules/cmd/dtrules@latest
```

**Option 2 — build from source:**

```bash
git clone https://github.com/DTRules/DTRules.git
cd DTRules
make build          # produces ./build/dtrules
make install        # installs to your GOBIN
```

**Option 3 — prebuilt binaries** from
[GitHub Releases](https://github.com/DTRules/DTRules/releases)
(linux-amd64/arm64, darwin-amd64/arm64, windows-amd64).

Verify:

```bash
dtrules version
dtrules docs            # browse the embedded manual
```

---

## Quick start

### Run the interactive demo

The fastest way to see DTRules in action is the embedded **SinusitisTherapy**
web demo — a single binary that carries its rules inside it (`//go:embed`) and
serves them as an interactive interview:

```bash
go run ./cmd/sinusitis-web
```

It picks a free port, opens your browser, and asks one question at a time
(diagnosis, age, weight, plasma creatinine, penicillin allergy). The rules
compute the recommended antibiotic, dose, renal adjustment, drug-interaction
warnings, and a plain-English rationale. Your answers stack on the page as you
go, and a **Review / edit answers** button lets you change any value and re-run.

### Run a project from the command line

```bash
# Batch: load input data via the project mapping, run a table, print the result
dtrules run sampleprojects/SinusitisTherapy \
  --entry Determine_Therapy \
  --input sampleprojects/SinusitisTherapy/testfiles/TestScenarios/ChallengeExample/input.xml

# Interactive: prompt for any input not supplied
dtrules run <project> --entry Determine_Therapy --interactive

# Web: serve the same interview in a browser
dtrules run <project> --entry Determine_Therapy --web
```

### Edit and build

```bash
dtrules init MyRules            # scaffold a new project
# ...edit MyRules in Excel...
dtrules build MyRules           # Excel → XML + compile to postfix
dtrules verify MyRules          # CI gate: Excel ↔ XML consistency + self-contained refs
```

---

## Core concepts

| Term | What it is |
|------|------------|
| **Decision table** | A table of *conditions* (rows) and *columns* (rules). When a column's condition pattern matches, its *actions* fire. The primary unit of logic. |
| **EDD** (Entity Data Dictionary) | The typed data model: entities and their fields (type, default, access, and — new — collection metadata). |
| **EL** (Expression Language) | The readable language conditions and actions are written in, e.g. `patient.age >= 18`, `set result.dose = 200`. The **only** language you author rules in. |
| **postfix** | The compiled bytecode EL is translated to. A generated artifact — **never** hand-written. |
| **VM** | A stack-based virtual machine that executes postfix against a session's entities. |
| **Mapping** | Optional XML that reconciles foreign input tag names to EDD field names when loading data. |

A rule set is a directory of XML files (`*_edd.xml`, `*_dt.xml`, optional
`*_map.xml`) plus the Excel workbooks they were authored in. `dtrules build`
extracts EL from Excel into XML and compiles it to postfix; the runtime
consumes the compiled XML.

---

## The authoring contract

> **Excel is the system of record for the DSL. `postfix` is a compiled
> artifact, never authored. Every tool that writes XML writes the same DSL back
> to Excel in the same operation.**

This is the central invariant of the project. There are exactly two ways to
change a rule:

1. **Edit Excel, then `dtrules build`** — extracts the EL into XML and compiles
   it. Excel is the input; XML is generated.
2. **Call the authoring API** (`dtrules table` / `dtrules edd`, or the MCP write
   tools) — it writes the XML DSL, compiles postfix, **and** updates Excel in the
   same operation. If the project has no Excel yet, it bootstraps one.

Hard rules:

- **Never hand-edit XML.** It is generated.
- **Never hand-write `postfix`.** It is compiled output.
- `dtrules verify` is the drift gate: it fails when rule XML has no Excel source,
  or when a table references an undefined table, field, or operator.

Full specification: [`docs/authoring-contract.md`](docs/authoring-contract.md)
or `dtrules docs authoring-contract`.

---

## The `dtrules` CLI

A single self-contained binary covering the whole authoring-to-execution
workflow. Run any command with no arguments for its usage, or `dtrules docs cli`
for a guided tour.

| Command | Purpose |
|---------|---------|
| `dtrules init` | Scaffold a new project directory |
| `dtrules build` | Extract DSL from Excel + compile postfix (the human path) |
| `dtrules run` | Run a decision table; `--interactive` / `--web` collect missing inputs |
| `dtrules table` | JSON-first per-table read/write (the programmatic path) |
| `dtrules edd` | JSON-first EDD read/write (the programmatic path) |
| `dtrules sync` | Fine-grained Excel/XML sync (`status`/`check`/`import`/`export`/`auto`) |
| `dtrules validate` | Check project structure + EL compliance |
| `dtrules verify` | CI gate: Excel↔XML consistency + self-contained references |
| `dtrules review` | Project-wide review (errors + advisory warnings) |
| `dtrules docs` | The embedded manual |
| `dtrules mcp` | MCP server over stdio (for AI agents) |
| `dtrules version` | Version, commit, build date |

### `dtrules run`

```
dtrules run [path] --entry <table> [options]

  --entry <table>      Decision table to run (required)
  --input <file.xml>   Input data loaded via the project mapping
  --interactive, -i    Prompt for any reached collect field not supplied
  --web                Serve an interactive web interview instead of a CLI run
  --port <n>           Port for --web (default: an unused port chosen by the OS)
  --data <file.xml>    Load canonical (mapping-free) data, authoritative
  --review <file.xml>  Load canonical data for re-interview (pre-filled, asked)
  --save <file.xml>    Save the collected data as canonical XML after the run
```

---

## Interactive data collection

DTRules can drive a rule set as an **interview**: instead of requiring all input
up front, it runs the rules and, each time it reaches a field marked `collect`
that hasn't been supplied, it asks for the value — on the command line or in a
web form. Fields not marked `collect` use their EDD defaults. Batch execution is
unchanged and pays no overhead.

### Marking a field collectable

Collection metadata lives on the EDD field and round-trips through Excel like
everything else. Set it via the authoring API:

```bash
dtrules edd patch --project MyRules <<'JSON'
{ "op": "update-field", "entity": "patient",
  "field": {
    "name": "pcr", "type": "double", "access": "rw", "default": "0.0",
    "collect": "true",
    "question_text": "Plasma creatinine (mg/dL)?",
    "question_type": "number",
    "question_ref_low": "0.7", "question_ref_high": "1.3", "question_units": "mg/dL"
  }
}
JSON
```

Question types: `multiple_choice` (with `options`), `ascii`, `number`, `date`.

**Reference ranges (lab-report style).** A `number` question may carry a
reference range (`question_ref_low` / `question_ref_high` / `question_units`).
Following lab convention this is *guidance, not validation*: the range is shown
when asking, and the entered value is flagged **High/Low** — but any value is
accepted, because out-of-range results are usually the important ones.

### The closed loop

The collected answers can be saved as **canonical data XML** (mapping-free,
1:1 with the EDD) and replayed or revised:

```bash
# Collect interactively and save the dataset
dtrules run MyRules --entry T --interactive --save case.xml

# Replay it as a batch job — identical result, no prompts
dtrules run MyRules --entry T --data case.xml

# Re-open it for review: every answer is re-asked, pre-filled, so you can edit
dtrules run MyRules --entry T --review case.xml --interactive
```

`--data` loads values **authoritatively** (marked collected → not re-asked);
`--review` loads them as **defaults** (re-asked, pre-filled). The web interview
exposes the same review/modify loop with a button.

### How it works

The engine adds a per-instance *collected / defaulted* bit to each entity field
and a resolver hook at the field-read chokepoint. A front-end (CLI prompt, web
form, or a test stub) implements a small `Asker` interface; the web front-end
runs each browser session's execution in its own goroutine that blocks on a
channel fed by HTTP form posts — so the goroutine *is* the continuation, with no
re-execution. With no resolver attached, the read path behaves exactly as
before.

See `cmd/sinusitis-web` for the embedded web demo and `cmd/dtrules/run_cmd.go`
for the CLI wiring.

---

## Embedding in a Go application

A compiled rule set is just XML; you can `//go:embed` it and ship one binary
with no external files. The `cmd/sinusitis-web` command is a complete worked
example — it embeds a rule set and serves it as a web interview:

```go
import "github.com/DTRules/DTRules/pkg/dtrules/web"

//go:embed rules/xml
var rulesFS embed.FS

// extract rulesFS to a temp dir, then:
web.ServeDir(addr, xmlDir, web.Options{Entry: "Determine_Therapy", Title: "My App"})
```

For programmatic (non-web) execution, the pipeline is:

```go
rs := session.NewRuleSet("MyRules")
rs.LoadFromDirectory(xmlDir)                 // load compiled EDD + decision tables
sess, _ := rs.NewSession()

m := mapping.NewMapping(sess)                // optional: set up entities + load input
m.LoadMapping(mapFile); m.Initialize(); m.LoadData(inputFile)

state := sess.GetState()
dt, _ := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Determine_Therapy"))
dt.Execute(state)

result, _ := state.FindEntity(dtrules.GetRName("result"))
```

> A higher-level `pkg/dtrules/sdk` package that wraps this glue into a one-call
> `Engine` API is in progress ([#757](https://github.com/DTRules/DTRules/issues/757)).
> Until it lands, both CLI binaries (`cmd/dtrules`, `cmd/api`) wire the pipeline
> directly, as shown above. See `dtrules docs embedding`.

---

## Project structure

```
DTRules/
├── cmd/
│   ├── dtrules/         # the main CLI
│   ├── api/             # HTTP API server (for the UI)
│   └── sinusitis-web/   # embedded interactive web demo
├── pkg/dtrules/
│   ├── authoring/       # typed authoring view + Project/EDD API
│   ├── collect/         # interactive-collection resolver bridge
│   ├── compiler/el/     # ANTLR-based EL → postfix compiler
│   ├── datafile/        # canonical (mapping-free) data XML reader/writer
│   ├── decisiontable/   # decision-table model + advisory pass
│   ├── entity/          # entities, the EDD runtime model
│   ├── excel/           # Excel ⇆ XML import/export
│   ├── interpreter/     # stack-based VM
│   ├── loader/          # XML loaders
│   ├── mapping/         # input-data mapping
│   ├── operators/       # operator registry (230+ operators)
│   ├── runtime/         # bytecode executors
│   ├── session/         # rule sets + execution sessions
│   ├── sync/            # Excel/XML sync + manifest
│   ├── web/             # interactive web interview server
│   └── version/         # build/version info
├── sampleprojects/      # example rule sets
├── ui/                  # TypeScript/React visual UI
├── docs/                # in-repo documentation
└── legacy/              # archived Java/ASM implementations
```

---

## Sample projects

`SinusitisTherapy` is the current flagship sample and the basis of the web demo.
Older samples remain for reference; some predate the current authoring contract.

| Project | Description |
|---------|-------------|
| **SinusitisTherapy** | Antibiotic selection, dosing, renal adjustment, and interaction checks — the interactive-collection demo |
| **CHIP** / **ChipApp** | Health-insurance eligibility determination |
| **KidAid** / **KidAid_Application** | Child-assistance program eligibility |
| **CorporateTax** / **StateTax** | Tax calculation |
| **Poker** | Decision-making by player archetype — the smallest end-to-end example |

---

## Expression Language (EL)

Conditions and actions are written in EL — the only language you author rules
in. EL is compiled to postfix at build time, so syntax errors are caught before
deployment, not at runtime.

**Conditions:**

```
patient.age >= 18
patient.diagnosis is equal to ignore case "Acute Sinusitis"
result.has_nexus is true
```

**Actions:**

```
set result.dose_mg = 200
add "Monitor INR closely" to result.warnings
perform Determine_Creatinine_Clearance
```

See `dtrules docs el`, `dtrules docs operators`, and
[`docs/el-reference.md`](docs/el-reference.md).

---

## Development

```bash
# Build the CLI
make build                       # -> ./build/dtrules

# Run the full gate BEFORE declaring a task done:
make check                       # go build ./... + go vet + the test suite

# Run tests
go test ./...

# Cross-compile
make build-all                   # linux, darwin, windows
```

`make check` is the authoritative gate — it runs a full-module build, `go vet`,
and the scoped test suite. A change is only done when `check` passes.

**Workflows:**

- **Excel-first (analysts):** edit the workbook → `dtrules build` → `dtrules verify`.
- **Programmatic (developers / AI):** `dtrules table put` / `dtrules edd patch`
  (write-through to Excel) → `dtrules verify`.

A React-based visual UI is available for editing tables and testing rules:

```bash
go run ./cmd/api          # backend
cd ui && npm install && npm run dev   # frontend at http://localhost:5173
```

---

## Documentation

The binary ships a complete manual — useful for humans and required reading for
AI agents driving the tools:

```bash
dtrules docs                     # list all topics
dtrules docs authoring-contract  # the Excel/postfix contract (read this first)
dtrules docs cli                 # guided tour of every subcommand
dtrules docs decision-tables     # how to write decision tables
dtrules docs el                  # the Expression Language
dtrules docs operators           # every operator with examples
dtrules docs edd                 # the Entity Data Dictionary
dtrules docs embedding           # embedding rules in a Go binary
dtrules docs fixed               # the 256-bit fixed-point type
dtrules docs workflow            # the build pipeline
```

In-repo: [`docs/`](docs/), [`CHANGELOG.md`](CHANGELOG.md), and the EL compiler
notes in [`pkg/dtrules/compiler/el/`](pkg/dtrules/compiler/el/).

---

## Performance

The Go engine is substantially faster than the original Java implementation on
the hot paths:

| Operation | Speedup vs Java |
|-----------|-----------------|
| Operator lookup | ~130× |
| Integer arithmetic | ~24× |
| String creation | ~3.7× |

---

## Requirements & license

- **Go 1.24+**

Licensed under the **Apache License, Version 2.0**.

The original Java implementation is archived under `legacy/` and is no longer the
primary implementation.

**Links:** [dtrules.com](https://dtrules.com) ·
[GitHub](https://github.com/DTRules/DTRules)

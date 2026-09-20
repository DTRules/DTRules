# DTRules — System Specification

Normative. This describes the system as it is.

---

# 1. Architecture — what we are doing

## 1.1 Purpose

DTRules executes business decision logic that domain experts author and own.
A tax analyst, a benefits caseworker or an actuary writes rules in a
spreadsheet; DTRules compiles that spreadsheet into an artifact a small
embeddable virtual machine runs, and into a trace a regulator can replay.

The decision table is the unit of logic: a set of conditions, a set of
actions, and a grid saying which actions fire for which combination of
condition outcomes. Everything else exists to get tables from a spreadsheet
into a running program without either side drifting from the other.

## 1.2 What the system is for

Three properties are the point, and the design pays for each:

- **The domain expert owns the rules.** The authored form is a spreadsheet,
  not source code, and it stays authoritative rather than becoming an import
  format.
- **The executed artifact is auditable.** What runs is a compiled, hash-
  stamped, byte-stable file that diffs in git and replays later against the
  same rules that produced it.
- **A machine can author safely.** An AI agent's edits go through the same
  funnel, the same validation and the same provenance as a human's, and cannot
  bypass them.

## 1.3 Surfaces

| Surface | For | Shape |
|---|---|---|
| Excel workbooks | domain experts | the authored system of record |
| `dtrules` CLI | developers, CI | build, run, verify, review, trace |
| Authoring API (`table`, `edd`, `map`, `project`) | agents and tools | JSON in, JSON out |
| MCP server | AI agents | the authoring API over Model Context Protocol |
| HTTP API + editor UI | interactive use | project browsing, execution, trace debugging |
| Go library | embedding | load a rule set, run a table, read the result |

## 1.4 Objects

- **Project** — a directory holding rules, their workbooks, and optionally
  scenarios. Resolved as `xml/` and `excel/` unless `DTRules.xml` says
  otherwise.
- **Decision table** — conditions, actions, and the column grid binding them,
  plus a policy (`FIRST`, `ALL`, `BALANCED`) governing how many columns fire.
- **EDD (Entity Data Dictionary)** — the declared entities and their typed
  fields. The type system for everything the rules touch.
- **Mapping** — which external XML tag becomes which entity or attribute, and
  which entities are on the stack before anything runs.
- **Entity stack** — the runtime scope chain. A bare field name resolves
  against it; `for all <array>` pushes an element for the duration of a table.
- **Postfix** — the compiled instruction stream the VM executes.
- **Trace** — a recording of one execution: initial data, every table, column
  and action, the resulting state, and the provenance to replay it.

## 1.5 Invariants

These are the rules with force. Each is enforced by a mechanism named in §2.7.

1. **Excel is the system of record for DSL.** The authored content of a rule
   is its EL text. Excel carries it; XML mirrors it.
2. **Postfix is compiled, never authored.** It is the output of DSL and is
   regenerated on every write. No surface may write it independently.
3. **Every writer is write-through.** A programmatic edit updates XML,
   compiles postfix, and updates Excel in one operation, or it fails. There is
   no XML-only mode.
4. **A rule set is self-contained.** Every table, field and operator a table
   names is declared inside the project.
5. **A name is case-insensitive and case-preserving.** `Status` and `status`
   are one name; matching normalizes, writing back uses the authored spelling.
6. **What executed can be replayed.** A trace carries the rules fingerprint
   and version that produced it, and replay against different rules is
   reported, not silently permitted.

## 1.6 What the system does not do

Stated because each is a boundary people assume otherwise:

- No DMN interchange. DMN is the live standard in this space and DTRules reads
  and writes none of it.
- No governance surface — no roles, approvals, or glossary.
- No visual model editor. The authoring surfaces are a spreadsheet and a JSON
  API.
- No per-state non-resident tax engine in the TaxReturn sample; see §2.9.

---

# 2. Specification — how it is implemented

## 2.1 Repository layout

```
cmd/
  dtrules/            the CLI, and the MCP server behind `dtrules mcp`
  api/                the HTTP API server
  tabledoc/           table documentation generator
  sinusitis-web/      a sample-specific web front end
pkg/dtrules/
  compiler/el/        ANTLR-based EL → postfix compiler
  decisiontable/      table model and the advisory pass
  interpreter/        the VM: state, entity stack, data stack (Go + amd64 asm)
  runtime/            bytecode executors — goruntime, nativeasm
  operators/          the operator registry
  entity/ session/    entities, rule sets, execution context
  excel/              Excel ↔ XML import and export
  authoring/          typed authoring SDK and the Excel write-through
  mapping/            external XML → entity loading
  analysis/           static analysis over a whole project
  trace/              trace capture, replay, reports and diffs
  apiserver/ web/     HTTP API and editor serving
sampleprojects/       13 rule sets used as tests and documentation
ui/                   TypeScript editor and debugger front end
docs/                 this file and the reference documents
```

## 2.2 The compilation pipeline

```
Excel workbook ──dtrules build──▶ XML (DSL + compiled postfix) ──load──▶ VM
     ▲                                   │
     └────────── write-through ──────────┘
                (every authoring API write)
```

- `dtrules build` reads the workbooks, extracts DSL into XML, and compiles
  DSL→postfix. It is one-directional: there is no build from XML.
- The authoring API writes XML DSL, compiles postfix, and exports Excel in the
  same operation.
- The loader is strict. It executes stored postfix and refuses DSL with
  missing or empty postfix; it does not recompile at load time.

## 2.3 The EL compiler

`pkg/dtrules/compiler/el` parses EL with ANTLR and emits postfix. Three entry
points — `CompileContext`, `CompileCondition`, `CompileAction` — corresponding
to the three kinds of row a table holds.

Locals are frame-relative slots. Contexts declare locals that live for the
whole table, so a condition can see a slot the context declared; conditions and
actions each emit their own `allocate` and therefore restart numbering from the
context's count. `Compiler.MarkLocalScope` / `ResetToLocalScope` implement that
boundary, and both compile paths (the authoring SDK and the Excel importer) use
them.

The compiler is injected with an operator existence check and an arity check so
a misspelled operator is refused at authoring time rather than at execution.

## 2.4 Execution

`pkg/dtrules/interpreter` holds the VM. Three stacks:

- **data stack** — operands and results.
- **entity stack** — the scope chain. `entitypush`/`entitypop` move it;
  `for all` pushes an element per iteration.
- **control stack** — frames for locals; `allocate`/`deallocate` and
  `local@`/`local!` address slots relative to the current frame.

Two executors live behind one interface in `pkg/dtrules/runtime`: a portable Go
one and an amd64 assembly one.

## 2.5 Data in

`pkg/dtrules/mapping` loads external XML against a mapping file:

- `<entity>` declarations give cardinality — `1` for a singleton, `*` for many.
- `<createentity tag id list>` says which tag creates which entity, which child
  attribute identifies an instance, and which array it is appended to. With no
  `list`, the loader falls back to the entity name plus `"s"`.
- `<initialentity epush>` names the entities pushed before any table runs.
  These are pushed **before** the document is read, because a `setattribute`
  resolves its enclosure against the entity stack while the document is being
  read.

A tag that resolves against nothing is dropped silently at load; this is why
§2.7 validates mappings at authoring time.

### 2.5.1 Field value constraints

An EDD field may declare what values it can legally hold:

- `allowed_values` — a closed vocabulary, written as one `<allowed_value
  value="…"/>` per member. Valid on `string` and `integer` fields, and
  independent of `collect`: a field nobody is asked for can still have one.
- `max_length` — the longest value a `string` field may hold, in characters.
- `max_words` — the most whitespace-separated words a `string` field may hold.

A vocabulary is matched the way every other name in the system is matched:
without regard to case (§1.5.5). `Acute Sinusitis` and `acute sinusitis` are
one value, and the spelling written back out is the authored one.

The constraints travel with the field through every layer — the entity model,
the EDD XML, `dtrules edd get|put|patch`, and columns N–P of the Excel EDD
sheet (`Allowed Values`, `Max Length`, `Max Words`) — so `dtrules build` and
`dtrules verify` round-trip them byte-identically. A sheet grows those three
columns only where some field declares a constraint: the importer reads by
column position and an absent column reads as empty, so a project that uses
none keeps the workbook it already has.

`dtrules validate` rejects a field whose own `default` its constraints reject,
and the authoring API refuses to write one. Such a default is unreachable
rather than merely odd: the field starts every run holding a value the rules
were told it can never hold.

Enforcement on the paths that write a field from outside the rules — `--input`
mapping, `--data`, the collect resolver, the web interview and the API — is
the second half of #1209 and is not yet wired.

## 2.6 Static analysis

`pkg/dtrules/analysis` runs project-wide:

- **EDD usage** — fields declared and never referenced, and fields written but
  never read. Bare identifiers are resolved against the entity stack: what the
  mapping pushes at initialization, plus what each table's contexts push.
- **External references** — a `perform` of an undefined table, a dotted
  `entity.attr` the EDD does not declare, an operator absent from the registry.
- **Dated constants** — a comment citing a year that is not the project's
  declared tax year.
- **Advisory pass** (`pkg/dtrules/decisiontable`) — redundant conditions,
  columns subsumed by another, no-op columns.

## 2.7 Enforcement of the invariants

| Invariant (§1.5) | Mechanism |
|---|---|
| 1. Excel is the record | `dtrules verify` rebuilds XML from the committed Excel and requires byte equality |
| 2. Postfix is compiled | the strict loader refuses missing postfix; every write regenerates it from DSL |
| 3. Writers are write-through | `authoring.Project.Save`/`SaveEDD` and the `map` writer export Excel in the same operation and fail if they cannot |
| 4. Self-contained | `verify`'s external-reference gate; `map put`/`map patch` validate against the EDD before writing |
| 5. Case-insensitive, case-preserving | matching uses `EqualFold` and lowercased keys; `AuthoredName()` is written back |
| 6. Replayable | traces carry a rules fingerprint and DTRules version; the debugger reports match, mismatch or unknown |

`dtrules verify` is the gate. It is read-only, and runs, in order:

1. **unique table names** across every file in the project
2. **build idempotency** — rebuilding from the committed Excel changes nothing
3. **source headers** — every artifact records the workbook it came from
4. **prefix ordering** — numeric filename prefixes agree with sheet order
5. **suffix content consistency**
6. **Excel presence** — a system-of-record workbook exists
7. **external references** (§2.6)

Implemented as `checkBuildIdempotency`, `checkSourceHeaders`,
`checkPrefixOrdering`, `checkSuffixContentConsistency`, `checkExcelPresence`
and `checkExternalRefs` in `cmd/dtrules/verify.go`, with the unique-name gate
ahead of them.

## 2.8 Trace and debugging

A trace records initial data, every table/column/action, the final state, and
provenance. `pkg/dtrules/trace` replays it to any node, producing a session
whose entity stack is what it was at that point.

The debug API (`/api/debug/*`) exposes: load, status, position, tree,
entity/array inspection, console, watch, report, baseline, speculate.

- **console** — evaluates an expression at the current position. Input is
  compiled as EL, falling back to raw postfix. Read-only: mutating operators
  are refused, against a list that depends on which language the input was read
  as, because EL emits balanced `entitypush`/`entitypop` for scoping.
- **watch** — walks forward until a predicate becomes true. Edge-triggered and
  bounded; a node where the expression cannot be evaluated is not a match and
  not a failure.
- **baseline** — a second trace to diff reports against, independent of
  speculation.
- **tree** — the whole tree by default; `depth` and `node` fetch part of it.

## 2.9 Known gaps in the sample rule sets

The samples are tests and documentation, not products, and one gap is
load-bearing enough to record:

- **TaxReturn's non-resident state tax uses resident deductions.**
  `Dispatch_State_Tax` runs every roster entry through its state's table
  (`Compute_Roster_State_Tax`, dispatched by state code over the full set of
  state and territory tables), so each `state_tax_result` carries its own
  liability and the other-state credit is real. What each `XX_Tax` table
  applies to a non-resident's sourced income is the same standard deduction
  and exemptions it applies to a resident; the proportional non-resident
  deductions the states actually use are not modelled, and neither are
  per-state credit ceilings (#1201). Four state tables (AR, LA, NM, OK) have
  conditions with no actions wired to any column and compute nothing; they
  were unreachable before and now record a zero honestly (#1200).

## 2.10 Versioning and release

The version comes from `git describe`; `pkg/dtrules/version` reports it and the
build stamps it. A `v*` tag triggers the release workflow, which builds five
platform binaries plus checksums and publishes a GitHub release. `CHANGELOG.md`
carries the notes.

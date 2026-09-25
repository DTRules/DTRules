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
  A condition cell holds one of four values and no others: `Y` (must be
  true), `N` (must be false), `-` (this column does not test this condition),
  `*` (otherwise). An action cell holds `X` or nothing.
- **Otherwise column** — a column marked `*`. `*` does not mean "don't care":
  it is allowed only in the **last column** of a table, and only when that
  column holds no `Y` and no `N`. The column executes if and only if no other
  column executes, under every table policy. It follows that a table has at
  most one. `*` outside the last column, or in a column that also holds `Y`
  or `N`, is a load error. There is no "always" column: an action that must
  always execute carries an `X` in every column, the otherwise column
  included. "Last" means the last column holding a `Y`, `N`, `*` or `X`;
  columns after it are **padding** — all `-`, no action — and are not columns
  of the table: no policy traces them (#1221). `add-column` on a table with an
  otherwise column inserts the new column in its place and moves it one
  right, so it stays last; `patch` reports the new column's number.
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
- No embedding SDK. An embedder imports the engine packages and drives them
  directly; see §2.11.
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
  interpreter/        the VM: state, entity stack, data stack
  runtime/            the runtime interface and its Go executor (goruntime)
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
scripts/merge-pr.sh   the merge gate (§2.10); the rest of scripts/ is untracked
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
  same operation. It sees the same file set as the loader: files
  `loader.SkipRuleFile` excludes (templates, test data, schemas) are not part
  of the project to it either (#1300).
- A table's workbook follows its file: `xml/<dir>/X_dt.xml` pairs with
  `excel/<dir>/X.xlsx`. Moving a table (`set-file`, or `put` with another
  `file`) gives it the target file's workbook, and a file the API creates has
  its workbook created in the same Save — the one exception to the refresh's
  rule of never creating a missing workbook (#1062), because a new file's
  workbook cannot exist yet (#1225). Save writes new files before the files
  tables left, so a failed write cannot lose a moved table.
- The first authoring write to a project with no Excel (no workbook, no
  manifest) bootstraps it, and leaves every rule file backed by a workbook:
  tables, EDDs (an entity naming no workbook takes its file's) and mappings
  (#1303). An EDD with an entity comment, which the EDD sheet has no cell for,
  keeps its XML instead.

- A refused `table put` (CLI or MCP `table_put`) names the kind of failure,
  so a caller knows where to look: `compile_error` for EL that does not
  compile and nothing else; `otherwise_rule` for a `*` the otherwise-column
  rule forbids; `invalid_number` for a table number outside its file's range
  or already taken; `invalid_table` for any other refusal of the body (#1222).
- `map get` emits a mapping's section comments as `{"comment": "..."}` entries
  in `attributes`, and `map put` writes them back, so a put of get's output
  loses nothing the mapping model holds.
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
A declaration must introduce a new name: one that is already an EDD attribute,
or a local in scope, is refused at compile time. The emitter makes that
decision, because the grammar cannot tell a defined identifier from an
undefined one.

The compiler is injected with an operator existence check and an arity check so
a misspelled operator is refused at authoring time rather than at execution.

The emitter also refuses, at authoring time, a field read where a name is
required: after `sort … by`, a field or local whose declared type is not `name`
is an error that points at `the name "<field>"`, because it would be evaluated
before the sort with no entity on the stack and fail at execution (#1227).

A name value is written `the name "foo"` or `(name) "foo"` (both compile to
`"foo" cvn`); a variable holding a name is the field itself. The `$foo`
spelling is removed (#1280): it lexed as a name, but the sigil rode into the
postfix, where a bare `$foo` is an executable lookup of an attribute literally
called `$foo`, which no EDD declares — so every use failed at execution. The
compiler refuses it and names the replacement. `perform $x` went with it, and
with it the `performName` alternative (#1254); a table chosen at runtime is
`perform table named (<string>) among …`, which keeps the call edges visible
to the analyzer.

The grammar sweep (`grammar_sweep_test.go`) holds every labelled alternative
in `EL.g4` to a corpus row in `testdata/`. Each row must compile to non-empty
postfix, and must parse through the label it is filed under (checked on the
tree the compiler itself parsed). Every rule/label named in `testdata/` must
exist in `EL.g4`. Exceptions are listed per row with a reason:
`grammar_known_fails.tsv` (does not compile), `grammar_label_misses.tsv` (the
parser can never choose the label) and `grammar_helpers.tsv` (a fragment that
cannot be compiled on its own). An exception that starts passing fails the
sweep, so each list can only shrink.

`grammar_label_misses.tsv` is empty: an alternative the parser can never
choose is deleted, with its emitter, rather than kept and listed (#1250). An
alternative that differs from an earlier one only by which `typedX : IDENT`
placeholder it names can never be chosen — the parser has no symbol table,
so it takes the first — and typing belongs in the emitter of the one that
is.

## 2.4 Execution

`pkg/dtrules/interpreter` holds the VM. Three stacks:

- **data stack** — operands and results.
- **entity stack** — the scope chain. `entitypush`/`entitypop` move it;
  `for all` pushes an element per iteration.
- **control stack** — frames for locals; `allocate`/`deallocate` and
  `local@`/`local!` address slots relative to the current frame. Every
  execution of a table opens a frame of its own (`RDecisionTable.Execute`,
  and `performaliased`, which runs a body without its context) and closes it
  on return, error included. The context runs the body through `executetable`
  inside that same frame, matching the compiler's per-table numbering; a
  performed table's slot 0 is therefore never its caller's (#1226). Frames
  count against `stackLimit` like every stack, so `perform` nested deeper
  than 1000 tables, recursion included, fails with "Control Stack Overflow",
  even when the tables declare no locals.

One executor lives behind the interface in `pkg/dtrules/runtime`: the portable
Go VM (`pkg/dtrules/interpreter/vm.go`, wrapped by `runtime/goruntime`). There
is no assembly executor; the amd64 one was removed because it addressed a
`DTState` layout the engine no longer has (#1267).

**Change tracking.** The state records whether execution changed entity data
(`dtrules.ChangeTracker`: `Changed`, `ResetChanged`), so a host running rules
on a timer can skip a no-op run without diffing its output (#1233). A change
is a write that leaves a different value behind:

- an attribute write (`DTState.Def`, which every rule write goes through, and a
  collector's answer) whose stored value would not save as the one it
  replaced: a different type, different text, a different entity (by
  identity), or an array whose elements differ pairwise. This is stricter than
  `Equals`, which lets doubles differ by 1e-9. Writing the value a field
  already holds is not a change;
- an in-place array operation that gained, lost or reordered elements
  (`addto`, `addat`, `remove`, `removeat`, `cleararray`, `copyelements`,
  `addarray`, `add_no_dups`, the sorts, `randomize`, and the operators that
  fill a destination array). An array built by `newarray` is *fresh* until an
  attribute or another array holds it, so filling an array literal is not a
  change; storing it is compared like any other attribute write;
- an entity stack that differs from the one `ResetChanged` recorded. The stack
  decides which instance each name resolves to, and so what a save writes, so
  an unbalanced `entitypush` or `entitypop` is a change. A balanced push/pop is
  not.

What is guaranteed: for writes made **by the running rules** (the paths above),
a run that changes what `--save` writes reads changed. The reverse does not
hold. The flag records writes, not a diff: a run that sets a field to 7 and back
to 3 reads changed, and storing an entity created during the run always
counts, because entities compare by identity.

Not counted: writes that bypass `Def` and the operators, made by the host or by
tooling rather than by the rules. These are the mapping and canonical data
loaders (that is input; `dtrules run` resets after them), `entity.Put` called
directly by a Go host, the authoring surfaces (`authoring/execute.go`, the
debug session), `pkg/dtrules/apiserver`, and trace replay. A host that writes
this way between `ResetChanged` and `Changed` must account for its own writes.
`populateErrorEntity` (inside `performcatcherror`) writes with `Put`, but only
into a new error entity that it pushes, and the stack comparison catches the
push.

The flag is sticky until reset;
`dtrules run` resets it after loading data, just before the entry table runs,
and writes it on the `--save` root as `<dtrules-data changed="true|false">`.
`datafile.Read` ignores the attribute.

**Dates and time.** A date value is an instant that carries a zone.

- `d + N days|months|years` is calendar arithmetic (Go `AddDate` in the date's
  zone): the wall-clock time is kept and the date moves, so `d + 1 days` across
  a daylight-saving change is 23 or 25 hours later. `months from` and `years
  from` compare calendar fields: months by year and month, years by
  anniversary, each date's fields read in the zone that date carries.
- `days from d1 to d2` (`daysbetween`) counts **calendar days, both ends
  included** (#1265): the number of dates from d1's date to d2's date, with
  both dates read in **d1's zone**. Mar 8 to Mar 10 is 3; a date to itself is
  1; Jan 1 to Dec 31 is 365. When d2's date is earlier the count is the same
  days, negative: Mar 10 to Mar 8 is -3. The result is never 0. Inclusive
  because the users are financial: a period from its first day to its last
  covers both. The time of day does not count: 23:00 to 01:00 the next
  morning is 2, and 01:00 to 23:00 on the same date is 1. Neither does a
  daylight-saving change: in America/Chicago 2026-03-08 00:00 to 2026-03-10
  00:00 is 47 hours and 3 days. So `days from d to d + N days` is N + 1 for
  N >= 0 and N - 1 for N < 0; it is not the inverse of `+ N days`. Reading d2
  in d1's zone keeps the sign honest when the zones differ: a d2 earlier than
  d1 is never a later date. It also makes the count depend on which date comes
  first when their zones differ (`days from d2 to d1` reads both in d2's zone
  and need not be the negation). The count is exact over years 1-9999. Before
  #1265 this was whole 24-hour periods of elapsed time, truncated toward zero;
  rules that added 1 to get an inclusive count (TaxReturn's
  `Allocate_Income_By_State`) no longer do.
- Below a day (#1232), `d + N seconds|minutes` and `seconds|minutes from d1 to
  d2` (operators `addseconds`, `addminutes`, `secondsbetween`,
  `minutesbetween`) are elapsed time between instants. A difference is `d2 -
  d1` in whole units, truncated toward zero, independent of the zones the dates
  carry; across a daylight-saving change it counts the time that actually
  passed. An offset keeps the zone of the date it moved. Both work on Unix
  seconds, so spans beyond `time.Duration`'s ~292 years are exact. An offset
  that would leave years 1-9999, or a difference too large for an integer, is
  an error rather than a wrapped value. Leap seconds are not counted, as in
  every other date operation.
- `current time` (`now`) is the current instant, in UTC. It is what a rule
  running on a timer measures against: `seconds from job.last_progress to
  current time > 120`. Anchored to UTC for the same reason as `today` (#743)
  — the zone a date carries decides what the calendar operators read, so it
  must not answer differently on a server in another zone. `current time in
  zone "<tz>"` is the same instant stamped with that zone (#1266).
- `current date` is today's date at midnight UTC (`today`), not the current
  instant. `current date in zone "<tz>"` (`currentdateinzone`) is also the
  current instant, stamped with that zone: `in zone` there changes the
  meaning and not only the zone, which is why `current time` exists.
- `current time` is a phrase, like `current date`, so it reserves no
  identifier: `current`, `time` and a field named `current.time` all keep
  working. It is still a `dexpr` alternative of its own (`dateCurrentTime`),
  so the compiler types it — `current time + 1` is a compile error where
  `current time + 1 days` is a date, and a comparison against a date uses the
  date comparison rather than the generic one.
- `get current timestamp` is removed (#1266). It emitted a bare
  `gettimestamp`, an operator that *pops* a date and formats it, so the
  statement underflowed the data stack or stringified whatever happened to be
  under it. Assign `current time` to a string field instead: `cvs` renders a
  date carrying a time as RFC3339Nano.

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

**Collection.** A field the EDD marks `collect` carries a question, and a run
may reach it without having a value. A `Collector` attached to the state is
called just before such a field is read; with none attached execution is pure
batch and pays nothing. Three collectors exist, and they differ only in what
answers: the CLI prompt (`dtrules run --interactive`), the web interview
(`--web`), and the recorder (`--pending <file.json>`).

The recorder is the non-blocking one. It records the question — entity,
instance id, field, text, type, options, reference range, units and the
default it substituted — writes the set as a JSON array, and lets the default
stand so the run finishes. Its result is therefore **provisional**: computed
from defaults nobody confirmed. `dtrules run` says so on the result and exits
`3`, distinct from `0` (complete, and the file holds `[]`) and from `1`
(error). A caller answers the questions, loads them with `--data`, and re-runs
until the exit code is `0`. Because the substituted defaults steer which
branches the run takes, the *set* of questions reached can change once real
answers arrive — so the loop is the contract, not a single pass. `--pending`
is mutually exclusive with `--interactive` and `--web`: it is the opposite of
asking, not a variation on it.

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

**Where they are enforced.** Every path that writes a field from *outside* the
rules refuses a violating value: the mapping (`--input`, XML and JSON), the
canonical data file (`--data` and `--review`), the `collect` resolver and so
the web interview, the API server's `/api/execute`, and the Go SDK
(`authoring.Project.SetAttribute` and `authoring.DebugSession.SetAttribute`,
which return the gate's error unchanged and leave the field's previous value
in place). Each calls one gate,
`entity.CheckExternalWrite`; none compares values itself.
`TestEveryDecodingWriterIsGated` holds the list closed: a file under
`pkg/dtrules` that decodes JSON or XML and calls `Put` must call the gate
(#1220, which deleted the one that did not: an unused `loader.JSONDataLoader`). A refusal names
`entity.field`, the offending value and the allowed set (or the limit and the
actual size); the CLI exits non-zero and prints no result, and the API answers
`400`. A refused interview answer leaves the field *uncollected*, still holding
its default, so the question can be asked again.

A field that is simply absent from the input is not a violation: it takes its
default. The check applies to values supplied, and an illegal default is a
`dtrules validate` error rather than a run-time one.

**Where they are not.** A rule's own assignment is never refused. `set
patient.diagnosis = "Bannana"` runs. A constraint describes what the outside
world may hand in, not what the rules may compute — a rule set is allowed to
know something its EDD's vocabulary has not been told yet, and a table that
halted mid-run because an intermediate value was not yet in the list would be
worse than the typo. `dtrules review` reports such a literal as an **advisory**
(`constraint_advisories`: the table, the action number, the field, the literal
and the set), because out of the set is nearly always a typo. Advisories never
gate deployment.

A field declaring no constraint is not checked at all — the gate returns on a
nil check, so a project that declares none pays nothing.

## 2.6 Static analysis

`pkg/dtrules/analysis` runs project-wide:

- **EDD usage** — fields declared and never referenced, and fields written but
  never read. Bare identifiers are resolved against the entity stack: what the
  mapping pushes at initialization, plus what each table's contexts push.
- **External references** — a `perform` of an undefined table, a dotted
  `entity.attr` the EDD does not declare, an operator absent from the registry.
- **Dated constants** — a comment citing a year that is not the project's
  declared tax year.
- **Constrained assignments** — a `set field = "literal"` whose literal is
  outside the field's declared vocabulary (§2.5.1). Advisory: a rule may write
  what it computes, but the literal is usually a typo.
- **Advisory pass** (`pkg/dtrules/decisiontable`) — redundant conditions,
  columns subsumed by another, no-op columns, and column actions that can never
  run in a table with no conditions (#1230). Such a table has nothing to select
  a column with: under FIRST or ALL the engine builds no tree and runs only the
  initial actions, so every marked action is dead; under BALANCED (the default
  when no policy is given) it runs column 1 only, so an action marked only in
  later columns is dead. The warning names those actions and points at initial
  actions (or column 1). The other advisory checks still run on the table.

## 2.7 Enforcement of the invariants

| Invariant (§1.5) | Mechanism |
|---|---|
| 1. Excel is the record | `dtrules verify` rebuilds XML (tables, EDDs and mappings) from the committed Excel and requires byte equality, and requires every rule file to be one a workbook produces |
| 2. Postfix is compiled | the strict loader refuses missing postfix; every write regenerates it from DSL; `TestSamplePostfixIsCompiledFromDSL` recompiles every stored DSL row in `sampleprojects/` and requires the stored postfix to equal it, with no postfix lacking DSL |
| 3. Writers are write-through | `authoring.Project.Save`/`SaveEDD` and the `map` writer export Excel in the same operation and fail if they cannot |
| 4. Self-contained | `verify`'s external-reference gate; `map put`/`map patch` validate against the EDD before writing |
| 5. Case-insensitive, case-preserving | matching uses `EqualFold` and lowercased keys; `AuthoredName()` is written back |
| 6. Replayable | traces carry a rules fingerprint and DTRules version; the debugger reports match, mismatch or unknown |

`dtrules verify` is the gate. It is read-only, and runs, in order:

1. **unique table names** across every file in the project
2. **build idempotency** — rebuilding from the committed Excel changes nothing.
   The rebuild includes mappings (`_map.xlsx` → `_map.xml`), which run outside
   the sync pipeline and were never compared before #1300.
3. **workbook provenance** — rebuilt on a copy with every rule file removed,
   each committed `_dt.xml`, `_edd.xml` and `_map.xml` comes back. Idempotency
   cannot see a file no workbook writes: its rebuild leaves the file in place,
   and it compares equal to itself. Only presence is checked here; content is
   (2)'s job. Files the loader does not read as rules (`loader.SkipRuleFile`:
   templates, test data, schemas) are exempt.
4. **source headers** — every artifact records the workbook it came from
5. **prefix ordering** — numeric filename prefixes agree with sheet order
6. **suffix content consistency**
7. **Excel presence** — a system-of-record workbook exists
8. **external references** (§2.6)

Implemented as `checkBuildIdempotency`, `checkWorkbookProvenance`, `checkSourceHeaders`,
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

A pull request reaches `main` only through `scripts/merge-pr.sh`. It squashes
the PR onto the current `main` in a scratch worktree and runs the checks on
that combined tree, not on the PR alone: the generated parser must match the
merged `EL.g4`, then `make check`, `dtrules verify` on every sample with an
`excel/` directory, and `dtrules validate` on CHIP. Only then does it merge,
with `--match-head-commit` set to the commit it tested. A PR that no longer
squashes cleanly is rebased and force-pushed (with lease). Commits inherited
from an already-merged PR are dropped. Only two conflicts are resolved
automatically: additions on both sides of a Markdown file, and generated
parser files, which are rebuilt from the merged grammar. Any other conflict
stops the run. `--check` runs the checks without pushing or merging.

## 2.11 Embedding the engine

The Go library surface (§1.3) is the engine packages themselves. There is no
wrapper package and none is planned: `pkg/dtrules/sdk` existed briefly and was
removed in `69774f70` because data already enters through the EDD as XML
(§2.5), so a parallel programmatic entity API restated the same surface in a
second, unvalidated shape. Both CLI binaries, `pkg/dtrules/web` and
`pkg/dtrules/interview` embed the engine this way; `pkg/dtrules/interview` is
the one packaged runner, and it wraps an interview, not the engine.

The supported sequence:

1. **Load.** `session.NewRuleSet(name)`, then `LoadFromDirectory(xmlDir)` or
   `LoadFromFS(fsys, root)` for a `//go:embed`ed `xml/` tree. A loaded rule set
   is immutable and shareable.
2. **Session.** `rs.NewSession()` per execution — a session owns one entity
   stack.
3. **Data in**, either way of §2.5:
   - a mapping — `mapping.NewMapping(sess)`, `LoadMapping`, then
     `LoadDataAndPushSingletons(doc)` (or `Initialize()` with no input);
   - canonical data — push the singletons the rules resolve bare names against
     (`sess.CreateEntity` + `state.EntityPush`, the set a mapping's
     `<initialentity>` would name), then `datafile.Read(r, find, create, mode)`.
4. **Execute.** `sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))`,
   then `dt.Execute(state)`. To learn whether the run changed anything, call
   `state.(dtrules.ChangeTracker).ResetChanged()` after the data load and read
   `Changed()` after execution (§2.4).
5. **Read out.** `state.FindEntity(...)` — the executed instance on the stack.
   `CreateEntity` returns a fresh, empty entity and is not how a result is read.
6. **Optionally trace.** `trace.WriteHeader`, `dts.SetOutput` + `dts.EnableTrace`
   before the data load, `trace.WriteFinalState` and `trace.WriteFooter` after
   execution (§2.8).

README.md and `dtrules docs embedding` document this sequence;
`pkg/dtrules/embedding_example_test.go` compiles and executes it against
`SinusitisTherapy`, both ways of loading data, and fails if either document
points embedders at the removed SDK again (#1211).

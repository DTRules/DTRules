# DTRules Authoring Contract

Status: **partially implemented.** The multi-file organization, table numbers,
per-file ranges, the `authoring-notes.md` journal, and EDD ordering (see
"Multi-file organization" below) have shipped. The core enforcement model in
this document — Excel write-through on every API write, removal of
`dtrules compile`, and the `verify` drift gate — remains **proposed**.

This document defines the single, enforceable authoring model for DTRules
rule sets. It exists because the authoring surfaces drifted into an
incoherent hybrid: the project claimed "Excel is the source of record"
while the tooling (`dtrules compile`, `table put/patch`, `edd put/patch`,
MCP write tools) authored directly into XML and treated Excel as an
afterthought — when it touched Excel at all. See the
[history](#how-we-got-here) below.

The rule, in one sentence:

> **Excel is the system of record for DSL. Every tool that writes XML must
> write the same DSL back to Excel in the same operation. Nothing writes
> rule content any other way.**

This contract is enforced **in the tools**, not by documentation. CLAUDE.md,
`dtrules docs`, and this file are signposts — an AI can ignore them, a
different agent never loads them, a human skims past them. So none of them
are relied on for correctness. The guarantees below live in code: the
authoring API is write-through and fail-closed, the bypass command is
deleted, and `dtrules verify` + the strict loader **reject** any rule set
where XML, Excel, and postfix have drifted. Improper authoring does not
produce a working, committable artifact — that is the enforcement, and it
holds whether or not anyone read the docs.

---

## The three invariants

### 1. Excel is the system of record — for DSL only

Humans own rules as Excel workbooks (`*.xlsx`). The authoritative content
of a rule is its **DSL** (the EL text in `condition_dsl`, `action_dsl`,
`context_dsl`, `initial_action_dsl`). Excel carries DSL and table
structure. It does **not** carry postfix.

### 2. postfix is a compiled artifact, never authored

`postfix` is the compiled output of DSL. No surface may write postfix
independently of the DSL it comes from. This is the existing contract from
#817/#818 (the `Action.Postfix` override field was removed; `syncToXML`
regenerates postfix from DSL on every write). This contract is **kept** and
is load-bearing for invariant #1: because postfix is always recompiled from
DSL, **Excel never needs to store postfix**, which is what makes the
XML→Excel direction lossless for everything that matters.

### 3. The authoring API is the only writer, and it is write-through

There is exactly one way to change a rule programmatically: the authoring
API (`pkg/dtrules/authoring`, surfaced as `dtrules table`/`dtrules edd` and
the MCP write tools). Every API write performs, atomically:

1. update the XML DSL,
2. compile DSL → postfix into the XML (invariant #2),
3. **update Excel** to match the new DSL.

Step 3 is mandatory and **fail-closed**: an API write that cannot land
Excel must fail, not silently skip it. If the project has no Excel yet, the
API **bootstraps** it from the XML (see [Bootstrap](#bootstrap)). There is
no "XML-only project" mode.

This is the whole point of routing AI authoring through the API: the API is
the one place where we can *guarantee* Excel stays current even though the
edit physically lands in XML first.

---

## Enforcement (in the tools, not in prose)

Each invariant maps to a mechanical control. Documentation is **not** in this
table because documentation is not a control.

| Invariant | Mechanical control | Failure mode for improper authoring |
|-----------|--------------------|-------------------------------------|
| 1. Excel is the record for DSL | `dtrules verify` rebuilds XML from Excel and asserts byte-equality with the committed XML. Wired into CI / pre-commit as a hard gate. | Hand-edited XML, or stale Excel, ⇒ `verify` exits non-zero. The change cannot be committed clean. |
| 2. postfix is compiled, never authored | The strict loader refuses DSL with missing postfix (already shipped). Extended: `verify` recompiles each DSL and rejects any stored postfix that differs from the fresh compile. | Hand-written or forged postfix ⇒ rejected at verify, and overwritten on the next `build`. It never executes. |
| 3. API is the only writer; write-through, fail-closed | `authoring.Project.Save`/`SaveEDD` and the MCP write tools always export Excel in the same operation; they error if they can't and bootstrap if Excel is absent. The `dtrules compile` bypass command is deleted. Open-workbook lock files block writes. | An API write that cannot land Excel ⇒ hard error, no partial write. There is no command that writes XML alone. |

The combined effect: the **only** way to obtain a rule set that builds,
loads, and passes `verify` is to author through Excel (`build`) or through
the write-through API. An agent that scribbles directly into XML produces a
state that the next `build` overwrites and that `verify` rejects in CI — so
the improper path is self-defeating, not merely discouraged.

Note: the filesystem cannot stop a process from writing bytes into an
`.xml` file. Enforcement therefore targets *survival*, not *prevention* — a
hand edit does not survive a build and does not pass the gate. Correctness
does not depend on the agent choosing to behave.

## Data flow

```
            ┌─────────────────────────────────────────────┐
 HUMAN ───▶ │  Excel workbook (*.xlsx)  ← system of record │
            └─────────────────────────────────────────────┘
                        │  ▲
            dtrules build│  │ write-through (every API write)
       (extract DSL,     │  │
        compile postfix) ▼  │
            ┌─────────────────────────────────────────────┐
   AI  ───▶ │  XML (*_dt.xml, *_edd.xml)                   │ ◀── authoring API
            │  DSL (mirror of Excel) + compiled postfix    │     (table/edd put/patch, MCP)
            └─────────────────────────────────────────────┘
                        │
                        ▼  loader consumes postfix only (strict; no recompile)
                  execution
```

- **Human path:** edit Excel → `dtrules build` extracts DSL to XML and
  compiles DSL→postfix. Excel is the input; XML is generated.
- **AI path:** call the authoring API → it writes XML DSL, compiles
  postfix, and writes the DSL back to Excel. Excel stays the record.
- **Compilation** always means DSL→postfix. The DSL it compiles always has
  a faithful twin in Excel — maintained by `build` (human path) or the API
  write-through (AI path).
- **The loader** is strict: it executes stored postfix and refuses to load
  DSL with missing/empty postfix. It does not recompile at load time.

<a name="bootstrap"></a>
## Bootstrap (no Excel yet) — one-time, not a build direction

`dtrules build` is **one-directional**: Excel → XML (+ compile). There is no
`build --from-xml`. Building *from* XML would treat XML as a source, which is
exactly the ambiguity this contract removes. XML is a generated artifact; you
never build from it.

XML→Excel happens only as a one-time **bootstrap**, in two situations, neither
of which is steady-state authoring:

- **Migration** of a pre-existing XML-only rule set into the contract.
- **Recovery** when Excel is lost but XML survives.

Bootstrap generates Excel from the XML's DSL and writes a `.sync-manifest.json`.
It is lossless because Excel only needs DSL (invariant #2). It is owned by the
authoring API's "Excel absent ⇒ create it" path and an explicit one-shot
migration command — **not** by `build`. After bootstrap the project is in
normal steady state and the write-through applies on every edit.

"No manifest ⇒ treat as XML-only and skip Excel" is **removed**. Absent Excel
means *bootstrap it*, never *skip it*.

> **Open:** the existing XML-only sample projects (KidAid, CHIP, TaxReturn, …)
> either get migrated once (preferred — no permanent second class of project)
> or are explicitly frozen as legacy fixtures outside the contract. Decision
> pending.

## What is removed

- **`dtrules compile`** — the public subcommand is deleted. It wrote postfix
  to XML without touching Excel: a writer outside the chokepoint. DSL→postfix
  remains as an internal step owned by `dtrules build` and the authoring API.
- **`dtrules build --from-xml`** — the XML→Excel build direction is deleted.
  `build` is Excel → XML only. The lone legitimate XML→Excel use is one-time
  bootstrap (see above), which is not a build mode.
- **"XML-only project" mode** — the no-manifest no-op path in
  `authoring.GuardExcelInDir` / `RefreshExcelInDir` and the tests that pin it
  (`TestGuardExcelInDir_NoManifestIsNoOp` and its refresh twin) are inverted:
  no-Excel triggers bootstrap, not a no-op.

## What stays as-is

- The strict loader (no auto-recompile) — already correct.
- The #818 postfix-is-compiled contract — already correct and now explicitly
  the reason Excel can stay DSL-only.
- `dtrules build` — already compiles from Excel via the importer's EL
  compiler. It becomes one-directional (Excel → XML); see removals.

## Conformance checklist (for the implementation phase)

Enforcement (the controls that must hold regardless of docs):

1. `authoring.Project.Save` / `SaveEDD`: Excel write-through is unconditional
   and fail-closed; absent Excel ⇒ bootstrap + create manifest. No "XML-only"
   skip path. Same guarantee on the MCP `table_put`/`table_patch`/`edd_put`
   tools — the AI's only programmatic authoring surface enforces the contract
   by construction.
2. Delete `cmd/dtrules/compile_cmd.go` and its registration in `cli.go`/
   `main.go`; remove `dtrules docs compile`. The bypass writer no longer exists.
3. `dtrules verify` is the drift gate and must (a) assert committed XML ==
   `build(Excel)` byte-for-byte, and (b) recompile every DSL and reject any
   stored postfix that differs. Exit non-zero on any drift. This is the
   mechanical backstop against hand-edited XML/postfix.
4. Invert the two no-op tests (`TestGuardExcelInDir_NoManifestIsNoOp` + refresh
   twin); add bootstrap-from-XML coverage and a verify-rejects-drift test.

Documentation (signposts only, not controls):

5. `dtrules docs workflow` updated to this contract; CLAUDE.md points at this
   file. Neither is relied on for correctness.

Proof case:

6. `sampleprojects/SinusitisTherapy` reconciled through the corrected path
   (its Excel generated as the record), then `dtrules verify` passes on it.

## Multi-file organization

Status: **implemented**. Table numbers, per-file ranges, `set-file`/`set-range`
moves, empty-file auto-deletion, the `authoring-notes.md` journal, and EDD
alphabetical ordering all shipped via the authoring SDK + `dtrules table`
surface. The broader contract above (Excel write-through enforcement, removing
`dtrules compile`, the `verify` drift gate) remains proposed.

A project's decision tables may be split across many `*_dt.xml` files (by
domain, by team, or one-per-state to avoid merge conflicts). The model already
supports this on read/save; this section defines how files are *specified and
managed* through the authoring API. The guiding purpose is to **aid LLM
authoring and maintenance**: a fresh session must be able to learn the project
structure, where new tables belong, and why it's organized that way.

### Files
- A file is its path **relative to `xml/`** (e.g. `states/CO_dt.xml`), must end
  in `_dt.xml` (auto-suffixed), and may live in subdirectories.
- **Creating a table requires a file** — `table put` on a new table errors
  without a `file` (body field or `--file`). There is no default file; an
  unplaced table is an error.
- `file` is a table property: `table get` reports it; `put` with a changed
  `file`, or the `set-file` patch op, **moves** the table.
- **Empty files auto-delete** — when the last table leaves (move or delete),
  the API drops the file entry and removes the orphaned `.xml` and its Excel
  workbook on Save.

### Per-file number ranges (decision tables)
- Each DT file declares a `TABLE_NUMBER` range `[lo, hi]` at creation
  (`--range 3000-3500`).
- **Ranges must not overlap** across files. Gaps between ranges are fine
  (1000–2000 then 3000–3500). Files order by `lo` — which is load/exec order.
- Auto-assigned numbers stay in range: next = `max(in file) + 10`, starting at
  `lo` for an empty file; exceeding `hi` is an error (widen via `setrange` or
  use another file). Gaps of 10 remain so tables can be inserted.
- Explicit numbers (`number` on put, `set-number`) must be in the file's range
  and unique, else error.
- **Move renumbers into the target range** (next available slot). Safe because
  tables are referenced by name (`perform`), never by number — the number is
  pure ordering.
- `setrange <file> <lo-hi>` resizes; rejects overlap and rejects shrinking
  below a number already in use.

### EDD ordering
- The EDD stays a **single file** for now.
- EDD entities are **not numbered**. They serialize and load **sorted ascending
  by lowercase entity name** (deterministic canonical order).
- If the EDD is ever split, partition by **non-overlapping entity-name ranges**
  (e.g. `a–h`, `i–p`, `q–z`), files ordered by range — mirroring DT number
  ranges but on names.

### authoring-notes.md (LLM onboarding doc)
A committed journal at the project root, maintained so a fresh LLM session can
onboard. **Not** part of the Excel/XML round-trip (it is documentation, not
build-reproducible; not gitignored).

```markdown
# Authoring Notes

## Files                 ← API-managed: path [lo-hi] — purpose
- `constants_dt.xml` [1000-2000] — tax constants and rate tables
- `budgets_dt.xml`   [3000-3500] — budget computation
- `states/CO_dt.xml` [8000-10000] — Colorado tax (own file, avoid merge conflicts)

## Conventions          ← free-form; humans/LLM edit directly
- Compute constants in the constants file; never inline literals in logic tables
- One file per state under states/

## Change log           ← API-appended, dated, with the reason from each op
- 2026-06-15 — move `CO_Tax` → `states/CO_dt.xml` — "group state logic"
- 2026-06-15 — delete empty `old_dt.xml` (last table moved out)
```

- **Structural ops require a `reason`** — create file, move table, **delete
  table**, delete file, setrange. The reason is appended to the Change log — it
  is the knowledge the next LLM reads, not bookkeeping. In-file edits (cells,
  DSL, condition/action rows) need no reason.
- The **Files** section and **Change log** are API-managed (don't hand-edit);
  **Conventions** is free-form.
- `dtrules table note "..."` appends a free-form dated entry for design
  discussion.
- `dtrules table files` returns the structured map (path, range, purpose,
  tables) so an agent gets the structure without parsing markdown.

### Surface summary
- `table put <name> --file F --range LO-HI --reason R` (range/reason only when F
  is new); `file`/`number` also accepted in the JSON body.
- Patch ops: `set-file` (move), `set-range` (resize) — both take `reason`.
- `table delete <name> --reason R`, `table files`, `table note`.
- SDK: `Project.AddTable(name, file, reason)`, `MoveTable(name, target, reason)`,
  `DeleteTable(name, reason)`, `SetFileRange(file, lo, hi, reason)`, `Files()`,
  `AppendNote(text)`; `Save()` deletes orphaned files.

### Edge cases & backward-compat (resolved)
1. **Unranged (legacy) files** — ranges are **opt-in per file**. A file with no
   declared range auto-numbers (max+10) but is exempt from overlap enforcement;
   enforcement applies only among files that declare a range. Adopt a range
   later via `set-range`, which validates current tables fit and don't overlap.
   Existing projects (TaxReturn etc.) keep working untouched until they opt in.
2. **Move/put to a non-existent file** = file creation → requires `range` +
   `reason`.
3. **`range` given for an existing file** — rejected if it differs from the
   file's current range (use `set-range`); accepted if it matches.
4. **Deleting a table requires a `reason`.** If the delete empties the file, the
   file is auto-deleted and a Change-log entry records both, citing the delete's
   reason.
5. **Move auto-renumbers** into the target file's next in-range slot; `set-file`
   takes no number. To pick a specific number, follow with `set-number`.
6. **authoring-notes.md** is created on the first structural op if absent; lives
   at the project root (for flat layouts, the dir holding the `*_dt.xml`); the
   API rewrites only the **Files** and **Change log** sections and preserves all
   other content (Conventions, any prose) byte-for-byte.

<a name="how-we-got-here"></a>
## How we got here

| PR | What it did |
|----|-------------|
| #782 | Added `dtrules compile` (XML→postfix backfill, no Excel). |
| #789/#790 | Advisory pass + EDD symbols for compile. |
| #796 | "restore Excel-is-source-of-record": retrofitted the XML-writing surfaces with an Excel guard/refresh — but made it a **no-op when no manifest exists**, which is the hole this contract closes. |
| #817/#818 | Removed the `Action.Postfix` authoring override; postfix is regenerated from DSL. |

The pattern — a feature that broke the source-of-record contract, followed
by partial patches — is what this document replaces with a single rule.

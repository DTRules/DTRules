# elcheck — the EL authoring verification loop

Reports, for every row of every decision table in a project, whether its stored
postfix is what its EL DSL actually compiles to today.

This tool existed twice before as a throwaway in a session scratchpad and was
lost both times — the campaign plan of record still says "rebuild as needed".
It is in the repo now.

## Why it is needed

`postfix` is a compiled artifact of the EL DSL, never authored
([the authoring contract](../../docs/authoring-contract.md)). Two things break
that invariant, and neither is visible from any other tool:

- **Hand rows** — postfix with no DSL. `syncToXML` regenerates every postfix
  from its DSL, so the next `dtrules table put` on that table silently empties
  the row. Four sample projects lost rows this way.
- **Stale rows** — DSL and postfix that no longer agree, usually Java-era
  postfix the Go runtime cannot execute correctly.

A project is only safe to edit through the authoring API when `hand` and `err`
are both zero. **Check that before any `table put`.**

## Use

```bash
go run ./tools/elcheck -project sampleprojects/SyntaxTests
go run ./tools/elcheck -project sampleprojects/CorporateTax -show all
```

```
TOTAL ok=661 prose=33 resolved=0 hand=0 diff=0 err=0
```

| bucket | meaning |
|---|---|
| `ok` | postfix matches what the DSL compiles to — healthy |
| `prose` | comment-only row: compiles to nothing, stores nothing |
| `hand` | **postfix with no DSL — a `table put` will delete this row** |
| `diff` | DSL and postfix disagree — postfix is stale |
| `err` | the DSL does not compile — a `table put` will empty this row too |
| `resolved` | a candidate supplied via `-overrides` compiled byte-identical |

Rows are compiled per table through one compiler in table order, so a local
declared in a context row is in scope for the rows beneath it (#965) — the same
way `syncToXML` does it.

## Authoring a hand row

Propose EL, compile it in the table's real scope, and apply only on an exact
postfix match:

```bash
cat > /tmp/try.json <<'JSON'
{ "Syntax_Examples": { "action@2": "for all clients in reverse { set eligible = true; }" } }
JSON
go run ./tools/elcheck -project sampleprojects/SyntaxTests -overrides /tmp/try.json
```

**Keys are 1-based row positions** — `context 3`, `initial action 6`,
`condition@12`, `action@37`. Never the number written in the XML: `table get`
renumbers rows to their position on load, so a number key lands on a different
row on the far side of the API. That silently overwrote two live rows in
SyntaxTests before a positional diff caught it.

`seed_from_comments.py` bulk-generates candidates for projects whose hand rows
carry their EL in a comment, which is how 24 of SyntaxTests' 48 and 65 of
CorporateTax's 413 were recovered:

```bash
python3 tools/elcheck/seed_from_comments.py /tmp/seeded.json
go run ./tools/elcheck -project sampleprojects/CorporateTax -overrides /tmp/seeded.json
```

## After applying

Diff positionally, never by row number: same row count in and out, and no row
that had postfix left without it. Every near-miss in the repair campaign —
type erasure (#972), row deletion (#974), 312 initial actions deleted in
SyntaxTests — was invisible in tool output and showed only in the diff.

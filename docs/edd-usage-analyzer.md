# EDD Usage Analyzer

> Status: as of v1.13.0 (entity-stack-aware) — phases 1 and 2 shipped; phase 3 (cross-table + enumeration-bounded dynamic strings) tracked in [#776](https://github.com/DTRules/DTRules/issues/776).

The EDD usage analyzer answers two questions about a DTRules project:

1. Which declared EDD fields are **never read** by any rule? (candidate for removal)
2. Which fields are **only written**, never read? (probable bug — the rule sets the field but nothing downstream consumes it)

Surfaced through `dtrules review` as `INFO` advisories.

## Why a separate analyzer

Before v1.13.0, EDD usage was a single regex pass over DSL text:

```go
var identifierPattern = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b`)
```

This catches dotted references like `taxpayer.income`, but misses the case that drove the rewrite: **bare identifiers resolved against the entity stack at runtime**.

A rule like:

```
for all taxpayers
  taxpayer.is_self_employed and earned_income > 0
```

references `taxpayer.earned_income` indirectly. The `for all taxpayers` pushes `taxpayer` onto the entity stack; the bare `earned_income` resolves against the top of that stack at evaluation time. To the regex pass, only `taxpayer.is_self_employed` is visible, and `earned_income` looks unreferenced anywhere — so the EDD declaration for `taxpayer.earned_income` is flagged as **unused** even though it's the most-read field in the table.

The downstream staking project surfaced ~97 such false-positive `unused` warnings before the rewrite landed. After the rewrite: ~5 genuine.

## What it tracks

The analyzer walks every `*_dt.xml` under the project's `xml/` directory. For each DSL fragment it threads an **entity stack** mirroring what the runtime does:

| Construct | Push effect (duration) |
|---|---|
| `for all <field>` block | type of the iterated entity, for the duration of the block |
| `for all <field> where <pred>` | same, plus the `where` clause is in scope of the iterated entity |
| `for first of <field> where <pred>` | same |
| `for first in <field> where <pred>` | same |
| `using <entity> { ... }` block | named entity, for the duration of the block |
| `for each <entity> in <array>` block | iterated entity, for the duration of the block |
| `for all <type> entities [where ...]` | iterated entity type (resolved via the EDD's `owns` declaration) |

At each reference site, bare identifiers resolve against the topmost entity. Dotted identifiers (`owner.field`) bypass the stack and resolve directly.

References are split into **read** and **write** kinds:

- `set <target> = <expr>` and `set <target> = ...` flavors record `<target>` as a **write**, every identifier in `<expr>` as a **read**.
- Plain identifier references in conditions and non-set actions are all **reads**.
- `for all <field>` records `<field>` as a **read** (the iterator must read the collection to iterate it).

## Output categories

| Category | Meaning |
|---|---|
| `INFO unused EDD field` | Declared in EDD, no read or write reference. Safe to remove, modulo external mapping/JSON consumers. |
| `INFO write-only EDD field` | Set somewhere, never read anywhere. Likely a bug — either the read got deleted, or the set is doing nothing. |

Both surface through `dtrules review` and the build pipeline's advisory channel.

## What it still misses (phase 3 backlog)

- **Cross-table references**: a `perform <Table>` call descends into the called table's context. The current pass treats each `*_dt.xml` independently. A field read only inside a table called via `perform` from another table is currently visible at the file granularity, but the **caller's** entity-stack push doesn't carry into the callee's reference resolution.
- **Dynamic dispatch**: `perform table named (<string-expression>)` (shipped v1.12.0) computes the target table name at runtime. The analyzer sees the string expression but cannot enumerate possible values. [#776](https://github.com/DTRules/DTRules/issues/776) proposes EDD-declared enumeration bounds so authors can give the analyzer the bounded set.
- **Postfix-only blocks**: the strict hand-coded-postfix gate (v1.11.0) means new code can't introduce hand-edited postfix anyway, but legacy fixtures with raw postfix and no EL DSL fall outside the analyzer's view.

When phase 3 lands, the categories grow to:

| Category | Meaning |
|---|---|
| `Used` | Reference visible somewhere reachable. |
| `Unused` | No reference anywhere. |
| `Possibly used` | Referenced inside an unbound-string dispatch target — the analyzer can't prove unreached. |
| `Write-only` | Set, never read. |

Unbound dynamic dispatch (no enumeration declared) escalates from a warning to an error in `dtrules build --require-review`.

## Running it

```bash
dtrules review <project>           # invokes the analyzer alongside every other check
dtrules build <project>            # surfaces the same warnings on the advisory channel
```

The analyzer is also accessible programmatically:

```go
import "github.com/DTRules/DTRules/pkg/dtrules/analysis"

warnings, err := analysis.AnalyzeEDDUsage("/path/to/project/xml")
```

Each `EDDWarning` carries:

```go
type EDDWarning struct {
    Field   string // "entity.attribute"
    EddFile string // source EDD file name
    Reason  string // human-readable explanation
}
```

## Related

- [#776](https://github.com/DTRules/DTRules/issues/776) — full design including phase 3 (cross-table + enumeration bounds)
- [#778](https://github.com/DTRules/DTRules/pull/778) — phase 2 rollout
- [#777](https://github.com/DTRules/DTRules/pull/777) — phase 1 rollout
- `pkg/dtrules/analysis/edd_unused.go` — implementation
- `dtrules docs edd` — embedded EDD reference (entity/field declaration syntax)

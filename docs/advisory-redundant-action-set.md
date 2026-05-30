# Redundant Action-Set Column Advisory

> Status: as of v1.14.6 (issue [#797](https://github.com/DTRules/DTRules/issues/797)).

A decision-table column is **redundant** when it reaches the same action set as an earlier column. The runtime would produce the same output if that column were removed and its inputs fell through, or if it were collapsed into the earlier column with all conditions set to `*`.

This is a **stronger check** than the cell-level redundant-condition advisory ([#762](https://github.com/DTRules/DTRules/issues/762), [#794](https://github.com/DTRules/DTRules/issues/794)): the cell-level check flags individual `Y`/`N` entries that are forced by other rows; this check flags entire **columns** whose behavior is already covered.

## The check, in one paragraph

After the table is compiled into its decision tree, walk the tree and group columns by the **ordered tuple** of action numbers they reach. For each group, keep the smallest column number ("first to introduce this action set") and flag every other column as redundant ("same action set already covered by column N").

## Why "ordered tuple"

The runtime executes actions in the order they're listed, and that order is observable: audit-trail line ordering, sequence dependencies between sets and reads, side effects between `set` and a read of the same field. So:

- Columns reaching `[3, 5, 7]` and `[3, 5, 7]` → same action set.
- Columns reaching `[3, 5, 7]` and `[7, 5, 3]` → **different** action sets, no warning.

This is conservative — sometimes order doesn't matter and the columns really are equivalent, but the analyzer can't prove that without modeling each action's read/write effects, so it errs on the side of "you said order matters, I'll take you at your word."

## What gets reported

```
WARN <table>: column 4 reaches the same action set as column 2 — consider
collapsing into a single column (all conditions = '-') or removing if the
inputs should fall through downstream
```

The output points at the **later** column (the redundant one) and names the **earlier** column (the one already covering this behavior).

## What it doesn't catch

- **Unreachable columns**: columns that no tree leaf maps to. These are caught by a separate `unreachable column` warning — they're skipped here because "doesn't reach this action set" doesn't apply when they don't reach anything at all.
- **Columns with empty action sets**: a column with no action numbers is the table's no-op column, covered by the `no-op column` warning.
- **Cross-table redundancy**: two different tables having redundant columns relative to each other isn't part of this check; the analyzer is per-table.

## Suggested remediation

Two reasonable fixes:

**Collapse into a catch-all.** If columns 2 and 4 are both producing action set `[3, 5, 7]`, the table is implicitly saying "any input matching either column 2 or column 4 should run these actions." Often the cleaner expression is a single column with all conditions = `-` (always matches), or columns 2 and 4 merged with their conditions OR'd via the cell-level entries.

**Remove and let it fall through.** If the redundant column was a defensive belt-and-braces ("just in case the earlier column doesn't catch this"), removing it and trusting the earlier column tends to make the table clearer. The advisory is a prompt to decide which intent the column was serving.

## Running it

The check runs as part of `decisiontable.Analyze`, which is invoked by:

```bash
dtrules build <project>            # advisory pass on every build
dtrules review <project>           # full review including this check
```

It also surfaces in the cmd/dtrules MCP advisory tool surfaces.

## Related

- `pkg/dtrules/decisiontable/tree_analysis.go:checkRedundantActionSetColumns` — implementation
- [#762](https://github.com/DTRules/DTRules/issues/762) — cell-level redundant-condition check
- [#794](https://github.com/DTRules/DTRules/issues/794) — leave-one rule capping the cell-level check at K-1
- `dtrules docs decision-tables` — embedded decision-table reference

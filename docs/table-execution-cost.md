# What one decision table costs to execute

`docs/performance.md` measures whole tax returns against user-facing latency
targets. This measures the unit underneath: **one small table, executed once.**

That number decides which work can move from Go into rules. A table run once
per request can cost 50 µs and nobody notices. A table run inside an
enumeration — the `expect` primitive #980 named as a later candidate,
exhaustive expected-value analysis, Monte-Carlo over a rule set — pays it on
every inner call, and the ceiling arrives fast (#1025).

## The measurement

The Cribbage sample (#984) is the subject. `Score_Hand` is about as small as a
real table gets — five performed tables scoring entities that three
combinatorial generators materialized — and `Score_Play` (#1023) is smaller
still. Both have compact Go equivalents, so the comparison is not rigged by an
expensive reference implementation.

```
go test ./pkg/dtrules/ -run '^$' -bench Cribbage -benchmem
```

On an Intel Core Ultra 9 275HX, Go 1.24:

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkCribbageScoreHand` | 57,000 | 42,139 | 791 |
| `BenchmarkCribbageScoreHandSetup` | 2,752 | 3,584 | 50 |
| `BenchmarkCribbageScorePlay` | 10,388 | 5,216 | 111 |

`…Setup` builds the five card entities, the hand, and its work arrays without
executing anything. Subtracting it gives the engine's own share:

- **Score_Hand executes in ~54 µs**, allocating ~38 KB across ~740 allocations.
- Host-side entity construction is **~5%** of the total. The cost is the table,
  not the embedding boundary.

## What the two tables say together

The interesting result is the 5.5× gap between them. Both run the same
machinery — contexts, condition dispatch, performed tables, policy statements
— over hands of nearly the same size. What differs is how many entities the
generators materialize:

- `Score_Hand` calls `subsets` on five cards: **31 combo entities**, each with
  its own members array, all garbage a moment later.
- `Score_Play` calls `suffixes` on a three-card stack: a couple of windows.

So the dominant cost is **materialization, not table overhead**. A small table
whose generators emit little runs in ~10 µs. That reframes the ceiling: the
limit on enumerate-and-average is not "a table call costs 45-57 µs" but "every
structure you ask a generator to materialize costs roughly 1.5 µs and 1.2 KB,
and subset enumeration asks for 2^n of them."

Two consequences worth keeping in view:

1. **Reuse is the lever.** The work arrays and their entities are rebuilt for
   every execution because the generators append. A pooled or resettable
   destination would attack ~95% of the cost directly.
2. **Prefer the generator that materializes least.** `suffixes` exists because
   pegging needed order-dependent structure; it also happens to be the cheap
   shape. Where a rule can be expressed over groups or runs rather than the
   full power set, it will be an order of magnitude cheaper.

## What this does not say

It does not say tables should replace numeric kernels. An exhaustive
expected-value loop — 683,790 scorings for one cribbage deal — is not policy;
no analyst will ever edit "average over 46 starters", and it belongs in Go at
190 ns per scoring. The point of measuring is to know where that boundary
actually falls, and to notice if it moves.

## Keeping it honest

Re-run the benchmark rather than quoting this table. Absolute numbers move with
hardware and Go version; the ratios — setup versus execution, `Score_Hand`
versus `Score_Play` — are what carry.

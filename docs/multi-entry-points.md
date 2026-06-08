# Multiple Entry Points Against a Single Session

> Status: as of v1.15.0 — pattern empirically verified, behavior pinned by `TestMultipleEntryPointsAgainstSameSession`.

A DTRules project usually has more than one decision table. Sometimes
you want to load the project once, populate a session with input data,
and then run **different decision tables as different entry points**
against that same loaded state — without re-loading rules or re-creating
the session for each table.

This is supported directly. `RSession.Execute(tableName)` is the entry
point selector: call it once for each table you want to run. The
session's entity stack and data dictionary persist across calls, so a
mutation made by the first table is visible to the second.

## When to use it

Three common shapes:

- **Multi-pass evaluation** — one table validates inputs and tags
  errors; a second computes a result only on records that passed; a
  third produces a summary. Each is its own entry point.
- **Read-then-write workflows** — `Compute_Eligibility` reads inputs
  and decides; `Generate_Audit_Trail` then reads the decision and
  populates an audit log entity.
- **Per-request branching** — a single bound session is reused to
  answer different "questions" against the same loaded data:
  `Check_Eligibility` for one HTTP request, `Compute_Risk` for the
  next, against the same client record.

If your tables are wired so that one table `perform`s the next, you
don't need this pattern — the call graph handles it for you. This page
is for the case where the caller (your Go code, your service, your
CLI) decides which entry point to invoke.

## The API

```go
import "github.com/DTRules/DTRules/pkg/dtrules/session"

rs := session.NewRuleSet("MyProject")
rs.LoadEDD(eddReader)
rs.LoadDecisionTables(dtReader)

sess, err := rs.NewSession()
if err != nil { /* handle */ }
rsess := sess.(*session.RSession)

// Populate data once.
entity, _ := rsess.CreateEntity(dtrules.GetRName("client"))
entity.Put(dtrules.GetRName("age"), dtrules.GetRIntegerValueFromInt(20))
rsess.GetState().EntityPush(entity)

// Entry point #1.
if err := rsess.Execute("Check_Eligibility"); err != nil {
    /* handle */
}

// Entry point #2 — same session, same entity, same data.
// State written by Check_Eligibility is visible here.
if err := rsess.Execute("Compute_Risk"); err != nil {
    /* handle */
}
```

`Execute(tableName)` is just a name-keyed lookup in the entity factory
that returns the compiled `RDecisionTable` and invokes its `Execute(state)`
method. There is no special "entry point" registration — every loaded
decision table is callable by name.

## What persists across calls

| State element | Persists? |
|---|---|
| Loaded entity definitions (EDD) | Yes — owned by the `RuleSet`, not the session. |
| Loaded decision tables | Yes — same. |
| Created entities (via `CreateEntity`, mapping loader, etc.) | Yes — they live in the session's `RDTState`. |
| Entity field values written by the first table | Yes — `set entity.field = ...` mutations are visible to the second. |
| Entity stack pushes from your Go code | Yes — `state.EntityPush(...)` outside any table call survives across `Execute` calls. |
| Entity stack pushes from inside a table's `<contexts>` | Pushed at table entry, popped at table exit. They do **not** leak across calls. |
| Data stack | Empty between calls. The runtime asserts the data stack is balanced at table boundaries. |
| Trace events | Accumulate if a trace tracker is attached; reset by attaching a fresh tracker. |

## ExecuteAt — push an entity, run, pop

For the common case "run table T with entity E as the current context,"
there's a convenience that pushes the entity before calling `Execute`
and pops it after (even on error):

```go
err := rsess.ExecuteAt("Compute_Risk", "client")
```

This is equivalent to:

```go
entity, _ := rsess.GetState().FindEntity(dtrules.GetRName("client"))
rsess.GetState().EntityPush(entity)
err := rsess.Execute("Compute_Risk")
rsess.GetState().EntityPop()
```

Use `ExecuteAt` when each entry point should run with a single
specific entity at the top of the stack and you don't want the manual
push/pop boilerplate.

## Errors and partial state

If the first `Execute` errors mid-way, mutations performed before the
error point have already been applied to the session. The session is
**not** rolled back automatically. Two reasonable patterns:

- **Snapshot, run, discard on error.** Before the first `Execute`,
  copy the entity values you care about. On error, restore them.
- **Idempotent tables.** Author the tables so re-running them
  produces the same result regardless of partial prior runs. This is
  the more common pattern in tax / eligibility domains where
  `set <output> = <pure_function_of_inputs>` is the dominant action.

## Pitfalls

- **Don't reuse a session across unrelated tenants.** A session
  carries every entity ever created in it. If you're serving multiple
  tenants from one server, give each request its own session via
  `rs.NewSession()` — the `RuleSet` is the cheap-to-share piece, the
  `RSession` isn't.
- **Watch the entity stack depth.** Tables that push and forget to
  pop are bugs. The runtime checks balance at table boundaries, so
  this surfaces as an error rather than silently corrupting state —
  but the first `Execute` after a broken table will see a deeper
  stack than expected.
- **`Execute("UnknownTable")` returns an error**, not a panic. Catch
  it on the caller side; treat undefined-table as a configuration bug
  worth surfacing to operators.

## Related

- [`dtrules docs cli`](../cmd/dtrules/docs.go) — top-level CLI surface.
- [`dtrules docs authoring`](../cmd/dtrules/docs.go) — programmatic SDK for editing projects.
- [`dtrules docs sdk`](../cmd/dtrules/docs.go) — embedding the engine into a Go application.
- `pkg/dtrules/multi_entry_test.go` — the regression test that pins this contract.

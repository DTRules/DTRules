# `first pass` Predicate

> Status: as of v1.12.0 (issue [#764](https://github.com/DTRules/DTRules/issues/764)).

The EL predicate `first pass` returns `true` exactly when the **innermost active loop** is on its first iteration, `false` otherwise. It's the canonical way to emit a one-time setup action or skip a separator before the first iteration of a `for all` / `for first of` / `foreach` block.

## Shape

```
for all customers
  if first pass then
    set total = 0;
  endif
  set total = total + customer.balance;
```

The `if first pass` guards `set total = 0` — it runs only on the first iteration; subsequent iterations see the accumulated `total` carried forward.

## What "innermost active loop" means

Every loop the runtime enters (`for`, `forr`, `forall`, `forallr`) pushes a fresh iteration counter onto an internal stack:

- The counter starts at 0 before the body's first execution.
- The runtime increments it after each successful body run.
- The counter pops when the loop exits.

`first pass` reads the topmost counter. So:

- Outside any loop: stack is empty → `first pass` is `false`. ("First pass of nothing" doesn't have a defensible meaning, and the alternative — `true` — would surprise rules that read `first pass` outside an obviously-looping context.)
- Inside one loop, first iteration: `true`.
- Inside nested loops, first iteration of the inner one: `true` regardless of outer state. The outer loop's iteration count is on the stack below but `first pass` reads the **top**.
- Inside nested loops, second iteration of the inner one (within the outer's first iteration): `false`.

## Common patterns

### Per-iteration accumulator initialization

```
for all line_items
  if first pass then
    set running_total = 0;
  endif
  set running_total = running_total + line_item.amount;
```

Idiomatic. The `if first pass` is the explicit "do this once before the loop body proper" hook.

### Skipping a separator before the first emitted value

```
for all states
  if first pass then
    set output = "";
  else
    set output = output + ", ";
  endif
  set output = output + state.abbr;
```

Produces `CA, NY, TX` rather than `, CA, NY, TX`.

### Nested loops

```
for all parents
  if first pass then
    set summary = "Parents:";
  endif
  for all parent.children
    if first pass then
      set summary = summary + " (";  // ← fires per parent
    endif
    set summary = summary + child.name;
  endfor
endfor
```

The inner `if first pass` fires once per parent (the inner loop's first iteration), not once for the whole nested walk. The outer `if first pass` fires once total, before the first parent.

## Postfix lowering

`first pass` compiles to the niladic `firstpass` operator. Source:

```
if first pass then
  set total = 0;
endif
```

Postfix:

```
firstpass { 0 cvi /total xdef } if
```

The runtime's `opFirstPass` pushes `true` or `false` based on `state.IsFirstLoopPass()`, which is itself just a peek at the iteration stack.

## Edge cases

- **Empty loop body**: if the loop has no iterations at all (`for all <empty-collection>`), the body never executes and `first pass` is never evaluated. There's no "you skipped me on iteration 0" surprise.
- **Conditional bodies**: `first pass` measures **loop iterations**, not body execution count. A loop body wrapped in `if some_condition then ... endif` still has `first pass` true only on the first iteration of the loop, regardless of whether `some_condition` was true that pass.
- **Recursive `perform` calls**: a `perform <Table>` from inside a loop body does not push a new iteration counter — `first pass` inside the called table reads the **caller's** topmost loop counter. This is usually what authors want; if you need a "first time entering this table" predicate, that's a separate concept (and not what `first pass` provides).

## When to reach for it

Whenever the alternative would be a sentinel field (`if not initialized then ... set initialized = true`). The sentinel approach requires:

- A field in the EDD for the flag
- A reset somewhere outside the loop
- Discipline that nothing else flips the flag

`first pass` is local to the loop and resets automatically when the loop exits.

## Related

- `pkg/dtrules/operators/control.go:opFirstPass` — runtime op
- `pkg/dtrules/compiler/el/postfix_emitter.go:VisitBoolFirstPass` — EL → postfix
- `dtrules docs el` — embedded EL reference
- `pkg/dtrules/operators/firstpass_test.go` — exhaustive regression suite covering nested loops and edge cases

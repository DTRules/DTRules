# `for all <type> entities` Loop

> Status: as of [#703](https://github.com/DTRules/DTRules/issues/703).

The EL action `for all <type> entities` iterates over **all instances of an entity type**, resolved through the EDD's ownership declaration rather than by walking a named array. It's the right form when you want "every customer" or "every account" without having to find the right `<owner>.<field>` to point at.

## Shape

```
for all customer entities
  set customer.flagged = false;
```

vs the array-walking form:

```
for all client.customers
  set customer.flagged = false;
```

Both iterate `customer` entities. The difference: the type form discovers the source collection via the EDD; the array form names it explicitly.

## With a `where` filter

```
for all customer entities where customer.balance > 0
  set customer.dunning_eligible = true;
```

Same semantics as `for all <field> where`, with the source collection resolved by type.

## How the EDD resolution works

Every entity type in the EDD declares its owner via an `owns` relationship:

```xml
<entity name="client">
  <field name="customers" type="array" subtype="customer" owns="true"/>
</entity>

<entity name="customer">
  <field name="balance" type="double"/>
  <!-- ... -->
</entity>
```

`client.customers` is declared with `owns="true"`. When the compiler sees `for all customer entities`, it asks the EDD: "which field owns `customer`?" and finds `client.customers`. The DSL gets rewritten to that path before postfix emission.

## Postfix lowering

`for all customer entities` lowers to the same shape as `for all client.customers`:

```
dup client.customers forall pop
```

For the `where` form, the body is wrapped in the standard predicate-execute pattern that `forallWhere` uses, with the resolved owner-field substituted for the array.

This means the runtime cost is identical — the type form is pure syntactic sugar over the resolved EDD path.

## When the resolution fails

If the EDD has no `owns="true"` field for the named type, the visitor emits nothing and the surrounding statement collapses. In a strict-loader build, this surfaces upstream as a compile error from the advisory pass; in non-strict modes it would be a silent no-op (one of the silent-failure classes [#803](https://github.com/DTRules/DTRules/issues/803) hardened against).

Typical fix: add the `owns="true"` declaration to the field that holds the collection. The compiler's error message names the type that couldn't be resolved.

## Why use it

The array-walking form (`for all client.customers`) requires the author to know:

- Which entity holds the collection
- What the collection field is called

The type form requires only the entity type. For tables that operate at the "for every X in the system" level, that's the clearer expression — readers don't have to chase the owner field, and the rule keeps working if the EDD owner gets restructured (renamed field, moved to a different owner) as long as the new owner declares `owns="true"`.

For tables that are scoped to a specific subset (`for all client.preferred_customers`), the array-walking form remains the right choice — it makes the scope explicit.

## Interaction with the entity stack

`for all customer entities` pushes `customer` onto the entity stack for the duration of the body, the same way `for all client.customers` does. Bare identifiers inside the body resolve against the iterated `customer`:

```
for all customer entities
  if balance > 1000 then          // balance → customer.balance
    set tier = "premium";          // tier    → customer.tier
  endif
```

The [EDD usage analyzer](edd-usage-analyzer.md) tracks this push so the bare-name references count as `customer.*` for usage purposes.

## Related

- `pkg/dtrules/compiler/el/postfix_emitter.go:VisitForallTypeEntities` and `:VisitForallTypeEntitiesWhere` — the EL → postfix lowering
- `pkg/dtrules/compiler/el/postfix_emitter.go:resolveEntitiesCollection` — the EDD-ownership lookup
- [`edd-usage-analyzer.md`](edd-usage-analyzer.md) — entity-stack tracking
- `dtrules docs edd` — embedded EDD reference (entity/field declaration syntax)
- `dtrules docs el` — embedded EL reference

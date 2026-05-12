# EL grammar gaps surfaced by the hand-coded-postfix gate

After the strict per-element hand-coded-postfix policy landed
(`feat(decisiontable): strict per-element hand-coded-postfix block`,
`feat(decisiontable): refuse to execute tables with hand-coded postfix`),
266 elements across the TaxReturn project were flagged as having
postfix without matching EL DSL. Authoring rounds brought the count
down to **10 elements across 4 distinct grammar gaps**. This document
catalogues those gaps so they can be closed by extending the EL
compiler (or by changing the operator surface so the missing pattern
isn't needed).

Each gap below names the EL syntax that doesn't exist, shows a real
example postfix that needs it, and proposes the EL surface that would
let the cleanup finish.

---

## Gap 1: Entity creation + entity-stack manipulation in action bodies

**Affected elements (4):**
- `Build_State_Tax_Result_For_Period` action 1 (resident-state branch)
- `Build_State_Tax_Result_For_Period` action 2 (non-resident-state branch)
- `Dispatch_State_Tax` initial_action 1 (synthesize state_tax_result when
  state_tax_results is empty)
- `Dispatch_State_Tax` action 10 (the MO/HI combined synthesis after
  whichever XX_Tax fires)

**Sample postfix (Build_State_Tax_Result_For_Period action 1):**

```
/state_tax_result createentity /st_result xdef
state_period.state_code /st_result.state_code xdef
true /st_result.is_resident xdef
state_period.allocated_income /st_result.state_source_income xdef
state_period.allocated_income /st_result.state_agi xdef
st_result.state_withholding state_period.allocated_withholding f+ /st_result.state_withholding xdef
"  Resident state: " state_period.state_code strconcat " (allocated income $" strconcat state_period.allocated_income cvs strconcat ")" strconcat job.audit_trail swap addto
st_result job.state_tax_results swap addto
```

**Why EL can't express this today:**

EL `set X = Y` writes to an existing field on an entity already on the
entity stack. There's no syntax for:

- constructing a new entity of a named type (`createentity`),
- pushing it onto the entity stack so its attributes are addressable
  with bare names (`entitypush` / `entitypop`),
- binding the new entity to a local name (`/st_result xdef` — `xdef`
  here is being abused as a local-variable bind; properly it expects
  the bound name to be an attribute on the entity stack).

**Proposed EL surface:**

```
create state_tax_result as st_result {
    state_code         = state_period.state_code
    is_resident        = true
    state_source_income = state_period.allocated_income
    state_agi          = state_period.allocated_income
    state_withholding  = state_withholding + state_period.allocated_withholding
};
add st_result to job.state_tax_results
```

Compiles to: push a fresh `state_tax_result` instance onto the entity
stack as the implicit context for the assignment block, bind it to
`st_result` for the rest of the action, then `addto`.

**Workaround that exists today:** none in EL. The dispatch synthesis
has to stay as postfix until this gap is closed. Either close the gap
or move the synthesis into a Go-side custom operator
(`/synthesize_state_tax_result performtable`) registered in the
operator registry so the postfix can be replaced with a single
operator call that EL *can* emit via `perform Synthesize_State_Tax_Result`.

---

## Gap 2: `ifelse` in context bodies (two-branch local-bind)

**Affected elements (1):**
- `SC_Tax_Brackets` context 1

**Sample postfix:**

```
{ sc_standard_deduction_joint }
{ sc_standard_deduction_single }
job.filing_status "MFJ" streq job.filing_status "QSS" streq or
ifelse
allocate
result.agi 0 local@ f- 0 fmax /result.sc_taxable_income xdef
execute
deallocate pop
```

The context picks the joint or single standard deduction by filing
status, allocates a local frame to hold the chosen value, computes
`result.sc_taxable_income = max(result.agi - chosen_deduction, 0)`, and
deallocates. The comment in the source even notes: *"Hard: EL context
grammar lacks inline if/else for branching on filing_status; postfix
uses ifelse with allocate/execute/deallocate stack manipulation. Postfix
retained as source of truth."*

**Why EL can't express this today:**

EL has `if X then Y` (single-arm), and the bracket conditions of the
decision table form the multi-armed equivalent. But for **context**
bodies — which run once before the decision tree — there's no
two-branch conditional construct, and no way to introduce a local
variable bound to one of two expressions.

**Proposed EL surface:**

```
if job.filing_status is equal to "MFJ" or job.filing_status is equal to "QSS" then
    set local sc_std = sc_standard_deduction_joint
else
    set local sc_std = sc_standard_deduction_single;
set result.sc_taxable_income = the maximum of (result.agi - sc_std) and 0
```

`set local X = Y` introduces a context-scoped local that's available
to subsequent statements in the same context body — equivalent to
postfix `allocate ... local!` / `local@`.

**Workaround that exists today:** lift the filing-status branch to a
condition+action pair on a *helper* decision table. The bracket table
becomes single-context with a pre-computed `result.sc_taxable_income`.
Already used elsewhere in the project (CO/CA pre-pipeline tables) and
works, but requires per-table refactor.

---

## Gap 3: Dynamic table dispatch via name strconcat + execute

**Affected elements (1):**
- `Dispatch_State_Tax` action 7 (the "fallback" branch that picks
  `Calculate_<state>_Tax` from `job.state`)

**Sample postfix:**

```
{ state_period.state_code /state_code xdef }
{ job.state /state_code xdef }
/state_period isdefined
ifelse
"Calculate_" state_code strconcat "_Tax" strconcat /table_name xdef
{ "  State tax calculation not yet implemented for " state_code strconcat job.audit_trail swap addto }
{ table_name execute }
table_name lookupEntity isnull
ifelse
```

The dispatch resolves a table name by string concatenation, looks it up
at runtime, executes if found, logs if not. Three EL gaps stack here:

1. `isdefined` test on an entity-stack name (the `/state_period
   isdefined ifelse` guard that picks state_period.state_code vs
   job.state)
2. dynamic table reference from a string (`table_name execute` where
   table_name is `"Calculate_" + state_code + "_Tax"`)
3. `lookupEntity` to test existence before executing

**Proposed EL surface:**

```
perform table named ("Calculate_" + job.state + "_Tax")
    otherwise add "  State tax calculation not yet implemented for " + job.state to job.audit_trail
```

`perform table named (<string>)` resolves at runtime; the `otherwise`
clause is the fallback when the named table doesn't exist.

**Workaround that exists today:** explicit per-state dispatch columns
(what the project does for the 19 states with implementations) — works
but doesn't scale and was the original reason the loader added this
fallback.

---

## Gap 4: Duplicate `<condition_number>` values

**Affected elements (2):**
- `Dispatch_State_Tax` condition 10 (two `condition_details` blocks
  both with `<condition_number>10</condition_number>`, one for MO and
  one for HI — relic of the pre-refactor dispatch)

**Why this is in this document:** the authoring API's
`update-condition-dsl` patch op locates conditions by `condition_number`.
When two condition_details share a number, the patch hits only the first
one; the second is unreachable. The legacy dispatch table is uniquely
afflicted (43 conditions across 41 numbers).

**This isn't an EL grammar gap** — it's a data-integrity issue in the
underlying XML. The authoring API should either reject duplicate
numbers at load time or expose a position-based update operation. The
fix that unblocks the cleanup is to re-number the duplicate so each
condition_details has a unique `condition_number`.

**Proposed authoring-API surface:**

```bash
# Re-number a condition by position rather than current number
echo '{"op":"renumber-condition","position":42,"new_number":43}' \
    | dtrules table patch Dispatch_State_Tax --project .
```

Or: load-time validation rejects duplicate numbers with an error rather
than silently letting both into the loader.

---

## Gap 5 (housekeeping): empty `<condition_number>` / `<action_number>` tags

**Affected elements (2):**
- `Calculate_Educator_Expenses` condition with empty `<condition_number></condition_number>`
- `Calculate_Educator_Expenses` action with empty `<action_number></action_number>`

The XML record has empty number tags — they parse as integer 0 (the
audit reports "condition 0", "action 0"). The authoring API can't
target a zero-numbered row because the schema requires
`{"minimum": 1}`. The runtime gate flags them, but the table itself
won't execute (the conditions-without-numbers wouldn't compile into a
decision tree).

This is a one-off data-integrity issue in the XML file, fixable by
setting the numbers to 1 in the source XML (which the round-trip work
in `pkg/dtrules/excel` will then preserve). Not strictly an EL grammar
gap, but it lands in the gate's "10 remaining" bucket because every
hand-coded-postfix element counts.

---

## Summary

| Gap | Affected elements | EL surface needed |
| --- | --- | --- |
| Entity creation + entity-stack manipulation | 4 | `create T as alias { … }`, `add alias to collection` |
| `ifelse` + locals in context bodies | 1 | `if … then … else …; set local X = Y` |
| Dynamic table dispatch | 1 | `perform table named (<string>) otherwise …` |
| Duplicate condition numbers (data, not grammar) | 2 | API: `renumber-condition`, or load-time validation |
| Empty condition/action numbers (data, not grammar) | 2 | Fix in source XML; round-trip already preserves |
| **Total** | **10** | |

Closing gaps 1 + 2 + 3 would let the runtime hand-coded-postfix gate
report **zero** violations across the TaxReturn project; gaps 4 + 5
are pre-existing data bugs that surface incidentally.

Until then, the affected tables refuse to execute at runtime (correctly,
per the strict policy). Tests that depend on those code paths
(`Dispatch_State_Tax` is in the path of every state tax test;
`Build_State_Tax_Result_For_Period` is reached when state_periods is
non-empty; `SC_Tax_Brackets` for SC tests; `Calculate_Educator_Expenses`
when any taxpayer is flagged as an educator) will fail — which is the
intended behaviour of the gate.

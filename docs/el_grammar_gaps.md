# EL grammar gaps surfaced by the hand-coded-postfix gate

After the strict per-element hand-coded-postfix policy landed
(`feat(decisiontable): strict per-element hand-coded-postfix block`),
266 elements across the TaxReturn project were flagged. Authoring
rounds brought the count down to **10 elements**. This document sorts
the remaining 10 into:

- **Real EL grammar gaps** — the postfix can't be expressed by any
  combination of existing EL syntax + DTRules primitives, and closing
  the gap requires extending the EL compiler.
- **Restructuring tasks** — the postfix *can* be expressed today, but
  the table is laid out wrong (logic in a context body that belongs in
  a condition row, etc.). No grammar change needed.
- **Data-integrity issues** — broken values in the source XML.

DTRules' control-flow primitive is the decision table itself —
conditions + actions + columns. EL action bodies do not have inline
`if … then … else …`, and that's intentional: branching is a
table-author concern. The right cleanup for most of these is to use
the primitives properly, not to extend the grammar.

---

## Real EL grammar gaps (5 elements)

### Gap A: Entity creation in action bodies

**Affected elements (4):**
- `Build_State_Tax_Result_For_Period` action 1 (resident-state branch)
- `Build_State_Tax_Result_For_Period` action 2 (non-resident-state branch)
- `Dispatch_State_Tax` initial_action 1 (synthesize state_tax_result
  when state_tax_results is empty)
- `Dispatch_State_Tax` action 10 (MO/HI combined synthesis)

**Sample postfix (Build_State_Tax_Result_For_Period action 1):**

```
/state_tax_result createentity /st_result xdef
state_period.state_code      /st_result.state_code xdef
true                         /st_result.is_resident xdef
state_period.allocated_income /st_result.state_source_income xdef
state_period.allocated_income /st_result.state_agi xdef
st_result.state_withholding state_period.allocated_withholding f+ /st_result.state_withholding xdef
st_result job.state_tax_results swap addto
```

**Why EL can't express this today:** `set X = Y` writes to an existing
field on an entity already on the entity stack. There is no EL syntax
for constructing a new entity of a named type, populating its fields,
and pushing it onto a collection.

**Proposed EL surface:**

```
create state_tax_result as st_result {
    state_code           = state_period.state_code
    is_resident          = true
    state_source_income  = state_period.allocated_income
    state_agi            = state_period.allocated_income
    state_withholding    = state_withholding + state_period.allocated_withholding
};
add st_result to job.state_tax_results
```

Compiles to: push a fresh `state_tax_result` instance onto the entity
stack (so the bare field names in the block resolve against it), run
the assignments, bind it to the local name `st_result`, pop the
entity stack, append to the named collection.

**Workaround until then:** none in EL. The synthesis stays as
hand-coded postfix and the affected tables refuse to execute.
Alternative: register a custom operator like `/build_state_tax_result`
in `pkg/dtrules/operators/` and have the action call it via
`perform <helper_table>` — but EL `perform` invokes decision tables,
not bare operators, so this only works if a wrapper helper table is
added (which itself still has the same gap in its action body).

---

### Gap B: `perform` with a string-valued table name

**Affected elements (1):**
- `Dispatch_State_Tax` action 7 (the "fallback" branch that picks
  `Calculate_<state>_Tax` based on `job.state`)

**Sample postfix:**

```
{ state_period.state_code /state_code xdef }
{ job.state               /state_code xdef }
/state_period isdefined
ifelse
"Calculate_" state_code strconcat "_Tax" strconcat /table_name xdef
{ "  State tax calculation not yet implemented for " state_code strconcat job.audit_trail swap addto }
{ table_name execute }
table_name lookupEntity isnull
ifelse
```

**Why EL can't express this today:** EL `perform TableName` requires a
literal identifier — the table name is fixed at compile time. There is
no syntax for `perform <expression>` where the expression evaluates to
a string at runtime.

**Proposed EL surface:**

```
perform table named ("Calculate_" + job.state + "_Tax")
```

If the named table doesn't exist at runtime, the call raises an
"undefined table" error — same as any other unknown-table reference.
There is **no** "otherwise" clause. Conditional fallback ("log if
unimplemented") belongs in the decision-table structure, not in the
EL action body: add a condition that detects the unknown-state case
(`job.state is not in known_states_array`) and fire a logging action
on that column.

The `state_period.state_code` vs `job.state` selection in the existing
postfix is also a decision-table thing — add a condition row
(`state_period is defined`) and let it flip a column.

**Workaround until then:** explicit per-state dispatch columns
(what the project does for the 19 implemented states). Doesn't scale to
50 states but works for the well-tested subset.

---

## Not grammar gaps — table restructuring (3 elements)

### R1: SC_Tax_Brackets context body uses ifelse + locals

**Affected element (1):** `SC_Tax_Brackets` context 1

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

**Why this isn't a grammar gap:** The context is doing conditional
logic — picking joint vs single deduction by filing status — which is
**precisely what conditions+actions are for**. EL deliberately doesn't
have inline `if … then … else …` inside contexts; you express branches
as columns in the decision table.

**Restructure:** drop the context entirely. Add three condition rows:

- `job.filing_status is equal to "MFJ" or job.filing_status is equal to "QSS"` → column 1 (joint)
- `job.filing_status is equal to "HOH"` → column 2 (HOH; same deduction as joint in SC)
- otherwise → column 3 (single/MFS)

In each column's action, set `result.sc_taxable_income` with the
matching deduction constant. Then the existing bracket conditions
(`result.sc_taxable_income <= sc_bracket_N_limit`) live on the same
table and `result.sc_state_tax` lands in the bracket-matched action.

(Or: keep SC_Tax_Brackets as the bracket-walk table, and move the
filing-status branch into the existing SC_Tax dispatcher that performs
it.)

---

### R2: Duplicate `<condition_number>` (Dispatch_State_Tax cond #10)

**Affected elements (2):** Dispatch_State_Tax has two
`<condition_details>` blocks both numbered 10 (legacy MO + HI, from
when the dispatch was patched in place during the original cleanup).

**Why this isn't a grammar gap or an authoring-API gap:** the
authoring API already supports add and delete:

```bash
# Remove the first condition_number=10:
echo '{"op":"delete-condition","condition_number":10}' \
    | dtrules table patch Dispatch_State_Tax --project .
# Re-add the duplicate with a fresh number and the columns it needs:
echo '{"op":"add-condition","condition_number":43,"dsl":"job.state is equal to \"HI\"","columns":{"43":"Y"}}' \
    | dtrules table patch Dispatch_State_Tax --project .
```

(The first `delete-condition condition_number=10` removes the first
match — the MO row — so the remaining duplicate becomes the only
condition with that number. Then we add it back as a new condition
with a unique number. Repeat as needed.)

The earlier doc proposed a `renumber-condition` op; that's not
necessary. add+delete is sufficient.

---

### R3: Dispatch_State_Tax fallback action body

**Affected element (1):** Dispatch_State_Tax action 10 (the MO/HI
combined dispatch). The synthesis postfix is the same as Gap A
(entity creation), but the **wrapping** — conditional `perform MO_Tax`
vs `perform HI_Tax` based on `job.state` — is a control-flow problem
that belongs in the decision-table layout.

**Restructure:** split the MO/HI combined dispatch into two distinct
columns:

- Condition: `job.state is equal to "MO"` → column N → action: `perform MO_Tax`
- Condition: `job.state is equal to "HI"` → column N+1 → action: `perform HI_Tax`

The synthesis postfix (entity creation) stays in Gap A's bucket and
isn't resolved until that grammar is added.

---

## Data-integrity issues (2 elements)

### D1: empty `<condition_number>` / `<action_number>` tags

**Affected elements (2):** `Calculate_Educator_Expenses` has one
`<condition_details>` block and one `<action_details>` block where the
number tags are present but empty (`<condition_number></condition_number>`).
The audit treats them as 0 and they're reported as
"condition 0 has hand-coded postfix without EL DSL".

**Fix:** set the number tags to 1 in the source XML. The round-trip
preservation in `pkg/dtrules/excel` (the `feat(authoring): preserve
every DT field on XML round-trip` commit) keeps whatever they're set
to. Not a grammar issue, not an authoring-API issue — a one-off broken
record in the source.

---

## Summary

| | Count | Resolution path |
| --- | --- | --- |
| **Real EL grammar gaps** | 5 | |
| &nbsp;&nbsp;A. Entity creation in action bodies | 4 | Add `create T as alias { … }` to EL grammar |
| &nbsp;&nbsp;B. `perform` with a string-valued name | 1 | Add `perform table named (<string>)` to EL grammar |
| **Table restructuring** | 3 | Use decision-table conditions+actions for branching |
| &nbsp;&nbsp;R1. SC_Tax_Brackets context ifelse | 1 | Flatten into per-status conditions on the table |
| &nbsp;&nbsp;R2. Duplicate condition_number 10 | 2 | `delete-condition` + `add-condition` with unique number |
| &nbsp;&nbsp;R3. Dispatch_State_Tax MO/HI action | 1 | Split into separate columns |
| **Data-integrity issues** | 2 | |
| &nbsp;&nbsp;D1. Empty number tags | 2 | Set numbers in source XML |
| **Total** | **10** | |

The restructuring + data fixes (5 elements) need no engine work — just
follow-up DT authoring via the API. The two grammar gaps (A + B) need
EL compiler additions; until then, the affected tables refuse to
execute, which is the intended behaviour of the strict gate.

---

## Proposed implementations

The two grammar gaps are mechanically small. Each adds one statement
form to the ANTLR grammar and one emitter visitor. No runtime VM
changes — the existing postfix operators (`createentity`, `execute`,
`strconcat`, `xdef`, `addto`) already handle both cases. The work is
purely on the EL compiler surface.

### Closing Gap A — `create <type> as <local>`

**Surface (v1, minimal):**

```
create <type-name> as <local-name>
```

That's the entire grammar addition. Field assignment is already in EL
(`set st_result.state_code = …` — dotted LHS on a local-bound entity
reference works today via the existing `set_statement` path; see
`pkg/dtrules/compiler/el/set_statement_test.go`). `addto` is already
in EL.

**Author pattern.** `Build_State_Tax_Result_For_Period` action 1
becomes:

```el
create state_tax_result as st_result;
set st_result.state_code = state_period.state_code;
set st_result.is_resident = true;
set st_result.state_source_income = state_period.allocated_income;
set st_result.state_agi = state_period.allocated_income;
set st_result.state_withholding = st_result.state_withholding
                                  + state_period.allocated_withholding;
add st_result to job.state_tax_results
```

**Postfix lowering.** The new statement lowers to:

```
/state_tax_result createentity /st_result xdef
```

The remaining lines already lower correctly (`set`, `add … to …`).
Total emitted postfix matches the existing hand-coded version
byte-for-byte.

**Implementation sites:**

- `pkg/dtrules/compiler/el/EL.g4` — add a `createStatement` alt to
  `statement`:
  ```
  createStatement : 'create' IDENTIFIER 'as' IDENTIFIER ;
  ```
  Regenerate parser/lexer/visitor with the checked-in ANTLR jar
  (`antlr-4.13.1-complete.jar`).
- `pkg/dtrules/compiler/el/postfix_emitter.go` — add
  `VisitCreateStatement(ctx)` that emits
  `/<type> createentity /<local> xdef`. Per the project memory
  rule on labeled alternatives, every labeled alt of the rule needs
  its own visitor — there's only one here.
- `pkg/dtrules/compiler/el/edd_loader.go` — validate the type name
  against the EDD at compile time; unknown types must fail compilation
  rather than blowing up at runtime.
- New unit test next to `set_statement_test.go` covering:
  create-then-set, create-then-add-to-collection, create-with-
  unknown-type fails compile, create-with-name-collision fails compile.
- Add a row to `pkg/dtrules/compiler/el/testdata/grammar_corpus.tsv`
  for the grammar sweep.

**Sugar to defer (v2).** A block form
`create T as alias { field = expr; … }` lowers to the same postfix.
Ship only if authors complain about verbosity.

**Things to *not* do:**

- No constructor expression form (`new T(field=…)`). That drags
  expression-vs-statement disambiguation into the grammar. Statements
  are enough.
- No auto-add-to-collection. Keep `add st_result to <collection>` as a
  separate statement so the target is explicit. The four affected
  elements all append, but future call sites may not.

### Closing Gap B — `perform table named (<string-expression>)`

**Surface:**

```
perform table named (<string-expression>)
```

`<string-expression>` is any EL expression of type string. No
"otherwise" clause, no result handling.

**Author pattern.** `Dispatch_State_Tax` action 7 becomes (after the
unrelated `state_period`-vs-`job.state` selection is lifted into a
condition row per R3 above):

```el
perform table named ("Calculate_" + job.state + "_Tax")
```

**Postfix lowering.** Compile the expression onto the stack (EL `+` on
strings already lowers to `strconcat` — see `grammar_corpus.tsv`),
then emit `execute`:

```
"Calculate_" job.state strconcat "_Tax" strconcat execute
```

That matches the existing hand-coded form, minus the
`/table_name xdef table_name execute` round-trip via a local — which
was only there because postfix authors needed the name twice (for the
`lookupEntity isnull` "otherwise" check we're not implementing).

**Runtime semantics.** At `execute` time the runtime resolves the
string against the table registry. If the table doesn't exist, raise
the same error a typo in a literal `perform` raises. **No fallback
semantics in EL.** Authors handle the unknown-state case via a
condition row in the dispatching table (a "no known state" column
that fires a logging action), not via syntax in the action body.

**Implementation sites:**

- `pkg/dtrules/compiler/el/EL.g4` — extend `performStatement` with
  labeled alternatives:
  ```
  performStatement
      : 'perform' IDENTIFIER                                 # PerformLiteral
      | 'perform' 'table' 'named' '(' expression ')'         # PerformDynamic
      ;
  ```
  This introduces labeled alts where there was previously one
  unlabeled rule — both arms need explicit visitors (project memory:
  labeled alternatives must each have their own `Visit<Label>`).
- `pkg/dtrules/compiler/el/postfix_emitter.go` — `VisitPerformLiteral`
  keeps the existing behaviour; new `VisitPerformDynamic(ctx)` visits
  the expression (leaving a string on the stack) then emits `execute`.
- Type-check the expression at compile time: must be `string`.
  Mismatch → compile error.
- Tests: dynamic perform with literal string, with concatenation, with
  unknown-state runtime error matching the literal-perform error path.

### Sequencing

1. **Gap B first.** Touches 1 element, no field-resolution
   subtleties, retires the `lookupEntity isnull ifelse` postfix idiom.
   Quick win.
2. **Gap A second.** 4 elements, touches the EDD entity-creation
   pathway, needs careful tests around field assignment after
   `create`. Bigger but still bounded.

After both land plus the three restructures (R1/R2/R3) and two data
fixes (D1), the strict hand-coded-postfix gate on TaxReturn drops to
zero violations and `TestTaxReturn_NoHandCodedPostfix` flips from
expected-fail to expected-pass.

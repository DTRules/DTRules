# DTRules Changelog

## v1.14.3 — 2026-05-24

Closes the three UX issues left open after v1.14.2's #790 blocker fix:

- **`dtrules review <path>` honors the positional path (#788).** The
  previous code parsed positional args into a local slice and then
  discarded it (`_ = parsedArgs`); only `--project` was honored. Now
  the first positional arg is treated as the project path when no
  flag is present. `--project` still wins if both are supplied.
  `TestRunReview_PositionalPath` pins this — runs `runReview <dir>`
  from a CWD that has no project of its own, asserts the report
  persisted under `<dir>` (not CWD) with a non-empty, non-SHA256-of-
  empty `project_hash`.

- **`authoring.OpenProject` accepts flat directories (#791).** The
  canonical `<path>/xml/*_dt.xml` layout still wins when present;
  when not, `OpenProject` falls back to scanning `<path>` itself
  for `*_dt.xml`. This is what makes `dtrules review`,
  `dtrules table list`, and `dtrules table warnings` reachable for
  library consumers (e.g. staking) whose rules live next to the Go
  code that embeds them. Five error-hint strings in `table_cmd.go`
  and `mcp_tools.go` updated to mention both layouts.
  `TestOpenProject_FlatLayout`, `_CanonicalLayoutStillWins`, and
  `_NeitherLayoutErrorsClearly` pin the discovery contract.

- **`dtrules build` runs the advisory pass even on "Nothing to do"
  (#787).** Previously the build short-circuited with
  `Nothing to do: all files are in sync.` when sync detected no
  changes — the advisory pass was skipped entirely. Now the default
  branch prints `Nothing to sync — running advisory pass on existing
  XML.` and invokes `runStaticAnalysis` against the current XML.
  Warnings print inline. Cheap, XML-driven, no reason to skip them.
  Extracted to a top-level `runNoSyncAdvisory(xmlDir)` helper so
  `TestRunNoSyncAdvisory` can pin the behavior directly, since
  reaching the "in sync" sync-detection state from a copied fixture
  is unreliable (mtime/hash drift).

### Also in this release: EL compiler — double-operand dispatch

The sibling of #790 in the EL compiler itself: `promoteArithType` had
no `TypeDouble` arm, so any expression promoting to Double
(`double × double`, mixed `int + double`, `the minimum of <double>
and <double>`) returned `TypeInteger` from the promotion, and
`arithOp` / `minMaxOp` then emitted the integer family (`*`, `min`,
`+`, `==`) on operands the runtime stores as `*RDouble`. Crashed
later at `IntValue` — the same way #790 played out for fixed.

- `promoteArithType` lattice extended to **Fixed > BigInt > Double >
  Integer**. Two doubles promote to Double; mixed int/double widens
  to Double.
- `arithOp(target, intOp, bigOp, dblOp, fpOp)` and
  `minMaxOp(target, intOp, dblOp, fpOp)` accept a new `dblOp`
  argument that routes Double targets to the f-prefixed ops
  (`f+`, `f-`, `fmul`, `fdiv`, `fmin`, `fmax`, `f<`, `f<=`, `f>`,
  `f>=`, `f!=`).
- All call sites updated to pass the matching double op. Integer
  callers still get integer ops; nothing other than the Double path
  changes.
- Regression coverage in `double_dispatch_test.go`:
  `TestDoubleDispatch_Arithmetic`, `_MinMax`,
  `_OrderingComparisons`, `_NestedExpression`,
  `_IntegerStillUsesIntOps`.

**Out of scope:** bare-name `==` and `!=` for double operands still
route through `BoolNameEq` / `BoolNameNeq` whose `identNumericType`
gate intentionally excludes Double to preserve the legacy
fixed↔double silent-snap safety. Pure-double `x == y` still falls
through to `streq` today. Documented as a separate follow-up; needs
a cross-type-safety check (let pure-double `==` use `f==`, keep
fixed↔double on `streq` or error). Ordering comparisons
(`>`, `<`, `>=`, `<=`) go through the IntGt/Lt family which uses
`promoteArithType` directly and *is* fixed.

## v1.14.2 — 2026-05-24

Patch: `dtrules compile` now passes a symbol table to the EL compiler so
fixed-point operands compile to fixed-point operators. v1.14.x's
standalone compile path was emitting integer ops on `fixed`-typed
operands; the strict loader then rejected those at runtime
(`[Conversion Error] IntValue: No Integer value exists for this type`).
This blocked staking and any other consumer whose tables use `fixed`
operands from migrating off v1.12.0.

- **`dtrules compile` discovers `*_edd.xml` near the target, parses out
  field → type symbols, and calls `el.Compiler.SetSymbols` before the
  compile loop (#790).** Mirrors what `session.RuleSet.buildSymbolTable`
  used to do for the in-loader compiler in v1.12.0 (removed in v1.14.0)
  and what the workbook importer does for the build pipeline. With
  symbols in play, the EL compiler picks `fp-` / `fpmin` / `cvfp` for
  fixed operands instead of `-` / `min` / `cvi`.

- **New `--force` flag.** Rewrites existing postfix from a fresh
  compile (default only fills empty postfix). Use after a compiler
  fix like this one to refresh stored postfix produced by an earlier
  buggy version.

- Regression test `TestCompile_FixedOperandsEmitFixedOps` pins the
  fixed → `fp-` / `fpmin` / `cvfp` behavior end-to-end through the
  CLI dispatch.

### Known limitation (not in scope for this patch)

The EL compiler's `double` dispatch is less reliable than its `fixed`
dispatch — `double` operands can still emit integer ops in some
patterns (notably `the minimum of A and (B * C)`). Affects DTRules'
own TaxReturn sample project, which still passes its test suite. No
downstream consumer reported this in #790; tracking separately.

### Migration

Consumers who ran `dtrules compile` against their rules under v1.14.0
or v1.14.1 should re-run with `--force` to overwrite stale postfix:

```bash
go get github.com/DTRules/DTRules@v1.14.2
dtrules compile --force <rules-dir>
git diff
git commit -am 'refresh postfix per v1.14.2 #790 fix'
```

## v1.14.1 — 2026-05-24

Single-flag follow-up to v1.14.0: `dtrules compile` now runs the
advisory pass by default so library consumers on flat layouts (no
`xml/` subdir) can finally see warnings against their tables.

- **`dtrules compile` runs `decisiontable.Analyze` after the
  postfix-fill loop.** Same call the build pipeline and
  `dtrules table warnings` use, so the warning set is bit-for-bit
  identical across surfaces. Output goes to stderr; the
  `advisory: N warning(s)` summary line goes to stdout.

  This closes the user-visible gap from v1.14.0: `dtrules build`
  and `dtrules review` require a canonical project layout
  (`<project>/xml/*_dt.xml`); a consumer like staking whose rules
  live at `pkg/<...>/rules/staking_dt.xml` flat saw zero warnings
  from any CLI surface. Now `dtrules compile <rules-dir>` works on
  any layout — flat or canonical — and surfaces the full advisory
  set (no-op columns, subsumed columns, FIRST-policy redundant
  conditions, assignment-only tables, DSL-negation unreachable
  columns).

  Opt-out via `--no-analyze` for callers that want a pure
  postfix-fill pass with no advisory output. Compile errors still
  drive exit code; advisory warnings do not (matching the
  established "warnings never fail the build" policy from #761).

  Smoke test on staking's `pkg/dtrules/rules/staking_dt.xml`: 41
  warnings now surface across 9 tables — including the
  `Calculate_Withholding column 4 row 1 (=Y) is implied by column
  2's failure` pattern that motivated this whole arc.

## v1.14.0 — 2026-05-24

Headline: the loader is strictly a postfix consumer. `compiler/el` is
no longer in the runtime load path; library binaries shrink and every
load is silent unless something is genuinely wrong. The two-step
authoring pipeline (`dtrules build` / new `dtrules compile`) is now
the only place EL DSL is compiled.

- **Loader strict policy (#785).** `pkg/dtrules/loader` no longer
  imports `compiler/el`. Previously the loader recompiled every DSL
  element at load time and preferred the fresh compile, which
  (a) shipped the entire EL grammar/parser/emitter into every
  consumer's runtime binary, (b) made stored postfix decorative
  because the loader always overrode it, and (c) spammed
  `loader: context N (...) — recompiled postfix differs from stored;
  using fresh compile` warnings that drowned the advisory output
  authors actually care about while cleaning up tables.

  Now: non-comment DSL paired with empty (or comment-only) postfix is
  a load error naming the table, the element kind and number, the DSL
  snippet, and directing the operator to `dtrules build` or
  `dtrules compile`. `SetSymbols` and `SetCollectionResolver` survive
  as no-ops for source compatibility with `session.RuleSet`.

  The XML-authored build's export step legitimately reads
  partially-compiled XML — it loads source to write Excel, then
  re-imports to compile. `DTLoader.Tolerant`, exposed via
  `RuleSet.LoadDecisionTablesTolerant` /
  `LoadDecisionTablesTolerantFile`, disables the postfix-presence
  check for that path. The workbook exporter opts in; **nothing else
  does**. Runtime consumers stay strict by default.

- **`dtrules compile` subcommand (#782).** Surgical EL→postfix
  backfill: walks every `*_dt.xml` under a directory, runs the EL
  compiler on each non-comment DSL element with an empty
  `<*_postfix>`, and writes the compiled postfix in place. **No
  Excel round-trip** — bytes outside the targeted postfix elements
  are untouched, sidestepping the lossy XML→Excel→XML rewrite that
  `dtrules build --from-xml` performs. `--strict` flips to
  atomic-or-nothing per file (for CI gates); default writes
  successful fills and reports errors as a to-do list.
  `TEMPLATE_*.xml` is skipped because placeholder content is
  intentionally not valid EL.

  This is the canonical backfill tool when XML authoring outpaces the
  build pipeline, or when migrating a project off the loader's
  (now-removed) recompile fallback.

- **Build & review now route through `decisiontable.Analyze` (#780).**
  The FIRST-policy redundancy check (#762) and assignment-only-table
  check (#763) were merged in v1.12.0 but had **zero production
  callers** — every authoring/build surface went through the legacy
  `AnalyzeTable` shim, which silently drops the `Policy` field. A
  FIRST table with obvious "implied by prior column failure" Y entries
  produced no warnings anywhere. Now:

  - `analyzeAuthoringTable` (used by `dtrules table warnings`,
    `dtrules table get`, the `project_full_review` MCP tool, and
    `dtrules review`'s per-table optimizer pass) goes through
    `decisiontable.Analyze(Inputs{Policy: t.Policy, …})`. FIRST-policy
    redundancy and assignment-only warnings now actually fire.
  - `cmd/dtrules/build.go` drops ~200 lines of inlined
    `analyzeTableStructure` and routes `runStaticAnalysis` through the
    same `decisiontable.Analyze` call. One source of truth across
    build, review, and table-warnings.

  Tree-based checks (#765 / #766) still aren't wired anywhere — the
  load + compile + optimize cost is too high to pay on every build
  (10-minute timeout reproduced on TaxReturn). Documented for a
  future `--full` flag.

### Migration for library consumers

Bump to this release and run the backfill once per rules directory:

```bash
go get github.com/DTRules/DTRules@v1.14.0
dtrules compile <rules-dir>     # fills postfix from DSL
git diff                        # review the change
git commit -am 'compile DSL → postfix'
```

After that, `session.LoadDecisionTables` is silent on load and the
runtime no longer pulls `compiler/el` into the binary.

### Sample-project status (intentionally out of scope)

These are tracked separately so they don't gate infrastructure:

- **#781** — 6 `<initial_action>` elements in TaxReturn with prose DSL
  + hand-coded postfix in legacy `<action_postfix>` aliases. Violates
  v1.11.0's zero-hand-coded-postfix policy; the auto-recompile
  fallback was masking them.
- **#783** — 2 DSL-without-postfix elements in SyntaxTests.
- **#784** — ~15 test fixtures using the auto-compile pattern.

None of these are on the runtime path that library consumers exercise.

## v1.13.0 — 2026-05-13

Headline: proper EDD usage analysis (#776). The regex-only pass that
flagged every bare-name access as "unused" is now entity-stack-aware
across every EL iteration form and `using` block.

- **EDD analyzer is entity-stack-aware (#776 phases 1 + 2, PRs #777
  + #778).** The regex-only pass at `pkg/dtrules/analysis/edd_unused.go`
  only matched dotted `entity.attr` references. Bare field names
  inside `for all <field>` contexts — the common idiom for "iterate
  this array and read fields off each element" — were invisible to
  it. Every field accessed that way was flagged as a false-positive
  "unused EDD field." A downstream consumer (staking) reported 97
  such false positives.

  **Phase 1 (#777):** the analyzer now reads each table's
  `<context_details>` block, extracts `for all <field>` patterns,
  resolves the field's declared array subtype against the EDD, and
  pushes that entity type onto a per-table entity stack. Bare
  identifiers in conditions/actions then resolve against the
  topmost entity (innermost-first). The iterating array field itself
  is also counted as a read so `job.taxpayers` isn't flagged as
  unused just because the only "read" was the `for all taxpayers`
  clause.

  **Phase 2 (#778):** coverage extended to every entity-stack push
  the EL grammar exposes — `for all` / `forall` (no-space form) /
  `for all … where` / `for all … whose` / `for first of … where` /
  `for first in … where` / `for each <var> in` /
  `using <a, b, c> { … }`. These are scanned both in
  `<context_details>` and inline inside individual DSL fragments,
  so the pattern staking flagged
  (`for first of accounts where identity_url == …`) resolves bare
  names in the where-clause against the iterated entity.
  Multi-entity `using` blocks (`using a, b { … }` and the no-comma
  adjacency form `using a b { … }`) are handled; the expression-level
  type-conversion form `using a (b)` is excluded by the `{` anchor.

  Effect on the in-tree TaxReturn corpus: 1220 → 1193 warnings.
  Downstream projects with bare-name idioms (staking's 97 reported)
  should see most of those collapse to genuine unused-field findings.

- **Remaining #776 phases (carried over):** cross-table reference
  graphs (`perform <Table>` descent), enumeration-bounded dynamic
  strings for `perform table named (<expr>)`, and a full EL AST
  walk in place of the regex + keyword-stoplist approximation.
  Phases 1 + 2 cover the static analysis the engine needs day-to-day;
  the remaining phases are accuracy / capability improvements
  layered on top. #776 stays open.

## v1.12.0 — 2026-05-12

Headline: compiler advisory pass + Full Review deployment gate (#767),
`first pass` EL predicate (#764), and tech-debt batch that unblocks
arm64 cross-compile and trims runtime overhead.

- **Decision-table compiler advisory pass (#767, PR #771).** A
  project-wide static analysis layer that surfaces structural and
  semantic findings without changing rule behaviour. Errors gate
  deployment; warnings never do. The pass has two channels:

  - **Per-table authoring channel (#761).** `dtrules table get / put
    / patch` (CLI + MCP) now embed a `warnings` array on every
    response. New `dtrules table warnings <name>` CLI and
    `table_warnings` MCP tool give a read-only fetch. `decisiontable.Warning`
    gains a stable JSON shape with `ConditionRow` so authoring UIs
    can pin warnings to specific rows.

  - **Project-wide Full Review (#768).** `dtrules review` CLI and
    `project_full_review` MCP tool produce a report (structure +
    EL compliance + load diagnostics + per-table optimizer + EDD-unused).
    Persisted to `.dtrules/last-review.json`. `dtrules build
    --require-review` reads the cached report, recomputes the
    project hash, and refuses unless the report is fresh
    (`--max-age`, default 24h) and `passed: true`. No `--force`
    — errors crash deployment, that's the bright line.

  - **New optimizer checks:** `redundant condition` (#762) flags
    Y/N entries in FIRST-policy tables already implied by a prior
    column's failure; `assignment-only table` (#763) flags tables
    where every action is a single `set` and every column assigns
    the same variables; `unreachable column` and `dead condition
    row` (#765, #766) use the compiled decision tree (ANode /
    CNode walks) to catch what the matrix-only checks miss.

- **EL: `first pass` predicate (#764, PR #772).** New boolean
  expression that returns true on the first iteration of the
  innermost active loop in the table's context, false on subsequent
  iterations, and false when no loop is active. Lets authors fold
  one-shot setup into a regular condition row without writing a
  degenerate single-assignment table. New `FIRSTPASS` lexer token
  (`'first' WS+ 'pass'`); new State methods `PushLoopFrame /
  PopLoopFrame / BumpLoopIteration / IsFirstLoopPass`; iteration
  ops (`for / forr / forall / forallr`) instrumented to push, bump,
  and pop. The action-level `for all X perform Y` request (#735)
  was closed in favour of this predicate plus the existing
  context-level `for all` pattern.

- **Cross-compile on arm64 (#748, PR #773).** `asm_stubs.go` and
  `asm_helpers.go` are now gated to `//go:build amd64`, and
  `asm_fallback.go` (`//go:build !amd64`) routes `ExecuteBytecodeASM`
  to the pure-Go `ExecuteBytecode` path. The `make release` targets
  for `linux-arm64` and `darwin-arm64` now build cleanly; before
  this, they silently failed at link time.

- **Operator registry hot-path (#755, PR #773).** Drop the RWMutex
  on `operators.Get`. The map is populated at init() and never
  mutates; `Get` is called from the bytecode `OpName` lookup on
  every operator dispatch. Race tests pass with the lock removed.

- **ASM fallback panics → errors (#756, PR #773).** Replace 18
  `panic()` calls in `asm_helpers.go` fallback stubs with
  `state.lastError = ...` + return code. DTRules is meant to be
  embedded; panic() takes the host process down. Same loud-failure
  intent, non-fatal delivery.

- **Authoring API: load parity (#757, PR #774).** Audit raised
  drift concerns between `cmd/dtrules.loadRuleSet` and
  `cmd/api`'s inline load block. Rather than extracting an SDK
  (the two callers have intentionally different error policies —
  CLI fatals, API logs-and-continues), this release adds parity
  tests in both packages that pin the load surfaces to a common
  contract against the CHIP fixture. Behaviour drift in either
  caller now fails its package's test rather than surfacing later
  in a downstream call. Extracted `cmd/api/load.go`'s
  `buildRuleSetFromXML` to make the load logic directly testable.

- **Documentation alignment (#749, #750, PR #773).** Project
  structure block in `.claude/CLAUDE.md` updated to match the tree:
  removed nonexistent `examples/`, `legacy/java/`, and the stale
  `pkg/dtrules/sdk` directory; added `cmd/api`, `authoring`,
  `compiler/el`, `interpreter`, `runtime`, `scripts`, `ui`, …
  The `dtrules docs sdk` code sample is annotated as preview, see
  #757. `make check` now includes 7 previously silently-excluded
  packages (`analysis`, `benchmark`, `entity`, `repository`,
  `ruleset`, `testsupport`, `trace`); the Makefile's exclusion
  block names every remaining skip and why.

- **Dead code removed (#753, #754, PR #773).** Deleted
  `pkg/dtrules/compiler/eltest/` (0 importers) and
  `pkg/dtrules/xmlvalue_stub.go` (TODO placeholder, only
  references were commented-out test code).

- **Closes** #748, #749, #750, #753, #754, #755, #756, #757,
  #761, #762, #763, #764, #765, #766, #767, #768.

## v1.11.0 — 2026-05-12

Headline: strict hand-coded-postfix runtime gate; TaxReturn cleared to
zero violations; two new EL grammar primitives close the last real
gaps (#769).

- **Strict hand-coded-postfix runtime block.** The interpreter now
  refuses to execute any decision table whose elements carry postfix
  without matching EL DSL. Hand-edited postfix bypasses the authoring
  API, which is the supported edit surface, and risks the next
  `dtrules build` re-emitting empty postfix from the empty DSL — so
  the loader flags any such table at load time and `Execute` /
  `ExecuteTable` return an error before running. A project-level
  test (`TestTaxReturn_NoHandCodedPostfix`) enforces zero violations
  across the TaxReturn sample on every CI run, plumbed into
  `make check`.

- **EL: `create <type-name> as <local-name>`.** Constructs a fresh
  entity instance of the named type and binds it to a local. Lowers
  to `/typeName createentity /localName xdef`. The local is then a
  valid target for `set <local>.<field> = X` (via the existing
  setEntityField path) and `add <local> to <collection>` (existing
  `addto`). Closes the entity-creation-in-action-bodies gap that 4
  TaxReturn dispatch elements were using as hand-coded postfix
  (Build_State_Tax_Result_For_Period actions 1 + 2;
  Dispatch_State_Tax initial_action 1 + action 10).

- **EL: `perform table named (<string-expression>)`.** Runtime
  resolution of a decision-table name from a string expression.
  Lowers to `<expr> performtable` — the existing
  `RString.RNameValue` coercion handles the string→name step.
  No `otherwise` clause: an unknown table at runtime is a normal
  undefined-table error, the same path a typo in a literal `perform`
  takes. Conditional fallback for unknown values belongs in the
  decision-table structure (an explicit "unknown" column with its
  own logging action), not in the action body.

- **Authoring API round-trip preservation.** `update-condition-dsl`,
  `update-action-dsl`, `update-context`, `update-initial-action`,
  and `add-`/`delete-` ops now preserve every unmodeled XML field on
  round-trip: hand-coded postfix on still-flagged elements, comments,
  `<context_entity>` directives, both modern (`<initial_action_dsl>`)
  and legacy (`<action_dsl>` inside `<initial_action>`) tag
  conventions, and `"-"` "don't care" column values. The earlier
  syncToXML rewrote these and broke runtime semantics; the typed view
  now matches against the original XML by Number/index before
  emitting.

- **TaxReturn cleared from ~266 hand-coded-postfix elements to 0.**
  Authoring rounds 1–7 cover the bulk; the final 10 split as:
  4 entity-creation elements closed by Gap A above; 1 dynamic
  dispatch closed by Gap B above; 1 broken-context restructure in
  SC_Tax_Brackets (helper table `SC_Compute_Taxable_Income` lifts
  the filing-status branch); 1 Dispatch_State_Tax initial_action
  restructure (`Log_State_Tax_Start` + `Apply_Part_Year_Allocations`
  helpers replace the postfix ifelse + nested forall); 1 MO/HI
  combined dispatch split into `Dispatch_MO_HI_Tax` +
  `Synthesize_State_Tax_Result_If_Empty` helpers; 2 duplicate
  `condition_number=10` rows renumbered (HI → 43, NJ → 44) and
  their stale "NJ" postfix corrected; 2 empty-number-tag data fixes
  in Calculate_Educator_Expenses; 1 dead action_number=7 dropped
  (it built `Calculate_<state>_Tax` names that match no table in
  the project).

- **41-state dispatch coverage.** Every state with income tax now
  has a wired `XX_Tax` table in `Dispatch_State_Tax`.
  `TestComprehensiveStateTaxes` is 135/135 across single, MFJ, and
  high-earner scenarios in every state plus DC. The territory stubs
  (AS, LA, MP, NM, PR, VI) and several states still use
  top-marginal-rate approximations rather than full brackets —
  tracked by #178.

- **Closes #496** (EL DSL for tax-computation tables 022-026) and
  **#499** (EL DSL for special-forms tables 040-101). Every element
  with postfix in TaxReturn now carries matching EL DSL.

## v1.10.0 — 2026-05-02

Headline: explicit timezone DSL for every tz-dependent date operator (#743).

- **Timezone DSL — `in zone <strexpr>` on every date op** (#743; phases 1–4
  in PRs #744, #745, #746, #747). Implicit-timezone behavior across `today`,
  `firstofmonth` / `firstofyear` / `endofmonth`, and `getyear` / `getmonth` /
  `getday` was a silent-correctness foot-gun: server-local timezone leaked
  into rule output, so the same instant produced different calendar
  components on different hosts. v1.10.0 fixes the bugs and adds explicit
  DSL primitives so authors can be self-describing about which zone a date
  question is asking in.

  - **Phase 1 — UTC anchor (#744).** Every previously-implicit-timezone
    op normalizes to UTC. Adds `<default_timezone>` element to
    `DTRules.xml` (defaults to UTC when absent). Tax-test baseline
    unchanged.

  - **Phase 2 — `in zone <strexpr>` clause (#745).** Construction
    (`date "2026-04-15" in zone "America/New_York"`, `new date Y, M, D
    in zone <expr>`, `current date in zone taxpayer.timezone`),
    conversion (rewrap an existing date for component extraction), and
    component extraction (`year of <date> in zone "UTC"`) all accept
    the clause. Resolver chain: IANA name via `time.LoadLocation` →
    ISO 8601 fixed offset (`+HH:MM` / `-HH:MM` / `Z`) → error with a
    pointer to both forms. Field-level `<field timezone="..."/>`
    declares the assumed zone for tz-naïve string parses on that
    field. No built-in regional aliases — authors declare project
    aliases as EDD `string` constants.

  - **Phase 3 — calendar comparison + week/quarter ops (#746).**
    New comparison ops, every one taking `in zone Z`:
    `same_calendar_day_as`, `same_calendar_week_as` (with
    `starting "Monday"|"Sunday"`), `same_calendar_month_as`,
    `same_calendar_quarter_as`, `same_calendar_year_as`. Existing
    `d<` / `d==` / `d>` remain instant comparisons. New bucket ops
    `firstofweek` / `endofweek` / `firstofquarter` / `endofquarter`.
    New extraction ops `getdayofweek` / `getweekofyear` / `gethour` /
    `getminute` / `getsecond`.

  - **Phase 4 — `format(date, layout)` + DST policy (#747).** New
    `format(date, layout)` and `format(date, layout) in zone Z` render
    a date with an explicit zone and a Go-style layout string, suitable
    for audit trails and CSV output. New
    `with dst_rule "earlier" | "later" | "error"` clause governs
    ambiguous and impossible local times at DST transitions
    (default: `error`, forces the author to clarify).

- **TaxReturn sampleproject: finish decision-table dedup** (#724, follow-up to
  #722 / #725). Resolves the remaining 24 duplicates. For each of the 22
  state-file copies of `Calculate_XX_Tax` / `Calculate_CA_Military_Retirement_Exemption`
  the state-file version carried empty or DSL-echoed `<action_postfix>` /
  `<condition_postfix>` bodies (zero runnable postfix tokens across all 22),
  while the aggregate copy in `TaxReturn_dt.xml` carried compiled, runnable
  postfix — so the duplicate block is deleted from `sampleprojects/TaxReturn/xml/states/XX_dt.xml`
  and the aggregate remains authoritative. The second byte-identical copy of
  `Calculate_CO_Tax` in `states/CO_dt.xml` is removed in-place. The CI
  escape hatch `DTRULES_ALLOW_DUPLICATE_TABLES=1` is removed from the
  `Verify TaxReturn` step in `.github/workflows/verify.yml`; `dtrules verify
  sampleprojects/TaxReturn` now exits 0.

- **`authoring.OpenProject` walks `xml/` recursively for `_dt.xml` files**
  (#724). Previously `loadDTFiles` used a flat `filepath.Glob` of `xml/*.xml`,
  so nested files like `xml/states/CO_dt.xml` were invisible to the authoring
  SDK — and therefore to `dtrules project diagnostics`. `dtrules verify` has
  always walked recursively, so the two surfaces disagreed. The loader now
  uses `filepath.WalkDir`, matching the verify pipeline. Covered by
  `TestDiagnostics_NestedDTFile`.

- **Detect duplicate decision-table names across XML files** (#722). Decision
  table names are keys for `perform`, `executetable`, the JSON CLI, MCP
  server, and every consumer of `authoring.Project.Table(name)`. When two
  files declared the same `<table_name>`, the loader previously picked one
  by load order and silently dropped the rest. The fix is a three-layer
  gate:

  1. **Edit-time, tolerant.** `authoring.OpenProject` keeps loading
     successfully but renames each duplicate after the first (by sorted
     filepath order) to `<name>-1`, `<name>-2`, .... Every rename is
     recorded as a `Diagnostic` (`Project.Diagnostics()`) and the new name
     persists when `Project.Save()` runs — so the condition becomes visible
     in git rather than being swallowed.
  2. **JSON CLI / MCP surface.** `dtrules project diagnostics --project <path>`
     emits the diagnostics list (exits 0 regardless of count). MCP gains a
     matching `project_diagnostics` tool.
  3. **Compile-time, strict.** `dtrules build`, `dtrules validate`, and
     `dtrules verify` fail non-zero when any table name matches the
     reserved `^(.+)-(\d+)$` marker or when two files declare the same
     real name. The suffix is grammar-invalid as an `IDENT` (per the EL
     grammar research on #722), so no rule author can accidentally
     `perform` a dup marker.

  Runtime loader behavior is unchanged — production rulesets that already
  resolved duplicates before this release continue to load.

  The TaxReturn sampleproject ships with pre-existing duplicates that
  motivated this work; deduplicating them is tracked as a follow-up.
  Tests that assert unrelated round-trip behavior over TaxReturn now set
  `DTRULES_ALLOW_DUPLICATE_TABLES=1` to opt out of the compile-time
  gate.

- **TaxReturn sampleproject: partial dedup of decision-table names**
  (follow-up to #722). Part 1 of the cleanup: 115 of 139 duplicates
  resolved. 114 `NNN_*_dt.xml` per-table files under
  `sampleprojects/TaxReturn/xml/` were uncompiled Excel-extraction stubs
  whose `<condition_postfix>`/`<action_postfix>` bodies were empty while
  the compiled copies lived in `TaxReturn_dt.xml`; each stub file is now
  an empty `<decision_tables/>` container and the compiled versions in
  the aggregate remain authoritative. The in-file duplicate of
  `Calculate_Mortgage_Interest_Credit` in `TaxReturn_dt.xml` is reduced
  to one copy: the per-certificate IRC 25 / Form 8396 implementation
  (TABLE_NUMBER 26000, `SpecialForms.xls`) is kept; the older
  property-iterating copy (TABLE_NUMBER 8400) is deleted. The remaining
  24 duplicates (23 between `TaxReturn_dt.xml` and `states/XX_dt.xml`,
  plus one in-file copy of `Calculate_CO_Tax` in `states/CO_dt.xml`) are
  tracked in #724; they require per-state content decisions because
  several state files carry richer or divergent implementations.
  `DTRULES_ALLOW_DUPLICATE_TABLES=1` stays set in CI until #724 lands.

## v1.9.0 — 2026-04-24

- **`for all X as <alias>` now executes at runtime** (#714). v1.8.1 shipped
  the `as`-alias grammar with compile-only tests; the emitted postfix
  immediately consumed the table body once (via `null allocate execute
  deallocate pop`) before the iteration started, so `alias.field` in the
  body never saw a populated slot. The emit now reserves the slot with
  `null allocate`, runs the wrapper per element inside the `for` loop, and
  releases the slot afterwards. Three additional fixes land alongside:
  `opGet` normalizes `/name` literals to the executable interned form
  (matching `Find`) so `<N> local@ /<field> get` actually reads the
  attribute; the loader resets the EL compiler's local-slot table between
  tables so indices start at zero per table; and `VisitTypedXmlValue` now
  runs the alias-access check, because the grammar routes bare IDENTs
  through `typedXmlValue` inside `strexpr` (e.g. string concatenation).
  Execution-based tests land in `pkg/dtrules/authoring/forall_as_runtime_test.go`.

- **JSON-first CLI for decision tables and the EDD** (#716). Adds a
  `dtrules table` and `dtrules edd` subcommand surface that wraps the
  existing `pkg/dtrules/authoring` SDK behind a JSON-in / JSON-out
  interface optimized for AI agents.

  Read:

  ```
  dtrules table list --project <path>
  dtrules table get <name> --project <path>
  dtrules table schema [--patch]
  dtrules edd  get     --project <path>
  dtrules edd  schema  [--patch]
  ```

  Write whole documents (EL is compiled and validated before save):

  ```
  dtrules table put <name> --project <path> < table.json
  dtrules edd  put         --project <path> < edd.json
  ```

  Targeted patches — one op per invocation, keyed on `op`:

  ```
  echo '{"op":"set-condition-cell","condition_number":1,"column":2,"value":"Y"}' \
    | dtrules table patch <name> --project <path>
  ```

  Table patch ops: `set-name`, `set-policy`, `set-condition-cell`,
  `set-action-cell`, `add-column`, `update-column`, `delete-column`,
  `add-condition`, `update-condition`, `update-condition-dsl`,
  `delete-condition`, and the matching `*-action`,
  `*-initial-action`, `*-context` families. EDD patch ops:
  `add-entity`, `delete-entity`, `add-field`, `update-field`,
  `delete-field`, `set-comment`.

  Every non-zero exit writes a JSON error record to stderr with
  `error` / `path` / `hint` / `detail` fields, so agents can react
  without parsing prose. The MCP server over this surface is the next bullet (#717).

- **Model Context Protocol server** (#717). Adds `dtrules mcp`, a
  stdio MCP server exposing the JSON authoring surface from #716 as
  MCP tools callable by agent clients such as Claude Code. Wire
  format is newline-delimited JSON-RPC 2.0 on stdin/stdout.
  Protocol version `2024-11-05`. See
  [spec.modelcontextprotocol.io](https://spec.modelcontextprotocol.io).

  ```
  dtrules mcp                         # project defaults to cwd
  dtrules mcp --project /path/to/prj  # explicit
  ```

  Tools exposed:

  - Read: `table_list`, `table_get`, `table_schema`, `edd_get`,
    `edd_schema`, `project_validate`.
  - Write: `table_put`, `table_patch`, `edd_put`, `edd_patch`.

  Each tool call is stateless: the server reopens the project,
  applies the op, saves, and discards — so concurrent clients and
  external edits to XML are safe by construction. EL compilation
  happens on save, and compile failures surface as MCP tool errors
  with the structured `{error, hint, detail}` payload from #716.

  Implementation note: the JSON-RPC framing is hand-rolled (~130
  lines) rather than pulled from an external SDK, since the
  protocol surface we expose is tiny and adding a dependency for
  it would have a higher long-term cost than the framing code.

- **Documentation: `dtrules docs cli` (new) and `dtrules docs authoring` (expanded)** (#715, #720). The `cli` topic is a task-oriented walkthrough of the binary — install, init, build, validate, verify, and typical workflows. The `authoring` topic is reorganized by task and surfaces ~10 previously-undocumented SDK methods plus the new JSON CLI and MCP wrapper surfaces.

## v1.8.1 — 2026-04-20

- **`for all <array> as <alias>` iteration alias** (#712). Adds an `as
  <alias>` clause to `for all` that binds each iteration entity to a local
  entity slot instead of pushing it on the entity stack. The alias is the
  only way to reach the current iteration entity, which makes nested
  same-list iterations non-shadowing:

  ```
  for all taxpayers as parent
      for all taxpayers as child where child.parent_id == parent.id
          // body can see both parent.* and child.*
  ```

  `<alias>.<field>` references resolve through the local slot via `<N>
  local@ /<field> get` — no `entitypush`/`entitypop` bracketing, no
  entity-stack shadowing. A `where`-clause variant (`for all X as y where
  y.active`) evaluates the predicate after the slot is populated, so alias
  references work inside the guard. Aliases that collide with an existing
  EDD symbol are rejected at compile time.

## v1.8.0 — 2026-04-19

Minor release. Introduces the `fixed` numeric type for blockchain / token math,
a first-class fp grammar front-end, and hardens the compiler / loader around
type-aware mutations, `for all <type> entities`, and EOF anchoring. Two
operator renames ship as breaking changes — see "Breaking changes" below.

## Breaking changes

- **`cvd` is now the double converter; `cvdate` is the new date converter**
  (#694, PR #695). Historically `cvd` was the DATE cast. That was the only
  cv* op whose target type didn't match its letter, and it confused every
  type-dispatch path in the compiler (notably the v1.7.x set/field-mutation
  family). `cvd` now means double; `cvr` is an alias retained for backward
  compatibility in hand-written postfix; the date cast moved to the new
  `cvdate` op. Any rule file that emits raw postfix (not EL) calling `cvd`
  and expects date semantics WILL break — recompile from EL, or rename
  `cvd` → `cvdate` in the stored postfix.

- **`sortarray asc=true` now produces ascending order** (#696, PR #701).
  The flag semantics were inverted: `asc=true` previously produced a
  descending sort and `asc=false` produced ascending. Both halves of the
  comparator, and every call site in the test suite, were written against
  that inversion, so the bug stayed hidden. Fixed to match the parameter
  name. External rules that called `sortarray ... true` expecting
  descending will see flipped output — swap the boolean. The implementation
  also replaced the bubble sort with `sort.SliceStable` for stability and
  complexity.

## Fixed-point (new type)

- **`RFixed` — 256-bit fixed-point type** (#684). Adds the `fixed` numeric
  type for amounts and rates on a 10⁻⁸ decimal grid. Signed 256-bit
  mantissa (~5.78 × 10⁵⁹ whole-token headroom), truncate-toward-zero on
  multiply/divide, exact add/sub, symmetric overflow bounds. Literal form
  `1.5fp`. StringValue always renders exactly 8 fractional digits so XML
  and JSON round-trips are bit-exact.

  Closes the precision gap that previously forced staking to use float64:
  every intermediate stays on the grid, so the final total can't drift
  from what the blockchain expects. Rejects `0.1 → 0.09999999`-style
  snapping by requiring an explicit `cvfp` cast for double operands;
  integer and bigint auto-promote exactly.

  Runtime: RFixed value type with Add/Sub/Mul/Div/Neg/Abs/Trunc,
  Equals/Compare, and every Object-interface conversion (truncate toward
  zero). Operators: `fp+ fp- fp* fp/ fpabs fpnegate fptrunc fp== fp!=
  fp> fp>= fp< fp<= cvfp`. EDD/XML/JSON loaders parse `type='fixed'`
  fields and fp values from strings (never float64).

- **`fpmin` / `fpmax` + type-aware Min/Max dispatch** (#688, PR #692).
  Adds `fpmin` and `fpmax` ops, and teaches the EL `maximum`/`minimum`
  visitors to dispatch by operand type so `the maximum of a and b` emits
  `fpmax` when both are fp, `bmax` when both are bigint, and the existing
  int/double dispatch otherwise.

- **First-class fp grammar front-end** (#689, PR #699). Adds the
  `FP_LITERAL` lexer token (`1.5fp`, `100.0FP`, etc.) and `fpexpr`
  productions. `local fixed x` and `(fixed) expr` are now parsed in the
  grammar itself rather than recognized ad-hoc. Strict digit validation
  (at most 8 fractional digits; rejects `--1`, bare `fp`) moved into the
  lexer.

## Compiler / EL

- **Type-aware field-mutation visitors** (#686, PR #690). The
  increment/decrement/add-to/subtract-from action family now dispatches
  by the target field's declared type — fp fields emit fp ops, bigint
  fields emit bigint ops, double fields emit double ops. Previously
  everything collapsed to integer with a trailing `cvi`, silently
  truncating bigint and fp mutations. Per the labeled-alternative rule,
  the whole family was fixed at once.

- **Missing fexpr visitors** (#687, PR #691). Added visitors for the
  double alternatives that had no `postfix_emitter.go` implementation:
  `absolute value`, `rounded`, `rounded to N`, and `rounded … with
  boundry B`. Previously the parser accepted these but the emitter
  dropped the subtree, producing wrong postfix.

- **Field-mutation visitors dispatch locals vs entity fields** (#693,
  PR #698). The same mutation family now emits `N local!` for local
  slots and `<field> xdef` for entity fields, rather than always
  assuming entity-field semantics. Hand-written `increment myLocal`
  code used to silently do nothing.

- **`set` statement dispatches locals and typed targets** (#709, PR
  #710). `set local = expr` now uses the local-slot machinery. Typed
  entity-field assignments insert the correct conversion op based on
  declared target type (including fp).

- **`for all <type> entities` grammar + EDD resolver** (#703, PR #707).
  The `forall` statement now accepts a type keyword (`for all customer
  entities where …`). The resolver consults the EDD to enumerate all
  entities of the given type from the current context.

- **EOF anchor in the compiler** (#703, PR #708). The EL grammar now
  requires EOF at the top of `done`, so trailing tokens after an
  otherwise-valid statement produce a loud parse error instead of being
  silently discarded. The broken context DSL in `context_details` that
  this exposed was rewritten as part of the same PR.

## Operators / runtime

- See **Breaking changes** above for `cvd` / `cvdate` (#694, PR #695)
  and `sortarray` (#696, PR #701).

## Infrastructure / tests

- **CI scope aligned with `make check`** (#700). CI now runs `go build
  ./...` (full module) + `go vet` + the scoped test suite, matching the
  project's local "done" signal. Also fixes a perform-table emit
  discovered while bringing CI into line.

- **Hotfix: `isFixedLiteralText` restored; fpmin/fpmax pinned in the
  registration test** (#702). The operator-registration matrix now
  includes every fp operator so additions can't regress silently.

- **Excel directories generated for 7 XML-only sample projects** (#705,
  PR #706). `dtrules build` now has a consistent starting point for
  every sample.

- **Behavior matrices** landed across the release covering all 233
  operators, the full `cv*` conversion matrix, datetime, hash, stack,
  bytes, entity, local-frame, sortarray, and string ordering. These
  are the harness that caught the `cvd` and `sortarray` breakers.

## Migration notes

- **`cvd` in stored postfix**: recompile the rule set from EL, or rename
  `cvd` → `cvdate` in any hand-authored postfix that relied on date
  semantics. Rules whose `cvd` calls were always numeric (the vast
  majority) need no change.
- **`sortarray asc=true` / `asc=false`**: flip the flag. Rules that
  relied on the previous inverted semantics were wrong-by-parameter-
  name; the fix now matches the parameter's advertised meaning.
- **Silent fp or bigint truncation in mutations**: if you had actions
  like `increment bigint_field` or `add to fp_field 1.5fp` that
  appeared to work but produced int64-truncated values, they will now
  run at full precision. Verify expected values if those mutations are
  tested against golden outputs.

## v1.7.3 — 2026-04-19

Patch release. Two more staking integration fixes.

- **`<context_details>` preserved on authoring SDK round-trip** (#681 / PR
  #682). My v1.7.2 fix taught `DTImporter.WriteXML` to emit structured
  `<context_details>` entries, but `DecisionTableXML.Contexts` stayed
  typed as `string`, so the decoder couldn't read the structured form
  back — `OpenProject → Save → Reopen` silently dropped every `for all`
  context. `Contexts` is now a typed string `ContextsField` with a
  custom `UnmarshalXML` that tolerates both the legacy raw-text form and
  the structured form, collapsing structured children into a newline-
  joined DSL string. The authoring SDK's `syncFromXML` now splits
  multi-line input into multiple `Context` entries (was keeping only the
  first).

- **`add entity to array` defaults to `swap addto`, not `addarray`**
  (#680 / PR #682). `VisitAddArrayToArray` used to default
  `srcIsArray = true` when the symbol table was empty; the
  `CheckAction(..., nil)` path (used in authoring checks and some tests)
  never gets a symbol table, so it always hit the wrong default. Now
  the default is the single-element-into-array pattern, which is the
  overwhelmingly common case in practice. Array-to-array merge
  (`addarray` with `true`) still works when the source is explicitly
  declared as `array` in the EDD.

## v1.7.2 — 2026-04-19

Patch release. Two more staking integration fixes.

- **Authoring SDK writes loader-compatible `<context_details>`** (#678).
  `DTImporter.WriteXML` previously emitted `<contexts>raw-text</contexts>`
  — the Excel-import intermediate form — but the loader expects
  `<context_details>` entries with per-statement child elements. Round-
  tripping through the authoring SDK produced XML the loader couldn't
  parse (staking worked around it with a fix-contexts.py post-process).
  Now the writer splits the raw Contexts string on newlines and emits one
  `<context_details>` entry per statement with an empty postfix; the
  loader compiles the DSL on load via the v1.7.1 recompile-on-load path.

- **`number of <arr>` emits `length`, not `numberof`** (#678). There is
  no `numberof` op registered; the mismatched token resolved to an
  executable-name lookup and failed at runtime. Now emits `length`.
  Also adds `VisitIntNumberOfWhere` for `number of <arr> where <b>` —
  a count-accumulator fold.

- Staking's third report (`add entity to array` emitting `addarray`
  instead of `addto`) was already fixed in v1.7.1 — wiring the EDD
  symbol table into the loader's EL compiler lets the existing
  type-detection in `VisitAddArrayToArray` see `srcType == TypeEntity`
  and emit `swap addto` correctly.

## v1.7.1 — 2026-04-19

Patch release. Two integration fixes uncovered by staking's v1.7.0 upgrade.

- **Loader now wires the EDD symbol table into the EL compiler** (#675).
  `RuleSet.LoadDecisionTables` builds a flat `field` / `entity.field` → type
  map from the factory's registered reference entities and passes it into
  the loader's compiler. Without this the compiler's `getExprType()`
  defaulted every operand to integer, so `bigint_field × integer_field`
  compiled to plain `*` with a trailing `cvi` — silently truncating the
  product to int64. After the fix the emit is
  `<left> <right> cvbi b* cvbi /target xdef`, preserving bigint semantics.

  Cascade: every symbol-table-driven emit path (`cvb`/`cvs`
  disambiguation in `VisitSetStringFromName` from v1.7.0, typed set/add
  field-type conversions) was previously receiving an empty symbol table
  via the runtime loader path; all now work as designed.

- **Loader always recompiles context DSL when it's valid EL** (#676).
  Previously the loader used stored context postfix verbatim when
  present, which meant compiler fixes (e.g. the v1.7.0 forallctl
  overhaul) didn't reach tables with cached stale postfix without a
  manual `dtrules build` step. Now a fresh compile always wins when DSL
  is valid; prose / comment-only DSL still falls back to stored postfix,
  matching the softer policy the condition and action paths already use.
  A mismatch between the fresh compile and the stored postfix is logged
  so users know to run `dtrules build` to refresh the XML on disk.

## v1.7.0 — 2026-04-18

Minor release with breaking changes. Drives the EL grammar sweep to zero
known failures (266 → 0), reworks boolean coercion, and removes a batch of
unused runtime ops plus the hash-table machinery.

### Grammar sweep: 100% coverage

PR #634 landed a sweep that compiles every labeled alternative in `EL.g4`
through the compiler and asserts non-empty postfix. After 25 PRs over this
release cycle, every one of the 565 labeled alternatives now produces valid
postfix. A new `grammar_helpers.tsv` exempts 30 grammar-fragment rules
(addtodest, blist, includeSearch, operatorlist, possessiveRef, arrayList,
tablelist, etc.) that aren't reachable from a top-level compile entry —
they're tested via parent-rule labels that wrap them.

New Visit methods span the whole grammar: forfirstctl/foreachblock/
firstblock/block/usingblock families (#650), debugstatement + ifElseIf
(#651), all dup-destination and no-dups add variants (#663), quantifier
and there-is bexpr family (#664), operatorstatements dispatch and ValueOf*
coercions (#671), policy-statement pop emits (#671), debug-before/after
wiring (#671), performCatchError (#671), removeEachWhere and boolMatchForall
nested forall (#672), and a symbol-driven `cvb` override in
VisitSetStringFromName so `set b = $name` picks the right coercion when the
target is declared boolean (#673).

A coverage guard test reads `EL.g4` and fails if any new `#label` lands
without a corpus entry. An unexpected-pass guard shrinks `known_fails.tsv`
automatically when fixes unblock previously-failing labels.

### Breaking: hash tables removed (#668)

`RTable` and the 8 operators (`newtable`, `tableget`/`tput`, `tableput`/
`tget`, `tablekeys`, `tablevalues`, `tablecontains`, `tableremove`,
`tablesize`) are deleted. `Object.RTableValue()` removed from the interface
and every implementation. `dtrules.TypeTable` and its aliases (`map`,
`dict`, `dictionary`, `hash`) gone. Bytecode `OpNewTable`/`OpTableGet`/
`OpTablePut` removed. `ruleset.AttrTypeTable` removed.

The grammar still parses `typedTable`/`texpr`/`tablelist`/`local table`,
but the emits for those forms now produce `elstmterror` at runtime with a
clear message. No sampleproject used hash tables — every `TABLE` reference
in `sampleprojects/**/*.xml` was for _decision tables_, which are a
different concept and completely unaffected.

### Breaking: unused ops nuked, aliased pairs consolidated (#669)

Removed (no callers or subsumed by other ops):

- `entityforall`       — iterate entity attributes
- `find_by_field` / `findbyfield` — array find-by-attribute
- `findmatch`          — relationship lookup (only used by `boolEntityIsOf`;
                         that emit now produces `elstmterror`)
- `add` (opArrayAdd)   — reverse-order alternative to `addto`
- `tokenize`           — string regex split (`split` covers the use case)
- `datecmp`            — three-way date compare
- `roundto`            — numeric rounding with an unused 3-arg signature
- `policystatements`   — stub op that returned an empty array; policy-
                         statement collection happens cmd-side from the XML

Consolidated (identical implementations → aliases):

- `createentity` → alias for `newentity`
- `yearof`       → alias for `getyear`
- `monthof`      → alias for `getmonth`
- `dayof`        → alias for `getday`

**Behavior note**: `monthof` previously returned 0-11 (Java-compat); now
returns 1-12 to match `getmonth`. No sampleproject used `monthof`, so safe
to align.

### Breaking: `cvb` strict coercion (#667)

`(boolean) <expr>` no longer silently returns null on coercion failure.
New rules:

| Input | Result |
|---|---|
| RBoolean | passthrough |
| string `"true"`/`"yes"`/`"y"`/`"t"`/`"1"` (case-insensitive, trimmed) | true |
| string `"false"`/`"no"`/`"n"`/`"f"`/`"0"` | false |
| other strings | error |
| integer/double/bigint zero | false |
| integer/double/bigint non-zero | true |
| any other type | error |

Previously a `(boolean) <value>` on an uninterpretable type returned null
silently; the null then flowed several ops downstream before failing with
a mystery "No Boolean value exists for this type." Now the cast site
itself fails loudly.

### Other

- Fixed silent fall-through of `AddDateToDest` (#663) — was defaulting to
  an arithmetic `+` emit via VisitChildren instead of the `addto` pattern
  used by AddEntity/Str/NumToDest.
- Many DSL-sample refinements across localvariables, dexpr, bytesexpr,
  eexpr, strexpr, iexpr corpus entries where the generator produced forms
  the parser rejected.

### Issues closed

- Epic #635 (drive EL grammar sweep known_fails to zero)
- #636–#649 (per-rule followups from the epic)

### Upgrade

```
go install github.com/DTRules/DTRules/cmd/dtrules@v1.7.0
```

Existing EL DSL continues to compile unchanged. Rules relying on:
- hash-table ops will need to restructure around arrays + entity-attribute
  lookup
- the removed ops (findmatch, tokenize, roundto, datecmp, find_by_field,
  policystatements) will need to use the surviving equivalents
- `(boolean) <value>` on unusual types will now error where it used to
  silently produce null

## v1.6.6 — 2026-04-18

Patch release. Completes #626 by covering every labeled alternative of `forallctl` in `PostfixEmitter`.

- **All seven `forall` context forms now compile to correct postfix** (#626, #632). v1.6.5 shipped `VisitForallSimple` only; the other six (`forallAllowRemove`, `forallWhere`, `forallWhereAllowRemove`, `forallInEntity`, `forallInEntityAllowRemove`, `forallInEntityWhere`) fell through to the base visitor and emitted nothing. Decision tables should never rely on hand-tuned postfix — the EL compiler must produce correct postfix for every documented form. Regression tests cover every form with three input fixtures (empty, all-authorized, mixed) — 21 subtests.

## v1.6.5 — 2026-04-18

Patch release. Four bugs the staking team filed against v1.6.4, all with regression tests.

- **`for all entity.array` compiled to empty postfix** (#626) — `PostfixEmitter` only overrode the non-labeled `VisitContextForallCtl` wrapper; the `forallSimple` labeled alternative fell through to the base visitor and emitted nothing. An empty context left `rcontext = nil`, so `Execute` skipped the forall loop and the table body ran once with no element entity on the stack ("not defined by any Entity on the Entity Stack"). Fixed by adding `VisitForallSimple` that emits `dup <array> forall pop`.
- **`LoadTestData` skipped mapping `<initialization>`** (#625) — authoring-SDK test-data loader didn't process the mapping's init section, so entities declared there never hit the stack before conditions ran.
- **`the minimum of / maximum of` action syntax** (#623) — grammar added for the two sugar forms; both compile to the existing min/max operators.
- **Spurious `cvi` in bigint arithmetic** (#624) — the emitter inserted `cvi` after subtraction/addition even when both operands were bigint, silently truncating to int64. Suppressed when both operands resolve to bigint.

## v1.6.4 — 2026-04-18

Minor release. A complete Go authoring SDK for DTRules projects: typed mutation, EL validation, execution, replay-to-breakpoint debugging, test scenarios, coverage, regression diff. Consumers no longer need to touch XML. Backward-compatible.

### `pkg/dtrules/authoring` — new public package

Open a project, mutate it via typed structs with EL-at-the-boundary validation, run test scenarios, assert outcomes and traces, diff against a prior version, save back through the canonical build pipeline.

```go
p, _ := authoring.OpenProject("./my_project")

// Mutate a table
tbl := p.Table("Compute_Eligibility")
_ = tbl.AddCondition(authoring.Condition{DSL: "applicant.age >= 65"})
_ = tbl.AddAction(authoring.Action{DSL: "set applicant.eligible = true"})
_ = tbl.AddColumn(map[int]string{1: "Y"}, []int{1})

// Run a scenario
s := &authoring.Scenario{
    Name: "senior eligible",
    EntryTable: "Compute_Eligibility",
    Inputs:   map[string]any{"applicant.age": 70},
    Expected: map[string]any{"applicant.eligible": true},
}
r := s.Run(p)
if !r.Pass { /* r.Failures names the attributes */ }

_ = p.Save()
```

### Sub-features

- **EL validation** (`CheckCondition`, `CheckAction`, `CheckContext`) — compile a single expression with a symbol table, return position-annotated errors. (#591)
- **Typed Table view** with per-artifact mutations (`AddCondition`, `UpdateAction`, etc.) and atomic column operations (`AddColumn`, `UpdateColumn`) so the table can never be left partially edited. (#592)
- **Typed EDD view** (`Entity`, `Attribute`) with type and default validation. (#593)
- **Typed Mapping view** with cross-artifact EDD validation: every `SetAttribute` mapping entry is checked against the EDD's entity+attribute+type. (#594)
- **Execute + Stepper** — run a table against loaded test data, get a trace, or step invocation-by-invocation. (#596)
- **Trace replay + debug session** — `Project.ResumeAt(trace, index)` pauses before any invocation in the original trace; `DebugSession` exposes `EntityStack`, `Resolve`, `Step`, `Continue`, and `SetAttribute` for what-if exploration. Deterministic replay; no back-step (earlier points via another `ResumeAt`). (#603)
- **Scenario struct + AssertState** — one-call pass/fail with structured failures. (#606)
- **Trace assertions** — `AssertVisited`, `AssertNotVisited`, `AssertSequence` on `RunTrace`. (#607)
- **`dtrules docs authoring`** — embedded doc topic; the SDK is discoverable from the CLI. (#608)
- **Batch scenario runner** — `Project.RunAllScenarios(dir)` reads every `*.json` and produces `BatchResult` with per-scenario pass/fail. (#609)
- **Table dependency graph** — `Table.Dependencies()` and `Table.Callers()` for impact analysis. (#610)
- **Scenario coverage** — `Cover(project, results)` returns which tables/columns were exercised and which weren't. (#611)
- **Regression diff** — `Diff(p1, p2, scenarios)` names the attributes that diverge between two project versions. (#612)
- **`TableInvocation.Column` populated** — `AssertVisited("Foo", 2)` and per-column coverage actually work. (#620 hotfix of the trace infrastructure.)

### Hardening

- Real-project round-trip tests on `sampleprojects/TaxReturn`.
- Destructive ops (`DeleteCondition`, `DeleteColumn`, `DeleteEntity`-with-references) tested.
- `SetAttribute` type-validates against the EDD.
- `LoadTestData` on malformed input returns clear errors without panic.
- Idempotent save (minimal + execute + TaxReturn fixtures).
- Fixed a non-idempotency bug in the DT XML postfix writer found by these tests.
- Coverage on `pkg/dtrules/authoring`: 81.4%.

### Issues closed

#591, #592, #593, #594, #596, #601, #603, #605, #606, #607, #608, #609, #610, #611, #612, #620.

### Consumer impact

Downstream repos (Accumulate staking, TaxReturn authoring work) can now:
- Write rule changes via typed Go calls instead of XML edits.
- Express test suites as Scenario structs with pass/fail assertions.
- Validate EL before committing a change.
- Debug execution by pausing at any trace point and inspecting entity state.
- Regression-diff a refactor against the prior version to prove behavior is unchanged.

Raw XML editing is still supported but discouraged — round-trip silently normalizes legacy prose-DSL to warnings (per #504/#583). Authoring SDK is the recommended path.

---

## v1.6.3 — 2026-04-17

Patch release. Visible build summary, silent-drop bug fixed, static analysis warnings, and a safety net against "scoped tests pass but full build fails" mistakes. No breaking changes.

### `dtrules build` summary

Every build now ends with a structured summary covering both steps:

```
Build Summary

Export step (XML → Excel):
  tables=11  actions=38  conditions=5  entities=10  mappings=0
  postfix-stripped=43
  files-written=1
  drops: none

Import step (Excel → XML):
  tables=11  actions=38  conditions=5  entities=10  mappings=0
  compiled=38
  files-written=3
  drops: none
  warnings: 0

Result: OK — no drops
```

Counts expose what was preserved. Drops are fatal (build exits non-zero and names table / column / item / reason). Warnings are advisory (legacy prose-DSL where the DSL field equals the comment text; preserved verbatim through the round-trip per #504). `--quiet` suppresses the summary on success. (#580, #582, #583 — PRs #581, #582, #584)

### Fix: import no longer clobbers `action_dsl` with `action_comment`

The combined-workbook parse path read column B (comment) into both `Action.Comment` and `Action.DSL`, silently replacing authored DSL with the comment text. After round-trip, `<action_dsl>garbage ~ {{</action_dsl>` would come back as `<action_dsl>broken action</action_dsl>`. Fixed: DSL now reads from column C as the exporter has always written it. (#585, PR #586)

### Static analysis warnings at build time

`dtrules build` surfaces three classes of advisory warnings with no CLI change:

- **No-op / subsumed columns** — a column with no actions, or whose conditions and actions are a subset of another column that always also matches.
- **Unreachable columns** — structurally contradictory condition requirements (simple string-level heuristics only; no solver).
- **Unused / write-only EDD fields** — attributes declared in the EDD but never read (or set but never read) by any decision-table DSL. Cross-table walk. Carve-outs for `mapping*key` and system-set fields.

On `sampleprojects/TaxReturn/` the analysis emits 1,419 advisory hits captured in `cmd/dtrules/testdata/golden/taxreturn-static-analysis.txt` so future drift is visible. Warnings never fail the build. (#555, PR #587)

### Safety net against scoped-test regressions

- `make check` — full-module `go build ./...` + `go vet` + scoped test suite (excluding the known-broken tax-content tests in `pkg/dtrules/` root). Authoritative "am I done" command.
- CI has a distinct `Build full module (go build ./...)` step so build failures surface independently of test failures.
- `.claude/CLAUDE.md` updated: agents are instructed to run `make check` before reporting success; scoped tests alone are not sufficient.
- Regression test asserts the Makefile retains `go build ./...`. (#529, PR #588)

Plus a local-only PR merge guard (`./scripts/merge-pr.sh`) that fetches the PR head, runs the full module build + tests, and only merges on green. Complements the safety net for interactive sessions.

### Issues closed

#529, #555, #580, #582, #583, #585.

### Stale issues closed

#530, #533, #535, #547 — already delivered by PR #542's doc consolidation.

### Consumer impact

The combination gives downstream repos (Accumulate staking, etc.) a build pipeline that:
- Never silently drops data (warnings vs drops classified explicitly, both visible).
- Proves what survived the round-trip via printed counts.
- Flags dead / unused rule code at build time.
- Doesn't ship broken main from Claude-driven merges.

---

## v1.6.2 — 2026-04-17

Patch release. Two consumer-ergonomic additions driven by Accumulate staking's migration path. No breaking changes.

### `session.LoadRulesFromFS(name, fs.FS, root)`

New primary API for loading rules from an `fs.FS` (typically an `//go:embed` tree). `LoadRulesFromDirectory(name, path)` still works and is now a thin wrapper around `LoadRulesFromFS(name, os.DirFS(path), ".")`. The underlying XML loaders already accepted `io.Reader`; the refactor is shallow and fully backwards-compatible.

Usage:

```go
//go:embed rules
var rulesFS embed.FS

rs, err := session.LoadRulesFromFS("MyApp", rulesFS, "rules")
```

No more tempdir workaround for embedded binary deployments. `dtrules docs embedding` updated. (#541, PR #575)

### `--xml-dir` / `--excel-dir` flags on `verify`, `validate`, `build`

Projects with non-standard layouts (e.g., `pkg/dtrules/rules/` + `pkg/dtrules/excel/`) can now drive the tooling without adopting the default `xml/` + `excel/` convention. Three levels of precedence:

1. CLI flag (`--xml-dir <path> --excel-dir <path>`).
2. `<xml_dir>` / `<excel_dir>` elements inside `DTRules.xml`.
3. Default `xml/` + `excel/` relative to project root.

Error messages on missing directories now name the flags instead of listing an absolute path that doesn't exist. (#574, PR #576, hotfix PR #577 for fixture cleanup)

### Documentation

- `dtrules docs project-layout` updated with the new flags and `DTRules.xml` elements.
- `dtrules docs embedding` no longer recommends the tempdir workaround — direct `fs.FS` loading is shown instead.

### Issues closed

#541, #574.

### Consumer impact

Accumulate staking can now:
- Use `session.LoadRulesFromFS("Staking", rulesFS, "rules")` in their engine without a tempdir.
- Run `dtrules verify --xml-dir pkg/dtrules/rules --excel-dir pkg/dtrules/excel` against their repo layout.

Combined with v1.6.1's #568 and #570, staking's DTRules integration has no remaining protocol blockers.

---

## v1.6.1 — 2026-04-17

Patch release driven by consumer (Accumulate staking) feedback. Two protocol additions, both backward-compatible.

### `type="date"` accepts timestamps

`RDate` now parses ISO-8601 / RFC3339 datetimes (with optional nanosecond precision) and space-separated datetime strings, in addition to the existing `YYYY-MM-DD` form. Pure dates still serialize back as `YYYY-MM-DD`; non-midnight values serialize as RFC3339 — no churn on existing date-only XML. Closes the "No Time value exists for this type" error the staking repo worked around. (#568, PR #571)

### EL `error` and `warn` statements

- `error "<msg>"` — halts rule execution and returns `*dtrules.ELStatementError` to the host. Callers distinguish rule-raised errors from VM errors via `errors.As(&ELStatementError{})`.
- `warn "<msg>"` — emits a `[warn]` log line, execution continues.
- Postfix: `elstmterror` (OpError=128, stack effect `(string --)`), `elstmtwarn` (OpWarn=129).
- `message` supports string concatenation (`error "bad: " + field`). (#570, PR #572)

### Documentation

- `dtrules docs el` updated: Date subsection covers timestamp acceptance; new `Errors and warnings` subsection.
- `docs/EL-REFERENCE.md` gains entries for timestamp parsing and `error`/`warn` with captured postfix.
- `docs/COMPILER-INTERNALS.md` adds OpError/OpWarn entries.

### Issues closed

#568, #570.

### Consumer impact

Accumulate staking (gitlab #180) had 43 entries needing migration to real EL. The 2 entries blocked on #570 and the `sync_time` issue blocked on #568 are now both unblocked. Staking can retire its native-Go fallback after consuming v1.6.1.

---

## v1.6.0 — 2026-04-17

Protocol release following v1.5.0's packaging epic. Adds blockchain-friendly EL types, field-inspection tooling for deployed binaries, performance encoding for sparse decision tables, and five new embedded doc topics. No breaking changes.

### New EL types and operators

- **`bytes`** — first-class byte-sequence type. Hex literals `0x…`, `length of`, concat `+`, slice `from n to m`, indexed access `[i]`, `is equal to` / `is not equal to` with `crypto/subtle.ConstantTimeCompare`.
- **Hash built-ins**: `sha256 of`, `keccak256 of`, `ripemd160 of`, `sha3 of` → bytes.
- **Encoding helpers**: `hex of` / `bytes of hex`, `base58check of … version N` / `bytes of base58check`, `bech32 of … hrp "…"` / `bytes of bech32`, `bytes of bigint n size k` / `bigint of bytes`. BIP-173 HRP validation.
- Signature verification remains host-side by design.

### New runtime: lazy `ALL` table encoding

Decision tables with sparse `Y`/don't-care columns used to expand to `2^N` tree leaves. When the leaf count would exceed 64, the runtime now switches to a lazy column-matching encoding that evaluates each referenced condition at most once, regardless of column count. Automatic at XML import. No authoring change. `FIRST` / `BALANCED` tables unaffected.

### New SDK function

- **`excel.ExtractExcel(src fs.FS, dst string) error`** — consumer apps embedding DTRules rules via `//go:embed` can dump their own compiled rules back to an editable `excel/` tree for debugging or audit. Pure XML→xlsx representation conversion; no session, no loader.

### File conventions and mapping

- Enforced `_dt` / `_edd` / `_map` suffix convention on both xlsx and xml. Single-artifact workbooks carry the type suffix; mixed-artifact workbooks stay suffix-free and route by A1 marker.
- Mapping xlsx import/export implemented end-to-end. `TaxReturn_map.xlsx` fixture committed.

### Documentation

Five new embedded topics:
- `dtrules docs project-layout` — folder conventions, `_dt`/`_edd`/`_map` rule, `.sync-manifest.json`, `DTRules.xml`.
- `dtrules docs mapping` — XML and xlsx mapping schema.
- `dtrules docs database` — KV design from the EDD: key composition, arrays, references, `mapping*key`.
- `dtrules docs architecture` — dev-time-vs-deploy-time; deployment is a single `//go:embed`'d binary.
- `dtrules docs embedding` — the `ExtractExcel` recipe and single-binary distribution pattern.
- `dtrules docs bytes` — blockchain types and operators.

Repository reference material:
- `docs/EL-REFERENCE.md` rebuilt against ANTLR4 with real compiler-captured postfix, BIP-173 examples, tax + eligibility examples.
- `docs/COMPILER-INTERNALS.md` added: opcodes 1–127 with stack-effect notation, VM / runtime / session model, error taxonomy. Replaces the deleted `docs/bytecode-spec.md`.
- Both carry a prominent "NOT FOR RULE AUTHORS" banner.

### Tests

- `pkg/dtrules/encoding/` ships at 96.5% coverage.
- Bech32 checksum fixed to BIP-173 spec (the property test caught an off-by-one XOR in a six-hour-old `bech32CreateChecksum`; shipped in hotfix #564).
- 464 new lines of encoding operator + grammar tests.
- Lazy ALL table: semantic-equivalence tests against the tree encoding for random fixtures ≤64 leaves, plus property tests for each-condition-evaluated-at-most-once.

### Issues closed

#522, #524, #525, #529, #530, #531, #532, #533, #534, #535, #543, #545, #547, #548, #549, #550, #554, #557, #558, #559, #560.

### Still open, not blocking

- #501 — declined (XML-to-XML compile would bypass Excel-as-SoR).
- #520 — pre-existing tax-content test failures in `pkg/dtrules/` package (rule content, not the binary).
- #541 — loader should accept `fs.FS` directly (deferred; workaround documented in `docs embedding`).
- #555 — static analysis warnings (dead columns, unreachable columns, unused EDD fields) — filed, not yet implemented.

---

## v1.5.0 — 2026-04-17

Packaging release. Establishes **Excel as the system of record** and provides a single authoring pipeline so AI and human authors produce canonical rule artifacts every time.

### New CLI

- **`dtrules build [path]`** — single-command normalize-and-compile pipeline. Detects whether Excel or XML was edited and runs the correct path.
  - `--from-excel`: Excel → XML
  - `--from-xml`: XML → pretty Excel → compile → XML
  - `--dry-run`: report what would change without writing
- **`dtrules verify [path]`** — CI / pre-commit gate. Fails if `dtrules build` would change any file, if a `<source>` header is missing or invalid, or if a `_dt.xlsx` filename disagrees with sheet content. `--diff` and `--strict` flags available.
- **`dtrules version`** — semver + commit sha + build date, injected via `-ldflags`.
- `sync import` / `sync export` demoted to `dtrules internal sync …` (still available, hidden from top-level help).

### File conventions

Single-artifact workbooks carry a type suffix matching their XML counterpart:

| Artifact | Excel | XML |
|---|---|---|
| Decision tables | `Foo_dt.xlsx` | `Foo_dt.xml` |
| EDD | `Foo_edd.xlsx` | `Foo_edd.xml` |
| Mapping | `Foo_map.xlsx` | `Foo_map.xml` |

Mixed-artifact workbooks (DT + EDD + MAP in one file) stay suffix-free and route by A1 marker (`DT:`, `EDD:`, `MAP:`). Recursive subdirectories under `excel/` and `xml/` are supported.

### Round-trip fidelity

- Each XML artifact carries a `<source>` element capturing `relative_path`, `file_name`, and `sheet_number` so export places artifacts back where they came from.
- Sheet order within a workbook follows the `NNN_` filename prefix.

### Mapping xlsx support

Mapping (formerly XML-only) now imports and exports through `TaxReturn_map.xlsx`. Full round-trip with uniform styling. Section comments preserved. Closes the `MAP:` sheet stub from the previous release.

### Documentation

- `dtrules docs el` is audited complete against `pkg/dtrules/compiler/el/EL.g4` and gated by tests.
- `dtrules docs expressions` and `pkg/dtrules/docs/bytecode-spec.md` removed. Postfix and bytecode are internal compilation targets — not authoring formats.
- Banner added to `docs el`: *"EL is the only language to author rules in. Postfix and bytecode are internal compilation targets — do not write them by hand."*

### Excel styling

One shared `Styler` used by DT, EDD, and Mapping exporters: bold header with `#E8E8E8` fill, thin borders, frozen header row, Calibri body, Consolas/Menlo for DSL cells. Clean, not flashy.

### Release infrastructure

- `make release` produces binaries for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64 with SHA-256 checksums.
- `.github/workflows/release.yml` triggers on `v*` tags and publishes binaries as release assets.
- `go install github.com/DTRules/DTRules/cmd/dtrules@latest` path documented.

### Tests

28 new tests across 7 files, covering content-level round-trip (Excel edit → XML, XML edit → Excel — for DT, EDD, and mapping), build idempotency (sha-256 hashes), verify exit codes and strict mode, EL doc coverage, uniform styling, `<source>` header inference, A1 marker routing, legacy unmarked fallback, and the version CLI.

### Issues closed

#502 (epic), #503, #504, #505, #506, #507, #508, #509, #513, #522, #524, #525.

---

## Version 5.0 (Release)

### Summary

DTRules v5.0 is a major release featuring a complete Go runtime implementation, ASM optimization framework, modernized DSL compilers, and comprehensive test infrastructure.

### Key Features

- **Go Runtime**: Complete Go port of the DTRules interpreter
  - All primitive types (RInteger, RDouble, RBoolean, RString, RDate, RArray, RTable)
  - Entity Data Dictionary (EDD) loading from XML
  - Decision Table loading and execution
  - Mapping file support for data loading
  - All standard operators implemented (70+)

- **ASM Optimization**: x86-64 assembly runtime for performance-critical paths
  - Full arithmetic, comparison, boolean, and stack operators
  - Mixed-type arithmetic support (integer + double)
  - SSE-based floating point operations

- **ANTLR 4 Migration**: Modernized EL/EBL DSL compilers
  - Drop-in compatible with legacy JFlex/CUP compilers
  - Improved error messages and performance

- **Test Infrastructure**: Cross-platform test vectors and CI/CD
  - 206 shared test vectors across 10 categories
  - GitHub Actions workflow for Go, Java, and ASM

### Test Results

| Project | Test Cases | Status |
|---------|------------|--------|
| CHIP | 13 | Pass |
| KidAid | 4 | Pass |
| SyntaxTests | Compile | Pass |
| TestProject | Compile | Pass |

### Security Updates

- Apache POI updated to 5.2.5

---

## Version 5.0-SNAPSHOT (Development)

### 2026-02-05: ASM Mixed-Type Arithmetic and Double Comparison Support

#### Summary
Fixed critical gaps in the x86-64 ASM implementation: mixed-type arithmetic (integer + double) operations now correctly convert integers to doubles, and comparison operators now support double and mixed-type comparisons.

#### Changes

**ASM Bytecode Fixes (`asm/src/vm/bytecode.asm`):**
- **Arithmetic operators** (`op_add`, `op_sub`, `op_mul`, `op_div`): Now handle mixed integer/double operands by converting integers to doubles
- **Comparison operators** (`op_lt`, `op_le`, `op_gt`, `op_ge`): Added full double and mixed-type support
- **Min/Max operators** (`op_min`, `op_max`): Added double and mixed-type support using SSE instructions

**Test Harness Enhancement (`asm/test/unit/test_harness.asm`):**
- Added `assert_double_eq` function for comparing double values with epsilon tolerance

**New Unit Tests:**

| Test File | New Tests | Description |
|-----------|-----------|-------------|
| test_arithmetic.asm | 8 | Mixed-type add/sub/mul/div, double arithmetic, double comparison |
| test_comparison.asm | 6 | Double comparisons (lt, gt, le, ge), mixed-type comparisons |

#### Before/After

**Before (Error):**
```
5 + 3.14 → ERR_TYPE_MISMATCH
3.14 < 5.0 → ERR_TYPE_MISMATCH
```

**After (Correct):**
```
5 + 3.14 → 8.14 (double)
3.14 < 5.0 → true (boolean)
```

#### Verification
- All 13 ASM unit test modules pass
- 19 new arithmetic tests pass (including 8 mixed-type tests)
- 18 new comparison tests pass (including 6 double/mixed tests)
- Go core tests pass
- NativeASM tests pass

---

### 2026-02-05: Unified Test Infrastructure

#### Summary
Implemented cross-platform test infrastructure with shared test vectors, CI/CD pipeline, and comprehensive documentation for all DTRules implementations (Go, Java, ASM).

#### New Files

**Test Infrastructure:**
- `test/run-all-tests.sh` - Cross-platform test orchestrator
- `test/readme.md` - Test infrastructure documentation
- `test/vectors/*.json` - Shared test vectors (206 tests across 10 categories)

**Shared Test Vectors:**
| File | Tests | Coverage |
|------|-------|----------|
| arithmetic.json | 28 | +, -, *, /, abs, negate, f+, f-, f*, fdiv |
| comparison.json | 24 | ==, !=, <, >, <=, >= |
| boolean.json | 17 | and, or, not, xor, beq |
| stack.json | 17 | pop, dup, swap, rot, over, pick, roll |
| string.json | 34 | concat, substring, trim, indexof, etc. |
| array.json | 19 | newarray, addto, length, getat, memberof |
| control.json | 12 | if, ifelse, for, while, forall |
| table.json | 16 | newtable, tableget, tableput |
| entity.json | 15 | def, lookup, entitypush, get |
| datetime.json | 24 | newdate, getyear, adddays, daysbetween |

**CI/CD:**
- `.github/workflows/tests.yml` - GitHub Actions workflow
  - Go tests: Linux, macOS, Windows (Go 1.21, 1.22)
  - Java tests: Linux, macOS, Windows (JDK 11, 17, 21)
  - ASM tests: Linux (NASM)
  - Comparison tests (ASM vs Go)
  - Performance benchmarks (main branch)

**Java Unit Tests:**
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RIntegerTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RDoubleTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RBooleanTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RStringTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RNameTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RArrayTest.java`

**Documentation:**
- `docs/testing.md` - Comprehensive testing guide

#### Verification

**ASM Test Results:**
- 13 unit test modules pass
- 100+ individual tests pass
- All arithmetic, comparison, boolean, stack, string, control flow operators verified

**NativeASM Test Results:**
- All tests pass
- Full coverage of arithmetic, comparison, boolean, and stack operations

**Go Test Results:**
- All 23+ test suites pass
- Comprehensive operator coverage

---

### 2026-01-30: ANTLR 4 Migration

#### Summary
Migrated the EL and EBL DSL compiler modules from JFlex/CUP to ANTLR 4, modernizing the parser infrastructure while maintaining backward compatibility.

#### Changes

**New ANTLR 4 Compilers:**
- `ELAntlr` - Expression Language compiler (replaces `EL`)
- `EBLAntlr` - Extended Business Language compiler (replaces `EBL`)

**New Files Added:**

| Module | Grammar | Compiler | Visitor | Type Resolver |
|--------|---------|----------|---------|---------------|
| EL | EL.g4 | ELAntlr.java | ELCompilerVisitor.java | ELTypeResolver.java |
| EBL | EBL.g4 | EBLAntlr.java | EBLCompilerVisitor.java | EBLTypeResolver.java |

**Build System Updates:**
- Added ANTLR 4 Maven plugin (v4.13.2) to EL and EBL module pom.xml files
- Added ANTLR 4 runtime dependency
- Retained JFlex/CUP dependencies for legacy compiler support

**Documentation:**
- Added `dsl/ANTLR_MIGRATION.md` with comprehensive migration guide
- Added `ELCompilerComparisonTest.java` for validating compiler parity
- Added `CompilerTest.java` - parameterized JUnit test suite (128 tests)

#### Compatibility
- **100% test compatibility** - 128 tests pass (64 tests x 2 compilers)
- Legacy compilers retained for backward compatibility
- Same `ICompiler` interface - drop-in replacement
- All sample projects verified working

#### How to Switch Compilers
Update `DTRules.xml`:
```xml
<!-- Use new ANTLR 4 compiler -->
<compileralias name="EL">com.dtrules.compiler.el.ELAntlr</compileralias>
```

#### Sample Projects Tested
All compile and run with 0 errors:
- CHIP - 2 test cases PASS
- KidAid - 2 test cases PASS
- SyntaxTests - 1 test case PASS
- TestProject - Compile PASS

### 2026-01-30: Project Cleanup

#### Removed Sample Projects
The following incomplete sample projects were removed:
- `Sudoku` - Custom DSL demo (incomplete test data)
- `eBook` - Multi-ruleset example (missing test cases)
- `eBookApp` - eBook application wrapper

#### Remaining Sample Projects
- `CHIP` - Health insurance eligibility (2 test cases)
- `ChipApp` - CHIP application wrapper
- `KidAid` - Child assistance eligibility (2 test cases)
- `KidAid_Application` - KidAid application wrapper
- `SyntaxTests` - EL language reference (1 test case)
- `TestProject` - Minimal template

---

## Version 4.x (Historical)

See git history for earlier changes.

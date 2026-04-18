# DTRules Changelog

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

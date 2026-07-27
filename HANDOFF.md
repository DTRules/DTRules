# HANDOFF — DTRules

**Generated:** 2026-07-21
**Reason:** 76-fun Thelio to System76 for NVMe repair; work continues elsewhere.
**Repo path on origin machine:** /home/paul/go/src/github.com/DTRules/DTRules
**Remote:** `origin` https://github.com/DTRules/DTRules.git (fetch + push);
`paulsnow-fork` https://github.com/PaulSnow/DTRules.git (fetch + push)

## 1. What this repo is

DTRules is a decision-table rules engine. **It is a Go project, not Java** — despite the
project's Java lineage, this tree is Go-first (`go.mod` module `github.com/DTRules/DTRules`,
Go 1.24, ANTLR + excelize + x/crypto). There is no `pom.xml` or `build.xml` anywhere; the only
Java residue is `.gitignore` entries. `legacy/` holds archived **Go** code, not Java. Business
rules are authored in Excel (or through a JSON authoring API), compiled through an ANTLR-based
Expression Language (EL) → postfix bytecode compiler, and executed on a stack-based VM with an
amd64 assembly fast path. Front-ends are a CLI (`cmd/dtrules`), an HTTP API (`cmd/api`), a
React UI (`ui/`), and embeddable library use. Includes a 256-bit fixed-point decimal type for
token/staking math and a manual embedded in the binary (`dtrules docs`).

## 2. Current work in flight

**Nothing is uncommitted or half-finished in the working tree.** `main` is clean and exactly in
sync with `origin/main`; the only untracked entries are a git worktree directory and an empty
`excel/` stub (see §3). The repo was left at a clean release boundary.

What was *actively* being worked on, from the last 15 commits and CHANGELOG:

- **The "staking gate-4 unblock" — v1.19.0, tagged/landed 2026-07-11 (HEAD, `d3d75cdc8`).**
  Three EL code-generation bugs filed while implementing the staking recipient-aggregation and
  budget decision tables were fixed with execution tests for each:
  - `#903` — fixed-point dispatch for `fexpr` mul/div; the six mul/div visitors emitted
    `fmul`/`fdiv` unconditionally, so `divide … rounding by` dividends silently degraded to
    double math above 2^53. Now routed through the promote/`cvfp`/dispatch path.
  - `#904` — in-action entity building (four defects that compiled clean and failed at runtime):
    scoped locals in action bodies, `create T as <alias>`, `add new T entity to coll`
    double-emitting its `swap addto` trailer, plus new `lowercase of` / `uppercase of` surfaces.
  - `#869` — `there is <x> in <array> where <p>` crashed at runtime because grammar entity
    alternatives shadow the array alternatives; array operands now route to the OR-accumulator
    `forall` fold.
- **The larger arc this sits in:** a months-long **EL → postfix codegen correctness campaign**.
  v1.18.0 (2026-07-02) was a broad cluster of type-dispatch / operand-order / unregistered-operator
  fixes (`#876 #877 #882 #884 #888 #889 #890 #894 #897 #898 #899`), several of which silently
  produced wrong results, plus a durable execution-test and emit-consistency guard layer. The
  method that found these was auditing the emitter against the operator registry and testing by
  compile→run→assert rather than token presence — keep using it.
- **The `#803` sweep** (12 numbered "batch" branches, `feat/issue-803-batch-*`) systematically
  replaced dead/default grammar visitors. Batches 1–12 exist as branches; several were merged via
  squash PRs (#813, #815, #822, #830). Anything of `#803` still open lives on those branches, not
  in the working tree.
- **Named as in-progress by `.claude/CLAUDE.md`, and NOT yet landed:** extraction of a
  `pkg/dtrules/sdk` package for embeddable engine wiring (**`#757`**). Until it lands,
  `cmd/dtrules` and `cmd/api` each glue the engine pipeline together independently. This is the
  most obvious next structural task.
- **Parked side-branch:** the git worktree at `.claude/worktrees/agent-a706f45f12675c99f` sits on
  `website-refresh-v1.16.0` (`5c1dce9f2`, "style(website): formatting pass — nav fit + hero
  refresh (#861)"), clean and in sync with its remote. Website refresh work, unmerged to main.
- There are **60 local branches**, ~40 of them unmerged into `main` (feature, fix, docs, and
  per-version changelog branches). Most correspond to already-squash-merged PRs and are stale;
  do not assume an unmerged branch means unfinished work without checking the PR.

## 3. State at handoff

- **Branch:** main (upstream: origin/main — in sync, 0 ahead / 0 behind)
- **Uncommitted changes:** 2 untracked paths, 0 modified tracked files
  - `.claude/worktrees/` — contains the linked git worktree `agent-a706f45f12675c99f` checked out
    on branch `website-refresh-v1.16.0` (clean, pushed). Not repo content; a Claude agent worktree.
  - `excel/` — effectively empty; contains only a `.gitignore` (72 bytes). No work in it.
- **Unpushed commits:** 1
```
70f8dffcf docs: v1.14.0 changelog — strict loader + dtrules compile + advisory wiring (#780, #782, #785)
```
  This sits on branch `chore/v1.14.0-changelog`, whose upstream is marked **gone** — the remote
  branch was deleted after its PR was squash-merged, so this commit's content is almost certainly
  already in `main` under a different SHA. Nothing on `main` is unpushed.
- **Recent commits:**
```
d3d75cdc8 docs(changelog): v1.19.0 — staking gate-4 unblock: #903/#904/#869 EL fixes
1ec3bd044 fix(el): in-action entity building — scoped locals, create-as, add-new, case-fold surface (#904) (#907)
cb644b2a3 fix(el): route array operands of there-is-in-where to the forall fold (#869) (#906)
cbacbce59 fix(el): fp dispatch for fexpr mul/div; divide-rounding operands coerced to fixed (#903) (#905)
4cb16a344 docs(changelog): v1.18.0 — EL codegen correctness cluster + test/guard layer
```
  Latest tag: `v1.19.0`. Last commit on `main`: 2026-07-11.

## 4. Claude setup

There **is** a Claude setup, but note it lives at `.claude/CLAUDE.md`, **not** at the repo root —
there is no root `CLAUDE.md`. There is **no `.mcp.json`** in this repo (the CLAUDE.md references
"MCP write tools" for authoring, but no MCP server is configured here).

`.claude/CLAUDE.md` — its actual directives:

- **Project map.** Go-first layout: `cmd/` (CLI + API), `pkg/dtrules/` (authoring, `compiler/el`,
  `decisiontable`, `interpreter`, `operators`, `runtime`, `session`, `sync`), `sampleprojects/`,
  `ui/`, `legacy/go/` (archived, ASM-dependent). Notes `pkg/dtrules/sdk` is being extracted (#757).
- **CRITICAL — output redirection.** *Always* redirect command output to log files
  (`go build ./... > /tmp/go-build.log 2>&1`, same for `go test`, `make build`, `git push`) and
  read them back with `tail -50`. "NEVER run builds or tests without redirecting output" —
  verbose output crashes the AI context. Git status/add/commit are exempt.
- **Use `dtrules docs`** (embedded manual: `xml-format`, `decision-tables`, `operators`, `sdk`,
  `examples`, `workflow`) instead of guessing at DTRules concepts.
- **The authoring contract (hard rules).** Excel is the system of record for DSL; `postfix` is a
  compiled artifact. **Never hand-edit XML** (it is generated). **Never hand-write postfix.**
  There are exactly two legal ways to change a rule: edit Excel then `dtrules build`, or call the
  authoring API (`dtrules table put` / `dtrules edd put`, MCP write tools) which writes XML,
  compiles postfix, *and* updates Excel in the same operation. **`dtrules compile` does not
  exist** — it was removed. Full spec: `docs/authoring-contract.md`.
- **State tax implementation rules.** Each US state gets exactly two files,
  `sampleprojects/TaxReturn/xml/states/XX_edd.xml` and `XX_dt.xml`, copied from the `TEMPLATE_*`
  files, to avoid merge conflicts. **Never edit the main `TaxReturn_edd.xml` / `TaxReturn_dt.xml`
  directly, and never commit the merged files** — run `sampleprojects/TaxReturn/scripts/merge-states.sh`
  to merge locally before testing. Research only the rates/deductions/exemptions you need, and add
  3 test cases per state under `testfiles/TestScenarios/State/XX/`.
- **Definition of done.** `make check` must pass before declaring any task complete; scoped tests
  are explicitly *not* sufficient.
- **Commit convention** and a "when to ask for help" list (see §5).

`.claude/settings.local.json` — one thing only: a **PreToolUse hook** matching `Bash` with
`if: Bash(gh pr merge *)` that runs `bash ./scripts/pre-merge-guard.sh` (10 s timeout, status
message "Checking PR verification sentinel"). The guard denies `gh pr merge <N>` unless
`scripts/merge-pr.sh` has verified that PR's tests within the last 10 minutes (it checks a
sentinel `/tmp/.dtrules-merge-verified-<N>`). `merge-pr.sh`'s own header explains why: "Claude has
merged PRs with failing tests 3+ times per session otherwise." It fetches the PR head into a
disposable worktree, runs `go build ./...` and a fast `go test` subset, aborts on failure, and
only then squash-merges with `--admin --delete-branch`.

There are **no** `.claude/agents/`, `.claude/skills/`, or `.claude/commands/` directories, and no
`.claude/settings.json` — only `CLAUDE.md`, `settings.local.json`, and the `worktrees/` directory.

**Important:** `scripts/` is **git-ignored** (`.gitignore` line 79). So `scripts/pre-merge-guard.sh`
and `scripts/merge-pr.sh` — the two files the hook depends on — **are not in git and will not
arrive via `git clone`.** See §6.

## 5. Directives and conventions

Verified against `Makefile`, `go.mod`, `.github/workflows/`, `README.md`, `.claude/CLAUDE.md`:

- **Build:** `make build` → `./build/dtrules` (injects version/commit/branch/date ldflags).
  `make install` → GOBIN. `make build-all` / `make release` cross-compile linux/darwin/windows
  (release also writes `dist/checksums.txt`). `make version` prints the injected values.
- **The gate:** `make check` — runs `go build ./...` (full module), then `go vet` over a curated
  package list (`compiler/el`, `interpreter`, and `pkg/dtrules/cmd` are excluded for documented
  reasons: ANTLR-generated unreachable-code noise, asm extern stubs, a pre-existing test ref),
  then `go test -count=1 ./...` across the whole module. Legacy failing tax/forall tests are
  archived behind the `archive` build tag (`go test -tags archive ./...`, tracked in `#520`), so
  the default suite is green end to end. **`make check` passing is the definition of done** —
  README and CLAUDE.md both say so.
- **Tests:** `go test ./...` or `make test`; scoped example `go test ./pkg/dtrules/... -run TestTaxReturn`.
  Always redirect to a log file (see §4).
- **CI:** `.github/workflows/` has `go-tests.yml`, `verify.yml` (runs `dtrules verify` over
  `sampleprojects/*` on PRs touching samples/excel/sync/CLI; TaxReturn is deliberately skipped as
  an archived legacy sample, `#872`), `release.yml`, `deploy.yml`, `deploy-api.yml`,
  `deploy-sinusitis.yml` (Fly.io — see `fly.toml`, `fly.sinusitis.toml`, `Dockerfile*`).
- **Authoring workflow:** analysts edit Excel → `dtrules build` → `dtrules verify`; developers/AI
  use `dtrules table put` / `dtrules edd patch` (write-through to Excel) → `dtrules verify`.
  Never hand-edit generated XML or postfix.
- **UI:** `go run ./cmd/api` for the backend, then `cd ui && npm install && npm run dev` for the
  React frontend at **http://localhost:5173** (machine-local; see §6).
- **Commit convention:** conventional commits with a scope and an issue number, e.g.
  `fix(el): <summary> (#903)`, `feat(compiler/el): …`, `docs(changelog): …`, `test(el): …`.
  Squash-merged PRs carry a trailing `(#PR)` in addition to the issue number.
- **Branch policy:** one branch per issue, named `fix/issue-<N>`, `feat/issue-<N>-<slug>`,
  `docs/<slug>`, or `chore/<slug>`; PR to `main`; squash-merge with branch deletion via
  `scripts/merge-pr.sh` (direct `gh pr merge` is blocked by the hook).
- **Ask for help when:** a rule domain is genuinely unclear, official source data can't be found,
  tests fail with unclear errors after checking logs, or context is filling despite redirection.
- There is no `CONTRIBUTING.md`. `NOTICE` carries the license notice.

## 6. Resuming on another machine

1. **Prefer the external-drive copy over a fresh clone**, because several things you need are
   git-ignored and exist only in the working tree:
   - `scripts/` (whole dir ignored) — `merge-pr.sh` and `pre-merge-guard.sh`, required by the
     Claude PreToolUse hook. Without them, every `gh pr merge` will fail the hook.
   - `plans/`, `docs-dev/`, `benchmark-results/` (ignored working-doc dirs).
   - `.dtrules/` (local `dtrules review` cache — regenerable, safe to drop).
   Alternatively clone fresh and re-create `scripts/` by hand:
   `git clone https://github.com/DTRules/DTRules.git` then restore those two scripts.
2. **Do NOT copy** these — they are large, machine-built, and regenerable: the `dtrules` (42 MB),
   `api` (26 MB) and `dtrules.test` (23 MB) binaries at the repo root, `build/`, `dist/`, and
   `.venv/` (a Python virtualenv with absolute paths baked into it — it will not work after a
   move; delete and recreate if anything needs it).
3. **The `.claude/worktrees/agent-a706f45f12675c99f` worktree will not survive a plain copy.** A
   linked worktree stores absolute paths in `.git` and in `.git/worktrees/<name>/gitdir`. If the
   repo lands at a different absolute path, run `git worktree repair` from the main checkout, or
   simply `git worktree remove` it and re-create it — its branch `website-refresh-v1.16.0` is
   fully pushed to origin, so nothing is lost.
4. Check out `main` (that is where HEAD is). `git fetch --all --prune` first; expect many stale
   local branches whose remotes are gone.
5. Install **Go 1.24+** (`go.mod` declares go 1.24.0 / toolchain go1.24.4). Then
   `go mod download > /tmp/mod.log 2>&1`. For the UI only, install Node/npm.
   ANTLR-generated parser code is committed — you do not need the ANTLR tool unless regenerating
   the grammar.
6. **Verify the build:** `make check > /tmp/make-check.log 2>&1 && tail -20 /tmp/make-check.log`.
   That is the authoritative gate; it should pass green from a clean tree at `d3d75cdc8`. Then
   `make build` and sanity-check with `./build/dtrules docs`.
   (Redirect output — per `.claude/CLAUDE.md`, never run builds or tests unredirected.)
7. **Pick up from §2:** the tree is at a clean v1.19.0 boundary, so start new work rather than
   finishing something. The two obvious continuations are (a) the `pkg/dtrules/sdk` extraction,
   `#757`, which `.claude/CLAUDE.md` still lists as pending and which removes the duplicated
   engine wiring in `cmd/dtrules` and `cmd/api`; and (b) the next staking gate — v1.19.0 only
   unblocked gate 4, and the staking recipient-aggregation/budget tables are what surfaced
   `#903/#904/#869`. Check open GitHub issues for what came after `#907`.
8. **Machine-specific things that will NOT transfer:**
   - **Credentials were removed from the origin machine (`~/.ssh`, `~/.aws`, `~/.gnupg` deleted).**
     Both remotes here are **HTTPS**, so anonymous fetch/clone of these public repos works, but
     **pushing, and every `gh` command the workflow depends on (`gh pr merge`, `merge-pr.sh`),
     requires you to re-authenticate `gh` on the new machine** (`gh auth login`). No SSH key and
     no GPG signing key will be present — if any commits were being signed, that will now fail.
   - Fly.io deploys (`fly.toml`, `fly.sinusitis.toml`, `deploy-*.yml`) need a `FLY_API_TOKEN` that
     is not in the repo; deploys run from GitHub Actions secrets, not from this machine.
   - `http://localhost:5173` (Vite dev server) and the local `cmd/api` backend are local-only.
   - The merge guard's sentinel path `/tmp/.dtrules-merge-verified-<N>` is per-machine and
     ephemeral — fine, it regenerates, but note `pre-merge-guard.sh` requires `jq` to be installed.
   - `.dtrules-manifest.json` / `.sync-manifest.json` are per-machine local sync state (ignored).
   - Absolute path `/home/paul/go/src/github.com/DTRules/DTRules` appears in the worktree wiring
     (see step 3) — nothing in the Go source hardcodes it.

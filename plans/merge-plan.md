# DTRules Repository Merge Plan

## Current State Analysis

### Remotes
| Remote | URL | Status |
|--------|-----|--------|
| origin | github.com/DTRules/DTRules | Active development |
| paulsnow | github.com/DTRules/DTRules | Legacy (101 commits behind) |
| eembach | github.com/eembach/DTRules | Contributor fork |

### Key Finding: PaulSnow/DTRules is NOT a concern
- `paulsnow/5.0-SNAPSHOT` has **0 unique commits** vs origin
- `origin/5.0-SNAPSHOT` has **101 commits ahead** of paulsnow
- Common ancestor: `32b4c0a` (paulsnow is simply the old version)
- **Action**: No merge needed. PaulSnow repo can be archived or updated via force-push.

---

## Branch Relationships

### Main Development Branches
```
origin/main ─────────────────────────────────────────────▶
                                                          │
                                              ┌───────────┴─── 2 unique commits
                                              │                (trace-analysis-tools)
origin/5.0-SNAPSHOT ──────────────────────────┴───────────▶
                                                          │
                                              ┌───────────┴─── 9 unique commits
                                              │                (assembly, benchmarks)
                                              │
issue-31-asm-optimization (local HEAD) ───────┴───────────▶
                                                          │
                                              ┌───────────┴─── 9 commits + local changes
                                              │
```

### Divergence Summary
| Branch | Unique Commits | Status |
|--------|---------------|--------|
| origin/main | 2 (trace tools) | Needs assembly work |
| origin/5.0-SNAPSHOT | 9 (assembly, benchmarks) | Primary development |
| issue-31-asm-optimization | 9 ahead of 5.0-SNAPSHOT | Local uncommitted changes |

---

## Local Changes (Current Working Directory)

### Uncommitted Changes (65+ files)
The local changes include:
1. **Import path fix**: `github.com/DTRules/DTRules` → `github.com/DTRules/DTRules` (65 Go files)
2. **Documentation updates**: README.md, docs/*.md
3. **Other modifications**: See `git status` for full list

### Untracked Files
- `plans/*.md` - Planning documents
- `sampleprojects/CHIP/lib/*.jar` - Updated POI libraries
- `docs/staking-summary-periods-140-164.md`

---

## Dirty Worktrees

| Worktree | Dirty Files | Type |
|----------|------------|------|
| issue-3 | 4 | Generated parser files (can discard) |
| issue-21 | 4 | Generated parser files (can discard) |
| issue-22 | 1 | package-lock.json (can regenerate) |
| issue-40 | 5 | **Real code changes - needs review** |

---

## Recommended Merge Plan

### Phase 1: Commit Local Import Path Changes
1. Stage all import path changes in current working directory
2. Create a commit: "fix: Update module path to github.com/DTRules/DTRules"
3. This preserves the critical fix

```bash
# Stage import path changes
git add go/go.mod go/README.md go/cmd/ go/pkg/
git add README.md docs/*.md

# Commit
git commit -m "fix: Update Go module path from PaulSnow to DTRules

- Update go.mod module declaration
- Update all import statements in 65+ Go files
- Update documentation references

This aligns the module path with the canonical repository location."
```

### Phase 2: Sync main and 5.0-SNAPSHOT
Option A (Recommended): Merge 5.0-SNAPSHOT into main
```bash
git checkout main
git merge origin/5.0-SNAPSHOT -m "Merge 5.0-SNAPSHOT assembly and benchmark work"
git push origin main
```

Option B: Keep them separate (if main is for releases only)

### Phase 3: Update issue-31-asm-optimization
```bash
git checkout issue-31-asm-optimization
git push origin issue-31-asm-optimization
```

### Phase 4: Clean Up Worktrees
```bash
# Discard generated files in issue-3, issue-21
cd /home/paul/go/src/github.com/DTRules/DTRules-worktrees/issue-3
git checkout -- .

cd /home/paul/go/src/github.com/DTRules/DTRules-worktrees/issue-21
git checkout -- .

# Regenerate package-lock in issue-22
cd /home/paul/go/src/github.com/DTRules/DTRules-worktrees/issue-22
git checkout -- ui/package-lock.json

# REVIEW issue-40 changes before discarding
cd /home/paul/go/src/github.com/DTRules/DTRules-worktrees/issue-40
git diff  # Review these changes
```

### Phase 5: Update PaulSnow Remote (Optional)
If you have push access to PaulSnow/DTRules:
```bash
git push paulsnow origin/5.0-SNAPSHOT:5.0-SNAPSHOT --force
```
Or simply archive that repository with a note pointing to DTRules/DTRules.

---

## Decision Points

### Question 1: What to do with uncommitted changes beyond import fix?
The `git status` shows many modified files. Options:
- A) Commit them all together with import fix
- B) Separate into logical commits (import fix, then other changes)
- C) Stash non-import changes for later review

### Question 2: What is the relationship between main and 5.0-SNAPSHOT?
- Is `main` the release branch and `5.0-SNAPSHOT` development?
- Should they be merged or kept separate?

### Question 3: What to do with issue-40 worktree changes?
Files modified:
- `go/cmd/dtrules/main.go`
- `go/pkg/dtrules/interpreter/vm.go`
- `go/pkg/dtrules/testsupport/coverage.go`
- `go/pkg/dtrules/testsupport/testsupport_test.go`
- `go/pkg/dtrules/trace/trace.go`

These look like real development work - need to decide if they should be committed.

---

## Summary

| Task | Priority | Risk |
|------|----------|------|
| Commit import path fix | HIGH | LOW - straightforward |
| Push issue-31 branch | HIGH | LOW - already tracked |
| Merge 5.0-SNAPSHOT → main | MEDIUM | LOW - fast-forward possible |
| Clean worktrees | LOW | LOW - mostly generated files |
| Update PaulSnow repo | LOW | NONE - can ignore |

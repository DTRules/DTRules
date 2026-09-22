#!/usr/bin/env bash
# merge-pr.sh [--check] <PR> [<PR> ...]
#
# The only way a PR reaches main. For each PR, in the order given:
#
#   1. Squash the PR's head onto the current origin/main in a disposable
#      worktree — the tree that will exist after the merge, not the PR alone.
#   2. If that conflicts, rebase the PR branch onto main and force-push it
#      (with lease):
#        - leading commits that belong to another issue already on main are
#          dropped first (a branch stacked on a PR that has since been
#          squash-merged);
#        - a conflict in a .md file where one side is the common base plus
#          added text is resolved by keeping both;
#        - a conflict only in the generated ANTLR files, with EL.g4 merged
#          cleanly, is resolved by regenerating the parser.
#      Anything else stops the run for a human.
#   3. Regenerate the parser from the merged EL.g4 and stop if the committed
#      generated files differ (a textually clean merge of el_parser.go can
#      still disagree with the merged grammar).
#   4. make check, dtrules verify on every sample with excel/, and
#      dtrules validate sampleprojects/CHIP — the local equivalent of CI.
#   5. Wait until GitHub reports the tested head, then
#      gh pr merge --squash --match-head-commit <tested sha>.
#
# --check runs steps 1, 3 and 4 only: nothing is pushed or merged, and a PR
# that needs a rebase is reported instead of rebased.
#
# Stops at the first PR that fails. Prints one line per PR; the detail is in
# the per-PR log under $LOGDIR.
#
# Environment:
#   DTRULES_ANTLR_JAR  antlr-4.13.1-complete.jar (default ~/tools/…). Without
#                      it, step 3 and the parser-regeneration fix are skipped
#                      with a warning.
#   LOGDIR             where logs go (default ${TMPDIR:-/tmp}/dtrules-merge-pr)

set -uo pipefail

CHECK_ONLY=0
if [[ "${1:-}" == --check ]]; then CHECK_ONLY=1; shift; fi
if [[ $# -lt 1 ]]; then
  echo "Usage: $0 [--check] <PR_NUMBER> [<PR_NUMBER> ...]" >&2
  exit 2
fi
for n in "$@"; do
  [[ "$n" =~ ^[0-9]+$ ]] || { echo "not a PR number: $n" >&2; exit 2; }
done

REPO=$(git rev-parse --show-toplevel) || exit 2
LOGDIR=${LOGDIR:-${TMPDIR:-/tmp}/dtrules-merge-pr}
ANTLR_JAR=${DTRULES_ANTLR_JAR:-$HOME/tools/antlr-4.13.1-complete.jar}
EL=pkg/dtrules/compiler/el
mkdir -p "$LOGDIR"
WT=$(mktemp -d -t dtrules-merge-pr-XXXX)
DTRULES_BIN="$WT.bin/dtrules"

cleanup() {
  git -C "$REPO" worktree remove --force "$WT" > /dev/null 2>&1 || true
  git -C "$REPO" worktree prune > /dev/null 2>&1 || true
  git -C "$REPO" for-each-ref --format='%(refname:short)' refs/heads/merge-pr/ \
    | xargs -r git -C "$REPO" branch -q -D > /dev/null 2>&1 || true
  rm -rf "$WT" "$WT.bin"
}
trap cleanup EXIT

have_antlr() { [[ -f "$ANTLR_JAR" ]] && command -v java > /dev/null; }
[[ -f "$ANTLR_JAR" ]] || echo "warning: $ANTLR_JAR not found; generated-parser checks are skipped" >&2

regen_parser() {
  (cd "$EL" && java -jar "$ANTLR_JAR" -Dlanguage=Go -visitor -listener -package el EL.g4)
}

# md_union FILE: resolve diff3 conflicts where one side is the base plus pure
# additions before or after it, by applying the other side's edit in place.
# An empty base is the both-added case. Non-zero on anything else.
md_union() {
  python3 - "$1" <<'PY'
import re, sys
p = sys.argv[1]; s = open(p).read()
pat = re.compile(r'<<<<<<< [^\n]*\n(.*?)\|\|\|\|\|\|\| [^\n]*\n(.*?)=======\n(.*?)>>>>>>> [^\n]*\n', re.S)
bad = []
def res(m):
    raw_o, _, raw_t = m.groups()
    trail = '\n\n' if raw_o.endswith('\n\n') or raw_t.endswith('\n\n') else '\n'
    ours, base, theirs = (g.strip('\n') for g in m.groups())
    def gap(a, b):
        a, b = a.strip('\n'), b.strip('\n')
        return (a + '\n\n' + b if a and b else a or b) + trail
    if not base:
        return gap(ours, theirs)
    for x, y in ((ours, theirs), (theirs, ours)):
        if x.endswith(base):
            return gap(x[:-len(base)], y)
        if x.startswith(base):
            return gap(y, x[len(base):])
    bad.append(1); return m.group(0)
s2 = pat.sub(res, s)
if bad or not pat.search(s) or '<<<<<<<' in s2: sys.exit(1)
open(p, 'w').write(s2)
PY
}

# rebase_branch N BR LOG: rebase origin/BR onto origin/main as local merge-pr/N.
rebase_branch() {
  local N=$1 BR=$2 LOG=$3
  git checkout -q -B "merge-pr/$N" "origin/$BR"

  # Drop leading commits that belong to other issues already on main.
  local OWN UP c subj tag
  OWN=$(grep -oE '[0-9]+' <<<"$BR" | head -1)
  UP=origin/main
  for c in $(git rev-list --reverse origin/main..HEAD); do
    subj=$(git log -1 --format=%s "$c")
    [[ -n "$OWN" ]] && grep -q "#$OWN\b" <<<"$subj" && break
    tag=$(grep -oE '#[0-9]+' <<<"$subj" | head -1)
    if [[ -n "$tag" ]] && git log origin/main --format=%s | grep -q "$tag\b"; then
      UP=$c
    else
      UP=origin/main; break
    fi
  done
  [[ "$UP" == origin/main ]] || echo "PR $N: dropping inherited commits up to $(git log -1 --format='%h %s' "$UP" | cut -c1-70)"

  git -c merge.conflictStyle=diff3 rebase --onto origin/main "$UP" >> "$LOG" 2>&1
  local U f ok regen
  while [[ -d "$(git rev-parse --git-path rebase-merge)" ]]; do
    U=$(git diff --name-only --diff-filter=U)
    ok=1; regen=0
    for f in $U; do
      case "$f" in
        *.md) if md_union "$f"; then git add "$f"; else ok=0; fi ;;
        $EL/EL.interp|$EL/ELLexer.interp|$EL/*.tokens|$EL/el_*.go) regen=1 ;;
        *) ok=0 ;;
      esac
    done
    if [[ $ok == 1 && $regen == 1 ]]; then
      if have_antlr && ! grep -q '<<<<<<<' "$EL/EL.g4" && regen_parser >> "$LOG" 2>&1; then
        git add "$EL"
        echo "PR $N: regenerated the ANTLR parser from the merged EL.g4"
      else
        ok=0
      fi
    fi
    if [[ -z "$U" || $ok == 0 ]]; then
      git rebase --abort
      echo "PR $N: CONFLICT that needs a human: $(tr '\n' ' ' <<<"${U:-?}")"
      return 1
    fi
    echo "PR $N: resolved additive conflict in $(tr '\n' ' ' <<<"$U")"
    GIT_EDITOR=true git -c merge.conflictStyle=diff3 rebase --continue >> "$LOG" 2>&1
  done

  git push -q --force-with-lease="$BR:$(git rev-parse "origin/$BR")" origin "merge-pr/$N:$BR" >> "$LOG" 2>&1 \
    || { echo "PR $N: push of the rebased branch FAILED"; return 1; }
  echo "PR $N: rebased onto main and pushed"
}

cd "$REPO" && git fetch -q origin main || exit 1
git worktree add -q --detach "$WT" origin/main || exit 1
cd "$WT" || exit 1
[[ -d "$REPO/ui/node_modules" ]] && ln -s "$REPO/ui/node_modules" ui/node_modules

for N in "$@"; do
  LOG="$LOGDIR/pr-$N.log"; : > "$LOG"
  BR=$(gh pr view "$N" --json headRefName,state -q 'select(.state=="OPEN") | .headRefName')
  [[ -n "$BR" ]] || { echo "PR $N: not an open PR"; exit 1; }
  git fetch -q origin main "$BR" >> "$LOG" 2>&1 || { echo "PR $N: fetch FAILED ($LOG)"; exit 1; }
  HEAD_SHA=$(git rev-parse "origin/$BR")

  git checkout -q --detach origin/main
  if ! git merge -q --squash "$HEAD_SHA" >> "$LOG" 2>&1; then
    git merge --abort 2> /dev/null; git reset -q --hard origin/main
    [[ $CHECK_ONLY == 1 ]] && { echo "PR $N: does not squash cleanly onto main; needs a rebase (run without --check)"; exit 3; }
    rebase_branch "$N" "$BR" "$LOG" || exit 3
    HEAD_SHA=$(git rev-parse "merge-pr/$N")
    git checkout -q --detach origin/main
    git merge -q --squash "$HEAD_SHA" >> "$LOG" 2>&1 || { echo "PR $N: CONFLICT after rebase ($LOG)"; exit 3; }
  fi
  git commit -q -m "merge-pr $N" >> "$LOG" 2>&1

  if have_antlr; then
    regen_parser >> "$LOG" 2>&1
    if [[ -n "$(git status --porcelain -- "$EL")" ]]; then
      git status --porcelain -- "$EL" >> "$LOG"
      echo "PR $N: the generated parser does not match the merged EL.g4 ($LOG)"; exit 4
    fi
  fi

  start=$(date +%s)
  make check >> "$LOG" 2>&1 || { echo "PR $N: make check FAILED ($LOG)"; exit 5; }
  mkdir -p "$(dirname "$DTRULES_BIN")"
  go build -o "$DTRULES_BIN" ./cmd/dtrules/ >> "$LOG" 2>&1 || { echo "PR $N: CLI build FAILED ($LOG)"; exit 5; }
  vf=0
  for dir in sampleprojects/*/; do
    [[ -d "${dir}excel" ]] || continue
    "$DTRULES_BIN" verify "$dir" >> "$LOG" 2>&1 || { vf=1; echo "verify failed: $dir" >> "$LOG"; }
  done
  [[ $vf == 0 ]] || { echo "PR $N: dtrules verify FAILED ($LOG)"; exit 5; }
  "$DTRULES_BIN" validate sampleprojects/CHIP >> "$LOG" 2>&1 || { echo "PR $N: CHIP validate FAILED ($LOG)"; exit 5; }

  if [[ $CHECK_ONLY == 1 ]]; then
    echo "PR $N: all checks green on main + PR ($(( $(date +%s) - start ))s); not merged (--check)"
    continue
  fi

  # GitHub can take a while to register a force-push; merging before it does
  # is refused as a conflict. Never close/reopen to hurry it: GitHub will not
  # reopen a PR whose branch no longer contains the head it recorded.
  for _ in $(seq 1 30); do
    [[ "$(gh pr view "$N" --json headRefOid -q .headRefOid)" == "$HEAD_SHA" ]] && break
    sleep 10
  done
  [[ "$(gh pr view "$N" --json headRefOid -q .headRefOid)" == "$HEAD_SHA" ]] \
    || { echo "PR $N: GitHub never picked up head $HEAD_SHA"; exit 6; }

  # Sentinel for the optional pre-merge-guard PreToolUse hook.
  date +%s > "/tmp/.dtrules-merge-verified-$N"
  T=$(gh pr view "$N" --json title -q .title)
  gh pr merge "$N" --squash --match-head-commit "$HEAD_SHA" --subject "$T (#$N)" >> "$LOG" 2>&1 \
    || { echo "PR $N: gh pr merge FAILED ($LOG)"; exit 7; }
  echo "PR $N: merged ($(( $(date +%s) - start ))s of checks, all green)"
  git fetch -q origin main
done
[[ $CHECK_ONLY == 1 ]] && echo "all checked" || echo "all merged"

#!/usr/bin/env bash
#
# fetch.sh — download the official corporate-tax forms listed in sources.tsv.
#
# The PDFs themselves are gitignored: they are large, they are republished
# material, and they are reissued every filing season. What the repo keeps is
# sources.tsv (jurisdiction, document, URL) and manifest.tsv (what was actually
# retrieved, when, and its sha256), so any claim traced to a form can be
# re-verified against the same bytes.
#
# Usage:  ./fetch.sh [STATE ...]      # default: every state in sources.tsv
set -uo pipefail

cd "$(dirname "$0")"
SOURCES=sources.tsv
MANIFEST=manifest.tsv
UA="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126 Safari/537.36"

[ -f "$SOURCES" ] || { echo "no $SOURCES"; exit 1; }

want=""
[ $# -gt 0 ] && want="$*"

printf 'jurisdiction\tdocument\tfile\tbytes\tsha256\tretrieved\turl\n' > "$MANIFEST.new"

ok=0; fail=0
while IFS=$'\t' read -r juris doc url; do
    case "$juris" in ''|'#'*|jurisdiction) continue ;; esac
    if [ -n "$want" ] && ! grep -qw "$juris" <<<"$want"; then continue; fi

    mkdir -p "forms/$juris"

    # Derive the filename. Several states end their download URLs with a
    # generic segment — NCDOR uses ".../open", Georgia and Massachusetts use
    # ".../download" — so a plain basename collides: North Carolina's return
    # and its instructions both landed on "forms/NC/open" and the second
    # silently overwrote the first. When the last segment is not a PDF name,
    # fall back to the segment above it, which is the document slug.
    path="${url%%\?*}"
    name="$(basename "$path")"
    case "$name" in
        *.pdf|*.PDF) ;;
        *)  # Name it from the document description. Deriving it from the URL
            # gives "open.pdf", "download.pdf" or "forms.in.gov.pdf" — unique
            # after the collision fix, but meaningless to anyone browsing
            # forms/. The description is unique per row by construction.
            name="$(printf '%s' "$doc" \
                    | tr '[:upper:]' '[:lower:]' \
                    | sed 's#[^a-z0-9]\+#-#g; s#^-##; s#-$##').pdf" ;;
    esac
    out="forms/$juris/$name"

    # `< /dev/null` matters: curl inherits the loop's stdin, which IS
    # sources.tsv, and will happily swallow the next lines of it. That showed
    # up as random rows never being attempted and others failing with
    # "cannot open <file>" because $juris had been mangled mid-read.
    code=$(curl -sSL -A "$UA" --max-time 120 --retry 2 --retry-delay 2 \
                -w '%{http_code}' -o "$out" "$url" < /dev/null 2>/dev/null)
    mime=$(file -b --mime-type "$out" 2>/dev/null)

    # A state site that has retired a form answers 200 with an HTML "not
    # found" page. Size and status are not enough — check it is really a PDF.
    if [ "$code" = "200" ] && [ "$mime" = "application/pdf" ]; then
        sz=$(stat -c%s "$out")
        sha=$(sha256sum "$out" | cut -d' ' -f1)
        printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
            "$juris" "$doc" "$out" "$sz" "$sha" "$(date -u +%Y-%m-%d)" "$url" >> "$MANIFEST.new"
        printf '  ok   %-4s %-34s %8s bytes\n' "$juris" "$doc" "$sz"
        ok=$((ok+1))
    else
        printf '  FAIL %-4s %-34s HTTP %s (%s)\n' "$juris" "$doc" "$code" "$mime"
        rm -f "$out"
        fail=$((fail+1))
    fi
done < "$SOURCES"

# Merge with any previously retrieved rows for states not fetched this run.
if [ -n "$want" ] && [ -f "$MANIFEST" ]; then
    tail -n +2 "$MANIFEST" | while IFS=$'\t' read -r j rest; do
        grep -qw "$j" <<<"$want" || printf '%s\t%s\n' "$j" "$rest" >> "$MANIFEST.new"
    done
fi
{ head -1 "$MANIFEST.new"; tail -n +2 "$MANIFEST.new" | sort -u; } > "$MANIFEST"
rm -f "$MANIFEST.new"

echo
echo "retrieved $ok, failed $fail — manifest: $MANIFEST"

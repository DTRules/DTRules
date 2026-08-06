#!/usr/bin/env python3
"""Derive candidate EL from stored postfix.

The hand-coded postfix in the legacy samples was written before the authoring
API existed, when reaching for postfix directly was easier than getting the EL
right. It is not a trustworthy oracle — a third of CorporateTax's uses
operators that were never registered (`add`, `sub`, `mul`) or has `xdef`
operands reversed, so it could not have executed. But it is a *readable
statement of intent*, and for the rows whose operators are real it is an exact
oracle: the EL is right when it recompiles to the same bytes.

This emits candidates for the shapes that repeat. Feed the output to elcheck,
which reports RESOLVED for a byte-identical match and DIFF otherwise. A DIFF is
not necessarily wrong — where the original is test-first `ifelse` (#943) or uses
untyped arithmetic on doubles, the new postfix *should* differ — but every DIFF
needs a human read.

    python3 tools/elcheck/derive_from_postfix.py sampleprojects/CorporateTax out.json
    go run ./tools/elcheck -project sampleprojects/CorporateTax \
        -exclude CorporateTax_dt.xml -overrides out.json
"""
import glob
import json
import os
import re
import sys
import xml.etree.ElementTree as ET

NAME = r'[A-Za-z_][A-Za-z_0-9.]*'
LIT = r'(?:-?\d+(?:\.\d+)?|"[^"]*"|\'[^\']*\'|true|false)'
TERM = r'(?:%s|%s)' % (LIT, NAME)

CMP = {'f>': '>', 'f<': '<', 'f>=': '>=', 'f<=': '<=', 'f==': '==', 'f!=': '!=',
       '>': '>', '<': '<', '>=': '>=', '<=': '<=', '==': '==', '!=': '!='}


def strip_comments(pf):
    out = []
    for line in pf.splitlines():
        i = line.find('//')
        if i >= 0:
            line = line[:i]
        out.append(line)
    return ' '.join(' '.join(out).split())


def q(tok):
    """Postfix string literals are single- or double-quoted; EL takes either."""
    return tok


def assignments(code):
    """`<value> cv? /<name> xdef` repeated -> `set <name> = <value>;` …

    Only matches when the whole row is assignments of a bare term, which is the
    shape that covers the zeroing rows (state_tax = 0.0, additions = 0.0, …).
    """
    pat = re.compile(r'^(%s)\s+(?:cvd|cvi|cvs|cvb|cve|cvdate)?\s*/(%s)\s+xdef\s*' % (TERM, NAME))
    stmts, rest = [], code
    while rest:
        m = pat.match(rest)
        if not m:
            return None
        stmts.append('set %s = %s;' % (m.group(2), q(m.group(1))))
        rest = rest[m.end():].strip()
    return ' '.join(stmts) if stmts else None


def comparison(code):
    """`<a> <b> f>` -> `a > b`; `<a> "s" streq` -> `a == "s"`."""
    m = re.fullmatch(r'(%s)\s+(%s)\s+(\S+)' % (TERM, TERM), code)
    if not m:
        return None
    a, b, op = m.group(1), m.group(2), m.group(3)
    if op in CMP:
        return '%s %s %s' % (a, CMP[op], b)
    if op in ('streq', 's=='):
        return '%s == %s' % (a, q(b))
    if op == 'beq':
        return '%s == %s' % (a, b)
    return None


def or_shape(code):
    """`A true beq { pop <B-expr> } over not if` -> `A == true or <B>`."""
    m = re.fullmatch(r'(%s)\s+(true|false)\s+beq\s+\{\s*pop\s+(.*?)\s*\}\s+over\s+not\s+if' % NAME, code)
    if not m:
        return None
    inner = comparison(m.group(3))
    if not inner:
        return None
    return '%s == %s or %s' % (m.group(1), m.group(2), inner)


def and_shape(code):
    """`A true beq { pop <B-expr> } over if` -> `A == true and <B>`."""
    m = re.fullmatch(r'(%s)\s+(true|false)\s+beq\s+\{\s*pop\s+(.*?)\s*\}\s+over\s+if' % NAME, code)
    if not m:
        return None
    inner = comparison(m.group(3))
    if not inner:
        return None
    return '%s == %s and %s' % (m.group(1), m.group(2), inner)


def eq_or_shape(code):
    """`A get true eq B get true eq or` — the `get`/`eq` spelling of the same
    disjunction, used by the states written in a later batch."""
    m = re.fullmatch(r'(%s)\s+get\s+(true|false)\s+eq\s+(%s)\s+get\s+(true|false)\s+eq\s+or' % (NAME, NAME), code)
    if m:
        return '%s == %s or %s == %s' % (m.group(1), m.group(2), m.group(3), m.group(4))
    m = re.fullmatch(r'(%s)\s+(true|false)\s+eq\s+(%s)\s+(true|false)\s+eq\s+or' % (NAME, NAME), code)
    if m:
        return '%s == %s or %s == %s' % (m.group(1), m.group(2), m.group(3), m.group(4))
    return None


RULES = (or_shape, and_shape, eq_or_shape, comparison, assignments)


def derive(code, kind):
    for rule in RULES:
        if kind == 'action' and rule is not assignments:
            continue
        if kind != 'action' and rule is assignments:
            continue
        try:
            el = rule(code)
        except Exception:
            el = None
        if el:
            return el
    return None


def main(project, outpath):
    out = {}
    seen = derived = 0
    for path in sorted(glob.glob(os.path.join(project, 'xml/states/*_corp_dt.xml'))):
        if 'TEMPLATE' in path:
            continue
        for dt in ET.parse(path).getroot().iter('decision_table'):
            table = dt.findtext('table_name')
            for kind, label in (('context', 'context'), ('condition', 'condition'), ('action', 'action')):
                for i, d in enumerate(dt.iter(kind + '_details')):
                    dsl = (d.findtext(kind + '_dsl') or '').strip()
                    pf = d.findtext(kind + '_postfix') or ''
                    if dsl or not pf.strip():
                        continue
                    seen += 1
                    code = strip_comments(pf)
                    if not code:
                        continue
                    el = derive(code, kind)
                    if not el:
                        continue
                    derived += 1
                    key = ('%s %d' % (label, i + 1)) if kind == 'context' else ('%s@%d' % (label, i + 1))
                    out.setdefault(table, {})[key] = el
    json.dump(out, open(outpath, 'w'), indent=2, sort_keys=True)
    print('hand rows %d, derived %d (%.0f%%), across %d tables'
          % (seen, derived, 100.0 * derived / max(seen, 1), len(out)))


main(sys.argv[1], sys.argv[2])

#!/usr/bin/env python3
"""Seed EL for CorporateTax's hand-coded rows from their own comments.

Every one of the 413 rows carries a comment shaped like

    <description>; <the EL>

and for actions the EL half is itself the original DSL, which starts with a
`// <description>;` prefix of its own:

    No nexus - no filing required; // No nexus - no filing required; set apportionment.state_tax = 0.0; ...
    Must have nexus to owe tax; apportionment.has_economic_nexus is true or ...

So the candidate EL is the comment with the leading description and any
`// ...;` echo stripped. Emits an elc overrides file keyed by row POSITION —
never by row number, which `table get` renumbers on load.
"""
import glob
import json
import re
import sys
import xml.etree.ElementTree as ET

OUT = {}


ASSIGN = re.compile(r'^[A-Za-z_][A-Za-z_0-9.]*\s*=')


def looks_like_el(s):
    """Reject a comment half that is just prose.

    Many rows carry only a description ("No nexus - no filing required") with
    no EL at all. Seeding that produces a parse error and buries the rows that
    could have been authored, so it is better to leave them unseeded and
    visible."""
    return bool(re.search(r'=|<|>|\bis\b|\bset\b|\bperform\b|\bnot\b', s))


def candidate(comment):
    """Recover the EL half of a '<description>; <EL>' comment."""
    s = ' '.join(comment.split())
    if not s:
        return None
    # Drop a leading '// ...;' echo wherever it appears, then any leading
    # prose up to the first '; ' that precedes something EL-shaped.
    m = re.search(r'//[^;]*;\s*(.*)$', s)
    if m:
        s = m.group(1).strip()
    elif '; ' in s:
        s = s.partition('; ')[2].strip()
    if not s or not looks_like_el(s):
        return None
    # Some rows drop the `set` keyword: "a.b = 0.0; a.c = a.d". Restore it per
    # statement, leaving conditions (which are bare boolean expressions) alone.
    parts = [x.strip() for x in s.split(';') if x.strip()]
    if parts and all(ASSIGN.match(x) for x in parts):
        s = '; '.join('set ' + x for x in parts) + ';'
    return s or None


for path in sorted(glob.glob('sampleprojects/CorporateTax/xml/states/*_corp_dt.xml')):
    if 'TEMPLATE' in path:
        continue
    root = ET.parse(path).getroot()
    for dt in root.iter('decision_table'):
        table = dt.findtext('table_name')
        for kind, label in (('context', 'context'), ('condition', 'condition'), ('action', 'action')):
            for i, d in enumerate(dt.iter(kind + '_details')):
                dsl = (d.findtext(kind + '_dsl') or '').strip()
                pf = (d.findtext(kind + '_postfix') or '').strip()
                if dsl or not pf:
                    continue                       # not a hand row
                el = candidate(d.findtext(kind + '_comment') or '')
                if not el:
                    continue
                key = ('%s %d' % (label, i + 1)) if kind == 'context' else ('%s@%d' % (label, i + 1))
                OUT.setdefault(table, {})[key] = el

json.dump(OUT, open(sys.argv[1], 'w'), indent=2, sort_keys=True)
print('%d rows seeded across %d tables' % (sum(len(v) for v in OUT.values()), len(OUT)))

#!/usr/bin/env python3
"""Declare the EDD fields the CorporateTax state tables reference but never declare.

Writing to an undeclared field is a runtime error, so these references are the
binding constraint on execution — ahead of the EL itself (see PLAN.md Phase 1).

Edits go to the per-state `xml/states/XX_corp_edd.xml` source files, not the
generated merged EDD, which the next merge-states.sh run would overwrite. The
EDD authoring API only targets a project's merged EDD, so for this project the
per-state files are the authoring source and are edited directly.

Type inference, in order of trust:
  1. the same field name declared anywhere already (shouldn't happen — these
     are the undeclared ones — but guards double runs);
  2. the same suffix declared by another state (ca_gross_receipts:double tells
     us mt_gross_receipts is double);
  3. usage evidence from the postfix (`X true beq` → boolean);
  4. name heuristics (has_/is_ → boolean, _code/_formula/_classification →
     string, everything money/rate/threshold-shaped → double).

Run with --dry-run to see the plan without writing.
"""
import collections
import glob
import os
import re
import sys
import xml.etree.ElementTree as ET

PROJECT = 'sampleprojects/CorporateTax'
DRY = '--dry-run' in sys.argv


def strip_code(pf):
    out = []
    for l in pf.splitlines():
        i = l.find('//')
        if i >= 0:
            l = l[:i]
        out.append(l)
    s = ' '.join(' '.join(out).split())
    return re.sub(r'"[^"]*"', ' ', re.sub(r"'[^']*'", ' ', s))


# --- gather declarations -----------------------------------------------------
declared = {}          # field -> type
suffix_types = collections.defaultdict(collections.Counter)   # suffix -> type votes
for p in glob.glob(os.path.join(PROJECT, 'xml/CorporateTax_edd_core.xml')) + \
         glob.glob(os.path.join(PROJECT, 'xml/states/*_corp_edd.xml')):
    if 'TEMPLATE' in p:
        continue
    for f in ET.parse(p).getroot().iter('field'):
        name, typ = f.get('name'), f.get('type') or 'double'
        declared[name] = typ
        m = re.match(r'^[a-z]{2}_(.+)$', name)
        if m:
            suffix_types[m.group(1)][typ] += 1

ents = {e.get('name') for e in
        ET.parse(os.path.join(PROJECT, 'xml/CorporateTax_edd.xml')).getroot().iter('entity')}

# --- gather references with usage evidence ----------------------------------
refs = collections.defaultdict(lambda: {'states': set(), 'entity': collections.Counter(),
                                        'bool_evidence': 0, 'str_evidence': 0,
                                        'array_evidence': 0})
for p in sorted(glob.glob(os.path.join(PROJECT, 'xml/states/*_corp_dt.xml'))):
    if 'TEMPLATE' in p:
        continue
    st = os.path.basename(p).split('_')[0]
    for dt in ET.parse(p).getroot().iter('decision_table'):
        for kind in ('context', 'condition', 'action'):
            for d in dt.iter(kind + '_details'):
                code = strip_code(d.findtext(kind + '_postfix') or '')
                toks = code.split()
                for i, t in enumerate(toks):
                    t2 = t.lstrip('/')
                    if '.' not in t2:
                        continue
                    ent, _, fld = t2.partition('.')
                    if ent not in ents or fld in declared:
                        continue
                    r = refs[fld]
                    r['states'].add(st)
                    r['entity'][ent] += 1
                    nxt = toks[i + 1:i + 3]
                    if nxt[:2] == ['true', 'beq'] or nxt[:2] == ['false', 'beq'] \
                       or (len(nxt) > 1 and nxt[0] in ('true', 'false') and nxt[1] == 'eq'):
                        r['bool_evidence'] += 1
                    if nxt[:1] == ['streq'] or nxt[:1] == ['s==']:
                        r['str_evidence'] += 1
                    # `<value> <target> swap addto` — the target of an addto
                    # is an array, whatever its name suggests.
                    if nxt[:2] == ['swap', 'addto']:
                        r['array_evidence'] += 1

BOOL_RE = re.compile(r'(^|_)(has|is)_')
STR_RE = re.compile(r'_(code|formula|classification|method|status|trail|statements|name|type)$')
INT_RE = re.compile(r'_(count|transactions)$')


def infer(fld, r):
    if fld in declared:
        return declared[fld], 'already declared'
    m = re.match(r'^[a-z]{2}_(.+)$', fld)
    if m and suffix_types.get(m.group(1)):
        typ = suffix_types[m.group(1)].most_common(1)[0][0]
        return typ, 'suffix declared by other states as %s' % typ
    if r['array_evidence']:
        return 'array', 'target of addto in postfix'
    if r['bool_evidence']:
        return 'boolean', 'compared against true/false in postfix'
    if r['str_evidence']:
        return 'string', 'string-compared in postfix'
    if BOOL_RE.search(fld):
        return 'boolean', 'has_/is_ name'
    if STR_RE.search(fld):
        return 'string', 'name heuristic'
    if INT_RE.search(fld):
        return 'integer', 'name heuristic'
    return 'double', 'default: money/rate-shaped'


DEFAULTS = {'double': '0.0', 'boolean': 'false', 'string': '', 'integer': '0', 'array': ''}

# --- plan the insertions -----------------------------------------------------
# field -> (target file, entity, type, why)
plan = []
for fld in sorted(refs):
    r = refs[fld]
    ent = r['entity'].most_common(1)[0][0]
    typ, why = infer(fld, r)
    m = re.match(r'^([a-z]{2})_', fld)
    if m and m.group(1).upper() in {os.path.basename(p).split('_')[0]
                                    for p in glob.glob(os.path.join(PROJECT, 'xml/states/*_corp_edd.xml'))}:
        target = os.path.join(PROJECT, 'xml/states/%s_corp_edd.xml' % m.group(1).upper())
    else:
        # unprefixed / shared fields belong to the federal core EDD
        target = os.path.join(PROJECT, 'xml/CorporateTax_edd_core.xml')
    plan.append((target, ent, fld, typ, why, sorted(r['states'])))

byfile = collections.defaultdict(list)
for target, ent, fld, typ, why, states in plan:
    byfile[target].append((ent, fld, typ, why, states))

print('%d undeclared fields across %d files' % (len(plan), len(byfile)))
if DRY:
    for target in sorted(byfile):
        print('\n## %s' % target)
        for ent, fld, typ, why, states in byfile[target]:
            print('   %-12s %-40s %-8s (%s)' % (ent, fld, typ, why))
    sys.exit(0)

# --- write -------------------------------------------------------------------
FIELD_TMPL = ('    <field name="%s" type="%s" default_value="%s"\n'
              '           comment="Declared from decision-table usage; '
              'referenced by %s but never declared (campaign #948 Phase 1)"/>\n')

for target in sorted(byfile):
    text = open(target, encoding='utf-8').read()
    additions = collections.defaultdict(str)
    for ent, fld, typ, why, states in byfile[target]:
        additions[ent] += FIELD_TMPL % (fld, typ, DEFAULTS[typ], ','.join(states))
    for ent, block in additions.items():
        marker = '<entity name="%s">' % ent
        if marker in text:
            text = text.replace(marker, marker + '\n' + block.rstrip('\n'), 1)
        else:
            # no block for this entity in the file yet — add one before the root close
            root_close = '</entity_data_dictionary>' if '</entity_data_dictionary>' in text \
                else '</entity_dictionary>'
            newblock = '  <entity name="%s">\n%s  </entity>\n\n%s' % (ent, block, root_close)
            text = text.replace(root_close, newblock, 1)
    open(target, 'w', encoding='utf-8').write(text)
    print('wrote %-58s +%d fields' % (target, len(byfile[target])))

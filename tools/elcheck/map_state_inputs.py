#!/usr/bin/env python3
"""Add mapping tags for the per-state taxpayer inputs the tables read.

Phase 1 declared every `result.XX_*` field with a default, so the state tables
load and run — but a scenario cannot *drive* them: without a `<setattribute>`
tag there is no way for input XML to supply a value, so every state computes
from its defaults. That is why only the `apportionment.*` states had runnable
scenarios (PLAN.md Phase 3, "known gap").

Two kinds of field are involved and only one of them belongs in the map:

  taxpayer input  what a filer reports — apportioned income, estimated
                  payments, gross receipts, credits, nexus flags. Varies per
                  return, so it must be mappable.
  jurisdiction    tax rates, bracket thresholds, minimum taxes, statutory
  constant        fees. Fixed by the state's law, correctly carried as the
                  EDD default, and NOT mapped — a return does not get to
                  supply Iowa's tax rate.

The split is by name shape, checked against the classification printed by
--dry-run before anything is written.
"""
import collections
import glob
import os
import re
import sys
import xml.etree.ElementTree as ET

PROJECT = 'sampleprojects/CorporateTax'
MAP = os.path.join(PROJECT, 'xml', 'CorporateTax_map.xml')
DRY = '--dry-run' in sys.argv

# Suffixes a filer reports. Anything not matching is treated as a
# jurisdiction constant and left to its EDD default.
INPUT_SUFFIXES = (
    'apportioned_income', 'estimated_payments', 'gross_receipts',
    'municipal_interest', 'us_interest_income', 'state_credits',
    'state_taxes_deducted', 'state_taxes_paid', 'state_tax_refunds',
    'state_tax_deduction', 'out_of_state_muni_interest',
    'dividends_from_ky_corps', 'qualified_dividend_deduction',
    'section_199a_addback', 'section_199a_deduction',
    'state_income_taxes_paid', 'federal_tax_liability',
    'waters_edge_election', 'unitary_group_count',
    'is_financial_institution', 'is_insurance_company',
    'is_retail_wholesale', 'in_mta_region',
    'business_classification', 'net_worth', 'nol_deduction',
    'total_revenue', 'cogs', 'compensation', 'excluded_receipts',
)
# Nexus flags are prefix-shaped rather than suffix-shaped.
INPUT_PREFIXES = ('has_physical_presence_',)


def is_input(field):
    if field.startswith(INPUT_PREFIXES):
        return True
    bare = re.sub(r'^[a-z]{2}_', '', field)
    return bare in INPUT_SUFFIXES or field in INPUT_SUFFIXES


def strip_code(pf):
    out = []
    for l in pf.splitlines():
        i = l.find('//')
        if i >= 0:
            l = l[:i]
        out.append(l)
    s = ' '.join(' '.join(out).split())
    return re.sub(r'"[^"]*"', ' ', re.sub(r"'[^']*'", ' ', s))


def main():
    decl = {}
    for p in glob.glob(os.path.join(PROJECT, 'xml/**/*_edd.xml'), recursive=True):
        if 'TEMPLATE' in p:
            continue
        for f in ET.parse(p).getroot().iter('field'):
            decl[f.get('name')] = f.get('type') or 'double'

    maptext = open(MAP, encoding='utf-8').read()
    mapped = set(re.findall(r"tag='([^']+)'", maptext))

    written, read = collections.defaultdict(set), collections.defaultdict(set)
    for p in sorted(glob.glob(os.path.join(PROJECT, 'xml/states/*_corp_dt.xml'))):
        st = os.path.basename(p).split('_')[0]
        for dt in ET.parse(p).getroot().iter('decision_table'):
            for kind in ('context', 'condition', 'action'):
                for d in dt.iter(kind + '_details'):
                    for t in strip_code(d.findtext(kind + '_postfix') or '').split():
                        if t.startswith('/') and t[1:].startswith('result.'):
                            written[st].add(t[1:].split('.', 1)[1])
                        elif t.startswith('result.'):
                            read[st].add(t.split('.', 1)[1])

    candidates = set()
    for st in read:
        candidates |= (read[st] - written[st])
    candidates -= mapped

    inputs = sorted(f for f in candidates if is_input(f))
    constants = sorted(f for f in candidates if not is_input(f))

    print('taxpayer inputs to map : %d' % len(inputs))
    print('jurisdiction constants left as EDD defaults: %d' % len(constants))
    if DRY:
        print('\n-- inputs --')
        for f in inputs:
            print('   %-38s %s' % (f, decl.get(f, '?')))
        print('\n-- constants (not mapped) --')
        for f in constants:
            print('   %-38s %s' % (f, decl.get(f, '?')))
        return

    lines = ["\n\t\t\t<!-- Per-state taxpayer inputs (campaign #948). Every field here is\n"
             "\t\t\t     something a filer reports; the jurisdiction's own constants (rates,\n"
             "\t\t\t     bracket thresholds, minimum taxes, statutory fees) are deliberately\n"
             "\t\t\t     absent — they belong to the state's law and stay as EDD defaults. -->\n"]
    for f in inputs:
        lines.append("\t\t\t<setattribute tag='%s' RAttribute='%s' enclosure='result' type='%s'></setattribute>\n"
                     % (f, f, decl.get(f, 'double')))

    src = open(MAP, encoding='utf-8').read().splitlines(keepends=True)
    first_map_end = next(i for i, l in enumerate(src) if '</map>' in l)
    src[first_map_end:first_map_end] = lines
    open(MAP, 'w', encoding='utf-8').write(''.join(src))
    print('wrote %d setattribute entries' % len(inputs))


main()

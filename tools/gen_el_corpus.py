#!/usr/bin/env python3
"""Generate candidate DSL snippets for every labeled alternative in EL.g4.

Output: a TSV with columns (rule, label, kind, dsl).
Reads EL.g4, extracts labels, builds sample DSL per label using a keyword
map from lexer rules and a sample-token map for non-terminals.

kind is the compiler entry point: condition | action | context | raw.
"""
import re
import sys
import os.path

G = '/home/paul/go/src/github.com/DTRules/DTRules/pkg/dtrules/compiler/el/EL.g4'

with open(G) as f:
    src = f.read()
# Strip line + block comments
src = re.sub(r'/\*.*?\*/', '', src, flags=re.DOTALL)
src = re.sub(r'//[^\n]*', '', src)

# --- Keyword terminals (UPPER_NAME : 'string' ;) ---
# Some have WS* between parts; we'll read multi-part forms and collapse WS.
kw = {}
for m in re.finditer(r'^([A-Z][A-Z0-9_]*)\s*:\s*(.*?);', src, flags=re.MULTILINE | re.DOTALL):
    name = m.group(1)
    body = m.group(2).strip()
    # Token body may be an alternation ('a' | 'b' | 'c'). Pick the FIRST arm only.
    # Within one arm, multiple consecutive literals ('for' WS* 'all') join with a space.
    first_arm = re.split(r'(?<![\\])\|', body)[0]
    literals = re.findall(r"'([^']*)'", first_arm)
    if literals:
        joined = ' '.join(l for l in literals if l.strip())
        if joined:
            kw[name] = joined

# Common overrides that want specific forms (a few tokens have fragment bits)
kw.update({
    'SEMI': ';',
    'COMMA': ',',
    'DOT': '.',
    'COLON': ':',
    'LCURLY': '{',
    'RCURLY': '}',
    'LPAREN': '(',
    'RPAREN': ')',
    'LBRACKET': '[',
    'RBRACKET': ']',
    'PLUS': '+',
    'MINUS': '-',
    'STAR': '*',
    'SLASH': '/',
    'PERCENT': '%',
    'EQUALS': '=',
    'LESSTHAN': '<',
    'GREATERTHAN': '>',
    'QUESTION': '?',
    'POSSESSIVE': "'s",
    'WS': '',  # whitespace token is implicit
    'ID': 'x',
    'NAME': 'x',
    'INT': '1',
    'FLOAT': '1.0',
    'STRING': '"hello"',
    'DATE': '2020-01-01',
    'BYTES_LITERAL': '0xdeadbeef',
    'HEX': '0xdeadbeef',
    'QUOTED_STR': '"hello"',
})

# --- Non-terminal sample tokens (reasonable defaults for each rule) ---
NT = {
    'arrayExpr':        'accounts',
    'arrayExpr2':       'accounts',
    'iexpr':            '1',
    'fexpr':            '1.0',
    'strexpr':          '"x"',
    'bexpr':            'true',
    'eexpr':            'account',
    'nexpr':            '/tbl',
    'dexpr':            '2020-01-01',
    'bigexpr':          '1',
    'bytesexpr':        '0xab',
    'includeSearch':    '5',
    'number':           '5',
    'typedEntity':      'account',
    'typedLong':        'n',
    'typedDouble':      'd',
    'typedString':      's',
    'typedBoolean':     'b',
    'typedDate':        'dt',
    'typedArray':       'arr',
    'typedTable':       'tbl',
    'typedName':        'nm',
    'typedDecisionTable': 'dt',
    'forallblock':      'for all accounts { set x = 1; }',
    'foreachblock':     'for each acc in accounts { set x = 1; }',
    'ifblock':          'if true then { set x = 1; } endif',
    'firstblock':       'for first of accounts where true then { set x = 1; } endff',
    'arrayList':        'x',
    'operatorlist':     'x',
    'localvariables':   'with local string s;',
    'debugstatement':   'debug "hi"',
    'forallctl':        'for all accounts',
    'forctl':           'for i = 0; i < 3; i++',
    'forfirstctl':      'for first of accounts where true',
    'foreachblock':     'for each acc in accounts { set x = 1; }',
    'contextForTable':  'for all accounts',
    'done':             'condition true',
    'block':            '{ set x = 1; }',
    'statementList':    'set x = 1;',
    'usingblock':       'using account { set x = 1; }',
    'setstatement':     'set x = 1',
    'addtostatement':   'add 1 to x',
    'subtostatement':   'subtract 1 from x',
    'performstatement': 'perform X',
    'datestatement':    'add 1 day to d',
    'randomstatements': 'randomize arr',
    'xmlvaluestatements': 'set attribute "a" = "b" of account',
    'addtodest':        'x',
    'subtodest':        'x',
    'optSemi':          ';',
    'possessiveRef':    "'s",
    'possessiveChain':  "'s",
    'attrList':         'x',
    'attrListAction':   'x',
    'literal':          '1',
    'mapping':          'mapping "key" as s',
    # Helper rules (simple lowercase, not labeled)
    'thereis':          'there is',
    'inthe':            'in',
    'blist':            '"a"',
    'blistIc':          '"a"',
    'aslong':           '',
    'beforeAssign':     '',
    'sesubstr':         '',
    'paramarr':         '',
    'paramsize':        '',
}

label_re = re.compile(r'#\s*([A-Za-z_][A-Za-z0-9_]*)\s*$')
rule_re = re.compile(r'(?ms)^([a-z][A-Za-z0-9_]*)\s*(?:\[[^\]]*\])?\s*:\s*(.*?);', re.MULTILINE)

def substitute(alt_body: str) -> str:
    """Convert grammar alternative body to candidate DSL."""
    # Token stream with literals intact
    out = []
    # Split keeping quoted strings intact
    i = 0
    toks = []
    while i < len(alt_body):
        c = alt_body[i]
        if c.isspace():
            i += 1
            continue
        if c == "'":
            j = alt_body.find("'", i+1)
            if j < 0: j = len(alt_body)
            toks.append(alt_body[i:j+1])
            i = j+1
            continue
        # Identifier (UPPER or lower)
        if c.isalpha() or c == '_':
            j = i
            while j < len(alt_body) and (alt_body[j].isalnum() or alt_body[j] == '_'):
                j += 1
            toks.append(alt_body[i:j])
            i = j
            continue
        # Syntax chars: | ( ) * + ? { } [ ]  — skip optionality/grouping markers
        toks.append(c)
        i += 1

    # Simplify: drop ANTLR metachars and consume optional/grouped constructs
    # We'll do a naive pass: '(' begins a group, ')' ends; '?' / '*' / '+' on the preceding item.
    # For a sample DSL we include OPTIONAL content (so ? means "include it once").
    # Groups are flattened.
    parts = []
    for t in toks:
        if t in ('(', ')', '?', '*', '+', '|', '~', '='):
            continue
        if t.startswith("'"):
            parts.append(t[1:-1])
        elif re.fullmatch(r'[A-Z][A-Z0-9_]*', t):
            parts.append(kw.get(t, t.lower()))
        elif re.fullmatch(r'[a-z][A-Za-z0-9_]*', t):
            parts.append(NT.get(t, t))
        else:
            parts.append(t)
    # Collapse whitespace
    return re.sub(r'\s+', ' ', ' '.join(p for p in parts if p)).strip()

# Classify entry point per rule by BFS from the three entry points in `done`.
# done's ACTION branches → kind=action; CONDITION → condition; CONTEXT → context.
# This is a rough classifier; refine by examining `done` body.
def classify_done_alts():
    m = re.search(r'(?ms)^done\s*:\s*(.*?);', src)
    body = m.group(1)
    kinds = {}   # label -> kind
    reachable = {'action': set(), 'condition': set(), 'context': set(), 'policy': set()}
    for alt in re.split(r'\n\s*\|', body):
        lab = re.search(r'#\s*(\w+)\s*$', alt.rstrip())
        if not lab: continue
        label = lab.group(1)
        txt = alt[:lab.start()]
        if 'ACTION' in txt:
            kinds[label] = 'action'
        elif 'CONDITION' in txt:
            kinds[label] = 'condition'
        elif 'CONTEXT' in txt:
            kinds[label] = 'context'
        elif 'POLICYSTATEMENT' in txt:
            kinds[label] = 'policy'
    return kinds

# Build ancestry: which top-level category reaches each rule. Start from entry children.
# Heuristic shortcut: examine rule body to decide. Simpler: classify by rule name.
def classify_rule(rule: str) -> str:
    # Context-reachable rules: forallctl etc. are wrapped in a `context` prefix.
    context_rules = {
        'forallctl', 'forctl', 'forfirstctl', 'contextForTable',
        'localvariables', 'debugstatement',
    }
    if rule in context_rules:
        return 'context'
    action_rules = {
        'setstatement', 'addtostatement', 'subtostatement', 'performstatement',
        'datestatement', 'randomstatements',
        'xmlvaluestatements', 'addtodest', 'subtodest', 'block', 'statementList',
        'foreachblock', 'foreachctl', 'forblock', 'firstblock',
        'usingblock', 'ifblock', 'ifstatement',
        'statement', 'operatorlist',
        'xmlvalues', 'xmlvalue', 'done_action',
    }
    if rule in action_rules:
        return 'action'
    if rule == 'done':
        return 'done'
    # Expressions default to condition-embedded tests
    return 'condition'

def wrap_for(kind: str, dsl: str, rule: str, label: str) -> (str, str):
    """Return (entry_kind, dsl_to_compile). entry_kind is one of
    condition/action/context/raw."""
    if rule == 'done':
        return 'raw', dsl  # full done alt
    if kind == 'context':
        # CompileContext auto-prepends "context ".
        # For localvariables, drop trailing ';' because CompileContext doesn't want it.
        d = dsl.rstrip(';').strip()
        return 'context', d
    if kind == 'action':
        # Many action rules produce statements — wrap in action context.
        if not dsl.endswith(';'):
            dsl = dsl + ';'
        return 'action', dsl
    if kind == 'condition':
        # If rule IS bexpr, embed directly. Else wrap in a boolean predicate.
        if rule == 'bexpr':
            return 'condition', dsl
        # For iexpr: compare to 0
        if rule in ('iexpr', 'bigexpr'):
            return 'condition', f'{dsl} > 0'
        if rule == 'fexpr':
            return 'condition', f'{dsl} > 0.0'
        if rule == 'strexpr':
            return 'condition', f'{dsl} is equal to "x"'
        if rule == 'dexpr':
            return 'condition', f'{dsl} is equal to 2020-01-01'
        if rule == 'eexpr':
            return 'condition', f'there is {dsl} where true'
        if rule == 'nexpr':
            return 'condition', f'{dsl} EQ /tbl'
        if rule == 'bytesexpr':
            return 'condition', f'{dsl} bytes== 0xab'
        if rule in ('arrayExpr', 'arrayExpr2'):
            return 'condition', f'{dsl} includes account'
        # Fallback
        return 'condition', dsl
    return 'condition', dsl

done_kinds = classify_done_alts()

print('rule\tlabel\tentry\tdsl')
for m in rule_re.finditer(src):
    rule = m.group(1)
    body = m.group(2)
    alts = re.split(r'\n\s*\|', body)
    for alt in alts:
        lab = re.search(r'#\s*(\w+)\s*$', alt.rstrip())
        if not lab: continue
        label = lab.group(1)
        content = alt[:lab.start()].rstrip()
        dsl = substitute(content)
        if rule == 'done':
            entry, compile_dsl = 'raw', dsl
        else:
            kind = classify_rule(rule)
            entry, compile_dsl = wrap_for(kind, dsl, rule, label)
        print(f"{rule}\t{label}\t{entry}\t{compile_dsl}")

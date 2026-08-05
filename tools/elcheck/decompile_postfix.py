#!/usr/bin/env python3
"""Decompile CorporateTax's hand-written postfix into candidate EL.

The stored postfix predates the authoring API and was never executed, so it is
read as *intent*, not as an oracle: a third of it uses operators that were
never registered (`add`, `sub`, `mul`) or has `xdef` operands reversed, and
even the well-formed rows carry the test-first `ifelse` the Go runtime cannot
execute (#943). This walks each row's postfix with a symbolic stack machine
and emits the EL that says what the row meant. elcheck then recompiles every
candidate: byte-identical (RESOLVED) proves the reading where the operators
were real; a DIFF is expected where they were not, and each one is reviewed.

Also repairs the `err` rows — existing DSL that no longer compiles — most of
which are assignments missing the `set` keyword; the rest are prose whose row
postfix is decompiled instead.

    python3 tools/elcheck/decompile_postfix.py sampleprojects/CorporateTax out.json
    go run ./tools/elcheck -project sampleprojects/CorporateTax -overrides out.json
"""
import collections
import glob
import json
import os
import re
import sys
import xml.etree.ElementTree as ET

# ---------------------------------------------------------------- lexing ----

TOKEN = re.compile(r'"[^"]*"|\'[^\']*\'|\S+')


def strip_comment_lines(text):
    out = []
    for line in text.splitlines():
        i = line.find('//')
        if i >= 0:
            line = line[:i]
        if line.strip().startswith('#'):
            line = ''
        out.append(line)
    return '\n'.join(out)


def lex(pf):
    return TOKEN.findall(strip_comment_lines(pf))


def tree(tokens):
    """Nest { ... } blocks."""
    def walk(i):
        out = []
        while i < len(tokens):
            t = tokens[i]
            if t == '{':
                sub, i = walk(i + 1)
                out.append(sub)
            elif t == '}':
                return out, i + 1
            else:
                out.append(t)
                i += 1
        return out, i
    top, _ = walk(0)
    return top


# ----------------------------------------------------------------- nodes ----

class N:
    """Expression node. prec: 0=atom, 1=or, 2=and, 3=cmp, 4=add, 5=mul, 6=unary."""

    def __init__(self, text, prec=0, kind='any'):
        self.text, self.prec, self.kind = text, prec, kind

    def r(self, parent_prec):
        if self.prec and self.prec <= parent_prec:
            return '(%s)' % self.text
        return self.text


def atom(t):
    if t.startswith('"') or t.startswith("'"):
        return N(t, kind='str')
    if t in ('true', 'false'):
        return N(t, kind='bool')
    if re.fullmatch(r'-?\d+(\.\d+)?', t):
        return N(t, kind='num')
    return N(t, kind='id')


ARITH = {'f+': ('+', 4), '+': ('+', 4), 'add': ('+', 4),
         'f-': ('-', 4), '-': ('-', 4), 'sub': ('-', 4),
         'fmul': ('*', 5), '*': ('*', 5), 'mul': ('*', 5),
         'fdiv': ('/', 5), '/': ('/', 5), 'div': ('/', 5)}
CMP = {'f>': '>', '>': '>', 'f<': '<', '<': '<', 'f>=': '>=', '>=': '>=',
       'f<=': '<=', '<=': '<=', 'f==': '==', '==': '==', 'f!=': '!=', '!=': '!=',
       'beq': '==', 'streq': '==', 's==': '==', 'eq': '==', 'req': '==', 'd==': '==',
       'ne': '!=', 'gt': '>', 'lt': '<', 'ge': '>=', 'le': '<='}
CONVS = {'cvd', 'cvi', 'cvb', 'cve', 'cvdate'}


class Conv(N):
    def __init__(self, inner, op):
        N.__init__(self, inner.text, inner.prec, inner.kind)
        self.inner, self.op = inner, op


class StrVal(N):
    def __init__(self, inner):
        N.__init__(self, 'string value of (%s)' % inner.r(0), prec=0, kind='str')
        self.inner = inner


class Cond(N):
    def __init__(self, cond, a, b):
        N.__init__(self, '<cond>', prec=0)
        self.cond, self.a, self.b = cond, a, b


class Fail(Exception):
    pass


# ------------------------------------------------------------- evaluator ----

class Machine:
    def __init__(self, symbols, kind, lenient=False):
        self.symbols = symbols          # entity.field / field -> type
        self.kind = kind                # 'condition' | 'action'
        self.lenient = lenient          # discovery pass: tolerate empty bodies
        self.incomplete = False
        self.stack = []
        self.stmts = []
        self.env = {}                   # fake locals -> current expression
        self.promote = {}               # fake local -> real destination field
        self.copies = {}                # observed `real = <bare temp>` copies

    def pop(self, n=1):
        if len(self.stack) < n:
            raise Fail('stack underflow')
        out = self.stack[-n:]
        del self.stack[-n:]
        return out if n > 1 else out[0]

    def is_temp(self, name):
        return ('.' not in name and name not in self.symbols
                and not name.startswith('/'))

    def push_term(self, t):
        if self.is_temp(t):
            if t in self.promote:
                self.stack.append(N(self.promote[t], kind='id'))
                return
            if t in self.env:
                src = self.env[t]
                node = N(src.text, src.prec, src.kind)
                node.src_temp = t
                self.stack.append(node)
                return
        self.stack.append(atom(t))

    def run(self, items):
        for it in items:
            if isinstance(it, list):
                self.stack.append(('block', it))
            else:
                self.op(it)

    def block_stmts(self, block):
        """Evaluate a nested block as statements with the shared env."""
        m = Machine(self.symbols, self.kind, self.lenient)
        m.env, m.promote, m.copies = self.env, self.promote, self.copies
        m.run(block)
        if m.stack:
            raise Fail('block leaves stack')
        self.incomplete = self.incomplete or m.incomplete
        return m.stmts

    def block_value(self, block):
        m = Machine(self.symbols, self.kind)
        m.env, m.promote, m.copies = self.env, self.promote, self.copies
        m.run(block)
        if m.stmts or len(m.stack) != 1:
            raise Fail('block is not a single value')
        return m.pop()

    def op(self, t):
        s = self.stack
        if t in ARITH:
            op, prec = ARITH[t]
            b, a = self.pop(), self.pop()
            s.append(N('%s %s %s' % (a.r(prec - 1), op, b.r(prec)), prec, 'num'))
        elif t in CMP:
            b, a = self.pop(), self.pop()
            s.append(N('%s %s %s' % (a.r(3), CMP[t], b.r(3)), 3, 'bool'))
        elif t in ('max', 'fmax', 'min', 'fmin'):
            b, a = self.pop(), self.pop()
            word = 'maximum' if t in ('max', 'fmax') else 'minimum'
            # prec=1: `the maximum of (a) and (b)` must be parenthesized the
            # moment it is embedded in arithmetic, or `and B` swallows the
            # surrounding operators.
            s.append(N('the %s of (%s) and (%s)' % (word, a.r(0), b.r(0)), 1, 'num'))
        elif t == 'strconcat':
            b, a = self.pop(), self.pop()
            s.append(N('%s + %s' % (a.r(4), b.r(4)), 4, 'str'))
        elif t == 'cvs':
            x = self.pop()
            s.append(x if isinstance(x, StrVal) or x.kind == 'str' else StrVal(x))
        elif t in CONVS:
            self.stack.append(Conv(self.pop(), t))
        elif t == 'get':
            pass                                     # field fetch — a no-op here
        elif t == 'not':
            x = self.pop()
            s.append(N('not (%s)' % x.r(0), 6, 'bool'))
        elif t == 'and' or t == 'or':
            prec = 2 if t == 'and' else 1
            b, a = self.pop(), self.pop()
            s.append(N('%s %s %s' % (a.r(prec - 1), t, b.r(prec)), prec, 'bool'))
        elif t == 'isnull':
            x = self.pop()
            s.append(N('%s is null' % x.r(3), 3, 'bool'))
        elif t == 'dup':
            s.append(s[-1])
        elif t == 'swap':
            s[-1], s[-2] = s[-2], s[-1]
        elif t == 'over':
            s.append(s[-2])
        elif t == 'pop':
            self.pop()
        elif t == 'xdef':
            self.xdef()
        elif t == 'addto':
            value, target = self.pop(), self.pop()
            self.stmts.append('add %s to %s;' % (value.r(0), target.text))
        elif t == 'ifelse':
            self.ifelse()
        elif t == 'if':
            self.if_()
        elif re.fullmatch(r'[A-Za-z_/$][A-Za-z_0-9.]*|-?\d+(\.\d+)?|"[^"]*"|\'[^\']*\'', t):
            self.push_term(t)
        else:
            raise Fail('op %r' % t)

    def xdef(self):
        name, value = self.pop(), self.pop()
        if isinstance(value, Conv):
            value = value.inner
        if isinstance(value, StrVal):
            pass                                     # keep explicit conversion
        if isinstance(name, tuple) or isinstance(value, tuple):
            raise Fail('block as xdef operand')

        def valueish(n):
            return (n.kind in ('num', 'str', 'bool') or n.prec > 0
                    or getattr(n, 'src_temp', None) is not None
                    or isinstance(n, (StrVal, Cond)))

        if name.text.startswith('/'):
            # the real operator: <value> /name xdef
            dest = name.text[1:]
        elif valueish(name) and value.kind == 'id':
            # fabricated `name value xdef` — the writer meant name := value
            dest, value = value.text, name
        elif valueish(value) and name.kind == 'id':
            # fabricated `value name xdef` — right order, slash forgotten
            dest = name.text
        elif name.kind == 'id' and value.kind == 'id':
            # two plain field reads: the corpus writes these name-first
            # (`refund_or_owed payments xdef` means refund := payments)
            dest, value = value.text, name
        else:
            raise Fail('xdef name %r' % name.text)
        if self.is_temp(dest) and dest in self.promote:
            dest = self.promote[dest]
        if isinstance(value, Cond):
            self.stmts.append('if %s then set %s = %s; else set %s = %s; endif'
                              % (value.cond.r(0), dest, value.a.r(0), dest, value.b.r(0)))
        elif self.is_temp(dest):
            self.env[dest] = value                   # inline, emit nothing
        else:
            src = getattr(value, 'src_temp', None)
            if src:
                self.copies[src] = dest
            if value.text == dest:
                return                               # self-copy: drop
            self.stmts.append('set %s = %s;' % (dest, value.r(0)))

    def _split_ifelse(self):
        three = self.pop(3)
        blocks = [x for x in three if isinstance(x, tuple)]
        conds = [x for x in three if not isinstance(x, tuple)]
        if len(blocks) != 2 or len(conds) != 1:
            raise Fail('ifelse shape')
        cond = conds[0]
        a, b = blocks[0][1], blocks[1][1]
        return cond, a, b

    def ifelse(self):
        cond, a, b = self._split_ifelse()
        try:
            va, vb = self.block_value(a), self.block_value(b)
            self.stack.append(Cond(cond, va, vb))
            return
        except Fail:
            pass
        sa, sb = self.block_stmts(a), self.block_stmts(b)
        if not sa or not sb:
            if self.lenient:
                self.incomplete = True
                return
            raise Fail('empty ifelse body — stack juggling this machine cannot follow')
        self.stmts.append('if %s then %s else %s endif'
                          % (cond.r(0), ' '.join(sa), ' '.join(sb)))

    def if_(self):
        two = self.pop(2)
        blocks = [x for x in two if isinstance(x, tuple)]
        conds = [x for x in two if not isinstance(x, tuple)]
        if len(blocks) != 1 or len(conds) != 1:
            raise Fail('if shape')
        sa = self.block_stmts(blocks[0][1])
        if not sa:
            if self.lenient:
                self.incomplete = True
                return
            raise Fail('empty if body — stack juggling this machine cannot follow')
        self.stmts.append('if %s then %s endif' % (conds[0].r(0), ' '.join(sa)))


# -------------------------------------------------- condition shape rules ----

NAME = r'[A-Za-z_][A-Za-z_0-9.]*'


def eval_single(items, symbols):
    m = Machine(symbols, 'condition')
    m.run(items)
    if m.stmts or len(m.stack) != 1:
        raise Fail('not a single value')
    return m.pop()


def fold_cond(items, symbols):
    """Fold `X { pop Y } over [not] if` chains into `X or/and Y`.

    The stored pattern for `A or B` is `A { pop B } over not if`; for `and`
    the `not` is absent. Chains associate left. The generic machine cannot
    simulate the conditional stack effect, so the shape is folded here and
    the pieces are evaluated as plain expressions.
    """
    i = 0
    while i < len(items):
        if (isinstance(items[i], list) and items[i][:1] == ['pop']
                and items[i + 1:i + 2] == ['over']
                and (items[i + 2:i + 3] == ['if'] or items[i + 2:i + 4] == ['not', 'if'])):
            is_or = items[i + 2:i + 4] == ['not', 'if']
            left = fold_cond(items[:i], symbols)
            right = fold_cond(items[i][1:], symbols)
            op = 'or' if is_or else 'and'
            combined = N('%s %s %s' % (left.r(1 if is_or else 2), op,
                                       right.r(2 if is_or else 3)),
                         1 if is_or else 2, 'bool')
            rest = items[i + (4 if is_or else 3):]
            if not rest:
                return combined
            return fold_cond([combined] + rest, symbols)
        i += 1
    # no fold points: plain expression (items may contain pre-built nodes)
    m = Machine(symbols, 'condition')
    for it in items:
        if isinstance(it, N):
            m.stack.append(it)
        elif isinstance(it, list):
            m.stack.append(('block', it))
        else:
            m.op(it)
    if m.stmts or len(m.stack) != 1:
        raise Fail('not a single value')
    return m.pop()


def cond_shapes(code):
    c = ' '.join(code.split())
    m = re.fullmatch(r'(%s)\s+(true|false)\s+beq\s+\{\s*pop\s+(.*?)\s*\}\s+over\s+not\s+if' % NAME, c)
    if m:
        inner = cond_shapes(m.group(3)) or simple_cmp(m.group(3))
        if inner:
            return '%s == %s or %s' % (m.group(1), m.group(2), inner)
    m = re.fullmatch(r'(%s)\s+(true|false)\s+beq\s+\{\s*pop\s+(.*?)\s*\}\s+over\s+if' % NAME, c)
    if m:
        inner = cond_shapes(m.group(3)) or simple_cmp(m.group(3))
        if inner:
            return '%s == %s and %s' % (m.group(1), m.group(2), inner)
    return simple_cmp(c)


def simple_cmp(c):
    toks = c.split()
    toks = [t for t in toks if t != 'get']
    if len(toks) == 3 and toks[2] in CMP:
        return '%s %s %s' % (toks[0], CMP[toks[2]], toks[1])
    if len(toks) == 7 and toks[2] in CMP and toks[5] in CMP and toks[6] in ('or', 'and'):
        return '%s %s %s %s %s %s %s' % (toks[0], CMP[toks[2]], toks[1], toks[6],
                                         toks[3], CMP[toks[5]], toks[4])
    if len(toks) == 4 and toks[2] in CMP and toks[3] == 'not':
        return 'not (%s %s %s)' % (toks[0], CMP[toks[2]], toks[1])
    if len(toks) == 3 and toks[1] == 'isnull' and toks[2] == 'not':
        return '%s is not null' % toks[0]
    return None


# ------------------------------------------------------------ row driver ----

def decompile(pf, kind, symbols):
    code = strip_comment_lines(pf)
    if not code.split():
        return None, 'comment-only'
    if kind == 'condition':
        got = cond_shapes(' '.join(lex(pf)))
        if got:
            return got, 'shape'
        try:
            return fold_cond(tree(lex(pf)), symbols).r(0), 'fold'
        except Fail:
            pass
    items = tree(lex(pf))
    # Discovery pass first, tolerating empty conditional bodies: a temp
    # accumulated inside an if-block only translates once its `real = temp`
    # copy is known, and that copy sits after the block that fails strict.
    m = Machine(symbols, kind, lenient=True)
    try:
        m.run(items)
    except Fail as e:
        return None, 'fail: %s' % e
    if m.copies:
        # A `real = <temp>` copy promotes the temp to its destination, so a
        # temp accumulated inside a conditional block becomes conditional
        # writes to the real field. Re-run with the promotion map.
        m2 = Machine(symbols, kind)
        m2.promote = dict(m.copies)
        try:
            m2.run(items)
            m = m2
        except Fail:
            pass
    if m.incomplete:
        return None, 'fail: conditional temp writes with no promotion'
    if kind == 'condition':
        if not m.stmts and len(m.stack) == 1:
            return m.stack[0].r(0), 'machine'
        return None, 'fail: condition shape'
    if m.stack:
        return None, 'fail: %d values left on stack' % len(m.stack)
    if not m.stmts:
        return None, 'fail: no statements'
    return ' '.join(m.stmts), 'machine'


# ----------------------------------------------------------- err repairs ----

ASSIGN_LINE = re.compile(r'^\s*(%s)\s*=\s*(.+?)\s*;?\s*$' % NAME)


def repair_dsl(dsl):
    """Fix the common err-row disease: assignments without `set`."""
    lines = [l for l in strip_comment_lines(dsl).splitlines() if l.strip()]
    if not lines:
        return None
    stmts = []
    for chunk in re.split(r';', ' '.join(lines)):
        chunk = chunk.strip()
        if not chunk:
            continue
        m = ASSIGN_LINE.match(chunk + ';') or ASSIGN_LINE.match(chunk)
        if not m:
            return None
        rhs = m.group(2)
        # The RHS must look like an expression, not prose: "CAT = $150 flat
        # minimum" is a sentence, and its meaning lives in the postfix.
        if not re.fullmatch(r"[A-Za-z_0-9.\s+\-*/()'\"<>=!]+", rhs) or '$' in rhs \
                or re.search(r'[a-zA-Z_][a-zA-Z_0-9.]*\s+[a-zA-Z_]', rhs.replace(' and ', ' ').replace(' or ', ' ')):
            return None
        stmts.append('set %s = %s;' % (m.group(1), rhs))
    return ' '.join(stmts) if stmts else None


# ----------------------------------------------------------------- main -----

def load_symbols(project):
    symbols = {}
    for p in glob.glob(os.path.join(project, 'xml/**/*_edd.xml'), recursive=True):
        if 'TEMPLATE' in p:
            continue
        try:
            root = ET.parse(p).getroot()
        except ET.ParseError:
            continue
        for e in root.iter('entity'):
            for f in e.iter('field'):
                symbols[f.get('name')] = f.get('type') or 'double'
                symbols['%s.%s' % (e.get('name'), f.get('name'))] = f.get('type') or 'double'
    return symbols


def main(project, outpath):
    symbols = load_symbols(project)
    out = {}
    stats = collections.Counter()
    fails = []
    for path in sorted(glob.glob(os.path.join(project, 'xml/states/*_corp_dt.xml'))):
        for dt in ET.parse(path).getroot().iter('decision_table'):
            table = dt.findtext('table_name')
            for kind, label in (('context', 'context'), ('initial_action', 'initial action'),
                                ('condition', 'condition'), ('action', 'action')):
                found = list(dt.iter(kind + '_details'))
                if kind == 'initial_action' and not found:
                    found = list(dt.iter('initial_action'))
                for i, d in enumerate(found):
                    dsl = (d.findtext(kind + '_dsl') or d.findtext('action_dsl') or '').strip()
                    pf = (d.findtext(kind + '_postfix') or d.findtext('action_postfix') or '')
                    key = ('%s %d' % (label, i + 1)) if kind in ('context', 'initial_action') \
                        else ('%s@%d' % (label, i + 1))
                    ckind = 'action' if kind == 'initial_action' else kind
                    el = None
                    if not dsl and pf.strip():                     # hand row
                        el, how = decompile(pf, ckind, symbols)
                        stats['hand:' + (how if el else how.split(':')[0])] += 1
                        if not el and how == 'comment-only':
                            # documentation row: its meaning is the comment
                            first = next((l.strip().lstrip('/# ').strip()
                                          for l in pf.splitlines() if l.strip()), '')
                            el = '// ' + first if first else None
                        elif not el:
                            fails.append((table, key, how, ' '.join(lex(pf))[:110]))
                    elif dsl.startswith('//') and strip_comment_lines(pf).split():
                        # comment DSL over real code: the code is the meaning
                        el, how = decompile(pf, ckind, symbols)
                        if el:
                            stats['comment-dsl:from-postfix'] += 1
                    elif dsl and not dsl.startswith('//'):          # possible err row
                        el = repair_dsl(dsl)
                        if el:
                            stats['err:set-repair'] += 1
                        elif pf.strip():
                            # prose DSL: the row's meaning lives in its
                            # postfix. Decompile that; if the postfix is
                            # comment-only, the row is documentation and the
                            # DSL becomes a comment.
                            el, how = decompile(pf, ckind, symbols)
                            if el:
                                stats['err:from-postfix'] += 1
                            elif how == 'comment-only':
                                el = '// ' + ' '.join(dsl.split())
                                stats['err:to-comment'] += 1
                        else:
                            # prose DSL with no postfix at all: documentation
                            el = '// ' + ' '.join(dsl.split())
                            stats['err:to-comment'] += 1
                    if el:
                        out.setdefault(table, {})[key] = el
    json.dump(out, open(outpath, 'w'), indent=2, sort_keys=True)
    total = sum(len(v) for v in out.values())
    print('%d candidates across %d tables' % (total, len(out)))
    for k, v in stats.most_common():
        print('   %-24s %d' % (k, v))
    if fails:
        print('\nfirst failures:')
        for t, k, how, pf in fails[:10]:
            print('   %s %s  [%s]\n      %s' % (t, k, how, pf))


main(sys.argv[1], sys.argv[2])

/**
 * Unit tests for the trace-tree navigation model — the frame/focus logic
 * every debugger verb is built on. The fixture mirrors real trace shape:
 * a root table whose pass fires a column with two actions; the second
 * action performs a nested table (one recorded pass) and a zero-pass
 * table (its context iterated zero times).
 */
import { describe, expect, it } from 'vitest';
import type { DebugFrame, DebugNode } from '@/api/client';
import {
  bucketSize,
  deriveFrame,
  enclosing,
  firstTable,
  focusTarget,
  frameInfo,
  indexTree,
  nextLanding,
  stackValue,
} from '@/lib/traceTree';

let n = 0;
function node(name: string, attrs?: Record<string, string>, children: DebugNode[] = []): DebugNode {
  return { number: ++n, name, attrs, children } as DebugNode;
}

function fixture() {
  n = 0;
  const trace = node('DTRulesTrace', undefined, []);
  const dt = node('decisiontable', { name: 'Root_Table' });
  const pass = node('execute_table');
  const cond = node('condition', { n: '1', result: 'true' });
  const col = node('column', { n: '2' });
  const a1 = node('action', { n: '1' });
  const a2 = node('action', { n: '2' });
  const calledDT = node('decisiontable', { name: 'Called_Table' });
  const calledPass = node('execute_table');
  const calledCol = node('column', { n: '1' });
  const calledA1 = node('action', { n: '1' });
  const zeroDT = node('decisiontable', { name: 'Zero_Pass_Table' });
  const fin = node('finalState');

  calledCol.children = [calledA1];
  calledPass.children = [calledCol];
  calledDT.children = [calledPass];
  a2.children = [calledDT, zeroDT];
  col.children = [a1, a2];
  pass.children = [cond, col];
  dt.children = [pass];
  trace.children = [dt, fin];
  return { trace, dt, pass, col, a1, a2, calledDT, calledPass, calledA1, zeroDT, fin };
}

describe('indexTree', () => {
  it('records parents, subtree extents, and the end node', () => {
    const f = fixture();
    const idx = indexTree(f.trace);
    expect(idx.parent.get(f.pass.number)).toBe(f.dt.number);
    expect(idx.subtreeMax.get(f.a2.number)).toBe(f.zeroDT.number);
    expect(idx.endNode).toBe(f.fin.number);
    expect(enclosing(idx, f.calledA1.number, 'decisiontable')?.attrs?.name).toBe('Called_Table');
    expect(nextLanding(idx, f.a1.number)).toBe(f.a2.number);
    expect(firstTable(f.trace)?.attrs?.name).toBe('Root_Table');
  });
});

describe('frameInfo', () => {
  it('describes a real pass: fired column, actions, caller', () => {
    const f = fixture();
    const idx = indexTree(f.trace);
    const fi = frameInfo(idx, f.pass)!;
    expect(fi.tableName).toBe('Root_Table');
    expect(fi.firedColumn).toBe('2');
    expect(fi.actions.map((a) => a.number)).toEqual([f.a1.number, f.a2.number]);
    expect(fi.conditionResults.get('1')).toBe('true');
    expect(fi.callerAction).toBeNull();

    const inner = frameInfo(idx, f.calledPass)!;
    expect(inner.tableName).toBe('Called_Table');
    expect(inner.callerAction?.number).toBe(f.a2.number);
    expect(inner.callerPass?.number).toBe(f.pass.number);
  });

  it('synthesizes a pseudo-frame for a zero-pass call', () => {
    const f = fixture();
    const idx = indexTree(f.trace);
    const fi = frameInfo(idx, f.zeroDT)!;
    expect(fi.tableName).toBe('Zero_Pass_Table');
    expect(fi.passes).toHaveLength(0);
    expect(fi.passIndex).toBe(-1);
    expect(fi.callerAction?.number).toBe(f.a2.number);
  });
});

describe('focusTarget', () => {
  it('entry replays to the pass node; action past its subtree; exit past the pass', () => {
    const f = fixture();
    const idx = indexTree(f.trace);
    expect(focusTarget(idx, f.pass, { kind: 'entry' })).toBe(f.pass.number);
    // After action 2 = past its whole subtree (the called tables).
    expect(focusTarget(idx, f.pass, { kind: 'action', node: f.a2.number })).toBe(f.zeroDT.number + 1);
    expect(focusTarget(idx, f.pass, { kind: 'exit' })).toBe(f.zeroDT.number + 1);
  });
});

describe('deriveFrame', () => {
  it('maps arbitrary replay positions back to frame + focus', () => {
    const f = fixture();
    const idx = indexTree(f.trace);
    expect(deriveFrame(idx, f.trace, f.pass.number)).toMatchObject({
      passNode: { number: f.pass.number },
      focus: { kind: 'entry' },
    });
    // Just past action 1's subtree = focus after action 1.
    expect(deriveFrame(idx, f.trace, f.a1.number + 1)).toMatchObject({
      focus: { kind: 'action', node: f.a1.number },
    });
    // A position past the LAST pass has no enclosing frame — derivation
    // falls back to the first table's entry. (Exit focus is preserved by
    // the caller via keepFrame, not by derivation.)
    const past = deriveFrame(idx, f.trace, f.fin.number)!;
    expect(past.passNode.number).toBe(f.pass.number);
    expect(past.focus.kind).toBe('entry');
  });
});

describe('bucketSize', () => {
  it('keeps every level at 20 groups or fewer', () => {
    expect(bucketSize(30)).toBe(25);
    expect(bucketSize(600)).toBe(100);
    expect(bucketSize(18000)).toBe(1000);
  });
});

describe('stackValue', () => {
  const stack: DebugFrame[] = [
    { name: 'budget_params', id: 1, attrs: { supply_limit: '500', weekly_budget: '' } },
    { name: 'account', id: 2, attrs: { balance: '100' } },
    { name: 'account', id: 3, attrs: { balance: '200' } },
  ];

  it('resolves entity.attr against the topmost frame of that entity', () => {
    expect(stackValue(stack, 'account.balance')).toBe('200');
    expect(stackValue(stack, 'budget_params.supply_limit')).toBe('500');
  });

  it('resolves bare names by searching the stack top-down', () => {
    expect(stackValue(stack, 'balance')).toBe('200');
    expect(stackValue(stack, 'supply_limit')).toBe('500');
  });

  it('is case-insensitive (EL) and preserves nothing-found as undefined', () => {
    expect(stackValue(stack, 'Account.BALANCE')).toBe('200');
    expect(stackValue(stack, 'perform')).toBeUndefined();
    expect(stackValue(stack, 'account.nope')).toBeUndefined();
  });

  it('shows empty values explicitly', () => {
    expect(stackValue(stack, 'weekly_budget')).toBe('(empty)');
  });
});

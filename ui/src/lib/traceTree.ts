/**
 * traceTree - navigation structure over a loaded trace tree.
 *
 * Shared by the debugger's table view (primary) and tree view (expert):
 * parent links, subtree extents, landing points for the step verbs, and
 * the frame/focus model the table view navigates with.
 *
 * @module lib/traceTree
 */

import type { DebugNode } from '@/api/client';

export interface TreeIndex {
  byNumber: Map<number, DebugNode>;
  parent: Map<number, number>;
  subtreeMax: Map<number, number>;
  landings: number[];
  /** Terminal position: the finalState node (end of execution). */
  endNode: number;
}

export function indexTree(root: DebugNode): TreeIndex {
  const byNumber = new Map<number, DebugNode>();
  const parent = new Map<number, number>();
  const subtreeMax = new Map<number, number>();
  const landings: number[] = [];
  let endNode = 0;

  const walk = (n: DebugNode): number => {
    byNumber.set(n.number, n);
    if (n.name === 'action' || n.name === 'initialaction') landings.push(n.number);
    if (n.name === 'finalState') endNode = n.number;
    let max = n.number;
    for (const c of n.children) {
      parent.set(c.number, n.number);
      max = Math.max(max, walk(c));
    }
    subtreeMax.set(n.number, max);
    return max;
  };
  const rootMax = walk(root);
  landings.sort((a, b) => a - b);
  if (!endNode) endNode = rootMax;
  return { byNumber, parent, subtreeMax, landings, endNode };
}

export function parentOf(idx: TreeIndex, n: number): DebugNode | null {
  const p = idx.parent.get(n);
  return p === undefined ? null : idx.byNumber.get(p) || null;
}

export function ancestorsOf(idx: TreeIndex, num: number): number[] {
  const out: number[] = [];
  let cur = idx.parent.get(num);
  while (cur !== undefined) {
    out.push(cur);
    cur = idx.parent.get(cur);
  }
  return out;
}

/** Nearest enclosing node (self included) with the given tag name. */
export function enclosing(idx: TreeIndex, num: number, name: string): DebugNode | null {
  let cur: number | undefined = num;
  while (cur !== undefined) {
    const n = idx.byNumber.get(cur);
    if (n?.name === name) return n;
    cur = idx.parent.get(cur);
  }
  return null;
}

/** First landing point strictly after `after`, or null. */
export function nextLanding(idx: TreeIndex, after: number): number | null {
  for (const l of idx.landings) {
    if (l > after) return l;
  }
  return null;
}

/** First decisiontable node — the start of execution. */
export function firstTable(root: DebugNode): DebugNode | null {
  if (root.name === 'decisiontable') return root;
  for (const c of root.children) {
    const r = firstTable(c);
    if (r) return r;
  }
  return null;
}

// ── range bucketing ──────────────────────────────────────────────────

export const GROUP_SIZES = [25, 100, 1000, 10000, 100000];

export function bucketSize(n: number): number {
  for (const size of GROUP_SIZES) {
    if (Math.ceil(n / size) <= 20) return size;
  }
  return GROUP_SIZES[GROUP_SIZES.length - 1];
}

/** Node kinds shown in the (expert) tree view. Raw replay events — defs,
 *  pushes, array wiring — are machinery, not something to read. */
export const STRUCTURAL_NODES = new Set([
  'DTRulesTrace',
  'decisiontable',
  'execute_table',
  'initialaction',
  'column',
  'action',
  'condition',
  'finalState',
]);

// ── table-view frame model ───────────────────────────────────────────
//
// A frame is one pass (execute_table) of one decision table. Focus within
// the frame determines the replay position:
//   entry      → state coming into the pass (replay to the pass node)
//   action N   → state after action N executed (replay past its subtree)
//   exit       → state leaving the pass (replay past the pass subtree)

export type Focus = { kind: 'entry' } | { kind: 'action'; node: number } | { kind: 'exit' };

export interface FrameInfo {
  passNode: DebugNode;
  dtNode: DebugNode;
  tableName: string;
  passes: DebugNode[];
  passIndex: number; // 0-based
  /** Executed initial actions of this pass, in order. */
  initialActions: DebugNode[];
  /** Condition results of this pass: condition number -> "true"/"false". */
  conditionResults: Map<string, string>;
  /** The fired column node (if any) and its executed actions in order. */
  columnNode: DebugNode | null;
  firedColumn: string;
  actions: DebugNode[];
  /** The action in the CALLER that performed this table, if any. */
  callerAction: DebugNode | null;
  /** The caller's pass, if any. */
  callerPass: DebugNode | null;
}

export function frameInfo(idx: TreeIndex, passNode: DebugNode): FrameInfo | null {
  // A decisiontable node itself acts as a pseudo-pass for calls whose
  // context iterated zero times: no execute_table was recorded, but the
  // user can still land on the table and see the state going into it.
  if (passNode.name === 'decisiontable') {
    const maybeAction = parentOf(idx, passNode.number);
    const callerAction =
      maybeAction && (maybeAction.name === 'action' || maybeAction.name === 'initialaction') ? maybeAction : null;
    return {
      passNode,
      dtNode: passNode,
      tableName: passNode.attrs?.name || 'decisiontable',
      passes: [],
      passIndex: -1,
      initialActions: [],
      conditionResults: new Map(),
      columnNode: null,
      firedColumn: '',
      actions: [],
      callerAction,
      callerPass: callerAction ? enclosing(idx, callerAction.number, 'execute_table') : null,
    };
  }

  const dtNode = parentOf(idx, passNode.number);
  if (!dtNode || dtNode.name !== 'decisiontable') return null;

  const passes = dtNode.children.filter((c) => c.name === 'execute_table');
  const passIndex = passes.findIndex((p) => p.number === passNode.number);

  const initialActions = passNode.children.filter((c) => c.name === 'initialaction');
  const conditionResults = new Map<string, string>();
  for (const c of passNode.children) {
    if (c.name === 'condition' && c.attrs?.n) conditionResults.set(c.attrs.n, c.attrs.result || '');
  }
  const columnNode = passNode.children.find((c) => c.name === 'column') || null;
  const actions = columnNode ? columnNode.children.filter((c) => c.name === 'action') : [];

  // Caller: decisiontable's parent is the performing action (when nested).
  const maybeAction = parentOf(idx, dtNode.number);
  const callerAction = maybeAction && (maybeAction.name === 'action' || maybeAction.name === 'initialaction') ? maybeAction : null;
  const callerPass = callerAction ? enclosing(idx, callerAction.number, 'execute_table') : null;

  return {
    passNode,
    dtNode,
    tableName: dtNode.attrs?.name || 'decisiontable',
    passes,
    passIndex,
    initialActions,
    conditionResults,
    columnNode,
    firedColumn: columnNode?.attrs?.n?.trim() || '',
    actions,
    callerAction,
    callerPass,
  };
}

/** The replay target node for a focus within a pass. */
export function focusTarget(idx: TreeIndex, passNode: DebugNode, focus: Focus): number {
  if (focus.kind === 'entry') return passNode.number;
  if (focus.kind === 'action') {
    const max = idx.subtreeMax.get(focus.node) ?? focus.node;
    return Math.min(max + 1, idx.endNode);
  }
  const max = idx.subtreeMax.get(passNode.number) ?? passNode.number;
  return Math.min(max + 1, idx.endNode);
}

/** Derive the frame + focus for an arbitrary replay position (tree clicks,
 *  breakpoints, go-to-node). */
export function deriveFrame(
  idx: TreeIndex,
  root: DebugNode,
  position: number
): { passNode: DebugNode; focus: Focus } | null {
  let et = enclosing(idx, position, 'execute_table');
  if (!et) {
    // Before any pass (e.g. positioned on a decisiontable node) — use the
    // first pass of the nearest table, or the first table in the trace.
    const dt = enclosing(idx, position, 'decisiontable') || firstTable(root);
    et = dt?.children.find((c) => c.name === 'execute_table') || null;
    if (!et) return null;
    return { passNode: et, focus: { kind: 'entry' } };
  }
  if (position === et.number) return { passNode: et, focus: { kind: 'entry' } };

  // After some action? position sits just past an action's subtree.
  const all = [
    ...et.children.filter((c) => c.name === 'initialaction'),
    ...(et.children.find((c) => c.name === 'column')?.children.filter((c) => c.name === 'action') || []),
  ];
  for (const a of all) {
    if ((idx.subtreeMax.get(a.number) ?? a.number) + 1 === position) {
      return { passNode: et, focus: { kind: 'action', node: a.number } };
    }
  }
  if (position > (idx.subtreeMax.get(et.number) ?? et.number)) {
    return { passNode: et, focus: { kind: 'exit' } };
  }
  return { passNode: et, focus: { kind: 'entry' } };
}

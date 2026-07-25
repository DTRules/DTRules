/**
 * DebugPanel - Trace debugger (v1).
 *
 * Loads a DTRules trace file into the server's debug session, then provides:
 * - a trust strip (DTRules version, rules fingerprint match, end-state
 *   verification result),
 * - the trace tree (click a node = "run to here"),
 * - step forward/back over trace nodes,
 * - the replayed entity stack at the current position,
 * - a read-only postfix console evaluated at the current position.
 *
 * The layout follows the trace-debugger design mock. Later increments add
 * the grid program counter, marks, and watch/breakpoint machinery.
 *
 * @module components/DebugPanel
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import {
  debugConsole,
  debugLoad,
  debugPosition,
  debugTree,
  type DebugFrame,
  type DebugLoadResponse,
  type DebugNode,
} from '@/api/client';
import { FileBrowser } from '@/components/FileBrowser';
import { DebugTableView } from '@/components/DebugTableView';
import {
  ancestorsOf,
  bucketSize,
  deriveFrame,
  enclosing,
  firstTable,
  focusTarget,
  frameInfo,
  indexTree,
  nextLanding,
  STRUCTURAL_NODES,
  type Focus,
  type TreeIndex,
} from '@/lib/traceTree';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { ArrowLeft, Bug, ExternalLink } from 'lucide-react';

/** Human label for a trace node. */
function nodeLabel(n: DebugNode, ordinal?: number): string {
  switch (n.name) {
    case 'decisiontable':
      return n.attrs?.name || 'decisiontable';
    case 'execute_table':
      return ordinal !== undefined ? `pass ${ordinal.toLocaleString()}` : 'pass';
    case 'column':
      return `column ${n.attrs?.n ?? ''}`;
    case 'action':
      return `action ${n.attrs?.n ?? ''}`;
    case 'initialaction':
      return `initial action ${n.attrs?.n ?? ''}`;
    case 'condition':
      return `condition ${n.attrs?.n} = ${n.attrs?.result}`;
    case 'def':
      return `def ${n.attrs?.entity}.${n.attrs?.name}`;
    case 'entitypush':
      return `push ${n.attrs?.entity}`;
    case 'entitypop':
      return 'pop';
    case 'finalState':
      return 'final state';
    default:
      return n.name;
  }
}

/**
 * TreeIndex precomputes navigation structure over the trace tree: parent
 * links, subtree extents, and the "landing points" the step verbs move
 * between (action / initialaction nodes, in document order).
 */
interface TreeCommon {
  position: number;
  onSelect: (n: number) => void;
  expanded: Set<number>;
  onToggle: (n: number) => void;
  subtreeMax: Map<number, number>;
  breakpoints: Set<number>;
  onToggleBreakpoint: (n: number) => void;
}

function NodeChildren({
  nodes: rawNodes,
  ordinalStart,
  depth,
  common,
}: {
  nodes: DebugNode[];
  ordinalStart: number;
  depth: number;
  common: TreeCommon;
}) {
  // The tree is an expert view of the EXECUTION — structural nodes only.
  // Raw replay events (defs, pushes, array wiring) are machinery.
  const nodes = rawNodes.filter((n) => STRUCTURAL_NODES.has(n.name));
  if (nodes.length <= 25) {
    return (
      <>
        {nodes.map((c, i) => (
          <TreeNodeView key={c.number} node={c} depth={depth} common={common} ordinal={ordinalStart + i} />
        ))}
      </>
    );
  }
  const size = bucketSize(nodes.length);
  const groups: number[] = [];
  for (let i = 0; i < nodes.length; i += size) groups.push(i);
  return (
    <>
      {groups.map((i) => (
        <RangeGroup
          key={i}
          nodes={nodes.slice(i, i + size)}
          ordinalStart={ordinalStart + i}
          depth={depth}
          common={common}
        />
      ))}
    </>
  );
}

function RangeGroup({
  nodes,
  ordinalStart,
  depth,
  common,
}: {
  nodes: DebugNode[];
  ordinalStart: number;
  depth: number;
  common: TreeCommon;
}) {
  const [open, setOpen] = useState(false);
  const first = nodes[0];
  const last = nodes[nodes.length - 1];
  const rangeEnd = common.subtreeMax.get(last.number) ?? last.number;
  const containsPC = common.position >= first.number && common.position <= rangeEnd;
  const isOpen = open || containsPC;
  let containsBP = false;
  for (const bp of common.breakpoints) {
    if (bp >= first.number && bp <= rangeEnd) {
      containsBP = true;
      break;
    }
  }

  return (
    <div>
      <div
        className={cn(
          'flex items-center gap-1 px-1 rounded-sm cursor-pointer hover:bg-accent font-mono text-xs leading-6 whitespace-nowrap text-blue-300/80',
          containsPC && 'text-amber-300/90'
        )}
        style={{ paddingLeft: depth * 12 + 4 }}
        onClick={() => setOpen(!open)}
        title={containsPC ? 'Contains the current position' : undefined}
      >
        <span className="w-3 shrink-0">{isOpen ? '▾' : '▸'}</span>
        <span>
          [{ordinalStart.toLocaleString()} … {(ordinalStart + nodes.length - 1).toLocaleString()}]
        </span>
        <span className="text-muted-foreground">· {nodes.length.toLocaleString()} items</span>
        {containsBP && <span className="text-red-500">●</span>}
      </div>
      {isOpen && (
        <NodeChildren nodes={nodes} ordinalStart={ordinalStart} depth={depth + 1} common={common} />
      )}
    </div>
  );
}

function TreeNodeView({
  node,
  depth,
  common,
  ordinal,
}: {
  node: DebugNode;
  depth: number;
  common: TreeCommon;
  ordinal?: number;
}) {
  const { position, onSelect, expanded: expandedSet, onToggle, breakpoints, onToggleBreakpoint } = common;
  const expanded = expandedSet.has(node.number);
  const hasChildren = node.children.length > 0;
  const structural = ['decisiontable', 'execute_table', 'column', 'action', 'initialaction', 'DTRulesTrace', 'finalState'].includes(node.name);

  return (
    <div>
      <div
        className={cn(
          'flex items-center gap-1 px-1 rounded-sm cursor-pointer hover:bg-accent font-mono text-xs leading-6 whitespace-nowrap',
          node.number === position && 'bg-amber-500/15 outline outline-1 outline-amber-500/40',
          !structural && 'text-muted-foreground'
        )}
        style={{ paddingLeft: depth * 12 + 4 }}
        onClick={() => onSelect(node.number)}
        onContextMenu={(e) => {
          e.preventDefault();
          onToggleBreakpoint(node.number);
        }}
        title={`Run to node ${node.number} — right-click to toggle breakpoint`}
        data-node={node.number}
      >
        <span
          className="w-3 text-muted-foreground shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            onToggle(node.number);
          }}
        >
          {hasChildren ? (expanded ? '▾' : '▸') : ''}
        </span>
        {breakpoints.has(node.number) && <span className="text-red-500">● </span>}
        {node.number === position && <span className="text-amber-400">▶ </span>}
        <span className={cn(structural && node.name === 'decisiontable' && 'font-semibold text-foreground')}>
          {nodeLabel(node, node.name === 'execute_table' ? ordinal : undefined)}
        </span>
      </div>
      {expanded && hasChildren && (
        <NodeChildren nodes={node.children} ordinalStart={1} depth={depth + 1} common={common} />
      )}
    </div>
  );
}

export function DebugPanel() {
  const { selectTable, setActiveTab, decisionTables } = useProjectStore();

  // Project tables by lowercased name (EL is case-insensitive) — lets the
  // table view offer static-definition links for tables the trace never ran.
  const knownTables = useMemo(
    () => new Map(decisionTables.map((t) => [t.name.toLowerCase(), t.name])),
    [decisionTables]
  );

  const [info, setInfo] = useState<DebugLoadResponse | null>(null);
  const [tree, setTree] = useState<DebugNode | null>(null);
  const [position, setPosition] = useState(1);
  const [nodeCount, setNodeCount] = useState(0);
  const [context, setContext] = useState<{ table?: string; column?: string; action?: string }>({});
  const [stack, setStack] = useState<DebugFrame[]>([]);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [browsing, setBrowsing] = useState(true);

  const [consoleLines, setConsoleLines] = useState<{ input: string; output: string; error?: boolean }[]>([]);
  const consoleInput = useRef<HTMLInputElement>(null);

  const [viewMode, setViewMode] = useState<'table' | 'tree'>('table');
  const [frame, setFrame] = useState<{ pass: number; focus: Focus } | null>(null);
  const treeRef = useRef<DebugNode | null>(null);

  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [breakpoints, setBreakpoints] = useState<Set<number>>(new Set());
  const [marks, setMarks] = useState<{ node: number; label: string; auto: boolean }[]>([]);
  const idxRef = useRef<TreeIndex | null>(null);
  const lastTableRef = useRef<number | null>(null);

  const toggleBreakpoint = useCallback((n: number) => {
    setBreakpoints((prev) => {
      const next = new Set(prev);
      if (next.has(n)) next.delete(n);
      else next.add(n);
      return next;
    });
  }, []);

  const toggleNode = useCallback((n: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(n)) next.delete(n);
      else next.add(n);
      return next;
    });
  }, []);

  const goTo = useCallback(async (node: number, opts?: { keepFrame?: boolean }) => {
    const r = await debugPosition(node);
    if (!r.success) return;
    const pos = r.position || node;
    setPosition(pos);
    setContext(r.context || {});
    setStack(r.stack || []);

    // Keep the table view in sync with wherever the position lands
    // (tree clicks, breakpoints, go-to-node) — unless the table view
    // itself drove the move and set its own frame.
    if (!opts?.keepFrame && idxRef.current && treeRef.current) {
      const derived = deriveFrame(idxRef.current, treeRef.current, pos);
      if (derived) setFrame({ pass: derived.passNode.number, focus: derived.focus });
    }

    const idx = idxRef.current;
    if (idx) {
      // Expand the tree along the path to the program counter and scroll it
      // into view.
      setExpanded((prev) => {
        const next = new Set(prev);
        for (const a of ancestorsOf(idx, pos)) next.add(a);
        next.add(pos);
        return next;
      });
      setTimeout(() => {
        document.querySelector(`[data-node="${pos}"]`)?.scrollIntoView({ block: 'center' });
      }, 50);

      // Auto-mark: entering a different decision table drops a mark at its
      // start, so "back to mark" returns to the top of the table.
      const dt = enclosing(idx, pos, 'decisiontable');
      if (dt && dt.number !== lastTableRef.current) {
        lastTableRef.current = dt.number;
        setMarks((m) =>
          m.some((x) => x.node === dt.number)
            ? m
            : [...m.slice(-19), { node: dt.number, label: dt.attrs?.name || 'table', auto: true }]
        );
      }
    }
  }, []);

  // Run: continue to the next breakpoint after the current position
  // (or the end of execution when none remain).
  const runToBreakpoint = useCallback(() => {
    const idx = idxRef.current;
    if (!idx) return;
    let next: number | null = null;
    for (const bp of breakpoints) {
      if (bp > position && (next === null || bp < next)) next = bp;
    }
    goTo(next ?? idx.endNode);
  }, [breakpoints, position, goTo]);

  // ── step verbs (computed from tree structure) ─────────────────────
  const stepOver = useCallback(() => {
    const idx = idxRef.current;
    if (!idx) return;
    const cur = idx.byNumber.get(position);
    const from =
      cur && (cur.name === 'action' || cur.name === 'initialaction')
        ? idx.subtreeMax.get(cur.number) || position
        : position;
    const next = nextLanding(idx, from);
    if (next && next < idx.endNode) goTo(next);
    else if (position < idx.endNode) goTo(idx.endNode);
  }, [position, nodeCount, goTo]);

  const stepInto = useCallback(() => {
    const idx = idxRef.current;
    if (!idx) return;
    const cur = idx.byNumber.get(position);
    if (cur && (cur.name === 'action' || cur.name === 'initialaction')) {
      const inner = nextLanding(idx, cur.number);
      const max = idx.subtreeMax.get(cur.number) || cur.number;
      if (inner && inner <= max) {
        goTo(inner);
        return;
      }
    }
    stepOver();
  }, [position, goTo, stepOver]);

  const stepPass = useCallback(() => {
    const idx = idxRef.current;
    if (!idx) return;
    const et = enclosing(idx, position, 'execute_table');
    if (!et) {
      stepOver();
      return;
    }
    // Finish this pass; on an iterating context the next execute_table of
    // the same table is the top of the next pass.
    const dt = enclosing(idx, et.number, 'decisiontable');
    const sibling = dt?.children.find((c) => c.name === 'execute_table' && c.number > et.number);
    if (sibling) {
      goTo(sibling.number);
      return;
    }
    const after = nextLanding(idx, idx.subtreeMax.get(dt?.number || et.number) || position);
    goTo(after && after < idx.endNode ? after : idx.endNode);
  }, [position, nodeCount, goTo, stepOver]);

  const stepUp = useCallback(() => {
    const idx = idxRef.current;
    if (!idx) return;
    const dt = enclosing(idx, position, 'decisiontable');
    if (!dt) {
      stepOver();
      return;
    }
    const after = nextLanding(idx, idx.subtreeMax.get(dt.number) || position);
    goTo(after && after < idx.endNode ? after : idx.endNode);
  }, [position, nodeCount, goTo, stepOver]);

  // ── table-view navigation ─────────────────────────────────────────
  const applyFocus = useCallback(
    (passNode: DebugNode, focus: Focus) => {
      const idx = idxRef.current;
      if (!idx) return;
      setFrame({ pass: passNode.number, focus });
      goTo(focusTarget(idx, passNode, focus), { keepFrame: true });
    },
    [goTo]
  );

  const framePassNode = frame ? idxRef.current?.byNumber.get(frame.pass) || null : null;

  const tableDrill = useCallback(
    (calledDT: DebugNode) => {
      const pass = calledDT.children.find((c) => c.name === 'execute_table');
      if (pass) applyFocus(pass, { kind: 'entry' });
    },
    [applyFocus]
  );

  const tableOut = useCallback(() => {
    const idx = idxRef.current;
    if (!idx || !framePassNode) return;
    const fi = frameInfo(idx, framePassNode);
    if (!fi?.callerPass || !fi.callerAction) return;
    const caller = frameInfo(idx, fi.callerPass);
    if (!caller) return;
    const ordered = [...caller.initialActions, ...caller.actions];
    const i = ordered.findIndex((a) => a.number === fi.callerAction!.number);
    const next = i >= 0 ? ordered[i + 1] : undefined;
    // Land on the following action in the caller — or its exit when the
    // perform was the last action.
    applyFocus(fi.callerPass, next ? { kind: 'action', node: next.number } : { kind: 'exit' });
  }, [framePassNode, applyFocus]);

  const tableStep = useCallback(() => {
    const idx = idxRef.current;
    if (!idx || !frame || !framePassNode) return;
    const fi = frameInfo(idx, framePassNode);
    if (!fi) return;
    const ordered = [...fi.initialActions, ...fi.actions];
    if (frame.focus.kind === 'entry') {
      if (ordered.length > 0) applyFocus(framePassNode, { kind: 'action', node: ordered[0].number });
      else applyFocus(framePassNode, { kind: 'exit' });
      return;
    }
    if (frame.focus.kind === 'action') {
      const i = ordered.findIndex((a) => a.number === (frame.focus as { node: number }).node);
      const next = ordered[i + 1];
      applyFocus(framePassNode, next ? { kind: 'action', node: next.number } : { kind: 'exit' });
      return;
    }
    // exit: continue to the next iteration, or out to the caller.
    if (fi.passIndex < fi.passes.length - 1) {
      applyFocus(fi.passes[fi.passIndex + 1], { kind: 'entry' });
    } else if (fi.callerPass) {
      tableOut();
    }
  }, [frame, framePassNode, applyFocus, tableOut]);

  const tableInto = useCallback(() => {
    const idx = idxRef.current;
    if (!idx || !frame) return;
    if (frame.focus.kind === 'action') {
      const a = idx.byNumber.get(frame.focus.node);
      const dt = a?.children.find((c) => c.name === 'decisiontable');
      if (dt) {
        tableDrill(dt);
        return;
      }
    }
    tableStep();
  }, [frame, tableDrill, tableStep]);

  const tablePass = useCallback(() => {
    const idx = idxRef.current;
    if (!idx || !framePassNode) return;
    const fi = frameInfo(idx, framePassNode);
    if (!fi) return;
    if (fi.passIndex < fi.passes.length - 1) applyFocus(fi.passes[fi.passIndex + 1], { kind: 'entry' });
    else applyFocus(framePassNode, { kind: 'exit' });
  }, [framePassNode, applyFocus]);

  const dropMark = useCallback(() => {
    setMarks((m) => [...m.slice(-19), { node: position, label: `node ${position}`, auto: false }]);
  }, [position]);

  const backToMark = useCallback(() => {
    setMarks((m) => {
      // Return to the most recent mark before the current position.
      const usable = m.filter((x) => x.node < position);
      const target = usable[usable.length - 1];
      if (target) {
        goTo(target.node);
        return m.filter((x) => x !== target || x.auto);
      }
      goTo(1);
      return m;
    });
  }, [position, goTo]);

  const handleLoad = async (path: string) => {
    setLoadError(null);
    const r = await debugLoad(path);
    if (!r.success) {
      setLoadError(r.error || 'Failed to load trace');
      return;
    }
    setInfo(r);
    setNodeCount(r.nodes || 0);
    setBrowsing(false);
    setMarks([]);
    setBreakpoints(new Set());
    lastTableRef.current = null;
    const t = await debugTree();
    if (t.success && t.tree) {
      setTree(t.tree);
      treeRef.current = t.tree;
      idxRef.current = indexTree(t.tree);
      setExpanded(new Set([t.tree.number, ...t.tree.children.map((c) => c.number)]));
      // Establish the initial entity stack: position at the start of
      // execution, with all initial data replayed.
      const start = firstTable(t.tree);
      await goTo(start ? start.number : 1);
      return;
    }
    await goTo(1);
  };

  const runConsole = async () => {
    const input = consoleInput.current?.value.trim();
    if (!input) return;
    consoleInput.current!.value = '';
    const r = await debugConsole(input);
    if (r.success) {
      setConsoleLines((l) => [...l, { input, output: (r.results || []).join(' · ') || '(empty stack)' }]);
    } else {
      setConsoleLines((l) => [...l, { input, output: r.error || 'error', error: true }]);
    }
  };

  // ── no trace loaded: picker ─────────────────────────────────────────
  if (!info || browsing) {
    return (
      <div className="h-full overflow-auto">
        <div className="max-w-2xl mx-auto p-8">
          <div className="flex items-center gap-3 mb-2">
            <Bug className="h-6 w-6 text-amber-400" />
            <h1 className="text-xl font-bold">Trace Debugger</h1>
          </div>
          <p className="text-sm text-muted-foreground mb-4">
            Load a trace produced by <span className="font-mono">dtrules run --trace</span> (or the
            test harness) to step through the recorded execution. The project whose rules produced
            the trace should be open.
          </p>
          {loadError && (
            <div className="mb-3 p-3 rounded bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
              {loadError}
            </div>
          )}
          <FileBrowser onSelect={handleLoad} selectFiles />
        </div>
      </div>
    );
  }

  // ── debugger ────────────────────────────────────────────────────────
  const verifyClean = (info.verifyMismatches || []).length === 0;
  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Trust strip */}
      <div className="px-4 py-1.5 border-b border-amber-500/25 bg-amber-500/[0.07] flex items-center gap-3 text-xs flex-wrap">
        <span className="font-mono">{info.tracePath?.split('/').pop()}</span>
        <span className="px-2 py-0.5 rounded-full border border-border text-muted-foreground">
          DTRules {info.dtrulesVersion || 'unknown'}
        </span>
        <span
          className={cn(
            'px-2 py-0.5 rounded-full border',
            info.fingerprintMatch === 'match' && 'border-green-500/40 text-green-500',
            info.fingerprintMatch === 'mismatch' && 'border-amber-500/50 text-amber-400',
            info.fingerprintMatch === 'unknown' && 'border-border text-muted-foreground'
          )}
        >
          rules {info.fingerprintMatch === 'match' ? 'match' : info.fingerprintMatch === 'mismatch' ? 'DIFFER from workspace' : 'fingerprint unknown'}
        </span>
        <span
          className={cn(
            'px-2 py-0.5 rounded-full border',
            verifyClean ? 'border-green-500/40 text-green-500' : 'border-red-500/50 text-red-400'
          )}
          title={verifyClean ? 'Replayed end state matches the recorded final state' : (info.verifyMismatches || []).slice(0, 5).join('\n')}
        >
          {verifyClean ? 'end-state verified' : `${info.verifyMismatches!.length} end-state mismatches`}
        </span>
        <Button variant="ghost" size="sm" className="h-6 px-2 ml-auto text-xs" onClick={() => setBrowsing(true)}>
          Load another trace
        </Button>
      </div>

      {/* Debug toolbar */}
      <div className="px-4 py-2 border-b border-border/50 bg-muted/20 flex items-center gap-1.5 flex-wrap">
        <Button variant="outline" size="sm" className="h-7 text-green-500 border-green-500/40" onClick={runToBreakpoint} title="Continue to the next breakpoint (right-click a tree node to set one)">
          ▶ Run{breakpoints.size > 0 && <span className="ml-1 text-[10px] text-muted-foreground">({breakpoints.size})</span>}
        </Button>
        <span className="w-px h-5 bg-border mx-1" />
        <Button variant="outline" size="sm" className="h-7 text-amber-400 border-amber-400/40" onClick={() => (viewMode === 'table' ? tableStep() : stepOver())} title="Execute one action">
          Step
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={() => (viewMode === 'table' ? tableInto() : stepInto())} title="Descend into the table this action performs">
          Into
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={() => (viewMode === 'table' ? tablePass() : stepPass())} title="Finish this pass of the table (lands at the next pass on iterating contexts)">
          Pass
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={() => (viewMode === 'table' ? tableOut() : stepUp())} title="Finish this table and return to the caller">
          Up
        </Button>
        <span className="w-px h-5 bg-border mx-1" />
        <Button variant="outline" size="sm" className="h-7 text-purple-400 border-purple-400/40" onClick={dropMark} title="Drop a mark at the current position">
          ⚑ Mark
        </Button>
        <Button variant="outline" size="sm" className="h-7 text-purple-400" onClick={backToMark} title="Step back to the most recent mark">
          <ArrowLeft className="h-3.5 w-3.5 mr-1" /> To mark
          {marks.length > 0 && <span className="ml-1 text-[10px] text-muted-foreground">({marks.length})</span>}
        </Button>
        {viewMode === 'tree' && (
          <span className="text-xs text-muted-foreground ml-2 hidden xl:inline">Click a node to run to it · right-click to set a breakpoint</span>
        )}
        <span className="ml-auto flex items-center gap-0.5 rounded-md border border-border p-0.5 text-xs">
          <button
            className={cn('px-2 py-0.5 rounded', viewMode === 'table' ? 'bg-accent text-foreground' : 'text-muted-foreground')}
            onClick={() => setViewMode('table')}
          >
            Table
          </button>
          <button
            className={cn('px-2 py-0.5 rounded', viewMode === 'tree' ? 'bg-accent text-foreground' : 'text-muted-foreground')}
            onClick={() => setViewMode('tree')}
            title="Expert view: the raw execution tree"
          >
            Tree
          </button>
        </span>
        <span className="font-mono text-xs text-muted-foreground flex items-center gap-1.5">
          node
          <input
            key={position}
            defaultValue={position}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const n = parseInt((e.target as HTMLInputElement).value, 10);
                if (!isNaN(n)) goTo(Math.min(Math.max(1, n), nodeCount));
              }
            }}
            className="w-16 h-6 px-1.5 rounded border border-input bg-transparent text-foreground text-right"
            title="Type a node number and press Enter to run to it"
          />
          / {nodeCount.toLocaleString()}
        </span>
      </div>

      {/* Context breadcrumb (tree mode; the table view carries its own) */}
      {viewMode === 'tree' && !context.table && position === idxRef.current?.endNode && (
        <div className="px-4 py-1.5 border-b border-border/50 text-sm text-muted-foreground">
          End of execution — final state{' '}
          <span className="text-green-500">(verified against replay on load)</span>
        </div>
      )}
      {viewMode === 'tree' && context.table && (
        <div className="px-4 py-1.5 border-b border-border/50 flex items-center gap-2 text-sm">
          <span className="font-semibold">{context.table}</span>
          {context.column && <span className="text-muted-foreground">· column {context.column}</span>}
          {context.action && <span className="text-muted-foreground">· action {context.action}</span>}
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs text-blue-400"
            onClick={() => {
              selectTable(context.table!);
              setActiveTab('dt');
            }}
          >
            <ExternalLink className="h-3 w-3 mr-1" /> Open in editor
          </Button>
        </div>
      )}

      {/* Tree + stack */}
      <div className="flex-1 grid grid-cols-[1fr_320px] overflow-hidden">
        <ScrollArea className="border-r border-border/50">
          {viewMode === 'table' ? (
            idxRef.current && framePassNode ? (
              <DebugTableView
                idx={idxRef.current}
                passNode={framePassNode}
                focus={frame!.focus}
                knownTables={knownTables}
                onFocus={(f) => applyFocus(framePassNode, f)}
                onDrill={tableDrill}
                onOut={tableOut}
                onPass={(p) => applyFocus(p, { kind: 'entry' })}
                onOpenTable={(name) => {
                  selectTable(name);
                  setActiveTab('dt');
                }}
              />
            ) : (
              <div className="p-6 text-sm text-muted-foreground">No table pass at this position.</div>
            )
          ) : (
            <div className="p-2">
              {tree && idxRef.current && (
                <TreeNodeView
                  node={tree}
                  depth={0}
                  common={{ position, onSelect: goTo, expanded, onToggle: toggleNode, subtreeMax: idxRef.current.subtreeMax, breakpoints, onToggleBreakpoint: toggleBreakpoint }}
                />
              )}
            </div>
          )}
        </ScrollArea>
        <ScrollArea>
          <div className="p-3 space-y-2">
            <div className="text-[11px] font-semibold tracking-widest uppercase text-muted-foreground">
              Entity stack
            </div>
            {[...stack].reverse().map((f, i) =>
              f.name === 'primitives' || f.name === 'decisiontables' ? (
                <div key={`${f.id}-${i}`} className="px-2.5 py-1 text-xs text-muted-foreground border border-border/30 rounded-md">
                  {f.name} <span className="text-[10px]">(system · {Object.keys(f.attrs).length} entries hidden)</span>
                </div>
              ) : (
              <div key={`${f.id}-${i}`} className="border border-border/40 rounded-md overflow-hidden">
                <div className="px-2.5 py-1 bg-muted/30 flex items-baseline gap-2">
                  <span className="text-sm font-semibold">{f.name}</span>
                  <span className="text-[10px] font-mono text-muted-foreground">#{f.id}</span>
                </div>
                <div className="px-2.5 py-1 font-mono text-[11px] leading-relaxed">
                  {Object.entries(f.attrs)
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([k, v]) => (
                      <div key={k} className="flex gap-2">
                        <span className="text-muted-foreground min-w-32 truncate">{k}</span>
                        <span className="truncate" title={v}>{v}</span>
                      </div>
                    ))}
                </div>
              </div>
              )
            )}
          </div>
        </ScrollArea>
      </div>

      {/* Console */}
      <div className="border-t border-border/50 h-44 flex flex-col">
        <div className="px-4 pt-2 text-[11px] font-semibold tracking-widest uppercase text-muted-foreground">
          Postfix console — leftover data stack is printed
        </div>
        <ScrollArea className="flex-1 px-4">
          <div className="py-1 font-mono text-xs leading-relaxed">
            {consoleLines.map((l, i) => (
              <div key={i}>
                <div>
                  <span className="text-purple-400 font-bold">▸ </span>
                  {l.input}
                </div>
                <div className={cn('pl-4', l.error ? 'text-amber-400' : 'text-muted-foreground')}>
                  {l.output}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
        <div className="px-4 pb-3">
          <input
            ref={consoleInput}
            onKeyDown={(e) => e.key === 'Enter' && runConsole()}
            placeholder="postfix tokens — evaluated at the current position, read-only"
            className="w-full h-8 px-3 rounded-md border border-input bg-transparent font-mono text-xs outline-none focus:border-purple-400/50"
          />
        </div>
      </div>
    </div>
  );
}

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

import { useCallback, useRef, useState } from 'react';
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
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { ArrowLeft, Bug, ExternalLink } from 'lucide-react';

/** Human label for a trace node. */
function nodeLabel(n: DebugNode): string {
  switch (n.name) {
    case 'decisiontable':
      return n.attrs?.name || 'decisiontable';
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
interface TreeIndex {
  byNumber: Map<number, DebugNode>;
  parent: Map<number, number>;
  subtreeMax: Map<number, number>;
  landings: number[];
  /** Terminal position: the finalState node (end of execution). */
  endNode: number;
}

function indexTree(root: DebugNode): TreeIndex {
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

/** Numbers of every ancestor of a node (for tree expansion). */
function ancestorsOf(idx: TreeIndex, num: number): number[] {
  const out: number[] = [];
  let cur = idx.parent.get(num);
  while (cur !== undefined) {
    out.push(cur);
    cur = idx.parent.get(cur);
  }
  return out;
}

/** Nearest enclosing node (self included) with the given tag name. */
function enclosing(idx: TreeIndex, num: number, name: string): DebugNode | null {
  let cur: number | undefined = num;
  while (cur !== undefined) {
    const n = idx.byNumber.get(cur);
    if (n?.name === name) return n;
    cur = idx.parent.get(cur);
  }
  return null;
}

/** First landing point strictly after `after`, or null. */
function nextLanding(idx: TreeIndex, after: number): number | null {
  for (const l of idx.landings) {
    if (l > after) return l;
  }
  return null;
}

/** First decisiontable node — the start of execution, where the initial
 *  data has fully loaded and the entity stack is established. */
function firstTable(root: DebugNode): DebugNode | null {
  if (root.name === 'decisiontable') return root;
  for (const c of root.children) {
    const r = firstTable(c);
    if (r) return r;
  }
  return null;
}

function TreeNodeView({
  node,
  position,
  onSelect,
  depth,
  expanded: expandedSet,
  onToggle,
}: {
  node: DebugNode;
  position: number;
  onSelect: (n: number) => void;
  depth: number;
  expanded: Set<number>;
  onToggle: (n: number) => void;
}) {
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
        title={`Run to node ${node.number}`}
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
        {node.number === position && <span className="text-amber-400">▶ </span>}
        <span className={cn(structural && node.name === 'decisiontable' && 'font-semibold text-foreground')}>
          {nodeLabel(node)}
        </span>
      </div>
      {expanded &&
        node.children.map((c) => (
          <TreeNodeView key={c.number} node={c} position={position} onSelect={onSelect} depth={depth + 1} expanded={expandedSet} onToggle={onToggle} />
        ))}
    </div>
  );
}

export function DebugPanel() {
  const { selectTable, setActiveTab } = useProjectStore();

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

  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [marks, setMarks] = useState<{ node: number; label: string; auto: boolean }[]>([]);
  const idxRef = useRef<TreeIndex | null>(null);
  const lastTableRef = useRef<number | null>(null);

  const toggleNode = useCallback((n: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(n)) next.delete(n);
      else next.add(n);
      return next;
    });
  }, []);

  const goTo = useCallback(async (node: number) => {
    const r = await debugPosition(node);
    if (!r.success) return;
    const pos = r.position || node;
    setPosition(pos);
    setContext(r.context || {});
    setStack(r.stack || []);

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
    lastTableRef.current = null;
    const t = await debugTree();
    if (t.success && t.tree) {
      setTree(t.tree);
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
        <Button variant="outline" size="sm" className="h-7 text-amber-400 border-amber-400/40" onClick={stepOver} title="Execute one action">
          Step
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={stepInto} title="Descend into the table this action performs">
          Into
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={stepPass} title="Finish this pass of the table (lands at the next pass on iterating contexts)">
          Pass
        </Button>
        <Button variant="outline" size="sm" className="h-7" onClick={stepUp} title="Finish this table and return to the caller">
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
        <span className="text-xs text-muted-foreground ml-2 hidden xl:inline">Click any tree node to run to it</span>
        <span className="ml-auto font-mono text-xs text-muted-foreground flex items-center gap-1.5">
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

      {/* Context breadcrumb */}
      {!context.table && position === idxRef.current?.endNode && (
        <div className="px-4 py-1.5 border-b border-border/50 text-sm text-muted-foreground">
          End of execution — final state{' '}
          <span className="text-green-500">(verified against replay on load)</span>
        </div>
      )}
      {context.table && (
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
          <div className="p-2">
            {tree && <TreeNodeView node={tree} position={position} onSelect={goTo} depth={0} expanded={expanded} onToggle={toggleNode} />}
          </div>
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

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
import { ArrowLeft, ArrowRight, Bug, ExternalLink } from 'lucide-react';

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

function TreeNodeView({
  node,
  position,
  onSelect,
  depth,
}: {
  node: DebugNode;
  position: number;
  onSelect: (n: number) => void;
  depth: number;
}) {
  const [expanded, setExpanded] = useState(depth < 2);
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
      >
        <span
          className="w-3 text-muted-foreground shrink-0"
          onClick={(e) => {
            e.stopPropagation();
            setExpanded(!expanded);
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
          <TreeNodeView key={c.number} node={c} position={position} onSelect={onSelect} depth={depth + 1} />
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

  const goTo = useCallback(async (node: number) => {
    const r = await debugPosition(node);
    if (r.success) {
      setPosition(r.position || node);
      setContext(r.context || {});
      setStack(r.stack || []);
    }
  }, []);

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
    const t = await debugTree();
    if (t.success && t.tree) setTree(t.tree);
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
      <div className="px-4 py-2 border-b border-border/50 bg-muted/20 flex items-center gap-2">
        <Button variant="outline" size="sm" className="h-7" onClick={() => goTo(Math.max(1, position - 1))} disabled={position <= 1}>
          <ArrowLeft className="h-3.5 w-3.5 mr-1" /> Back
        </Button>
        <Button variant="outline" size="sm" className="h-7 text-amber-400 border-amber-400/40" onClick={() => goTo(Math.min(nodeCount, position + 1))} disabled={position >= nodeCount}>
          Step <ArrowRight className="h-3.5 w-3.5 ml-1" />
        </Button>
        <span className="text-xs text-muted-foreground ml-2">Click any tree node to run to it</span>
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
            {tree && <TreeNodeView node={tree} position={position} onSelect={goTo} depth={0} />}
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

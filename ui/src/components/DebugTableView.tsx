/**
 * DebugTableView - the primary view of a trace: the decision table under
 * execution, rendered as the sheet grid.
 *
 * - The fired column is highlighted; condition rows carry their actual
 *   evaluated results from the trace.
 * - Executed actions are click-to-focus: entry row = state coming into the
 *   pass, an action row = state just after that action, exit row = state
 *   leaving the pass. The entity stack panel follows the focus.
 * - A performed table in an executed action is a link that drills into the
 *   called table's fired pass; stepping out returns to the caller at the
 *   following action.
 * - Iterating contexts: the pass navigator shows which of N iterations is
 *   in view — arrows step iterations, the number field jumps to any one.
 *
 * @module components/DebugTableView
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import { getDecisionTable } from '@/api/client';
import type { DebugFrame, DebugNode } from '@/api/client';
import type { DecisionTable } from '@/types/dtrules';
import { cn } from '@/lib/utils';
import { focusTarget, frameInfo, type Focus, type TreeIndex } from '@/lib/traceTree';
import { ChevronLeft, ChevronRight, CornerLeftUp } from 'lucide-react';
import { Button } from '@/components/ui/button';

const PERFORM_RE = /\bperform\s+([A-Za-z_][A-Za-z0-9_]*)|\/([A-Za-z_][A-Za-z0-9_]*)\s+performtable\b/gi;

/** How a performed-table name in DSL resolves (EL names are case-blind):
 *  - drill: this action called it right here — drill into that execution
 *  - jump: it executed elsewhere in the trace — move the trace there
 *  - open: never executed in this trace — open the static definition */
type LinkTarget =
  | { kind: 'drill'; dt: DebugNode }
  | { kind: 'jump'; pass: DebugNode }
  | { kind: 'open'; table: string };

const IDENT_RE = /[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?/g;

/** Resolves a DSL identifier against the entity stack at the current
 *  replay position. `entity.attr` finds the topmost frame of that entity;
 *  a bare name finds the topmost frame carrying that attribute. EL names
 *  are case-insensitive. Returns undefined when nothing matches (keywords,
 *  literals, unknown names). */
function stackValue(stack: DebugFrame[], ident: string): string | undefined {
  const lc = ident.toLowerCase();
  const [head, tail] = lc.includes('.') ? lc.split('.', 2) : ['', lc];
  for (let i = stack.length - 1; i >= 0; i--) {
    const f = stack[i];
    if (head && f.name.toLowerCase() !== head) continue;
    for (const [k, v] of Object.entries(f.attrs)) {
      if (k.toLowerCase() === tail) return v === '' ? '(empty)' : v;
    }
    if (head) return undefined; // right entity, no such attribute
  }
  return undefined;
}

/** Wraps identifiers in a plain-text DSL segment with hover tooltips that
 *  show their value on the entity stack at the current position. */
function withValueHovers(text: string, stack: DebugFrame[], keyBase: number): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  let last = 0;
  for (const m of text.matchAll(IDENT_RE)) {
    const value = stackValue(stack, m[0]);
    if (value === undefined) continue;
    const start = m.index ?? 0;
    parts.push(text.slice(last, start));
    parts.push(
      <span
        key={`${keyBase}-${start}`}
        className="cursor-help decoration-dotted underline-offset-2 hover:underline hover:text-foreground"
        title={`${m[0]} = ${value}`}
      >
        {m[0]}
      </span>
    );
    last = start + m[0].length;
  }
  parts.push(text.slice(last));
  return parts;
}

/** Render DSL with performed-table names as navigation links, and every
 *  other known field hoverable to show its value at the current position. */
function DSLWithLinks({
  text,
  resolve,
  stack,
  actionNumber,
  onDrill,
  onJump,
  onOpenTable,
}: {
  text: string;
  resolve: (name: string) => LinkTarget | null;
  stack: DebugFrame[];
  /** Ordinal of the enclosing action; calls inside a multi-call action are
   *  sub-numbered actionNumber.1, actionNumber.2, ... */
  actionNumber?: number;
  onDrill: (dt: DebugNode) => void;
  onJump: (pass: DebugNode) => void;
  onOpenTable: (table: string) => void;
}) {
  const totalCalls = [...text.matchAll(PERFORM_RE)].length;
  let callIndex = 0;
  const parts: React.ReactNode[] = [];
  let last = 0;
  for (const match of text.matchAll(PERFORM_RE)) {
    const name = match[1] || match[2];
    callIndex++;
    const callLabel =
      actionNumber !== undefined && totalCalls > 1 ? `${actionNumber}.${callIndex}` : undefined;
    const target = resolve(name);
    if (!target) continue;
    const start = (match.index ?? 0) + match[0].indexOf(name);
    parts.push(...withValueHovers(text.slice(last, start), stack, last));
    if (callLabel) {
      parts.push(
        <span key={`call-${start}`} className="text-[9px] text-muted-foreground align-super mr-0.5">
          {callLabel}
        </span>
      );
    }
    parts.push(
      <button
        key={`link-${start}`}
        className={cn(
          'underline underline-offset-2',
          target.kind === 'open'
            ? 'text-muted-foreground decoration-dashed decoration-muted-foreground/50 hover:text-foreground'
            : 'text-blue-400 decoration-blue-400/40 hover:text-blue-300'
        )}
        onClick={(e) => {
          e.stopPropagation();
          if (target.kind === 'drill') onDrill(target.dt);
          else if (target.kind === 'jump') onJump(target.pass);
          else onOpenTable(target.table);
        }}
        title={
          target.kind === 'drill'
            ? `Drill into ${name}`
            : target.kind === 'jump'
              ? `Go to ${name}'s execution (moves the trace position there)`
              : `${name} did not run in this trace — open its definition (Esc returns here)`
        }
      >
        {name}
      </button>
    );
    last = start + name.length;
  }
  parts.push(...withValueHovers(text.slice(last), stack, last));
  return <>{parts}</>;
}

export function DebugTableView({
  idx,
  passNode,
  focus,
  knownTables,
  stack,
  onFocus,
  onDrill,
  onOut,
  onPass,
  onOpenTable,
}: {
  idx: TreeIndex;
  passNode: DebugNode;
  focus: Focus;
  /** Project tables: lowercased name -> canonical name (EL is case-blind). */
  knownTables: Map<string, string>;
  /** Entity stack at the current replay position (bottom first) — the
   *  source for hover value lookups; changes as the focus moves. */
  stack: DebugFrame[];
  onFocus: (f: Focus) => void;
  onDrill: (calledDT: DebugNode) => void;
  onOut: () => void;
  onPass: (passNode: DebugNode) => void;
  onOpenTable: (name: string) => void;
}) {
  const frame = frameInfo(idx, passNode);
  const [table, setTable] = useState<DecisionTable | null>(null);
  const tableCache = useRef<Record<string, DecisionTable>>({});

  // Every execution pass in the trace, by lowercased table name, in trace
  // order — so a table name is navigable even when this action didn't call
  // it (jump to where it DID run, moving the trace position there).
  const executions = useMemo(() => {
    const m = new Map<string, DebugNode[]>();
    for (const n of idx.byNumber.values()) {
      if (n.name !== 'decisiontable' || !n.attrs?.name) continue;
      const key = n.attrs.name.toLowerCase();
      const list = m.get(key) || [];
      for (const c of n.children) if (c.name === 'execute_table') list.push(c);
      m.set(key, list);
    }
    for (const list of m.values()) list.sort((a, b) => a.number - b.number);
    return m;
  }, [idx]);

  useEffect(() => {
    const name = frame?.tableName;
    if (!name) return;
    if (tableCache.current[name]) {
      setTable(tableCache.current[name]);
      return;
    }
    setTable(null);
    getDecisionTable(name).then((r) => {
      if (r.success) {
        const t = r as unknown as DecisionTable;
        tableCache.current[name] = t;
        setTable(t);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [frame?.tableName]);

  if (!frame) return <div className="p-6 text-sm text-muted-foreground">Not inside a table pass.</div>;

  // Executed action nodes by trace "n" — the 1-based POSITION of the
  // action in the table's action list (the engine traces positions, not
  // authored numbers, which may have gaps like 1,2,4,7).
  const executedByPosition = new Map<string, DebugNode>();
  for (const a of frame.actions) {
    if (a.attrs?.n) executedByPosition.set(a.attrs.n, a);
  }
  // Called tables per executed action, by table name.
  const calledTablesOf = (a: DebugNode): Map<string, DebugNode> => {
    const m = new Map<string, DebugNode>();
    for (const c of a.children) {
      if (c.name === 'decisiontable' && c.attrs?.name) m.set(c.attrs.name.toLowerCase(), c);
    }
    return m;
  };

  // Resolve a performed-table name for one action's DSL. Preference order:
  // the call right here, then the execution nearest before the current
  // position (else the first after — "go back in the trace to that point"),
  // then the static definition for tables this trace never ran.
  const currentPos = focusTarget(idx, passNode, focus);
  const resolverFor = (called: Map<string, DebugNode>) => (name: string): LinkTarget | null => {
    const lc = name.toLowerCase();
    const dt = called.get(lc);
    // Zero-pass calls (context iterated zero times) drill too — the view
    // shows the table with the state going into it and a banner.
    if (dt) return { kind: 'drill', dt };
    const passes = executions.get(lc);
    if (passes && passes.length > 0) {
      let pick = passes[0];
      for (const p of passes) {
        if (p.number <= currentPos) pick = p;
        else break;
      }
      return { kind: 'jump', pass: pick };
    }
    const canonical = knownTables.get(lc);
    if (canonical) return { kind: 'open', table: canonical };
    return null;
  };

  // Caller breadcrumb chain (outermost first).
  const crumbs: { name: string; pass: DebugNode }[] = [];
  let walker = frame.callerPass;
  let walkerIdx = idx;
  while (walker) {
    const fi = frameInfo(walkerIdx, walker);
    if (!fi) break;
    crumbs.unshift({ name: fi.tableName, pass: walker });
    walker = fi.callerPass;
  }

  const cols = Array.from({ length: table?.columnCount || 0 }, (_, i) => `${i + 1}`);
  const fired = frame.firedColumn;

  const cellOf = (columns: Record<string | number, string> | undefined, c: string): string => {
    if (!columns) return '';
    const v = columns[c] ?? columns[parseInt(c, 10)];
    return v ? String(v).trim().toUpperCase() : '';
  };

  const focusedAction = focus.kind === 'action' ? focus.node : null;

  return (
    <div className="p-4">
      {/* Call stack + pass navigator */}
      <div className="flex items-center gap-2 flex-wrap mb-2">
        {crumbs.map((c) => (
          <span key={c.pass.number} className="flex items-center gap-2 text-sm">
            <button className="text-blue-400 hover:underline underline-offset-2" onClick={() => onPass(c.pass)}>
              {c.name}
            </button>
            <span className="text-muted-foreground">▸</span>
          </span>
        ))}
        {(() => {
          // Label this table with its call position in the caller: action
          // ordinal, plus .k when that action performs several tables
          // (action 5's calls are 5.1, 5.2, ... — matching the DSL row).
          const a = frame.callerAction;
          if (!a) return null;
          const calls = a.children.filter((c) => c.name === 'decisiontable');
          const prefix = a.name === 'initialaction' ? 'I' : '';
          const n = `${prefix}${a.attrs?.n ?? '?'}`;
          const k = calls.length > 1 ? `.${calls.findIndex((c) => c.number === frame.dtNode.number) + 1}` : '';
          return (
            <span className="text-sm font-mono text-muted-foreground" title="Call position in the caller (action, and call within the action)">
              {n}
              {k}
            </span>
          );
        })()}
        <span className="text-lg font-bold">{frame.tableName}</span>
        {frame.callerPass && (
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={onOut} title="Return to the caller at the following action">
            <CornerLeftUp className="h-3 w-3 mr-1" /> Out
          </Button>
        )}
        {/* Iteration navigator: which of N passes is in view */}
        {frame.passes.length === 0 ? (
          <span className="ml-auto text-xs text-amber-400/90">
            executed here — its “for all” context had nothing to iterate, so no passes were recorded
          </span>
        ) : (
        <span className="ml-auto flex items-center gap-1.5 text-xs text-muted-foreground">
          {frame.passes.length > 1 ? 'Iteration' : 'Pass'}
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            disabled={frame.passIndex <= 0}
            onClick={() => onPass(frame.passes[frame.passIndex - 1])}
            title="Previous iteration"
          >
            <ChevronLeft className="h-3.5 w-3.5" />
          </Button>
          <input
            key={frame.passNode.number}
            defaultValue={frame.passIndex + 1}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const n = parseInt((e.target as HTMLInputElement).value, 10);
                if (!isNaN(n) && n >= 1 && n <= frame.passes.length) onPass(frame.passes[n - 1]);
              }
            }}
            className="w-14 h-6 px-1 rounded border border-input bg-transparent text-center font-mono text-foreground"
            title="Type an iteration number and press Enter"
          />
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            disabled={frame.passIndex >= frame.passes.length - 1}
            onClick={() => onPass(frame.passes[frame.passIndex + 1])}
            title="Next iteration"
          >
            <ChevronRight className="h-3.5 w-3.5" />
          </Button>
          of <b className="text-foreground">{frame.passes.length.toLocaleString()}</b>
        </span>
        )}
      </div>

      {!table ? (
        <div className="text-sm text-muted-foreground p-4">Loading table…</div>
      ) : (
        <div className="overflow-x-auto border border-border/40 rounded-md">
          <table className="border-collapse w-full text-xs">
            <thead>
              <tr>
                <th className="border border-border/40 bg-muted/30 w-8 text-muted-foreground font-semibold">#</th>
                <th className="border border-border/40 bg-muted/30 px-2 text-left text-muted-foreground font-semibold">DSL</th>
                {cols.map((c) => (
                  <th
                    key={c}
                    className={cn(
                      'border border-border/40 w-8 text-center font-semibold',
                      c === fired ? 'bg-blue-500/20 text-blue-300' : 'bg-muted/30 text-muted-foreground'
                    )}
                  >
                    {c === fired ? `${c} ▶` : c}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {/* entry focus row */}
              <tr
                className={cn(
                  'cursor-pointer',
                  focus.kind === 'entry' ? 'bg-amber-500/15' : 'hover:bg-accent/40'
                )}
                onClick={() => onFocus({ kind: 'entry' })}
                title="State coming into this pass"
              >
                <td colSpan={2 + cols.length} className="border border-border/40 px-2 py-1 text-muted-foreground">
                  {focus.kind === 'entry' && <span className="text-amber-400 font-bold">▶ </span>}
                  Entering {frame.tableName} —{' '}
                  {frame.passes.length === 0 ? 'state going into this call' : 'state before this pass'}
                </td>
              </tr>

              {/* initial actions */}
              {frame.initialActions.map((ia) => {
                const isFocus = focusedAction === ia.number;
                const called = calledTablesOf(ia);
                return (
                  <tr
                    key={ia.number}
                    className={cn('cursor-pointer', isFocus ? 'bg-amber-500/15' : 'hover:bg-accent/40')}
                    onClick={() => onFocus({ kind: 'action', node: ia.number })}
                    title="State after this initial action"
                  >
                    <td className="border border-border/40 text-center text-muted-foreground">I{ia.attrs?.n}</td>
                    <td className="border border-border/40 px-2 py-1 font-mono whitespace-pre-wrap" colSpan={1 + cols.length}>
                      {isFocus && <span className="text-amber-400 font-bold">▶ </span>}
                      <DSLWithLinks
                        text={initialActionText(table, ia.attrs?.n)}
                        resolve={resolverFor(called)}
                        stack={stack}
                        onDrill={onDrill}
                        onJump={onPass}
                        onOpenTable={onOpenTable}
                      />
                    </td>
                  </tr>
                );
              })}

              {/* conditions with actual results */}
              {(table.conditions || []).map((cond, condIdx) => {
                return (
                  <tr key={`c${condIdx}`}>
                    <td className="border border-border/40 text-center text-muted-foreground">{condIdx + 1}</td>
                    <td className="border border-border/40 px-2 py-1 font-mono whitespace-pre-wrap">
                      {withValueHovers(cond.description || cond.postfix || '', stack, cond.number)}
                    </td>
                    {cols.map((c) => {
                      const v = cellOf(cond.columns as Record<string, string>, c);
                      return (
                        <td
                          key={c}
                          className={cn(
                            'border border-border/40 text-center font-bold',
                            c === fired && 'bg-blue-500/10',
                            v === 'Y' && 'text-green-500',
                            v === 'N' && 'text-red-500',
                            (v === '-' || v === '*') && 'text-muted-foreground'
                          )}
                        >
                          {v === '-' ? '' : v}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}

              {/* actions: executed ones focusable, others dimmed */}
              {(table.actions || []).map((act, actIdx) => {
                const executed = executedByPosition.get(String(actIdx + 1));
                const isFocus = executed ? focusedAction === executed.number : false;
                const called = executed ? calledTablesOf(executed) : new Map<string, DebugNode>();
                return (
                  <tr
                    key={`a${actIdx}`}
                    className={cn(
                      executed ? 'cursor-pointer' : 'opacity-40',
                      isFocus ? 'bg-amber-500/15' : executed && 'hover:bg-accent/40'
                    )}
                    onClick={() => executed && onFocus({ kind: 'action', node: executed.number })}
                    title={executed ? 'State after this action' : 'Not executed in this pass'}
                  >
                    <td className="border border-border/40 text-center text-muted-foreground">{actIdx + 1}</td>
                    <td className="border border-border/40 px-2 py-1 font-mono whitespace-pre-wrap">
                      {isFocus && <span className="text-amber-400 font-bold">▶ </span>}
                      <DSLWithLinks
                        text={act.description || act.postfix || ''}
                        resolve={resolverFor(called)}
                        stack={stack}
                        actionNumber={actIdx + 1}
                        onDrill={onDrill}
                        onJump={onPass}
                        onOpenTable={onOpenTable}
                      />
                    </td>
                    {cols.map((c) => {
                      const v = cellOf(act.columns as Record<string, string>, c);
                      return (
                        <td
                          key={c}
                          className={cn(
                            'border border-border/40 text-center font-bold',
                            c === fired && 'bg-blue-500/10',
                            v === 'X' && 'text-blue-400'
                          )}
                        >
                          {v}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}

              {/* exit focus row */}
              <tr
                className={cn('cursor-pointer', focus.kind === 'exit' ? 'bg-amber-500/15' : 'hover:bg-accent/40')}
                onClick={() => onFocus({ kind: 'exit' })}
                title="State leaving this pass"
              >
                <td colSpan={2 + cols.length} className="border border-border/40 px-2 py-1 text-muted-foreground">
                  {focus.kind === 'exit' && <span className="text-amber-400 font-bold">▶ </span>}
                  Leaving {frame.tableName} — state after this pass
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

/** The DSL text of initial action n (1-based) from the table definition. */
function initialActionText(table: DecisionTable, n?: string): string {
  const lines = (table.initialActions || '').split('\n').filter((l) => l.trim() !== '');
  const i = n ? parseInt(n, 10) - 1 : -1;
  return (i >= 0 && lines[i]) || `initial action ${n || ''}`;
}

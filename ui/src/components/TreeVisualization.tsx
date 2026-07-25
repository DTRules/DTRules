/**
 * TreeVisualization - Decision table tree as a vertical expandable list.
 *
 * Each node is one row; nodes with children get an expand arrow. A
 * condition's branches carry Y/N tags; conditions are blue, actions green
 * (matching the legend). Large child lists group into ordinal ranges
 * ([1…100] → [26…50] → rows) so any child is a few clicks away instead of
 * paging through a flood.
 *
 * @module components/TreeVisualization
 */

import { useCallback, useEffect, useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { getDecisionTree } from '@/api/client';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import { GitBranch } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { TreeNode } from '@/types/dtrules';

/** A child edge: the node plus the branch it hangs from (Y/N for conditions). */
interface ChildEdge {
  branch?: 'Y' | 'N';
  node: TreeNode;
}

function childEdges(n: TreeNode): ChildEdge[] {
  const out: ChildEdge[] = [];
  if (n.trueChild) out.push({ branch: 'Y', node: n.trueChild });
  if (n.falseChild) out.push({ branch: 'N', node: n.falseChild });
  for (const c of n.children || []) out.push({ node: c });
  return out;
}

const GROUP_SIZES = [25, 100, 1000, 10000];

function bucketSize(n: number): number {
  for (const size of GROUP_SIZES) {
    if (Math.ceil(n / size) <= 20) return size;
  }
  return GROUP_SIZES[GROUP_SIZES.length - 1];
}

function NodeRow({ edge, depth }: { edge: ChildEdge; depth: number }) {
  const { node, branch } = edge;
  const kids = childEdges(node);
  const [expanded, setExpanded] = useState(depth < 3);

  const chip =
    node.type === 'condition'
      ? 'bg-blue-600/20 border-blue-500/50 text-blue-300'
      : node.type === 'action' || node.type === 'actions'
        ? 'bg-green-600/20 border-green-500/50 text-green-300'
        : 'bg-muted/40 border-border text-muted-foreground';

  return (
    <div>
      <div
        className="flex items-start gap-1.5 py-0.5 rounded-sm hover:bg-accent/50 cursor-pointer"
        style={{ paddingLeft: depth * 16 + 4 }}
        onClick={() => setExpanded(!expanded)}
      >
        <span className="w-3 pt-1 text-muted-foreground text-xs shrink-0 font-mono">
          {kids.length > 0 ? (expanded ? '▾' : '▸') : ''}
        </span>
        {branch && (
          <span
            className={cn(
              'mt-0.5 w-4 text-center rounded text-[10px] font-bold shrink-0',
              branch === 'Y' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
            )}
          >
            {branch}
          </span>
        )}
        <span className={cn('px-2 py-0.5 rounded-md border text-xs', chip)}>
          <span className="font-semibold">{node.label}</span>
          {node.description && (
            <span className="ml-2 opacity-75 font-mono text-[11px]">{node.description}</span>
          )}
        </span>
      </div>
      {expanded && kids.length > 0 && <ChildList edges={kids} ordinalStart={1} depth={depth + 1} />}
    </div>
  );
}

function ChildList({
  edges,
  ordinalStart,
  depth,
}: {
  edges: ChildEdge[];
  ordinalStart: number;
  depth: number;
}) {
  if (edges.length <= 25) {
    return (
      <>
        {edges.map((e, i) => (
          <NodeRow key={e.node.id || i} edge={e} depth={depth} />
        ))}
      </>
    );
  }
  const size = bucketSize(edges.length);
  const groups: number[] = [];
  for (let i = 0; i < edges.length; i += size) groups.push(i);
  return (
    <>
      {groups.map((i) => (
        <RangeGroup
          key={i}
          edges={edges.slice(i, i + size)}
          ordinalStart={ordinalStart + i}
          depth={depth}
        />
      ))}
    </>
  );
}

function RangeGroup({
  edges,
  ordinalStart,
  depth,
}: {
  edges: ChildEdge[];
  ordinalStart: number;
  depth: number;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div>
      <div
        className="flex items-center gap-1 py-0.5 rounded-sm hover:bg-accent/50 cursor-pointer font-mono text-xs text-blue-300/80"
        style={{ paddingLeft: depth * 16 + 4 }}
        onClick={() => setOpen(!open)}
      >
        <span className="w-3 shrink-0">{open ? '▾' : '▸'}</span>
        <span>
          [{ordinalStart.toLocaleString()} … {(ordinalStart + edges.length - 1).toLocaleString()}]
        </span>
        <span className="text-muted-foreground">· {edges.length.toLocaleString()} items</span>
      </div>
      {open && <ChildList edges={edges} ordinalStart={ordinalStart} depth={depth + 1} />}
    </div>
  );
}

export function TreeVisualization() {
  const { decisionTables, currentTable } = useProjectStore();
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [tree, setTree] = useState<TreeNode | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadTree = useCallback(async (name: string) => {
    setError(null);
    try {
      const response = await getDecisionTree(name);
      if (response.success && response.tree) {
        setTree(response.tree);
      } else {
        setTree(null);
        setError(response.error || 'No tree available');
      }
    } catch {
      setTree(null);
      setError('Failed to load tree');
    }
  }, []);

  // Follow the table selected elsewhere in the app; allow local override.
  useEffect(() => {
    const name = currentTable?.tableName;
    if (name && name !== selectedTable) {
      setSelectedTable(name);
      loadTree(name);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTable?.tableName]);

  const handleSelect = (name: string) => {
    setSelectedTable(name);
    loadTree(name);
  };

  if (!decisionTables.length) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        <p>No decision tables loaded. Open a project to see the tree view.</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <div className="p-4 border-b border-border/50 flex items-end gap-4">
        <div className="grid gap-1">
          <Label className="text-xs text-muted-foreground">Decision Table</Label>
          <Select value={selectedTable} onValueChange={handleSelect}>
            <SelectTrigger className="w-72 h-9">
              <SelectValue placeholder="Select a table" />
            </SelectTrigger>
            <SelectContent>
              {decisionTables.map((t) => (
                <SelectItem key={t.name} value={t.name}>
                  {t.tableNumber ? `${t.tableNumber} · ` : ''}
                  {t.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="ml-auto flex items-center gap-4 text-xs pb-2">
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-sm bg-blue-500" /> Condition
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-sm bg-green-500" /> Action
          </span>
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4">
          {error && <p className="text-sm text-amber-400">{error}</p>}
          {!tree && !error && (
            <p className="text-sm text-muted-foreground flex items-center gap-2">
              <GitBranch className="h-4 w-4" /> Select a decision table to view its tree.
            </p>
          )}
          {tree && <NodeRow edge={{ node: tree }} depth={0} />}
        </div>
      </ScrollArea>
    </div>
  );
}

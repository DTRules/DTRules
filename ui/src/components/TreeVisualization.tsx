import { useCallback, useEffect, useState } from 'react';
import {
  ReactFlow,
  Node,
  Edge,
  Controls,
  Background,
  Handle,
  Position,
  useNodesState,
  useEdgesState,
  MarkerType,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
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
import { GitBranch } from 'lucide-react';
import type { TreeNode } from '@/types/dtrules';

// Custom node components with handles for edge connections
const ConditionNode = ({ data }: { data: { label: string; description?: string } }) => (
  <div className="px-4 py-2 shadow-md rounded-md bg-blue-600 border-2 border-blue-400 min-w-[150px]">
    <Handle type="target" position={Position.Top} className="!bg-blue-400" />
    <div className="font-bold text-white text-sm">{data.label}</div>
    {data.description && (
      <div className="text-blue-200 text-xs mt-1">{data.description}</div>
    )}
    <Handle type="source" position={Position.Bottom} className="!bg-blue-400" />
  </div>
);

const ActionNode = ({ data }: { data: { label: string; description?: string } }) => (
  <div className="px-4 py-2 shadow-md rounded-md bg-green-600 border-2 border-green-400 min-w-[150px]">
    <Handle type="target" position={Position.Top} className="!bg-green-400" />
    <div className="font-bold text-white text-sm">{data.label}</div>
    {data.description && (
      <div className="text-green-200 text-xs mt-1">{data.description}</div>
    )}
    <Handle type="source" position={Position.Bottom} className="!bg-green-400" />
  </div>
);

const StartNode = ({ data }: { data: { label: string } }) => (
  <div className="px-4 py-2 shadow-md rounded-full bg-gray-600 border-2 border-gray-400">
    <Handle type="target" position={Position.Top} className="!bg-gray-400" />
    <div className="font-bold text-white text-sm">{data.label}</div>
    <Handle type="source" position={Position.Bottom} className="!bg-gray-400" />
  </div>
);

// Default fallback node for any unrecognized types
const DefaultNode = ({ data }: { data: { label: string; description?: string } }) => (
  <div className="px-4 py-2 shadow-md rounded-md bg-purple-600 border-2 border-purple-400 min-w-[150px]">
    <Handle type="target" position={Position.Top} className="!bg-purple-400" />
    <div className="font-bold text-white text-sm">{data.label}</div>
    {data.description && (
      <div className="text-purple-200 text-xs mt-1">{data.description}</div>
    )}
    <Handle type="source" position={Position.Bottom} className="!bg-purple-400" />
  </div>
);

const nodeTypes = {
  condition: ConditionNode,
  action: ActionNode,
  actions: ActionNode,
  start: StartNode,
  end: StartNode,
  default: DefaultNode,
};

// Valid node types that have registered components
const validNodeTypes = new Set(['condition', 'action', 'actions', 'start', 'end']);

function treeToFlow(tree: TreeNode | null): { nodes: Node[]; edges: Edge[] } {
  if (!tree) return { nodes: [], edges: [] };

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const nodeIds = new Set<string>(); // Track created nodes for edge validation

  function traverse(node: TreeNode, x: number, y: number, parentId?: string, edgeLabel?: string) {
    const id = node.id;
    // Normalize node type - use 'default' for unrecognized types
    const nodeType = validNodeTypes.has(node.type) ? node.type : 'default';

    nodeIds.add(id);

    nodes.push({
      id,
      type: nodeType,
      position: { x, y },
      data: { label: node.label, description: node.description },
    });

    // Only create edge if parent node was created
    if (parentId && nodeIds.has(parentId)) {
      edges.push({
        id: `edge-${parentId}-${id}`,
        source: parentId,
        target: id,
        label: edgeLabel,
        markerEnd: { type: MarkerType.ArrowClosed },
        style: { stroke: edgeLabel === 'N' ? '#ef4444' : edgeLabel === 'Y' ? '#22c55e' : '#888' },
        labelStyle: { fill: '#fff', fontWeight: 700 },
      });
    }

    const childSpacing = 250;

    if (node.trueChild) {
      traverse(node.trueChild, x - 150, y + 150, id, 'Y');
    }

    if (node.falseChild) {
      traverse(node.falseChild, x + 150, y + 150, id, 'N');
    }

    if (node.children) {
      node.children.forEach((child, i) => {
        const offsetX = (i - (node.children!.length - 1) / 2) * childSpacing;
        traverse(child, x + offsetX, y + 150, id);
      });
    }
  }

  traverse(tree, 400, 50);

  return { nodes, edges };
}

export function TreeVisualization() {
  const { decisionTables } = useProjectStore();
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [isLoading, setIsLoading] = useState(false);

  // Auto-select first table when tables load and nothing is selected
  useEffect(() => {
    if (!selectedTable && decisionTables.length > 0) {
      setSelectedTable(decisionTables[0].name);
    }
  }, [decisionTables, selectedTable]);

  const loadTree = useCallback(async (tableName: string) => {
    if (!tableName) {
      setNodes([]);
      setEdges([]);
      return;
    }

    setIsLoading(true);
    try {
      const response = await getDecisionTree(tableName);
      if (response.success && response.tree) {
        const { nodes: newNodes, edges: newEdges } = treeToFlow(response.tree);
        setNodes(newNodes);
        setEdges(newEdges);
      }
    } catch (err) {
      console.error('Failed to load tree:', err);
    } finally {
      setIsLoading(false);
    }
  }, [setNodes, setEdges]);

  useEffect(() => {
    if (selectedTable) {
      loadTree(selectedTable);
    }
  }, [selectedTable, loadTree]);

  return (
    <div className="h-full flex flex-col" data-tutorial="tree-visualization">
      <div className="p-4 border-b border-border/50 bg-gradient-to-r from-muted/30 via-transparent to-muted/30 flex items-center gap-4">
        <div className="grid gap-1 w-64">
          <Label className="text-xs text-muted-foreground">Decision Table</Label>
          <Select value={selectedTable} onValueChange={setSelectedTable}>
            <SelectTrigger>
              <SelectValue placeholder="Select a table to visualize" />
            </SelectTrigger>
            <SelectContent>
              {decisionTables.map((table) => (
                <SelectItem key={table.name} value={table.name}>
                  {table.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {isLoading && (
          <span className="text-sm text-muted-foreground">Loading...</span>
        )}

        <div className="flex-1" />

        <div className="flex items-center gap-4 text-sm text-muted-foreground bg-muted/30 px-3 py-1.5 rounded-lg">
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-3 bg-blue-600 rounded shadow-[0_0_6px_rgba(59,130,246,0.5)]"></span>
            <span>Condition</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-3 bg-green-600 rounded shadow-[0_0_6px_rgba(34,197,94,0.5)]"></span>
            <span>Action</span>
          </div>
        </div>
      </div>

      <div className="flex-1">
        {nodes.length > 0 ? (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
          >
            <Controls />
            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
          </ReactFlow>
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-blue-500/20 to-green-500/20 flex items-center justify-center">
                <GitBranch className="h-8 w-8 text-blue-400/60" />
              </div>
              <p className="text-muted-foreground">Select a decision table to visualize its structure.</p>
              <p className="text-xs text-muted-foreground/60 mt-1">The tree shows how rules connect and flow</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

import { useProjectStore } from '@/stores/projectStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { reorderDecisionTables, reorderEntities } from '@/api/client';
import {
  FileText,
  Table2,
  Map,
  ChevronRight,
  ChevronDown,
  Folder,
} from 'lucide-react';
import { useState } from 'react';

/**
 * New order after dragging `from` onto `to`: `from` lands before `to` when
 * dragged upward and after it when dragged downward.
 */
function orderAfterMove(names: string[], from: string, to: string): string[] {
  const fromIdx = names.indexOf(from);
  const toIdx = names.indexOf(to);
  if (fromIdx < 0 || toIdx < 0 || from === to) return names;
  const list = names.filter((n) => n !== from);
  let insertIdx = list.indexOf(to);
  if (fromIdx < toIdx) insertIdx += 1;
  list.splice(insertIdx, 0, from);
  return list;
}

interface TreeNodeProps {
  label: React.ReactNode;
  icon: React.ReactNode;
  children?: React.ReactNode;
  onClick?: () => void;
  selected?: boolean;
  modified?: boolean;
  dataTutorial?: string;
}

function TreeNode({ label, icon, children, onClick, selected, modified, dataTutorial }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = !!children;

  return (
    <div data-tutorial={dataTutorial}>
      <div
        className={cn(
          "flex items-center gap-1 px-2 py-1 hover:bg-accent cursor-pointer rounded-sm",
          selected && "bg-accent"
        )}
        onClick={() => {
          if (hasChildren) {
            setExpanded(!expanded);
          }
          onClick?.();
        }}
      >
        {hasChildren ? (
          expanded ? (
            <ChevronDown className="h-4 w-4 shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 shrink-0" />
          )
        ) : (
          <span className="w-4" />
        )}
        {icon}
        <span className={cn("text-sm truncate", modified && "italic")}>
          {label}
          {modified && " *"}
        </span>
      </div>
      {hasChildren && expanded && (
        <div className="ml-4">{children}</div>
      )}
    </div>
  );
}

export function ProjectExplorer() {
  const {
    projectPath,
    mapFiles,
    entities,
    decisionTables,
    currentEntity,
    currentTable,
    selectEntity,
    selectTable,
    setActiveTab,
    loadEDD,
    loadDecisionTables,
    readOnly,
  } = useProjectStore();

  // Drag-and-drop reordering state: what's being dragged and what row the
  // pointer is currently over.
  const [dragging, setDragging] = useState<{ kind: 'entity' | 'table'; name: string } | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);

  const handleDrop = async (kind: 'entity' | 'table', targetName: string) => {
    if (!dragging || dragging.kind !== kind || dragging.name === targetName) return;
    if (kind === 'table') {
      const order = orderAfterMove(
        decisionTables.map((t) => t.name),
        dragging.name,
        targetName
      );
      await reorderDecisionTables(order);
      await loadDecisionTables();
    } else {
      const order = orderAfterMove(
        entities.map((e) => e.name),
        dragging.name,
        targetName
      );
      await reorderEntities(order);
      await loadEDD();
    }
  };

  /** Wraps a row with HTML5 drag-and-drop handlers for reordering. */
  const draggableRow = (kind: 'entity' | 'table', name: string, child: React.ReactNode) => (
    <div
      key={name}
      draggable={!readOnly}
      onDragStart={() => setDragging({ kind, name })}
      onDragOver={(e) => {
        if (dragging?.kind === kind) {
          e.preventDefault();
          setDropTarget(name);
        }
      }}
      onDragLeave={() => setDropTarget((t) => (t === name ? null : t))}
      onDrop={(e) => {
        e.preventDefault();
        handleDrop(kind, name);
        setDragging(null);
        setDropTarget(null);
      }}
      onDragEnd={() => {
        setDragging(null);
        setDropTarget(null);
      }}
      className={cn(
        dragging?.kind === kind && dropTarget === name && dragging.name !== name &&
          'outline outline-1 outline-blue-400/60 rounded-sm'
      )}
    >
      {child}
    </div>
  );

  if (!projectPath) {
    return (
      <div className="p-4 text-muted-foreground text-sm">
        <p>No project open.</p>
        <p className="mt-2">Use the Open button in the toolbar to open a DTRules project.</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      <div className="p-3 border-b border-border/50 bg-gradient-to-r from-muted/50 to-transparent">
        <span className="text-sm font-semibold text-foreground/80">Project Explorer</span>
      </div>
      <ScrollArea className="flex-1">
        <div className="p-2">
        {/* Entities */}
        {(() => {
          const validEntities = entities.filter(e => e.name && e.name.trim() !== '');
          return (
            <TreeNode
              label={`Entities (${validEntities.length})`}
              icon={<Folder className="h-4 w-4 text-blue-500" />}
              dataTutorial="entities-section"
            >
              {validEntities.map((entity) =>
                draggableRow(
                  'entity',
                  entity.name,
                  <TreeNode
                    label={
                      <>
                        {entity.number && (
                          <span className="text-muted-foreground font-mono mr-1.5">
                            {entity.number}
                          </span>
                        )}
                        {entity.name}
                      </>
                    }
                    icon={<FileText className="h-4 w-4 text-blue-400" />}
                    onClick={() => {
                      selectEntity(entity.name);
                      setActiveTab('edd');
                    }}
                    selected={currentEntity?.name === entity.name}
                  />
                )
              )}
            </TreeNode>
          );
        })()}

        {/* Decision Tables */}
        {(() => {
          const validTables = decisionTables.filter(t => t.name && t.name.trim() !== '');
          return (
            <TreeNode
              label={`Decision Tables (${validTables.length})`}
              icon={<Folder className="h-4 w-4 text-green-500" />}
              dataTutorial="decision-tables-section"
            >
              {validTables.map((table) =>
                draggableRow(
                  'table',
                  table.name,
                  <TreeNode
                    label={
                      <>
                        {table.tableNumber && (
                          <span className="text-muted-foreground font-mono mr-1.5">
                            {table.tableNumber}
                          </span>
                        )}
                        {table.name}
                      </>
                    }
                    icon={<Table2 className="h-4 w-4 text-green-400" />}
                    onClick={() => {
                      selectTable(table.name);
                      setActiveTab('dt');
                    }}
                    selected={currentTable?.tableName === table.name}
                  />
                )
              )}
            </TreeNode>
          );
        })()}

        {/* Map Files */}
        {mapFiles.length > 0 && (
          <TreeNode
            label="Maps"
            icon={<Folder className="h-4 w-4 text-yellow-500" />}
          >
            {mapFiles.map((file) => (
              <TreeNode
                key={file}
                label={file}
                icon={<Map className="h-4 w-4 text-purple-400" />}
              />
            ))}
          </TreeNode>
        )}
        </div>
      </ScrollArea>
    </div>
  );
}

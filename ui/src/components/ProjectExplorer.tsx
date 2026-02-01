import { useProjectStore } from '@/stores/projectStore';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import {
  FileText,
  Table2,
  Map,
  ChevronRight,
  ChevronDown,
  Folder,
} from 'lucide-react';
import { useState } from 'react';

interface TreeNodeProps {
  label: string;
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
    eddFiles,
    dtFiles,
    mapFiles,
    entities,
    decisionTables,
    currentEntity,
    currentTable,
    selectEntity,
    selectTable,
    setActiveTab,
    files,
  } = useProjectStore();

  const getFileModified = (path: string): boolean => {
    const file = files.find(f => f.path === path);
    return file?.modified ?? false;
  };

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
        {/* EDD Files */}
        <TreeNode
          label="Entities (EDD)"
          icon={<Folder className="h-4 w-4 text-yellow-500" />}
          dataTutorial="entities-section"
        >
          {eddFiles.map((file) => (
            <TreeNode
              key={file}
              label={file}
              icon={<FileText className="h-4 w-4 text-blue-400" />}
              modified={getFileModified(file)}
            >
              {entities.map((entity) => (
                <TreeNode
                  key={entity.name}
                  label={entity.name}
                  icon={<FileText className="h-4 w-4 text-blue-300" />}
                  onClick={() => {
                    selectEntity(entity.name);
                    setActiveTab('edd');
                  }}
                  selected={currentEntity?.name === entity.name}
                />
              ))}
            </TreeNode>
          ))}
        </TreeNode>

        {/* Decision Tables */}
        <TreeNode
          label="Decision Tables"
          icon={<Folder className="h-4 w-4 text-yellow-500" />}
          dataTutorial="decision-tables-section"
        >
          {dtFiles.map((file) => (
            <TreeNode
              key={file}
              label={file}
              icon={<Table2 className="h-4 w-4 text-green-400" />}
              modified={getFileModified(file)}
            >
              {decisionTables.map((table) => (
                <TreeNode
                  key={table.name}
                  label={table.name}
                  icon={<Table2 className="h-4 w-4 text-green-300" />}
                  onClick={() => {
                    selectTable(table.name);
                    setActiveTab('dt');
                  }}
                  selected={currentTable?.tableName === table.name}
                />
              ))}
            </TreeNode>
          ))}
        </TreeNode>

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

/**
 * ProjectPicker - Recently-opened projects plus a file system browser.
 *
 * Shared by the toolbar's Open dialog and the welcome screen so a project
 * can be reopened with one click, or found by navigating the file system
 * instead of typing a path.
 *
 * @module components/ProjectPicker
 */

import { useProjectStore } from '@/stores/projectStore';
import { FileBrowser } from '@/components/FileBrowser';
import { Button } from '@/components/ui/button';
import { Clock, FolderOpen, X } from 'lucide-react';

interface ProjectPickerProps {
  /** Called with the chosen directory path */
  onOpen: (path: string) => void;
}

export function ProjectPicker({ onOpen }: ProjectPickerProps) {
  const { recentProjects, removeRecentProject } = useProjectStore();

  return (
    <div className="flex flex-col gap-4">
      {recentProjects.length > 0 && (
        <div>
          <div className="flex items-center gap-1.5 mb-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
            <Clock className="h-3.5 w-3.5" />
            Recent Projects
          </div>
          <div className="border border-border/50 rounded-md divide-y divide-border/40 max-h-40 overflow-y-auto">
            {recentProjects.map((path) => (
              <div
                key={path}
                className="group flex items-center gap-2 px-2 py-1.5 hover:bg-accent cursor-pointer"
                onClick={() => onOpen(path)}
                title={path}
              >
                <FolderOpen className="h-4 w-4 text-blue-400 shrink-0" />
                <span className="text-sm font-medium shrink-0">
                  {path.replace(/\/+$/, '').split('/').pop()}
                </span>
                <span className="flex-1 text-xs font-mono text-muted-foreground truncate">
                  {path}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 opacity-0 group-hover:opacity-100 shrink-0"
                  title="Remove from history"
                  onClick={(e) => {
                    e.stopPropagation();
                    removeRecentProject(path);
                  }}
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <div className="mb-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
          Browse
        </div>
        <FileBrowser onSelect={onOpen} />
      </div>
    </div>
  );
}

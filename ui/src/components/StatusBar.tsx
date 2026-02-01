import { cn } from '@/lib/utils';
import { Loader2 } from 'lucide-react';

interface StatusBarProps {
  projectPath: string | null;
  isLoading: boolean;
  backendConnected: boolean;
}

export function StatusBar({ projectPath, isLoading, backendConnected }: StatusBarProps) {
  return (
    <div className="h-7 border-t border-border/50 flex items-center px-4 text-xs text-muted-foreground bg-gradient-to-r from-muted/30 via-transparent to-muted/30">
      {/* Connection status with glow dot */}
      <div className="flex items-center gap-2 mr-4">
        <span className={cn(
          "h-2 w-2 rounded-full animate-pulse",
          backendConnected
            ? "bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]"
            : "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]"
        )} />
        <span className={backendConnected ? "text-green-500/80" : "text-red-500/80"}>
          {backendConnected ? 'Connected' : 'Disconnected'}
        </span>
      </div>

      {/* Loading indicator */}
      {isLoading && (
        <div className="flex items-center gap-1 mr-4">
          <Loader2 className="h-3 w-3 animate-spin" />
          <span>Loading...</span>
        </div>
      )}

      {/* Project path */}
      <div className="flex-1">
        {projectPath ? (
          <span>Project: {projectPath}</span>
        ) : (
          <span className="italic">No project open</span>
        )}
      </div>
    </div>
  );
}

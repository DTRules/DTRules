import { cn } from '@/lib/utils';
import { Loader2, CheckCircle2, XCircle } from 'lucide-react';

interface StatusBarProps {
  projectPath: string | null;
  isLoading: boolean;
  backendConnected: boolean;
}

export function StatusBar({ projectPath, isLoading, backendConnected }: StatusBarProps) {
  return (
    <div className="h-6 border-t border-border flex items-center px-4 text-xs text-muted-foreground bg-muted/30">
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

      {/* Backend status */}
      <div className={cn(
        "flex items-center gap-1",
        backendConnected ? "text-green-500" : "text-red-500"
      )}>
        {backendConnected ? (
          <>
            <CheckCircle2 className="h-3 w-3" />
            <span>Backend Connected</span>
          </>
        ) : (
          <>
            <XCircle className="h-3 w-3" />
            <span>Backend Disconnected</span>
          </>
        )}
      </div>
    </div>
  );
}

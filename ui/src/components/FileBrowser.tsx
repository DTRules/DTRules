import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Folder, File, Home, ArrowUp } from 'lucide-react';
import { browseDirectory, type BrowseEntry } from '@/api/client';

/** 1234 → "1.2 KB" — enough precision to tell a real trace (MBs) from a
 *  header-only one (a few hundred bytes) at a glance. */
function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

interface FileBrowserProps {
  onSelect: (path: string) => void;
  /** When set, clicking a file selects it (directories still navigate). */
  selectFiles?: boolean;
  /** Directory to open at (e.g. where the last file was picked from);
   *  falls back to the server's default when omitted or invalid. */
  initialPath?: string;
}

export function FileBrowser({ onSelect, selectFiles, initialPath }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState('');
  const [entries, setEntries] = useState<BrowseEntry[]>([]);
  const [isProject, setIsProject] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pathInput, setPathInput] = useState('');

  const fetchDirectory = async (path: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await browseDirectory(path || undefined);
      if (data.success && data.currentPath) {
        setCurrentPath(data.currentPath);
        setPathInput(data.currentPath);
        setEntries(data.entries || []);
        setIsProject(data.isProject || false);
        return true;
      }
      setError(data.error || 'Failed to load directory');
      return false;
    } catch {
      setError('Failed to connect to server');
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    (async () => {
      // Open where the caller left off; fall back to the server default
      // when that directory is gone or inaccessible.
      if (initialPath && (await fetchDirectory(initialPath))) return;
      fetchDirectory('');
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleEntryClick = (entry: BrowseEntry) => {
    if (entry.isDir) {
      fetchDirectory(entry.path);
    } else if (selectFiles) {
      onSelect(entry.path);
    }
  };

  const handlePathSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!pathInput) return;
    // With a live filter narrowing to exactly one entry, Enter opens it.
    if (filter) {
      const matches = visibleEntries.filter((en) => en.name !== '..');
      if (matches.length === 1) {
        handleEntryClick(matches[0]);
        return;
      }
    }
    fetchDirectory(pathInput);
  };

  const handleSelectCurrent = () => {
    onSelect(currentPath);
  };

  // Live filter: typing beyond the current directory in the path bar
  // narrows the listing to matching names (case-insensitive substring).
  const prefix = currentPath.endsWith('/') ? currentPath : currentPath + '/';
  const filter =
    pathInput.startsWith(prefix) && !pathInput.slice(prefix.length).includes('/')
      ? pathInput.slice(prefix.length).toLowerCase()
      : '';
  const visibleEntries = filter
    ? entries.filter((e) => e.name === '..' || e.name.toLowerCase().includes(filter))
    : entries;

  const handleGoHome = () => {
    fetchDirectory('');
  };

  const handleGoUp = () => {
    const parentEntry = entries.find(e => e.name === '..');
    if (parentEntry) {
      fetchDirectory(parentEntry.path);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={handlePathSubmit} className="flex gap-2">
        <Button type="button" variant="outline" size="icon" onClick={handleGoHome} title="Home">
          <Home className="h-4 w-4" />
        </Button>
        <Button type="button" variant="outline" size="icon" onClick={handleGoUp} title="Up">
          <ArrowUp className="h-4 w-4" />
        </Button>
        <Input
          value={pathInput}
          onChange={(e) => setPathInput(e.target.value)}
          placeholder="Enter path..."
          className="flex-1 font-mono text-sm"
        />
        <Button type="submit" variant="outline">Go</Button>
      </form>

      {currentPath && (
        <div className="flex flex-wrap items-center text-xs font-mono text-muted-foreground px-1 -mt-1">
          {currentPath.split('/').filter(Boolean).map((seg, i, segs) => {
            const target = '/' + segs.slice(0, i + 1).join('/');
            const isCurrent = i === segs.length - 1;
            return (
              <span key={target} className="flex items-center">
                <span className="mx-0.5">/</span>
                <button
                  type="button"
                  className={isCurrent ? 'text-foreground' : 'hover:text-foreground hover:underline'}
                  title={isCurrent ? undefined : `Go to ${target}`}
                  onClick={() => !isCurrent && fetchDirectory(target)}
                >
                  {seg}
                </button>
              </span>
            );
          })}
        </div>
      )}

      {error && (
        <div className="text-sm text-destructive bg-destructive/10 p-2 rounded">{error}</div>
      )}

      <ScrollArea className="h-[300px] border rounded-md">
        <div className="p-2">
          {isLoading ? (
            <div className="text-center text-muted-foreground py-4">Loading...</div>
          ) : (
            <div className="space-y-1">
              {visibleEntries.map((entry) => (
                <div
                  key={entry.path}
                  className="flex items-center gap-2 p-2 rounded hover:bg-accent cursor-pointer"
                  onClick={() => handleEntryClick(entry)}
                >
                  {entry.isDir ? (
                    <Folder className="h-4 w-4 text-blue-500" />
                  ) : (
                    <File className="h-4 w-4 text-gray-500" />
                  )}
                  <span className="text-sm truncate">{entry.name}</span>
                  {!entry.isDir && entry.size !== undefined && (
                    <span className="ml-auto text-xs text-muted-foreground font-mono shrink-0">
                      {humanSize(entry.size)}
                    </span>
                  )}
                </div>
              ))}
            </div>
          )}
          {!isLoading && filter && visibleEntries.filter((e) => e.name !== '..').length === 0 && (
            <div className="text-center text-muted-foreground text-sm py-4">
              Nothing matches “{filter}”
            </div>
          )}
        </div>
      </ScrollArea>

      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">
          {selectFiles
            ? 'Click a file to select it'
            : isProject
              ? <span className="text-green-500">DTRules project detected</span>
              : 'Navigate to project folder'}
        </span>
        {!selectFiles && (
          <Button onClick={handleSelectCurrent} disabled={!currentPath}>
            Select This Folder
          </Button>
        )}
      </div>
    </div>
  );
}

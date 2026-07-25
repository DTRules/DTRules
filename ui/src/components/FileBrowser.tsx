import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Folder, File, Home, ArrowUp } from 'lucide-react';
import { browseDirectory, type BrowseEntry } from '@/api/client';

interface FileBrowserProps {
  onSelect: (path: string) => void;
  /** When set, clicking a file selects it (directories still navigate). */
  selectFiles?: boolean;
}

export function FileBrowser({ onSelect, selectFiles }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState('');
  const [entries, setEntries] = useState<BrowseEntry[]>([]);
  const [isProject, setIsProject] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pathInput, setPathInput] = useState('');

  const fetchDirectory = async (path: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await browseDirectory(path || undefined);
      if (data.success && data.currentPath) {
        setCurrentPath(data.currentPath);
        setPathInput(data.currentPath);
        setEntries(data.entries || []);
        setIsProject(data.isProject || false);
      } else {
        setError(data.error || 'Failed to load directory');
      }
    } catch {
      setError('Failed to connect to server');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchDirectory('');
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
    if (pathInput) {
      fetchDirectory(pathInput);
    }
  };

  const handleSelectCurrent = () => {
    onSelect(currentPath);
  };

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

      {error && (
        <div className="text-sm text-destructive bg-destructive/10 p-2 rounded">{error}</div>
      )}

      <ScrollArea className="h-[300px] border rounded-md">
        <div className="p-2">
          {isLoading ? (
            <div className="text-center text-muted-foreground py-4">Loading...</div>
          ) : (
            <div className="space-y-1">
              {entries.map((entry) => (
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
                </div>
              ))}
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

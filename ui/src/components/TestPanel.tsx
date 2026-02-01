import { useState } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import Editor from '@monaco-editor/react';
import { executeRules } from '@/api/client';
import { Play, CheckCircle2, XCircle, AlertCircle } from 'lucide-react';
import type { TraceEntry } from '@/types/dtrules';

const SAMPLE_TEST_DATA = `{
  "job": {
    "program": "CHIP",
    "currentdate": "2024-01-15",
    "effectivedate": "2024-02-01"
  },
  "case": {
    "county_cd": "AA"
  },
  "clients": [
    {
      "id": 1,
      "age": 12,
      "applying": true,
      "validatedCitizenship": true,
      "uninsured": true,
      "pregnant": false
    }
  ]
}`;

export function TestPanel() {
  const { decisionTables } = useProjectStore();
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [testData, setTestData] = useState<string>(SAMPLE_TEST_DATA);
  const [enableTrace, setEnableTrace] = useState(true);
  const [isExecuting, setIsExecuting] = useState(false);
  const [result, setResult] = useState<Record<string, unknown> | null>(null);
  const [trace, setTrace] = useState<TraceEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  const handleExecute = async () => {
    if (!selectedTable) return;

    setIsExecuting(true);
    setError(null);
    setResult(null);
    setTrace([]);

    try {
      const data = JSON.parse(testData);
      const response = await executeRules(selectedTable, data, enableTrace);

      if (response.success) {
        setResult(response.result || null);
        setTrace(response.trace || []);
      } else {
        setError(response.error || 'Execution failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to execute');
    } finally {
      setIsExecuting(false);
    }
  };

  return (
    <div className="h-full flex" data-tutorial="test-panel">
      {/* Test data input */}
      <div className="flex-1 flex flex-col border-r border-border" data-tutorial="test-input">
        <div className="p-4 border-b border-border flex items-center gap-4">
          <div className="grid gap-1 flex-1">
            <Label className="text-xs text-muted-foreground">Decision Table</Label>
            <Select value={selectedTable} onValueChange={setSelectedTable}>
              <SelectTrigger>
                <SelectValue placeholder="Select a table to execute" />
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
          <div className="flex items-center gap-2 pt-5">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={enableTrace}
                onChange={(e) => setEnableTrace(e.target.checked)}
                className="rounded"
              />
              Enable Trace
            </label>
            <Button onClick={handleExecute} disabled={!selectedTable || isExecuting}>
              <Play className="h-4 w-4 mr-2" />
              Execute
            </Button>
          </div>
        </div>

        <div className="flex-1 flex flex-col">
          <div className="p-2 text-sm font-medium bg-muted border-b border-border">
            Test Data (JSON)
          </div>
          <div className="flex-1">
            <Editor
              height="100%"
              defaultLanguage="json"
              value={testData}
              onChange={(v) => setTestData(v || '')}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                scrollBeyondLastLine: false,
                automaticLayout: true,
              }}
            />
          </div>
        </div>
      </div>

      {/* Results panel */}
      <div className="w-96 flex flex-col" data-tutorial="test-results">
        <div className="p-4 border-b border-border">
          <h3 className="font-medium">Execution Results</h3>
        </div>

        <ScrollArea className="flex-1">
          <div className="p-4 space-y-4">
            {error && (
              <div className="p-4 border border-red-500/50 rounded-md bg-red-500/10">
                <div className="flex items-center gap-2 text-red-500 mb-2">
                  <XCircle className="h-4 w-4" />
                  <span className="font-medium">Error</span>
                </div>
                <p className="text-sm text-red-400">{error}</p>
              </div>
            )}

            {result && (
              <div className="p-4 border border-green-500/50 rounded-md bg-green-500/10">
                <div className="flex items-center gap-2 text-green-500 mb-2">
                  <CheckCircle2 className="h-4 w-4" />
                  <span className="font-medium">Result</span>
                </div>
                <pre className="text-xs font-mono overflow-auto max-h-64">
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
            )}

            {trace.length > 0 && (
              <div className="space-y-2">
                <h4 className="font-medium text-sm">Execution Trace</h4>
                {trace.map((entry, i) => (
                  <div
                    key={i}
                    className="p-2 border border-border rounded text-xs font-mono"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-muted-foreground">
                        {entry.tableName}
                      </span>
                      <span className="text-muted-foreground">
                        Column {entry.column}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span
                        className={
                          entry.type === 'condition'
                            ? 'text-blue-400'
                            : 'text-green-400'
                        }
                      >
                        {entry.type === 'condition' ? 'COND' : 'ACT'} #{entry.index}
                      </span>
                      {entry.result !== undefined && (
                        <span
                          className={
                            entry.result ? 'text-green-400' : 'text-red-400'
                          }
                        >
                          {entry.result ? 'TRUE' : 'FALSE'}
                        </span>
                      )}
                      {entry.value && (
                        <span className="text-muted-foreground">
                          = {entry.value}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {!error && !result && trace.length === 0 && (
              <div className="text-center text-muted-foreground py-8">
                <AlertCircle className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>Select a table and click Execute to run rules.</p>
                <p className="text-xs mt-2">
                  Note: Full execution requires the Java DTRules engine.
                </p>
              </div>
            )}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}

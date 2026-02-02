import { useState, useEffect } from 'react';
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
import { Play, CheckCircle2, XCircle } from 'lucide-react';
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

  // Auto-select first table when tables load and nothing is selected
  useEffect(() => {
    if (!selectedTable && decisionTables.length > 0) {
      setSelectedTable(decisionTables[0].name);
    }
  }, [decisionTables, selectedTable]);

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
      <div className="w-1/2 flex flex-col border-r border-border/50" data-tutorial="test-input">
        <div className="p-4 border-b border-border/50 bg-gradient-to-r from-muted/30 via-transparent to-muted/30 flex items-center gap-4">
          <div className="grid gap-1 flex-1">
            <Label className="text-xs text-muted-foreground">Decision Table</Label>
            <Select value={selectedTable} onValueChange={setSelectedTable}>
              <SelectTrigger>
                <SelectValue placeholder="Select a table to execute" />
              </SelectTrigger>
              <SelectContent>
                {decisionTables
                  .filter((table) => table.name && table.name.trim() !== '')
                  .map((table) => (
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
            <Button onClick={handleExecute} disabled={!selectedTable || isExecuting} className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0 disabled:opacity-50">
              <Play className="h-4 w-4 mr-2" />
              Execute
            </Button>
          </div>
        </div>

        <div className="flex-1 flex flex-col">
          <div className="p-3 text-sm font-semibold bg-gradient-to-r from-blue-500/10 to-transparent border-b border-border/50 text-foreground/80">
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

      {/* Results panel - now takes half the space */}
      <div className="w-1/2 flex flex-col" data-tutorial="test-results">
        <div className="p-4 border-b border-border/50 bg-gradient-to-r from-purple-500/10 to-transparent">
          <h3 className="font-semibold text-foreground/80">Execution Results</h3>
        </div>

        <ScrollArea className="flex-1">
          <div className="p-4 space-y-4">
            {error && (
              <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 backdrop-blur">
                <div className="flex items-center gap-2 text-red-500 mb-2">
                  <XCircle className="h-4 w-4" />
                  <span className="font-medium">Error</span>
                </div>
                <p className="text-sm text-red-400">{error}</p>
              </div>
            )}

            {result && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-green-500">
                  <CheckCircle2 className="h-4 w-4" />
                  <span className="font-medium">Execution Successful</span>
                </div>
                <div className="text-xs text-muted-foreground mb-2">
                  Entity state after execution:
                </div>
                <div className="h-64 border border-border/50 rounded-lg overflow-hidden">
                  <Editor
                    height="100%"
                    defaultLanguage="json"
                    value={JSON.stringify(result, null, 2)}
                    theme="vs-dark"
                    options={{
                      readOnly: true,
                      minimap: { enabled: false },
                      fontSize: 12,
                      scrollBeyondLastLine: false,
                      automaticLayout: true,
                      lineNumbers: 'off',
                      folding: true,
                      wordWrap: 'on',
                    }}
                  />
                </div>
              </div>
            )}

            {trace.length > 0 && (
              <div className="space-y-2">
                <h4 className="font-medium text-sm">Execution Trace</h4>
                {trace.map((entry, i) => (
                  <div
                    key={i}
                    className="p-3 border border-border/50 rounded-lg bg-muted/20 hover:bg-muted/30 transition-colors text-xs font-mono"
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
              <div className="text-center py-8">
                <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-purple-500/20 to-blue-500/20 flex items-center justify-center">
                  <Play className="h-8 w-8 text-purple-400/60" />
                </div>
                <p className="text-muted-foreground">Select a table and click Execute to run rules.</p>
                <p className="text-xs text-muted-foreground/60 mt-2">
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

import { useState, useEffect, useMemo, useCallback } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { AgGridReact } from 'ag-grid-react';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import { Save } from 'lucide-react';
import type { ColDef, CellValueChangedEvent } from 'ag-grid-community';
import type { DecisionTable } from '@/types/dtrules';

// Color coding for decision table cells matching the tutorial
const getCellStyle = (params: { value: unknown }): Record<string, string> => {
  const value = String(params.value ?? '').toUpperCase().trim();

  switch (value) {
    case 'Y':
      // Green - condition must be true
      return {
        backgroundColor: 'rgba(34, 197, 94, 0.15)',
        color: 'rgb(34, 197, 94)',
        fontWeight: 'bold',
        textAlign: 'center',
      };
    case 'N':
      // Red - condition must be false
      return {
        backgroundColor: 'rgba(239, 68, 68, 0.15)',
        color: 'rgb(239, 68, 68)',
        fontWeight: 'bold',
        textAlign: 'center',
      };
    case 'X':
      // Blue - execute this action
      return {
        backgroundColor: 'rgba(59, 130, 246, 0.15)',
        color: 'rgb(59, 130, 246)',
        fontWeight: 'bold',
        textAlign: 'center',
      };
    case '-':
    case '*':
      // Muted - don't care
      return {
        backgroundColor: 'rgba(128, 128, 128, 0.1)',
        color: 'rgba(156, 163, 175, 0.8)',
        textAlign: 'center',
      };
    default:
      return { textAlign: 'center' };
  }
};

export function DTEditor() {
  const { decisionTables, currentTable, selectTable, updateTable } = useProjectStore();
  const [editedTable, setEditedTable] = useState<DecisionTable | null>(null);

  useEffect(() => {
    if (currentTable) {
      setEditedTable({ ...currentTable });
    } else {
      setEditedTable(null);
    }
  }, [currentTable]);

  const handleSave = async () => {
    if (!editedTable) return;
    await updateTable(editedTable);
  };

  // Generate column definitions for the grid
  const columnDefs = useMemo<ColDef[]>(() => {
    if (!editedTable) return [];

    const cols: ColDef[] = [
      {
        headerName: '#',
        field: 'number',
        width: 50,
        pinned: 'left',
        editable: false,
      },
      {
        headerName: 'Description',
        field: 'description',
        width: 300,
        pinned: 'left',
        editable: true,
      },
    ];

    // Add column for each decision column
    for (let i = 1; i <= editedTable.columnCount; i++) {
      cols.push({
        headerName: `C${i}`,
        field: `col_${i}`,
        width: 60,
        editable: true,
        cellEditor: 'agSelectCellEditor',
        cellEditorParams: {
          values: ['Y', 'N', '-', '*', 'X', ''],
        },
        cellStyle: getCellStyle,
      });
    }

    return cols;
  }, [editedTable]);

  // Generate row data for conditions
  const conditionRowData = useMemo(() => {
    if (!editedTable || !editedTable.conditions) return [];

    return editedTable.conditions.map((cond) => {
      const row: Record<string, string | number> = {
        type: 'condition',
        number: cond.number,
        description: cond.description,
        postfix: cond.postfix,
        comment: cond.comment,
      };

      for (let i = 1; i <= editedTable.columnCount; i++) {
        // Handle both string and number keys from API
        const cols = cond.columns as Record<string, string> | undefined;
        row[`col_${i}`] = cols?.[i] || cols?.[String(i)] || '-';
      }

      return row;
    });
  }, [editedTable]);

  // Generate row data for actions
  const actionRowData = useMemo(() => {
    if (!editedTable || !editedTable.actions) return [];

    return editedTable.actions.map((action) => {
      const row: Record<string, string | number> = {
        type: 'action',
        number: action.number,
        description: action.description,
        postfix: action.postfix,
        comment: action.comment,
      };

      for (let i = 1; i <= editedTable.columnCount; i++) {
        // Handle both string and number keys from API
        const cols = action.columns as Record<string, string> | undefined;
        row[`col_${i}`] = cols?.[i] || cols?.[String(i)] || '';
      }

      return row;
    });
  }, [editedTable]);

  const handleConditionCellValueChanged = useCallback((event: CellValueChangedEvent) => {
    if (!editedTable) return;

    const rowIndex = event.rowIndex;
    if (rowIndex === null) return;

    const field = event.colDef.field;
    if (!field) return;

    const newConditions = [...editedTable.conditions];

    if (field === 'description') {
      newConditions[rowIndex] = {
        ...newConditions[rowIndex],
        description: event.newValue,
      };
    } else if (field.startsWith('col_')) {
      const colNum = parseInt(field.replace('col_', ''));
      newConditions[rowIndex] = {
        ...newConditions[rowIndex],
        columns: {
          ...newConditions[rowIndex].columns,
          [colNum]: event.newValue,
        },
      };
    }

    setEditedTable({ ...editedTable, conditions: newConditions });
  }, [editedTable]);

  const handleActionCellValueChanged = useCallback((event: CellValueChangedEvent) => {
    if (!editedTable) return;

    const rowIndex = event.rowIndex;
    if (rowIndex === null) return;

    const field = event.colDef.field;
    if (!field) return;

    const newActions = [...editedTable.actions];

    if (field === 'description') {
      newActions[rowIndex] = {
        ...newActions[rowIndex],
        description: event.newValue,
      };
    } else if (field.startsWith('col_')) {
      const colNum = parseInt(field.replace('col_', ''));
      newActions[rowIndex] = {
        ...newActions[rowIndex],
        columns: {
          ...newActions[rowIndex].columns,
          [colNum]: event.newValue,
        },
      };
    }

    setEditedTable({ ...editedTable, actions: newActions });
  }, [editedTable]);

  if (!decisionTables.length) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground" data-tutorial="dt-editor">
        <p>No decision tables loaded. Open a project to see tables.</p>
      </div>
    );
  }

  return (
    <div className="h-full flex" data-tutorial="dt-editor">
      {/* Table list sidebar */}
      <div className="w-56 border-r border-border flex flex-col" data-tutorial="dt-list">
        <div className="p-2 border-b border-border">
          <span className="text-sm font-medium">Decision Tables</span>
        </div>
        <ScrollArea className="flex-1">
          {decisionTables.map((table) => (
            <div
              key={table.name}
              className={cn(
                "px-3 py-2 cursor-pointer hover:bg-accent",
                currentTable?.tableName === table.name && "bg-accent"
              )}
              onClick={() => selectTable(table.name)}
            >
              <div className="text-sm font-medium">{table.name}</div>
              <div className="text-xs text-muted-foreground">
                {table.conditionCount} conditions, {table.actionCount} actions
              </div>
            </div>
          ))}
        </ScrollArea>
      </div>

      {/* Table editor */}
      <div className="flex-1 flex flex-col" data-tutorial="dt-grid">
        {editedTable ? (
          <>
            {/* Table header */}
            <div className="p-4 border-b border-border flex items-center gap-4">
              <div className="flex-1 flex items-center gap-4">
                <div className="grid gap-1">
                  <Label className="text-xs text-muted-foreground">Table Name</Label>
                  <div className="text-sm font-medium">{editedTable.tableName}</div>
                </div>
                <div className="grid gap-1">
                  <Label className="text-xs text-muted-foreground">Type</Label>
                  <Select
                    value={editedTable.type}
                    onValueChange={(v) => setEditedTable({ ...editedTable, type: v as 'FIRST' | 'ALL' })}
                  >
                    <SelectTrigger className="w-24 h-8">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="FIRST">FIRST</SelectItem>
                      <SelectItem value="ALL">ALL</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1">
                  <Label className="text-xs text-muted-foreground">Columns</Label>
                  <div className="text-sm">{editedTable.columnCount}</div>
                </div>
              </div>
              <Button size="sm" onClick={handleSave}>
                <Save className="h-4 w-4 mr-2" />
                Save
              </Button>
            </div>

            {/* Decision table grids */}
            <Tabs defaultValue="matrix" className="flex-1 flex flex-col">
              <div className="border-b border-border px-4">
                <TabsList className="h-10">
                  <TabsTrigger value="matrix">Decision Matrix</TabsTrigger>
                  <TabsTrigger value="contexts">Contexts</TabsTrigger>
                  <TabsTrigger value="policies">Policy Statements</TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="matrix" className="flex-1 m-0 overflow-hidden">
                <div className="h-full flex flex-col">
                  {/* Conditions grid */}
                  <div className="flex-1 min-h-0">
                    <div className="p-2 text-sm font-medium bg-muted">Conditions</div>
                    <div className="ag-theme-alpine-dark h-[calc(100%-2rem)]">
                      <AgGridReact
                        columnDefs={columnDefs}
                        rowData={conditionRowData}
                        onCellValueChanged={handleConditionCellValueChanged}
                        defaultColDef={{
                          resizable: true,
                          sortable: false,
                        }}
                      />
                    </div>
                  </div>

                  {/* Actions grid */}
                  <div className="flex-1 min-h-0 border-t border-border">
                    <div className="p-2 text-sm font-medium bg-muted">Actions</div>
                    <div className="ag-theme-alpine-dark h-[calc(100%-2rem)]">
                      <AgGridReact
                        columnDefs={columnDefs}
                        rowData={actionRowData}
                        onCellValueChanged={handleActionCellValueChanged}
                        defaultColDef={{
                          resizable: true,
                          sortable: false,
                        }}
                      />
                    </div>
                  </div>
                </div>
              </TabsContent>

              <TabsContent value="contexts" className="flex-1 m-0 p-4 overflow-auto">
                <div className="space-y-4">
                  <h3 className="text-sm font-medium">Iteration Contexts</h3>
                  {!editedTable.contexts || editedTable.contexts.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No contexts defined.</p>
                  ) : (
                    editedTable.contexts.map((ctx, index) => (
                      <div key={index} className="p-4 border border-border rounded-md space-y-2">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">Context {ctx.number}</span>
                          <span className="text-sm text-muted-foreground">{ctx.description}</span>
                        </div>
                        <div className="font-mono text-xs bg-muted p-2 rounded">
                          {ctx.postfix}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </TabsContent>

              <TabsContent value="policies" className="flex-1 m-0 p-4 overflow-auto">
                <div className="space-y-4">
                  <h3 className="text-sm font-medium">Policy Statements</h3>
                  {!editedTable.policyStatements || editedTable.policyStatements.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No policy statements defined.</p>
                  ) : (
                    editedTable.policyStatements.map((ps, index) => (
                      <div key={index} className="p-4 border border-border rounded-md space-y-2">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">Column {ps.column}</span>
                        </div>
                        <p className="text-sm">{ps.description}</p>
                      </div>
                    ))
                  )}
                </div>
              </TabsContent>
            </Tabs>
          </>
        ) : (
          <div className="h-full flex items-center justify-center text-muted-foreground">
            <p>Select a decision table to edit</p>
          </div>
        )}
      </div>
    </div>
  );
}

import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import type { ReactNode } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { AgGridReact } from 'ag-grid-react';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { ArrowLeft, ArrowRight, Code2, Eye, Pencil, Save, Table2 } from 'lucide-react';
import type {
  ColDef,
  CellValueChangedEvent,
  ColumnResizedEvent,
  ICellRendererParams,
} from 'ag-grid-community';
import type { DecisionTable } from '@/types/dtrules';

/**
 * Matches cross-table references in either surface:
 * - EL DSL:  `perform <TableName>`
 * - postfix: `/<TableName> performtable`
 * The referenced name lands in capture group 1 or 2.
 */
const PERFORM_RE = /\bperform\s+([A-Za-z_][A-Za-z0-9_]*)|\/([A-Za-z_][A-Za-z0-9_]*)\s+performtable\b/gi;

/** Grid context shared with cell renderers. EL is case-insensitive (what
 *  the user types is preserved for display), so table-name resolution maps
 *  lowercased names to their canonical form. */
interface DSLCellContext {
  tableNames?: Map<string, string>;
  navigate?: (name: string) => void;
}

/**
 * DSL cell renderer: decision-table calls render as colored links that jump
 * to the referenced table.
 */
function DSLCell(props: ICellRendererParams) {
  const { tableNames, navigate } = (props.context || {}) as DSLCellContext;
  const text = String(props.value ?? '');

  if (!tableNames || !navigate || props.data?.type === 'header') {
    return <span className="whitespace-pre-wrap">{text}</span>;
  }

  const parts: ReactNode[] = [];
  let last = 0;
  for (const match of text.matchAll(PERFORM_RE)) {
    const name = match[1] || match[2];
    const canonical = tableNames.get(name.toLowerCase());
    if (!canonical) continue;
    const start = (match.index ?? 0) + match[0].indexOf(name);
    parts.push(text.slice(last, start));
    parts.push(
      <button
        key={start}
        className="text-blue-400 underline decoration-blue-400/40 underline-offset-2 hover:text-blue-300"
        title={`Go to ${canonical}`}
        onClick={(e) => {
          e.stopPropagation();
          navigate(canonical);
        }}
      >
        {name}
      </button>
    );
    last = start + name.length;
  }
  parts.push(text.slice(last));
  return <span className="whitespace-pre-wrap">{parts}</span>;
}

// User-adjusted column widths, persisted across tables and sessions.
// Rule columns share one width so the matrix stays uniform.
const COL_WIDTHS_KEY = 'dtrules.dtEditorColWidths';

interface ColWidths {
  number: number;
  comment: number;
  dsl: number;
  rule: number;
}

const DEFAULT_COL_WIDTHS: ColWidths = { number: 40, comment: 220, dsl: 320, rule: 30 };

function loadColWidths(): ColWidths {
  try {
    const saved = JSON.parse(localStorage.getItem(COL_WIDTHS_KEY) || '{}');
    return { ...DEFAULT_COL_WIDTHS, ...saved };
  } catch {
    return { ...DEFAULT_COL_WIDTHS };
  }
}

// Rule-cell color coding: bright letters only — no background fills, so the
// column separators stay the dominant vertical structure.
// justifyContent centers within the flex-layout cells of the dense theme.
const getCellStyle = (params: { value: unknown }): Record<string, string> => {
  const value = String(params.value ?? '').toUpperCase().trim();
  const centered = { justifyContent: 'center', textAlign: 'center' };

  switch (value) {
    case 'Y':
      // Green - condition must be true
      return { ...centered, color: 'rgb(34, 197, 94)', fontWeight: 'bold' };
    case 'N':
      // Red - condition must be false
      return { ...centered, color: 'rgb(239, 68, 68)', fontWeight: 'bold' };
    case 'X':
      // Blue - execute this action
      return { ...centered, color: 'rgb(59, 130, 246)', fontWeight: 'bold' };
    case '-':
    case '*':
      // Muted - don't care
      return { ...centered, color: 'rgba(156, 163, 175, 0.8)' };
    default:
      return { ...centered };
  }
};

export function DTEditor() {
  const { decisionTables, currentTable, selectTable, updateTable, readOnly } = useProjectStore();
  const [editedTable, setEditedTable] = useState<DecisionTable | null>(null);
  // When on, the DSL column shows the compiled postfix instead (read-only —
  // postfix is a build artifact and is never hand-edited).
  const [showPostfix, setShowPostfix] = useState(false);
  // Viewing mode by default; edit mode unlocks inline editing.
  const [editMode, setEditMode] = useState(false);

  // A read-only backend permits no edit mode at all.
  useEffect(() => {
    if (readOnly) setEditMode(false);
  }, [readOnly]);
  const [colWidths, setColWidths] = useState<ColWidths>(loadColWidths);

  // Table-level navigation history for perform-link jumps (browser-style).
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const navigatingRef = useRef(false);

  useEffect(() => {
    const name = currentTable?.tableName;
    if (!name) return;
    if (navigatingRef.current) {
      navigatingRef.current = false;
      return;
    }
    setHistory((h) => {
      if (h[historyIndex] === name) return h;
      const next = h.slice(0, historyIndex + 1);
      next.push(name);
      setHistoryIndex(next.length - 1);
      return next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTable?.tableName]);

  const navigateToTable = useCallback(
    (name: string) => {
      if (name !== currentTable?.tableName) selectTable(name);
    },
    [currentTable?.tableName, selectTable]
  );

  const goBack = () => {
    if (historyIndex <= 0) return;
    navigatingRef.current = true;
    setHistoryIndex(historyIndex - 1);
    selectTable(history[historyIndex - 1]);
  };

  const goForward = () => {
    if (historyIndex >= history.length - 1) return;
    navigatingRef.current = true;
    setHistoryIndex(historyIndex + 1);
    selectTable(history[historyIndex + 1]);
  };

  // Lowercased name -> canonical name: EL references are case-insensitive.
  const tableNames = useMemo(
    () => new Map(decisionTables.map((t) => [t.name.toLowerCase(), t.name])),
    [decisionTables]
  );

  // Record drag-resizes so widths carry over to every table (and session).
  const handleColumnResized = useCallback((event: ColumnResizedEvent) => {
    if (!event.finished || !event.source?.startsWith('ui')) return;
    const columns = event.columns ?? (event.column ? [event.column] : []);
    if (!columns.length) return;

    setColWidths((prev) => {
      const next = { ...prev };
      for (const col of columns) {
        const id = col.getColId();
        const width = Math.round(col.getActualWidth());
        if (id === 'number') next.number = width;
        else if (id === 'comment') next.comment = width;
        else if (id === 'description' || id === 'postfix') next.dsl = width;
        else if (id.startsWith('col_')) next.rule = width;
      }
      try {
        localStorage.setItem(COL_WIDTHS_KEY, JSON.stringify(next));
      } catch {
        // localStorage unavailable — widths just won't persist
      }
      return next;
    });
  }, []);

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

  // Commit a table-number change: the server shifts colliding/following
  // tables down by 100 and the reloaded list reflects the new order.
  const handleRenumber = async () => {
    if (!editedTable || !currentTable) return;
    if (editedTable.tableNumber === currentTable.tableNumber) return;
    await updateTable(editedTable);
  };

  // Column definitions for the table grid. The whole decision table —
  // contexts, initial actions, conditions, actions, policy statements — is
  // one grid with section header rows, mirroring the Excel sheet, so the
  // rule columns stay vertically aligned top to bottom.
  const columnDefs = useMemo<ColDef[]>(() => {
    if (!editedTable) return [];

    const notSeparator = (params: { data?: { type?: string } }) =>
      editMode &&
      (params.data?.type === 'condition' || params.data?.type === 'action');

    const totalCols = 3 + editedTable.columnCount;

    const cols: ColDef[] = [
      {
        headerName: '#',
        field: 'number',
        width: colWidths.number,
        editable: false,
        cellDataType: 'text',
        // Section header rows put their label here and span the full grid
        // width so titles sit at the left edge, like the sheet's banded rows.
        colSpan: (params) => (params.data?.type === 'header' ? totalCols : 1),
        cellStyle: (params) => {
          const style: Record<string, string> = {};
          if (params.data?.type === 'header') {
            style.fontWeight = '600';
            style.letterSpacing = '0.05em';
            style.justifyContent = 'flex-start';
          } else {
            style.justifyContent = 'center';
            style.color = 'rgba(156, 163, 175, 0.7)';
          }
          return style;
        },
      },
      {
        headerName: 'Comments',
        field: 'comment',
        width: colWidths.comment,
        editable: notSeparator,
        wrapText: true,
        autoHeight: true,
        cellStyle: {
          fontStyle: 'italic',
          color: 'rgba(156, 163, 175, 0.9)',
          lineHeight: '1.3',
          wordBreak: 'break-word',
        },
      },
      {
        headerName: showPostfix ? 'Postfix (read-only)' : 'DSL',
        field: showPostfix ? 'postfix' : 'description',
        width: colWidths.dsl,
        editable: (params) => !showPostfix && notSeparator(params),
        cellRenderer: DSLCell,
        wrapText: true,
        autoHeight: true,
        cellStyle: (params) => {
          const style: Record<string, string> = {};
          if (params.data?.type === 'header') {
            style.fontWeight = '600';
            style.letterSpacing = '0.05em';
            return style;
          }
          style.fontFamily = 'monospace';
          style.fontSize = '12px';
          style.lineHeight = '1.4';
          style.wordBreak = 'break-word';
          if (showPostfix) style.color = 'rgba(156, 163, 175, 0.9)';
          return style;
        },
      },
    ];

    for (let i = 1; i <= editedTable.columnCount; i++) {
      cols.push({
        headerName: `${i}`,
        field: `col_${i}`,
        width: colWidths.rule,
        headerClass: 'rule-col-header',
        editable: notSeparator,
        cellEditor: 'agSelectCellEditor',
        // Conditions cycle Y/N/-/*; actions are X or blank
        cellEditorParams: (params: { data?: { type?: string } }) => ({
          values: params.data?.type === 'action' ? ['X', ''] : ['Y', 'N', '-', '*'],
        }),
        cellStyle: getCellStyle,
      });
    }

    return cols;
  }, [editedTable, showPostfix, colWidths, editMode]);

  // The full decision table as one row set, in the Excel sheet's order.
  // Section header rows separate CONTEXTS / INITIAL ACTIONS / CONDITIONS /
  // ACTIONS / POLICY STATEMENTS. Condition and action rows carry their index
  // within their own list so edits route back correctly.
  const matrixRowData = useMemo(() => {
    if (!editedTable) return [];

    const rows: Record<string, string | number>[] = [];
    // The label lives in `number` — the # column spans the full grid width
    // for header rows, so the title renders from the left edge.
    const header = (label: string) =>
      rows.push({ type: 'header', number: label, comment: '', description: label, postfix: label });

    header('CONTEXTS');
    (editedTable.contexts || []).forEach((ctx, i) => {
      rows.push({
        type: 'context',
        number: ctx.number || i + 1,
        comment: ctx.comment,
        description: ctx.description,
        postfix: ctx.postfix,
      });
    });

    header('INITIAL ACTIONS');
    (editedTable.initialActions || '')
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line !== '')
      .forEach((line, i) => {
        rows.push({ type: 'initialAction', number: i + 1, comment: '', description: line, postfix: line });
      });

    header('CONDITIONS');
    (editedTable.conditions || []).forEach((cond, idx) => {
      const row: Record<string, string | number> = {
        type: 'condition',
        idx,
        number: cond.number,
        description: cond.description,
        postfix: cond.postfix,
        comment: cond.comment,
      };
      for (let i = 1; i <= editedTable.columnCount; i++) {
        const cols = cond.columns as Record<string, string> | undefined;
        row[`col_${i}`] = cols?.[i] || cols?.[String(i)] || '-';
      }
      rows.push(row);
    });

    header('ACTIONS');
    (editedTable.actions || []).forEach((action, idx) => {
      const row: Record<string, string | number> = {
        type: 'action',
        idx,
        number: action.number,
        description: action.description,
        postfix: action.postfix,
        comment: action.comment,
      };
      for (let i = 1; i <= editedTable.columnCount; i++) {
        const cols = action.columns as Record<string, string> | undefined;
        row[`col_${i}`] = cols?.[i] || cols?.[String(i)] || '';
      }
      rows.push(row);
    });

    header('POLICY STATEMENTS');
    (editedTable.policyStatements || []).forEach((ps) => {
      rows.push({
        type: 'policy',
        number: ps.column,
        comment: '',
        description: ps.description,
        postfix: ps.postfix || ps.description,
      });
    });

    return rows;
  }, [editedTable]);

  // Section header row tints, echoing the Excel sheet's banded headers
  const headerRowStyle = (label: unknown): Record<string, string> => {
    switch (label) {
      case 'CONTEXTS':
        return { background: 'rgba(168, 85, 247, 0.10)' };
      case 'INITIAL ACTIONS':
        return { background: 'rgba(245, 158, 11, 0.08)' };
      case 'CONDITIONS':
        return { background: 'rgba(59, 130, 246, 0.10)' };
      case 'ACTIONS':
        return { background: 'rgba(34, 197, 94, 0.08)' };
      default:
        return { background: 'rgba(148, 163, 184, 0.08)' };
    }
  };

  const handleCellValueChanged = useCallback((event: CellValueChangedEvent) => {
    if (!editedTable) return;

    const field = event.colDef.field;
    const rowType = event.data?.type;
    const idx = event.data?.idx;
    if (!field || typeof idx !== 'number' || (rowType !== 'condition' && rowType !== 'action')) return;

    const isCondition = rowType === 'condition';
    const list = isCondition ? [...editedTable.conditions] : [...editedTable.actions];

    if (field === 'description' || field === 'comment') {
      list[idx] = { ...list[idx], [field]: event.newValue };
    } else if (field.startsWith('col_')) {
      const colNum = parseInt(field.replace('col_', ''));
      list[idx] = {
        ...list[idx],
        columns: { ...list[idx].columns, [colNum]: event.newValue },
      };
    }

    setEditedTable(
      isCondition
        ? { ...editedTable, conditions: list as DecisionTable['conditions'] }
        : { ...editedTable, actions: list as DecisionTable['actions'] }
    );
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
      {/* Table editor; the table list lives in the Project Explorer */}
      <div className="flex-1 flex flex-col" data-tutorial="dt-grid">
        {editedTable ? (
          <>
            {/* Table header */}
            <div className="px-4 py-3 border-b border-border/50 bg-gradient-to-r from-muted/30 via-transparent to-muted/30 flex items-start gap-4">
              {/* Table navigation: back/forward across perform-link jumps */}
              <div className="flex items-center gap-0.5 pt-0.5">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={goBack}
                  disabled={historyIndex <= 0}
                  title="Back to previous table"
                >
                  <ArrowLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={goForward}
                  disabled={historyIndex >= history.length - 1}
                  title="Forward"
                >
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  {editMode ? (
                    <input
                      value={editedTable.tableNumber}
                      onChange={(e) => setEditedTable({ ...editedTable, tableNumber: e.target.value })}
                      onKeyDown={(e) => e.key === 'Enter' && handleRenumber()}
                      onBlur={handleRenumber}
                      className="h-7 w-20 px-2 rounded-md border border-input bg-transparent font-mono text-sm"
                      title="Table number — saving a colliding number shifts following tables down"
                    />
                  ) : (
                    <span className="text-xl font-mono text-muted-foreground">
                      {editedTable.tableNumber}
                    </span>
                  )}
                  <h1 className="text-xl font-bold">{editedTable.tableName}</h1>
                  <span
                    className="h-7 px-2.5 inline-flex items-center rounded-md text-sm font-semibold text-emerald-400 border border-emerald-400/40 bg-emerald-400/5"
                    title="Table type: FIRST stops at the first matching rule; ALL executes every matching rule"
                  >
                    {editedTable.type}
                  </span>
                </div>
                {editedTable.comments && editedTable.comments.trim() !== '' && (
                  <p className="mt-1 text-sm text-muted-foreground whitespace-pre-wrap">
                    {editedTable.comments}
                  </p>
                )}
              </div>
              {/* Mode indicator + toggle: viewing (default) vs editing.
                  On a read-only server there is no edit mode to offer. */}
              {!readOnly && (
                <Button
                  size="sm"
                  variant={editMode ? 'secondary' : 'outline'}
                  onClick={() => setEditMode(!editMode)}
                  className={cn(editMode && 'text-amber-400')}
                  title={editMode ? 'Editing — click to return to viewing' : 'Viewing — click to edit'}
                >
                  {editMode ? <Pencil className="h-4 w-4 mr-2" /> : <Eye className="h-4 w-4 mr-2" />}
                  {editMode ? 'Editing' : 'Viewing'}
                </Button>
              )}
              <Button
                size="sm"
                variant={showPostfix ? 'secondary' : 'outline'}
                onClick={() => setShowPostfix(!showPostfix)}
                title={showPostfix ? 'Show authored DSL' : 'Show compiled postfix'}
              >
                <Code2 className="h-4 w-4 mr-2" />
                Postfix
              </Button>
              {!readOnly && (
                <Button
                  size="sm"
                  onClick={handleSave}
                  disabled={!editMode}
                  className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0"
                >
                  <Save className="h-4 w-4 mr-2" />
                  Save
                </Button>
              )}
            </div>

            {/* The entire decision table as one grid, in the Excel sheet's
                order: CONTEXTS, INITIAL ACTIONS, CONDITIONS, ACTIONS, POLICY
                STATEMENTS — section header rows inside the grid, every rule
                column aligned top to bottom. */}
            <div className={cn('flex-1 overflow-auto', editMode && 'bg-amber-500/[0.04]')}>
              <div className="p-2">
                <div className="border border-border/40 rounded-md overflow-hidden">
                  <div className="ag-theme-alpine-dark ag-tight">
                    <AgGridReact
                      columnDefs={columnDefs}
                      rowData={matrixRowData}
                      context={{ tableNames, navigate: navigateToTable }}
                      onCellValueChanged={handleCellValueChanged}
                      onColumnResized={handleColumnResized}
                      domLayout="autoHeight"
                      getRowStyle={(params) =>
                        params.data?.type === 'header'
                          ? headerRowStyle(params.data?.description)
                          : undefined
                      }
                      defaultColDef={{
                        resizable: true,
                        sortable: false,
                        suppressMovable: true,
                      }}
                    />
                  </div>
                </div>
              </div>
            </div>
          </>
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-green-500/20 to-blue-500/20 flex items-center justify-center">
                <Table2 className="h-8 w-8 text-green-400/60" />
              </div>
              <p className="text-muted-foreground">Select a decision table to edit</p>
              <p className="text-xs text-muted-foreground/60 mt-1">Choose from the list on the left</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * API client for communicating with the DTRules backend server.
 *
 * The backend runs on localhost:8080 and provides REST endpoints for:
 * - Project management (open, save, list files)
 * - Entity Definition (EDD) CRUD operations
 * - Decision Table (DT) CRUD operations
 * - Expression compilation and validation
 * - Rule execution with optional tracing
 *
 * @module api/client
 */

import type {
  ApiResponse,
  ProjectOpenResponse,
  EDDResponse,
  DTListResponse,
  DTDetailResponse,
  DTTreeResponse,
  CompileExpressionResponse,
  ExecuteResponse,
  Entity,
  DecisionTable,
  FileInfo,
} from '@/types/dtrules';

/** Base URL for all API requests - can be overridden via VITE_API_URL env variable */
// API base resolution: explicit override first; in dev (vite on :5173) the
// backend is the separate cmd/api process on :8080; in a production build
// the UI is served BY the API server (dtrules edit), so use its own origin —
// hard-coding :8080 made an editor on any other port silently talk to
// whatever happened to be listening on 8080.
const API_BASE =
  import.meta.env.VITE_API_URL ||
  (import.meta.env.DEV ? 'http://localhost:8080/api' : '/api');

/**
 * Generic fetch wrapper that handles JSON serialization/deserialization.
 *
 * @template T - The expected response type
 * @param url - The full URL to fetch
 * @param options - Optional fetch options (method, body, headers, etc.)
 * @returns Promise resolving to the parsed JSON response
 * @throws Error if the network request fails
 */
async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });
  return response.json();
}

// ============================================================================
// Project Endpoints
// ============================================================================

/**
 * Opens a DTRules project from the specified file system path.
 *
 * @param path - Absolute path to the project directory containing XML files
 * @returns Promise with project metadata (EDD files, DT files, map files)
 *
 * @example
 * const result = await openProject('/path/to/project/xml');
 * if (result.success) {
 *   console.log('EDD files:', result.eddFiles);
 *   console.log('DT files:', result.dtFiles);
 * }
 */
export async function openProject(path: string): Promise<ProjectOpenResponse> {
  return fetchJSON(`${API_BASE}/project/open`, {
    method: 'POST',
    body: JSON.stringify({ path }),
  });
}

/**
 * Saves all modified files in the current project.
 *
 * @returns Promise indicating success and list of saved files
 */
export async function saveProject(): Promise<{ success: boolean; error?: string; savedFiles?: string[] }> {
  return fetchJSON(`${API_BASE}/project/save`, {
    method: 'POST',
  });
}

/**
 * Lists all files in the current project with their modification status.
 *
 * @returns Promise with array of file information
 */
export async function listFiles(): Promise<{ success: boolean; error?: string; files?: FileInfo[] }> {
  return fetchJSON(`${API_BASE}/project/files`);
}

/** One entry in a directory listing from the browse endpoint. */
export interface BrowseEntry {
  name: string;
  path: string;
  isDir: boolean;
  /** File size in bytes (absent for directories). */
  size?: number;
}

/** Response from the directory browse endpoint. */
export interface BrowseResponse {
  success: boolean;
  error?: string;
  /** Absolute path of the listed directory */
  currentPath?: string;
  /** Directory contents: '..' parent first, then dirs, then files */
  entries?: BrowseEntry[];
  /** True when the directory contains *_dt.xml / *_edd.xml files */
  isProject?: boolean;
}

/**
 * Lists a server-side directory for the project picker.
 *
 * @param path - Directory to list; omit for the server's home directory
 */
export async function browseDirectory(path?: string): Promise<BrowseResponse> {
  const url = path ? `${API_BASE}/browse?path=${encodeURIComponent(path)}` : `${API_BASE}/browse`;
  return fetchJSON(url);
}

/**
 * Reports the project the backend already has loaded (e.g. passed to
 * `dtrules edit` at startup). `path` is empty when none is loaded.
 */
export async function getCurrentProject(): Promise<{
  success: boolean;
  path?: string;
  readOnly?: boolean;
  eddFiles?: string[];
  dtFiles?: string[];
  mapFiles?: string[];
}> {
  return fetchJSON(`${API_BASE}/project/current`);
}

/**
 * Renumbers decision tables to match the given order (100, 200, ...).
 * Used by drag-and-drop reordering.
 */
export async function reorderDecisionTables(order: string[]): Promise<ApiResponse> {
  return fetchJSON(`${API_BASE}/dt/reorder`, {
    method: 'POST',
    body: JSON.stringify({ order }),
  });
}

/**
 * Renumbers entities to match the given order (100, 200, ...).
 * Used by drag-and-drop reordering.
 */
export async function reorderEntities(order: string[]): Promise<ApiResponse> {
  return fetchJSON(`${API_BASE}/edd/reorder`, {
    method: 'POST',
    body: JSON.stringify({ order }),
  });
}

// ============================================================================
// Entity Definition (EDD) Endpoints
// ============================================================================

/**
 * Retrieves all entities from the project or a specific EDD file.
 *
 * @param file - Optional specific EDD file to load from
 * @returns Promise with array of entity definitions
 *
 * @example
 * // Get all entities
 * const all = await getEDD();
 *
 * // Get entities from specific file
 * const specific = await getEDD('CHIP_edd.xml');
 */
export async function getEDD(file?: string): Promise<EDDResponse> {
  const url = file ? `${API_BASE}/edd?file=${encodeURIComponent(file)}` : `${API_BASE}/edd`;
  return fetchJSON(url);
}

/**
 * Retrieves a single entity by name with full details.
 *
 * @param name - The entity name to retrieve
 * @returns Promise with the entity definition including all fields
 */
export async function getEntity(name: string): Promise<{ success: boolean; error?: string; entity?: Entity }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`);
}

/**
 * Creates a new entity in the specified EDD file.
 *
 * @param file - The EDD file to add the entity to
 * @param entity - The entity definition (name, access, comment, fields)
 * @returns Promise indicating success or failure with message
 */
export async function createEntity(file: string, entity: Partial<Entity>): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity?file=${encodeURIComponent(file)}`, {
    method: 'POST',
    body: JSON.stringify(entity),
  });
}

/**
 * Updates an existing entity.
 *
 * @param name - The current entity name
 * @param entity - The updated entity definition
 * @returns Promise indicating success or failure with message
 */
export async function updateEntity(name: string, entity: Entity): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(entity),
  });
}

/**
 * Deletes an entity by name.
 *
 * @param name - The entity name to delete
 * @returns Promise indicating success or failure with message
 */
export async function deleteEntity(name: string): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// Decision Table (DT) Endpoints
// ============================================================================

/**
 * Lists all decision tables in the project with summary information.
 *
 * @returns Promise with array of decision table summaries
 */
export async function listDecisionTables(): Promise<DTListResponse> {
  return fetchJSON(`${API_BASE}/dt`);
}

/**
 * Retrieves a decision table with full details including conditions, actions, and columns.
 *
 * @param name - The decision table name
 * @returns Promise with complete decision table structure
 */
export async function getDecisionTable(name: string): Promise<DTDetailResponse> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`);
}

/**
 * Creates a new decision table in the specified DT file.
 *
 * @param file - The DT file to add the table to
 * @param table - The table definition (tableName, type, comments)
 * @returns Promise indicating success or failure with message
 */
export async function createDecisionTable(
  file: string,
  table: { tableName: string; type: string; comments: string }
): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt?file=${encodeURIComponent(file)}`, {
    method: 'POST',
    body: JSON.stringify(table),
  });
}

/**
 * Updates an existing decision table.
 *
 * @param name - The current table name
 * @param table - The updated table definition (partial update supported)
 * @returns Promise indicating success or failure with message
 */
export async function updateDecisionTable(name: string, table: Partial<DecisionTable>): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(table),
  });
}

/**
 * Deletes a decision table by name.
 *
 * @param name - The table name to delete
 * @returns Promise indicating success or failure with message
 */
export async function deleteDecisionTable(name: string): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
}

/**
 * Retrieves the decision tree structure for visualization.
 *
 * The tree shows the hierarchical flow of conditions and actions,
 * including calls to other decision tables.
 *
 * @param name - The decision table name
 * @returns Promise with tree node structure for rendering
 */
export async function getDecisionTree(name: string): Promise<DTTreeResponse> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}/tree`);
}

// ============================================================================
// Compilation Endpoints
// ============================================================================

/**
 * Compiles and validates a postfix expression.
 *
 * Used for real-time validation of condition and action expressions
 * in the decision table editor.
 *
 * @param expression - The postfix expression to compile
 * @param entityName - Optional entity context for field resolution
 * @returns Promise with compilation result (success, errors, warnings)
 */
export async function compileExpression(expression: string, entityName?: string): Promise<CompileExpressionResponse> {
  return fetchJSON(`${API_BASE}/compile/expression`, {
    method: 'POST',
    body: JSON.stringify({ expression, entityName }),
  });
}

/**
 * Retrieves the list of available operators for autocomplete.
 *
 * @returns Promise with array of operator names
 */
export async function getOperators(): Promise<{ operators: string[] }> {
  return fetchJSON(`${API_BASE}/compile/operators`);
}

/**
 * Retrieves the list of entity fields for autocomplete.
 *
 * @returns Promise with array of field names
 */
export async function getEntityFields(): Promise<{ fields: string[] }> {
  return fetchJSON(`${API_BASE}/compile/fields`);
}

// ============================================================================
// Execution Endpoints
// ============================================================================

/**
 * Executes a decision table with the provided test data.
 *
 * @param tableName - The entry point decision table to execute
 * @param data - The test data matching entity structure
 * @param trace - Whether to include execution trace (default: false)
 * @returns Promise with execution result and optional trace entries
 *
 * @example
 * const result = await executeRules('Compute_Eligibility', {
 *   client: { age: 12, citizenship: 'US' },
 *   case: { county_cd: 'AA' }
 * }, true);
 *
 * if (result.success) {
 *   console.log('Result:', result.result);
 *   console.log('Trace:', result.trace);
 * }
 */
export async function executeRules(
  tableName: string,
  data: Record<string, unknown>,
  trace = false
): Promise<ExecuteResponse> {
  return fetchJSON(`${API_BASE}/execute`, {
    method: 'POST',
    body: JSON.stringify({ tableName, data, trace }),
  });
}

/**
 * Validates execution data without actually running the rules.
 *
 * Checks that the data structure matches expected entity definitions.
 *
 * @param tableName - The decision table that would be executed
 * @param data - The test data to validate
 * @returns Promise indicating validation success or failure
 */
export async function validateExecution(
  tableName: string,
  data: Record<string, unknown>
): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/execute/validate`, {
    method: 'POST',
    body: JSON.stringify({ tableName, data }),
  });
}

// ============================================================================
// Health Check
// ============================================================================

/**
 * Checks if the backend server is running and responsive.
 *
 * Called periodically by the UI to update connection status.
 *
 * @returns Promise with server status
 */
export async function healthCheck(): Promise<{ status: string }> {
  return fetchJSON(`${API_BASE}/health`);
}

// ============================================================================
// Sample Projects Discovery
// ============================================================================

/**
 * Sample project information returned by the API.
 */
export interface SampleProject {
  name: string;
  path: string;
  description: string;
}

/**
 * Retrieves available sample projects from the backend.
 *
 * The backend discovers sample projects by scanning for directories
 * containing xml subdirectories with EDD files.
 *
 * @returns Promise with array of available sample projects
 *
 * @example
 * const result = await getSampleProjects();
 * if (result.success && result.samples.length > 0) {
 *   const chipProject = result.samples.find(s => s.name === 'CHIP');
 *   if (chipProject) {
 *     await openProject(chipProject.path);
 *   }
 * }
 */
export async function getSampleProjects(): Promise<{
  success: boolean;
  samples: SampleProject[];
  message?: string;
}> {
  return fetchJSON(`${API_BASE}/samples`);
}

// ============================================================================
// Trace Debugger Endpoints
// ============================================================================

/** A node in the loaded trace tree. */
export interface DebugNode {
  number: number;
  name: string;
  attrs?: Record<string, string>;
  body?: string;
  children: DebugNode[];
}

/** An entity frame in the replayed entity stack. */
export interface DebugFrame {
  name: string;
  id: number;
  attrs: Record<string, string>;
}

/** Result of loading a trace for debugging. */
export interface DebugLoadResponse {
  success: boolean;
  error?: string;
  tracePath?: string;
  nodes?: number;
  dtrulesVersion?: string;
  rulesFingerprint?: string;
  fingerprintMatch?: 'match' | 'mismatch' | 'unknown' | 'speculative';
  verifyMismatches?: string[];
  /** True when the active session is a speculative rerun. */
  speculative?: boolean;
}

/** Result of positioning the replay session at a trace node. */
export interface DebugPositionResponse {
  success: boolean;
  error?: string;
  position?: number;
  nodes?: number;
  context?: { table?: string; column?: string; action?: string };
  stack?: DebugFrame[];
}

/** Loads a trace file into the server's debug session. */
export async function debugLoad(path: string): Promise<DebugLoadResponse> {
  return fetchJSON(`${API_BASE}/debug/load`, { method: 'POST', body: JSON.stringify({ path }) });
}

/** Fetches the loaded trace as a tree. */
export async function debugTree(): Promise<{ success: boolean; error?: string; tree?: DebugNode }> {
  return fetchJSON(`${API_BASE}/debug/tree`);
}

/** Reports whether the server already has a debug session (e.g. preloaded
 *  by `dtrules debug`), with the same fields as debugLoad when it does. */
export async function debugStatus(): Promise<DebugLoadResponse & { loaded?: boolean }> {
  return fetchJSON(`${API_BASE}/debug/status`);
}

/** One condition requirement in a find why-chain. */
export interface FindConditionStep {
  number: number;
  dsl: string;
  required: string;
  actual: string;
}

/** One frame of a find why-chain (innermost table first). */
export interface FindChainLink {
  table: string;
  pass: number;
  passCount: number;
  column: string;
  action: string;
  passNode: number;
  conditions: FindConditionStep[] | null;
}

/** One recorded write of a searched field. */
export interface FindHit {
  node: number;
  entity: string;
  id: string;
  attr: string;
  value: string;
  chain: FindChainLink[];
}

/** Searches the loaded trace for writes of a field (EL case-insensitive). */
export async function debugFind(attr: string, entity?: string, value?: string): Promise<{
  success: boolean;
  error?: string;
  total?: number;
  hits?: FindHit[];
}> {
  const q = new URLSearchParams({ attr });
  if (entity) q.set('entity', entity);
  if (value) q.set('value', value);
  return fetchJSON(`${API_BASE}/debug/find?${q}`);
}

// ── report generator ─────────────────────────────────────────────────

/** One predicate on a report section's rows. */
export interface ReportFilter {
  field: string;
  op: string;
  value: string;
}

/** One section of a report spec: entity instances or array elements. */
export interface ReportSection {
  title?: string;
  entity?: string;
  source?: string;
  fields?: string[];
  where?: ReportFilter[];
  sort?: string;
  key?: string;
}

/** A user-composed, saveable report description. */
export interface ReportSpec {
  name: string;
  sections: ReportSection[];
}

/** One rendered report section. */
export interface ReportSectionResult {
  title: string;
  entity: string;
  key: string;
  fields: string[];
  rows: Record<string, string>[] | null;
  total: number;
  error?: string;
}

export interface ReportResult {
  name: string;
  sections: ReportSectionResult[];
}

export interface RowChange {
  key: string;
  before: Record<string, string>;
  after: Record<string, string>;
  fields: string[];
}

export interface SectionDiff {
  title: string;
  fields: string[];
  added: Record<string, string>[] | null;
  removed: Record<string, string>[] | null;
  changed: RowChange[] | null;
}

export interface ReportDiffResult {
  name: string;
  sections: SectionDiff[];
}

/** Runs a report spec against the active debug trace (and the baseline,
 *  with a diff, when a speculative session is active). */
export async function debugReport(spec: ReportSpec): Promise<{
  success: boolean;
  error?: string;
  report?: ReportResult;
  baseline?: ReportResult;
  diff?: ReportDiffResult;
}> {
  return fetchJSON(`${API_BASE}/debug/report`, { method: 'POST', body: JSON.stringify(spec) });
}

/** Reruns the trace's execution with ONE table speculatively edited —
 *  same inputs (seeded from the trace), project files untouched. The
 *  speculative trace becomes the active session; the original stays as
 *  the baseline for report diffs and restore. */
export async function debugSpeculate(table: unknown): Promise<DebugLoadResponse> {
  return fetchJSON(`${API_BASE}/debug/speculate`, { method: 'POST', body: JSON.stringify(table) });
}

/** Restores the baseline trace session after a speculation. */
export async function debugSpeculateReset(): Promise<DebugLoadResponse> {
  return fetchJSON(`${API_BASE}/debug/speculate/reset`, { method: 'POST', body: '{}' });
}

/** Lists report specs saved in the project (reports/*.report.json). */
export async function listReportSpecs(): Promise<{
  success: boolean;
  error?: string;
  specs?: { name: string; spec: ReportSpec }[];
}> {
  return fetchJSON(`${API_BASE}/reports`);
}

/** Saves a report spec into the project. */
export async function saveReportSpec(name: string, spec: ReportSpec): Promise<{ success: boolean; error?: string }> {
  return fetchJSON(`${API_BASE}/reports`, { method: 'POST', body: JSON.stringify({ name, spec }) });
}

/** Replays to a trace node ("run to here" / stepping). */
export async function debugPosition(node: number): Promise<DebugPositionResponse> {
  return fetchJSON(`${API_BASE}/debug/position`, { method: 'POST', body: JSON.stringify({ node }) });
}

/** Executes read-only postfix at the current position; returns the leftover data stack. */
export async function debugConsole(postfix: string): Promise<{ success: boolean; error?: string; results?: string[] }> {
  return fetchJSON(`${API_BASE}/debug/console`, { method: 'POST', body: JSON.stringify({ postfix }) });
}

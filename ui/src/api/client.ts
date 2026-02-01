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

/** Base URL for all API requests */
const API_BASE = 'http://localhost:8080/api';

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

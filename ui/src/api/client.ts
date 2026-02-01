// API client for communicating with the DTRules backend

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

const API_BASE = 'http://localhost:8080/api';

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

// Project endpoints
export async function openProject(path: string): Promise<ProjectOpenResponse> {
  return fetchJSON(`${API_BASE}/project/open`, {
    method: 'POST',
    body: JSON.stringify({ path }),
  });
}

export async function saveProject(): Promise<{ success: boolean; error?: string; savedFiles?: string[] }> {
  return fetchJSON(`${API_BASE}/project/save`, {
    method: 'POST',
  });
}

export async function listFiles(): Promise<{ success: boolean; error?: string; files?: FileInfo[] }> {
  return fetchJSON(`${API_BASE}/project/files`);
}

// EDD endpoints
export async function getEDD(file?: string): Promise<EDDResponse> {
  const url = file ? `${API_BASE}/edd?file=${encodeURIComponent(file)}` : `${API_BASE}/edd`;
  return fetchJSON(url);
}

export async function getEntity(name: string): Promise<{ success: boolean; error?: string; entity?: Entity }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`);
}

export async function createEntity(file: string, entity: Partial<Entity>): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity?file=${encodeURIComponent(file)}`, {
    method: 'POST',
    body: JSON.stringify(entity),
  });
}

export async function updateEntity(name: string, entity: Entity): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(entity),
  });
}

export async function deleteEntity(name: string): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/edd/entity/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
}

// Decision Table endpoints
export async function listDecisionTables(): Promise<DTListResponse> {
  return fetchJSON(`${API_BASE}/dt`);
}

export async function getDecisionTable(name: string): Promise<DTDetailResponse> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`);
}

export async function createDecisionTable(
  file: string,
  table: { tableName: string; type: string; comments: string }
): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt?file=${encodeURIComponent(file)}`, {
    method: 'POST',
    body: JSON.stringify(table),
  });
}

export async function updateDecisionTable(name: string, table: Partial<DecisionTable>): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(table),
  });
}

export async function deleteDecisionTable(name: string): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
}

export async function getDecisionTree(name: string): Promise<DTTreeResponse> {
  return fetchJSON(`${API_BASE}/dt/${encodeURIComponent(name)}/tree`);
}

// Compile endpoints
export async function compileExpression(expression: string, entityName?: string): Promise<CompileExpressionResponse> {
  return fetchJSON(`${API_BASE}/compile/expression`, {
    method: 'POST',
    body: JSON.stringify({ expression, entityName }),
  });
}

export async function getOperators(): Promise<{ operators: string[] }> {
  return fetchJSON(`${API_BASE}/compile/operators`);
}

export async function getEntityFields(): Promise<{ fields: string[] }> {
  return fetchJSON(`${API_BASE}/compile/fields`);
}

// Execute endpoints
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

export async function validateExecution(
  tableName: string,
  data: Record<string, unknown>
): Promise<{ success: boolean; error?: string; message?: string }> {
  return fetchJSON(`${API_BASE}/execute/validate`, {
    method: 'POST',
    body: JSON.stringify({ tableName, data }),
  });
}

// Health check
export async function healthCheck(): Promise<{ status: string }> {
  return fetchJSON(`${API_BASE}/health`);
}

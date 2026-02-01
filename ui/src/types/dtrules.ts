// DTRules TypeScript interfaces

export interface Entity {
  name: string;
  access: string;
  comment: string;
  fields: Field[];
}

export interface Field {
  name: string;
  type: FieldType;
  subtype: string;
  access: string;
  input: string;
  defaultValue: string;
  comment: string;
}

export type FieldType =
  | 'string'
  | 'integer'
  | 'boolean'
  | 'date'
  | 'double'
  | 'entity'
  | 'array';

export interface DecisionTableSummary {
  name: string;
  type: DecisionTableType;
  comments: string;
  conditionCount: number;
  actionCount: number;
  columnCount: number;
}

export interface DecisionTable {
  tableName: string;
  xlsFile: string;
  type: DecisionTableType;
  comments: string;
  tableNumber: string;
  contexts: Context[];
  initialActions: string;
  conditions: Condition[];
  actions: Action[];
  policyStatements: PolicyStatement[];
  columnCount: number;
}

export type DecisionTableType = 'FIRST' | 'ALL';

export interface Context {
  number: number;
  comment: string;
  description: string;
  postfix: string;
}

export interface Condition {
  number: number;
  comment: string;
  requirement: string;
  description: string;
  postfix: string;
  columns: Record<number, ConditionValue>;
}

export type ConditionValue = 'Y' | 'N' | '-' | '*';

export interface Action {
  number: number;
  comment: string;
  requirement: string;
  description: string;
  postfix: string;
  columns: Record<number, ActionValue>;
}

export type ActionValue = 'X' | '';

export interface PolicyStatement {
  column: string;
  description: string;
  postfix: string;
}

export interface TreeNode {
  id: string;
  type: 'start' | 'condition' | 'action' | 'actions' | 'end';
  label: string;
  description?: string;
  column?: number;
  trueChild?: TreeNode;
  falseChild?: TreeNode;
  children?: TreeNode[];
}

export interface FileInfo {
  name: string;
  path: string;
  type: 'edd' | 'dt' | 'map' | 'other';
  modified: boolean;
}

export interface TraceEntry {
  tableName: string;
  column: number;
  type: 'condition' | 'action';
  index: number;
  value?: string;
  result?: boolean;
}

// API Response types
export interface ApiResponse<T = unknown> {
  success: boolean;
  error?: string;
  message?: string;
  data?: T;
}

export interface ProjectOpenResponse {
  success: boolean;
  error?: string;
  eddFiles?: string[];
  dtFiles?: string[];
  mapFiles?: string[];
}

export interface EDDResponse {
  success: boolean;
  error?: string;
  entities?: Entity[];
  filePath?: string;
}

export interface DTListResponse {
  success: boolean;
  error?: string;
  tables?: DecisionTableSummary[];
}

export interface DTDetailResponse extends DecisionTable {
  success: boolean;
  error?: string;
}

export interface DTTreeResponse {
  success: boolean;
  error?: string;
  tree?: TreeNode;
}

export interface CompileExpressionResponse {
  valid: boolean;
  error?: string;
  warnings?: string[];
  tokens?: string[];
}

export interface ExecuteResponse {
  success: boolean;
  error?: string;
  result?: Record<string, unknown>;
  trace?: TraceEntry[];
}

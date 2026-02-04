/**
 * DTRules TypeScript Type Definitions
 *
 * This module contains all TypeScript interfaces and types used throughout
 * the DTRules UI application. These types mirror the data structures used
 * by the Go backend and XML file format.
 *
 * @module types/dtrules
 */

// ============================================================================
// Entity Definition (EDD) Types
// ============================================================================

/**
 * Represents an entity definition in the EDD (Entity Definition Document).
 *
 * Entities define the data structures (vocabulary) that decision tables
 * operate on. Think of them as classes or database tables.
 *
 * @example
 * const clientEntity: Entity = {
 *   name: 'client',
 *   access: 'rw',
 *   comment: 'Represents an applicant',
 *   fields: [
 *     { name: 'age', type: 'integer', ... },
 *     { name: 'name', type: 'string', ... }
 *   ]
 * };
 */
export interface Entity {
  /** Unique entity name (e.g., 'client', 'case', 'income') */
  name: string;
  /** Access level: 'r' (read), 'w' (write), or 'rw' (read-write) */
  access: string;
  /** Documentation comment describing the entity's purpose */
  comment: string;
  /** Array of field definitions for this entity */
  fields: Field[];
}

/**
 * Represents a field (attribute) within an entity.
 *
 * Fields define the properties of an entity, including their type,
 * access level, and default values.
 */
export interface Field {
  /** Field name (e.g., 'age', 'firstName', 'isEligible') */
  name: string;
  /** Data type of the field */
  type: FieldType;
  /** Subtype for entity/array types (e.g., entity name for references) */
  subtype: string;
  /** Access level: 'r', 'w', or 'rw' */
  access: string;
  /** Input mapping for data loading */
  input: string;
  /** Default value if not provided */
  defaultValue: string;
  /** Documentation comment */
  comment: string;
}

/**
 * Valid field types in DTRules.
 *
 * - `string`: Text values
 * - `integer`: Whole numbers
 * - `boolean`: True/false values
 * - `date`: Date values
 * - `double`: Floating-point numbers
 * - `entity`: Reference to another entity
 * - `array`: Collection of values
 */
export type FieldType =
  | 'string'
  | 'integer'
  | 'boolean'
  | 'date'
  | 'double'
  | 'entity'
  | 'array';

// ============================================================================
// Decision Table Types
// ============================================================================

/**
 * Summary information about a decision table.
 *
 * Used for listing tables without loading full details.
 */
export interface DecisionTableSummary {
  /** Table name (unique identifier) */
  name: string;
  /** Execution type: FIRST (stop at first match) or ALL (execute all matches) */
  type: DecisionTableType;
  /** Documentation comment */
  comments: string;
  /** Number of condition rows */
  conditionCount: number;
  /** Number of action rows */
  actionCount: number;
  /** Number of rule columns */
  columnCount: number;
}

/**
 * Complete decision table definition with all conditions, actions, and columns.
 *
 * A decision table encodes business rules in a matrix format where:
 * - Rows are conditions (if) and actions (then)
 * - Columns are individual rules
 * - Cells indicate whether conditions must match and actions should execute
 */
export interface DecisionTable {
  /** Unique table name */
  tableName: string;
  /** Source Excel file name */
  xlsFile: string;
  /** Execution type */
  type: DecisionTableType;
  /** Documentation comment */
  comments: string;
  /** Table number (ordering) */
  tableNumber: string;
  /** Iteration contexts for looping over collections */
  contexts: Context[];
  /** Actions to execute before evaluating columns */
  initialActions: string;
  /** Condition rows (evaluated top-to-bottom) */
  conditions: Condition[];
  /** Action rows (executed when column matches) */
  actions: Action[];
  /** Policy statements describing each column */
  policyStatements: PolicyStatement[];
  /** Number of rule columns */
  columnCount: number;
}

/**
 * Decision table execution type.
 *
 * - `FIRST`: Stop after the first matching column (exclusive rules)
 * - `ALL`: Execute all matching columns (accumulative rules)
 */
export type DecisionTableType = 'FIRST' | 'ALL';

/**
 * Iteration context for looping over entity collections.
 *
 * Contexts allow decision tables to iterate over arrays of entities
 * (e.g., "for each client in clients").
 */
export interface Context {
  /** Context number (ordering) */
  number: number;
  /** Documentation comment */
  comment: string;
  /** Human-readable description */
  description: string;
  /** Postfix expression for iteration */
  postfix: string;
}

/**
 * A condition row in a decision table.
 *
 * Conditions are evaluated for each column. All conditions in a column
 * must match for that column's actions to execute.
 */
export interface Condition {
  /** Row number (ordering) */
  number: number;
  /** Documentation comment */
  comment: string;
  /** Requirement reference */
  requirement: string;
  /** Human-readable description of the condition */
  description: string;
  /** Postfix expression that evaluates to boolean */
  postfix: string;
  /** Column values: maps column number to expected value */
  columns: Record<number, ConditionValue>;
}

/**
 * Valid values for condition cells in a decision table.
 *
 * - `Y`: Condition must evaluate to TRUE
 * - `N`: Condition must evaluate to FALSE
 * - `-`: Don't care (condition is ignored for this column)
 * - `*`: Wildcard/special handling
 */
export type ConditionValue = 'Y' | 'N' | '-' | '*';

/**
 * An action row in a decision table.
 *
 * Actions are executed when all conditions in a column match.
 */
export interface Action {
  /** Row number (ordering) */
  number: number;
  /** Documentation comment */
  comment: string;
  /** Requirement reference */
  requirement: string;
  /** Human-readable description of the action */
  description: string;
  /** Postfix expression to execute */
  postfix: string;
  /** Column values: maps column number to execution flag */
  columns: Record<number, ActionValue>;
}

/**
 * Valid values for action cells in a decision table.
 *
 * - `X`: Execute this action
 * - `''`: Skip this action (empty string)
 */
export type ActionValue = 'X' | '';

/**
 * Policy statement describing a rule column.
 *
 * Policy statements provide documentation for each column,
 * explaining the business rule it represents.
 */
export interface PolicyStatement {
  /** Column identifier */
  column: string;
  /** Human-readable description of the rule */
  description: string;
  /** Optional postfix expression */
  postfix: string;
}

// ============================================================================
// Visualization Types
// ============================================================================

/**
 * Node in the decision tree visualization.
 *
 * Used by React Flow to render the decision table hierarchy.
 */
export interface TreeNode {
  /** Unique node identifier */
  id: string;
  /** Node type for styling and behavior */
  type: 'start' | 'condition' | 'action' | 'actions' | 'end';
  /** Display label */
  label: string;
  /** Optional description tooltip */
  description?: string;
  /** Column number (for condition nodes) */
  column?: number;
  /** Child node for TRUE branch (condition nodes) */
  trueChild?: TreeNode;
  /** Child node for FALSE branch (condition nodes) */
  falseChild?: TreeNode;
  /** Child nodes for sequential execution */
  children?: TreeNode[];
}

// ============================================================================
// Project & File Types
// ============================================================================

/**
 * Information about a file in the project.
 */
export interface FileInfo {
  /** File name without path */
  name: string;
  /** Full file path */
  path: string;
  /** File type based on content */
  type: 'edd' | 'dt' | 'map' | 'other';
  /** Whether the file has unsaved changes */
  modified: boolean;
}

// ============================================================================
// Execution & Tracing Types
// ============================================================================

/**
 * Entry in the execution trace log.
 *
 * Traces show the step-by-step execution of decision tables,
 * useful for debugging and understanding rule behavior.
 */
export interface TraceEntry {
  /** Name of the decision table being executed */
  tableName: string;
  /** Column number being evaluated */
  column: number;
  /** Whether this is a condition or action step */
  type: 'condition' | 'action';
  /** Row index within conditions or actions */
  index: number;
  /** Value that was computed (for actions) */
  value?: string;
  /** Evaluation result (for conditions) */
  result?: boolean;
}

// ============================================================================
// API Response Types
// ============================================================================

/**
 * Generic API response wrapper.
 *
 * All API endpoints return responses in this format.
 */
export interface ApiResponse<T = unknown> {
  /** Whether the operation succeeded */
  success: boolean;
  /** Error message if operation failed */
  error?: string;
  /** Success message */
  message?: string;
  /** Response payload */
  data?: T;
}

/**
 * Response from opening a project.
 */
export interface ProjectOpenResponse {
  success: boolean;
  error?: string;
  /** List of EDD file names in the project */
  eddFiles?: string[];
  /** List of DT file names in the project */
  dtFiles?: string[];
  /** List of map file names in the project */
  mapFiles?: string[];
}

/**
 * Response containing entity definitions.
 */
export interface EDDResponse {
  success: boolean;
  error?: string;
  /** Array of entity definitions */
  entities?: Entity[];
  /** Path to the loaded EDD file */
  filePath?: string;
}

/**
 * Response containing decision table list.
 */
export interface DTListResponse {
  success: boolean;
  error?: string;
  /** Array of decision table summaries */
  tables?: DecisionTableSummary[];
}

/**
 * Response containing full decision table details.
 */
export interface DTDetailResponse extends DecisionTable {
  success: boolean;
  error?: string;
}

/**
 * Response containing decision tree for visualization.
 */
export interface DTTreeResponse {
  success: boolean;
  error?: string;
  /** Root node of the decision tree */
  tree?: TreeNode;
}

/**
 * Response from expression compilation.
 */
export interface CompileExpressionResponse {
  /** Whether the expression is valid */
  valid: boolean;
  /** Compilation error if invalid */
  error?: string;
  /** Non-fatal warnings */
  warnings?: string[];
  /** Parsed tokens (for debugging) */
  tokens?: string[];
}

/**
 * Response from rule execution.
 */
export interface ExecuteResponse {
  success: boolean;
  error?: string;
  /** Execution result (modified entity data) */
  result?: Record<string, unknown>;
  /** Execution trace if tracing was enabled */
  trace?: TraceEntry[];
  /** Warnings from entity loading or execution (non-fatal issues) */
  warnings?: string[];
  /** Additional context when execution fails */
  context?: {
    tableName?: string;
    stackDepth?: number;
    entityDepth?: number;
  };
}

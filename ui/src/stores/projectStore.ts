/**
 * Project Store - Central state management for DTRules projects.
 *
 * This store manages all project-related state including:
 * - Project metadata (path, loading state, errors)
 * - File lists (EDD, DT, map files)
 * - Loaded data (entities, decision tables)
 * - Currently selected items (entity, table)
 * - UI state (active tab, navigation history)
 *
 * Uses Zustand for lightweight, TypeScript-first state management.
 *
 * @module stores/projectStore
 *
 * @example
 * // In a component
 * const { projectPath, entities, selectEntity } = useProjectStore();
 *
 * // Select specific state (prevents unnecessary re-renders)
 * const entities = useProjectStore((state) => state.entities);
 */

import { create } from 'zustand';
import type {
  Entity,
  DecisionTableSummary,
  DecisionTable,
  FileInfo,
} from '@/types/dtrules';
import * as api from '@/api/client';

/**
 * Complete state interface for the project store.
 *
 * Includes both state values and action methods.
 */
interface ProjectState {
  // Project state
  projectPath: string | null;
  isLoading: boolean;
  error: string | null;

  // Files
  files: FileInfo[];
  eddFiles: string[];
  dtFiles: string[];
  mapFiles: string[];

  // Data
  entities: Entity[];
  decisionTables: DecisionTableSummary[];
  currentEntity: Entity | null;
  currentTable: DecisionTable | null;

  // UI state
  activeTab: 'edd' | 'dt' | 'test' | 'tree';
  selectedFile: string | null;

  // Navigation history (browser-style back/forward)
  navigationHistory: Array<'edd' | 'dt' | 'test' | 'tree'>;
  navigationIndex: number;

  // Actions
  openProject: (path: string) => Promise<boolean>;
  saveProject: () => Promise<boolean>;
  refreshFiles: () => Promise<void>;

  // EDD actions
  loadEDD: (file?: string) => Promise<void>;
  selectEntity: (name: string) => Promise<void>;
  createEntity: (entity: Partial<Entity>) => Promise<boolean>;
  updateEntity: (entity: Entity) => Promise<boolean>;
  deleteEntity: (name: string) => Promise<boolean>;

  // DT actions
  loadDecisionTables: () => Promise<void>;
  selectTable: (name: string) => Promise<void>;
  createTable: (table: { tableName: string; type: string; comments: string }) => Promise<boolean>;
  updateTable: (table: Partial<DecisionTable>) => Promise<boolean>;
  deleteTable: (name: string) => Promise<boolean>;

  // UI actions
  setActiveTab: (tab: 'edd' | 'dt' | 'test' | 'tree') => void;
  setActiveTabWithHistory: (tab: 'edd' | 'dt' | 'test' | 'tree') => void;
  setSelectedFile: (file: string | null) => void;
  clearError: () => void;
  closeProject: () => void;
  autoSelectFirstItems: () => Promise<void>;

  // Navigation actions (browser-style)
  canGoBack: () => boolean;
  canGoForward: () => boolean;
  goBack: () => void;
  goForward: () => void;
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  // Initial state
  projectPath: null,
  isLoading: false,
  error: null,
  files: [],
  eddFiles: [],
  dtFiles: [],
  mapFiles: [],
  entities: [],
  decisionTables: [],
  currentEntity: null,
  currentTable: null,
  activeTab: 'edd',
  selectedFile: null,
  navigationHistory: ['edd'],
  navigationIndex: 0,

  // Project actions
  openProject: async (path: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.openProject(path);
      if (!response.success) {
        set({ isLoading: false, error: response.error });
        return false;
      }

      set({
        projectPath: path,
        eddFiles: response.eddFiles || [],
        dtFiles: response.dtFiles || [],
        mapFiles: response.mapFiles || [],
        isLoading: false,
      });

      // Load initial data
      await get().loadEDD();
      await get().loadDecisionTables();
      await get().refreshFiles();

      return true;
    } catch (err) {
      set({ isLoading: false, error: String(err) });
      return false;
    }
  },

  saveProject: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.saveProject();
      if (!response.success) {
        set({ isLoading: false, error: response.error });
        return false;
      }
      set({ isLoading: false });
      await get().refreshFiles();
      return true;
    } catch (err) {
      set({ isLoading: false, error: String(err) });
      return false;
    }
  },

  refreshFiles: async () => {
    try {
      const response = await api.listFiles();
      if (response.success && response.files) {
        set({ files: response.files });
      }
    } catch (err) {
      console.error('Failed to refresh files:', err);
    }
  },

  // EDD actions
  loadEDD: async (file?: string) => {
    try {
      const response = await api.getEDD(file);
      if (response.success && response.entities) {
        set({ entities: response.entities });
        if (file) {
          set({ selectedFile: file });
        }
      }
    } catch (err) {
      console.error('Failed to load EDD:', err);
    }
  },

  selectEntity: async (name: string) => {
    try {
      const response = await api.getEntity(name);
      if (response.success && response.entity) {
        set({ currentEntity: response.entity });
      }
    } catch (err) {
      console.error('Failed to select entity:', err);
    }
  },

  createEntity: async (entity: Partial<Entity>) => {
    const { eddFiles } = get();
    if (eddFiles.length === 0) {
      set({ error: 'No EDD file loaded' });
      return false;
    }

    try {
      const response = await api.createEntity(eddFiles[0], entity);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadEDD();
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  updateEntity: async (entity: Entity) => {
    try {
      const response = await api.updateEntity(entity.name, entity);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadEDD();
      set({ currentEntity: entity });
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  deleteEntity: async (name: string) => {
    try {
      const response = await api.deleteEntity(name);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadEDD();
      set({ currentEntity: null });
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  // DT actions
  loadDecisionTables: async () => {
    try {
      const response = await api.listDecisionTables();
      if (response.success && response.tables) {
        set({ decisionTables: response.tables });
      }
    } catch (err) {
      console.error('Failed to load decision tables:', err);
    }
  },

  selectTable: async (name: string) => {
    try {
      const response = await api.getDecisionTable(name);
      if (response.success) {
        set({
          currentTable: {
            tableName: response.tableName,
            xlsFile: response.xlsFile,
            type: response.type as 'FIRST' | 'ALL',
            comments: response.comments,
            tableNumber: response.tableNumber,
            contexts: response.contexts,
            initialActions: response.initialActions,
            conditions: response.conditions,
            actions: response.actions,
            policyStatements: response.policyStatements,
            columnCount: response.columnCount,
          },
        });
      }
    } catch (err) {
      console.error('Failed to select table:', err);
    }
  },

  createTable: async (table: { tableName: string; type: string; comments: string }) => {
    const { dtFiles } = get();
    if (dtFiles.length === 0) {
      set({ error: 'No DT file loaded' });
      return false;
    }

    try {
      const response = await api.createDecisionTable(dtFiles[0], table);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadDecisionTables();
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  updateTable: async (table: Partial<DecisionTable>) => {
    const { currentTable } = get();
    if (!currentTable) {
      set({ error: 'No table selected' });
      return false;
    }

    try {
      const response = await api.updateDecisionTable(currentTable.tableName, table);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadDecisionTables();
      await get().selectTable(table.tableName || currentTable.tableName);
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  deleteTable: async (name: string) => {
    try {
      const response = await api.deleteDecisionTable(name);
      if (!response.success) {
        set({ error: response.error });
        return false;
      }
      await get().loadDecisionTables();
      set({ currentTable: null });
      return true;
    } catch (err) {
      set({ error: String(err) });
      return false;
    }
  },

  // UI actions
  setActiveTab: (tab) => set({ activeTab: tab }),

  setActiveTabWithHistory: (tab) => set((state) => {
    // Don't add to history if it's the same tab
    if (state.activeTab === tab) return state;

    // Truncate forward history when navigating to new tab
    const newHistory = state.navigationHistory.slice(0, state.navigationIndex + 1);
    newHistory.push(tab);

    return {
      activeTab: tab,
      navigationHistory: newHistory,
      navigationIndex: newHistory.length - 1,
    };
  }),

  setSelectedFile: (file) => set({ selectedFile: file }),
  clearError: () => set({ error: null }),

  closeProject: () => set({
    projectPath: null,
    files: [],
    eddFiles: [],
    dtFiles: [],
    mapFiles: [],
    entities: [],
    decisionTables: [],
    currentEntity: null,
    currentTable: null,
    activeTab: 'edd',
    selectedFile: null,
    error: null,
    navigationHistory: ['edd'],
    navigationIndex: 0,
  }),

  autoSelectFirstItems: async () => {
    const { entities, decisionTables, selectEntity, selectTable } = get();

    // Auto-select first entity to populate EDD editor
    if (entities.length > 0 && entities[0].name) {
      await selectEntity(entities[0].name);
    }

    // Auto-select first decision table to populate DT editor
    if (decisionTables.length > 0 && decisionTables[0].name) {
      await selectTable(decisionTables[0].name);
    }
  },

  // Navigation actions (browser-style)
  canGoBack: () => get().navigationIndex > 0,

  canGoForward: () => {
    const state = get();
    return state.navigationIndex < state.navigationHistory.length - 1;
  },

  goBack: () => set((state) => {
    if (state.navigationIndex <= 0) return state;
    const newIndex = state.navigationIndex - 1;
    return {
      navigationIndex: newIndex,
      activeTab: state.navigationHistory[newIndex],
    };
  }),

  goForward: () => set((state) => {
    if (state.navigationIndex >= state.navigationHistory.length - 1) return state;
    const newIndex = state.navigationIndex + 1;
    return {
      navigationIndex: newIndex,
      activeTab: state.navigationHistory[newIndex],
    };
  }),
}));

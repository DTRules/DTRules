# DTRules UI Architecture

This document describes the architecture and design decisions of the DTRules UI application.

## Overview

DTRules UI is a React-based desktop application built with Electron. It provides a visual interface for creating and managing business rules using decision tables.

```
┌─────────────────────────────────────────────────────────────┐
│                        Electron Shell                        │
├─────────────────────────────────────────────────────────────┤
│                         React App                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                      Toolbar                             ││
│  ├────────────┬────────────────────────────────────────────┤│
│  │            │              Tab Content                    ││
│  │  Project   │  ┌─────────────────────────────────────┐   ││
│  │  Explorer  │  │  EDD Editor / DT Editor / Test /    │   ││
│  │            │  │  Tree Visualization                  │   ││
│  │            │  └─────────────────────────────────────┘   ││
│  ├────────────┴────────────────────────────────────────────┤│
│  │                      Status Bar                          ││
│  └─────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────┤
│                     Go Backend Server                        │
│                    (localhost:8080/api)                      │
└─────────────────────────────────────────────────────────────┘
```

## Technology Choices

### React + TypeScript

- **Why React**: Component-based architecture enables modular, reusable UI elements
- **Why TypeScript**: Strong typing catches errors at compile time and improves IDE support

### Zustand for State Management

- **Why not Redux**: Zustand is simpler, less boilerplate, and sufficient for our needs
- **Benefits**: Minimal API, TypeScript-first, built-in persistence middleware

### Vite + Electron

- **Why Vite**: Fast HMR, native ES modules, excellent TypeScript support
- **Why Electron**: Cross-platform desktop app with native file system access

### shadcn/ui + Tailwind CSS

- **Why shadcn**: Customizable components built on Radix UI primitives
- **Why Tailwind**: Utility-first CSS enables rapid UI development

### AG Grid

- **Why AG Grid**: Best-in-class data grid for complex tabular editing
- **Use case**: Decision table condition/action matrix editing

### Monaco Editor

- **Why Monaco**: VSCode's editor provides excellent code editing experience
- **Use case**: JSON test data input, postfix expression editing

### React Flow

- **Why React Flow**: Excellent library for interactive node-based diagrams
- **Use case**: Decision table hierarchy visualization

## Component Architecture

### Layer Structure

```
┌─────────────────────────────────────────────────────┐
│                    UI Components                     │
│  (App, Toolbar, ProjectExplorer, Editors, etc.)     │
├─────────────────────────────────────────────────────┤
│                   State Management                   │
│         (projectStore, onboardingStore)             │
├─────────────────────────────────────────────────────┤
│                     API Client                       │
│                   (client.ts)                        │
├─────────────────────────────────────────────────────┤
│                   Go Backend                         │
│              (REST API on :8080)                     │
└─────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
App
├── WelcomeScreen (conditional)
│   └── TutorialOfferDialog
├── Toolbar
│   ├── Navigation (Back/Forward)
│   ├── Project Actions (Open/Save/Refresh)
│   └── Help Menu
├── ProjectExplorer
│   ├── Entities Tree
│   ├── Decision Tables Tree
│   └── Maps Tree
├── Tab Content
│   ├── EDDEditor (Entity Definitions)
│   │   ├── Entity List
│   │   └── Entity Form
│   ├── DTEditor (Decision Tables)
│   │   ├── Table List
│   │   └── AG Grid (Conditions/Actions)
│   ├── TestPanel
│   │   ├── Monaco Editor (JSON input)
│   │   └── Results Panel
│   └── TreeVisualization
│       └── React Flow Diagram
├── StatusBar
├── TutorialController
│   ├── ConceptModals (Phase 1)
│   ├── GuidedTutorial (Phase 2)
│   └── TutorialNavigation
└── Toaster (Notifications)
```

## State Management

### Project Store (`projectStore.ts`)

Manages all project-related state:

```typescript
interface ProjectState {
  // Project metadata
  projectPath: string | null;
  isLoading: boolean;
  error: string | null;

  // File lists
  eddFiles: string[];
  dtFiles: string[];
  mapFiles: string[];
  files: FileInfo[];

  // Loaded data
  entities: Entity[];
  decisionTables: DecisionTableSummary[];
  currentEntity: Entity | null;
  currentTable: DecisionTable | null;

  // UI state
  activeTab: 'edd' | 'dt' | 'test' | 'tree';
  selectedFile: string | null;

  // Navigation history (browser-style)
  navigationHistory: Array<'edd' | 'dt' | 'test' | 'tree'>;
  navigationIndex: number;
}
```

**Key Actions:**
- `openProject(path)` - Load project from disk
- `saveProject()` - Persist changes
- `selectEntity(name)` / `selectTable(name)` - Load item details
- `setActiveTabWithHistory(tab)` - Navigate with history tracking
- `goBack()` / `goForward()` - Browser-style navigation

### Onboarding Store (`onboardingStore.ts`)

Manages tutorial and welcome screen state:

```typescript
interface OnboardingState {
  // Welcome screen
  showWelcome: boolean;

  // Tutorial state
  tutorialActive: boolean;
  tutorialCompleted: boolean;
  tutorialStepIndex: number;

  // Two-phase tutorial
  conceptPhaseActive: boolean;
  conceptPhaseComplete: boolean;
  currentConceptStep: number;
  uiTourActive: boolean;

  // Preferences
  offerTutorial: boolean;
  dontAskAgain: boolean;
}
```

**Persistence**: Uses Zustand's `persist` middleware to save tutorial completion state in localStorage.

## Data Flow

### Opening a Project

```
User clicks "Open" → Toolbar.handleOpen()
    ↓
projectStore.openProject(path)
    ↓
api.openProject(path) → Backend loads XML files
    ↓
Store updates: eddFiles, dtFiles, mapFiles
    ↓
projectStore.loadEDD() + loadDecisionTables()
    ↓
api.getEDD() + api.listDecisionTables()
    ↓
Store updates: entities, decisionTables
    ↓
projectStore.autoSelectFirstItems()
    ↓
UI renders with loaded data
```

### Editing a Decision Table

```
User clicks table in list → DTEditor.handleTableSelect()
    ↓
projectStore.selectTable(name)
    ↓
api.getDecisionTable(name)
    ↓
Store updates: currentTable
    ↓
AG Grid renders conditions/actions
    ↓
User edits cell → onCellValueChanged()
    ↓
projectStore.updateTable(changes)
    ↓
api.updateDecisionTable(name, changes)
    ↓
Backend updates XML → returns success
```

## API Communication

### Request/Response Pattern

All API calls follow this pattern:

```typescript
interface ApiResponse<T> {
  success: boolean;
  error?: string;
  data?: T;
}
```

### Error Handling

```typescript
try {
  const response = await api.someAction();
  if (!response.success) {
    // Handle API-level error
    set({ error: response.error });
    return false;
  }
  // Handle success
  return true;
} catch (err) {
  // Handle network/parsing errors
  set({ error: String(err) });
  return false;
}
```

## Tutorial System

### Two-Phase Design

**Phase 1: Concept Modals**
- 5 modal dialogs teaching DTRules concepts
- No element targeting (simpler, no positioning issues)
- Sequential progression with Back/Next

**Phase 2: Interactive UI Tour**
- 20 steps using React Joyride
- Targets specific UI elements via `data-tutorial` attributes
- Automatic tab switching when needed
- Spotlight effect highlights current element

### Tutorial Flow

```
Welcome Screen
    ↓
Open CHIP Project
    ↓
Tutorial Offer Dialog
    ↓ (Start Tour)
Phase 1: Concept Modals (5 steps)
    ↓ (Complete)
Phase 2: UI Tour (20 steps)
    ↓ (Finish)
Tutorial Complete (persisted)
```

## Styling Architecture

### CSS Variables (Theme)

```css
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --primary: 222.2 47.4% 11.2%;
  /* ... */
}

.dark {
  --background: 222.2 84% 4.9%;
  --foreground: 210 40% 98%;
  /* ... */
}
```

### Component Styling Pattern

```tsx
// Using cn() utility for conditional classes
<div className={cn(
  "base-classes",
  condition && "conditional-classes",
  variant === 'primary' && "variant-classes"
)}>
```

### Special Styles

- **Glassmorphism**: `.glass` class with backdrop blur
- **Glow effects**: `.glow-primary`, `.glow-success`
- **Gradient text**: `.gradient-text`
- **AG Grid theme**: Custom dark theme variables
- **Joyride**: Disabled transitions to prevent flashing

## Performance Considerations

### Optimizations Applied

1. **Lazy loading**: Monaco Editor loads on demand
2. **Memoization**: AG Grid column definitions memoized
3. **Efficient re-renders**: Zustand selectors prevent unnecessary updates
4. **CSS transitions disabled**: On Joyride elements to prevent artifacts

### Potential Improvements

1. **Virtual scrolling**: For large entity/table lists
2. **Code splitting**: Dynamic imports for tab content
3. **Service Worker**: Cache static assets
4. **Web Workers**: Offload heavy computations

## Security Considerations

### Electron Security

- Context isolation enabled
- Node integration disabled in renderer
- Preload script for safe IPC

### Input Validation

- Project paths validated before opening
- JSON parsed with try/catch
- API responses checked for success

## Testing Strategy

### Recommended Tests

1. **Unit Tests**: Store actions, utility functions
2. **Component Tests**: Render and interaction tests
3. **Integration Tests**: API client with mock server
4. **E2E Tests**: Full workflows with Playwright

### Test Data

The CHIP sample project serves as the primary test fixture.

## Future Considerations

### Planned Features

1. **Autosave**: Automatic saving with debounce
2. **Undo/Redo**: Full edit history
3. **Recent Projects**: Quick access to previous projects
4. **Settings Panel**: User preferences

### Architecture Evolution

1. **Plugin System**: Custom editors/validators
2. **Collaboration**: Real-time multi-user editing
3. **Cloud Sync**: Project storage in cloud

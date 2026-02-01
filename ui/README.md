# DTRules UI

A modern React-based visual editor for creating and managing DTRules decision tables, entity definitions, and business rules.

## Features

- **Entity Editor (EDD)** - Visual editor for defining data structures with attributes, types, and validation rules
- **Decision Table Editor** - AG Grid-based editor with color-coded conditions and actions
- **Test & Execute Panel** - JSON input editor with execution tracing for validating rules
- **Tree Visualization** - Interactive flow diagram showing decision table hierarchy
- **Guided Tutorial** - Two-phase onboarding with concept learning and interactive UI tour

## Tech Stack

- **React 18** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Electron** - Desktop application framework
- **Zustand** - State management
- **Tailwind CSS** - Utility-first styling
- **shadcn/ui** - Component library
- **AG Grid** - Data grid for decision tables
- **Monaco Editor** - Code editor for JSON and expressions
- **React Flow** - Flow diagram visualization
- **React Joyride** - Interactive tutorials

## Prerequisites

- Node.js 18+
- npm or yarn
- Go 1.21+ (for backend server)

## Quick Start

```bash
# 1. Start the backend server (in one terminal)
cd ../go
go run ./cmd/api

# 2. Start the frontend (in another terminal)
cd ui
npm install
npm run dev
```

The backend runs on http://localhost:8080 and the frontend on http://localhost:5173.

## Installation

```bash
# Install dependencies
npm install

# Start development server (frontend only)
npm run dev

# Start with Electron (frontend + desktop app)
npm run electron:dev
```

## Development

```bash
# Run development server
npm run dev

# Type checking
npm run typecheck

# Linting
npm run lint

# Build for production
npm run build

# Build Electron app
npm run electron:build
```

## Project Structure

```
src/
├── api/                    # Backend API client
│   └── client.ts           # REST API functions
├── components/             # React components
│   ├── ui/                 # shadcn UI components
│   ├── App.tsx             # Main application
│   ├── EDDEditor.tsx       # Entity definition editor
│   ├── DTEditor.tsx        # Decision table editor
│   ├── TestPanel.tsx       # Test execution panel
│   ├── TreeVisualization.tsx # Flow diagram
│   ├── ProjectExplorer.tsx # Project navigation
│   ├── Toolbar.tsx         # Top toolbar
│   ├── StatusBar.tsx       # Bottom status bar
│   ├── WelcomeScreen.tsx   # Landing page
│   ├── GuidedTutorial.tsx  # Interactive tour
│   ├── ConceptModals.tsx   # Concept learning dialogs
│   └── ...
├── stores/                 # Zustand state stores
│   ├── projectStore.ts     # Project and data state
│   └── onboardingStore.ts  # Tutorial state
├── types/                  # TypeScript definitions
│   └── dtrules.ts          # Domain types
├── lib/                    # Utilities
│   └── utils.ts            # Helper functions
├── index.css               # Global styles
└── main.tsx                # Entry point
```

## Backend API

The UI communicates with a Go backend server at `http://localhost:8080/api`. Key endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Health check |
| `/api/project/open` | POST | Open project by path |
| `/api/project/save` | POST | Save current project |
| `/api/edd` | GET | Get all entities |
| `/api/edd/entity/{name}` | GET/PUT/DELETE | Entity CRUD |
| `/api/dt` | GET | List decision tables |
| `/api/dt/{name}` | GET/PUT/DELETE | Decision table CRUD |
| `/api/dt/{name}/tree` | GET | Get decision tree |
| `/api/execute` | POST | Execute rules |

## Decision Table Notation

| Symbol | Meaning |
|--------|---------|
| **Y** | Condition must be true |
| **N** | Condition must be false |
| **-** | Don't care (any value) |
| **X** | Execute this action |

## Sample Project

The CHIP (Children's Health Insurance Program) sample project demonstrates:
- Entity definitions: client, case, income
- Decision tables: Compute_Eligibility, Check_Age, Check_Income, Check_Citizenship
- Test data and execution tracing

Location: `../sampleprojects/CHIP/xml` (relative to the ui directory)

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `http://localhost:8080/api` | Backend API URL |

### Build Configuration

See `vite.config.ts` for Vite configuration and `electron/main.ts` for Electron settings.

## License

See the main DTRules project for licensing information.

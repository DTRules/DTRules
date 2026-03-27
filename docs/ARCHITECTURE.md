# DTRules Architecture

This document describes the architecture of DTRules, including its components, data flow, and extension points.

## Table of Contents

- [System Overview](#system-overview)
- [Core Components](#core-components)
- [Execution Flow](#execution-flow)
- [Module Structure](#module-structure)
- [Data Model](#data-model)
- [Stack-Based Interpreter](#stack-based-interpreter)
- [Extension Points](#extension-points)

---

## System Overview

DTRules is a Decision Table based Rules Engine that separates business logic from application code. Business analysts define rules in Excel spreadsheets using a Domain Specific Language, which are then compiled to XML and executed by the rules engine.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Your Application                                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────┐ │
│  │ Input Data  │───▶│  DTRules    │───▶│  Results/Decisions          │ │
│  │ (XML/Java)  │    │  Engine     │    │  (XML/Java)                 │ │
│  └─────────────┘    └─────────────┘    └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Compiled Rules (XML)                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │
│  │   Entity        │  │   Decision      │  │   Data                  │ │
│  │   Definitions   │  │   Tables        │  │   Mappings              │ │
│  │   (*_edd.xml)   │  │   (*_dt.xml)    │  │   (*_map.xml)           │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                              ▲
                              │ Compile
                              │
┌─────────────────────────────────────────────────────────────────────────┐
│                      Excel Source Files                                 │
│  ┌─────────────────┐  ┌─────────────────────────────────────────────┐  │
│  │   Entity        │  │              Decision Tables                │  │
│  │   Definition    │  │  ┌─────────────────────────────────────┐   │  │
│  │   Document      │  │  │ Conditions (EL expressions)         │   │  │
│  │   (*_edd.xls)   │  │  │ Actions (EL statements)             │   │  │
│  │                 │  │  │ Rule columns (Y/N/X)                │   │  │
│  └─────────────────┘  │  └─────────────────────────────────────┘   │  │
│                       └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### RulesDirectory

The entry point for loading rule configurations.

```
┌─────────────────────────────────────────┐
│            RulesDirectory               │
├─────────────────────────────────────────┤
│ - Loads DTRules.xml configuration       │
│ - Manages multiple RuleSets             │
│ - Provides file I/O abstraction         │
│ - Thread-safe, shareable                │
└─────────────────────────────────────────┘
              │
              │ contains
              ▼
┌─────────────────────────────────────────┐
│             RuleSet                     │
├─────────────────────────────────────────┤
│ - Entity definitions (schema)          │
│ - Decision table definitions            │
│ - Mapping configurations                │
│ - Session factory                       │
│ - Thread-safe, shareable                │
└─────────────────────────────────────────┘
              │
              │ creates
              ▼
┌─────────────────────────────────────────┐
│             IRSession                   │
├─────────────────────────────────────────┤
│ - Execution context                     │
│ - Entity instances                      │
│ - State management                      │
│ - NOT thread-safe (one per evaluation) │
└─────────────────────────────────────────┘
```

### Session and State

Each evaluation creates a new session with its own state:

```
┌─────────────────────────────────────────────────────────────┐
│                        IRSession                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                     DTState                          │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │   │
│  │  │ Data Stack  │ │ Ctrl Stack  │ │Entity Stack │    │   │
│  │  │ (operands)  │ │ (frames)    │ │ (context)   │    │   │
│  │  └─────────────┘ └─────────────┘ └─────────────┘    │   │
│  │                                                      │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │          Entity Instances                    │    │   │
│  │  │  HashMap<Object, IREntity>                   │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   Data Maps     │  │   Entity        │                  │
│  │   (mappings)    │  │   Factory       │                  │
│  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### Entity Model

Entities are the data objects processed by decision tables:

```
┌─────────────────────────────────────────────────────────────┐
│                       IREntity                              │
├─────────────────────────────────────────────────────────────┤
│  name: RName              // Entity type name               │
│  id: int                  // Unique instance ID             │
│  readonly: boolean        // Immutable flag                 │
├─────────────────────────────────────────────────────────────┤
│  Attributes:                                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ REntityEntry                                         │   │
│  │  - name: RName                                       │   │
│  │  - type: RType                                       │   │
│  │  - defaultValue: IRObject                            │   │
│  │  - readable/writable: boolean                        │   │
│  │  - comment: String                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Values: List<IRObject>   // Attribute values by index      │
└─────────────────────────────────────────────────────────────┘
```

---

## Execution Flow

### 1. Initialization

```
Application                 DTRules
    │                          │
    │  new RulesDirectory()    │
    │─────────────────────────▶│
    │                          │ Load DTRules.xml
    │                          │ Parse rule sets
    │                          │
    │  getRuleSet("name")      │
    │─────────────────────────▶│
    │                          │ Return RuleSet
    │◀─────────────────────────│
```

### 2. Session Creation

```
Application                 RuleSet                  Session
    │                          │                        │
    │  newSession()            │                        │
    │─────────────────────────▶│                        │
    │                          │  Create IRSession      │
    │                          │─────────────────────▶│
    │                          │  Initialize DTState    │
    │                          │  Create EntityFactory  │
    │                          │                        │
    │◀─────────────────────────────────────────────────│
    │       Session                                     │
```

### 3. Data Loading

```
Application                 Session                  Mapping
    │                          │                        │
    │  loadXml(input, "map")   │                        │
    │─────────────────────────▶│                        │
    │                          │  getMapping("map")     │
    │                          │───────────────────────▶│
    │                          │                        │
    │                          │  Parse XML             │
    │                          │  Create entities       │
    │                          │  Set attributes        │
    │                          │                        │
    │◀─────────────────────────│                        │
```

### 4. Rule Execution

```
Session                     DTState                  DecisionTable
    │                          │                        │
    │  execute("TableName")    │                        │
    │─────────────────────────▶│                        │
    │                          │  Load table            │
    │                          │───────────────────────▶│
    │                          │                        │
    │                          │  For each column:      │
    │                          │    Evaluate conditions │
    │                          │    Execute actions     │
    │                          │    (stack operations)  │
    │                          │                        │
    │◀─────────────────────────│                        │
```

### 5. Result Extraction

```
Application                 Session                  Entity
    │                          │                        │
    │  getState().find()       │                        │
    │─────────────────────────▶│                        │
    │                          │  Lookup entity         │
    │                          │───────────────────────▶│
    │                          │                        │
    │◀──────────────────────────────────────────────────│
    │     Result entity                                 │
    │                                                   │
    │  printEntityReport()     │                        │
    │─────────────────────────▶│                        │
    │                          │  Serialize to XML      │
    │◀─────────────────────────│                        │
```

---

## Module Structure

```
DTRules/
├── dtrules-engine/          # Core runtime engine
│   └── com.dtrules/
│       ├── session/         # Session, RuleSet, RulesDirectory
│       ├── entity/          # Entity model (IREntity, REntity)
│       ├── interpreter/     # Stack machine, operators
│       ├── decisiontables/  # Decision table execution
│       ├── mapping/         # XML-to-entity mapping
│       └── automapping/     # Java-to-entity reflection mapping
│
├── compilerutil/            # Compilation utilities
│   └── com.dtrules/
│       ├── compiler/        # Excel-to-XML compilation
│       ├── testsupport/     # Test harness base classes
│       └── trace/           # Execution tracing
│
├── dsl/                     # Domain Specific Languages
│   ├── el/                  # Expression Language
│   │   ├── antlr4/          # ANTLR 4 grammar (EL.g4) - recommended
│   │   ├── flex/            # Legacy lexer (scanner.flex)
│   │   └── cup/             # Legacy parser (parser.cup)
│   ├── ebl/                 # Entity Business Language
│   │   ├── antlr4/          # ANTLR 4 grammar (EBL.g4)
│   │   └── ...              # Legacy JFlex/CUP files
│   └── sudoku_language/     # Custom DSL example
│       ├── antlr4/          # ANTLR 4 grammar (Sudoku.g4)
│       └── ...              # Legacy JFlex/CUP files
│
└── sampleprojects/          # Example implementations
```

### Module Dependencies

```
                    ┌─────────────────┐
                    │  sampleprojects │
                    └────────┬────────┘
                             │ depends on
              ┌──────────────┴──────────────┐
              ▼                              ▼
    ┌─────────────────┐            ┌─────────────────┐
    │  compilerutil   │            │      dsl        │
    └────────┬────────┘            └────────┬────────┘
             │                              │
             └──────────────┬───────────────┘
                            ▼
                  ┌─────────────────┐
                  │  dtrules-engine │
                  └─────────────────┘
```

---

## Data Model

### Type System

All runtime values implement IRObject:

```
                        IRObject (interface)
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
   Primitives          Collections          Complex
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │RInteger │         │ RArray  │         │ REntity │
   │RDouble  │         │         │         │ RTable  │
   │RString  │         └─────────┘         │RXmlValue│
   │RBoolean │                             └─────────┘
   │ RDate   │
   │ RName   │
   │ RNull   │
   └─────────┘
```

### Entity Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                    Entity Definition                         │
│                    (from EDD Excel)                          │
├─────────────────────────────────────────────────────────────┤
│  Entity: case                                                │
│    └── clients: array of client                             │
│    └── relationships: array of relationship                 │
│                                                              │
│  Entity: client                                              │
│    └── name: string                                          │
│    └── age: integer                                          │
│    └── income: array of income                              │
│                                                              │
│  Entity: income                                              │
│    └── amount: double                                        │
│    └── type: string                                          │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ At runtime
┌─────────────────────────────────────────────────────────────┐
│                    Entity Instances                          │
├─────────────────────────────────────────────────────────────┤
│  case#1                                                      │
│    └── clients: [client#1, client#2]                        │
│                                                              │
│  client#1                                                    │
│    └── name: "John"                                          │
│    └── age: 35                                               │
│    └── income: [income#1]                                   │
│                                                              │
│  client#2                                                    │
│    └── name: "Jane"                                          │
│    └── age: 32                                               │
│    └── income: [income#2, income#3]                         │
└─────────────────────────────────────────────────────────────┘
```

---

## Stack-Based Interpreter

DTRules uses a stack-based interpreter (similar to Forth or PostScript) for executing compiled rules.

### Three Stacks

```
┌─────────────────────────────────────────────────────────────┐
│                        DTState                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Data Stack (datastk)           Control Stack (ctrlstk)    │
│  ┌─────────────────┐            ┌─────────────────┐        │
│  │ IRObject values │            │ Decision table  │        │
│  │ (operands)      │            │ call frames     │        │
│  │                 │            │                 │        │
│  │ ───────────     │            │ ───────────     │        │
│  │ operand 2       │            │ return addr     │        │
│  │ operand 1       │            │ table ref       │        │
│  │ ───────────     │            │ ───────────     │        │
│  └─────────────────┘            └─────────────────┘        │
│                                                             │
│  Entity Stack (entitystk)                                   │
│  ┌─────────────────┐                                        │
│  │ Current context │                                        │
│  │ entities        │                                        │
│  │                 │                                        │
│  │ ───────────     │                                        │
│  │ client          │  ◀── Attribute lookups search here    │
│  │ case            │                                        │
│  │ ───────────     │                                        │
│  └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

### Execution Example

EL expression: `client.age > 18`

Compiles to postfix: `client age get 18 >`

Execution:
```
Step 1: Push "client" name
  Data: [client_name]

Step 2: Execute "age" (get attribute)
  Data: [35]           (looked up client.age)

Step 3: Push 18
  Data: [35, 18]

Step 4: Execute ">" (greater than)
  Data: [true]         (35 > 18)
```

### Operators

Operators are registered in static initializers and execute against the stacks:

```java
public abstract class ROperator implements IRObject {

    // Operator name for lookup
    abstract RName getName();

    // Execute against state
    abstract void execute(DTState state) throws RulesException;

    // Stack signature documentation
    // e.g., "( int int -- int )" for addition
}
```

---

## Extension Points

### Custom DSL

Create a custom compiler by implementing the compiler interface:

```
┌─────────────────────────────────────────────────────────────┐
│                    Custom DSL                               │
├─────────────────────────────────────────────────────────────┤
│  Option A: ANTLR 4 (Recommended)                            │
│  1. Define grammar (MyDSL.g4)                               │
│  2. Create visitor class extending generated base visitor   │
│  3. Implement ICompiler interface                           │
│  4. Generate postfix code in visitor methods                │
│                                                             │
│  Option B: JFlex/CUP (Legacy)                               │
│  1. Define grammar (scanner.flex, parser.cup)               │
│  2. Implement ICompiler interface                           │
│  3. Generate postfix code in parser actions                 │
│                                                             │
│  Register in DTRules.xml:                                   │
│     <compileralias name="MyDSL">                           │
│       com.example.MyDSLCompiler                             │
│     </compileralias>                                        │
│     <compiler>MyDSL</compiler>                              │
└─────────────────────────────────────────────────────────────┘
```

See `dsl/ANTLR_MIGRATION.md` for detailed examples of both approaches.

### Custom Operators

Add new operators by extending ROperator:

```java
public class MyOperator extends ROperator {
    static {
        new MyOperator(); // Self-registration
    }

    public RName getName() {
        return RName.getRName("myop");
    }

    public void execute(DTState state) throws RulesException {
        IRObject arg = state.datapop();  // Get arguments
        // ... perform operation ...
        state.datapush(result);          // Push result
    }
}
```

### Custom Entity Factory

Override entity creation behavior:

```java
public class MyEntityFactory extends EntityFactory {

    @Override
    public IREntity newInstance(RName entityName, IRSession session) {
        // Custom entity creation logic
        // Useful for integration with ORM or other systems
    }
}
```

### Custom Data Mapping

Implement custom mapping between external formats and entities:

```java
// AutoDataMap uses reflection for Java objects
AutoDataMap mapper = session.getAutoDataMap(myJavaObject, definition);
IREntity entity = mapper.mapToEntity();

// Mapping interface for custom formats
public interface IDataMap {
    void loadData(IRSession session, InputStream input);
    void outputData(IRSession session, OutputStream output);
}
```

---

## Thread Safety

| Component | Thread Safe | Notes |
|-----------|-------------|-------|
| RulesDirectory | Yes | Share across threads |
| RuleSet | Yes | Share across threads |
| IRSession | **No** | Create per thread/evaluation |
| DTState | **No** | Part of session |
| IREntity | **No** | Part of session state |

### Recommended Pattern

```java
// Initialize once (application startup)
RulesDirectory directory = new RulesDirectory(path, "DTRules.xml");
RuleSet ruleSet = directory.getRuleSet("MyRules");

// Per-request processing
public Result evaluate(Input input) {
    IRSession session = ruleSet.newSession();  // New session per call
    try {
        // Load, execute, extract results
    } finally {
        // Session can be garbage collected
    }
}
```

---

## Performance Considerations

1. **RulesDirectory/RuleSet Loading** - Load once, reuse
2. **Session Creation** - Lightweight, create freely
3. **Entity Instances** - Pooled within session
4. **Decision Table Compilation** - Done once at load time
5. **Stack Operations** - O(1) for push/pop
6. **Entity Lookup** - HashMap, O(1) average

---

## Layered Architecture: Rule Set Management vs Runtime

DTRules has a clear separation between **Rule Set Management** and the **Runtime**. This distinction is critical for understanding how multiple runtime implementations can execute the same rules.

### The Two Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      RULE SET MANAGEMENT LAYER                          │
│                                                                         │
│  Responsibilities:                                                      │
│  - Load rule definitions (XML, Excel)                                   │
│  - Compile EL expressions to bytecode                                   │
│  - Manage entity definitions (schema)                                   │
│  - Store decision table definitions                                     │
│  - Provide session factory                                              │
│                                                                         │
│  Components: RulesDirectory, RuleSet, EntityFactory, Compiler           │
│                                                                         │
│  Characteristics:                                                       │
│  - Stateless (definitions only)                                         │
│  - Thread-safe, shareable                                               │
│  - Loads once, used many times                                          │
│                                                                         │
│  Output: Bytecode chunks, entity definitions, decision table metadata   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Provides bytecode to
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           RUNTIME LAYER                                 │
│                                                                         │
│  Responsibilities:                                                      │
│  - Execute bytecode                                                     │
│  - Manage execution state (stacks)                                      │
│  - Implement operators                                                  │
│  - Handle entity instances                                              │
│                                                                         │
│  Components: Runtime State, Bytecode Executor, Operator Table           │
│                                                                         │
│  Characteristics:                                                       │
│  - Stateful (owns execution state)                                      │
│  - NOT thread-safe (one per evaluation)                                 │
│  - Self-contained and complete                                          │
│  - Interchangeable implementations                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

### Critical Design Principle: Self-Contained Runtimes

Each runtime implementation MUST be self-contained. The execution state (data stack, entity stack, control stack) belongs to the Runtime, not to Rule Set Management.

```
WRONG: State split between layers
┌──────────────────────┐     ┌──────────────────────┐
│  Rule Set Management │     │       Runtime        │
│                      │     │                      │
│  ┌────────────────┐  │     │  ┌────────────────┐  │
│  │  Partial State │◀─┼─────┼─▶│  Partial State │  │
│  └────────────────┘  │     │  └────────────────┘  │
│                      │     │        │             │
└──────────────────────┘     │        ▼ fallback    │
                             │  ┌────────────────┐  │
                             │  │  Other Runtime │  │
                             │  └────────────────┘  │
                             └──────────────────────┘

RIGHT: Runtime owns complete state
┌──────────────────────┐     ┌──────────────────────┐
│  Rule Set Management │     │       Runtime        │
│                      │     │                      │
│  ┌────────────────┐  │     │  ┌────────────────┐  │
│  │   Bytecode     │──┼────▶│  │ Complete State │  │
│  │   Definitions  │  │     │  │ - Data Stack   │  │
│  │   Metadata     │  │     │  │ - Entity Stack │  │
│  └────────────────┘  │     │  │ - Ctrl Stack   │  │
│                      │     │  │ - Operators    │  │
└──────────────────────┘     │  └────────────────┘  │
                             │                      │
                             │  No fallbacks.       │
                             │  No syncing.         │
                             │  Complete execution. │
                             └──────────────────────┘
```

### Why This Matters

A runtime could execute on a different machine than where the rules are managed. Consider:

```
┌─────────────────────┐          ┌─────────────────────┐
│  Management Server  │          │  Execution Node     │
│                     │          │                     │
│  - Load rules       │  ─────▶  │  - Runtime only     │
│  - Compile bytecode │ bytecode │  - Execute rules    │
│  - No execution     │          │  - No compilation   │
└─────────────────────┘          └─────────────────────┘
```

If the runtime tries to "fall back" to another runtime or "sync" state with management code, this architecture breaks. The runtime must be complete and self-contained.

### Multiple Runtime Implementations

DTRules supports multiple runtime implementations, all executing the same bytecode:

```
                    ┌──────────────────────────────┐
                    │     Rule Set Management      │
                    │                              │
                    │  Compiles to bytecode once   │
                    └──────────────┬───────────────┘
                                   │
                                   │ Same bytecode
                    ┌──────────────┼──────────────┐
                    │              │              │
                    ▼              ▼              ▼
          ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
          │ Go Runtime  │ │Java Runtime │ │ ASM Runtime │
          │             │ │             │ │             │
          │ ┌─────────┐ │ │ ┌─────────┐ │ │ ┌─────────┐ │
          │ │DTState  │ │ │ │DTState  │ │ │ │VMState  │ │
          │ │(Go)     │ │ │ │(Java)   │ │ │ │(Native) │ │
          │ └─────────┘ │ │ └─────────┘ │ │ └─────────┘ │
          │             │ │             │ │             │
          │ Operators   │ │ Operators   │ │ Operators   │
          │ (Go impl)   │ │ (Java impl) │ │ (ASM impl)  │
          └─────────────┘ └─────────────┘ └─────────────┘

          Each runtime is COMPLETE and SELF-CONTAINED.
          No cross-runtime dependencies at execution time.
```

### The PostScript Model

DTRules follows the PostScript execution model: a single stack machine with uniform execution.

Key principles:
1. **One operand stack** - All values go on the same stack
2. **Uniform execution** - Everything is either data or executable
3. **No special cases** - Operators like `forall` just execute; no "nested call" handling
4. **Context via entity stack** - Entities provide the namespace for lookups

```
PostScript-style execution (CORRECT):
┌─────────────────────────────────────────────────────┐
│                    Single Stack                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  value  value  value  value  value  ...     │    │
│  └─────────────────────────────────────────────┘    │
│                        ▲                             │
│                        │                             │
│            All operations work here.                 │
│            forall? Same stack.                       │
│            executetable? Same stack.                 │
│            No "nested contexts". Just execute.       │
└─────────────────────────────────────────────────────┘

NOT PostScript (WRONG):
┌─────────────────────────────────────────────────────┐
│  ┌─────────┐   ┌─────────┐   ┌─────────┐           │
│  │ Stack 1 │   │ Stack 2 │   │ Stack 3 │           │
│  │ (outer) │──▶│ (inner) │──▶│ (nested)│           │
│  └─────────┘   └─────────┘   └─────────┘           │
│       ▲              ▲              ▲               │
│       │              │              │               │
│    sync           sync           sync               │
│                                                     │
│  Multiple stacks with syncing = WRONG               │
└─────────────────────────────────────────────────────┘
```

### DTState: Part of the Runtime

DTState (or VMState, or whatever the runtime calls its state) is **part of the Runtime layer**, not Rule Set Management. It contains:

- **Data Stack** - Operand stack for all values
- **Entity Stack** - Context stack for name lookups
- **Control Stack** - Call frames for decision tables

DTState is created fresh for each evaluation and destroyed afterward. It is NOT shared, NOT synced to another runtime, and NOT split across layers.

```
Session (coordination)
├── Rule Set reference (from Management layer)
└── Runtime instance (owns execution state)
    └── DTState / VMState
        ├── Data Stack
        ├── Entity Stack
        └── Control Stack
```

### Implications for Runtime Implementation

When implementing a new runtime:

1. **Own your state completely** - Don't reference another runtime's state
2. **Implement all operators** - Don't fall back to another runtime
3. **Execute bytecode completely** - No partial execution with handoffs
4. **Single stack model** - Follow PostScript semantics
5. **No syncing** - If you're syncing state, the architecture is wrong

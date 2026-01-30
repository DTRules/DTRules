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

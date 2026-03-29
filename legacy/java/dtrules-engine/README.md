# DTRules Engine

The core runtime engine for executing DTRules decision tables.

## Overview

This module provides:
- Decision table execution engine
- Entity model and type system
- Session and state management
- Data mapping (XML and Java auto-mapping)
- Tracing and debugging support

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     DTRules Engine                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Session   │  │   Entity    │  │  Decision Table     │  │
│  │   Manager   │  │   Model     │  │  Interpreter        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│         │                │                    │              │
│         ▼                ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    DTState                               ││
│  │  (Entity Stack, Execution Stack, Variables)             ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Key Packages

| Package | Description |
|---------|-------------|
| `com.dtrules.session` | Session and rule set management |
| `com.dtrules.entity` | Entity model and instances |
| `com.dtrules.decisiontables` | Decision table execution |
| `com.dtrules.interpreter` | Stack-based interpreter |
| `com.dtrules.mapping` | XML data mapping |
| `com.dtrules.automapping` | Java object auto-mapping |

## Core Classes

### RulesDirectory
Loads and manages rule set configurations:
```java
RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");
```

### RuleSet
Contains compiled entities, decision tables, and mappings:
```java
RuleSet rs = rd.getRuleSet("MyRules");
```

### IRSession
Execution context for a single evaluation:
```java
IRSession session = rs.newSession();
session.execute("DecisionTableName");
```

### DTState
Runtime state including entity stack and execution stack:
```java
DTState state = session.getState();
IREntity result = state.find("results");
```

## Building

```bash
mvn clean install
```

## Documentation

Legacy documentation available in `docs/`:
- `DTRules.doc` - Core engine documentation
- `OperatorList.doc` - Available operators reference
- `Overview_of_DTRules_and_EL.pdf` - System overview

## Dependencies

- No external runtime dependencies (standalone engine)
- OSGi bundle packaging for modular deployment

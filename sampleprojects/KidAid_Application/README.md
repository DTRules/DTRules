# KidAid_Application - KidAid Application Wrapper

A standalone application demonstrating programmatic DTRules execution.

## Overview

This project wraps the KidAid eligibility rules in a simple application, demonstrating the core pattern for loading and executing DTRules programmatically. It's a simpler example than ChipApp, focusing on the essential integration steps.

## What This Demonstrates

- Loading rules from configuration
- Creating rule sessions
- Mapping data to entities
- Executing decision tables
- Retrieving results
- Basic application structure

## Project Structure

```
KidAid_Application/
├── repository/
│   └── DTRules.xml               # Rules configuration
├── lib/
│   └── DTRulesNoSource.jar       # Compiled DTRules library
└── src/main/java/
    └── com/dtrules/SampleProject2/application/
        └── Application.java       # Main application
```

## Core Integration Pattern

The `Application.java` demonstrates the fundamental DTRules usage pattern:

```java
public class Application {
    public static void main(String[] args) throws Exception {
        // 1. Load rules directory
        RulesDirectory rd = new RulesDirectory(
            "repository/",
            "DTRules.xml"
        );

        // 2. Get the rule set
        RuleSet rs = rd.getRuleSet("SampleProject2");

        // 3. Create a session (thread-safe)
        IRSession session = rs.newSession();
        DTState state = session.getState();

        // 4. Get mapping for data loading
        Mapping mapping = session.getMapping();

        // 5. Create/load entities
        IREntity job = state.findEntity("job");
        // ... populate with data

        // 6. Execute decision table
        session.execute("Compute_Eligibility");

        // 7. Extract results
        IREntity results = state.find("results");
        // ... read result values

        // 8. Output results
        XMLPrinter xout = new XMLPrinter(System.out);
        session.printEntityReport(xout, false, state, "results", results);
    }
}
```

## Running This Sample

### Prerequisites

1. Build DTRules:
   ```bash
   cd /path/to/DTRules
   mvn clean install
   ```

2. Compile KidAid rules (if not already done):
   ```bash
   cd sampleprojects/KidAid
   mvn exec:java -Dexec.mainClass="com.dtrules.samples.kidaid.CompileKidAid"
   ```

### Run the Application

```bash
cd sampleprojects/KidAid_Application
mvn exec:java -Dexec.mainClass="com.dtrules.SampleProject2.application.Application"
```

## Key Classes Used

| Class | Purpose |
|-------|---------|
| `RulesDirectory` | Loads DTRules.xml configuration |
| `RuleSet` | Contains compiled rules and entity definitions |
| `IRSession` | Execution context for a single evaluation |
| `DTState` | Runtime state (entity instances, stack) |
| `Mapping` | Data transformation configuration |
| `IREntity` | Entity instance with attribute values |
| `XMLPrinter` | Formats output as XML |

## Session Lifecycle

```
RulesDirectory (load once)
       │
       ▼
   RuleSet (reuse)
       │
       ▼
   IRSession (create per evaluation)
       │
       ├── Load input data
       ├── Execute decision tables
       ├── Extract results
       │
       ▼
   Dispose session
```

## Thread Safety

- `RulesDirectory` and `RuleSet` are thread-safe and should be shared
- `IRSession` is NOT thread-safe - create one per thread/evaluation
- Multiple sessions can execute concurrently from the same `RuleSet`

## Comparison with ChipApp

| Feature | KidAid_Application | ChipApp |
|---------|-------------------|---------|
| Complexity | Simple | Full-featured |
| Threading | Single-threaded | Multi-threaded |
| Configuration | Minimal | External settings |
| Data Objects | Direct entity access | POJO mapping |
| Use Case | Learning/prototyping | Production template |

## Related Projects

- **KidAid** - The underlying rule set
- **ChipApp** - Full-featured application example

## DSL

Uses rules compiled with **EL (Expression Language)**.

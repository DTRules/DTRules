# ChipApp - CHIP Application Wrapper

A standalone application demonstrating how to embed DTRules in a production application.

## Overview

ChipApp wraps the CHIP eligibility rules in a complete application framework. It demonstrates multi-threaded rule execution, configuration management, result aggregation, and performance testing - everything needed for a production deployment.

## What This Demonstrates

- **Application integration** - Embedding DTRules in a runnable application
- **Multi-threaded execution** - Parallel case processing
- **Configuration management** - External settings via XML
- **Result aggregation** - Collecting and summarizing outcomes
- **Performance testing** - Comparing rule-based vs. native implementations
- **Data mapping** - Java objects to/from rule entities

## Project Structure

```
ChipApp/
├── DTRules.xml                    # Rules configuration
├── settings.xml                   # Application settings
├── xml/                           # Compiled rules
│   ├── CHIP_edd.xml
│   ├── CHIP_dt.xml
│   └── CHIP_map.xml
└── src/main/java/
    └── com/dtrules/samples/chipeligibility/app/
        ├── ChipApp.java           # Main application
        ├── EvaluateJob.java       # Job interface
        ├── EvaluateJobDTRules.java # DTRules implementation
        ├── EvaluateJobJava.java   # Pure Java implementation
        ├── EvaluateJobNone.java   # Baseline (no-op)
        ├── RunThread.java         # Thread management
        ├── GenCase.java           # Test case generator
        ├── LoadSettings.java      # Settings loader
        └── data/                  # Data objects
            ├── Case.java
            ├── Client.java
            ├── Income.java
            ├── Job.java
            ├── Relationship.java
            └── Result.java
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      ChipApp                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │  Settings   │  │  GenCase    │  │   RunThread     │ │
│  │  Loader     │  │  Generator  │  │   Pool          │ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
│                          │                   │          │
│                          ▼                   ▼          │
│                    ┌─────────────────────────────────┐  │
│                    │       EvaluateJob              │  │
│                    │  ┌──────────┬──────────────┐   │  │
│                    │  │ DTRules  │    Java      │   │  │
│                    │  │ Engine   │  (baseline)  │   │  │
│                    │  └──────────┴──────────────┘   │  │
│                    └─────────────────────────────────┘  │
│                                   │                      │
│                                   ▼                      │
│                         ┌─────────────────┐             │
│                         │   Results       │             │
│                         │  Aggregation    │             │
│                         └─────────────────┘             │
└─────────────────────────────────────────────────────────┘
```

## Key Components

### ChipApp.java
Main application that:
- Loads configuration from `settings.xml`
- Initializes the DTRules engine
- Spawns worker threads
- Processes cases in parallel
- Aggregates results (approved/denied counts)
- Reports performance metrics

### EvaluateJobDTRules.java
DTRules-based evaluation:
```java
// Create session
IRSession session = ruleSet.newSession();

// Map input data to entities
session.getAutoDataMap().mapToEntity(caseData, "case");

// Execute decision table
session.execute("Compute_Eligibility");

// Extract results
session.getAutoDataMap().mapFromEntity("results", result);
```

### EvaluateJobJava.java
Pure Java implementation of the same logic for:
- Performance comparison
- Validation of rule behavior
- Fallback option

### Data Objects
POJO classes matching the rule entities:
- `Case` - Container for clients
- `Client` - Applicant information
- `Income` - Income records
- `Result` - Eligibility outcome

## Running This Sample

### Prerequisites

1. Build DTRules:
   ```bash
   cd /path/to/DTRules
   mvn clean install
   ```

2. Compile CHIP rules (if not already done):
   ```bash
   cd sampleprojects/CHIP
   mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.CompileChip"
   ```

### Run the Application

```bash
cd sampleprojects/ChipApp
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.app.ChipApp"
```

## Configuration

### settings.xml
```xml
<settings>
  <threads>4</threads>           <!-- Worker thread count -->
  <cases>1000</cases>            <!-- Number of cases to process -->
  <trace>false</trace>           <!-- Enable tracing -->
  <evaluator>DTRules</evaluator> <!-- DTRules, Java, or None -->
</settings>
```

### DTRules.xml
Same configuration as the CHIP project - references the compiled rule files.

## Use Cases

### Production Integration
Use this as a template for integrating DTRules into your application:
1. Copy the application structure
2. Replace data objects with your domain
3. Update rule references
4. Configure threading based on workload

### Performance Testing
Compare execution strategies:
```xml
<evaluator>DTRules</evaluator>  <!-- Rule-based -->
<evaluator>Java</evaluator>     <!-- Native code -->
<evaluator>None</evaluator>     <!-- Baseline overhead -->
```

### Load Testing
Adjust case count and thread pool:
```xml
<threads>8</threads>
<cases>10000</cases>
```

## Related Projects

- **CHIP** - The underlying rule set
- **KidAid_Application** - Similar wrapper for KidAid

## DSL

Uses rules compiled with **EL (Expression Language)**.

# TestProject - Minimal Template

A minimal DTRules project template for creating new rule sets.

## Overview

This is the simplest possible DTRules project. Use it as a starting point when creating your own projects. It contains only the essential files needed to define and execute decision tables.

## What This Demonstrates

- Minimum required project structure
- Basic configuration
- Simple entity definition
- Minimal decision table

## Project Structure

```
TestProject/
├── DTRules.xml                    # Project configuration
├── DecisionTables/
│   └── test_dt.xls                # Minimal decision table
├── edd/
│   └── test_edd.xls               # Minimal entity definitions
├── xml/                           # Compiled XML (generated)
├── repository/                    # Packaged rules
└── src/main/java/
    └── com/dtrules/samples/test/
        ├── Compile_Test.java      # Compiles Excel → XML
        └── Test_Test.java         # Runs tests
```

## Entities

| Entity | Description |
|--------|-------------|
| `job` | Simple container with notes and things array |
| `thing` | Basic entity with a value field |

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

```bash
cd sampleprojects/TestProject
mvn exec:java -Dexec.mainClass="com.dtrules.samples.test.Compile_Test"
```

### Run Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.test.Test_Test"
```

## Creating Your Own Project

### Step 1: Copy This Template

```bash
cp -r sampleprojects/TestProject sampleprojects/MyProject
cd sampleprojects/MyProject
```

### Step 2: Update pom.xml

```xml
<artifactId>MyProject</artifactId>
<name>DTRules :: MyProject</name>
```

### Step 3: Update DTRules.xml

```xml
<RuleSet name="MyProject" source="file">
  <Entities        name="MyProject_edd.xml" />
  <Decisiontables  name="MyProject_dt.xml"  />
  <Map             name="MyProject_map.xml" />
</RuleSet>
```

### Step 4: Rename Excel Files

- `edd/test_edd.xls` → `edd/MyProject_edd.xls`
- `DecisionTables/test_dt.xls` → `DecisionTables/MyProject_dt.xls`

### Step 5: Define Your Entities

Edit `edd/MyProject_edd.xls` to define your data model.

### Step 6: Create Decision Tables

Edit `DecisionTables/MyProject_dt.xls` to define your business rules.

### Step 7: Update Java Classes

Rename and update the Java classes in `src/main/java/` to match your project.

## Configuration

`DTRules.xml`:
```xml
<DTRules>
  <compiler>EL</compiler>
  <RuleSet name="TestProject" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <WorkingDirectory>/temp</WorkingDirectory>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="Test_edd.xml" />
    <Decisiontables  name="Test_dt.xml"  />
    <Map             name="Test_map.xml" />
  </RuleSet>
</DTRules>
```

## Next Steps

After getting comfortable with this template:

1. **SyntaxTests** - Learn all EL language features
2. **CHIP** - See a real-world eligibility example
3. **eBook** - Explore multi-ruleset projects

## DSL

Uses **EL (Expression Language)** for conditions and actions.

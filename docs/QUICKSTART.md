# DTRules Quick Start Guide

This guide walks you through running your first DTRules project and understanding how the system works.

## Prerequisites

- Java 8 or higher installed
- Apache Maven 3.x installed
- Git (to clone the repository)

Verify your setup:
```bash
java -version
mvn -version
```

## Step 1: Clone and Build

```bash
# Clone the repository
git clone https://github.com/PaulSnow/DTRules.git
cd DTRules

# Build all modules
mvn clean install
```

The build compiles:
- The core rules engine
- Compiler utilities for Excel processing
- Domain Specific Language parsers
- All sample projects

## Step 2: Understand the CHIP Sample Project

We'll use the **CHIP** (Children's Health Insurance Program) sample project. This demonstrates eligibility determination logic - a common use case for rules engines.

### Project Layout

```
sampleprojects/CHIP/
├── DTRules.xml                           # Configuration file
├── edd/
│   └── ChipEligibility_edd.xls          # Entity Definition Document
├── DecisionTables/
│   └── ChipEligibility_dt.xls           # Decision Tables with rules
├── xml/                                  # Compiled XML (generated)
├── testfiles/TestScenarios/              # Test input files
└── src/main/java/.../
    ├── CompileChip.java                  # Compiles Excel → XML
    └── TestChip.java                     # Runs test cases
```

## Step 3: Compile the Decision Tables

Decision Tables are authored in Excel but must be compiled to XML before execution.

```bash
cd sampleprojects/CHIP
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.CompileChip"
```

This reads the Excel files and generates:
- `xml/CHIP_edd.xml` - Compiled entity definitions
- `xml/CHIP_dt.xml` - Compiled decision tables
- `xml/CHIP_map.xml` - Data mapping configuration

## Step 4: Run the Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.TestChip"
```

This:
1. Loads the compiled rules
2. Processes test cases from `testfiles/`
3. Executes the `Compute_Eligibility` decision table
4. Generates output in `testfiles/output/`

## Step 5: Examine the Key Files

### DTRules.xml - Configuration

```xml
<DTRules>
  <compiler>EL</compiler>                    <!-- Use Expression Language -->

  <RuleSet name="CHIP" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>  <!-- Where compiled XML goes -->
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>

    <Entities        name="CHIP_edd.xml" />
    <Decisiontables  name="CHIP_dt.xml"  />
    <Map             name="CHIP_map.xml" />
  </RuleSet>
</DTRules>
```

### Entity Definition Document (EDD)

Open `edd/ChipEligibility_edd.xls` in Excel. This defines:

- **Entities** - Data objects (e.g., `applicant`, `household`, `results`)
- **Attributes** - Properties of each entity with types
- **Relationships** - How entities relate to each other

Example attributes:
```
Entity: applicant
  - name (string)
  - age (integer)
  - income (double)
  - is_citizen (boolean)
```

### Decision Tables

Open `DecisionTables/ChipEligibility_dt.xls` in Excel. Each sheet is a decision table with:

- **Conditions** (rows starting with `c`) - Boolean expressions
- **Actions** (rows starting with `a`) - Operations to perform
- **Columns** - Each column is a rule; `Y`/`N`/`-` indicate condition matches

Example decision table structure:
```
           | Rule 1 | Rule 2 | Rule 3 |
-----------+--------+--------+--------+
c age < 19 |   Y    |   Y    |   N    |
c income   |   Y    |   N    |   -    |
  < limit  |        |        |        |
-----------+--------+--------+--------+
a eligible |   X    |        |        |
  = true   |        |        |        |
a reason = |        |   X    |   X    |
  "denied" |        |        |        |
```

## Step 6: Create Your Own Project

### Copy the Template

```bash
cp -r sampleprojects/TestProject sampleprojects/MyProject
cd sampleprojects/MyProject
```

### Update pom.xml

Edit `pom.xml`:
```xml
<artifactId>MyProject</artifactId>
<name>DTRules :: MyProject</name>
```

### Update DTRules.xml

Edit `DTRules.xml`:
```xml
<RuleSet name="MyProject" source="file">
  ...
  <Entities        name="MyProject_edd.xml" />
  <Decisiontables  name="MyProject_dt.xml"  />
  <Map             name="MyProject_map.xml" />
</RuleSet>
```

### Create Your Entity Definitions

Create `edd/MyProject_edd.xls` defining your data model.

### Create Your Decision Tables

Create `DecisionTables/MyProject_dt.xls` with your business rules.

### Create Compile and Test Classes

Copy and modify the Java classes from TestProject, updating package names and file references.

## Expression Language (EL) Basics

EL is the DSL used to write conditions and actions in decision tables.

### Data Types
- `integer` - Whole numbers
- `double` - Decimal numbers
- `string` - Text
- `boolean` - true/false
- `date` - Date values
- `array` - Collections

### Conditions (Examples)
```
applicant.age < 19
household.income <= poverty_limit * 2
applicant.is_citizen = true
member.status is in ["active", "pending"]
```

### Actions (Examples)
```
results.eligible = true
results.reason = "Income exceeds limit"
add member to eligible_members
remove applicant from pending_list
```

### Operators
- Comparison: `=`, `<>`, `<`, `>`, `<=`, `>=`
- Logical: `and`, `or`, `not`
- Arithmetic: `+`, `-`, `*`, `/`
- Collection: `is in`, `is not in`, `add to`, `remove from`

For complete EL documentation, see `sampleprojects/SyntaxTests/EL Documentation-1.odt`.

## Integrating with Your Application

### Basic Integration Pattern

```java
import com.dtrules.session.*;
import com.dtrules.entity.IREntity;

public class RulesEvaluator {

    private RulesDirectory rulesDirectory;
    private RuleSet ruleSet;

    public void initialize(String configPath) throws Exception {
        rulesDirectory = new RulesDirectory(configPath, "DTRules.xml");
        ruleSet = rulesDirectory.getRuleSet("MyProject");
    }

    public Result evaluate(Input input) throws Exception {
        // Create a new session for this evaluation
        IRSession session = ruleSet.newSession();

        // Load input data (from XML or map Java objects)
        session.loadXml(inputStream, "main");

        // Execute the decision table
        session.execute("Main_Decision_Table");

        // Extract results
        IREntity results = session.getState().find("results");
        // ... extract values from results entity
    }
}
```

### Using Auto-Mapping

DTRules can automatically map Java objects to entities:

```java
// Define mapping in your EDD
// Then in code:
AutoDataMap mapper = session.getAutoDataMap();
mapper.mapToEntity(myJavaObject, "entity_name");

// Execute rules
session.execute("DecisionTable");

// Map results back to Java
mapper.mapFromEntity("results", resultObject);
```

## Troubleshooting

### Build Failures

**Problem**: Maven build fails with dependency errors
```bash
# Try updating dependencies
mvn clean install -U
```

### Compilation Errors

**Problem**: Excel2XML fails to compile decision tables
- Check that Excel files are not open in another application
- Verify file paths in DTRules.xml are correct
- Check for syntax errors in EL expressions (see compiler output)

### Runtime Errors

**Problem**: Decision table not found
- Ensure you compiled the Excel files first
- Check that the table name matches exactly

**Problem**: Entity not found
- Verify entity is defined in the EDD
- Check spelling and case sensitivity

## Next Steps

Follow the recommended learning path:

1. **[SyntaxTests](../sampleprojects/SyntaxTests/)** - Learn all EL language features
2. **[CHIP](../sampleprojects/CHIP/)** - See a complete real-world example
3. **[ChipApp](../sampleprojects/ChipApp/)** - Learn application integration patterns
4. **[eBook](../sampleprojects/eBook/)** - Explore multi-ruleset projects with EBL
5. **[Sudoku](../sampleprojects/Sudoku/)** - Understand custom DSL creation

### Additional Resources

- [EL Language Reference](EL-REFERENCE.md) - Complete syntax documentation
- [Architecture Guide](ARCHITECTURE.md) - System design and components
- [API Guide](API-GUIDE.md) - Java integration examples

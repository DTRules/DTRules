# DTRules API Guide

This guide provides practical examples for integrating DTRules into your Java applications.

## Table of Contents

- [Basic Integration](#basic-integration)
- [Loading and Executing Rules](#loading-and-executing-rules)
- [Working with Entities](#working-with-entities)
- [Data Mapping](#data-mapping)
- [AutoDataMap (Java Objects)](#autodatamap-java-objects)
- [Error Handling](#error-handling)
- [Tracing and Debugging](#tracing-and-debugging)
- [Advanced Patterns](#advanced-patterns)

---

## Basic Integration

### Minimal Example

```java
import com.dtrules.session.*;
import com.dtrules.entity.IREntity;

public class BasicExample {

    public static void main(String[] args) throws Exception {
        // 1. Load rules configuration
        String path = System.getProperty("user.dir") + "/";
        RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");

        // 2. Get the rule set
        RuleSet rs = rd.getRuleSet("MyRules");

        // 3. Create a session
        IRSession session = rs.newSession();

        // 4. Create and populate entities
        DTState state = session.getState();
        IREntity job = state.findEntity("job");

        // 5. Execute decision table
        session.execute("Main_Decision_Table");

        // 6. Get results
        IREntity results = state.find("results");
        System.out.println("Result: " + results.get("status"));
    }
}
```

### Production Pattern

```java
public class RulesEngine {

    private final RulesDirectory rulesDirectory;
    private final RuleSet ruleSet;

    // Initialize once (e.g., at application startup)
    public RulesEngine(String configPath) throws RulesException {
        this.rulesDirectory = new RulesDirectory(configPath, "DTRules.xml");
        this.ruleSet = rulesDirectory.getRuleSet("MyRules");
    }

    // Call per evaluation (thread-safe if each call gets new session)
    public Result evaluate(Input input) throws RulesException {
        IRSession session = ruleSet.newSession();

        // Load input data
        loadInput(session, input);

        // Execute rules
        session.execute("Compute_Result");

        // Extract and return results
        return extractResult(session);
    }

    private void loadInput(IRSession session, Input input) throws RulesException {
        DTState state = session.getState();
        IREntity inputEntity = state.findEntity("input");
        inputEntity.put(session, RName.getRName("value"),
                       RInteger.getRIntegerValue(input.getValue()));
    }

    private Result extractResult(IRSession session) throws RulesException {
        DTState state = session.getState();
        IREntity results = state.find("results");

        Result result = new Result();
        result.setStatus(results.get("status").stringValue());
        result.setCode(results.get("code").intValue());
        return result;
    }
}
```

---

## Loading and Executing Rules

### RulesDirectory

```java
// Load from file system
RulesDirectory rd = new RulesDirectory("/path/to/rules/", "DTRules.xml");

// Load from classpath (using custom stream source)
rd.setStreamSource(new ClasspathStreamSource());

// Access rule set by name
RuleSet rs = rd.getRuleSet("MyRuleSet");

// List all available rule sets
for (RuleSet ruleSet : rd.getRuleSets()) {
    System.out.println("Rule set: " + ruleSet.getName());
}

// Store custom attributes
rd.setAttribute("version", "1.0");
String version = (String) rd.getAttribute("version");
```

### RuleSet

```java
// Create sessions
IRSession session = ruleSet.newSession();
IRSession taggedSession = ruleSet.newSession("session-123");

// Access configuration
String name = ruleSet.getName().stringValue();
String resourcePath = ruleSet.getResourcePath();

// Get entity/table names
ArrayList<String> entityNames = ruleSet.getEDDNames();
ArrayList<String> tableNames = ruleSet.getDTNames();
```

### Executing Decision Tables

```java
// Execute by name
session.execute("DecisionTableName");

// Execute at entry point
session.executeAt("entrypoint");

// Execute postfix code directly (advanced)
session.execute("10 20 + dup *");  // Stack: 900
```

---

## Working with Entities

### Creating Entities

```java
DTState state = session.getState();

// Get entity definition (not an instance)
IREntity entityDef = state.findEntity("client");

// Create new instance
IREntity client = session.createEntity(null, "client");

// Create with specific ID
IREntity client2 = session.createEntity("client-001", "client");
```

### Setting Attributes

```java
// Using RName (faster for repeated access)
RName ageName = RName.getRName("age");
client.put(session, ageName, RInteger.getRIntegerValue(25));

// Using string name
client.put(session, RName.getRName("name"),
           RString.newRString("John Doe"));

// Setting different types
client.put(session, RName.getRName("income"),
           RDouble.getRDoubleValue(50000.00));

client.put(session, RName.getRName("active"),
           RBoolean.getRBoolean(true));

client.put(session, RName.getRName("birthdate"),
           RDate.getRDate(session, "1990-05-15"));

// Setting arrays
RArray addresses = new RArray(0);
addresses.add(address1);
addresses.add(address2);
client.put(session, RName.getRName("addresses"), addresses);
```

### Getting Attributes

```java
// Get as IRObject (generic)
IRObject value = client.get("age");

// Convert to specific types
int age = client.get("age").intValue();
String name = client.get("name").stringValue();
double income = client.get("income").doubleValue();
boolean active = client.get("active").booleanValue();
Date birthdate = client.get("birthdate").dateValue();

// Get array
RArray addresses = (RArray) client.get("addresses");
for (int i = 0; i < addresses.size(); i++) {
    IREntity address = (IREntity) addresses.get(i);
    String city = address.get("city").stringValue();
}

// Check for null
IRObject phone = client.get("phone");
if (phone == null || phone instanceof RNull) {
    // Handle missing value
}
```

### Entity Metadata

```java
// Get all attribute names
Iterator<RName> attrs = client.getAttributeIterator();
while (attrs.hasNext()) {
    RName attrName = attrs.next();
    System.out.println("Attribute: " + attrName);
}

// Check if attribute exists
boolean hasPhone = client.containsAttribute(RName.getRName("phone"));

// Get attribute definition
REntityEntry entry = client.getEntry(RName.getRName("age"));
RType type = entry.getType();           // INTEGER
boolean writable = entry.isWritable();  // true
String comment = entry.getComment();    // "Client's age in years"
```

### Finding Entities

```java
DTState state = session.getState();

// Find by name (returns first match)
IREntity results = state.find("results");

// Find entity definition
IREntity clientDef = state.findEntity("client");

// Find in context stack
IREntity currentEntity = state.entitypeek();
```

---

## Data Mapping

### XML Mapping

```java
// Load data from XML file
FileInputStream input = new FileInputStream("testcase.xml");
Mapping mapping = session.getMapping();
DataMap dataMap = session.getDataMap(mapping, "main");
dataMap.loadXML(input);
input.close();

// Output results to XML
FileOutputStream output = new FileOutputStream("results.xml");
XMLPrinter printer = new XMLPrinter(output);
session.printEntityReport(printer, false, false, state, "results",
                          state.find("results"));
output.close();
```

### Mapping Configuration

In your mapping XML file:
```xml
<mapping>
  <!-- Map XML elements to entities -->
  <entity name="client" tag="Client">
    <attribute name="name" tag="Name" />
    <attribute name="age" tag="Age" type="integer" />
    <attribute name="income" tag="Income" type="double" />
  </entity>

  <!-- Map nested structures -->
  <entity name="address" tag="Address" parent="client" attribute="addresses">
    <attribute name="street" tag="Street" />
    <attribute name="city" tag="City" />
  </entity>
</mapping>
```

---

## AutoDataMap (Java Objects)

AutoDataMap uses reflection to map Java POJOs to DTRules entities.

### Java Domain Classes

```java
// Java classes must have public getters/setters
public class Client {
    private String name;
    private int age;
    private double income;
    private List<Address> addresses;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public int getAge() { return age; }
    public void setAge(int age) { this.age = age; }

    public double getIncome() { return income; }
    public void setIncome(double income) { this.income = income; }

    public List<Address> getAddresses() { return addresses; }
    public void setAddresses(List<Address> addresses) {
        this.addresses = addresses;
    }
}

public class Address {
    private String street;
    private String city;

    // Getters and setters...
}
```

### Mapping Java to Entities

```java
// Create Java object
Client client = new Client();
client.setName("John Doe");
client.setAge(30);
client.setIncome(75000.00);

// Get AutoDataMap
AutoDataMapDef mapDef = ruleSet.getAutoDataMapDef();
AutoDataMap mapper = ruleSet.getAutoDataMap(session, client, mapDef);

// Map Java object to entity
mapper.mapToEntity();

// Execute rules
session.execute("Evaluate_Client");

// Map results back to Java
Result result = new Result();
mapper.mapFromEntity("results", result);
```

### AutoDataMap with Collections

```java
// Java object with nested collections
Case caseObj = new Case();
caseObj.setClients(Arrays.asList(client1, client2, client3));

// Map entire object graph
AutoDataMap mapper = ruleSet.getAutoDataMap(session, caseObj, mapDef);
mapper.mapToEntity();

// After rule execution, map back
mapper.mapFromEntity("case", caseObj);

// Collections are updated in place
List<Client> processedClients = caseObj.getClients();
```

---

## Error Handling

### RulesException

```java
try {
    session.execute("Decision_Table");
} catch (RulesException e) {
    // Get error details
    String message = e.getMessage();
    Throwable cause = e.getCause();

    // Log with context
    System.err.println("Rule execution failed: " + message);
    System.err.println("Table: " + state.getCurrentTable());
    System.err.println("Section: " + state.getCurrentTableSection());

    // Rethrow or handle
    throw new BusinessException("Eligibility check failed", e);
}
```

### Validation Errors

```java
// Collect validation errors during execution
IREntity errors = state.find("errors");
if (errors != null) {
    RArray errorList = (RArray) errors.get("messages");
    for (int i = 0; i < errorList.size(); i++) {
        String errorMsg = errorList.get(i).stringValue();
        System.err.println("Validation error: " + errorMsg);
    }
}
```

### Pre-Execution Validation

```java
// Check required entities exist
IREntity client = state.find("client");
if (client == null) {
    throw new IllegalStateException("Client entity must be loaded");
}

// Check required attributes
IRObject age = client.get("age");
if (age == null || age instanceof RNull) {
    throw new IllegalArgumentException("Client age is required");
}
```

---

## Tracing and Debugging

### Enable Debug Mode

```java
DTState state = session.getState();
state.setDebug(true);  // Enable debug output

// Set custom output streams
state.setOut(System.out);  // Standard output
state.setErr(System.err);  // Error output

session.execute("Decision_Table");
```

### Trace Execution

```java
// Using test harness for comprehensive tracing
import com.dtrules.testsupport.ATestHarness;

public class MyTest extends ATestHarness {

    @Override
    public boolean Trace() { return true; }   // Enable tracing

    @Override
    public boolean Console() { return true; } // Output to console

    @Override
    public boolean Verbose() { return true; } // Detailed output

    public static void main(String[] args) throws Exception {
        MyTest test = new MyTest();
        test.load("xml/testParms.xml");
        test.runTests();
    }
}
```

### Inspecting State

```java
// Print current stack contents
System.out.println("Data stack depth: " + state.ddepth());
for (int i = 0; i < state.ddepth(); i++) {
    System.out.println("  [" + i + "]: " + state.getds(i));
}

// Print entity context stack
System.out.println("Entity stack depth: " + state.edepth());

// Print entity contents
session.dump((REntity) state.find("client"));
```

### Decision Table Coverage

```java
import com.dtrules.testsupport.Coverage;

// After running tests
Coverage coverage = new Coverage(ruleSet);
coverage.compute(traceFiles);

// Generate coverage report
coverage.printReport(System.out);

// Check specific table coverage
double tablePercent = coverage.getTableCoverage("Main_Table");
```

---

## Advanced Patterns

### Multi-Threaded Execution

```java
public class ParallelEvaluator {
    private final RuleSet ruleSet;
    private final ExecutorService executor;

    public ParallelEvaluator(RuleSet ruleSet, int threads) {
        this.ruleSet = ruleSet;
        this.executor = Executors.newFixedThreadPool(threads);
    }

    public List<Result> evaluateAll(List<Input> inputs) {
        List<Future<Result>> futures = inputs.stream()
            .map(input -> executor.submit(() -> evaluate(input)))
            .collect(Collectors.toList());

        return futures.stream()
            .map(this::getResult)
            .collect(Collectors.toList());
    }

    private Result evaluate(Input input) throws RulesException {
        // Each thread gets its own session
        IRSession session = ruleSet.newSession();
        loadInput(session, input);
        session.execute("Evaluate");
        return extractResult(session);
    }

    private Result getResult(Future<Result> future) {
        try {
            return future.get();
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }
}
```

### Caching Rule Sets

```java
public class RuleSetCache {
    private static final Map<String, RuleSet> cache =
        new ConcurrentHashMap<>();

    public static RuleSet get(String name, String configPath)
            throws RulesException {
        return cache.computeIfAbsent(name, k -> {
            try {
                RulesDirectory rd = new RulesDirectory(configPath,
                                                       "DTRules.xml");
                return rd.getRuleSet(name);
            } catch (RulesException e) {
                throw new RuntimeException(e);
            }
        });
    }

    public static void clear() {
        cache.clear();
    }
}
```

### Custom Date Parser

```java
// Implement custom date parsing
IDateParser customParser = new IDateParser() {
    private SimpleDateFormat format =
        new SimpleDateFormat("yyyy-MM-dd");

    @Override
    public Date parse(String dateString) throws ParseException {
        return format.parse(dateString);
    }

    @Override
    public String format(Date date) {
        return format.format(date);
    }
};

// Set on session
session.setDateParser(customParser);
```

### Dynamic Table Selection

```java
// Choose table based on business logic
String tableName;
switch (input.getType()) {
    case "INDIVIDUAL":
        tableName = "Evaluate_Individual";
        break;
    case "BUSINESS":
        tableName = "Evaluate_Business";
        break;
    default:
        tableName = "Evaluate_Default";
}

session.execute(tableName);
```

### Batch Processing

```java
public class BatchProcessor {

    public void process(List<Case> cases, String outputDir)
            throws Exception {
        RuleSet ruleSet = getRuleSet();

        for (int i = 0; i < cases.size(); i++) {
            Case c = cases.get(i);

            // Create fresh session per case
            IRSession session = ruleSet.newSession();

            try {
                // Load case data
                loadCase(session, c);

                // Execute rules
                session.execute("Process_Case");

                // Write results
                writeResults(session, outputDir, i);

            } catch (RulesException e) {
                System.err.println("Failed case " + i + ": " + e.getMessage());
                writeError(outputDir, i, e);
            }
        }
    }

    private void writeResults(IRSession session, String dir, int index)
            throws Exception {
        String filename = dir + "/result_" + index + ".xml";
        FileOutputStream out = new FileOutputStream(filename);
        XMLPrinter printer = new XMLPrinter(out);
        DTState state = session.getState();
        session.printEntityReport(printer, false, false, state,
                                  "results", state.find("results"));
        out.close();
    }
}
```

---

## Quick Reference

### Key Classes

| Class | Purpose | Thread Safe |
|-------|---------|-------------|
| `RulesDirectory` | Load configuration | Yes |
| `RuleSet` | Rule definitions | Yes |
| `IRSession` | Execution context | No |
| `DTState` | Runtime state | No |
| `IREntity` | Data object | No |
| `Mapping` | XML mapping | Yes |
| `AutoDataMap` | Java mapping | No |

### Common Operations

```java
// Initialize
RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");
RuleSet rs = rd.getRuleSet("MyRules");

// Execute
IRSession session = rs.newSession();
session.execute("TableName");

// Get entity
DTState state = session.getState();
IREntity entity = state.find("entityName");

// Get attribute
String value = entity.get("attributeName").stringValue();

// Set attribute
entity.put(session, RName.getRName("attr"), RString.newRString("value"));

// Output XML
XMLPrinter printer = new XMLPrinter(outputStream);
session.printEntityReport(printer, false, false, state, "tag", entity);
```

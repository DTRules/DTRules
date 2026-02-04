# DTRules DSL Modules

Domain Specific Language implementations for DTRules.

## Modules

| Module | Description | Parser |
|--------|-------------|--------|
| **el** | Expression Language - Standard DSL | ANTLR 4 |
| **ebl** | Entity Business Language - Enhanced DSL | ANTLR 4 |
| **sudoku_language** | Sudoku DSL - Custom example | JFlex/CUP |

## Expression Language (EL)

The primary DSL for writing conditions and actions in decision tables. Features:
- Human-readable syntax
- Strong typing with entity references
- Natural language operators ("is greater than", "is equal to")
- Array and collection operations
- Date handling

### Example Conditions
```
client.age < 18
income is greater than poverty_level * 1.5
applicant.status is in ["active", "pending"]
```

### Example Actions
```
set result.eligible = true
add client to approved_list
set reason = "Approved - meets all criteria"
```

## Entity Business Language (EBL)

Extends EL with additional operators for complex entity queries:
- `FIND` - Entity lookup operations
- `ISWITHIN` - Range and containment tests

## Building

```bash
# Build all DSL modules
cd dsl
mvn clean install

# Build individual module
mvn clean install -pl el
```

## ANTLR 4 Migration

The EL and EBL modules have been migrated from JFlex/CUP to ANTLR 4 for improved maintainability and tooling.

See [ANTLR_MIGRATION.md](ANTLR_MIGRATION.md) for details.

## Testing

```bash
# Run EL tests (128 parameterized tests)
cd el
mvn test
```

## Usage

Compilers are configured in `DTRules.xml`:

```xml
<compiler>EL</compiler>
```

Or use the new ANTLR implementation directly:

```xml
<compileralias name="EL">com.dtrules.compiler.el.ELAntlr</compileralias>
```

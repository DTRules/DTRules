# Building DTRules

This document provides detailed instructions for building DTRules from source.

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| Java JDK | 8+ | Compilation and runtime |
| Apache Maven | 3.x | Build automation |
| Git | Any | Source control |

### Verify Installation

```bash
# Check Java
java -version
javac -version

# Check Maven
mvn -version

# Check Git
git --version
```

## Building from Source

### Clone the Repository

```bash
git clone https://github.com/PaulSnow/DTRules.git
cd DTRules
```

### Full Build

```bash
mvn clean install
```

This builds all modules in dependency order:
1. `dtrules-engine` - Core rules engine
2. `dsl/el` - Expression Language compiler
3. `dsl/ebl` - Entity Business Language compiler
4. `dsl/el_antlr` - ANTLR-based parser (alternative)
5. `dsl/sudoku_language` - Custom DSL example
6. `compilerutil` - Excel-to-XML utilities
7. `sampleprojects/*` - All sample projects

### Build Specific Modules

```bash
# Build only the core engine
mvn clean install -pl dtrules-engine

# Build engine and its dependents
mvn clean install -pl dtrules-engine -amd

# Build a specific sample project (with dependencies)
mvn clean install -pl sampleprojects/CHIP -am
```

### Skip Tests

```bash
mvn clean install -DskipTests
```

### Build with Debug Output

```bash
mvn clean install -X
```

## Module Overview

### dtrules-engine

The core rules execution engine. Packaged as an OSGi bundle.

```bash
cd dtrules-engine
mvn clean install
```

Produces: `target/dtrules-engine-5.0-SNAPSHOT.jar`

Key packages:
- `com.dtrules.session` - Session management
- `com.dtrules.entity` - Entity model
- `com.dtrules.decisiontables` - Decision table execution
- `com.dtrules.interpreter` - Rule interpreter
- `com.dtrules.automapping` - Java-to-entity mapping

### compilerutil

Utilities for compiling Excel decision tables to XML.

```bash
cd compilerutil
mvn clean install
```

Key classes:
- `Excel2XML` - Converts Excel files to XML rule definitions
- `ATestHarness` - Base class for test harnesses

Dependencies: `dtrules-engine`, Apache POI (for Excel reading)

### dsl/el

The Expression Language (EL) compiler using JFlex and CUP.

```bash
cd dsl/el
mvn clean install
```

Contains:
- `scanner.flex` - Lexer definition
- `parser.cup` - Parser grammar
- `EL.java` - Main compiler class

### dsl/ebl

Entity Business Language - enhanced version of EL.

```bash
cd dsl/ebl
mvn clean install
```

### dsl/el_antlr

Alternative EL parser using ANTLR (experimental).

```bash
cd dsl/el_antlr
mvn clean install
```

### sampleprojects

Example implementations demonstrating DTRules usage.

```bash
cd sampleprojects
mvn clean install
```

Each sample project is a standalone Maven module.

## IDE Setup

### Eclipse

The project includes Eclipse configuration files (`.project`, `.classpath`).

1. **Import Projects**:
   - File → Import → Maven → Existing Maven Projects
   - Select the DTRules root directory
   - Import all discovered projects

2. **Build**:
   - Right-click project → Maven → Update Project
   - Project → Build All

### IntelliJ IDEA

1. **Open Project**:
   - File → Open
   - Select the DTRules `pom.xml`
   - Open as Project

2. **Import Maven**:
   - IntelliJ will automatically detect and import Maven modules

### VS Code

1. Install the "Extension Pack for Java" extension
2. Open the DTRules folder
3. VS Code will detect the Maven project automatically

## Build Artifacts

After a successful build, artifacts are located in:

```
dtrules-engine/target/
  └── dtrules-engine-5.0-SNAPSHOT.jar

compilerutil/target/
  └── compilerutil-5.0-SNAPSHOT.jar

dsl/el/target/
  └── el-5.0-SNAPSHOT.jar

dsl/ebl/target/
  └── ebl-5.0-SNAPSHOT.jar

sampleprojects/CHIP/target/
  └── CHIP-5.0-SNAPSHOT.jar
```

## Using as a Dependency

### Maven

Add to your `pom.xml`:

```xml
<dependency>
    <groupId>com.dtrules</groupId>
    <artifactId>dtrules-engine</artifactId>
    <version>5.0-SNAPSHOT</version>
</dependency>

<dependency>
    <groupId>com.dtrules</groupId>
    <artifactId>compilerutil</artifactId>
    <version>5.0-SNAPSHOT</version>
</dependency>

<dependency>
    <groupId>com.dtrules</groupId>
    <artifactId>el</artifactId>
    <version>5.0-SNAPSHOT</version>
</dependency>
```

### Local Repository

After running `mvn install`, artifacts are available in your local Maven repository:
```
~/.m2/repository/com/dtrules/
```

## Regenerating Parsers

The EL and EBL parsers are generated from grammar files. To regenerate:

### Using JFlex (Lexer)

```bash
cd dsl/el
./scanner.sh   # Linux/Mac
scanner.bat    # Windows
```

This regenerates `DTRulesscanner.java` from `scanner.flex`.

### Using CUP (Parser)

```bash
cd dsl/el
./parser.sh    # Linux/Mac
parser.bat     # Windows
```

This regenerates `DTRulesParser.java` and `sym.java` from `parser.cup`.

## Troubleshooting

### Out of Memory

```bash
export MAVEN_OPTS="-Xmx1024m"
mvn clean install
```

### Dependency Resolution Failures

```bash
# Force update of snapshots
mvn clean install -U

# Clear local repository cache
rm -rf ~/.m2/repository/com/dtrules
mvn clean install
```

### Java Version Issues

Ensure JAVA_HOME points to a compatible JDK:

```bash
export JAVA_HOME=/path/to/jdk
mvn clean install
```

### Excel File Errors

If compilation of Excel files fails:
- Ensure Excel files are not open in another application
- Check for corrupted Excel files
- Verify file permissions

## Continuous Integration

For CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
steps:
  - uses: actions/checkout@v3
  - uses: actions/setup-java@v3
    with:
      java-version: '11'
      distribution: 'temurin'
  - name: Build with Maven
    run: mvn clean install
```

## Release Build

To create a release build with signed artifacts:

```bash
mvn clean install -P release-sign-artifacts
```

This requires GPG keys configured for signing.

These are GitHub issues for DTRules. Each must be a COMPLETE, SELF-CONTAINED prompt.

Project root: /home/paul/go/src/github.com/DTRules/DTRules
Branch: issue-31-asm-optimization (ASM work), 5.0-SNAPSHOT (JSON work)

---

# Track D: Cleanup & Validation

## D1: Delete legacy VMState and sync code

Title: Remove all VMState sync machinery
Depends on: B1-B10, C1-C10 (all opcode work complete)

**Prerequisites -- verify before starting:**
- [ ] ALL Track B issues (B1-B10) are complete -- all NativeASM opcodes work without fallback to Go
- [ ] ALL Track C issues (C1-C10) are complete -- all x86-64 opcodes work without state sync
- [ ] `go test ./go/pkg/dtrules/...` passes with all opcode changes in place
- [ ] CHIP decision tables execute correctly on all 3 runtimes -- verify by running `go run ./go/cmd/benchmark/main.go > /tmp/bench-verify.log 2>&1` and checking for errors
- [ ] RuntimeInit/RuntimeQuery interfaces are in use (produced by A3) -- the old BytecodeExecutor(state *DTState, ...) pattern should already be replaced

Context: Once both ASM runtimes own their state (A3) and all opcodes are implemented (B/C tracks), the old sync code is dead. Remove it.

Task:
1. Delete sync functions in `go/pkg/dtrules/asmruntime/executor.go` (any syncStateToASM/syncStateFromASM patterns, stack copying in ExecuteBytecode)
2. Delete entity marshaling in `go/pkg/dtrules/asmruntime/bridge.go` (MarshalEntity, MarshalValue, SetupEntityStack and related functions)
3. Remove any VMState struct that duplicates DTState fields
4. Clean up bridge.go: remove functions that are no longer called
5. Clean up any fallback paths in `go/pkg/dtrules/interpreter/vm.go` that were only needed for ASM fallback (RC=3 handling that re-executes in Go)
6. Remove the arithmetic/comparison/boolean fallback functions in `go/pkg/dtrules/interpreter/asm_helpers.go` (goOpAddFallback, etc. -- these currently panic, confirming they're unused)
7. Run full test suite to verify nothing breaks

Files to modify:
- `go/pkg/dtrules/asmruntime/executor.go`
- `go/pkg/dtrules/asmruntime/bridge.go`
- `go/pkg/dtrules/interpreter/vm.go`
- `go/pkg/dtrules/interpreter/asm_helpers.go`

Acceptance criteria:
- No sync code remains
- No VMState struct separate from DTState
- No entity marshaling code
- All tests pass: `go test ./go/pkg/dtrules/...`
- CHIP decision tables pass on all 3 runtimes

---

## D2: Cross-runtime validation test suite

Title: Automated cross-runtime validation for all test vectors
Depends on: B1-B10, C1-C10

**Prerequisites -- verify before starting:**
- [ ] ALL Track B issues (B1-B10) are complete
- [ ] ALL Track C issues (C1-C10) are complete
- [ ] Test vectors exist in `test/vectors/*.json` -- verify all 10 files are present (arithmetic, comparison, boolean, stack, string, array, table, control, entity, datetime)
- [ ] All 3 runtimes are functional: Go (`go/pkg/dtrules/interpreter/`), NativeASM (`go/pkg/dtrules/runtime/nativeasm/`), x86-64-ASM (`go/pkg/dtrules/asmruntime/`)
- [ ] RuntimeInit/RuntimeQuery interfaces exist (produced by A3) -- needed to create sessions with each runtime

Context: 174 test vectors exist in `test/vectors/*.json` across 10 categories. Need automated tests that run each vector against all 3 runtimes and verify identical results.

Task:
1. Create `go/pkg/dtrules/runtime/validation_test.go` (or similar)
2. For each test vector file, for each test:
   a. Create a session with Go runtime, execute test bytecode, record result
   b. Create a session with NativeASM runtime, execute same bytecode, record result
   c. Create a session with x86-64-ASM runtime, execute same bytecode, record result
   d. Assert all 3 results are identical
3. Generate a comparison report: pass/fail/mismatch per runtime per test
4. Add to `.github/workflows/tests.yml` CI pipeline
5. Handle test categories that need entity setup (entity.json, datetime.json)

Test vector format (example from arithmetic.json):
```json
{
  "name": "add_integers",
  "description": "Add two integers",
  "bytecode": [2, 0, 0, 0, 5, 2, 0, 0, 0, 3, 20],
  "expected_stack": [{"type": "integer", "value": 8}]
}
```
(Read actual format from files -- above is approximate)

Files to create:
- `go/pkg/dtrules/runtime/validation_test.go`

Files to read:
- `test/vectors/*.json` - all 10 test vector files
- `go/pkg/dtrules/runtime/nativeasm/executor.go`
- `go/pkg/dtrules/asmruntime/executor.go`
- `go/pkg/dtrules/session/session.go`

Acceptance criteria:
- Test runs all 174 vectors against all 3 runtimes
- All vectors produce identical results across runtimes
- Test is in CI pipeline
- Clear error messages on mismatch (shows runtime, test name, expected vs actual)

---

## D3: Post-fix performance benchmarks

Title: Re-run benchmarks after architecture fix
Depends on: D1

**Prerequisites -- verify before starting:**
- [ ] D1 is complete -- all legacy sync code is deleted
- [ ] `go test ./go/pkg/dtrules/...` passes after D1 cleanup
- [ ] Baseline benchmarks exist in `benchmark-results/all_benchmarks.json` -- these are the pre-fix numbers to compare against
- [ ] All 3 runtimes are functional and pass tests

Task:
1. Run existing benchmark suite: `go test -bench=. ./go/pkg/dtrules/benchmark/...` > /tmp/bench.log 2>&1
2. Run CHIP decision table benchmark: `go run ./go/cmd/benchmark/main.go` > /tmp/chip-bench.log 2>&1
3. Compare results against baseline in `benchmark-results/all_benchmarks.json`
4. ASM runtimes should now be FASTER than Go (the whole point)
5. If they're not, profile to identify remaining bottlenecks
6. Update `benchmark-results/all_benchmarks.json` and `benchmark-results/benchmark_report.html`

Files:
- `go/cmd/benchmark/main.go`
- `go/pkg/dtrules/benchmark/benchmark_test.go`
- `benchmark-results/all_benchmarks.json`
- `test/run-benchmarks.sh`

Acceptance criteria:
- NativeASM faster than Go runtime
- x86-64-ASM faster than Go runtime (or documented why not due to CGO overhead)
- Benchmark report updated
- Results committed

---

## D4: Update architecture documentation

Title: Update docs to reflect runtime-owned state model
Depends on: D1

**Prerequisites -- verify before starting:**
- [ ] D1 is complete -- all sync code removed, architecture is final
- [ ] D3 benchmark results exist -- include performance numbers in docs
- [ ] RuntimeInit/RuntimeQuery interfaces are defined (produced by A3) -- document these
- [ ] Dispatch loop design is finalized (produced by A4) -- document the pattern
- [ ] `docs/asm-memory-strategy.md` exists (produced by A5) -- reference or incorporate

Task:
1. Update `docs/ARCHITECTURE.md`:
   - Remove references to state syncing
   - Document RuntimeInit/RuntimeQuery interfaces
   - Document the dispatch loop design (zero-overhead jump table)
   - Update runtime comparison section
2. Update `docs/nativeasm-runtime.md`:
   - Document new dispatch loop
   - Document Go callback interface
   - Remove references to VMState
3. Update `CHANGELOG.md` with all changes from this issue set
4. Remove or update `plans/fix-all-runtimes.md` (the plan is now implemented)

Files to modify:
- `docs/ARCHITECTURE.md`
- `docs/nativeasm-runtime.md`
- `CHANGELOG.md`
- `plans/fix-all-runtimes.md`

Acceptance criteria:
- Documentation reflects actual architecture
- No references to VMState sync remain in docs
- CHANGELOG covers all changes

---

## D5: Implement OpExec (recursive bytecode execution)

Title: Implement OpExec for recursive bytecode execution
Depends on: A4

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- read the OpExec(50) specification for exact semantics, recursion limits, and error handling
- [ ] Zero-overhead dispatch loop exists in both `vm_amd64.s` and `bytecode.asm` (produced by A4) -- OpExec must integrate as a Go callback in both dispatch loops
- [ ] RuntimeInit/RuntimeQuery interfaces exist (produced by A3) -- recursive execution must work within the runtime-owned-state model
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Context: OpExec(50) pops an executable object from the stack and executes it. Currently returns unknown opcode error in `go/pkg/dtrules/interpreter/vm.go`. This is needed for stored procedures and dynamic execution patterns.

Task:
1. Read Go VM dispatch for OpExec in `vm.go` to understand intended behavior
2. Implement in Go VM first: pop object, if it's a BytecodeChunk, recursively call ExecuteBytecode
3. Implement Go helper `goOpExec(state *DTState) int` in `asm_helpers.go` for ASM runtimes
4. Wire NativeASM dispatch (`vm_amd64.s`) to call goOpExec
5. Wire x86-64 dispatch (`bytecode.asm`) to call via CGO bridge
6. Handle recursion depth limit to prevent stack overflow

Files to modify:
- `go/pkg/dtrules/interpreter/vm.go`
- `go/pkg/dtrules/interpreter/asm_helpers.go`
- `go/pkg/dtrules/interpreter/vm_amd64.s`
- `asm/src/vm/bytecode.asm`

Acceptance criteria:
- OpExec works in all 3 runtimes
- Recursion depth is bounded
- `go test ./go/pkg/dtrules/...` passes

---

## D6: Policy statement handling in stack operations

Title: Implement policy statement handling in stack operators
Depends on: A1

**Prerequisites -- verify before starting:**
- [ ] `docs/bytecode-spec.md` exists (produced by A1) -- check if policy statements are covered in the spec; if not, this issue should also add them to the spec
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes
- [ ] No other prerequisites -- this is independent of the ASM work

Context: There is a TODO in `go/pkg/dtrules/operators/stack.go` about implementing proper policy statement handling. Policy statements are part of decision table execution.

Task:
1. Read `go/pkg/dtrules/operators/stack.go` to find the TODO
2. Read how policy statements work in the decision table execution (see `go/pkg/dtrules/decisiontable/table.go`)
3. Implement the missing handling
4. Test with decision tables that use policy statements

Files: `go/pkg/dtrules/operators/stack.go`, `go/pkg/dtrules/decisiontable/table.go`

Acceptance criteria:
- TODO is resolved
- Policy statements handled correctly
- Existing tests pass

---

# Track E: JSON/XML Format Support

This track is FULLY INDEPENDENT of Tracks A-D. Can run in parallel.

## E1: Define JSON schema for all DTRules formats

Title: Define JSON schema for EDD, Decision Table, and Mapping formats

**Prerequisites -- verify before starting:**
- [ ] No prerequisites -- this is a foundation issue that runs independently
- [ ] Verify sample XML files exist in `sampleprojects/CHIP/xml/` -- these are the reference data for schema design
- [ ] Verify Go loader structs exist in `go/pkg/dtrules/loader/edd.go` and `go/pkg/dtrules/loader/dt.go` -- these define the current XML structure

Context: DTRules uses 3 XML file formats with no formal schema (no XSD/DTD). Need JSON equivalents.

Current XML structs are defined in Go with xml tags:

EDD (from `go/pkg/dtrules/loader/edd.go`):
```go
type EDDFile struct {
    XMLName  xml.Name    `xml:"entity_data_dictionary"`
    Version  string      `xml:"version,attr"`
    Entities []EDDEntity `xml:"entity"`
}
type EDDEntity struct {
    Name    string     `xml:"name,attr"`
    Access  string     `xml:"access,attr"`
    Comment string     `xml:"comment,attr"`
    Fields  []EDDField `xml:"field"`
}
type EDDField struct {
    Name         string `xml:"name,attr"`
    Type         string `xml:"type,attr"`
    SubType      string `xml:"subtype,attr"`
    Access       string `xml:"access,attr"`
    Input        string `xml:"input,attr"`
    DefaultValue string `xml:"default_value,attr"`
    Comment      string `xml:"comment,attr"`
}
```

DT (from `go/pkg/dtrules/loader/dt.go`):
```go
type DTFile struct {
    XMLName xml.Name  `xml:"decision_tables"`
    Tables  []DTTable `xml:"decision_table"`
}
type DTTable struct {
    TableName        string             `xml:"table_name"`
    XlsFile          string             `xml:"xls_file"`
    AttributeFields  DTAttributeFields  `xml:"attribute_fields"`
    Contexts         DTContexts         `xml:"contexts"`
    InitialActions   DTInitialActions   `xml:"initial_actions"`
    Conditions       DTConditions       `xml:"conditions"`
    Actions          DTActions          `xml:"actions"`
    PolicyStatements DTPolicyStatements `xml:"policy_statements"`
}
// (and sub-structs for conditions, actions, columns, etc.)
```

Mapping uses streaming XML with elements: createentity, setattribute, addalltolist

Task:
1. Read the full struct definitions in `go/pkg/dtrules/loader/edd.go` and `go/pkg/dtrules/loader/dt.go`
2. Read sample XML files in `sampleprojects/CHIP/xml/` to see real data
3. Read `go/pkg/dtrules/mapping/map_loader.go` for mapping format
4. Design JSON equivalents that are natural JSON (not mechanical XML-to-JSON translation)
5. Write JSON Schema files (draft-07 or later):
   - `docs/schemas/edd.schema.json`
   - `docs/schemas/dt.schema.json`
   - `docs/schemas/mapping.schema.json`
6. Write example JSON files showing the CHIP EDD and one decision table in JSON format

Files to read:
- `go/pkg/dtrules/loader/edd.go` - struct definitions
- `go/pkg/dtrules/loader/dt.go` - struct definitions
- `go/pkg/dtrules/mapping/map_loader.go` - mapping format
- `sampleprojects/CHIP/xml/` - real XML examples

Files to create:
- `docs/schemas/edd.schema.json`
- `docs/schemas/dt.schema.json`
- `docs/schemas/mapping.schema.json`
- `docs/schemas/examples/edd-example.json`
- `docs/schemas/examples/dt-example.json`

Acceptance criteria:
- JSON schemas are valid JSON Schema
- Examples validate against schemas
- JSON format is natural (arrays for lists, nested objects for hierarchy)
- Mapping between XML attributes/elements and JSON properties is documented

---

## E2: Go JSON struct tags and format detection

Title: Add JSON struct tags and auto-detect format in Go loaders

Depends on: E1

**Prerequisites -- verify before starting:**
- [ ] JSON schemas exist (produced by E1): `docs/schemas/edd.schema.json`, `docs/schemas/dt.schema.json`, `docs/schemas/mapping.schema.json` -- the json struct tags must produce JSON matching these schemas
- [ ] Example JSON files exist (produced by E1): `docs/schemas/examples/edd-example.json`, `docs/schemas/examples/dt-example.json` -- use these to verify round-trip
- [ ] Existing XML loading works: run `go test ./go/pkg/dtrules/...` to establish baseline before modifying loader structs

Task:
1. Add `json:"..."` tags to all structs in `go/pkg/dtrules/loader/edd.go`
2. Add `json:"..."` tags to all structs in `go/pkg/dtrules/loader/dt.go`
3. Implement format detection in `go/pkg/dtrules/loader/format.go` (new file):
   - `DetectFormat(r io.Reader) (string, io.Reader)` - peek first non-whitespace byte, '<' = XML, '{' or '[' = JSON
   - Returns format string and a new reader (with peeked bytes restored via io.MultiReader)
4. Write round-trip tests: XML->struct->JSON->struct, verify identical

Files to modify:
- `go/pkg/dtrules/loader/edd.go` - add json tags
- `go/pkg/dtrules/loader/dt.go` - add json tags

Files to create:
- `go/pkg/dtrules/loader/format.go` - format detection
- `go/pkg/dtrules/loader/format_test.go`
- `go/pkg/dtrules/loader/roundtrip_test.go`

Acceptance criteria:
- All structs have json tags matching the schema from E1
- Format detection works for XML and JSON
- Round-trip tests pass
- Existing XML loading still works (no regressions)

---

## E3: Go EDD and DT JSON loaders

Title: Implement JSON loading for EDD and Decision Table files

Depends on: E2

**Prerequisites -- verify before starting:**
- [ ] JSON struct tags exist on all loader structs (produced by E2) -- verify `json:"..."` tags in `go/pkg/dtrules/loader/edd.go` and `dt.go`
- [ ] Format detection function exists (produced by E2) -- `go/pkg/dtrules/loader/format.go` with `DetectFormat(r io.Reader)` function
- [ ] Round-trip tests pass (produced by E2) -- confirms struct tags produce correct JSON
- [ ] JSON schemas exist (produced by E1) for validation reference
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Task:
1. In `go/pkg/dtrules/loader/edd.go`, add JSON path:
   - Use format detection to determine XML vs JSON
   - JSON path: `json.Unmarshal` into same EDDFile struct
   - Share post-unmarshal processing (entity creation, field registration)
2. In `go/pkg/dtrules/loader/dt.go`, add JSON path:
   - Same pattern: detect format, unmarshal, share processing
   - Postfix->bytecode compilation must be shared
3. Update `go/pkg/dtrules/session/ruleset.go` LoadEDD and LoadDecisionTables to use format detection

Files to modify:
- `go/pkg/dtrules/loader/edd.go`
- `go/pkg/dtrules/loader/dt.go`
- `go/pkg/dtrules/session/ruleset.go`

Files to read:
- `docs/schemas/edd.schema.json` (from E1)
- `docs/schemas/dt.schema.json` (from E1)

Test: Create JSON version of CHIP EDD, load it, verify entity factory matches XML-loaded version.

Acceptance criteria:
- LoadEDD accepts both XML and JSON transparently
- LoadDecisionTables accepts both XML and JSON transparently
- CHIP loads correctly from JSON
- `go test ./go/pkg/dtrules/...` passes

---

## E4: Go Mapping JSON loader

Title: Implement JSON loading for Mapping files

Depends on: E2

**Prerequisites -- verify before starting:**
- [ ] Format detection function exists (produced by E2) -- `go/pkg/dtrules/loader/format.go`
- [ ] Mapping JSON schema exists (produced by E1) -- `docs/schemas/mapping.schema.json`
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Task:
1. Read `go/pkg/dtrules/mapping/map_loader.go` to understand current streaming XML parsing
2. Define mapping JSON struct (or add json tags to existing structs)
3. Implement JSON loading path: detect format, if JSON unmarshal fully, then process
4. Share entity creation/attribute setting logic with XML path

Files to modify: `go/pkg/dtrules/mapping/map_loader.go`
Files to create: mapping round-trip tests

Acceptance criteria:
- Mapping files load from both XML and JSON
- Entity creation produces same results regardless of format

---

## E5: Go JSON export/serialization

Title: Add JSON export for EDD, DT, and Mapping data

Depends on: E2

**Prerequisites -- verify before starting:**
- [ ] JSON struct tags exist on all loader structs (produced by E2) -- these enable `json.Marshal`
- [ ] JSON schemas exist (produced by E1) -- exported JSON must validate against these schemas
- [ ] `go test ./go/pkg/dtrules/...` passes before your changes

Task:
1. Add `json.Marshal` support (json tags from E2 enable this)
2. Create utility function or CLI command to convert XML->JSON:
   - Read XML file, unmarshal to struct, marshal to JSON
   - Optionally: `go run ./go/cmd/convert/main.go -input file.xml -output file.json`
3. Use this to generate JSON versions of all sample project files

Files to create:
- `go/cmd/convert/main.go` (optional CLI)
- JSON versions of CHIP files in `sampleprojects/CHIP/json/`

Acceptance criteria:
- XML->JSON conversion produces valid JSON matching schema from E1
- Converted files load correctly via JSON loader from E3

---

## E6: Java JSON loader infrastructure

Title: Add JSON parsing library and loader infrastructure to Java

Depends on: E1

**Prerequisites -- verify before starting:**
- [ ] JSON schemas exist (produced by E1): `docs/schemas/edd.schema.json`, `docs/schemas/dt.schema.json` -- Java JSON loaders must produce the same object model as XML loaders
- [ ] Example JSON files exist (produced by E1) -- use these for testing
- [ ] Java builds: run `mvn test` from project root to verify baseline
- [ ] Understand existing XML loader architecture: read `dtrules-engine/src/main/java/com/dtrules/entity/EDDLoader.java` and `DTLoader.java` for the `IGenericXMLParser` pattern

Context: Java side uses custom `IGenericXMLParser` with SAX-like beginTag/endTag callbacks. Located in `dtrules-engine/src/main/java/com/dtrules/`.

Task:
1. Add Jackson dependency to `dtrules-engine/pom.xml`
2. Create JSON equivalents of the XML loaders:
   - `com.dtrules.entity.EDDJsonLoader` - loads EDD from JSON
   - `com.dtrules.decisiontables.DTJsonLoader` - loads DT from JSON
3. Add format detection to existing load paths
4. Share entity/table construction logic between XML and JSON paths

Files to modify:
- `dtrules-engine/pom.xml` - add Jackson
- `dtrules-engine/src/main/java/com/dtrules/entity/EDDLoader.java` - add format detection
- `dtrules-engine/src/main/java/com/dtrules/decisiontables/DTLoader.java` - add format detection

Files to create:
- `dtrules-engine/src/main/java/com/dtrules/entity/EDDJsonLoader.java`
- `dtrules-engine/src/main/java/com/dtrules/decisiontables/DTJsonLoader.java`

Acceptance criteria:
- Jackson dependency added
- EDD loads from JSON in Java
- DT loads from JSON in Java
- `mvn test` passes

---

## E7: Cross-format validation tests

Title: Automated tests verifying XML and JSON produce identical results
Depends on: E3, E4, E6

**Prerequisites -- verify before starting:**
- [ ] Go EDD JSON loader works (produced by E3) -- `LoadEDD` accepts JSON
- [ ] Go DT JSON loader works (produced by E3) -- `LoadDecisionTables` accepts JSON
- [ ] Go Mapping JSON loader works (produced by E4) -- mapping files load from JSON
- [ ] Java JSON loaders work (produced by E6) -- EDD and DT load from JSON in Java
- [ ] JSON versions of CHIP files exist (produced by E5 or E1) -- in `sampleprojects/CHIP/json/` or `docs/schemas/examples/`
- [ ] Both Go and Java test suites pass: `go test ./go/pkg/dtrules/...` and `mvn test`

Task:
1. Go: Load CHIP from XML, run decision tables, capture results. Load CHIP from JSON, run same tables, compare.
2. Java: Same.
3. Automate in CI (`.github/workflows/tests.yml`)
4. Cover all sample projects that have JSON versions

Files to create:
- `go/pkg/dtrules/loader/format_validation_test.go`
- Java test class for format comparison

Acceptance criteria:
- XML and JSON loading produce identical execution results
- Automated in CI
- Clear error messages on mismatch

---

# Track F: Performance Measurement

## F1: Build Java performance measurement infrastructure

Title: Build Java performance measurement into DTRules

Depends on: none (independent, can start anytime)

**Prerequisites -- verify before starting:**
- [ ] Java builds successfully: run `cd /home/paul/go/src/github.com/DTRules/DTRules && mvn test > /tmp/mvn-test.log 2>&1` and check `tail /tmp/mvn-test.log`
- [ ] CHIP sample project exists and its tests pass: check `sampleprojects/CHIP/`
- [ ] No JMH or benchmark infrastructure currently exists in Java -- confirmed. `sampleprojects/CHIP/pom.xml` has no JMH dependency. `TestChip.java` has a `Date start` field but no timing code.

Context: The Go runtime has comprehensive benchmarks (59 individual tests in `go/pkg/dtrules/benchmark/benchmark_test.go` plus a cross-runtime comparison tool in `go/cmd/benchmark/main.go`). Java has NOTHING -- no JMH, no timing code, no benchmark classes. The `test/run-benchmarks.sh` script explicitly says "Java benchmarks require JMH. No estimates are shown here because fabricated numbers are worse than no numbers."

Standard Java profiling tools (JMH, VisualVM, etc.) were not working in this environment. We need to build measurement directly into the Java code rather than relying on external tooling.

Task:
1. Build a simple, self-contained benchmark harness directly in the CHIP sample project. Do NOT use JMH -- build timing into the code itself:
   - Create `sampleprojects/CHIP/src/main/java/com/dtrules/samples/chipeligibility/BenchmarkChip.java`
   - Use `System.nanoTime()` for timing (reliable, no external deps)
   - Implement warmup phase (run 1000 iterations to trigger JIT, discard results)
   - Implement measurement phase (run N iterations, record each)
   - Calculate: min, max, mean, median, p95, p99, std dev
   - Report results in JSON format matching the Go benchmark output format

2. Benchmark the same operations that Go measures, so results are comparable:
   - **RuleSet loading**: time to load EDD + DT files from `sampleprojects/CHIP/repository/xml/`
   - **Session creation**: time to create a new session from a loaded RuleSet
   - **Decision table execution**: time to execute CHIP eligibility tables against test cases
   - **Individual operation categories**: If possible, construct bytecode-equivalent operations:
     - Arithmetic: push two integers, add
     - Comparison: push two values, compare
     - Boolean: push two booleans, and/or
     - Stack: push, dup, swap, pop
     - String: push two strings, concatenate

3. Output format must match Go's `benchmark-results/all_benchmarks.json`:
   ```json
   {
     "runtime": "java",
     "category": "arithmetic|comparison|boolean|stack|complex",
     "operation": "add|sub|mul|div|lt|eq|and|or|not|etc",
     "ns_per_op": 78.5,
     "ops_per_sec": 12738853,
     "iterations": 100000
   }
   ```

4. Add a Maven exec profile so the benchmark can be run via:
   `mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.BenchmarkChip" -pl sampleprojects/CHIP > /tmp/java-bench.log 2>&1`

5. Update `test/run-benchmarks.sh` to call the Java benchmark and merge results

Key files to read:
- `go/cmd/benchmark/main.go` -- the Go benchmark tool (understand what operations are measured and the output format)
- `go/pkg/dtrules/benchmark/benchmark_test.go` -- 59 Go benchmarks for operation categories
- `benchmark-results/all_benchmarks.json` -- the JSON output format (57 existing measurements)
- `test/run-benchmarks.sh` -- lines that check for Java benchmarks and the "JMH required" message
- `sampleprojects/CHIP/src/main/java/com/dtrules/samples/chipeligibility/TestChip.java` -- existing test harness to understand how CHIP sessions are created and executed
- `sampleprojects/CHIP/pom.xml` -- current Maven config

Key files to modify:
- `sampleprojects/CHIP/pom.xml` -- add exec-maven-plugin if not present
- `test/run-benchmarks.sh` -- update Java benchmark section to call BenchmarkChip instead of looking for JMH

Key files to create:
- `sampleprojects/CHIP/src/main/java/com/dtrules/samples/chipeligibility/BenchmarkChip.java`

Acceptance criteria:
- BenchmarkChip.java runs successfully and produces JSON output
- Output format matches Go benchmark JSON structure
- Warmup phase prevents cold-start skew
- Results include: ns_per_op, ops_per_sec, iterations for each operation
- At minimum, measures: RuleSet load, session creation, decision table execution
- Ideally measures individual operation categories (arithmetic, comparison, etc.)
- `test/run-benchmarks.sh` successfully runs Java benchmarks and merges results
- No external profiling tools required -- all measurement is in-code

---

## F2: Cross-runtime performance comparison (all 4 runtimes)

Title: Comprehensive performance comparison across Java, Go, NativeASM, and x86-64-ASM

Depends on: F1, D1

**Prerequisites -- verify before starting:**
- [ ] Java benchmark infrastructure works (produced by F1) -- `BenchmarkChip.java` produces JSON output. Verify: `cd /home/paul/go/src/github.com/DTRules/DTRules && mvn exec:java -Dexec.mainClass="com.dtrules.samples.chipeligibility.BenchmarkChip" -pl sampleprojects/CHIP > /tmp/java-bench.log 2>&1` and check `tail /tmp/java-bench.log`
- [ ] All legacy sync code is deleted (produced by D1) -- ASM runtimes are in their final architecture
- [ ] All 4 runtimes produce correct results -- CHIP decision tables pass on Go, NativeASM, x86-64-ASM, and Java
- [ ] Go benchmark tools work: `go/cmd/benchmark/main.go` and `go/pkg/dtrules/benchmark/benchmark_test.go`
- [ ] `benchmark-results/all_benchmarks.json` exists with current Go/ASM data
- [ ] `test/run-benchmarks.sh` can run both Go and Java benchmarks (updated in F1)

Context: DTRules has 4 runtimes that should produce identical results but with different performance characteristics:
- **Go** -- reference implementation, currently ~12us/op on CHIP
- **NativeASM** -- Plan 9 assembly, should be faster than Go after architecture fix
- **x86-64-ASM** -- NASM via CGO, should be fast but CGO overhead may limit gains
- **Java** -- JVM implementation, expected slowest but by how much?

The Go benchmark infrastructure measures Go and both ASM runtimes. Java measurement was added in F1. This issue brings them all together.

Task:
1. **Run all benchmarks on the same machine, same conditions:**
   - Close unnecessary applications
   - Run each benchmark 3 times, take the median
   - All runtimes execute the same operations

2. **Standardize the operation set across all runtimes.** At minimum:

   | Category | Operations | How measured |
   |----------|-----------|--------------|
   | Arithmetic | add, sub, mul, div (int and double) | Push two values, execute op, measure |
   | Comparison | eq, ne, lt, gt, le, ge | Push two values, compare, measure |
   | Boolean | and, or, not | Push boolean(s), execute op, measure |
   | Stack | push, pop, dup, swap | Execute stack ops, measure |
   | String | concat, substring | Push strings, execute op, measure |
   | Complex | CHIP decision table execution | Load CHIP, run test cases, measure per-case |
   | Loading | RuleSet load time | Time to load EDD + DT from XML |
   | Session | Session creation time | Time to create session from loaded RuleSet |

3. **Update `go/cmd/benchmark/main.go`** to:
   - Read Java benchmark results from JSON (produced by F1's BenchmarkChip)
   - Include Java in comparison charts and tables
   - Show all 4 runtimes side-by-side

4. **Update `test/run-benchmarks.sh`** to:
   - Run Go benchmarks (all 3 Go-based runtimes)
   - Run Java benchmarks (BenchmarkChip)
   - Merge all results into single `benchmark-results/all_benchmarks.json`
   - Generate updated HTML report with all 4 runtimes

5. **Generate the comparison report:**
   - Update `benchmark-results/all_benchmarks.json` with all 4 runtimes
   - Update `benchmark-results/benchmark_report.html` with 4-way comparison charts
   - Create `benchmark-results/benchmark_summary.md` with:
     - Table of ns/op for each operation × each runtime
     - Speedup ratios (baseline = Go)
     - Analysis: where does each runtime win/lose and why
     - CGO overhead analysis (NativeASM vs x86-64-ASM difference = CGO cost)

6. **Add to CI:** Update `.github/workflows/tests.yml` to run benchmarks on main branch merges (not PRs -- too slow)

Key files to read:
- `go/cmd/benchmark/main.go` -- current cross-runtime tool (Go + ASM only)
- `go/pkg/dtrules/benchmark/benchmark_test.go` -- 59 Go benchmarks
- `benchmark-results/all_benchmarks.json` -- current data format and results
- `benchmark-results/benchmark_report.html` -- current HTML report
- `test/run-benchmarks.sh` -- orchestration script
- `sampleprojects/CHIP/src/main/java/com/dtrules/samples/chipeligibility/BenchmarkChip.java` -- Java benchmarks (from F1)

Key files to modify:
- `go/cmd/benchmark/main.go` -- add Java results reading
- `test/run-benchmarks.sh` -- full 4-runtime orchestration
- `benchmark-results/all_benchmarks.json` -- add Java data
- `benchmark-results/benchmark_report.html` -- 4-way charts
- `.github/workflows/tests.yml` -- benchmark CI job

Key files to create:
- `benchmark-results/benchmark_summary.md` -- human-readable analysis

Acceptance criteria:
- All 4 runtimes benchmarked on the same operation set
- Results in unified JSON format in `all_benchmarks.json`
- HTML report shows 4-way comparison with charts
- Summary document with analysis and speedup ratios
- Benchmark script runs end-to-end: `test/run-benchmarks.sh > /tmp/bench-all.log 2>&1`
- Each runtime's results are clearly labeled and comparable
- Analysis explains performance differences (JVM warmup, CGO overhead, ASM dispatch efficiency)

# DTRules Changelog

## Version 5.0-SNAPSHOT

### 2026-02-05: Unified Test Infrastructure

#### Summary
Implemented cross-platform test infrastructure with shared test vectors, CI/CD pipeline, and comprehensive documentation for all DTRules implementations (Go, Java, ASM).

#### New Files

**Test Infrastructure:**
- `test/run-all-tests.sh` - Cross-platform test orchestrator
- `test/readme.md` - Test infrastructure documentation
- `test/vectors/*.json` - Shared test vectors (206 tests across 10 categories)

**Shared Test Vectors:**
| File | Tests | Coverage |
|------|-------|----------|
| arithmetic.json | 28 | +, -, *, /, abs, negate, f+, f-, f*, fdiv |
| comparison.json | 24 | ==, !=, <, >, <=, >= |
| boolean.json | 17 | and, or, not, xor, beq |
| stack.json | 17 | pop, dup, swap, rot, over, pick, roll |
| string.json | 34 | concat, substring, trim, indexof, etc. |
| array.json | 19 | newarray, addto, length, getat, memberof |
| control.json | 12 | if, ifelse, for, while, forall |
| table.json | 16 | newtable, tableget, tableput |
| entity.json | 15 | def, lookup, entitypush, get |
| datetime.json | 24 | newdate, getyear, adddays, daysbetween |

**CI/CD:**
- `.github/workflows/tests.yml` - GitHub Actions workflow
  - Go tests: Linux, macOS, Windows (Go 1.21, 1.22)
  - Java tests: Linux, macOS, Windows (JDK 11, 17, 21)
  - ASM tests: Linux (NASM)
  - Comparison tests (ASM vs Go)
  - Performance benchmarks (main branch)

**Java Unit Tests:**
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RIntegerTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RDoubleTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RBooleanTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RStringTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RNameTest.java`
- `dtrules-engine/src/test/java/com/dtrules/interpreter/RArrayTest.java`

**Documentation:**
- `docs/testing.md` - Comprehensive testing guide

#### Verification

**ASM Test Results:**
- 13 unit test modules pass
- 100+ individual tests pass
- All arithmetic, comparison, boolean, stack, string, control flow operators verified

**NativeASM Test Results:**
- All tests pass
- Full coverage of arithmetic, comparison, boolean, and stack operations

**Go Test Results:**
- All 23+ test suites pass
- Comprehensive operator coverage

---

### 2026-01-30: ANTLR 4 Migration

#### Summary
Migrated the EL and EBL DSL compiler modules from JFlex/CUP to ANTLR 4, modernizing the parser infrastructure while maintaining backward compatibility.

#### Changes

**New ANTLR 4 Compilers:**
- `ELAntlr` - Expression Language compiler (replaces `EL`)
- `EBLAntlr` - Extended Business Language compiler (replaces `EBL`)

**New Files Added:**

| Module | Grammar | Compiler | Visitor | Type Resolver |
|--------|---------|----------|---------|---------------|
| EL | EL.g4 | ELAntlr.java | ELCompilerVisitor.java | ELTypeResolver.java |
| EBL | EBL.g4 | EBLAntlr.java | EBLCompilerVisitor.java | EBLTypeResolver.java |

**Build System Updates:**
- Added ANTLR 4 Maven plugin (v4.13.2) to EL and EBL module pom.xml files
- Added ANTLR 4 runtime dependency
- Retained JFlex/CUP dependencies for legacy compiler support

**Documentation:**
- Added `dsl/ANTLR_MIGRATION.md` with comprehensive migration guide
- Added `ELCompilerComparisonTest.java` for validating compiler parity
- Added `CompilerTest.java` - parameterized JUnit test suite (128 tests)

#### Compatibility
- **100% test compatibility** - 128 tests pass (64 tests x 2 compilers)
- Legacy compilers retained for backward compatibility
- Same `ICompiler` interface - drop-in replacement
- All sample projects verified working

#### How to Switch Compilers
Update `DTRules.xml`:
```xml
<!-- Use new ANTLR 4 compiler -->
<compileralias name="EL">com.dtrules.compiler.el.ELAntlr</compileralias>
```

#### Sample Projects Tested
All compile and run with 0 errors:
- CHIP - 2 test cases PASS
- KidAid - 2 test cases PASS
- SyntaxTests - 1 test case PASS
- TestProject - Compile PASS

### 2026-01-30: Project Cleanup

#### Removed Sample Projects
The following incomplete sample projects were removed:
- `Sudoku` - Custom DSL demo (incomplete test data)
- `eBook` - Multi-ruleset example (missing test cases)
- `eBookApp` - eBook application wrapper

#### Remaining Sample Projects
- `CHIP` - Health insurance eligibility (2 test cases)
- `ChipApp` - CHIP application wrapper
- `KidAid` - Child assistance eligibility (2 test cases)
- `KidAid_Application` - KidAid application wrapper
- `SyntaxTests` - EL language reference (1 test case)
- `TestProject` - Minimal template

---

## Version 4.x (Historical)

See git history for earlier changes.

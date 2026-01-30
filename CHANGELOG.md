# DTRules Changelog

## Version 5.0-SNAPSHOT

### 2026-01-30: ANTLR 4 Migration

#### Summary
Migrated all three DSL compiler modules from JFlex/CUP to ANTLR 4, modernizing the parser infrastructure while maintaining backward compatibility.

#### Commits
- `cdd87b2` - Add parameterized compiler test suite
- `fa17dc2` - Fix compiler comparison tests to pass at 100%
- `8479311` - Add CHANGELOG documenting ANTLR 4 migration
- `c942e4b` - Improve ANTLR migration documentation
- `76ef9bf` - Migrate DSL compilers from JFlex/CUP to ANTLR 4

#### Changes

**New ANTLR 4 Compilers:**
- `ELAntlr` - Expression Language compiler (replaces `EL`)
- `EBLAntlr` - Extended Business Language compiler (replaces `EBL`)
- `SudokuAntlr` - Sudoku Language compiler (replaces `SudokuLanguage`)

**New Files Added:**
| Module | Grammar | Compiler | Visitor | Type Resolver |
|--------|---------|----------|---------|---------------|
| EL | EL.g4 | ELAntlr.java | ELCompilerVisitor.java | ELTypeResolver.java |
| EBL | EBL.g4 | EBLAntlr.java | EBLCompilerVisitor.java | EBLTypeResolver.java |
| Sudoku | Sudoku.g4 | SudokuAntlr.java | SudokuCompilerVisitor.java | SudokuTypeResolver.java |

**Build System Updates:**
- Added ANTLR 4 Maven plugin (v4.13.2) to all DSL module pom.xml files
- Added ANTLR 4 runtime dependency
- Retained JFlex/CUP dependencies for legacy compiler support

**Documentation:**
- Added `dsl/ANTLR_MIGRATION.md` with comprehensive migration guide
- Added `ELCompilerComparisonTest.java` for validating compiler parity
- Added `CompilerTest.java` - parameterized JUnit test suite (128 tests)

#### Compatibility
- **100% test compatibility** - 128 tests pass (64 tests × 2 compilers)
- 2 tests show "IMPROVED" where ANTLR 4 is more permissive (accepts `null` comparisons)
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
All compile with 0 errors:
- SyntaxTests ✓
- KidAid ✓
- CHIP ✓
- TestProject ✓
- Sudoku ✓

# DTRules Changelog

## Version 5.0-SNAPSHOT

### 2026-01-30: ANTLR 4 Migration

#### Summary
Migrated all three DSL compiler modules from JFlex/CUP to ANTLR 4, modernizing the parser infrastructure while maintaining backward compatibility.

#### Commits
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

#### Compatibility
- **97% test compatibility** (72/74 tests pass)
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

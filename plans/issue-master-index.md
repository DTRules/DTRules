# DTRules Issue Master Index

## Issue Summary

| ID | GH# | Title | Track | Depends On | Parallel Group |
|----|-----|-------|-------|-----------|----------------|
| **A1** | [#35](https://github.com/DTRules/DTRules/issues/35) | Formal bytecode specification | Architecture | -- | Foundation |
| **A2** | [#36](https://github.com/DTRules/DTRules/issues/36) | DTState memory layout verification | Architecture | A1 | Foundation |
| **A3** | [#40](https://github.com/DTRules/DTRules/issues/40) | Move VMState into all runtimes | Architecture | A1 | Foundation |
| **A4** | [#44](https://github.com/DTRules/DTRules/issues/44) | Zero-overhead ASM dispatch loop | Architecture | A2, A3 | Foundation |
| **A5** | [#49](https://github.com/DTRules/DTRules/issues/49) | ASM memory management strategy | Architecture | A1, A3 | Foundation |
| **B1** | [#37](https://github.com/DTRules/DTRules/issues/37) | NativeASM arithmetic opcodes | NativeASM | A4 | B-simple |
| **B2** | [#42](https://github.com/DTRules/DTRules/issues/42) | NativeASM comparison opcodes | NativeASM | A4 | B-simple |
| **B3** | [#47](https://github.com/DTRules/DTRules/issues/47) | NativeASM boolean opcodes | NativeASM | A4 | B-simple |
| **B4** | [#51](https://github.com/DTRules/DTRules/issues/51) | NativeASM stack manipulation opcodes | NativeASM | A4 | B-simple |
| **B5** | [#55](https://github.com/DTRules/DTRules/issues/55) | NativeASM string opcodes | NativeASM | A4, A5 | B-complex |
| **B6** | [#57](https://github.com/DTRules/DTRules/issues/57) | NativeASM array opcodes | NativeASM | A4, A5 | B-complex |
| **B7** | [#59](https://github.com/DTRules/DTRules/issues/59) | NativeASM table opcodes | NativeASM | A4, A5 | B-complex |
| **B8** | [#63](https://github.com/DTRules/DTRules/issues/63) | NativeASM control flow opcodes | NativeASM | A4 | B-control |
| **B9** | [#67](https://github.com/DTRules/DTRules/issues/67) | NativeASM entity opcodes | NativeASM | A4, A5 | B-control |
| **B10** | [#70](https://github.com/DTRules/DTRules/issues/70) | NativeASM datetime opcodes | NativeASM | A4, A5 | B-complex |
| **C1** | [#39](https://github.com/DTRules/DTRules/issues/39) | x86-64 arithmetic opcodes | x86-64-ASM | A4 | C-simple |
| **C2** | [#41](https://github.com/DTRules/DTRules/issues/41) | x86-64 comparison opcodes | x86-64-ASM | A4 | C-simple |
| **C3** | [#46](https://github.com/DTRules/DTRules/issues/46) | x86-64 boolean opcodes | x86-64-ASM | A4 | C-simple |
| **C4** | [#50](https://github.com/DTRules/DTRules/issues/50) | x86-64 stack manipulation opcodes | x86-64-ASM | A4 | C-simple |
| **C5** | [#54](https://github.com/DTRules/DTRules/issues/54) | x86-64 string opcodes | x86-64-ASM | A4, A5 | C-complex |
| **C6** | [#58](https://github.com/DTRules/DTRules/issues/58) | x86-64 array opcodes | x86-64-ASM | A4, A5 | C-complex |
| **C7** | [#61](https://github.com/DTRules/DTRules/issues/61) | x86-64 table opcodes | x86-64-ASM | A4, A5 | C-complex |
| **C8** | [#64](https://github.com/DTRules/DTRules/issues/64) | x86-64 control flow opcodes | x86-64-ASM | A4 | C-control |
| **C9** | [#68](https://github.com/DTRules/DTRules/issues/68) | x86-64 entity opcodes | x86-64-ASM | A4, A5 | C-control |
| **C10** | [#72](https://github.com/DTRules/DTRules/issues/72) | x86-64 datetime opcodes | x86-64-ASM | A4, A5 | C-complex |
| **D1** | [#38](https://github.com/DTRules/DTRules/issues/38) | Delete legacy VMState/sync code | Cleanup | B1-B10, C1-C10 | Cleanup |
| **D2** | [#43](https://github.com/DTRules/DTRules/issues/43) | Cross-runtime validation test suite | Validation | B1-B10, C1-C10 | Cleanup |
| **D3** | [#45](https://github.com/DTRules/DTRules/issues/45) | Post-fix performance benchmarks | Validation | D1 | Cleanup |
| **D4** | [#48](https://github.com/DTRules/DTRules/issues/48) | Update architecture documentation | Docs | D1 | Cleanup |
| **D5** | [#52](https://github.com/DTRules/DTRules/issues/52) | Implement OpExec (recursive execution) | Cross-cutting | A4 | B-control |
| **D6** | [#53](https://github.com/DTRules/DTRules/issues/53) | Policy statement handling | Fix | A1 | Independent |
| **E1** | [#56](https://github.com/DTRules/DTRules/issues/56) | Define JSON schema for all formats | JSON | -- | E-foundation |
| **E2** | [#60](https://github.com/DTRules/DTRules/issues/60) | Go JSON struct tags + format detection | JSON | E1 | E-go |
| **E3** | [#62](https://github.com/DTRules/DTRules/issues/62) | Go EDD and DT JSON loaders | JSON | E2 | E-go |
| **E4** | [#65](https://github.com/DTRules/DTRules/issues/65) | Go Mapping JSON loader | JSON | E2 | E-go |
| **E5** | [#66](https://github.com/DTRules/DTRules/issues/66) | Go JSON export/serialization | JSON | E2 | E-go |
| **E6** | [#69](https://github.com/DTRules/DTRules/issues/69) | Java JSON loader infrastructure | JSON | E1 | E-java |
| **E7** | [#71](https://github.com/DTRules/DTRules/issues/71) | Cross-format validation tests | JSON | E3, E4, E6 | E-validation |
| **F1** | [#33](https://github.com/DTRules/DTRules/issues/33) | Java performance measurement infrastructure | Perf | -- | F-foundation |
| **F2** | [#34](https://github.com/DTRules/DTRules/issues/34) | Cross-runtime performance comparison (all 4) | Perf | F1, D1 | F-final |

**Total: 40 issues**

## Dependency Graph

```
                    A1 (#35) ──────────────────── E1 (#56)
                   / |  \                           |      \
                  /  |   \                          |       \
           A2(#36) A3(#40) A5(#49)                E2(#60)  E6(#69)
                 \   |   /                        / | \      |
                  \  |  /                     E3 E4 E5    E6
                  A4 (#44)                       \  |    /
                  / | \                           E7(#71)
                 /  |  \
    ┌───────────┘   |   └───────────┐
    |               |               |
 B-simple       B-control       C-simple
 B1,B2,B3,B4   B8,B9,D5       C1,C2,C3,C4
 #37,42,47,51  #63,67,52      #39,41,46,50
    |               |               |
 B-complex      (after A5)     C-complex
 B5,B6,B7,B10                 C5,C6,C7,C10
 #55,57,59,70                 #54,58,61,72
    |                               |
    └───────────┐   ┌───────────────┘
                |   |
           D1 (#38, delete sync)
           D2 (#43, validation)
                |
           D3 (#45, benchmarks)
           D4 (#48, docs)

F1 (#33, Java perf) ────────────── [independent, start anytime]
└── F2 (#34, 4-runtime comparison) ←── also needs D1 (#38)
```

## Parallel Session Orchestration Plan

### Wave 1: Foundations (3 sessions, fully parallel)

| Session | Issues | GH# | Description |
|---------|--------|-----|-------------|
| 1 | **A1** | #35 | Bytecode specification |
| 2 | **E1** | #56 | JSON schema definition |
| 3 | **F1** | #33 | Java performance measurement |

### Wave 2: Architecture + JSON Go (3 sessions, after Wave 1)

| Session | Issues | GH# | Description |
|---------|--------|-----|-------------|
| 4 | **A2, A3** | #36, #40 | DTState layout + VMState refactor |
| 5 | **A5** | #49 | Memory management strategy |
| 6 | **E2, E3, E4, E5** | #60, #62, #65, #66 | Go JSON loaders and export |

### Wave 3: Dispatch + Java JSON (3 sessions, after Wave 2)

| Session | Issues | GH# | Description |
|---------|--------|-----|-------------|
| 7 | **A4** | #44 | Zero-overhead dispatch loop |
| 8 | **E6** | #69 | Java JSON infrastructure |
| 9 | **D5, D6** | #52, #53 | OpExec + policy statement fix |

### Wave 4: Opcodes (6 sessions, after Wave 3 -- main parallel burst)

| Session | Issues | GH# | Description |
|---------|--------|-----|-------------|
| 10 | **B1, B2, B3, B4** | #37, #42, #47, #51 | NativeASM simple opcodes |
| 11 | **B5, B6, B7, B10** | #55, #57, #59, #70 | NativeASM complex opcodes |
| 12 | **B8, B9** | #63, #67 | NativeASM control + entity |
| 13 | **C1, C2, C3, C4** | #39, #41, #46, #50 | x86-64 simple opcodes |
| 14 | **C5, C6, C7, C10** | #54, #58, #61, #72 | x86-64 complex opcodes |
| 15 | **C8, C9** | #64, #68 | x86-64 control + entity |

### Wave 5: Finalize (4 sessions, after Wave 4)

| Session | Issues | GH# | Description |
|---------|--------|-----|-------------|
| 16 | **D1** | #38 | Delete all legacy sync code |
| 17 | **D2** | #43 | Cross-runtime validation |
| 18 | **D4, E7** | #48, #71 | Documentation + cross-format validation |
| 19 | **F2** | #34 | Full 4-runtime performance comparison |

### Totals
- **5 waves, 19 sessions**
- **Wave 4 is the widest: 6 parallel sessions**
- **Track E runs independently throughout Waves 1-3, 5**
- **F1 runs independently starting Wave 1; F2 runs after D1 + F1**

## File Locations

- Track A issues: `plans/track-a-architecture.md`
- Track B issues: `plans/track-b-nativeasm.md`
- Track C issues: `plans/track-c-x86-64-asm.md`
- Track D+E+F issues: `plans/track-d-e-f-cleanup-json-perf.md`

Note: D3 (#45, post-fix benchmarks) focuses on verifying ASM is faster than Go after the architecture fix.
F2 (#34, cross-runtime comparison) is the comprehensive 4-runtime comparison including Java.

## Notes for Session Orchestration

1. **Issues are the prompts.** Each GitHub issue body is a self-contained prompt for a Claude Code session. Use the issue body directly.

2. **Branch strategy:** Each session should create a feature branch from `issue-31-asm-optimization`:
   - `a1-bytecode-spec`, `a2-dtstate-layout`, etc.
   - Merge back to `issue-31-asm-optimization` before dependent sessions start

3. **File contention risks:**
   - Sessions 10-12 (NativeASM) all modify `vm_amd64.s` -- opcode handlers should be in separate labeled sections
   - Sessions 13-15 (x86-64) all modify `bytecode.asm` -- same issue
   - Session 4 (A3) modifies many files -- should complete before opcode sessions start

4. **Validation checkpoints:**
   - After Wave 2: `go test ./go/pkg/dtrules/...` must pass
   - After Wave 4: all 174 test vectors pass on all runtimes
   - After Wave 5: benchmarks show ASM faster than Go

5. **If an issue discovers a gap in the spec (A1 / #35),** create a follow-up issue and notify dependent sessions.

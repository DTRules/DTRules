# DTRules Orchestration Plan

## Overview

This plan organizes remaining work into parallel workstreams that can be executed by agent teams.

## Summary

| Category | Issues | Branches | Priority | Parallelizable |
|----------|--------|----------|----------|----------------|
| Infrastructure | 5 | 2 | HIGH | No (sequential) |
| Tax Test Groups | 18 | 0 | HIGH | Yes (6 parallel) |
| State Tax Completion | 8 | 5 | MEDIUM | Yes (8 parallel) |
| Corporate Tax | 24 | 0 | MEDIUM | Partial (by phase) |
| Website | 14 | 0 | LOW | Yes (7 parallel) |
| NASM/ASM | 15 | 20+ | LOW | Defer |
| Trace Navigator | 8 | 0 | LOW | Defer |
| Branch Cleanup | 0 | 35 | HIGH | Yes |

---

## Phase 1: Branch Cleanup (Immediate)

**Goal**: Evaluate and close/merge stale branches

### Stale fix/* Branches (Likely Obsolete)
These reference old issue numbers and are likely superseded:
- `fix/issue-1`, `fix/issue-3`, `fix/issue-7`, `fix/issue-8`
- `fix/issue-14`, `fix/issue-20`, `fix/issue-21`, `fix/issue-22`, `fix/issue-24`
- `fix/issue-29`, `fix/issue-31`, `fix/issue-33`, `fix/issue-35`, `fix/issue-36`
- `fix/issue-40`, `fix/issue-43`, `fix/issue-45`, `fix/issue-48`, `fix/issue-49`
- `fix/issue-52`, `fix/issue-53`, `fix/issue-56`, `fix/issue-60`, `fix/issue-62`
- `fix/issue-65`, `fix/issue-66`

**Action**: Launch 1 agent to evaluate each branch, close if superseded.

### Feature Branches to Evaluate
- `feature/issue-192` - Rhode Island tax (issue still open)
- `feature/issue-208` - Unknown
- `feature/issue-214` - Mississippi tax (issue still open)
- `feature/issue-231` - Multi-state allocation (issue closed, check if merged)
- `feature/issue-350-update-sample-projects` - Active work

### Other Branches
- `add-state-income-tax-sample` - Local only, delete
- `paulsnow-fork/add-state-income-tax-sample` - Fork reference
- `issue-29-anniversary-post` - Website content
- `fix/module-path-update` - Evaluate
- `main` - Legacy, keep or delete

---

## Phase 2: Infrastructure (Sequential - Dependencies)

**Goal**: Enable multi-file XML and Excel sync

### Order of Operations
1. **#342**: Implement multi-file XML loader ← FIRST
2. **#343**: Restructure TaxReturn XML to multi-file
3. **#344**: Create state tax dispatcher table
4. **#345**: Integrate state tax into main flow
5. **#346**: Update Excel extraction for new structure
6. **#347**: Apply multi-file to all sample projects
7. **#350**: Add FILE_PATH metadata

**Agents**: 1 agent, sequential execution

---

## Phase 3: Tax Test Groups (Parallel)

**Goal**: Implement 18 test groups for tax rules

### High Priority (Launch First)
| Issue | Test Group | Agent |
|-------|------------|-------|
| #351 | Federal Core & Credits | Agent 1 |
| #352 | Schedules A, C, D, E | Agent 2 |
| #359 | Military Tax Provisions | Agent 3 |
| #365 | State Tax - Partial Military | Agent 4 |
| #368 | Integration & Real-World | Agent 5 |

### Medium Priority (Wave 2)
| Issue | Test Group | Agent |
|-------|------------|-------|
| #353 | Above-the-Line Deductions | Agent 1 |
| #354 | Self-Employment & Additional Taxes | Agent 2 |
| #355 | Retirement & Social Security | Agent 3 |
| #356 | Business & Partnership Forms | Agent 4 |
| #358 | OBBBA 2025 & Special Deductions | Agent 5 |
| #360 | Foreign Income & Tax | Agent 6 |
| #361 | Kiddie Tax & Dependent Returns | Agent 1 |
| #362 | Special Situations & Edge Cases | Agent 2 |
| #364 | State Tax - Full Military | Agent 3 |
| #366 | Multi-State Scenarios | Agent 4 |

### Low Priority (Wave 3)
| Issue | Test Group | Agent |
|-------|------------|-------|
| #357 | Household Employment & Schedule H | Agent 1 |
| #363 | State Tax - No Income Tax States | Agent 2 |
| #367 | Estimated Tax & Penalties | Agent 3 |

**Agents**: 6 parallel agents, 3 waves

---

## Phase 4: State Tax Completion (Parallel)

**Goal**: Complete remaining state implementations

| Issue | State | Agent |
|-------|-------|-------|
| #192 | Rhode Island | Agent 1 |
| #214 | Mississippi | Agent 2 |
| #215 | Louisiana | Agent 3 |
| #216 | Arkansas | Agent 4 |
| #217 | Oklahoma | Agent 5 |
| #222 | New Mexico | Agent 6 |
| #223 | Utah | Agent 7 |
| #263 | California (Part 2) | Agent 8 |

**Related Issues** (may be covered):
- #178: Full 41-state coverage (epic)
- #234: Reciprocal state agreements
- #241: Local income taxes (future)
- #279: Refactor state files

**Agents**: 8 parallel agents

---

## Phase 5: Corporate Tax (Phased)

**Goal**: Implement corporate tax calculation

### Phase 1: Foundation (#316-322)
Sequential within phase, 1-2 agents:
1. #316: Define core EDD
2. #317: Create mapping layer
3. #318: Income calculation tables
4. #319: Deduction calculation tables
5. #320: Tax calculation tables
6. #321: Core test scenarios
7. #322: Go test harness integration

### Phase 2: Depreciation & Credits (#323-328)
2-3 agents parallel:
- Agent 1: #323, #324, #325 (depreciation)
- Agent 2: #326, #327 (credits)
- Agent 3: #328 (test cases)

### Phase 3: State Corporate Tax (#329-332)
Sequential design, parallel implementation:
1. #329: Design architecture (1 agent)
2. #330: Apportionment formulas (1 agent)
3. #331: 10 major states (5 agents parallel)
4. #332: Remaining 40 states (8 agents parallel)

### Phase 4: Advanced Features (#333-339)
3 agents parallel:
- Agent 1: #333, #334 (M-1, M-2)
- Agent 2: #335, #336 (Balance sheet, NOL)
- Agent 3: #337, #338, #339 (Foreign, S-Corp, benchmarks)

---

## Phase 6: Website (Parallel - Low Priority)

**Goal**: Update website with tax demo

### Content Updates (4 agents)
| Agent | Issues |
|-------|--------|
| 1 | #301, #302, #303 (fixes) |
| 2 | #304, #305, #306 (documentation) |
| 3 | #307, #308, #309 (compliance/hub) |
| 4 | #310, #311, #312, #313, #314 (features) |

---

## Deferred Work

### NASM/ASM VM (#41-72, #161-171)
- 15 issues for native assembly implementation
- 20+ branches with partial work
- **Recommendation**: Defer until core tax functionality complete
- Many branches may be obsolete (superseded by merged NASM PRs)

### Trace Navigator (#142-151)
- 8 issues for visual debugging tool
- **Recommendation**: Defer until test coverage complete

---

## Execution Commands

### Phase 1: Branch Cleanup
```
Launch 1 agent to evaluate all fix/* branches
Close branches where issues are already closed or work is superseded
```

### Phase 2: Infrastructure
```
Launch 1 agent for sequential infrastructure work
Issues: 342 → 343 → 344 → 345 → 346 → 347 → 350
```

### Phase 3: Test Groups (Wave 1)
```
Launch 5 agents in parallel for high-priority test groups
Issues: 351, 352, 359, 365, 368
```

### Phase 4: State Tax
```
Launch 8 agents in parallel for remaining states
Issues: 192, 214, 215, 216, 217, 222, 223, 263
```

---

## Metrics

| Phase | Issues | Est. Agents | Parallelism |
|-------|--------|-------------|-------------|
| 1: Branches | 35 branches | 1 | Sequential |
| 2: Infrastructure | 7 | 1 | Sequential |
| 3: Test Groups | 18 | 6 | 3 waves |
| 4: State Tax | 8 | 8 | Full parallel |
| 5: Corporate Tax | 24 | 8 | 4 phases |
| 6: Website | 14 | 4 | Full parallel |
| **Total** | **71 issues** | | |

---

## Notes

1. **Dependencies**: Phase 2 (Infrastructure) should complete before Phase 3-4 for proper XML structure
2. **State Files**: Use separate `states/XX_dt.xml` and `states/XX_edd.xml` per CLAUDE.md
3. **Testing**: Each agent should run tests with output redirected to `/tmp/*.log`
4. **Commits**: Follow conventional commit format with issue references

# Competitive landscape: decision-management platforms vs DTRules

Living document for #849. Profiles cite vendor documentation, not memory;
"vs DTRules" sections are our own analysis. Platforms not yet profiled are
listed as stubs at the end — this is a tranche, not the finished survey.

Last updated: 2026-08-24 (first tranche: OpenRules, Trisotech, Sapiens,
FlexRule, Sparkling Logic).

---

## OpenRules — the platform that made our bet

**Sources**: [openrules.com/ruleengine.htm](https://openrules.com/ruleengine.htm),
[rulesdeployment.htm](https://openrules.com/rulesdeployment.htm),
[docs/man_rules.html](https://openrules.com/docs/man_rules.html),
[openrulesdecisionmanager.com](https://openrulesdecisionmanager.com/)

- **Source of truth**: Excel (or Google Sheets). Rules are authored and *live*
  in spreadsheets — the same bet as our authoring contract.
- **Author**: business analysts, with FEEL formulas and inline Java snippets
  in cells for the sharp edges.
- **Execution**: `OpenRulesEngine` is a Java class; the engine **reads the
  Excel directly and re-initializes itself when the file's mtime changes**.
  Deployable embedded, as a web service, or AWS Lambda.
- **Drift**: *the problem is dissolved rather than solved* — there is no
  second artifact. The spreadsheet is both the authoring surface and the
  executable model; nothing can drift from it.
- **AI**: "built-in LLM capabilities" for authoring assistance (vendor claim,
  depth unverified).
- **Model**: commercial (Decision Manager), historically open-core; Java.

**vs DTRules.** The closest architectural relative, and the sharpest mirror.
They avoided our entire sync problem by never compiling: the engine
interprets Excel and hot-reloads on mtime. We compile Excel → XML → postfix,
which bought us a tiny embeddable Go VM, byte-stable artifacts that diff and
version in git, and provenance — at the price of the round-trip machinery
that took #1091/#1124/#1130/#1154 to get right. Notably, their hot-reload
trigger is **mtime** — the exact proxy our provenance work retired as
unreliable across clones, containers and CI. Our compiled posture is the
harder road but the more auditable one: their executable state is whatever
the spreadsheet says right now; ours is a hash-stamped artifact a regulator
can replay.

## Trisotech — the standards flagship

**Sources**: [trisotech.com/dmn](https://www.trisotech.com/dmn/),
[decision-model-creation-and-deployment-using-dmn](https://www.trisotech.com/decision-model-creation-and-deployment-using-dmn/)

- **Source of truth**: a model repository of DMN files (with BPMN/CMMN
  siblings); DMN **conformance level 3** including FEEL and boxed expressions.
- **Author**: business SMEs in a visual modeler; the pitch is "automated
  directly from the visual model into decision engines without IT
  translation" — i.e., the model *is* the executable, interpretation-style.
- **Drift**: same dissolution as OpenRules, at the model-repository level.
- **Standards**: the reference implementation mindset — DMN/BPMN/CMMN
  interoperability, explicit anti-lock-in positioning.

**vs DTRules.** We are not DMN and should say so plainly: our decision tables
predate DMN's hit policies but map loosely (FIRST/ALL ≈ F/C hit policies);
EL is our FEEL. Adopting DMN import/export would be the single biggest
interoperability move available to us — and also a large one. Their weakness
from our seat: models interpreted in a vendor cloud are hard to embed in a
Go binary and hard to hash-audit.

## Sapiens Decision — governance-first enterprise

**Source**: [methodandstyle.com/blog/dmn-tool-update](https://www.methodandstyle.com/blog/dmn-tool-update/)

- **Posture**: TDM-methodology repository with business glossary, role-based
  governance, audit trail; banks and insurers. DMN support arriving rather
  than native.
- **vs DTRules.** Their governance surface (roles, approvals, glossary) is
  what an enterprise buyer will ask us for and we do not have. What we have
  that they do not: an enforced *mechanical* contract — our funnel test and
  provenance hashes make bypass impossible rather than policy-forbidden.

## FlexRule — the analyst-market composite

**Sources**: [flexrule.com/platform](https://www.flexrule.com/platform/),
[flexrule.com/platform/capabilities/business-rules](https://www.flexrule.com/platform/capabilities/business-rules/)

- **Posture**: "Open Decision Intelligence Platform"; Forrester Wave AI
  Decisioning Q2-2025 and Gartner Market Guide presence. DMN CL3, plus
  tabular/tree/natural-language rule forms; API-first service composition.
- **vs DTRules.** Breadth play — many modeling forms, orchestration,
  analytics. Our counter is depth on one form done rigorously: decision
  tables with static advisory analysis (unreachable columns, redundant
  conditions, dated constants) they market as premium "validation" features.

## Sparkling Logic SMARTS — author-on-data

**Sources**: [sparklinglogic.com/smarts-rules-authoring](https://www.sparklinglogic.com/smarts-rules-authoring/),
[sparklinglogic.com/product](https://www.sparklinglogic.com/product/)

- **Posture**: the "Red Pen" metaphor — experts edit rules *directly on data
  samples*; SparkL SQL-like natural language; enterprise decisioning.
- **vs DTRules.** Author-on-data is genuinely good pedagogy and we have a
  seed of it: our scenario ratchet runs 500 cases and the rules validate
  themselves against expectations. Turning that loop into an authoring
  experience (edit rule → see 500 scenarios re-judge live) is an idea worth
  stealing outright.

## Market note: the LLM story

InRule shipped "InRule AI Core" (mid-2025): rules generated from natural-
language prompts. Vendors are converging on LLM-assisted *authoring*. Our
differentiator is the inverse and stronger claim: an **enforced machine
authoring interface** — the JSON-first CLI/MCP surface where an agent's
writes go through the same funnel, guards, compilation and provenance as a
human's, and *cannot* bypass them (the funnel test fails the build). Nobody
in this tranche enforces their API; they document it. For the Apr-2026
agentic challenge, that is the lead.

---

## Synthesis so far

**DTRules' defensible ground** (all mechanically enforced, not policy):
1. Excel as system of record with a *compiled*, hash-stamped, git-diffable
   artifact chain — auditability OpenRules' interpret-and-reload cannot offer.
2. A tiny embeddable Go runtime (one binary, 13MB loads) where the enterprise
   platforms are Java stacks or SaaS.
3. Machine authoring as an enforced contract — the agentic story.
4. Static rigor: advisory analysis, provenance, a 505-scenario self-validating
   ratchet.

**Real gaps**: no DMN interchange (isolation from the standard ecosystem);
no governance surface (roles, approvals, glossary); no visual modeler; a
community of one.

**Ideas worth adopting**: author-on-data (Sparkling Logic) on top of our
scenario corpus; DMN import as an on-ramp (Trisotech-compatible subset);
OpenRules-style LLM assist *inside* our enforced funnel.

## Not yet profiled (stubs)

Blue Polaris · RapidGen · Aletyx · Rules Matix · KU Leuven (research) ·
RuleML (standard). Same question set as above; add per tranche.

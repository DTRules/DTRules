# Competitive landscape: decision-management platforms vs DTRules

Living document for #849. Profiles cite vendor documentation, not memory;
"vs DTRules" sections are our own analysis. Platforms not yet profiled are
listed as stubs at the end — this is a tranche, not the finished survey.

Last updated: 2026-08-31. First tranche: OpenRules, Trisotech, Sapiens,
FlexRule, Sparkling Logic. Second tranche: Aletyx, RapidGen, Blue Polaris,
Rules Matix, KU Leuven (research), RuleML (standard) — the platform list from
#849 is now complete.

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

## Aletyx — the Drools engineers, gone independent

**Sources**: [aletyx.com](https://aletyx.com/),
[aletyx.ai/about-us](https://aletyx.ai/about-us/),
[aletyx.ai/pricing](https://aletyx.ai/pricing/),
[docs.aletyx.ai/core/decisions/dmn](https://docs.aletyx.ai/core/decisions/dmn/),
[kie.apache.org](https://kie.apache.org/)

- **Source of truth**: standard DMN files on disk, plus DRL for Drools-native
  rules. Their VS Code extension edits `.dmn`/`.bpmn` in the project tree, and
  they sell portability explicitly. A hosted layer (Decision Control, Control
  Tower) sits above for analyst authoring and promotion; whether it holds its
  own authoritative copy is not documented.
- **Author**: both, deliberately split — architects own infrastructure,
  analysts own decision policy. Visual DMN canvas with FEEL, a VS Code
  extension, and a browser AI assistant.
- **Execution**: Drools/Kogito, **DMN conformance level 3** across DMN 1.1–1.5
  — the most complete tier claimed by anyone here. JVM: Spring Boot, Quarkus,
  Kubernetes. The "30× faster" headline names no baseline or workload; treat
  as marketing.
- **Drift**: mostly dissolved — you run the file the analyst drew. The
  residual risk (hosted copy vs git, DMN vs the DRL beside it) is answered
  with process — RBAC, approval workflows, release records — not mechanism.
  No `verify` equivalent is documented.
- **AI**: an assistant ($95/user/month) that generates DMN from a policy,
  spreadsheet or prompt and presents edits as reviewable visual diffs; plus
  MCP so existing rules become deterministic guardrails for agents. Direction
  credible, depth unpublished — no model, eval or accuracy data.
- **Model**: hybrid. The engines are Apache-licensed upstream at **Apache KIE
  (incubating)** — Drools, jBPM, OptaPlanner, Kogito, all donated to the ASF —
  and Aletyx states it is not affiliated with the ASF. What they sell is a
  hardened Enterprise Build plus proprietary layers, **$25k–$200k/year**, flat
  fee, unlimited cores. Java. Their own GitHub org is small; the value lives
  upstream and in closed products.

**vs DTRules.** The incumbent we actually have to position against, and it made
the opposite bet at every layer: the model *is* the executable artifact, so
there is no compile step to keep honest and no drift problem to solve. Where we
spend real machinery — the authoring API, the `verify` gate, hash-stamped
artifacts — keeping Excel and XML in lockstep, they simply have one file. That
buys standards portability and a mature modeler; it costs the determinism
story, because a git-diffable postfix artifact you can replay byte-for-byte is
a stronger audit primitive than release records backed by a hosted control
plane. We win on artifact integrity, footprint and cost; they win on standards,
ecosystem, tooling maturity, and employing the people who wrote the largest
open-source rules engine on earth.

## RapidGen (Genius Suite) — DMN in, native machine code out

**Sources**: [rapidgen.com](https://rapidgen.com/),
[dmn-execution](https://rapidgen.com/dmn-execution/),
[Software Overview PDF](https://rapidgen.com/wp-content/uploads/RapidGen-Software-Overview.pdf),
[support/FAQ](https://rapidgen.com/support/)

- **Source of truth**: the DMN model, stated in a headed callout — *"Decision
  Model is Primary."* They author none of it: you model in your own tool and
  hand them DMN 1.3 XML.
- **Author**: analysts, in a third-party modeler. Their named enemy is the
  hand-coding step — "manually maintained code can become inconsistent with the
  decision model yielding a false impression of the organization's
  decision-making."
- **Execution**: compiled to **native machine code**, no VM. DMN XML →
  translator → RPL (their own decision-table language) → single-pass compiler →
  platform executable, with conditions inlined and rule state held in a machine
  register manipulated bitwise. Vendor claims 100k+ decisions/sec; no
  methodology published. Batch and data-pipeline shaped — reads XLS/CSV/JSON/
  SQL/ISAM and writes the same. **No embeddable library, HTTP decision service,
  or container deployment is documented**, and licensing is node-locked.
- **Drift**: largely dissolved by generation, with two caveats worth recording.
  RPL's table format "differs from that described in the DMN standard", so the
  translation is a vendor assurance a customer cannot diff; and step 3 of their
  flow binds DMN variables to data sources *outside* the model — a second
  artifact, quietly reintroduced.
- **AI**: about *governing* AI, not authoring rules — "caging" excessive model
  output and "ensembling" so no single model decides, plus sanity-checking
  AI-generated input. No LLM authoring claimed. A vendor post asserts explicit
  type checking that "none of the top five DMN modelling environments"
  supports: plausible, self-asserted, unverifiable.
- **Model**: commercial, closed, node-locked. DMN 1.3 import is the whole
  premise but **no conformance level is published** and nothing says whether
  FEEL and DRD semantics are covered or decision tables alone. Company active
  since 2005 (Companies House 05585419), one office, no public docs, flagship
  technical PDF © 2017. A credible niche specialist, not a market force.

**vs DTRules.** The closest architectural cousin in the survey: both refuse to
hand-maintain an implementation, both compile one authored artifact into
something fast, both argue "one source of truth, everything else is generated."
The divergence is which artifact is authoritative and how far compilation goes
— they take someone else's DMN and go all the way to native code for
throughput; we take Excel and stop at postfix on a small Go VM. That buys them
raw speed, DMN interchange and legacy data reach we do not have; it costs
portability, embeddability and inspectability, since the deliverable is a
node-licensed binary rather than a hash-stamped artifact anyone can replay.
The sharper contrast is on safety mechanics: our contract is enforced by a
gate a user can run, theirs — faithful translation, type checking,
completeness — are assurances with no public documentation to audit.

## Blue Polaris — a DMN front-end renting IBM's engine

**Sources**: [bluepolaris.com](https://bluepolaris.com/),
[decisionsfirst-modeler](https://bluepolaris.com/decisionsfirst-modeler/),
[recommended-software](https://bluepolaris.com/recommended-software/),
[agentic-ai-decision-agents](https://bluepolaris.com/agentic-ai-decision-agents/)

- **Source of truth**: a DMN model in **DecisionsFirst Modeler**, which the
  vendor calls "a front-end to your Business Rules Management Systems" — so the
  model is explicitly *not* the executable artifact.
- **Author**: analysts and decision architects, in a graphical DMN editor. The
  published features are modeling and review: version history with rollback,
  impact analysis, tag explorer, decision-service simulation, Excel export.
- **Execution**: nothing they build. A Premier IBM Gold partner routing
  execution to ODM, ADS and Cloud Pak. The rest is consulting, hosting and
  training.
- **Drift**: the structural weak point, and they do not claim to have solved
  it. Model and deployed ruleset are two artifacts in two tools; the answer is
  linkage and impact analysis, not a compiled pipeline. No statement that the
  model generates the executing rules, and no enforcement gate.
- **AI**: governance *over* AI rather than AI authoring — ODM decision services
  exposed to agent frameworks over MCP, with the explicit position that "large
  language models are fundamentally unsuited for critical business decisions."
- **Model**: commercial, product-plus-services weighted to services. Founder
  James Taylor co-submitted the DMN standard; no conformance level published.

*Health note:* the rebrand from Decision Management Solutions is incompletely
executed — as of 2026-08-30 both `decisionmanagementsolutions.com` and
`decisionsfirst.com` fail TLS with expired certificates, and much of the
indexed product documentation lives on those dead domains.

**vs DTRules.** The mirror image: they own modeling and rent execution, we own
the whole pipeline and have no visual modeler. That buys them the DMN
graphical surface, the analyst governance story and IBM's support contract —
three of the four gaps in our synthesis below. It costs them the thing our
contract is built around: with model and ruleset in separate tools they can
*trace* drift but cannot make it structurally impossible. Less a competitor
than a complement-shaped rival, overlapping only on the claim to be where
decision logic is authored.

## Rules Matix — a consultancy, not a platform

**Sources**: [rulesmatix.com](https://rulesmatix.com/),
[services](https://rulesmatix.com/services/),
[dmcommunity sponsors](https://dmcommunity.org/sponsors/current-sponsors/)

Not a competitor, and the survey should say so plainly. Rulesmatix sells hours,
not software: rule harvesting, business object model design, repository review,
mentoring and testing, delivered on **Sparkling Logic SMARTS, FICO Blaze
Advisor, Drools, IBM ODM and OpenRules** — every one of them a product we would
compete with directly. No engine, no repository format, no DMN or Excel
mention, no AI claim. US firm, founded 2007; blog archive stops at 2021–22
though they remain a listed DMCommunity sponsor.

**vs DTRules.** The useful signal is what their service list reveals about
where client money goes: vendor selection, rule harvesting, object model
design, repository review, testing. Four of those five are people-work that
exists precisely because the underlying platforms make safe authoring hard —
which is the problem we attack mechanically with the authoring contract, the
advisory pass and the scenario corpus. A firm like this is a distribution
channel rather than a rival, but with a portfolio built on six-figure
commercial BRMS products, a free Go engine with no DMN and no governance
surface gives them nothing to bill against.

## KU Leuven — cDMN, or a decision table read as a constraint

**Sources**: [arXiv 2005.09998](https://arxiv.org/abs/2005.09998),
[TPLP 23(3):535-558](https://www.cambridge.org/core/journals/theory-and-practice-of-logic-programming/article/abs/tackling-the-dm-challenges-with-cdmn-a-tight-integration-of-dmn-and-constraint-reasoning/D38F60660726639717319D81639C0809),
[cdmn.readthedocs.io](https://cdmn.readthedocs.io/),
[idp-z3.be](https://www.idp-z3.be/)

- **What it is**: a research programme, not a product. **IDP-Z3** is a knowledge
  base engine for FO(·) — first-order logic with types, aggregates, inductive
  definitions and arithmetic — and **cDMN** is a spreadsheet-authored extension
  of OMG DMN that compiles to it. Anchor paper: Vandevelde, Aerts & Vennekens,
  TPLP 2023 (Best Paper, RuleML+RR 2020).
- **Core idea**: the *Knowledge Base paradigm* — state the knowledge once,
  declaratively, then apply many generic inferences to it rather than one
  hard-coded direction. cDMN adds four things to DMN: a typed **glossary**;
  **constraint tables** (hit policy `E*`, where every row must hold, so a table
  asserts a requirement instead of computing an output); **quantification**
  ("For every Person p1" in one row); and **data tables** separating instance
  data from rule knowledge. A **Goal table** then picks the inference —
  `Get x models`, `Minimize/Maximize`, or `Propagate`.
- **Status**: active but research-scale. IDP-Z3 last active 2026-08; the cDMN
  solver 2026-03; PyPI `cdmn` 2.1.2 (2024-05). Two and fifteen GitLab stars
  respectively. **No evidence OMG has absorbed cDMN into DMN** — an academic
  extension, not a standards track.
- **Relationship to DMN**: a deliberate superset — everything in standard DMN
  except boxed expressions and the `C` hit policy. Input is `.xlsx`.
- **Tooling**: runnable and maintained — a Python converter plus a browser
  editor, backed by IDP-Z3. **GPLv3**, which is a licensing non-starter for
  embedding.

**vs DTRules.** The idea worth stealing is the **constraint table** and the
**Goal table**: that a table of conditions is more useful when it can be read
as an assertion the world must satisfy and not only as a function to evaluate,
and that *which inference you want* belongs in the model as an explicit
artifact. Our advisory pass already reasons statically over column overlap for
redundancy and subsumption — that machinery is the same shape as constraint
propagation, so a `Propagate`-style "given these three inputs, what is already
forced?" is a plausible extension of analysis we have rather than a new engine.
Against that: cDMN compiles onto a general SMT solver, so its runtime is Python
plus Z3, where we compile to postfix on a small Go VM with predictable
execution cost — the right shape for a batch tax corpus. And note that both
projects independently landed on **spreadsheets as the authoring surface**.
That is useful external validation of the Excel bet, from people optimising
purely for domain-expert readability. GPLv3 means the idea is borrowable and
the code is not.

## RuleML — a living conference attached to a dead specification

**Sources**: [deliberation-ruleml](https://github.com/RuleML/deliberation-ruleml),
[W3C RIF Overview](https://www.w3.org/TR/rif-overview/),
[W3C SWRL submission](https://www.w3.org/submissions/SWRL/),
[RuleML+RR 2026](https://2026.declarativeai.net/ruleml-rr),
[OMG DMN](https://www.omg.org/spec/DMN)

- **What it is**: a family of XML rule-interchange schemas — Deliberation,
  Reaction and Consumer RuleML, plus the MYNG modular schema generator —
  governed by a non-profit founded in 2000. The name also denotes RuleML+RR,
  an annual academic conference, which is now the only part with a pulse.
- **Core idea**: don't build one universal rule language, build a *lattice* of
  sublanguages so a system declares exactly the expressivity it supports and
  rules interchange where the schemas overlap.
- **Status**: **historical as a specification.** `deliberation-ruleml`'s last
  commit is 2018-09-06 and there is no final 1.03 release tag; `reaction-`
  stopped in 2017, `consumer-` in 2015. `wiki.ruleml.org` returns 404, and
  `ruleml.org/1.03/` now redirects to a generic WordPress blog about web
  design with no specifications on it. Meanwhile the conference reached its
  10th edition in Vilnius, August 2026.
- **Relationship to DMN**: essentially none, and that is the point. RuleML's
  interchange ambitions ran through the **semantic web** stack. W3C RIF was
  declared complete in 2013; SWRL never became a Recommendation at all — it is
  a 2004 Member Submission carrying W3C's "no endorsement" disclaimer. DMN
  went the other way and took the commercial ground: OMG DMN 1.5 adopted
  August 2024.
- **Tooling**: effectively unmaintained. PSOATransRun last pushed 2021, OO
  jDREW 2012. No reference implementation worth building against.

**vs DTRules.** The lesson is negative and cheap to learn: RuleML is the
cautionary tale for interchange-format ambition. Twenty-five years produced a
beautifully factored schema lattice, W3C Recommendations and a healthy
conference — and no surviving toolchain, no reachable spec URLs and no
commercial adopters, while DMN took the market with a cruder model. **Our
missing DMN interchange is a real gap; RuleML/RIF/SWRL support would not be a
gap, it would be a liability**, and this profile closes that question. The one
idea still worth borrowing is MYNG's conformance-profile discipline: if we do
implement DMN import, declaring precisely which conformance level, which hit
policies and which FEEL subset — as cDMN does, plainly, in a sentence — is more
honest and more useful than claiming "DMN support" wholesale.

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

**What the second tranche changed.** Two findings sharpen the position:

1. **Aletyx is the real competitor**, not the modeling front-ends. They employ
   the engineers who wrote Drools, Kogito and jBPM, claim DMN conformance
   level 3, and sell the governance surface and visual modeler we lack — from a
   $25k/year floor. We compete below them on footprint and cost, and above them
   on artifact integrity: they have no mechanical equivalent of `verify`.
2. **Nobody else compiles.** Of eleven entries, only RapidGen also refuses to
   interpret — and it goes to native machine code from someone else's DMN,
   losing embeddability and inspectability to get there. Compiling to a
   git-diffable, hash-stamped, replayable artifact remains ours alone.

**Ideas worth adopting**: author-on-data (Sparkling Logic) on top of our
scenario corpus; cDMN's **constraint tables and Goal tables** (KU Leuven), which
our advisory pass is already shaped to support; DMN import as an on-ramp
(Trisotech-compatible subset), declared with cDMN's conformance-profile
honesty rather than as blanket "DMN support"; OpenRules-style LLM assist
*inside* our enforced funnel.

**Settled by this tranche**: RuleML/RIF/SWRL interchange is **not** worth
implementing — the schemas are unmaintained and their canonical URLs 404.
DMN is the only live interchange target.

## Coverage

Every platform named in #849 is now profiled. Two turned out not to be
competitors and are recorded as such rather than padded: **Rules Matix** is a
consultancy that sells hours on other vendors' engines, and **RuleML** is a
specification family whose canonical URLs no longer resolve.

Worth revisiting when they move: **InRule** (profiled only in the LLM market
note below), **FICO Blaze Advisor** and **IBM ODM** — the two incumbents that
show up repeatedly as what everyone else integrates with or migrates off.

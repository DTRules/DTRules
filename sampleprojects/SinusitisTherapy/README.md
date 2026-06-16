# SinusitisTherapy — Agentic Medical Services

A DTRules sample project for the **Challenge Apr-2026: Agentic Medical
Services Solutions** (Decision Management Community).

Source: https://dmcommunity.org/challenge-apr-2026/

The challenge explores how an LLM can orchestrate rule-based decision
services. It defines three *loosely coupled* medical services that an LLM
should call when helping a doctor determine a therapy for a patient with
**Acute Sinusitis**. This project implements those three services as
independent DTRules decision tables, plus a thin orchestration table that
wires them together in the correct order.

## The three services

| # | Service | Entry table | What it does |
|---|---------|-------------|--------------|
| 1 | Medication & dosing | `Determine_Medication_And_Dosing` | Picks the drug of choice and the dose / frequency / duration |
| 2 | Creatinine clearance | `Determine_Creatinine_Clearance` | Computes CCr from age, lean body weight and PCr |
| 3 | Drug interactions | `Check_Drug_Interactions` | Flags conflicts between the recommended drug and the patient's active medications |

`Determine_Therapy` is the top-level entry point. It runs the three services
in dependency order (CCr first, because dosing depends on it; medication
selection before the interaction check, because interactions depend on the
chosen drug):

```
Determine_Therapy
├── perform Determine_Creatinine_Clearance        (service 2)
├── perform Determine_Medication_And_Dosing        (service 1)
│   ├── perform Select_Medication
│   └── perform Determine_Dose
└── perform Check_Drug_Interactions                (service 3)
```

Because the services are loosely coupled, an LLM agent may also call any one
of them directly (e.g. just `Determine_Creatinine_Clearance`).

## Rules implemented

**Service 1 — Medication & dosing**

- Penicillin-allergic → **Levofloxacin** (beta-lactam cross-reactivity, so
  this overrides the age rule).
- Otherwise, age ≥ 18 → **Amoxicillin**; age < 18 → **Cefuroxime**.
- Age 15–60 → **500 mg** every 24 h for 14 days.
- Age < 15 or > 60 → **200 mg** every 24 h for 14 days.
- Renal impairment (PCr > 1.4 **and** CCr < 50) → **200 mg** every 24 h for
  14 days (overrides the age-based dose).

**Service 2 — Creatinine clearance** (Cockcroft–Gault form used by the
challenge):

```
CCr [mL/min] = ((140 − age) × Lean Body Weight [kg]) / (PCr × 72)
```

**Service 3 — Drug interactions**

Conflicts between the recommended drug and the patient's active medications
are listed in [`ConflictingMedications.csv`](ConflictingMedications.csv) and
encoded in the `Check_Drug_Interactions` decision table. Each detected
conflict appends a warning to `result.warnings`.

## Reference request

> "I have a patient diagnosed with Acute Sinusitis. He is 58 years old,
> weighs 78 kg, and has a creatinine level of 1.85. Keep in mind that he is
> Penicillin-allergic and takes Coumadin."

Expected determination (see `testfiles/TestScenarios/ChallengeExample`):

- Drug: **Levofloxacin** (penicillin-allergic)
- CCr = ((140 − 58) × 78) / (1.85 × 72) ≈ **48.02 mL/min**
- Dose: **200 mg** every 24 h for 14 days (PCr 1.85 > 1.4 and CCr < 50 →
  renal adjustment)
- Warning: Levofloxacin may potentiate the anticoagulant effect of
  warfarin/Coumadin — monitor coagulation.

## Layout

```
SinusitisTherapy/
├── ConflictingMedications.csv      # drug-interaction knowledge base (service 3)
├── xml/
│   ├── sinusitis_edd.xml           # entities: patient, medication, result, constants
│   ├── sinusitis_dt.xml            # the six decision tables
│   └── sinusitis_map.xml           # input XML → entity mapping
└── testfiles/TestScenarios/
    ├── ChallengeExample/input.xml  # the reference request above
    ├── AdultStandard/input.xml     # healthy adult → Amoxicillin 500 mg
    └── PediatricPatient/input.xml  # child → Cefuroxime 200 mg
```

## Running

The Go integration test `TestSinusitisTherapy`
(`pkg/dtrules/sinusitis_test.go`) loads the project, runs each scenario
through `Determine_Therapy`, and asserts the drug, dose, CCr, and warnings:

```bash
go test ./pkg/dtrules/ -run TestSinusitisTherapy -v
```

Rules are authored in EL in the `*_dsl` tags and compiled to postfix with:

```bash
dtrules compile sampleprojects/SinusitisTherapy
```

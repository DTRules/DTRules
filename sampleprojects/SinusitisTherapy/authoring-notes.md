# Authoring Notes

## Files
- `therapy_dt.xml` [1000-1099] — orchestrator: entry point that performs the three services in dependency order
- `service2_creatinine_dt.xml` [2000-2099] — service 2: creatinine clearance (Cockcroft-Gault)
- `service1_medication_dt.xml` [3000-3099] — service 1: medication selection + dosing
- `service3_interactions_dt.xml` [4000-4099] — service 3: drug-interaction warnings

## Conventions

- One file per loosely-coupled service, plus `therapy_dt.xml` for the
  orchestrator. The orchestrator performs services **by name**, so table
  numbers/files can be reorganized freely.
- Run order matters and is encoded in `Determine_Therapy`: service 2 (CCr)
  before service 1 (dosing reads `result.ccr`) before service 3 (interactions
  read `result.recommended_drug`).
- Thresholds and dose constants live in the `constants` entity in the EDD —
  never inline literals in the logic tables.
- The drug-interaction conflicts in `service3_interactions_dt.xml` mirror the
  canonical list in `ConflictingMedications.csv`; keep the two in sync.
- New medication families get their own file with a fresh, non-overlapping
  range (the gaps of ~1000 between services leave room to insert).

## Change log
- 2026-06-16 — add file `therapy_dt.xml` [1000-1099] — "orchestrator: entry point that performs the three services in dependency order"
- 2026-06-16 — move `Determine_Therapy`: `sinusitis_dt.xml` → `therapy_dt.xml` — "orchestrator: entry point that performs the three services in dependency order"
- 2026-06-16 — add file `service2_creatinine_dt.xml` [2000-2099] — "service 2: creatinine clearance (Cockcroft-Gault)"
- 2026-06-16 — move `Determine_Creatinine_Clearance`: `sinusitis_dt.xml` → `service2_creatinine_dt.xml` — "service 2: creatinine clearance (Cockcroft-Gault)"
- 2026-06-16 — add file `service1_medication_dt.xml` [3000-3099] — "service 1: medication selection + dosing"
- 2026-06-16 — move `Determine_Medication_And_Dosing`: `sinusitis_dt.xml` → `service1_medication_dt.xml` — "service 1: medication selection + dosing"
- 2026-06-16 — move `Select_Medication`: `sinusitis_dt.xml` → `service1_medication_dt.xml` — "service 1: medication selection + dosing"
- 2026-06-16 — move `Determine_Dose`: `sinusitis_dt.xml` → `service1_medication_dt.xml` — "service 1: medication selection + dosing"
- 2026-06-16 — add file `service3_interactions_dt.xml` [4000-4099] — "service 3: drug-interaction warnings"
- 2026-06-16 — move `Check_Drug_Interactions`: `sinusitis_dt.xml` → `service3_interactions_dt.xml` — "service 3: drug-interaction warnings"
- 2026-06-16 — delete empty file `sinusitis_dt.xml` (last table moved out: service 3: drug-interaction warnings)







# Authoring Notes

## Files
- `orchestrator_dt.xml` [100-199] — entry/orchestration tables live above the per-state ranges
- `states/AK_corp_dt.xml`
- `states/AL_corp_dt.xml`
- `states/AR_corp_dt.xml`
- `states/AZ_corp_dt.xml`
- `states/CA_corp_dt.xml`
- `states/CO_corp_dt.xml`
- `states/CT_corp_dt.xml`
- `states/DC_corp_dt.xml`
- `states/DE_corp_dt.xml`
- `states/FL_corp_dt.xml`
- `states/GA_corp_dt.xml`
- `states/HI_corp_dt.xml`
- `states/IA_corp_dt.xml`
- `states/ID_corp_dt.xml`
- `states/IL_corp_dt.xml`
- `states/IN_corp_dt.xml`
- `states/KS_corp_dt.xml`
- `states/KY_corp_dt.xml`
- `states/LA_corp_dt.xml`
- `states/MA_corp_dt.xml`
- `states/MD_corp_dt.xml`
- `states/ME_corp_dt.xml`
- `states/MI_corp_dt.xml`
- `states/MN_corp_dt.xml`
- `states/MO_corp_dt.xml`
- `states/MS_corp_dt.xml`
- `states/MT_corp_dt.xml`
- `states/NC_corp_dt.xml`
- `states/ND_corp_dt.xml`
- `states/NE_corp_dt.xml`
- `states/NH_corp_dt.xml`
- `states/NJ_corp_dt.xml`
- `states/NM_corp_dt.xml`
- `states/NV_corp_dt.xml`
- `states/NY_corp_dt.xml`
- `states/OH_corp_dt.xml`
- `states/OK_corp_dt.xml`
- `states/OR_corp_dt.xml`
- `states/PA_corp_dt.xml`
- `states/RI_corp_dt.xml`
- `states/SC_corp_dt.xml`
- `states/SD_corp_dt.xml`
- `states/TN_corp_dt.xml`
- `states/TX_corp_dt.xml`
- `states/UT_corp_dt.xml`
- `states/VA_corp_dt.xml`
- `states/VT_corp_dt.xml`
- `states/WA_corp_dt.xml`
- `states/WI_corp_dt.xml`
- `states/WV_corp_dt.xml`
- `states/WY_corp_dt.xml`

## Conventions

## Change log
- 2026-08-05 — add table `Calculate_OH_Income_Adjustments` to `states/OH_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_OH_State_Tax` to `states/OH_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_TN_State_Tax` to `states/TN_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_TX_Income_Adjustments` to `states/TX_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_TX_State_Tax` to `states/TX_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_WA_Income_Adjustments` to `states/WA_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add table `Calculate_WA_State_Tax` to `states/WA_corp_dt.xml` — "standard-name wrapper so state dispatch resolves (#948 Phase 3)"
- 2026-08-05 — add file `orchestrator_dt.xml` [100-199] — "entry/orchestration tables live above the per-state ranges"
- 2026-08-05 — add table `Run_Corporate_Tax` to `orchestrator_dt.xml` — "entry table: dispatches to the state trio by apportionment.state_code (#948 Phase 3)"


# Authoring Notes

## Files
- `TaxReturn_dt.xml`
- `states/AK_dt.xml`
- `states/AL_dt.xml`
- `states/AR_dt.xml`
- `states/AS_dt.xml`
- `states/AZ_dt.xml`
- `states/CA_dt.xml`
- `states/CO_dt.xml`
- `states/CT_dt.xml`
- `states/DC_dt.xml`
- `states/DE_dt.xml`
- `states/FL_dt.xml`
- `states/GA_dt.xml`
- `states/GU_dt.xml`
- `states/HI_dt.xml`
- `states/IA_dt.xml`
- `states/ID_dt.xml`
- `states/IL_dt.xml`
- `states/IN_dt.xml`
- `states/KS_dt.xml`
- `states/KY_dt.xml`
- `states/LA_dt.xml`
- `states/MA_dt.xml`
- `states/MD_dt.xml`
- `states/ME_dt.xml`
- `states/MI_dt.xml`
- `states/MN_dt.xml`
- `states/MO_dt.xml`
- `states/MP_dt.xml`
- `states/MS_dt.xml`
- `states/MT_dt.xml`
- `states/NC_dt.xml`
- `states/ND_dt.xml`
- `states/NE_dt.xml`
- `states/NH_dt.xml`
- `states/NJ_dt.xml`
- `states/NM_dt.xml`
- `states/NV_dt.xml`
- `states/NY_dt.xml`
- `states/OH_dt.xml`
- `states/OK_dt.xml`
- `states/OR_dt.xml`
- `states/PA_dt.xml`
- `states/PR_dt.xml`
- `states/RI_dt.xml`
- `states/SC_dt.xml`
- `states/SD_dt.xml`
- `states/TN_dt.xml`
- `states/TX_dt.xml`
- `states/UT_dt.xml`
- `states/VA_dt.xml`
- `states/VI_dt.xml`
- `states/VT_dt.xml`
- `states/WA_dt.xml`
- `states/WI_dt.xml`
- `states/WV_dt.xml`
- `states/WY_dt.xml`

## Conventions

## Change log
- 2026-09-13 — renamed table Dispatch_NonResident_State_Tax -> NR_State_Pass
- 2026-09-13 — delete table `NR_State_Pass` from `TaxReturn_dt.xml` — "Non-resident dispatch pass did not execute; parked until the cause is understood (#1177)"
- 2026-09-13 — delete table `No_State_Income_Tax` from `TaxReturn_dt.xml` — "Only existed as the fallback for the parked non-resident dispatch (#1177)"
- 2026-09-25 — delete table `Filter_Rental_Property` from `TaxReturn_dt.xml` — "Unreached, and redundant: Process_Rental_Income filters rentals in its own context (#1207)"





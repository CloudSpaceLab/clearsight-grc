# Spreadsheet vendor proposal acceptance

Verified locally on 9 September 2026. The source workbooks were read from the user-supplied local folder without alteration, upload or inclusion in the repository. They were passed through the real `Extract` → `InspectTabularArtifact` → `ProposeFormTemplate` functions, without a mocked extraction or static proposal.

| Source | Retained rows | Proposed fields | Exact row anchors | Sections including General | Limits |
| --- | ---: | ---: | ---: | ---: | --- |
| Sample Third-Party Risk Register (1).xlsx | 6 including header | 5 | 5 | 2 | All five original deadlines retained; two long rows have explicit shortened-context warnings |
| NDPA_Compliance_Checklist (1).xlsx | 48 including header | 47 | 47 | 14 | No omitted fields or shortened context |

Both extractions returned `EXTRACTED`. Every proposed field remains optional and unscored. Every checklist field has a scope/source/applicability review warning; every historical field has a vendor/service/current-status review warning. Neither source reached the 200-field or 20-section limits. Shortening a question label/description does not remove its complete source row or source anchor and is reported with `ROW_CONTEXT_TRUNCATED`.

Six regression tests were added before the corresponding behavior. Observed failures included the six checklist columns becoming questions instead of two requirement rows, the seven register columns becoming questions instead of two historical follow-ups, old metadata-only sources being accepted as column questions, and a long historical finding displacing its recorded deadline. The final tests also cover stable row identities, conditional/time context without inferred applicability, bounded retention/field limits and cell newlines that must not invent rows. Existing generic table and DOCX proposal tests continue to pass.

`go test ./internal/documentimport -count=1` passed. The affected focused spreadsheet/proposal/XLSX suite also passed. `git diff --check -- internal/documentimport` reported no whitespace errors. All test fixtures are synthetic; no private workbook text was copied into committed fixtures.

The user path is: upload the source in Forms, inspect the row-based proposal and its warnings, choose fields, create a draft, edit the questions and collection rules, then use ordinary independent approval before selecting the active form for onboarding or reassessment. Historical register rows require manual matching to the correct vendor/service and separation of internal actions from vendor requests. Uploading a historical finding is not evidence that the finding is still open or that a vendor has satisfied it.

This receipt covers importer behavior and source counts. The parent release task owns rendered proposal-review checks, full application tests, integration and deployment. It does not claim automatic source-law validation, requirement-set activation, vendor identity matching, creation/closure of issues, or a legally verified compliance result.

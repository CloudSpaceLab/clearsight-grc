# Multi-framework starter rollout

## Verified on 17 November 2026

The isolated PostgreSQL acceptance test exercises draft saving, submission, independent adoption approval and pinned native Forms question groups for NIST CSF 2.0, PCI DSS v4.0.1, Nigeria NDPA, UK DPA, DORA and SOC 2. It also checks ISO adoption and verifies that rerunning starter backfill preserves existing revisions, links and staff wording. The six new cases use test-only approved Form revisions. They do not establish that questionnaire content, production approvals or assessment delivery have been accepted.

## Source review queue

| Pack | Publisher source | Review still needed |
| --- | --- | --- |
| NIST CSF 2.0 | [NIST Core Reference Tool](https://csrc.nist.gov/Projects/cybersecurity-framework/Filters) | Map each prompt to the relevant Core categories and subcategories. Record the uncovered outcomes. |
| PCI DSS v4.0.1 | [PCI SSC Document Library](https://www.pcisecuritystandards.org/document_library/) | Check exact requirement IDs against the selected edition. Confirm scoped payment systems and applicable validation route. |
| Nigeria NDPA | [NDPC Act](https://ndpc.gov.ng/wp-content/uploads/2024/03/Nigeria_Data_Protection_Act_2023.pdf), [GAID 2025](https://ndpc.gov.ng/wp-content/uploads/2025/03/NDP-ACT-GAID-2025-MARCH-20TH.pdf) | Review legal basis, rights, safeguards, incident obligations and applicable oversight guidance separately. |
| UK DPA | [Revised Act](https://www.legislation.gov.uk/ukpga/2018/12/contents) | Review general processing and the relevant UK GDPR provisions before mapping rights and safeguards. Identify which statutory parts apply to the Program. |
| DORA | [EUR-Lex regulation](https://eur-lex.europa.eu/eli/reg/2022/2554/oj) | Refine the existing chapter references to articles and applicable technical measures. |
| SOC 2 | [AICPA Trust Services Criteria](https://www.aicpa-cima.com/resources/download/2017-trust-services-criteria-with-revised-points-of-focus-2022) | Agree the system boundary, examination period and categories before approving criteria mappings. |
| ISO/IEC 27001 | [ISO edition](https://www.iso.org/standard/27001), [2024 amendment](https://www.iso.org/standard/88435.html) | Cross-check clause and Annex A references against the licensed edition and amendment. |
| EU GDPR | [EUR-Lex regulation](https://eur-lex.europa.eu/eli/reg/2016/679) | Verify existing article references against scoped obligations and identify gaps. |

Keep starter mapping proposals separate from approved catalog entries. Record the source edition and locator, the Program scope and exclusions, the pinned question group, and the reviewer decision before treating a mapping as approved.

## Runtime work pending

- Question validity needs a versioned rule on the question, a documented start date, and a current or expired result calculated against the pinned response. The existing collection expiry and renewal path provides whole-response behavior.
- External answer validation needs an approved source binding and rule. Use exact connector and rule revisions, retain the validation result and provenance, and distinguish mismatches from an unavailable or stale source.
- Scheduled dispatch needs an approved schedule and deterministic responder selection within adopted units. Recheck authorization and pinned Forms versions at dispatch time. Reuse one request per occurrence and recipient across retries.
- Browser acceptance for each remaining pack must cover actual Form approval, adoption, submission and review with fictional evidence. Validate automation time boundaries, unavailable connectors and duplicate prevention before an authorized production validation run.

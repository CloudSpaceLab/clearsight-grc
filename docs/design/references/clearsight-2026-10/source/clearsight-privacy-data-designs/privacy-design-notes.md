# ClearSight — Privacy & Data | design 08

## Screens
- Overview: four current-posture metrics, ROPA review coverage by department, DPIA decision stages, period-filtered incidents, three priority reviews and a compact rotating Copilot.
- ROPA register: searchable processing activity list, lawful basis, data subjects, owner, review dates and record drawer. On narrow screens it becomes readable cards.
- DPIA: screening → required → DPO review → approval. Approval and screening remain separate, including a go-live decision.
- Breaches: incident scope, investigation, affected subjects, required regulator notification decision, closure and owner.
- Vendor privacy: required DPO check, status, contract documents and activation condition.
- Data subject requests: rights request and due-date tracking.

## Interaction contracts
- Date control changes the **incident activity series only**. Current processing records, DPIAs, breach investigations and vendor reviews are as-of posture, not historical counts.
- Branch/department/group filters demonstrate scope selection with a bounded set of synthetic records. When a filter is applied, enterprise-wide metrics show an unknown state rather than fabricating a smaller population total.
- Copilot supports exactly three operator-selected suggestions, each opening a related sample record, without generating compliance claims.
- New record actions open non-persistent prototype drafts; no submission, approval, notification or upload takes place.
- Intended implementation reuses the existing ClearSight ROPA register, Programs/Forms, Evidence, Vendors and Work/approval contracts. Do not create another forms or workflow engine.

## Visual checks
- Rendered desktop at 1880 CSS pixels at 2x device scale.
- Rendered mobile at 390 CSS pixels at 2x device scale.
- Tested ROPA text filter, tab navigation, drawers, Copilot navigation, date change and organization filters. No reported runtime JS errors; no horizontal overflow at 390/1880 width.
- Figures and names are **illustrative, not stored ClearSight data**.

## Implementation gates
- Do not classify a review due date as missed if it is only upcoming; schedule events must use the bank time zone and the approved review cycle.
- No pending DPIA, empty answer, accepted file or screening submission should be interpreted as approval.
- Do not infer data subject affected counts or notification obligations from an incident classification.
- Mandatory DPO approval must block vendor activation where policy applies, even when other vendor evidence is complete.
- Unknown, stale and missing ROPA records must not become current/compliant merely because a workflow completed.

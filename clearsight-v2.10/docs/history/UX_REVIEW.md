> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

# V2.4 interaction review and implementation decisions

## The actual problem

The previous implementation used a generic right drawer for tasks that had different cognitive requirements: initial registration, investigation, maintenance and approval. A larger drawer would not solve that mismatch. An owner needs staged guidance; a returning operator needs direct access to a record section; a reviewer needs source context alongside the decision; an analyst needs list position and filters to survive a drill-down.

## Interaction rules applied

**Wizard:** a new object or bounded assessment requiring interdependent configuration, validation, ownership and a final review. Save partial work. Show where the user is, what remains, and the fixed target. Final submission must revalidate all steps, even when the user visits steps out of order. Implemented for ROPA, DPIA, vendor engagement, risk, loss and RCSA setup; the existing framework-adoption journey remains intact.

**Full workspace:** a durable object investigated or maintained over time. It needs a stable link, meaningful tabs, history and contextual relationships. Implemented for processing activity, DPIA, rights/incident case, risk, loss and control. An existing activity can open revision directly in its relevant section; maintenance is not forced through a blank registration flow.

**Review workspace:** evidence/context and decision visible together, with a readable rationale area. Full-page evidence and requirement review preserve current evidence, scope and independent-review gates. Privacy review distinguishes publishing an inventory record from approving processing or satisfying a requirement.

**Inline editing:** a person composing or adjusting a form should edit the document itself. V2.3 question/grid editing is preserved. There is no permanent inspector added back to the form.

**Small centered dialog:** a short atomic choice or confirmation, such as recording one recovery, assigning one value or discarding an unsent draft. These remain intentionally modal. The goal is not to ban all dialogs or make every task a wizard.

## Privacy information architecture

Overview is for the day's work: submitted records, overdue review, unresolved assessment work and links to cases. There is no invented privacy-compliance score.

Processing register is the inventory. Purpose/basis rows, data category sets, system rows, recipients/engagements, retention rules, control links and screening are distinct records within the activity. A table field cannot adequately represent this whole model.

Sharing & transfers derives from the activity relationships; it does not create a second vendor register. Recipient geography is context, not a declaration that a transfer is lawful.

Impact assessments are linked to a specific activity revision. They assess risks to individuals, not only the bank's loss or reputation. Proposed mitigations, implementation evidence and review decisions remain distinct. Open mitigation work reuses the shared task queue; record and case review work is projected into the same queue rather than copied into a privacy inbox.

Requests and incidents have short intake and long-lived handling workspaces. Unknown inventory linkage does not block initial recording. Partial investigation saves do not demand a premature conclusion. Date/clock assumptions are explicit, not silently inferred from a universal deadline.

## Revisions and review

Draft edits do not change a published record. Submission locks the proposal in the demo UI. A change request returns it to the owner; publication appends a snapshot and preserves earlier revisions. DPIA decisions validate the related activity revision. Reviewers cannot approve their own submissions in the demonstrated workflow.

The production version needs authenticated actors, authorization enforced server-side, optimistic concurrency, immutable accepted snapshots and durable event retention. Browser-side checks are solely a design demonstration.

## Navigation and scope

Branch → Department → Unit is a persistent list/reporting scope. A creation draft owns an explicit target; changing navigation scope must not reassign it. Scoped detail guards preserve context rather than silently expanding it. Return navigation restores list filtering and scroll. Sources that are global templates remain reusable; program/assessment bindings remain specific.

Risk dashboard totals continue to use the selected scope and loss occurrence period. The risk-record loss tab is explicitly lifetime-to-snapshot, not mislabeled as the dashboard period. Recoveries remain received-to-snapshot amounts for selected events; no FX aggregation is introduced.

## Deliberate remaining limitations

The prototype remains a connected design reference, not a complete enterprise front end. UI test coverage is Chromium, not full accessibility certification. Evidence has metadata rather than binary previews. Full offline synchronization, multiuser conflicts, bulk ROPA imports, multientity responsibility, formal legal clock handling, vendor portal authentication and regulator communication need production implementation. The ROPA CSV is a summary export, not a claimed regulator-ready comprehensive dossier; single-record JSON retains the richer local relationships.

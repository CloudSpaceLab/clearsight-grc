# Issue workspace reuse and board-brief design

**Status:** Proposed for implementation review
**Date:** 6 October 2026

## Purpose

Make each program issue immediately understandable and actionable without creating a parallel case-management, form-distribution, collaboration, loss, or reporting subsystem. A busy bank stakeholder must be able to establish the applicable organisation area, accountable owner, deadline and impact from the issue list and from the issue header, then complete the appropriate governed action.

## Scope and non-goals

This change covers issue summaries, the issue record workspace, existing form/distribution routes, linked loss and source context, and an executive board brief generated through the established report service.

It does not create:

- a second activity or comment model;
- a second action assignment, update-request, or email-notification workflow;
- an editable loss ledger inside an issue;
- a separate issue-form or vendor-request database model when the existing governed form distribution can express the request;
- a browser-only or ungoverned export path; or
- a duplicate organisation classification field.

`organization_scope_id` remains the structured department, group or unit applicability. The existing free-text `scope.affected_area` remains the affected service, product, process or system. They are distinct facts and must not be merged in UI, import, or command logic.

## Remote integration boundary

`origin/feat/346-s5-loss-risk-refinement` is a divergent remote branch, not a fast-forward of `main`. It contains the reusable named-owner, organisation-label, recovery and current-risk-position work needed by the issue experience. It must be integrated while retaining the report-download fixes already present on `main` (`dde462957` and its ancestry).

The integration sequence is:

1. bring current `main` into the loss/risk branch or merge the branch into a current-main integration branch;
2. resolve only real overlap in routing and application-shell files, retaining both the report download authority fix and loss/risk improvements;
3. run the targeted loss/risk and report checks before treating that branch as an issue-workspace dependency; and
4. preserve the user’s untracked presentation files throughout.

No issue-workspace change may reimplement the branch’s loss formatting, named-owner resolution or organisation-path presentation.

## List experience

Every issue row, whether reached from a Program or the general Issues and changes register, has a compact fact rail before optional detail:

| Fact | Source | Presentation |
| --- | --- | --- |
| Applies to | `organization_scope_id`, resolved only within the current actor’s authorised organisation view | Department / group / unit path, or an explicit unavailable/not recorded state |
| Owner | Stored accountable owner resolved through the current authority read | Name, or an explicit recorded-owner-unavailable state; never a raw principal ID |
| Due | `due_at` | Local date plus overdue/current state |
| Impact | Existing Matter priority | Human label and status tone; no duplicate severity attribute |

The summary read resolves these display values in the existing bounded, keyset-paginated query path. The browser must not load individual issue records merely to decorate a page of summaries. Access filtering remains in the repository and API, not in a browser filter.

`Affected service or process` is available in expanded/list detail only where recorded. It does not replace `Applies to`.

## Issue record layout

The existing `MatterRecordWorkspace` remains the single issue record. Its header repeats the four fixed facts and retains current status and record version. The main column uses concise task tabs; the existing right activity rail remains the source of activity history and communication.

### Overview

Overview contains:

- department/group/unit applicability and affected service/process;
- accountable owner, due date, impact and program links;
- a compact linked-loss card when a loss references the issue, showing gross loss, recovered amount, net loss and recovery state, with a link to the canonical loss record;
- existing indicator or other source context, resolved from the recorded source reference; and
- an explicit unavailable state when a linked source is not accessible or cannot be loaded.

The loss card is contextual only. Recording recoveries, changing loss ownership and loss history remain in the loss record.

### Work

Work reuses `MatterActionsPanel`, including action creation, reassignment, status transition and status-update requests. The current update-request command and durable email-delivery route remain the only follow-up mechanism. An issue-level accountable-owner change remains distinct from an action-performer change.

### Evidence and requests

Evidence and requests combines existing issue evidence work without duplicating it:

- `MatterFormRemediationPanel` remains the controlled route for a response that maps missing facts and verification requirements to a remediation outcome;
- `VendorRelationshipLinks` and `VendorWorkPanel` remain the vendor relationship and vendor-request route;
- existing submitted responses are shown as linked records with form, recipient, status, submission/review state and a direct open action; and
- a form draft may be created with the issue as its recorded target. Normal form approval remains required before distribution.

The existing governed distribution composer already supports verified internal principals and typed subjects. The issue workspace reuses it for an employee request by preselecting `MATTER` and the current issue ID. Vendor requests continue through the existing linked-relationship and vendor-work route, which prevents an unscoped external address from being treated as a vendor recipient.

The two request routes are:

1. **Employee:** an eligible internal principal resolved from current authority and directory scope.
2. **Vendor:** an existing legal-entity-scoped vendor relationship and its approved delivery route.

The form request retains its existing distribution, consent, expiry, audit and notification semantics. No free-text employee email, unscoped vendor email or implicit access grant is introduced. An evidence-remediation binding must not be repurposed for an ordinary consultation or information request.

### Decisions

The existing decision and outcome panels remain unchanged in responsibility and authority. A submitted response or vendor assurance never silently completes an action, changes a decision, or proves the issue outcome.

### Activity

`MatterActivityTimeline` remains the right-side timeline and comment composer. It retains paged activity history, append-only comments, @mentions, and the existing durable email notification for mentioned eligible colleagues. Timeline entries show delivery state when available; a notification failure does not discard the stored comment or workflow event.

At narrow widths, the existing responsive single-column/Activity-tab treatment replaces the persistent rail; it is not a squeezed two-column layout.

## Direct actions

The issue header exposes only actions the current verified actor is authorised to perform:

- **Create form draft** opens the existing form-authoring route with the issue target preselected.
- **Send form** opens the existing governed distribution path and requires either an eligible employee or an existing vendor relationship.
- **Generate board brief** queues a governed report run.

An unavailable action must explain the business condition or authority limitation. Controls must never appear enabled when the underlying command is unavailable.

## Executive board brief

The existing Reports service gains an issue-scoped board-brief report definition and a deterministic PDF output. A request creates a normal report run and its downloadable artefact is listed with other generated reports. The issue record does not generate an untracked browser download.

The brief contains, as of its run timestamp:

- issue reference, title, status and current record version;
- department/group/unit, affected service/process, owner, due date and impact;
- program linkage and current issue summary;
- actions, decisions and outcome state;
- linked loss/recovery figures where authorised;
- linked forms, requests, submitted responses and vendor work;
- recorded integration/source context and freshness/availability; and
- data source and report-generation receipt information.

The report labels unavailable, out-of-scope and unverified information rather than inventing a complete position. It does not expose vendor data, internal comments, protected records or recipient details beyond the requesting actor’s authorised report scope.

## Data, authority and performance

- Existing verified actor, tenant, legal-entity and organisation-scope checks protect every summary, record, link and command.
- Owner labels, organisation paths, linked-loss summaries and source labels are server-resolved bounded projections. Unknown/unavailable values remain explicit.
- Issue lists retain keyset pagination. Linked records are loaded by exact issue ID with indexed queries and bounded limits.
- Form delivery, report creation, assignments, comments and notifications retain transactional event/outbox handling and current authority re-evaluation.
- Historical issue reads remain reconstructable; a current display label must not be presented as historical fact when it is resolved from current authority data.

## Acceptance criteria

1. A list row shows applicable organisation area, owner, due state and impact without opening the issue.
2. The same four facts appear in the issue header; affected service/process remains separately labelled.
3. Linked loss/recovery context appears for every authorised issue with an explicit `matter_id` link, while loss changes remain in the loss record.
4. Existing comments, mentions, update requests, email receipts, action reassignment and activity pagination continue to use their current durable workflows.
5. Creating a form draft and sending a form from an issue use existing governed form/distribution capabilities, with verified employee or vendor recipients only.
6. The board brief is a central, permission-checked report run and uses a deterministic PDF artefact.
7. No visible enabled control lacks an authorised command; unavailable states provide a concise recovery or limitation.
8. Desktop and narrow rendered states preserve a readable primary work area and activity history without duplicate panels or excessive vertical chrome.
9. The customer-copy gate, affected workflow checks and targeted responsive renders pass.

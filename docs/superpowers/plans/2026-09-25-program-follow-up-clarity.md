# Program Follow-up Clarity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Distinguish outdated third-party answers, documents, and internal work in Program review, and deliver reliable follow-up notifications.

**Architecture:** Reuse the versioned vendor-assessment clarification request for a selected subset of fields. Add a presentation-only follow-up classifier in the web client and extend the existing durable staff-notification worker to cover Matter comment mentions.

**Tech Stack:** React, TypeScript, Vitest, Go, PostgreSQL outbox, existing email delivery adapter.

---

### Task 1: Program follow-up summary

**Files:**
- Create: `web/src/components/programFollowUpSummary.ts`
- Test: `web/src/components/programFollowUpSummary.test.ts`
- Modify: `web/src/components/ProgramCurrentPosition.tsx`

- [ ] Write a failing classifier test for vendor answer, vendor document, and internal follow-up records, including unique vendor counts.
- [ ] Run `npm test -- programFollowUpSummary.test.ts` and confirm the new expectations fail.
- [ ] Implement the pure classifier and compact summary component using the existing semantic response/review data.
- [ ] Run the focused test and Program workspace test.

### Task 2: Targeted vendor refresh presentation

**Files:**
- Modify: `web/src/components/VendorDueDiligence.tsx`
- Test: `web/src/components/VendorDueDiligence.test.tsx`
- Modify: `web/src/components/forms/ResponseAssessment.tsx`

- [ ] Write a failing UI test that expects “Request updated fields” with only selected outdated fields.
- [ ] Run `npm test -- VendorDueDiligence.test.tsx` and confirm it fails.
- [ ] Rename and contextualize the existing clarification UI without changing its protected endpoint or payload.
- [ ] Run the focused vendor test and typecheck.

### Task 3: Mention notification delivery

**Files:**
- Modify: `internal/workflow/assignment_notification.go`
- Modify: `internal/evidence/operational_notification_render.go`
- Test: `internal/workflow/assignment_notification_test.go`
- Modify: `web/src/components/MatterActivityTimeline.tsx`

- [ ] Write a failing worker test for an added Matter comment with one mention.
- [ ] Run `go test ./internal/workflow -run Mention` and confirm it fails.
- [ ] Extend recipient resolution, notification rendering, and deduplication to email mentioned staff.
- [ ] Update the comment confirmation to state the stored comment and notified recipients.
- [ ] Run the focused Go and React tests.

### Task 4: Release proof

**Files:**
- Modify: `docs/quality/rendered-ui-evidence.md`

- [ ] Run web typecheck and focused tests.
- [ ] Render Program and targeted vendor-response fixtures at desktop and mobile widths.
- [ ] Record the rendered states and update the evidence index.

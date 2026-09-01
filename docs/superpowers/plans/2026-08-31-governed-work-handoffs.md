# Governed Work Handoffs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Matter handoffs truthful and operable, migrate vendor/assignment/signing controls to the shared UI system, and deliver durable staff assignment email from existing outbox events.

**Architecture:** Preserve Matter, Action, authority, Workflow and outbox ownership. Fix the browser's state-to-operation selection, then add one bounded assignment-notification consumer that resolves active SCIM contact data and records redacted idempotent receipts. UI changes reuse the existing closed component contracts and state fixtures.

**Tech Stack:** Go 1.25, PostgreSQL 18 migrations/pgx, React 19, TypeScript 7, react-aria-components, Vitest/Testing Library, Vite, SMTP STARTTLS, deterministic rendered UI review.

---

## File map

- `web/src/components/matterHandoff.ts`: pure state-to-operation selector and responsibility presentation.
- `web/src/components/MatterCurrentHandoff.tsx`: truthful current handoff rendering.
- `web/src/components/MatterDetailsPanel.tsx`: focused owner reassignment UI.
- `web/src/components/MatterActionsPanel.tsx`: focused Action performer reassignment UI.
- `web/src/components/VendorWorkPanel.tsx`: shared-component request composer.
- `web/src/components/MatterDecisionResponsePanel.tsx`: explicit review/sign/transmit/acknowledge interaction.
- `web/src/components/ui/TextField.tsx`: date/time constraint typing used by shared fields.
- `web/src/components/ui/SelectField.tsx`: stable non-modal option-list positioning within the active workspace landmark.
- `web/src/design-system/components/fields.css`: date indicator and field-state theme treatment.
- `internal/workflow/assignment_notification.go`: event decoder, message context and consumer policy.
- `internal/workflow/assignment_notification_postgres.go`: active SCIM contact/current Matter lookup and redacted receipt persistence.
- `internal/evidence/operational_notification_render.go`: shared email-safe assignment presentation over the existing mail shell.
- `cmd/worker/staff_notification_services.go`: SMTP/contact consumer composition.
- `cmd/worker/services_postgres.go`: publisher registration.
- `migrations/000061_staff_assignment_notifications.*.sql`: receipt ledger.
- `deploy/scripts/seed-demo-foundation.sh`: active demo positions/roles and optional SCIM contact seed.
- `docs/architecture/durable-schema-ownership.md`: durable ownership registration.
- `DESIGN.md`, `web/ui-contract-migrations.json`, `docs/design/ui-component-adoption.md`: component and handoff contract updates.

### Task 1: Prove and fix current-handoff selection

**Files:**
- Create: `web/src/components/matterHandoff.ts`
- Create: `web/src/components/matterHandoff.test.ts`
- Modify: `web/src/components/MatterCurrentHandoff.tsx`
- Test: `web/src/components/MatterRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing selector tests**

Add fixtures proving that `next_action="Confirm scope and owner"` selects `matter.assign` even when a different `matter.transition` operation is actionable, that named Action work selects its exact `matter.action.transition`, and that response signatory work selects the exact response subresource.

- [ ] **Step 2: Run the red tests**

Run: `npm test -- src/components/matterHandoff.test.ts src/components/MatterRecordWorkspace.test.tsx`

Expected: FAIL because the selector does not exist and the CRO fixture still renders **Authorize issue status** as dominant.

- [ ] **Step 3: Implement the pure selector and responsibility label**

Return `{operation, responsibilityLabel}` without changing authority data. Do not use a global `find(can_act)` fallback when multiple operations exist.

- [ ] **Step 4: Render the selected responsibility**

Update `MatterCurrentHandoff` to say **Accountable owner**, **Authorizer**, **Signatory**, etc., and keep unrelated actionable operations in their own panels.

- [ ] **Step 5: Run the green tests**

Run: `npm test -- src/components/matterHandoff.test.ts src/components/MatterRecordWorkspace.test.tsx`

Expected: PASS.

### Task 2: Repair demo principal resolution through seeded identity records

**Files:**
- Modify: `deploy/scripts/seed-demo-foundation.sh`
- Modify: `deploy/tests/deployment_config_test.py`
- Test: `internal/access/postgres_batch_test.go`
- Test: `internal/httpapi/principal_labels_test.go`

- [ ] **Step 1: Add failing seed-contract tests**

Assert that the demo seed creates one active legal-entity position and position-role binding per demo principal, uses stable UUIDs/idempotent conflict checks, and accepts optional `CLEARSIGHT_DEMO_STAFF_EMAIL` without committing an address.

- [ ] **Step 2: Run the red tests**

Run: `python -m unittest deploy.tests.deployment_config_test`

Expected: FAIL because the seed currently inserts only principals and routing policies.

- [ ] **Step 3: Add the operational identity fixtures**

Seed role templates, positions and bindings used by the normal resolver. When `CLEARSIGHT_DEMO_STAFF_EMAIL` is set, pass it to psql as a protected variable and create/update one active demo SCIM source/user without printing the value.

- [ ] **Step 4: Prove resolver behavior**

Extend PostgreSQL integration coverage so every seeded stored responsibility resolves a display name and a missing position still yields a partial-label result.

- [ ] **Step 5: Run the green tests**

Run: `python -m unittest deploy.tests.deployment_config_test`

Run: `go test -tags postgres ./internal/access ./internal/httpapi -run 'Principal|Label|Demo'`

Expected: PASS.

### Task 3: Define the durable assignment-notification receipt

**Files:**
- Create: `migrations/000061_staff_assignment_notifications.up.sql`
- Create: `migrations/000061_staff_assignment_notifications.down.sql`
- Modify: `docs/architecture/durable-schema-ownership.md`
- Test: `internal/integration/schema_test.go`

- [ ] **Step 1: Write the failing schema ownership test**

Require `staff_assignment_notification_deliveries` to be registered as an active infrastructure ledger with workflow ownership, redacted recipient fingerprint, status, provider message ID, attempted/delivered timestamps and unique `(outbox_event_id, principal_id, notification_kind)`.

- [ ] **Step 2: Run the red schema test**

Run: `go test -tags 'postgres postgresintegration' ./internal/integration -run 'SchemaOwnership|Migration'`

Expected: FAIL because migration 000061 and its ownership entry do not exist.

- [ ] **Step 3: Add additive up/down migrations**

Use tenant/principal/outbox foreign keys, bounded status checks, no address/body/link columns, and indexes for event lookup and recent delivery diagnostics.

- [ ] **Step 4: Register ownership and rerun**

Run: `go test -tags 'postgres postgresintegration' ./internal/integration -run 'SchemaOwnership|Migration'`

Expected: PASS, including latest rollback/reapply.

### Task 4: Add test-first staff assignment delivery

**Files:**
- Create: `internal/workflow/assignment_notification.go`
- Create: `internal/workflow/assignment_notification_test.go`
- Create: `internal/workflow/assignment_notification_postgres.go`
- Create: `internal/workflow/assignment_notification_postgres_integration_test.go`
- Create: `internal/evidence/operational_notification_render.go`
- Create: `internal/evidence/operational_notification_render_test.go`

- [ ] **Step 1: Write failing event-policy tests**

Cover `MATTER_OWNER_CHANGED`, `ACTION_ASSIGNED`, unrelated events, malformed payload, same-owner/no-op protection, valid HTTPS deep link generation, missing contact, temporary SMTP failure, terminal rejection and replay after a final receipt.

- [ ] **Step 2: Run the red unit tests**

Run: `go test ./internal/workflow ./internal/evidence -run 'AssignmentNotification|OperationalNotification'`

Expected: FAIL because the consumer and renderer do not exist.

- [ ] **Step 3: Implement the minimal consumer and renderer**

Decode only existing safe event fields; load current bounded Matter context; resolve an active SCIM email; render multipart-safe subject/plain/HTML through the shared presentation; call the existing SMTP adapter; persist only the redacted receipt.

- [ ] **Step 4: Add PostgreSQL contact and idempotency tests**

Prove inactive source/user, wrong tenant/entity, non-email username, missing Matter, duplicate event and already-final receipt handling.

- [ ] **Step 5: Run the green tests**

Run: `go test ./internal/workflow ./internal/evidence -run 'AssignmentNotification|OperationalNotification'`

Run: `go test -tags 'postgres postgresintegration' ./internal/workflow -run AssignmentNotification`

Expected: PASS without protected values in errors or formatted receipts.

### Task 5: Compose the notification worker with protected SMTP settings

**Files:**
- Create: `cmd/worker/staff_notification_services.go`
- Create: `cmd/worker/staff_notification_services_test.go`
- Modify: `cmd/worker/services_postgres.go`
- Modify: `internal/platform/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: Write failing composition tests**

Require staff delivery to be disabled when the application base URL or SMTP is absent in non-production, fail closed when explicitly enabled in production but incomplete, and reuse `CLEARSIGHT_SMTP_SECRET_REF` without loading or printing plaintext in configuration objects.

- [ ] **Step 2: Run the red tests**

Run: `go test ./cmd/worker ./internal/platform/config -run 'StaffNotification|SMTP'`

Expected: FAIL because no staff consumer is composed.

- [ ] **Step 3: Build and register the consumer**

Construct one SMTP adapter, Postgres notification repository and assignment consumer; append it to the existing composite publisher after the Matter projectors. Document only variable names in `.env.example`.

- [ ] **Step 4: Run the green tests**

Run: `go test ./cmd/worker ./internal/platform/config -run 'StaffNotification|SMTP'`

Run: `go test -tags postgres ./cmd/worker`

Expected: PASS.

### Task 6: Migrate Matter reassignment controls

**Files:**
- Modify: `web/src/components/MatterDetailsPanel.tsx`
- Modify: `web/src/components/MatterActionsPanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`
- Modify: `web/ui-contract-migrations.json`

- [ ] **Step 1: Write failing interaction tests**

Assert shared `FocusedSheet` dialogs, labelled eligible-owner selection, required reason, current-owner context, loading/disabled feedback, conflict recovery and notification consequence for Matter and Action reassignment.

- [ ] **Step 2: Run the red tests**

Run: `npm test -- src/components/MatterRecordWorkspace.test.tsx`

Expected: FAIL because assignment is inline and uses raw controls.

- [ ] **Step 3: Implement with shared components**

Use `FocusedSheet`, `SelectField`, `TextArea`, `Button` and `Notice`. Keep `assignMatter`/`assignMatterAction`, expected version and server authority unchanged.

- [ ] **Step 4: Add the migrated files to the contract manifest**

Run: `npm run check:ui-contracts`

Expected: PASS with no prohibited raw interactive control in the declared migrated files.

- [ ] **Step 5: Run the green tests**

Run: `npm test -- src/components/MatterRecordWorkspace.test.tsx`

Expected: PASS.

### Task 7: Migrate the vendor request composer

**Files:**
- Modify: `web/src/components/VendorWorkPanel.tsx`
- Modify: `web/src/components/VendorWorkPanel.test.tsx`
- Modify: `web/src/vendor-work.css`
- Modify: `web/ui-contract-migrations.json`
- Modify: `docs/design/ui-component-adoption.md`

- [ ] **Step 1: Write failing sheet and validation tests**

Cover wide desktop sheet, full-height mobile semantics, dark-mode listbox, labelled native date input, field-level email/date errors, one primary submit action, disabled reason and partial-delivery recovery.

- [ ] **Step 2: Run the red tests**

Run: `npm test -- src/components/VendorWorkPanel.test.tsx`

Expected: FAIL because the composer is inline and uses raw controls.

- [ ] **Step 3: Implement the shared-component composer**

Use `FocusedSheet`, `SelectField`, `TextField`, `TextArea`, `Button` and `Notice`; preserve the existing prepare-then-send behavior and state.

- [ ] **Step 4: Remove feature-level component restyling**

Keep layout only in `vendor-work.css`; move no fill/border/focus/foreground rules outside the shared component layer.

- [ ] **Step 5: Run component and contract tests**

Run: `npm test -- src/components/VendorWorkPanel.test.tsx`

Run: `npm run check:ui-contracts`

Expected: PASS.

### Task 8: Make authorization and signing actions role-specific

**Files:**
- Create: `web/src/components/matterResponsePresentation.ts`
- Create: `web/src/components/matterResponsePresentation.test.ts`
- Modify: `web/src/components/MatterDecisionResponsePanel.tsx`
- Modify: `web/src/components/MatterOutcomePanel.tsx`
- Modify: `web/src/components/MatterRecordWorkspace.test.tsx`

- [ ] **Step 1: Write failing presentation tests**

For Reviewer, Signatory, Transmitter, Acknowledgement recorder and Authorizer operations, assert distinct action labels, consequence copy and permitted targets. Assert that sign-off does not claim transmission and transmission does not claim acknowledgement.

- [ ] **Step 2: Run the red tests**

Run: `npm test -- src/components/matterResponsePresentation.test.ts src/components/MatterRecordWorkspace.test.tsx`

Expected: FAIL because response actions are all labelled **Update response status**.

- [ ] **Step 3: Implement role-specific focused forms**

Keep `matter.response.transition`, `matter.decision.record` and `matter.transition` unchanged. Present current state, selected target, responsibility and required rationale using shared components.

- [ ] **Step 4: Run the green tests**

Run: `npm test -- src/components/matterResponsePresentation.test.ts src/components/MatterRecordWorkspace.test.tsx`

Expected: PASS.

### Task 9: Stabilize shared selects, finish date theming and document the contracts

**Files:**
- Modify: `web/src/components/ui/SelectField.tsx`
- Modify: `web/src/components/ui/SelectField.test.tsx`
- Modify: `web/src/components/ui/TextField.tsx`
- Modify: `web/src/components/ui/Field.test.tsx`
- Modify: `web/src/design-system/components/fields.css`
- Modify: `DESIGN.md`
- Modify: `docs/design/ui-component-adoption.md`

- [ ] **Step 1: Write failing shared-field tests**

Prove an open select does not lock or resize the document scroll container, its listbox remains within the active workspace landmark, and date min/max string forwarding, required/error anatomy and shared class ownership remain correct.

- [ ] **Step 2: Run the red test**

Run: `npm test -- src/components/ui/SelectField.test.tsx src/components/ui/Field.test.tsx`

Expected: FAIL on document-root scroll locking and date constraint typing/presentation assertions.

- [ ] **Step 3: Implement stable select overlay and token-driven date treatment**

Open shared option lists as non-modal, keyboard-dismissible popovers inside the closest dialog or main landmark. Extend the date prop types and theme the calendar indicator, padding, color-scheme, focus and disabled states through component tokens in both themes.

- [ ] **Step 4: Update contracts and rerun**

Run: `npm test -- src/components/ui/SelectField.test.tsx src/components/ui/Field.test.tsx`

Run: `npm run check:ui-contracts`

Expected: PASS.

### Task 10: Render, verify, merge and deploy

**Files:**
- Modify: `web/src/uiReviewFixtures.tsx`
- Modify: `docs/quality/rendered-ui-evidence.md`
- Modify: `docs/implementation-plan.md`

- [ ] **Step 1: Add deterministic workflow fixtures**

Add CRO handoff, owner reassignment, vendor request, contact-unavailable delivery, signatory, transmitter and acknowledgement states.

- [ ] **Step 2: Run focused and full frontend gates**

Run: `npm test`

Run: `npm run typecheck`

Run: `npm run check:ui-contracts`

Run: `npm run review:ui`

Run: `npm run build`

Expected: all PASS, no axe violations, no 320px horizontal scrolling and correct light/dark contrast.

- [ ] **Step 3: Run backend and migration gates**

Run: `gofmt -w <changed-go-files>` then `git diff --check`

Run: `go test ./...`

Run: `go test -race ./...`

Run: `go test -tags postgres ./...`

Run: `go vet ./...`

Run: the repository's latest migration rollback/reapply and serialized PostgreSQL integration gate.

Expected: all PASS on the exact final head.

- [ ] **Step 4: Inspect rendered evidence and fix the highest-impact defect**

Inspect desktop/light, desktop/dark, 390px and 320px/200% fixtures. Record the defect and rerender its state after correction.

- [ ] **Step 5: Commit, review and integrate**

Commit in bounded increments, push `codex/governed-workflow-audit`, open the PR, review exact-head diff/checks, merge only when green, deploy `main`, and verify readiness reports the exact merge SHA.

- [ ] **Step 6: Run controlled staff-notification acceptance**

With the protected demo staff email already configured on the host, reassign one non-production issue/action, verify a redacted final delivery receipt, confirm inbox receipt manually, open the authenticated deep link and confirm the new assignee sees the exact work. Do not print the recipient, SMTP secret or message link.

## Self-review

- Spec coverage: handoff truth, seeded identity, Matter/Action reassignment, durable email, vendor composer, authorization/signing, date theming, rendered/deployment evidence are each mapped to a task.
- Placeholder scan: no TBD/TODO or unspecified test step remains.
- Type consistency: events use existing `MATTER_OWNER_CHANGED`/`ACTION_ASSIGNED`; UI uses current `MatterOperation`; worker uses existing outbox and SMTP interfaces; no client actor field is introduced.

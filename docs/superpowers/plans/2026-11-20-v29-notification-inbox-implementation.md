# 2.9 Inbox recipient state implementation plan

**Status:** Proposed implementation contract; not implemented or tested. This covers recipient read/archive/snooze behavior only, not the 2.9 local discussion, mention delivery or approval-workflow system. Those require separate source, authorization, retention and notification contracts.

**Goal:** Supply the 2.9 Inbox's personal views while retaining the existing source-backed Work and recipient-isolated notification owner.

**Architecture:** Extend `internal/workflow`'s existing per-recipient `in_app_notifications` row; source workflow remains authoritative for whether action is needed. The notification list applies verified recipient scope and current source visibility before pagination; the client cannot close source work by changing its Inbox receipt.

**Tech stack:** Go, pgx/PostgreSQL, existing `internal/httpapi` route registry, React/TypeScript, Vitest and browser evidence.

## Current contract

`GET /api/v1/notifications?cursor=&limit=&unread_only=` requires a verified identity. `limit` is 1–100, default 25. It returns `workflow.NotificationPage` (`items`, optional `next_cursor`, `unread_count`, `as_of`). Each `InAppNotification` currently returns `id`, `kind`, `title`, `summary`, `subject_type`, `subject_id`, `action_path`, `occurred_at`, optional `read_at`; tenant/entity/principal are not serialized. `POST /api/v1/notifications/{id}/read` marks one recipient row read. See `internal/httpapi/notification_handlers.go`, `internal/workflow/notification_model.go`, `notification_service.go`, `notification_postgres.go` and migration `000098_in_app_notifications.up.sql`.

The current Postgres list query scopes rows by tenant, legal entity and principal, then filters/orders/paginates. The inspected query selects stored source title/summary without a source-visibility join; a new feature must not assume current recipient ownership alone is sufficient to disclose a restricted source after access is revoked. Establish the existing source-access resolver and prove authorization/redaction before extending the response.

## Proposed exact API

1. Extend `GET /api/v1/notifications` with `view=INBOX|UNREAD|SNOOZED|ARCHIVED` (default `INBOX`), optional `kind`, optional `search` (UTF-8, max 200 characters), existing `cursor` and `limit`. `unread_only=true` remains accepted for old clients only when `view` is absent; combining it with `view` returns `400 notification_filter_invalid`. `INBOX` excludes archived and not-yet-due snoozed rows; `UNREAD` additionally requires `read_at IS NULL`; `SNOOZED` requires `snoozed_until > as_of` and no archive; `ARCHIVED` requires `archived_at IS NOT NULL`. The opaque cursor must bind query filters, authorized scope and the fixed `as_of` used to determine snooze visibility. A changed filter or invalid cursor returns `400 notification_cursor_invalid`.
2. Keep `POST /api/v1/notifications/{id}/read` unchanged. Add `POST /api/v1/notifications/{id}/receipt` with body `{ "expected_version": 1, "action": "MARK_UNREAD"|"ARCHIVE"|"RESTORE"|"SNOOZE"|"UNSNOOZE", "snoozed_until": "2026-11-21T09:00:00Z" }`. `snoozed_until` is required only for `SNOOZE`, must be a valid future UTC timestamp and no more than 90 days after server time. Any supplied value for other actions is invalid. `MARK_UNREAD` sets `read_at=NULL`; `ARCHIVE` sets `archived_at` and clears snooze; `RESTORE` clears archive; `SNOOZE` requires unarchived row and sets its time; `UNSNOOZE` clears it. Repeating an operation on an already equivalent state returns the current row without changing version; a conflicting changed version returns `409 notification_receipt_conflict`.
3. Success for either mutation is `200` with the existing notification fields plus `version` (positive integer), optional `archived_at` and optional `snoozed_until`. Add these three fields additively to list items. The new handler derives `tenant_id`, `legal_entity_id`, `principal_id` and time from the verified context/server; body actor or scope fields are rejected by strict decoding. Unknown/inaccessible notification IDs return the existing `404 notification_not_found`. Malformed actions return `422 notification_receipt_invalid`; unavailable storage returns the existing 503 family.
4. Add `counts` to the list response as `{ "inbox_unread": integer, "as_of": timestamp, "population": "visible_to_actor" }`. Retain `unread_count` for old clients until they are migrated; redefine neither its existing meaning nor its endpoint silently. The new count is calculated after the same source authorization and view predicates, across the full bounded query population rather than loaded rows. Mark unavailable rather than return zero if source visibility cannot be calculated.

These are proposed new paths and semantics, not existing endpoints. Source-resolution and policy code must be specified before implementation: if a source is no longer visible, either filter the whole row before count/LIMIT or return an authorized generic receipt without title, summary, source ID or action path. Select one uniform treatment in the executable contract. Do not paginate first and then hide rows, and do not expose stale sensitive titles in a count or search result.

## Schema migration

Allocate the next unique migration number after checking the ledger. The following additive PostgreSQL statements are the intended up migration, inside `BEGIN`/`COMMIT`:

```sql
ALTER TABLE in_app_notifications
  ADD COLUMN receipt_version bigint NOT NULL DEFAULT 1 CHECK (receipt_version > 0),
  ADD COLUMN archived_at timestamptz,
  ADD COLUMN snoozed_until timestamptz,
  ADD CONSTRAINT in_app_notifications_archive_clears_snooze
    CHECK (archived_at IS NULL OR snoozed_until IS NULL);

CREATE INDEX in_app_notifications_actor_active_idx
  ON in_app_notifications
    (tenant_id, legal_entity_id, principal_id, occurred_at DESC, id DESC)
  WHERE archived_at IS NULL;

CREATE INDEX in_app_notifications_actor_archive_idx
  ON in_app_notifications
    (tenant_id, legal_entity_id, principal_id, occurred_at DESC, id DESC)
  WHERE archived_at IS NOT NULL;
```

The database does not infer that snooze expiration means a source item was resolved; queries compare to a fixed `as_of` and no background job is necessary merely to reappear. The 90-day bound is a service rule because it depends on server time. Existing rows retain `receipt_version=1`, `archived_at=NULL`, `snoozed_until=NULL`, preserving current read state. The existing unique delivery identity and unread index remain intact until verified consumers have migrated.

An automatic down migration after receipt state is recorded would erase user state. The down migration must refuse if any row has `receipt_version > 1`, non-null `archived_at` or non-null `snoozed_until`; otherwise it may drop the two new indexes, constraint and columns. Before downgrade, preserve a backup and verify no deployment is still writing the new fields. Production rollback after receipt updates should use a compatibility reader and roll forward, not delete the state.

## Transaction and privacy behavior

Scope the mutation `UPDATE` by verified tenant, entity, principal, notification ID and `receipt_version`, then distinguish not found from version conflict without disclosing another recipient's row. Lock/read the selected row in a transaction for state transitions. A recipient receipt is presentation state, never an approval, source decision, action completion, message delivery or source-resolution event. Expiration of snooze is derived and does not alter source state. A revoked source must not expose cached title, summary, path or details.

The current notification row is a recipient projection of an outbox event. Keep its existing deduplication key; do not introduce a second Inbox table or duplicate source workflow. Privacy-safe source authorization must be compatible with filtering before LIMIT and exact count; if a generic source-access adapter cannot do this within bounded queries, implement per-source SQL adapters and admit only supported source kinds. Do not fetch an unbounded population and filter in Go.

## Files and ordered tasks

- [ ] Extend `internal/workflow/notification_model.go` with receipt fields, view/filter and fixed-as-of cursor; add tests in `internal/workflow/notification_service_test.go` or a new focused `notification_receipt_test.go` for every action, invalid action/date, version conflict, repeated action and cursor/filter mismatch. Confirm tests fail before implementation.
- [ ] Add the paired migration and schema tests. Update `internal/workflow/notification_postgres.go` and the memory implementation for identical state semantics. Bind source authorization before count and page; add dedicated SQL/access tests for protected source revocation. Check existing historical notification sources before enforcing a new source-kind allowlist.
- [ ] Extend `internal/httpapi/notification_handlers.go` and the owning notification route registry definition with the new route and request DTO; add route-class and `api/runtime.openapi.json` parity tests. Source identity and receipt owner come exclusively from `identity.Require`. Add unauthorized, cross-tenant, other-recipient, revoked-source, malformed-time and stale-version HTTP tests.
- [ ] Extend `web/src/notificationApi.ts`, its inbox hook and `components/notifications/NotificationsWorkspace.tsx` to call real server actions. Preserve `read_at` semantics and use returned versions; show conflict/retry without resetting source work. 2.9 visual anatomy includes Inbox, selected notice and personal views, but unsupported discussion/mentions must not appear as enabled controls.
- [ ] Run from root `go test ./internal/workflow ./internal/httpapi`; with an isolated migrated `TEST_DATABASE_URL`, run `go test -tags "postgres postgresintegration" ./internal/workflow ./internal/httpapi` and inspect skips. From `web`, run `npm test -- --run src/components/notifications/NotificationsWorkspace.test.tsx src/components/notifications/NotificationActivity.test.tsx src/copyQuality.test.ts`, `npm run typecheck`, `npm run check:runtime-truth` and `npm run check:ui-contracts`. Record exact exit status and output.
- [ ] Add browser scenarios for read/unread, archive/restore, snooze expiry, cursor paging, reload, old-client behavior, source access revoked between list and open, 390/768/1024/1280/1440 widths, keyboard/focus and reduced motion. Capture same-screen reference/runtime pairs; reconcile counts with directly authorized source rows.

## Unresolved prerequisite

The current notification projector's exact source-kind inventory, source permission joins, retention policy and correct recipient-state route class still need inspection before code or deployment. The API and DDL above are specific proposed decisions, but the slice is **not yet fully code-ready** until the source-authorization implementation and SQL query plan are specified. No tests or migrations have been run.

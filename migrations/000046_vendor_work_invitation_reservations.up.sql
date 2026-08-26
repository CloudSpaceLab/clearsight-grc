BEGIN;

CREATE TABLE third_party_work_invitation_reservations (
    invitation_id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    work_request_id uuid NOT NULL,
    request_id uuid NOT NULL,
    capture_sequence integer NOT NULL CHECK (capture_sequence > 0),
    state text NOT NULL CHECK (state IN ('PENDING','FINALIZED','SUPERSEDED')),
    created_at timestamptz NOT NULL,
    resolved_at timestamptz,
    UNIQUE (invitation_id,tenant_id,request_id),
    FOREIGN KEY (work_request_id,tenant_id,legal_entity_id)
        REFERENCES third_party_work_requests(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (request_id,tenant_id)
        REFERENCES capture_requests(id,tenant_id),
    CHECK ((state='PENDING' AND resolved_at IS NULL) OR
           (state IN ('FINALIZED','SUPERSEDED') AND resolved_at IS NOT NULL))
);

-- The reserved invitation identifier is recorded before the Evidence invitation
-- exists, so this table intentionally has no pre-issuance foreign key to
-- capture_invitations. Finalization verifies that relationship through the
-- existing work and capture-link foreign keys in one transaction.
CREATE UNIQUE INDEX third_party_work_invitation_reservations_pending_idx
    ON third_party_work_invitation_reservations(tenant_id,legal_entity_id,work_request_id)
    WHERE state='PENDING';
CREATE INDEX third_party_work_invitation_reservations_request_idx
    ON third_party_work_invitation_reservations(tenant_id,request_id,created_at DESC);

ALTER TABLE third_party_work_events DROP CONSTRAINT third_party_work_events_event_type_check;
ALTER TABLE third_party_work_events ADD CONSTRAINT third_party_work_events_event_type_check CHECK (event_type IN (
    'VendorWorkPrepared','VendorWorkCaptureAttached','VendorWorkInvitationReserved','VendorWorkInvitationReady',
    'VendorWorkSent','VendorWorkResponseReceived','VendorWorkReviewStarted','VendorWorkChangesRequested',
    'VendorWorkAccepted','VendorWorkCancelled','VendorWorkDeliveryRetryRequired','VendorWorkPreparationRetryRequired'
));

COMMIT;

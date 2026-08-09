BEGIN;

ALTER TABLE capture_requests
    ADD COLUMN recipient_state text,
    ADD COLUMN recipient_revision bigint NOT NULL DEFAULT 1,
    ADD COLUMN recipient_issue_reason text NOT NULL DEFAULT '';

UPDATE capture_requests
SET recipient_state = CASE
    WHEN recipient_type IS NULL THEN 'LEGACY_UNASSIGNED'
    ELSE 'ASSIGNED'
END,
recipient_revision = CASE
    WHEN recipient_type IS NULL THEN 0
    ELSE 1
END;

ALTER TABLE capture_requests
    ALTER COLUMN recipient_state SET NOT NULL,
    ALTER COLUMN recipient_state SET DEFAULT 'ASSIGNED',
    ADD CONSTRAINT capture_requests_recipient_state_check CHECK (
        (recipient_type IS NULL AND recipient_state='LEGACY_UNASSIGNED' AND recipient_revision=0 AND recipient_issue_reason='') OR
        (recipient_type IS NOT NULL AND recipient_state IN ('ASSIGNED','REASSIGNMENT_REQUIRED') AND recipient_revision >= 1)
    ),
    ADD CONSTRAINT capture_requests_id_tenant_unique UNIQUE(id,tenant_id);

ALTER TABLE capture_invitations
    ADD COLUMN recipient_revision bigint NOT NULL DEFAULT 1 CHECK (recipient_revision >= 1);

ALTER TABLE capture_sessions
    ADD COLUMN recipient_revision bigint NOT NULL DEFAULT 1 CHECK (recipient_revision >= 1);

CREATE TABLE capture_recipient_history (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    request_id uuid NOT NULL,
    event_type text NOT NULL CHECK (event_type IN ('WRONG_RECIPIENT','REASSIGNED')),
    from_recipient_type text,
    from_principal_id uuid,
    from_audience_hash bytea,
    from_audience_hint text NOT NULL DEFAULT '',
    to_recipient_type text,
    to_principal_id uuid,
    to_audience_hash bytea,
    to_audience_hint text NOT NULL DEFAULT '',
    actor_principal_id uuid NOT NULL,
    reason text NOT NULL,
    recipient_revision bigint NOT NULL CHECK (recipient_revision >= 1),
    request_version bigint NOT NULL CHECK (request_version >= 1),
    occurred_at timestamptz NOT NULL,
    CONSTRAINT capture_recipient_history_request_fk
        FOREIGN KEY(request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    CONSTRAINT capture_recipient_history_actor_fk
        FOREIGN KEY(actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT capture_recipient_history_from_shape CHECK (
        (from_recipient_type='INTERNAL_PRINCIPAL' AND from_principal_id IS NOT NULL AND from_audience_hash IS NULL AND from_audience_hint='') OR
        (from_recipient_type='EXTERNAL_AUDIENCE' AND from_principal_id IS NULL AND octet_length(from_audience_hash)=32 AND from_audience_hint<>'')
    ),
    CONSTRAINT capture_recipient_history_to_shape CHECK (
        (event_type='WRONG_RECIPIENT' AND to_recipient_type IS NULL AND to_principal_id IS NULL AND to_audience_hash IS NULL AND to_audience_hint='') OR
        (event_type='REASSIGNED' AND (
            (to_recipient_type='INTERNAL_PRINCIPAL' AND to_principal_id IS NOT NULL AND to_audience_hash IS NULL AND to_audience_hint='') OR
            (to_recipient_type='EXTERNAL_AUDIENCE' AND to_principal_id IS NULL AND octet_length(to_audience_hash)=32 AND to_audience_hint<>'')
        ))
    )
);

CREATE INDEX capture_recipient_history_request_idx
    ON capture_recipient_history(tenant_id,request_id,occurred_at DESC,id DESC);

COMMIT;

BEGIN;

ALTER TABLE capture_requests
    ADD COLUMN recipient_type text,
    ADD COLUMN recipient_principal_id uuid,
    ADD COLUMN recipient_audience_hash bytea,
    ADD COLUMN recipient_hint text NOT NULL DEFAULT '';

ALTER TABLE capture_requests
    ADD CONSTRAINT capture_requests_recipient_tenant_fk
        FOREIGN KEY (recipient_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT capture_requests_recipient_shape_check CHECK (
        recipient_type IS NULL
        OR (
            recipient_type='INTERNAL_PRINCIPAL'
            AND recipient_principal_id IS NOT NULL
            AND recipient_audience_hash IS NULL
            AND recipient_hint=''
        )
        OR (
            recipient_type='EXTERNAL_AUDIENCE'
            AND recipient_principal_id IS NULL
            AND recipient_audience_hash IS NOT NULL
            AND octet_length(recipient_audience_hash)=32
            AND recipient_hint<>''
        )
    );

-- Existing rows are intentionally left recipient_type=NULL. There is no safe
-- historical backfill from created_by, why_you or invitation copy. They remain
-- readable through administrative/current-record paths but are not recipient
-- assigned actor work until explicitly replaced/recreated under this contract.
CREATE INDEX capture_requests_internal_recipient_queue_idx
    ON capture_requests(tenant_id,recipient_principal_id,status,deadline,id)
    WHERE recipient_type='INTERNAL_PRINCIPAL' AND status IN ('READY','IN_PROGRESS');

COMMIT;

BEGIN;

CREATE TABLE notification_email_deliveries (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    episode_id uuid,
    notice_sequence integer CHECK (notice_sequence > 0),
    digest_date date,
    source_event_id uuid,
    principal_id uuid NOT NULL,
    delivery_class text NOT NULL CHECK (delivery_class IN ('ATTENTION_CRITICAL','DAILY_DIGEST')),
    recipient_fingerprint bytea,
    status text NOT NULL CHECK (status IN (
        'DELIVERY_STARTED','DELIVERY_OUTCOME_UNKNOWN','DELIVERED','CONTACT_UNAVAILABLE',
        'NOTICE_SUPERSEDED','RECIPIENT_REJECTED','PERMANENT_FAILURE','TEMPORARY_FAILURE'
    )),
    failure_code text NOT NULL DEFAULT '' CHECK (failure_code=btrim(failure_code) AND char_length(failure_code)<=128),
    provider_message_id text NOT NULL DEFAULT '' CHECK (provider_message_id=btrim(provider_message_id) AND char_length(provider_message_id)<=512),
    attempt_count integer NOT NULL DEFAULT 1 CHECK (attempt_count > 0),
    first_attempted_at timestamptz NOT NULL,
    last_attempted_at timestamptz NOT NULL,
    delivered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (
        (delivery_class='ATTENTION_CRITICAL'
         AND episode_id IS NOT NULL
         AND notice_sequence IS NOT NULL
         AND digest_date IS NULL
         AND source_event_id IS NOT NULL)
        OR
        (delivery_class='DAILY_DIGEST'
         AND episode_id IS NULL
         AND notice_sequence IS NULL
         AND digest_date IS NOT NULL
         AND source_event_id IS NULL)
    )
);

CREATE UNIQUE INDEX notification_email_deliveries_attention_uq
    ON notification_email_deliveries(tenant_id,episode_id,notice_sequence,principal_id)
    WHERE delivery_class='ATTENTION_CRITICAL';

CREATE UNIQUE INDEX notification_email_deliveries_digest_uq
    ON notification_email_deliveries(tenant_id,digest_date,principal_id)
    WHERE delivery_class='DAILY_DIGEST';

CREATE INDEX notification_email_deliveries_principal_history_idx
    ON notification_email_deliveries(tenant_id,principal_id,last_attempted_at DESC,id DESC);

CREATE INDEX notification_email_deliveries_failure_idx
    ON notification_email_deliveries(tenant_id,last_attempted_at DESC,id DESC)
    WHERE status IN ('DELIVERY_OUTCOME_UNKNOWN','CONTACT_UNAVAILABLE','RECIPIENT_REJECTED','PERMANENT_FAILURE','TEMPORARY_FAILURE');

COMMIT;

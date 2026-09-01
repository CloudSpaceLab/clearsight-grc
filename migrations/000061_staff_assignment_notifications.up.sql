BEGIN;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_id_tenant_key UNIQUE (id, tenant_id);

CREATE TABLE staff_assignment_notification_deliveries (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    outbox_event_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    notification_kind text NOT NULL CHECK (notification_kind IN ('MATTER_OWNER_ASSIGNED','ACTION_PERFORMER_ASSIGNED')),
    recipient_fingerprint bytea,
    status text NOT NULL CHECK (status IN ('DELIVERY_STARTED','DELIVERY_OUTCOME_UNKNOWN','DELIVERED','CONTACT_UNAVAILABLE','ASSIGNMENT_SUPERSEDED','RECIPIENT_REJECTED','PERMANENT_FAILURE','TEMPORARY_FAILURE')),
    failure_code text NOT NULL DEFAULT '' CHECK (failure_code = btrim(failure_code) AND char_length(failure_code) <= 128),
    provider_message_id text NOT NULL DEFAULT '' CHECK (provider_message_id = btrim(provider_message_id) AND char_length(provider_message_id) <= 512),
    attempt_count integer NOT NULL DEFAULT 1 CHECK (attempt_count > 0),
    last_attempted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    delivered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id, outbox_event_id, principal_id, notification_kind),
    FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    FOREIGN KEY (outbox_event_id, tenant_id) REFERENCES outbox_events(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (principal_id, tenant_id) REFERENCES principals(id, tenant_id),
    CHECK (recipient_fingerprint IS NULL OR octet_length(recipient_fingerprint) = 32),
    CHECK ((status = 'CONTACT_UNAVAILABLE' AND recipient_fingerprint IS NULL) OR status <> 'CONTACT_UNAVAILABLE'),
    CHECK ((status = 'DELIVERED' AND delivered_at IS NOT NULL) OR (status <> 'DELIVERED' AND delivered_at IS NULL)),
    CHECK (updated_at >= created_at)
);

CREATE INDEX staff_assignment_notification_event_idx
    ON staff_assignment_notification_deliveries(tenant_id, outbox_event_id);
CREATE INDEX staff_assignment_notification_recent_idx
    ON staff_assignment_notification_deliveries(tenant_id, last_attempted_at DESC, id DESC);

COMMIT;

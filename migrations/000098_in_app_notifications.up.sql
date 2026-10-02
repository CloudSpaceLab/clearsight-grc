BEGIN;

CREATE TABLE in_app_notifications (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    outbox_event_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    notification_kind text NOT NULL
        CHECK (notification_kind = btrim(notification_kind) AND notification_kind <> '' AND char_length(notification_kind) <= 64),
    subject_type text NOT NULL
        CHECK (subject_type = btrim(subject_type) AND subject_type <> '' AND char_length(subject_type) <= 64),
    subject_id uuid NOT NULL,
    title text NOT NULL
        CHECK (title = btrim(title) AND title <> '' AND char_length(title) <= 240),
    summary text NOT NULL DEFAULT ''
        CHECK (summary = btrim(summary) AND char_length(summary) <= 500),
    action_path text NOT NULL
        CHECK (action_path = btrim(action_path) AND action_path LIKE '#%' AND char_length(action_path) <= 512),
    occurred_at timestamptz NOT NULL,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id, outbox_event_id, principal_id, notification_kind),
    FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    FOREIGN KEY (principal_id, tenant_id) REFERENCES principals(id, tenant_id),
    CHECK (read_at IS NULL OR read_at >= occurred_at)
);

CREATE INDEX in_app_notifications_actor_recent_idx
    ON in_app_notifications(tenant_id, legal_entity_id, principal_id, occurred_at DESC, id DESC);

CREATE INDEX in_app_notifications_actor_unread_idx
    ON in_app_notifications(tenant_id, legal_entity_id, principal_id, occurred_at DESC, id DESC)
    WHERE read_at IS NULL;

COMMIT;

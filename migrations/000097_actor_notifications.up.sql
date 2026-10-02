BEGIN;

CREATE TABLE actor_notifications (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    outbox_event_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    notification_kind text NOT NULL
        CHECK (notification_kind IN ('MATTER_OWNER_ASSIGNED','ACTION_PERFORMER_ASSIGNED','ACTION_UPDATE_REQUESTED','MATTER_COMMENT_MENTIONED')),
    matter_id uuid NOT NULL,
    action_id uuid,
    occurred_at timestamptz NOT NULL,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id,outbox_event_id,principal_id,notification_kind),
    FOREIGN KEY (tenant_id,legal_entity_id) REFERENCES legal_entities(tenant_id,id),
    FOREIGN KEY (outbox_event_id,tenant_id) REFERENCES outbox_events(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CHECK (read_at IS NULL OR read_at>=occurred_at)
);

CREATE INDEX actor_notifications_actor_recent_idx
    ON actor_notifications(tenant_id,legal_entity_id,principal_id,occurred_at DESC,id DESC);

CREATE INDEX actor_notifications_actor_unread_idx
    ON actor_notifications(tenant_id,legal_entity_id,principal_id,occurred_at DESC,id DESC)
    WHERE read_at IS NULL;

COMMIT;

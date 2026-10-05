BEGIN;

CREATE TABLE attention_episodes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    condition_key text NOT NULL
        CHECK (condition_key IN ('risks_outside_appetite','indicator_breaches','assurance_failures','losses_without_issue')),
    member_id uuid NOT NULL,
    subject_type text NOT NULL CHECK (subject_type IN ('RISK','LOSS')),
    subject_id uuid NOT NULL,
    state text NOT NULL CHECK (state IN ('OPEN','CLEARED')),
    last_condition_state text NOT NULL CHECK (last_condition_state=btrim(last_condition_state) AND last_condition_state<>''),
    opened_source_id uuid NOT NULL,
    last_source_id uuid NOT NULL,
    opened_at timestamptz NOT NULL,
    last_observed_at timestamptz NOT NULL,
    cleared_at timestamptz,
    notice_sequence integer NOT NULL DEFAULT 1 CHECK (notice_sequence>0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT attention_episode_entity_fk
        FOREIGN KEY(legal_entity_id,tenant_id)
        REFERENCES legal_entities(id,tenant_id),
    CHECK (last_observed_at>=opened_at),
    CHECK (updated_at>=created_at),
    CHECK ((state='OPEN' AND cleared_at IS NULL) OR
           (state='CLEARED' AND cleared_at IS NOT NULL AND cleared_at>=opened_at))
);

CREATE UNIQUE INDEX attention_episodes_open_member_uq
    ON attention_episodes(tenant_id,legal_entity_id,condition_key,member_id)
    WHERE state='OPEN';

CREATE INDEX attention_episodes_subject_idx
    ON attention_episodes(tenant_id,legal_entity_id,subject_type,subject_id,opened_at DESC,id DESC);

CREATE INDEX attention_episodes_open_idx
    ON attention_episodes(tenant_id,legal_entity_id,condition_key,last_observed_at,id)
    WHERE state='OPEN';

CREATE UNIQUE INDEX domain_metric_snapshot_outbox_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type)
    WHERE aggregate_type='DOMAIN_METRIC_SNAPSHOT';

COMMIT;

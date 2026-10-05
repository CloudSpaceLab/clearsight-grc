BEGIN;

CREATE TABLE user_notification_preferences (
    tenant_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    daily_digest_enabled boolean NOT NULL DEFAULT true,
    digest_minute smallint NOT NULL DEFAULT 420 CHECK (digest_minute BETWEEN 0 AND 1439),
    time_zone text NOT NULL DEFAULT 'UTC'
        CHECK (time_zone = btrim(time_zone) AND char_length(time_zone) BETWEEN 1 AND 64),
    quiet_hours_enabled boolean NOT NULL DEFAULT false,
    quiet_start_minute smallint NOT NULL DEFAULT 1320 CHECK (quiet_start_minute BETWEEN 0 AND 1439),
    quiet_end_minute smallint NOT NULL DEFAULT 420 CHECK (quiet_end_minute BETWEEN 0 AND 1439),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, principal_id),
    FOREIGN KEY (principal_id, tenant_id) REFERENCES principals(id, tenant_id) ON DELETE CASCADE,
    CHECK (NOT quiet_hours_enabled OR quiet_start_minute <> quiet_end_minute)
);

COMMIT;

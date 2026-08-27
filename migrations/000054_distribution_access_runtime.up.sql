BEGIN;

ALTER TABLE capture_otp_challenges
    ADD COLUMN resends integer NOT NULL DEFAULT 0 CHECK (resends BETWEEN 0 AND 3),
    ADD COLUMN max_resends integer NOT NULL DEFAULT 3 CHECK (max_resends BETWEEN 1 AND 3 AND resends <= max_resends);

ALTER TABLE capture_distribution_recipients
    ADD CONSTRAINT capture_distribution_recipients_session_binding_key
    UNIQUE (id,tenant_id,legal_entity_id,distribution_id,request_id);

CREATE UNIQUE INDEX capture_access_routes_active_shared_uq
    ON capture_access_routes(tenant_id,legal_entity_id,distribution_id)
    WHERE revoked_at IS NULL AND access_policy='SHARED_LINK_EMAIL_OTP';

CREATE UNIQUE INDEX capture_access_routes_active_direct_uq
    ON capture_access_routes(tenant_id,legal_entity_id,distribution_id,recipient_id)
    WHERE revoked_at IS NULL AND access_policy<>'SHARED_LINK_EMAIL_OTP';

CREATE TABLE capture_distribution_sessions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    recipient_id uuid NOT NULL,
    request_id uuid NOT NULL,
    route_id uuid NOT NULL,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
    audience_hint text NOT NULL DEFAULT '' CHECK (char_length(audience_hint) <= 320),
    assurance text NOT NULL CHECK (assurance IN ('LINK_POSSESSION','EMAIL_VERIFIED')),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    UNIQUE (id,tenant_id,legal_entity_id,distribution_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id)
        REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (route_id,tenant_id,legal_entity_id,distribution_id)
        REFERENCES capture_access_routes(id,tenant_id,legal_entity_id,distribution_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id,tenant_id,legal_entity_id,distribution_id,request_id)
        REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id,distribution_id,request_id),
    FOREIGN KEY (request_id,tenant_id,legal_entity_id,distribution_id)
        REFERENCES capture_requests(id,tenant_id,legal_entity_id,distribution_id),
    CHECK (expires_at > created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX capture_distribution_sessions_route_idx
    ON capture_distribution_sessions(tenant_id,legal_entity_id,distribution_id,route_id,expires_at,id)
    WHERE revoked_at IS NULL;

COMMIT;

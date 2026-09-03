BEGIN;

ALTER TABLE capture_access_routes
    ADD COLUMN max_redemptions integer,
    ADD COLUMN redemptions integer;

UPDATE capture_access_routes
SET max_redemptions = CASE WHEN access_policy = 'SHARED_LINK_EMAIL_OTP' THEN 20 ELSE 1 END,
    redemptions = CASE
        WHEN access_policy = 'SHARED_LINK_EMAIL_OTP' THEN LEAST(20::bigint, (
            SELECT count(*) FROM capture_distribution_sessions session
            WHERE session.route_id = capture_access_routes.id
              AND session.tenant_id = capture_access_routes.tenant_id
              AND session.legal_entity_id = capture_access_routes.legal_entity_id
              AND session.distribution_id = capture_access_routes.distribution_id
        ))::integer
        WHEN EXISTS (
            SELECT 1 FROM capture_distribution_sessions session
            WHERE session.route_id = capture_access_routes.id
              AND session.tenant_id = capture_access_routes.tenant_id
              AND session.legal_entity_id = capture_access_routes.legal_entity_id
              AND session.distribution_id = capture_access_routes.distribution_id
        ) THEN 1
        ELSE 0
    END;

ALTER TABLE capture_access_routes
    ALTER COLUMN max_redemptions SET DEFAULT 1,
    ALTER COLUMN max_redemptions SET NOT NULL,
    ALTER COLUMN redemptions SET DEFAULT 0,
    ALTER COLUMN redemptions SET NOT NULL,
    ADD CONSTRAINT capture_access_routes_max_redemptions_check CHECK (max_redemptions BETWEEN 1 AND 20),
    ADD CONSTRAINT capture_access_routes_redemptions_check CHECK (redemptions >= 0 AND redemptions <= max_redemptions);

COMMIT;

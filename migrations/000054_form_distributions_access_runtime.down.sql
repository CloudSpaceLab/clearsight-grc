BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM capture_distribution_sessions) THEN
        RAISE EXCEPTION 'cannot roll back distribution access runtime after external sessions exist';
    END IF;
END;
$$;

DROP TABLE IF EXISTS capture_distribution_sessions;
DROP INDEX IF EXISTS capture_access_routes_active_direct_uq;
DROP INDEX IF EXISTS capture_access_routes_active_shared_uq;

DROP INDEX IF EXISTS capture_otp_challenges_active_idx;
CREATE INDEX capture_otp_challenges_active_idx
    ON capture_otp_challenges(tenant_id,distribution_id,route_id,expires_at DESC,id DESC)
    WHERE consumed_at IS NULL AND attempts < max_attempts;

ALTER TABLE capture_distribution_recipients
    DROP CONSTRAINT IF EXISTS capture_distribution_recipients_session_binding_key;

COMMIT;

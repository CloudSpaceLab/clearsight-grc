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

ALTER TABLE capture_distribution_recipients
    DROP CONSTRAINT IF EXISTS capture_distribution_recipients_session_binding_key;

ALTER TABLE capture_otp_challenges
    DROP COLUMN IF EXISTS max_resends,
    DROP COLUMN IF EXISTS resends;

COMMIT;

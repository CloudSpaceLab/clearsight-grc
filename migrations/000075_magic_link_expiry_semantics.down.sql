BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM capture_access_routes
        WHERE revoked_at IS NULL AND access_policy='DIRECT_MAGIC_LINK'
        GROUP BY tenant_id,legal_entity_id,distribution_id,recipient_id
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot restore single-active-link constraint while multiple unrevoked magic links exist';
    END IF;
END;
$$;

DROP INDEX capture_access_routes_active_direct_otp_uq;

CREATE UNIQUE INDEX capture_access_routes_active_direct_uq
    ON capture_access_routes(tenant_id,legal_entity_id,distribution_id,recipient_id)
    WHERE revoked_at IS NULL AND access_policy<>'SHARED_LINK_EMAIL_OTP';

COMMIT;

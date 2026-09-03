BEGIN;

DROP INDEX capture_access_routes_active_direct_uq;

CREATE UNIQUE INDEX capture_access_routes_active_direct_otp_uq
    ON capture_access_routes(tenant_id,legal_entity_id,distribution_id,recipient_id)
    WHERE revoked_at IS NULL AND access_policy='DIRECT_LINK_EMAIL_OTP';

COMMIT;

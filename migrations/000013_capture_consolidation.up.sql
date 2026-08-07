BEGIN;

-- The executable capture domain has been owned by capture_requests and its
-- invitation/session/submission/artifact tables since migration 000005. These
-- foundation-era tables have no runtime owner and represented a second,
-- incompatible request/invitation model.
DROP TABLE IF EXISTS invitation_grants;
DROP TABLE IF EXISTS evidence_requests;

COMMIT;

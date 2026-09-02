BEGIN;

DO $$
DECLARE constraint_name text;
BEGIN
    SELECT c.conname INTO constraint_name
    FROM pg_constraint c
    WHERE c.conrelid='third_party_assessment_request_links'::regclass
      AND c.contype='f'
      AND c.confrelid='capture_invitations'::regclass;
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE third_party_assessment_request_links DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- This is an unreleased environment. Old invitation references cannot be
-- redeemed by the canonical capture application and must not be retained as
-- if they were current access routes.
UPDATE third_party_assessment_request_links SET invitation_id=NULL WHERE invitation_id IS NOT NULL;

ALTER TABLE third_party_assessment_request_links
    ADD CONSTRAINT third_party_assessment_request_links_access_route_fk
    FOREIGN KEY (invitation_id,tenant_id) REFERENCES capture_access_routes(id,tenant_id);

COMMIT;

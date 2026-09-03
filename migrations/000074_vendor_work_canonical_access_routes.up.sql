BEGIN;

DO $$
DECLARE constraint_name text;
BEGIN
    SELECT c.conname INTO constraint_name
    FROM pg_constraint c
    WHERE c.conrelid='third_party_work_requests'::regclass
      AND c.contype='f'
      AND c.confrelid='capture_invitations'::regclass;
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE third_party_work_requests DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

DO $$
DECLARE constraint_name text;
BEGIN
    SELECT c.conname INTO constraint_name
    FROM pg_constraint c
    WHERE c.conrelid='third_party_work_capture_links'::regclass
      AND c.contype='f'
      AND c.confrelid='capture_invitations'::regclass;
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE third_party_work_capture_links DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- No released data depends on the retired invitation issuer. Old references
-- cannot authorize the canonical capture application and are cleared instead
-- of being interpreted as access-route identifiers.
UPDATE third_party_work_requests SET current_invitation_id=NULL WHERE current_invitation_id IS NOT NULL;
UPDATE third_party_work_capture_links SET invitation_id=NULL WHERE invitation_id IS NOT NULL;

ALTER TABLE third_party_work_requests
    ADD CONSTRAINT third_party_work_requests_access_route_fk
    FOREIGN KEY (current_invitation_id,tenant_id) REFERENCES capture_access_routes(id,tenant_id);

ALTER TABLE third_party_work_capture_links
    ADD CONSTRAINT third_party_work_capture_links_access_route_fk
    FOREIGN KEY (invitation_id,tenant_id) REFERENCES capture_access_routes(id,tenant_id);

COMMIT;

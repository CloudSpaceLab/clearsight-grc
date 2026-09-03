BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM third_party_work_requests WHERE current_invitation_id IS NOT NULL)
       OR EXISTS (SELECT 1 FROM third_party_work_capture_links WHERE invitation_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot roll back canonical vendor-work access routes while audit associations exist';
    END IF;
END $$;

ALTER TABLE third_party_work_requests
    DROP CONSTRAINT IF EXISTS third_party_work_requests_access_route_fk;
ALTER TABLE third_party_work_capture_links
    DROP CONSTRAINT IF EXISTS third_party_work_capture_links_access_route_fk;

DROP TRIGGER IF EXISTS validate_vendor_work_access_route_proof ON third_party_work_invitation_reservations;
DROP FUNCTION IF EXISTS validate_vendor_work_access_route_proof();
ALTER TABLE third_party_work_invitation_reservations
    DROP CONSTRAINT IF EXISTS third_party_work_reservations_access_route_fk,
    DROP CONSTRAINT IF EXISTS third_party_work_reservations_access_route_uq,
    DROP COLUMN IF EXISTS access_route_id;

ALTER TABLE third_party_work_requests
    ADD CONSTRAINT third_party_work_requests_invitation_fk
    FOREIGN KEY (current_invitation_id,tenant_id,current_request_id) REFERENCES capture_invitations(id,tenant_id,request_id);
ALTER TABLE third_party_work_capture_links
    ADD CONSTRAINT third_party_work_capture_links_invitation_fk
    FOREIGN KEY (invitation_id,tenant_id,request_id) REFERENCES capture_invitations(id,tenant_id,request_id);

COMMIT;

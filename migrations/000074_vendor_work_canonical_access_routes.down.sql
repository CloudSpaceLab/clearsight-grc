BEGIN;

ALTER TABLE third_party_work_requests
    DROP CONSTRAINT IF EXISTS third_party_work_requests_access_route_fk;
ALTER TABLE third_party_work_capture_links
    DROP CONSTRAINT IF EXISTS third_party_work_capture_links_access_route_fk;

UPDATE third_party_work_requests SET current_invitation_id=NULL WHERE current_invitation_id IS NOT NULL;
UPDATE third_party_work_capture_links SET invitation_id=NULL WHERE invitation_id IS NOT NULL;

ALTER TABLE third_party_work_requests
    ADD CONSTRAINT third_party_work_requests_invitation_fk
    FOREIGN KEY (current_invitation_id,tenant_id,current_request_id) REFERENCES capture_invitations(id,tenant_id,request_id);
ALTER TABLE third_party_work_capture_links
    ADD CONSTRAINT third_party_work_capture_links_invitation_fk
    FOREIGN KEY (invitation_id,tenant_id,request_id) REFERENCES capture_invitations(id,tenant_id,request_id);

COMMIT;

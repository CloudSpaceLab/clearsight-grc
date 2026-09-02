BEGIN;

ALTER TABLE third_party_assessment_request_links
    DROP CONSTRAINT IF EXISTS third_party_assessment_request_links_access_route_fk;
UPDATE third_party_assessment_request_links SET invitation_id=NULL WHERE invitation_id IS NOT NULL;
ALTER TABLE third_party_assessment_request_links
    ADD CONSTRAINT third_party_assessment_request_links_invitation_fk
    FOREIGN KEY (invitation_id,tenant_id,request_id) REFERENCES capture_invitations(id,tenant_id,request_id);

COMMIT;

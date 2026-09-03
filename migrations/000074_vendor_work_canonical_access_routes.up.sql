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

ALTER TABLE third_party_work_invitation_reservations
    ADD COLUMN access_route_id uuid;

ALTER TABLE third_party_work_invitation_reservations
    ADD CONSTRAINT third_party_work_reservations_access_route_uq
        UNIQUE (access_route_id,tenant_id,request_id),
    ADD CONSTRAINT third_party_work_reservations_access_route_fk
        FOREIGN KEY (access_route_id,tenant_id) REFERENCES capture_access_routes(id,tenant_id);

CREATE FUNCTION validate_vendor_work_access_route_proof() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.access_route_id IS NOT NULL AND NOT EXISTS (
        SELECT 1
        FROM capture_access_routes route
        JOIN capture_distribution_recipients recipient
          ON recipient.id=route.recipient_id
         AND recipient.tenant_id=route.tenant_id
         AND recipient.legal_entity_id=route.legal_entity_id
         AND recipient.distribution_id=route.distribution_id
        WHERE route.id=NEW.access_route_id
          AND route.tenant_id=NEW.tenant_id
          AND route.legal_entity_id=NEW.legal_entity_id
          AND recipient.request_id=NEW.request_id
    ) THEN
        RAISE EXCEPTION 'vendor-work access route does not belong to the reserved request'
            USING ERRCODE='foreign_key_violation';
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER validate_vendor_work_access_route_proof
    BEFORE INSERT OR UPDATE OF access_route_id,tenant_id,legal_entity_id,request_id
    ON third_party_work_invitation_reservations
    FOR EACH ROW EXECUTE FUNCTION validate_vendor_work_access_route_proof();

ALTER TABLE third_party_work_requests
    ADD CONSTRAINT third_party_work_requests_access_route_fk
    FOREIGN KEY (current_invitation_id,tenant_id,current_request_id)
        REFERENCES third_party_work_invitation_reservations(access_route_id,tenant_id,request_id)
        NOT VALID;

ALTER TABLE third_party_work_capture_links
    ADD CONSTRAINT third_party_work_capture_links_access_route_fk
    FOREIGN KEY (invitation_id,tenant_id,request_id)
        REFERENCES third_party_work_invitation_reservations(access_route_id,tenant_id,request_id)
        NOT VALID;

COMMIT;

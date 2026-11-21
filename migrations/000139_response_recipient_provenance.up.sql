BEGIN;

-- Link-issued submissions retain their authorized recipient route. Historical
-- rows remain unknown; a route is not proof of the identity of its holder.
ALTER TABLE capture_response_revisions
    ADD COLUMN submitted_recipient_id uuid,
    ADD COLUMN submitted_recipient_principal_id uuid,
    ADD COLUMN submitted_recipient_type text,
    ADD COLUMN submitted_recipient_label text,
    ADD CONSTRAINT capture_response_revisions_recipient_scope_fk
      FOREIGN KEY (submitted_recipient_id,tenant_id,legal_entity_id,distribution_id)
      REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id,distribution_id),
    ADD CONSTRAINT capture_response_revisions_recipient_principal_fk
      FOREIGN KEY (submitted_recipient_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT capture_response_revisions_recipient_principal_check
      CHECK ((submitted_recipient_id IS NULL AND submitted_recipient_principal_id IS NULL AND submitted_recipient_type IS NULL AND submitted_recipient_label IS NULL)
        OR (submitted_recipient_id IS NOT NULL AND submitted_recipient_type IS NOT NULL AND char_length(COALESCE(submitted_recipient_label,'')) <= 320 AND
          ((submitted_recipient_type = 'INTERNAL_PRINCIPAL' AND submitted_recipient_principal_id IS NOT NULL)
           OR (submitted_recipient_type = 'EXTERNAL_AUDIENCE' AND submitted_recipient_principal_id IS NULL))));

CREATE FUNCTION guard_response_recipient_provenance() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' THEN
    IF ROW(NEW.submitted_recipient_id,NEW.submitted_recipient_principal_id,NEW.submitted_recipient_type,NEW.submitted_recipient_label)
       IS DISTINCT FROM ROW(OLD.submitted_recipient_id,OLD.submitted_recipient_principal_id,OLD.submitted_recipient_type,OLD.submitted_recipient_label) THEN
      RAISE EXCEPTION 'response recipient provenance is immutable';
    END IF;
    RETURN NEW;
  END IF;
  IF NEW.submitted_recipient_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM capture_distribution_recipients recipient
    JOIN capture_submissions submission ON submission.id=NEW.submission_id AND submission.tenant_id=NEW.tenant_id
    WHERE recipient.id=NEW.submitted_recipient_id AND recipient.tenant_id=NEW.tenant_id
      AND recipient.legal_entity_id=NEW.legal_entity_id AND recipient.distribution_id=NEW.distribution_id
      AND recipient.request_id=submission.request_id AND recipient.role='TO'
      AND recipient.recipient_type=NEW.submitted_recipient_type
      AND recipient.principal_id IS NOT DISTINCT FROM NEW.submitted_recipient_principal_id
      AND recipient.contact_label=COALESCE(NEW.submitted_recipient_label,'')
  ) THEN
    RAISE EXCEPTION 'response recipient provenance does not match the submission';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER capture_response_recipient_provenance_guard
BEFORE INSERT OR UPDATE OF submitted_recipient_id,submitted_recipient_principal_id,submitted_recipient_type,submitted_recipient_label
ON capture_response_revisions FOR EACH ROW EXECUTE FUNCTION guard_response_recipient_provenance();

COMMIT;

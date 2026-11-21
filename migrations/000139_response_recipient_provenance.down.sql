BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM capture_response_revisions WHERE submitted_recipient_id IS NOT NULL LIMIT 1) THEN
    RAISE EXCEPTION 'cannot discard recorded response recipient provenance';
  END IF;
END;
$$;

DROP TRIGGER capture_response_recipient_provenance_guard ON capture_response_revisions;
DROP FUNCTION guard_response_recipient_provenance();
ALTER TABLE capture_response_revisions
    DROP CONSTRAINT capture_response_revisions_recipient_principal_check,
    DROP CONSTRAINT capture_response_revisions_recipient_principal_fk,
    DROP CONSTRAINT capture_response_revisions_recipient_scope_fk,
    DROP COLUMN submitted_recipient_label,
    DROP COLUMN submitted_recipient_type,
    DROP COLUMN submitted_recipient_principal_id,
    DROP COLUMN submitted_recipient_id;

COMMIT;

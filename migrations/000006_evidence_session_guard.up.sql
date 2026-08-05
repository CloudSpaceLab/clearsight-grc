BEGIN;

CREATE OR REPLACE FUNCTION enforce_open_evidence_session()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM capture_requests er
        WHERE er.id = NEW.request_id
          AND er.tenant_id = NEW.tenant_id
          AND er.status IN ('READY','IN_PROGRESS')
    ) THEN
        RAISE EXCEPTION 'evidence request is not open' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER capture_sessions_open_request_guard
BEFORE INSERT ON capture_sessions
FOR EACH ROW EXECUTE FUNCTION enforce_open_evidence_session();

COMMIT;

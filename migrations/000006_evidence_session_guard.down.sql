BEGIN;
DROP TRIGGER IF EXISTS capture_sessions_open_request_guard ON capture_sessions;
DROP FUNCTION IF EXISTS enforce_open_evidence_session();
COMMIT;

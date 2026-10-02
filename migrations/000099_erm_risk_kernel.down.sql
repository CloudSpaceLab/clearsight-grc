BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM risks LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed risk history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS risk_events_immutable ON risk_events;
DROP TRIGGER IF EXISTS risk_assessments_immutable ON risk_assessments;
DROP TRIGGER IF EXISTS risk_appetite_immutable ON risk_appetite_statements;
DROP TRIGGER IF EXISTS risk_revisions_immutable ON risk_revisions;
DROP FUNCTION IF EXISTS protect_risk_immutable();
DROP TABLE IF EXISTS risk_events;
DROP TABLE IF EXISTS risk_assessments;
DROP TABLE IF EXISTS risk_appetite_statements;
DROP TABLE IF EXISTS risk_revisions;
DROP TABLE IF EXISTS risks;

COMMIT;

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM risk_indicator_links LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed Risk indicator history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS risk_indicator_links_immutable ON risk_indicator_links;
DROP TABLE IF EXISTS risk_indicator_links;
DROP INDEX IF EXISTS monitoring_checks_indicator_scope_idx;

COMMIT;

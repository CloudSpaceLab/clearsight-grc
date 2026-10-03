BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM rcsa_cycles LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed RCSA cycle history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS rcsa_cycle_events_immutable ON rcsa_cycle_events;
DROP TRIGGER IF EXISTS rcsa_cycle_revisions_immutable ON rcsa_cycle_revisions;
DROP TRIGGER IF EXISTS rcsa_cycle_controls_immutable ON rcsa_cycle_controls;
DROP TRIGGER IF EXISTS rcsa_cycle_risks_immutable ON rcsa_cycle_risks;
DROP FUNCTION IF EXISTS protect_rcsa_immutable();
DROP TABLE IF EXISTS rcsa_cycle_events;
DROP TABLE IF EXISTS rcsa_cycle_revisions;
DROP TABLE IF EXISTS rcsa_cycle_controls;
DROP TABLE IF EXISTS rcsa_cycle_risks;
DROP TABLE IF EXISTS rcsa_cycles;
DROP INDEX IF EXISTS risk_control_links_rcsa_scope_idx;

COMMIT;

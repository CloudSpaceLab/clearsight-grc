BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM operational_losses LIMIT 1)
       OR EXISTS (SELECT 1 FROM operational_loss_recoveries LIMIT 1)
       OR EXISTS (SELECT 1 FROM operational_loss_revisions LIMIT 1)
       OR EXISTS (SELECT 1 FROM operational_loss_events LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed operational loss history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS operational_loss_events_immutable ON operational_loss_events;
DROP TRIGGER IF EXISTS operational_loss_recoveries_immutable ON operational_loss_recoveries;
DROP TRIGGER IF EXISTS operational_loss_revisions_immutable ON operational_loss_revisions;
DROP FUNCTION IF EXISTS protect_operational_loss_immutable();

DROP TABLE IF EXISTS operational_loss_events;
DROP TABLE IF EXISTS operational_loss_recoveries;
DROP TABLE IF EXISTS operational_loss_revisions;
DROP TABLE IF EXISTS operational_losses;

COMMIT;

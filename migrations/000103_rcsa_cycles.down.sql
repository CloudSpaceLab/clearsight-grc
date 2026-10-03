BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM rcsa_cycle_item_distributions LIMIT 1)
       OR EXISTS (SELECT 1 FROM rcsa_cycle_item_controls LIMIT 1)
       OR EXISTS (SELECT 1 FROM rcsa_cycle_items LIMIT 1)
       OR EXISTS (SELECT 1 FROM rcsa_cycles LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed RCSA cycle history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS rcsa_cycle_item_distributions_immutable ON rcsa_cycle_item_distributions;
DROP TRIGGER IF EXISTS rcsa_cycle_item_controls_immutable ON rcsa_cycle_item_controls;
DROP TRIGGER IF EXISTS rcsa_cycle_items_immutable ON rcsa_cycle_items;
DROP TRIGGER IF EXISTS rcsa_cycles_immutable ON rcsa_cycles;
DROP FUNCTION IF EXISTS protect_rcsa_immutable();
DROP TABLE IF EXISTS rcsa_cycle_item_distributions;
DROP TABLE IF EXISTS rcsa_cycle_item_controls;
DROP TABLE IF EXISTS rcsa_cycle_items;
DROP TABLE IF EXISTS rcsa_cycles;
DROP INDEX IF EXISTS risk_control_links_rcsa_scope_idx;

COMMIT;

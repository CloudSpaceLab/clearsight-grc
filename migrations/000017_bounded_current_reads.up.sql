BEGIN;

ALTER TABLE matter_decisions
    ADD COLUMN matter_version bigint NOT NULL DEFAULT 0;

UPDATE matter_decisions d
SET matter_version = e.aggregate_version
FROM continuity_events e
WHERE e.tenant_id = d.tenant_id
  AND e.aggregate_type = 'MATTER'
  AND e.aggregate_id = d.matter_id
  AND e.event_type = 'DECISION_ADDED'
  AND e.payload->>'id' = d.id::text;

CREATE INDEX matter_decisions_current_order_idx
    ON matter_decisions(tenant_id,matter_id,matter_version,id);

ALTER TABLE response_packages
    ADD COLUMN matter_version bigint NOT NULL DEFAULT 0;

UPDATE response_packages r
SET matter_version = source.aggregate_version
FROM (
    SELECT tenant_id,aggregate_id,(payload->>'id')::uuid AS response_id,max(aggregate_version) AS aggregate_version
    FROM continuity_events
    WHERE aggregate_type='MATTER'
      AND event_type IN ('RESPONSE_PACKAGE_ADDED','RESPONSE_PACKAGE_STATE_CHANGED')
      AND payload ? 'id'
    GROUP BY tenant_id,aggregate_id,(payload->>'id')::uuid
) source
WHERE source.tenant_id=r.tenant_id
  AND source.aggregate_id=r.matter_id
  AND source.response_id=r.id;

CREATE INDEX response_packages_current_order_idx
    ON response_packages(tenant_id,matter_id,matter_version,id);

-- Projection rows are written before their continuity event in the same
-- transaction. Capture the authoritative aggregate sequence after that event
-- lands so current reads never need to search the event log for ordering.
CREATE OR REPLACE FUNCTION sync_continuity_current_order()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    record_id uuid;
BEGIN
    IF NEW.aggregate_type <> 'MATTER' OR NOT (NEW.payload ? 'id') THEN
        RETURN NEW;
    END IF;
    record_id := NULLIF(NEW.payload->>'id','')::uuid;
    IF record_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.event_type = 'DECISION_ADDED' THEN
        UPDATE matter_decisions
        SET matter_version=NEW.aggregate_version
        WHERE tenant_id=NEW.tenant_id AND matter_id=NEW.aggregate_id AND id=record_id;
    ELSIF NEW.event_type IN ('RESPONSE_PACKAGE_ADDED','RESPONSE_PACKAGE_STATE_CHANGED') THEN
        UPDATE response_packages
        SET matter_version=NEW.aggregate_version
        WHERE tenant_id=NEW.tenant_id AND matter_id=NEW.aggregate_id AND id=record_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER continuity_current_order_trg
AFTER INSERT ON continuity_events
FOR EACH ROW EXECUTE FUNCTION sync_continuity_current_order();

-- A Matter Action owns domain truth. Its actor-facing Workflow Task is a
-- projection, so one projection workflow is enough for each action.
CREATE UNIQUE INDEX workflow_instances_matter_action_subject_idx
    ON workflow_instances(tenant_id,kind,subject_type,subject_id)
    WHERE kind='MATTER_ACTION';

COMMIT;

BEGIN;

ALTER TABLE matter_decisions
    DROP CONSTRAINT IF EXISTS matter_decisions_status_check;
ALTER TABLE matter_decisions
    ADD CONSTRAINT matter_decisions_status_check CHECK (
        status IN ('PROPOSED','IN_REVIEW','CHALLENGED','APPROVED','CONDITIONALLY_APPROVED','REJECTED','RETURNED','EXPIRED','SUPERSEDED')
    );

ALTER TABLE matter_decisions
    ADD COLUMN proposed_by uuid,
    ADD COLUMN reviewed_by uuid,
    ADD COLUMN challenged_by uuid;
ALTER TABLE matter_decisions
    ADD CONSTRAINT matter_decisions_proposed_by_tenant_fk FOREIGN KEY (proposed_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT matter_decisions_reviewed_by_tenant_fk FOREIGN KEY (reviewed_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT matter_decisions_challenged_by_tenant_fk FOREIGN KEY (challenged_by,tenant_id) REFERENCES principals(id,tenant_id);

ALTER TABLE response_packages
    ADD COLUMN prepared_by uuid,
    ADD COLUMN reviewed_by uuid,
    ADD COLUMN rejected_by uuid,
    ADD COLUMN withdrawn_by uuid,
    ADD COLUMN transmitted_by uuid,
    ADD COLUMN acknowledged_by uuid;
ALTER TABLE response_packages
    ADD CONSTRAINT response_packages_prepared_by_tenant_fk FOREIGN KEY (prepared_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT response_packages_reviewed_by_tenant_fk FOREIGN KEY (reviewed_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT response_packages_rejected_by_tenant_fk FOREIGN KEY (rejected_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT response_packages_withdrawn_by_tenant_fk FOREIGN KEY (withdrawn_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT response_packages_transmitted_by_tenant_fk FOREIGN KEY (transmitted_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT response_packages_acknowledged_by_tenant_fk FOREIGN KEY (acknowledged_by,tenant_id) REFERENCES principals(id,tenant_id);

-- Backfill lifecycle actors from the append-only event envelope. This preserves
-- historical actor truth without trusting legacy payload actor fields.
UPDATE matter_decisions d
SET proposed_by = e.actor_id,
    authority_principal_id = NULL
FROM continuity_events e
WHERE e.tenant_id = d.tenant_id
  AND e.event_type = 'DECISION_ADDED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = d.id
  AND d.status = 'PROPOSED';

UPDATE matter_decisions d
SET reviewed_by = e.actor_id,
    authority_principal_id = NULL
FROM continuity_events e
WHERE e.tenant_id = d.tenant_id
  AND e.event_type = 'DECISION_ADDED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = d.id
  AND d.status = 'RETURNED';

UPDATE response_packages r
SET prepared_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_ADDED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id;

UPDATE response_packages r
SET reviewed_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_STATE_CHANGED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id
  AND e.payload->>'status' = 'IN_REVIEW';

UPDATE response_packages r
SET rejected_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_STATE_CHANGED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id
  AND e.payload->>'status' = 'REJECTED';

UPDATE response_packages r
SET withdrawn_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_STATE_CHANGED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id
  AND e.payload->>'status' = 'WITHDRAWN';

UPDATE response_packages r
SET transmitted_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_STATE_CHANGED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id
  AND e.payload->>'status' = 'TRANSMITTED';

UPDATE response_packages r
SET acknowledged_by = e.actor_id
FROM continuity_events e
WHERE e.tenant_id = r.tenant_id
  AND e.event_type = 'RESPONSE_PACKAGE_STATE_CHANGED'
  AND e.actor_id IS NOT NULL
  AND (e.payload->>'id')::uuid = r.id
  AND e.payload->>'status' = 'ACKNOWLEDGED';

CREATE OR REPLACE FUNCTION sync_continuity_lifecycle_actor()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    record_id uuid;
    lifecycle_status text;
BEGIN
    IF NEW.actor_id IS NULL THEN
        RETURN NEW;
    END IF;
    record_id := NULLIF(NEW.payload->>'id','')::uuid;
    lifecycle_status := NEW.payload->>'status';

    IF NEW.event_type = 'DECISION_ADDED' AND record_id IS NOT NULL THEN
        UPDATE matter_decisions
        SET proposed_by = CASE WHEN lifecycle_status='PROPOSED' THEN NEW.actor_id ELSE proposed_by END,
            reviewed_by = CASE WHEN lifecycle_status IN ('IN_REVIEW','RETURNED') THEN NEW.actor_id ELSE reviewed_by END,
            challenged_by = CASE WHEN lifecycle_status='CHALLENGED' THEN NEW.actor_id ELSE challenged_by END,
            authority_principal_id = CASE
                WHEN lifecycle_status IN ('APPROVED','CONDITIONALLY_APPROVED','REJECTED','EXPIRED','SUPERSEDED') THEN NEW.actor_id
                WHEN lifecycle_status IN ('PROPOSED','IN_REVIEW','CHALLENGED','RETURNED') THEN NULL
                ELSE authority_principal_id
            END
        WHERE tenant_id=NEW.tenant_id AND id=record_id;
    ELSIF NEW.event_type IN ('RESPONSE_PACKAGE_ADDED','RESPONSE_PACKAGE_STATE_CHANGED') AND record_id IS NOT NULL THEN
        UPDATE response_packages
        SET prepared_by = CASE WHEN lifecycle_status='DRAFT' THEN NEW.actor_id ELSE prepared_by END,
            reviewed_by = CASE WHEN lifecycle_status='IN_REVIEW' THEN NEW.actor_id ELSE reviewed_by END,
            rejected_by = CASE WHEN lifecycle_status='REJECTED' THEN NEW.actor_id ELSE rejected_by END,
            withdrawn_by = CASE WHEN lifecycle_status='WITHDRAWN' THEN NEW.actor_id ELSE withdrawn_by END,
            transmitted_by = CASE WHEN lifecycle_status='TRANSMITTED' THEN NEW.actor_id ELSE transmitted_by END,
            acknowledged_by = CASE WHEN lifecycle_status='ACKNOWLEDGED' THEN NEW.actor_id ELSE acknowledged_by END
        WHERE tenant_id=NEW.tenant_id AND id=record_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER continuity_lifecycle_actor_trg
AFTER INSERT ON continuity_events
FOR EACH ROW EXECUTE FUNCTION sync_continuity_lifecycle_actor();

COMMIT;

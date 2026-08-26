BEGIN;

ALTER TABLE capture_requests ADD COLUMN legal_entity_id uuid;

-- Resolve subject types with authoritative entity-bearing parents first.
UPDATE capture_requests cr
SET legal_entity_id=p.legal_entity_id
FROM programs p
WHERE cr.tenant_id=p.tenant_id
  AND cr.subject_type='PROGRAM'
  AND cr.subject_id=p.id::text;

UPDATE capture_requests cr
SET legal_entity_id=m.legal_entity_id
FROM matters m
WHERE cr.tenant_id=m.tenant_id
  AND cr.subject_type='MATTER'
  AND cr.subject_id=m.id::text;

-- Historical CONTROL records used both names for the implementation subject.
-- New commands remain limited to subject adapters implemented in code, but an
-- exact existing implementation can still be reconstructed for legacy reads.
UPDATE capture_requests cr
SET legal_entity_id=p.legal_entity_id
FROM control_implementations ci
JOIN programs p
  ON p.tenant_id=ci.tenant_id
 AND p.id=ci.program_id
WHERE cr.tenant_id=ci.tenant_id
  AND cr.subject_type IN ('CONTROL','CONTROL_IMPLEMENTATION')
  AND cr.subject_id=ci.id::text;

-- Other historical subject adapters do not yet have canonical tables. As in
-- migration 000035, a row can only be reconstructed automatically when the
-- tenant has one currently effective legal entity. Multi-entity tenants must
-- repair those rows explicitly before this migration can complete.
WITH single_entity AS (
    SELECT tenant_id,min(id::text)::uuid AS legal_entity_id
    FROM legal_entities
    WHERE valid_from<=clock_timestamp()
      AND (valid_until IS NULL OR clock_timestamp()<valid_until)
    GROUP BY tenant_id
    HAVING count(*)=1
)
UPDATE capture_requests cr
SET legal_entity_id=se.legal_entity_id
FROM single_entity se
WHERE cr.tenant_id=se.tenant_id
  AND cr.legal_entity_id IS NULL;

DO $$
DECLARE unresolved_count bigint;
BEGIN
    SELECT count(*) INTO unresolved_count
    FROM capture_requests
    WHERE legal_entity_id IS NULL;
    IF unresolved_count>0 THEN
        RAISE EXCEPTION 'evidence request legal-entity migration unresolved rows: %',unresolved_count;
    END IF;
END $$;

ALTER TABLE capture_requests ALTER COLUMN legal_entity_id SET NOT NULL;

ALTER TABLE capture_requests
    ADD CONSTRAINT capture_requests_legal_entity_tenant_fk
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id);

CREATE INDEX capture_requests_entity_queue_idx
    ON capture_requests(tenant_id,legal_entity_id,status,deadline,id);

CREATE INDEX capture_requests_entity_subject_idx
    ON capture_requests(tenant_id,legal_entity_id,subject_type,subject_id,created_at DESC);

CREATE TRIGGER capture_requests_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON capture_requests
FOR EACH ROW EXECUTE FUNCTION prevent_legal_entity_change();

COMMIT;

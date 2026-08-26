BEGIN;

ALTER TABLE matters ADD COLUMN legal_entity_id uuid;

-- A legacy Program without entity identity is only safe to resolve when its
-- tenant has exactly one currently effective legal entity. Multi-entity or
-- otherwise unresolved tenants must be repaired explicitly before retrying.
WITH single_entity AS (
    SELECT tenant_id, min(id::text)::uuid AS legal_entity_id
    FROM legal_entities
    WHERE valid_from<=clock_timestamp()
      AND (valid_until IS NULL OR clock_timestamp()<valid_until)
    GROUP BY tenant_id
    HAVING count(*)=1
)
UPDATE programs p
SET legal_entity_id=s.legal_entity_id
FROM single_entity s
WHERE s.tenant_id=p.tenant_id
  AND p.legal_entity_id IS NULL;

DO $$
DECLARE unresolved_program_count bigint;
BEGIN
    SELECT count(*) INTO unresolved_program_count FROM programs WHERE legal_entity_id IS NULL;
    IF unresolved_program_count > 0 THEN
        RAISE EXCEPTION 'program legal-entity migration unresolved rows: %', unresolved_program_count;
    END IF;
END $$;

UPDATE continuity_events ce
SET payload=ce.payload || jsonb_build_object('legal_entity_id',p.legal_entity_id::text)
FROM programs p
WHERE ce.tenant_id=p.tenant_id
  AND ce.aggregate_type='PROGRAM'
  AND ce.aggregate_id=p.id
  AND ce.event_type='PROGRAM_CREATED';

UPDATE outbox_events oe
SET payload=oe.payload || jsonb_build_object('legal_entity_id',p.legal_entity_id::text)
FROM programs p
WHERE oe.tenant_id=p.tenant_id
  AND oe.aggregate_type='PROGRAM'
  AND oe.aggregate_id=p.id
  AND oe.event_type='PROGRAM_CREATED'
  AND oe.published_at IS NULL;

ALTER TABLE programs ALTER COLUMN legal_entity_id SET NOT NULL;

DROP INDEX IF EXISTS programs_active_code_idx;
CREATE UNIQUE INDEX programs_active_code_idx
    ON programs(tenant_id,legal_entity_id,code)
    WHERE status<>'RETIRED';

-- Only infer a legacy Matter's legal entity when every linked Program with an
-- entity points to the same entity. Unlinked, entity-less and ambiguous legacy
-- Matters remain NULL and are intentionally invisible to entity-scoped actors.
WITH unambiguous AS (
    SELECT ml.tenant_id, ml.matter_id, min(p.legal_entity_id::text)::uuid AS legal_entity_id
    FROM matter_links ml
    JOIN programs p
      ON p.tenant_id=ml.tenant_id
     AND p.id=ml.program_id
    GROUP BY ml.tenant_id, ml.matter_id
    HAVING count(DISTINCT p.legal_entity_id)=1
       AND count(*) FILTER (WHERE p.legal_entity_id IS NULL)=0
)
UPDATE matters m
SET legal_entity_id=u.legal_entity_id
FROM unambiguous u
WHERE u.tenant_id=m.tenant_id
  AND u.matter_id=m.id;

-- An unlinked legacy Matter can only be resolved automatically when its
-- tenant has exactly one current legal entity.
WITH single_entity AS (
    SELECT tenant_id, min(id::text)::uuid AS legal_entity_id
    FROM legal_entities
    WHERE valid_from<=clock_timestamp()
      AND (valid_until IS NULL OR clock_timestamp()<valid_until)
    GROUP BY tenant_id
    HAVING count(*)=1
)
UPDATE matters m
SET legal_entity_id=s.legal_entity_id
FROM single_entity s
WHERE s.tenant_id=m.tenant_id
  AND m.legal_entity_id IS NULL;

DO $$
DECLARE unresolved_count bigint;
DECLARE cross_entity_link_count bigint;
BEGIN
    SELECT count(*) INTO unresolved_count FROM matters WHERE legal_entity_id IS NULL;
    IF unresolved_count > 0 THEN
        RAISE EXCEPTION 'matter legal-entity migration unresolved rows: %', unresolved_count;
    END IF;
    SELECT count(*) INTO cross_entity_link_count
    FROM matter_links ml
    JOIN matters m ON m.tenant_id=ml.tenant_id AND m.id=ml.matter_id
    JOIN programs p ON p.tenant_id=ml.tenant_id AND p.id=ml.program_id
    WHERE m.legal_entity_id<>p.legal_entity_id OR p.legal_entity_id IS NULL;
    IF cross_entity_link_count > 0 THEN
        RAISE EXCEPTION 'matter legal-entity migration cross-entity links: %', cross_entity_link_count;
    END IF;
END $$;

-- legal_entity_id is immutable Matter identity. Add it to the creation event
-- so replay, history and derived follow-up use the same scope as current state.
UPDATE continuity_events ce
SET payload=ce.payload || jsonb_build_object('legal_entity_id',m.legal_entity_id::text)
FROM matters m
WHERE ce.tenant_id=m.tenant_id
  AND ce.aggregate_type='MATTER'
  AND ce.aggregate_id=m.id
  AND ce.event_type='MATTER_CREATED';

UPDATE outbox_events oe
SET payload=oe.payload || jsonb_build_object('legal_entity_id',m.legal_entity_id::text)
FROM matters m
WHERE oe.tenant_id=m.tenant_id
  AND oe.aggregate_type='MATTER'
  AND oe.aggregate_id=m.id
  AND oe.event_type='MATTER_CREATED'
  AND oe.published_at IS NULL;

ALTER TABLE matters ALTER COLUMN legal_entity_id SET NOT NULL;

ALTER TABLE matters
    ADD CONSTRAINT matters_legal_entity_tenant_fk
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id);

CREATE INDEX matters_entity_queue_idx
    ON matters(tenant_id,legal_entity_id,status,priority DESC,due_at,updated_at DESC,id);

CREATE INDEX matters_entity_summary_idx
    ON matters(tenant_id,legal_entity_id,priority DESC,updated_at DESC,id DESC);

CREATE INDEX matters_entity_open_summary_idx
    ON matters(tenant_id,legal_entity_id,priority DESC,updated_at DESC,id DESC)
    WHERE status NOT IN ('CLOSED','CANCELLED');

CREATE INDEX matters_entity_status_summary_idx
    ON matters(tenant_id,legal_entity_id,status,priority DESC,updated_at DESC,id DESC);

CREATE INDEX programs_entity_summary_idx
    ON programs(tenant_id,legal_entity_id,
        (CASE status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END),
        updated_at DESC,id DESC);

ALTER TABLE program_trigger_events
    DROP CONSTRAINT program_trigger_events_tenant_id_dedupe_key_key;
ALTER TABLE program_trigger_events
    ADD CONSTRAINT program_trigger_events_program_dedupe_key
    UNIQUE (tenant_id,program_id,dedupe_key);

DROP INDEX IF EXISTS matters_open_trigger_idx;
CREATE UNIQUE INDEX matters_open_trigger_idx
    ON matters(tenant_id,trigger_id)
    WHERE trigger_id IS NOT NULL AND status NOT IN ('CLOSED','CANCELLED');

CREATE OR REPLACE FUNCTION enforce_matter_program_legal_entity()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE matter_entity uuid;
DECLARE program_entity uuid;
BEGIN
    SELECT legal_entity_id INTO matter_entity
    FROM matters
    WHERE tenant_id=NEW.tenant_id AND id=NEW.matter_id;

    SELECT legal_entity_id INTO program_entity
    FROM programs
    WHERE tenant_id=NEW.tenant_id AND id=NEW.program_id;

    IF matter_entity IS NULL OR program_entity IS NULL OR matter_entity<>program_entity THEN
        RAISE EXCEPTION 'Matter and Program legal entities must match';
    END IF;
    RETURN NEW;
END $$;

CREATE CONSTRAINT TRIGGER matter_links_legal_entity_match
AFTER INSERT OR UPDATE OF tenant_id,matter_id,program_id ON matter_links
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW EXECUTE FUNCTION enforce_matter_program_legal_entity();

-- Legal entity is material object identity, not mutable operating state. This
-- also prevents an already-valid Matter/Program link from becoming cross-entity
-- through a later parent-row update.
CREATE OR REPLACE FUNCTION prevent_legal_entity_change()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.legal_entity_id IS DISTINCT FROM NEW.legal_entity_id THEN
        RAISE EXCEPTION USING
            ERRCODE='23514',
            MESSAGE=format('%s legal_entity_id is immutable after insert',TG_TABLE_NAME);
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER programs_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON programs
FOR EACH ROW EXECUTE FUNCTION prevent_legal_entity_change();

CREATE TRIGGER matters_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON matters
FOR EACH ROW EXECUTE FUNCTION prevent_legal_entity_change();

COMMIT;

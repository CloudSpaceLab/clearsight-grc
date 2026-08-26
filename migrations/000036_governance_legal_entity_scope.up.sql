BEGIN;

ALTER TABLE routing_policies ADD COLUMN legal_entity_id uuid;
ALTER TABLE routing_policy_versions ADD COLUMN legal_entity_id uuid;
ALTER TABLE delegations ADD COLUMN legal_entity_id uuid;

DO $$
DECLARE
    unresolved integer;
BEGIN
    SELECT count(*) INTO unresolved
    FROM delegations d
    WHERE EXISTS (
        SELECT 1 FROM jsonb_object_keys(COALESCE(d.scope,'{}'::jsonb)) key
        WHERE key NOT IN ('legal_entity_id','object_type','object_id','decision_type','min_materiality','max_materiality')
    ) OR (NULLIF(d.scope->>'object_id','') IS NOT NULL AND NULLIF(d.scope->>'object_type','') IS NULL)
      OR upper(COALESCE(d.scope->>'object_type','')) NOT IN ('','PROGRAM','MATTER')
      OR d.scope->>'object_id'='*'
      OR (d.scope ? 'min_materiality' AND (jsonb_typeof(d.scope->'min_materiality')<>'number' OR d.scope->>'min_materiality' !~ '^[0-5]$'))
      OR (d.scope ? 'max_materiality' AND (jsonb_typeof(d.scope->'max_materiality')<>'number' OR d.scope->>'max_materiality' !~ '^[0-5]$'))
      OR CASE
           WHEN COALESCE(d.scope->>'min_materiality','') ~ '^[0-5]$' AND COALESCE(d.scope->>'max_materiality','') ~ '^[0-5]$'
           THEN (d.scope->>'min_materiality')::integer>(d.scope->>'max_materiality')::integer
           ELSE false
         END;
    IF unresolved > 0 THEN
        RAISE EXCEPTION 'unresolved governance legal-entity rows: % delegation scopes contain unsupported keys', unresolved;
    END IF;
END $$;

DO $$
DECLARE
    unresolved integer;
BEGIN
    SELECT count(*) INTO unresolved FROM (
        SELECT rp.id
        FROM routing_policies rp
        JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id
        CROSS JOIN LATERAL jsonb_array_elements(COALESCE(rpv.definition->'rules','[]'::jsonb)) rule(value)
        WHERE NULLIF(rule.value->>'legal_entity_id','') IS NOT NULL
        GROUP BY rp.id,rp.tenant_id
        HAVING count(DISTINCT rule.value->>'legal_entity_id')>1
           OR bool_or((SELECT count(*) FROM legal_entities le
                       WHERE le.tenant_id=rp.tenant_id AND le.valid_until IS NULL
                         AND (le.id::text=rule.value->>'legal_entity_id' OR lower(le.code)=lower(rule.value->>'legal_entity_id')))<>1)
        UNION ALL
        SELECT d.id FROM delegations d
        WHERE NULLIF(d.scope->>'legal_entity_id','') IS NOT NULL
          AND (SELECT count(*) FROM legal_entities le
               WHERE le.tenant_id=d.tenant_id AND le.valid_until IS NULL
                 AND (le.id::text=d.scope->>'legal_entity_id' OR lower(le.code)=lower(d.scope->>'legal_entity_id')))<>1
    ) ambiguous;
    IF unresolved > 0 THEN
        RAISE EXCEPTION 'unresolved governance legal-entity rows: % ambiguous or mixed entity references', unresolved;
    END IF;
END $$;

WITH tenant_entities AS (
    SELECT tenant_id,(array_agg(id ORDER BY id))[1] AS entity_id,count(*) AS entity_count
    FROM legal_entities WHERE valid_until IS NULL GROUP BY tenant_id
), policy_refs AS (
    SELECT rp.id,rp.tenant_id,
           array_remove(array_agg(DISTINCT NULLIF(rule.value->>'legal_entity_id','')),NULL) AS refs
    FROM routing_policies rp
    JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(rpv.definition->'rules','[]'::jsonb)) rule(value)
    GROUP BY rp.id,rp.tenant_id
), resolved AS (
    SELECT pr.id,
           CASE
             WHEN cardinality(pr.refs)=0 AND te.entity_count=1 THEN te.entity_id
             WHEN cardinality(pr.refs)=1 THEN (
                 SELECT le.id FROM legal_entities le
                 WHERE le.tenant_id=pr.tenant_id AND le.valid_until IS NULL
                   AND (le.id::text=pr.refs[1] OR lower(le.code)=lower(pr.refs[1]))
                 LIMIT 1
             )
           END AS entity_id
    FROM policy_refs pr LEFT JOIN tenant_entities te ON te.tenant_id=pr.tenant_id
)
UPDATE routing_policies rp SET legal_entity_id=r.entity_id FROM resolved r WHERE r.id=rp.id;

WITH tenant_entities AS (
    SELECT tenant_id,(array_agg(id ORDER BY id))[1] AS entity_id,count(*) AS entity_count
    FROM legal_entities WHERE valid_until IS NULL GROUP BY tenant_id
), resolved AS (
    SELECT d.id,
           CASE
             WHEN NULLIF(d.scope->>'legal_entity_id','') IS NOT NULL THEN (
                 SELECT le.id FROM legal_entities le
                 WHERE le.tenant_id=d.tenant_id AND le.valid_until IS NULL
                   AND (le.id::text=d.scope->>'legal_entity_id' OR lower(le.code)=lower(d.scope->>'legal_entity_id'))
                 LIMIT 1
             )
             WHEN te.entity_count=1 THEN te.entity_id
           END AS entity_id
    FROM delegations d LEFT JOIN tenant_entities te ON te.tenant_id=d.tenant_id
)
UPDATE delegations d SET legal_entity_id=r.entity_id FROM resolved r WHERE r.id=d.id;

UPDATE routing_policy_versions rpv
SET legal_entity_id=rp.legal_entity_id,
    definition=jsonb_set(
        rpv.definition,'{rules}',
        COALESCE((SELECT jsonb_agg(jsonb_set(rule.value,'{legal_entity_id}',to_jsonb(rp.legal_entity_id::text),true) ORDER BY rule.ordinality)
                  FROM jsonb_array_elements(COALESCE(rpv.definition->'rules','[]'::jsonb)) WITH ORDINALITY rule(value,ordinality)),'[]'::jsonb),
        true)
FROM routing_policies rp WHERE rp.id=rpv.policy_id AND rp.legal_entity_id IS NOT NULL;

UPDATE delegations
SET scope=jsonb_set(COALESCE(scope,'{}'::jsonb),'{legal_entity_id}',to_jsonb(legal_entity_id::text),true)
WHERE legal_entity_id IS NOT NULL;

UPDATE outbox_events o
SET payload=jsonb_set(COALESCE(o.payload,'{}'::jsonb),'{legal_entity_id}',to_jsonb(rp.legal_entity_id::text),true)
FROM routing_policies rp
WHERE o.aggregate_type='ROUTING_POLICY' AND o.aggregate_id=rp.id AND o.published_at IS NULL;
UPDATE outbox_events o
SET payload=jsonb_set(COALESCE(o.payload,'{}'::jsonb),'{legal_entity_id}',to_jsonb(d.legal_entity_id::text),true)
FROM delegations d
WHERE o.aggregate_type='DELEGATION' AND o.aggregate_id=d.id AND o.published_at IS NULL;

DO $$
DECLARE
    unresolved integer;
BEGIN
    SELECT count(*) INTO unresolved FROM (
        SELECT id FROM routing_policies WHERE legal_entity_id IS NULL
        UNION ALL SELECT id FROM routing_policy_versions WHERE legal_entity_id IS NULL
        UNION ALL SELECT id FROM delegations WHERE legal_entity_id IS NULL
    ) rows;
    IF unresolved > 0 THEN
        RAISE EXCEPTION 'unresolved governance legal-entity rows: %; migration aborted without operational nullable state', unresolved;
    END IF;
END $$;

ALTER TABLE routing_policies DROP CONSTRAINT routing_policies_tenant_id_code_key;
ALTER TABLE routing_policies ADD CONSTRAINT routing_policies_id_legal_entity_key UNIQUE(id,legal_entity_id);

ALTER TABLE routing_policies ALTER COLUMN legal_entity_id SET NOT NULL;
ALTER TABLE routing_policy_versions ALTER COLUMN legal_entity_id SET NOT NULL;
ALTER TABLE delegations ALTER COLUMN legal_entity_id SET NOT NULL;

ALTER TABLE routing_policies ADD CONSTRAINT routing_policies_tenant_legal_entity_fk
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id);
ALTER TABLE delegations ADD CONSTRAINT delegations_tenant_legal_entity_fk
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id);
ALTER TABLE routing_policy_versions ADD CONSTRAINT routing_policy_versions_policy_entity_fk
    FOREIGN KEY(policy_id,legal_entity_id) REFERENCES routing_policies(id,legal_entity_id);

CREATE UNIQUE INDEX routing_policies_entity_code_uidx ON routing_policies(tenant_id,legal_entity_id,code);
CREATE INDEX routing_policies_entity_inventory_idx ON routing_policies(tenant_id,legal_entity_id,code,status,id);
CREATE INDEX delegations_entity_inventory_idx ON delegations(tenant_id,legal_entity_id,status,created_at DESC,id);
CREATE INDEX delegations_entity_resolution_idx ON delegations(tenant_id,legal_entity_id,from_principal_id,responsibility,starts_at,ends_at)
    WHERE status IN ('APPROVED','ACTIVE');

CREATE OR REPLACE FUNCTION prevent_governance_legal_entity_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id THEN
        RAISE EXCEPTION 'governance legal_entity_id is immutable';
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER routing_policies_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON routing_policies FOR EACH ROW EXECUTE FUNCTION prevent_governance_legal_entity_change();
CREATE TRIGGER routing_policy_versions_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON routing_policy_versions FOR EACH ROW EXECUTE FUNCTION prevent_governance_legal_entity_change();
CREATE TRIGGER delegations_legal_entity_immutable
BEFORE UPDATE OF legal_entity_id ON delegations FOR EACH ROW EXECUTE FUNCTION prevent_governance_legal_entity_change();

CREATE OR REPLACE FUNCTION validate_governance_entity_payload()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_TABLE_NAME='routing_policy_versions' THEN
        IF EXISTS (
            SELECT 1 FROM jsonb_array_elements(COALESCE(NEW.definition->'rules','[]'::jsonb)) rule(value)
            WHERE COALESCE(rule.value->>'legal_entity_id','')<>NEW.legal_entity_id::text
        ) THEN
            RAISE EXCEPTION 'routing policy rules must match canonical legal_entity_id';
        END IF;
    ELSIF TG_TABLE_NAME='delegations' THEN
        IF COALESCE(NEW.scope->>'legal_entity_id','')<>NEW.legal_entity_id::text
           OR (NEW.scope-ARRAY['legal_entity_id','object_type','object_id','decision_type','min_materiality','max_materiality']::text[])<>'{}'::jsonb
           OR (NULLIF(NEW.scope->>'object_id','') IS NOT NULL AND NULLIF(NEW.scope->>'object_type','') IS NULL)
           OR upper(COALESCE(NEW.scope->>'object_type','')) NOT IN ('','PROGRAM','MATTER')
           OR NEW.scope->>'object_id'='*'
           OR (NEW.scope ? 'min_materiality' AND (jsonb_typeof(NEW.scope->'min_materiality')<>'number' OR NEW.scope->>'min_materiality' !~ '^[0-5]$'))
           OR (NEW.scope ? 'max_materiality' AND (jsonb_typeof(NEW.scope->'max_materiality')<>'number' OR NEW.scope->>'max_materiality' !~ '^[0-5]$'))
           OR CASE
                WHEN COALESCE(NEW.scope->>'min_materiality','') ~ '^[0-5]$' AND COALESCE(NEW.scope->>'max_materiality','') ~ '^[0-5]$'
                THEN (NEW.scope->>'min_materiality')::integer>(NEW.scope->>'max_materiality')::integer
                ELSE false
              END
        THEN
            RAISE EXCEPTION 'delegation scope is not canonical or contains unsupported fields';
        END IF;
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER routing_policy_versions_entity_payload
BEFORE INSERT OR UPDATE ON routing_policy_versions
FOR EACH ROW EXECUTE FUNCTION validate_governance_entity_payload();
CREATE TRIGGER delegations_entity_payload
BEFORE INSERT OR UPDATE ON delegations
FOR EACH ROW EXECUTE FUNCTION validate_governance_entity_payload();

COMMIT;

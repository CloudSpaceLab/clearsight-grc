BEGIN;

CREATE TABLE effective_authority_routes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_policy_id uuid NOT NULL REFERENCES routing_policies(id) ON DELETE CASCADE,
    source_rule_id text NOT NULL,
    policy_version text NOT NULL,
    legal_entity_ref text NOT NULL DEFAULT '*',
    object_type text NOT NULL DEFAULT '*',
    object_id text NOT NULL DEFAULT '*',
    responsibility text NOT NULL,
    decision_type text NOT NULL DEFAULT '',
    min_materiality integer NOT NULL DEFAULT 0 CHECK (min_materiality BETWEEN 0 AND 5),
    priority integer NOT NULL DEFAULT 0,
    selector_kind text NOT NULL,
    selector_ref text NOT NULL,
    resolution_strategy text NOT NULL DEFAULT 'DIRECT' CHECK (resolution_strategy IN ('DIRECT','CANDIDATE_SET')),
    valid_from timestamptz NOT NULL,
    valid_until timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (source_policy_id, policy_version, source_rule_id),
    CHECK (valid_until IS NULL OR valid_from < valid_until)
);

CREATE INDEX effective_authority_resolution_idx
    ON effective_authority_routes (
        tenant_id,
        responsibility,
        object_type,
        object_id,
        decision_type,
        priority DESC,
        min_materiality
    );
CREATE INDEX effective_authority_entity_idx
    ON effective_authority_routes (tenant_id, legal_entity_ref, priority DESC);

CREATE OR REPLACE FUNCTION refresh_effective_authority_routes(target_tenant uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM effective_authority_routes WHERE tenant_id = target_tenant;

    INSERT INTO effective_authority_routes(
        tenant_id, source_policy_id, source_rule_id, policy_version,
        legal_entity_ref, object_type, object_id, responsibility, decision_type,
        min_materiality, priority, selector_kind, selector_ref,
        resolution_strategy, valid_from, valid_until
    )
    SELECT
        rp.tenant_id,
        rp.id,
        rule.value->>'id',
        rp.code || ':v' || rpv.version::text,
        COALESCE(NULLIF(rule.value->>'legal_entity_id',''), '*'),
        COALESCE(NULLIF(rule.value->>'object_type',''), '*'),
        COALESCE(NULLIF(rule.value->>'object_id',''), '*'),
        upper(rule.value->>'responsibility'),
        COALESCE(rule.value->>'decision_type',''),
        COALESCE((rule.value->>'min_materiality')::integer, 0),
        COALESCE((rule.value->>'priority')::integer, 0),
        upper(rule.value->'selector'->>'kind'),
        rule.value->'selector'->>'ref',
        CASE
            WHEN upper(rule.value->'selector'->>'kind') IN ('ROLE','TEAM','QUEUE','COMMITTEE')
                THEN 'CANDIDATE_SET'
            ELSE 'DIRECT'
        END,
        COALESCE(rpv.effective_from, rp.approved_at, rp.updated_at, rp.created_at),
        rpv.effective_until
    FROM routing_policies rp
    JOIN routing_policy_versions rpv
      ON rpv.policy_id = rp.id AND rpv.version = rp.current_version
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(rpv.definition->'rules','[]'::jsonb)) AS rule(value)
    WHERE rp.tenant_id = target_tenant
      AND rp.status = 'ACTIVE'
      AND (rpv.effective_from IS NULL OR rpv.effective_from <= clock_timestamp())
      AND (rpv.effective_until IS NULL OR clock_timestamp() < rpv.effective_until)
      AND COALESCE(rule.value->>'id','') <> ''
      AND COALESCE(rule.value->>'responsibility','') <> ''
      AND COALESCE(rule.value->'selector'->>'kind','') <> ''
      AND COALESCE(rule.value->'selector'->>'ref','') <> '';
END;
$$;

CREATE OR REPLACE FUNCTION refresh_effective_authority_routes_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    tenant uuid;
BEGIN
    IF TG_TABLE_NAME = 'routing_policies' THEN
        tenant := COALESCE(NEW.tenant_id, OLD.tenant_id);
    ELSE
        SELECT tenant_id INTO tenant
        FROM routing_policies
        WHERE id = COALESCE(NEW.policy_id, OLD.policy_id);
    END IF;
    IF tenant IS NOT NULL THEN
        PERFORM refresh_effective_authority_routes(tenant);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

CREATE TRIGGER routing_policies_effective_routes_trg
AFTER INSERT OR UPDATE OR DELETE ON routing_policies
FOR EACH ROW EXECUTE FUNCTION refresh_effective_authority_routes_trigger();

CREATE TRIGGER routing_policy_versions_effective_routes_trg
AFTER INSERT OR UPDATE OR DELETE ON routing_policy_versions
FOR EACH ROW EXECUTE FUNCTION refresh_effective_authority_routes_trigger();

DO $$
DECLARE
    tenant uuid;
BEGIN
    FOR tenant IN SELECT DISTINCT tenant_id FROM routing_policies LOOP
        PERFORM refresh_effective_authority_routes(tenant);
    END LOOP;
END;
$$;

COMMIT;

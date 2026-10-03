BEGIN;

CREATE TABLE organization_scopes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    parent_scope_id uuid,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 240),
    kind text NOT NULL CHECK (kind IN ('ORGANIZATION_UNIT','BRANCH','DEPARTMENT','FUNCTION','BUSINESS_UNIT','CRITICAL_SERVICE')),
    department_path text[] NOT NULL DEFAULT '{}'::text[],
    origin text NOT NULL CHECK (origin IN ('LEGACY_DEPARTMENT_PATH','MANAGED')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (tenant_id, legal_entity_id, id),
    FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    FOREIGN KEY (tenant_id, legal_entity_id, parent_scope_id)
        REFERENCES organization_scopes(tenant_id, legal_entity_id, id),
    CHECK (parent_scope_id IS NULL OR parent_scope_id <> id),
    CHECK (cardinality(department_path) BETWEEN 1 AND 12),
    CHECK (array_position(department_path, '') IS NULL),
    CHECK (valid_until IS NULL OR valid_from < valid_until),
    CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX organization_scopes_active_code_idx
    ON organization_scopes(tenant_id, legal_entity_id, code)
    WHERE status='ACTIVE' AND valid_until IS NULL;
CREATE UNIQUE INDEX organization_scopes_active_path_idx
    ON organization_scopes(tenant_id, legal_entity_id, department_path)
    WHERE status='ACTIVE' AND valid_until IS NULL;
CREATE INDEX organization_scopes_parent_idx
    ON organization_scopes(tenant_id, legal_entity_id, parent_scope_id, id)
    WHERE status='ACTIVE' AND valid_until IS NULL;
CREATE INDEX organization_scopes_path_idx
    ON organization_scopes USING gin (department_path)
    WHERE status='ACTIVE' AND valid_until IS NULL;

ALTER TABLE org_positions
    ADD COLUMN organization_scope_id uuid;
ALTER TABLE org_positions
    ADD CONSTRAINT org_positions_organization_scope_fk
    FOREIGN KEY (tenant_id, legal_entity_id, organization_scope_id)
    REFERENCES organization_scopes(tenant_id, legal_entity_id, id);
CREATE INDEX org_positions_organization_scope_idx
    ON org_positions(tenant_id, legal_entity_id, organization_scope_id)
    WHERE valid_until IS NULL AND organization_scope_id IS NOT NULL;

ALTER TABLE directory_group_role_bindings
    ADD COLUMN organization_scope_id uuid;
ALTER TABLE directory_group_role_bindings
    ADD CONSTRAINT directory_group_role_bindings_organization_scope_fk
    FOREIGN KEY (tenant_id, legal_entity_id, organization_scope_id)
    REFERENCES organization_scopes(tenant_id, legal_entity_id, id);
CREATE INDEX directory_group_role_bindings_organization_scope_idx
    ON directory_group_role_bindings(tenant_id, legal_entity_id, organization_scope_id)
    WHERE valid_until IS NULL AND organization_scope_id IS NOT NULL;

CREATE FUNCTION organization_scope_for_department_path(
    p_tenant_id uuid,
    p_legal_entity_id uuid,
    p_department_path text[],
    p_valid_from timestamptz
) RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    normalized_path text[];
    prefix_path text[];
    current_scope_id uuid;
    current_parent_id uuid;
    depth integer;
BEGIN
    IF p_legal_entity_id IS NULL OR cardinality(p_department_path)=0 THEN
        RETURN NULL;
    END IF;
    IF cardinality(p_department_path)>12 THEN
        RAISE EXCEPTION 'department path supports at most 12 levels';
    END IF;

    normalized_path := ARRAY(
        SELECT upper(btrim(segment.value))
        FROM unnest(p_department_path) WITH ORDINALITY AS segment(value, ordinal)
        ORDER BY segment.ordinal
    );
    IF EXISTS (
        SELECT 1 FROM unnest(normalized_path) AS segment(value)
        WHERE segment.value='' OR char_length(segment.value)>80
    ) THEN
        RAISE EXCEPTION 'department path contains an invalid segment';
    END IF;

    FOR depth IN 1..cardinality(normalized_path) LOOP
        prefix_path := normalized_path[1:depth];

        INSERT INTO organization_scopes(
            tenant_id, legal_entity_id, parent_scope_id, code, name, kind,
            department_path, origin, status, valid_from
        )
        VALUES(
            p_tenant_id,
            p_legal_entity_id,
            current_parent_id,
            'LEGACY-' || upper(substr(md5(p_legal_entity_id::text || ':' || array_to_string(prefix_path, E'\x1f')), 1, 20)),
            prefix_path[depth],
            'ORGANIZATION_UNIT',
            prefix_path,
            'LEGACY_DEPARTMENT_PATH',
            'ACTIVE',
            COALESCE(p_valid_from, clock_timestamp())
        )
        ON CONFLICT DO NOTHING;

        SELECT scope.id INTO current_scope_id
        FROM organization_scopes scope
        WHERE scope.tenant_id=p_tenant_id
          AND scope.legal_entity_id=p_legal_entity_id
          AND scope.department_path=prefix_path
          AND scope.status='ACTIVE'
          AND scope.valid_until IS NULL
        LIMIT 1;

        IF current_scope_id IS NULL THEN
            RAISE EXCEPTION 'organization scope could not be reconciled';
        END IF;

        UPDATE organization_scopes
        SET parent_scope_id=current_parent_id
        WHERE id=current_scope_id
          AND organization_scopes.parent_scope_id IS DISTINCT FROM current_parent_id;

        current_parent_id := current_scope_id;
    END LOOP;

    RETURN current_scope_id;
END;
$$;

CREATE FUNCTION bind_legacy_organization_scope() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.legal_entity_id IS NULL THEN
        NEW.organization_scope_id := NULL;
        RETURN NEW;
    END IF;
    IF cardinality(NEW.department_path)=0 THEN
        RETURN NEW;
    END IF;

    IF NEW.organization_scope_id IS NULL
       OR (TG_OP='UPDATE'
           AND NEW.department_path IS DISTINCT FROM OLD.department_path
           AND NEW.organization_scope_id IS NOT DISTINCT FROM OLD.organization_scope_id) THEN
        NEW.organization_scope_id := organization_scope_for_department_path(
            NEW.tenant_id,
            NEW.legal_entity_id,
            NEW.department_path,
            NEW.valid_from
        );
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION reconcile_organization_scopes() RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    inserted_count integer := 0;
BEGIN
    WITH raw_paths AS (
        SELECT op.tenant_id, op.legal_entity_id,
               ARRAY(
                   SELECT upper(btrim(segment.value))
                   FROM unnest(op.department_path) WITH ORDINALITY AS segment(value, ordinal)
                   ORDER BY segment.ordinal
               ) AS department_path,
               op.valid_from
        FROM org_positions op
        WHERE op.legal_entity_id IS NOT NULL
          AND op.valid_until IS NULL
          AND cardinality(op.department_path) BETWEEN 1 AND 12
          AND NOT EXISTS (
              SELECT 1 FROM unnest(op.department_path) AS segment(value)
              WHERE btrim(segment.value)='' OR char_length(btrim(segment.value))>80
          )
        UNION ALL
        SELECT binding.tenant_id, binding.legal_entity_id,
               ARRAY(
                   SELECT upper(btrim(segment.value))
                   FROM unnest(binding.department_path) WITH ORDINALITY AS segment(value, ordinal)
                   ORDER BY segment.ordinal
               ) AS department_path,
               binding.valid_from
        FROM directory_group_role_bindings binding
        WHERE binding.valid_until IS NULL
          AND cardinality(binding.department_path) BETWEEN 1 AND 12
          AND NOT EXISTS (
              SELECT 1 FROM unnest(binding.department_path) AS segment(value)
              WHERE btrim(segment.value)='' OR char_length(btrim(segment.value))>80
          )
    ), prefixes AS (
        SELECT raw.tenant_id, raw.legal_entity_id,
               raw.department_path[1:depth.value] AS department_path,
               min(raw.valid_from) AS valid_from
        FROM raw_paths raw
        CROSS JOIN LATERAL generate_series(1, cardinality(raw.department_path)) AS depth(value)
        GROUP BY raw.tenant_id, raw.legal_entity_id, raw.department_path[1:depth.value]
    )
    INSERT INTO organization_scopes(
        tenant_id, legal_entity_id, code, name, kind, department_path, origin, status, valid_from
    )
    SELECT prefix.tenant_id,
           prefix.legal_entity_id,
           'LEGACY-' || upper(substr(md5(prefix.legal_entity_id::text || ':' || array_to_string(prefix.department_path, E'\x1f')), 1, 20)),
           prefix.department_path[cardinality(prefix.department_path)],
           'ORGANIZATION_UNIT',
           prefix.department_path,
           'LEGACY_DEPARTMENT_PATH',
           'ACTIVE',
           prefix.valid_from
    FROM prefixes prefix
    ON CONFLICT DO NOTHING;

    GET DIAGNOSTICS inserted_count = ROW_COUNT;

    UPDATE organization_scopes child
    SET parent_scope_id=parent.id
    FROM organization_scopes parent
    WHERE child.origin='LEGACY_DEPARTMENT_PATH'
      AND child.status='ACTIVE' AND child.valid_until IS NULL
      AND parent.status='ACTIVE' AND parent.valid_until IS NULL
      AND parent.tenant_id=child.tenant_id
      AND parent.legal_entity_id=child.legal_entity_id
      AND cardinality(child.department_path)>1
      AND parent.department_path=child.department_path[1:cardinality(child.department_path)-1]
      AND child.parent_scope_id IS DISTINCT FROM parent.id;

    UPDATE org_positions position
    SET organization_scope_id=scope.id
    FROM organization_scopes scope
    WHERE position.legal_entity_id IS NOT NULL
      AND cardinality(position.department_path)>0
      AND scope.tenant_id=position.tenant_id
      AND scope.legal_entity_id=position.legal_entity_id
      AND scope.status='ACTIVE' AND scope.valid_until IS NULL
      AND scope.department_path=ARRAY(
          SELECT upper(btrim(segment.value))
          FROM unnest(position.department_path) WITH ORDINALITY AS segment(value, ordinal)
          ORDER BY segment.ordinal
      )
      AND position.organization_scope_id IS DISTINCT FROM scope.id;

    UPDATE directory_group_role_bindings binding
    SET organization_scope_id=scope.id
    FROM organization_scopes scope
    WHERE cardinality(binding.department_path)>0
      AND scope.tenant_id=binding.tenant_id
      AND scope.legal_entity_id=binding.legal_entity_id
      AND scope.status='ACTIVE' AND scope.valid_until IS NULL
      AND scope.department_path=ARRAY(
          SELECT upper(btrim(segment.value))
          FROM unnest(binding.department_path) WITH ORDINALITY AS segment(value, ordinal)
          ORDER BY segment.ordinal
      )
      AND binding.organization_scope_id IS DISTINCT FROM scope.id;

    RETURN inserted_count;
END;
$$;

SELECT reconcile_organization_scopes();

CREATE TRIGGER org_positions_bind_organization_scope
    BEFORE INSERT OR UPDATE OF legal_entity_id,department_path,organization_scope_id
    ON org_positions
    FOR EACH ROW EXECUTE FUNCTION bind_legacy_organization_scope();

CREATE TRIGGER directory_group_role_bindings_bind_organization_scope
    BEFORE INSERT OR UPDATE OF legal_entity_id,department_path,organization_scope_id
    ON directory_group_role_bindings
    FOR EACH ROW EXECUTE FUNCTION bind_legacy_organization_scope();

COMMIT;

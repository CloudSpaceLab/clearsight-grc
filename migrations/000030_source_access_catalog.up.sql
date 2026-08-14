BEGIN;

CREATE FUNCTION source_catalog_text_array_valid(value jsonb, minimum_count integer, maximum_count integer, allowed_values text[] DEFAULT NULL)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    total_count integer;
    distinct_count integer;
    has_invalid boolean;
BEGIN
    IF jsonb_typeof(value) IS DISTINCT FROM 'array' THEN
        RETURN false;
    END IF;
    total_count := jsonb_array_length(value);
    IF total_count < minimum_count OR total_count > maximum_count THEN
        RETURN false;
    END IF;
    SELECT count(DISTINCT item #>> '{}'),
           bool_or(
               jsonb_typeof(item) <> 'string'
               OR (item #>> '{}') = ''
               OR octet_length(item #>> '{}') > 256
               OR (item #>> '{}') ~ '[[:cntrl:]]'
               OR (allowed_values IS NOT NULL AND NOT ((item #>> '{}') = ANY(allowed_values)))
           )
      INTO distinct_count,has_invalid
      FROM jsonb_array_elements(value) AS items(item);
    RETURN distinct_count = total_count AND NOT COALESCE(has_invalid,false);
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$$;

CREATE FUNCTION source_catalog_native_schema_valid(value jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    item jsonb;
    field_name text;
    native_type text;
    seen_names text[] := ARRAY[]::text[];
BEGIN
    IF jsonb_typeof(value) IS DISTINCT FROM 'array' OR jsonb_array_length(value) > 512 THEN
        RETURN false;
    END IF;
    FOR item IN SELECT element FROM jsonb_array_elements(value) AS elements(element)
    LOOP
        IF jsonb_typeof(item) <> 'object' THEN
            RETURN false;
        END IF;
        field_name := item->>'name';
        native_type := item->>'native_type';
        IF field_name IS NULL OR field_name = '' OR octet_length(field_name) > 256 OR field_name ~ '[[:cntrl:]]' THEN
            RETURN false;
        END IF;
        IF native_type IS NULL OR btrim(native_type) = '' OR native_type <> btrim(native_type) OR octet_length(native_type) > 256 OR native_type ~ '[[:cntrl:]]' THEN
            RETURN false;
        END IF;
        IF item ? 'nullable' AND jsonb_typeof(item->'nullable') <> 'boolean' THEN
            RETURN false;
        END IF;
        IF field_name = ANY(seen_names) THEN
            RETURN false;
        END IF;
        seen_names := array_append(seen_names,field_name);
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$$;

CREATE TABLE source_connections (
    revision_id uuid PRIMARY KEY DEFAULT uuidv7(),
    connection_id uuid NOT NULL DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id uuid NOT NULL,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 512),
    adapter_kind text NOT NULL CHECK (adapter_kind=btrim(adapter_kind) AND char_length(adapter_kind) BETWEEN 1 AND 128),
    adapter_version text NOT NULL CHECK (adapter_version=btrim(adapter_version) AND char_length(adapter_version) BETWEEN 1 AND 128),
    secret_ref text NOT NULL DEFAULT '' CHECK (secret_ref=btrim(secret_ref) AND char_length(secret_ref) <= 256),
    definition jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(definition)='object' AND octet_length(definition::text) <= 32768),
    declared_capabilities jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (source_catalog_text_array_valid(declared_capabilities,0,5,ARRAY['INSPECT','PAGE','LOOKUP','AGGREGATE','CHANGES'])),
    verified_capabilities jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (source_catalog_text_array_valid(verified_capabilities,0,5,ARRAY['INSPECT','PAGE','LOOKUP','AGGREGATE','CHANGES']) AND verified_capabilities <@ declared_capabilities),
    owner_principal_id uuid,
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','PAUSED','REJECTED','RETIRED')),
    is_current boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE (connection_id,version),
    UNIQUE (connection_id,tenant_id,source_id,version),
    FOREIGN KEY (source_id,tenant_id) REFERENCES evidence_sources(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (
        (status IN ('ACTIVE','PAUSED') AND is_current AND effective_from IS NOT NULL AND effective_until IS NULL)
        OR (status='RETIRED' AND NOT is_current AND effective_from IS NOT NULL AND effective_until IS NOT NULL AND effective_until >= effective_from)
        OR (status IN ('DRAFT','PENDING_APPROVAL','REJECTED') AND NOT is_current AND effective_from IS NULL AND effective_until IS NULL)
    )
);
ALTER TABLE source_connections ADD CONSTRAINT source_connections_reference_shape_ck CHECK (adapter_kind<>'REFERENCE' OR (adapter_version='reference-v1' AND secret_ref='' AND declared_capabilities='[]'::jsonb AND verified_capabilities='[]'::jsonb AND jsonb_typeof(definition->'endpoint')='string' AND NULLIF(btrim(definition->>'endpoint'),'') IS NOT NULL AND NOT ((definition->>'endpoint') ~ '[[:cntrl:]]')));
CREATE UNIQUE INDEX source_connections_current_id_idx ON source_connections(connection_id) WHERE is_current;
CREATE UNIQUE INDEX source_connections_current_code_idx ON source_connections(tenant_id,source_id,code) WHERE is_current;
CREATE INDEX source_connections_source_idx ON source_connections(tenant_id,source_id,is_current,code,connection_id);
CREATE INDEX source_connections_history_idx ON source_connections(connection_id,version DESC);

CREATE TABLE source_views (
    revision_id uuid PRIMARY KEY DEFAULT uuidv7(),
    view_id uuid NOT NULL DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    connection_version bigint NOT NULL CHECK (connection_version > 0),
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 512),
    definition jsonb NOT NULL CHECK (jsonb_typeof(definition)='object' AND octet_length(definition::text) <= 32768),
    output_kind text NOT NULL CHECK (output_kind='RECORDS'),
    stable_keys jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (source_catalog_text_array_valid(stable_keys,0,4,NULL)),
    native_schema jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (source_catalog_native_schema_valid(native_schema)),
    schema_fingerprint text NOT NULL DEFAULT '' CHECK (schema_fingerprint='' OR schema_fingerprint ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','PAUSED','REJECTED','RETIRED')),
    is_current boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE (view_id,version),
    UNIQUE (view_id,tenant_id,source_id,version),
    FOREIGN KEY (connection_id,tenant_id,source_id,connection_version) REFERENCES source_connections(connection_id,tenant_id,source_id,version),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (NOT is_current OR (schema_fingerprint<>'' AND jsonb_array_length(native_schema)>0)),
    CHECK (
        (status IN ('ACTIVE','PAUSED') AND is_current AND effective_from IS NOT NULL AND effective_until IS NULL)
        OR (status='RETIRED' AND NOT is_current AND effective_from IS NOT NULL AND effective_until IS NOT NULL AND effective_until >= effective_from)
        OR (status IN ('DRAFT','PENDING_APPROVAL','REJECTED') AND NOT is_current AND effective_from IS NULL AND effective_until IS NULL)
    )
);
CREATE UNIQUE INDEX source_views_current_id_idx ON source_views(view_id) WHERE is_current;
CREATE UNIQUE INDEX source_views_current_code_idx ON source_views(tenant_id,connection_id,code) WHERE is_current;
CREATE INDEX source_views_connection_idx ON source_views(tenant_id,connection_id,is_current,code,view_id);
CREATE INDEX source_views_history_idx ON source_views(view_id,version DESC);

CREATE TABLE source_bindings (
    revision_id uuid PRIMARY KEY DEFAULT uuidv7(),
    binding_id uuid NOT NULL DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id uuid NOT NULL,
    view_id uuid NOT NULL,
    view_version bigint NOT NULL CHECK (view_version > 0),
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 512),
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 256),
    operations jsonb NOT NULL CHECK (source_catalog_text_array_valid(operations,1,5,ARRAY['INSPECT','PAGE','LOOKUP','AGGREGATE','CHANGES'])),
    selected_fields jsonb NOT NULL CHECK (source_catalog_text_array_valid(selected_fields,1,512,NULL)),
    key_fields jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (source_catalog_text_array_valid(key_fields,0,4,NULL)),
    page_rows integer NOT NULL CHECK (page_rows BETWEEN 1 AND 1000),
    response_bytes bigint NOT NULL CHECK (response_bytes BETWEEN 1 AND 4194304),
    lookup_values integer NOT NULL CHECK (lookup_values BETWEEN 1 AND 100),
    timeout_ms integer NOT NULL CHECK (timeout_ms BETWEEN 1 AND 30000),
    mapping jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(mapping)='object' AND octet_length(mapping::text) <= 32768),
    parameter_schema jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(parameter_schema)='object' AND octet_length(parameter_schema::text) <= 32768),
    output_schema jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output_schema)='object' AND octet_length(output_schema::text) <= 32768),
    required_freshness_minutes integer NOT NULL DEFAULT 0 CHECK (required_freshness_minutes BETWEEN 0 AND 525600),
    completeness text NOT NULL CHECK (completeness IN ('ALLOW_PARTIAL','REQUIRE_COMPLETE')),
    sensitivity_handling jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(sensitivity_handling)='object' AND octet_length(sensitivity_handling::text) <= 32768),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','PAUSED','REJECTED','RETIRED')),
    is_current boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE (binding_id,version),
    UNIQUE (binding_id,tenant_id,source_id,version),
    FOREIGN KEY (view_id,tenant_id,source_id,view_version) REFERENCES source_views(view_id,tenant_id,source_id,version),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (
        (status IN ('ACTIVE','PAUSED') AND is_current AND effective_from IS NOT NULL AND effective_until IS NULL)
        OR (status='RETIRED' AND NOT is_current AND effective_from IS NOT NULL AND effective_until IS NOT NULL AND effective_until >= effective_from)
        OR (status IN ('DRAFT','PENDING_APPROVAL','REJECTED') AND NOT is_current AND effective_from IS NULL AND effective_until IS NULL)
    )
);
CREATE UNIQUE INDEX source_bindings_current_id_idx ON source_bindings(binding_id) WHERE is_current;
CREATE UNIQUE INDEX source_bindings_current_code_idx ON source_bindings(tenant_id,view_id,code) WHERE is_current;
CREATE INDEX source_bindings_view_idx ON source_bindings(tenant_id,view_id,is_current,code,binding_id);
CREATE INDEX source_bindings_history_idx ON source_bindings(binding_id,version DESC);

CREATE FUNCTION source_connection_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP='UPDATE' THEN
        IF ROW(OLD.revision_id,OLD.connection_id,OLD.tenant_id,OLD.source_id,OLD.code,OLD.name,OLD.adapter_kind,OLD.adapter_version,OLD.secret_ref,OLD.definition,OLD.declared_capabilities,OLD.verified_capabilities,OLD.owner_principal_id,OLD.version,OLD.created_by,OLD.created_at)
           IS DISTINCT FROM
           ROW(NEW.revision_id,NEW.connection_id,NEW.tenant_id,NEW.source_id,NEW.code,NEW.name,NEW.adapter_kind,NEW.adapter_version,NEW.secret_ref,NEW.definition,NEW.declared_capabilities,NEW.verified_capabilities,NEW.owner_principal_id,NEW.version,NEW.created_by,NEW.created_at) THEN
            RAISE EXCEPTION 'source connection revision definition is immutable' USING ERRCODE='23514';
        END IF;
        IF NEW.updated_at < OLD.updated_at THEN
            RAISE EXCEPTION 'source connection updated_at cannot move backwards' USING ERRCODE='23514';
        END IF;
        IF OLD.is_current AND (NOT NEW.is_current OR NEW.status NOT IN ('ACTIVE','PAUSED')) AND EXISTS (
            SELECT 1 FROM source_views child
             WHERE child.tenant_id=OLD.tenant_id AND child.source_id=OLD.source_id
               AND child.connection_id=OLD.connection_id AND child.connection_version=OLD.version
               AND child.is_current
        ) THEN
            RAISE EXCEPTION 'current source connection has current views' USING ERRCODE='23514';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM source_connections existing
         WHERE existing.connection_id=NEW.connection_id
           AND existing.revision_id<>NEW.revision_id
           AND (existing.tenant_id<>NEW.tenant_id OR existing.source_id<>NEW.source_id)
    ) THEN
        RAISE EXCEPTION 'source connection identity cannot change tenant or source' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER source_connection_revision_guard_trigger BEFORE INSERT OR UPDATE ON source_connections FOR EACH ROW EXECUTE FUNCTION source_connection_revision_guard();

CREATE FUNCTION source_view_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    parent_current boolean;
    parent_kind text;
BEGIN
    IF TG_OP='UPDATE' THEN
        IF ROW(OLD.revision_id,OLD.view_id,OLD.tenant_id,OLD.source_id,OLD.connection_id,OLD.connection_version,OLD.code,OLD.name,OLD.definition,OLD.output_kind,OLD.stable_keys,OLD.native_schema,OLD.schema_fingerprint,OLD.version,OLD.created_by,OLD.created_at)
           IS DISTINCT FROM
           ROW(NEW.revision_id,NEW.view_id,NEW.tenant_id,NEW.source_id,NEW.connection_id,NEW.connection_version,NEW.code,NEW.name,NEW.definition,NEW.output_kind,NEW.stable_keys,NEW.native_schema,NEW.schema_fingerprint,NEW.version,NEW.created_by,NEW.created_at) THEN
            RAISE EXCEPTION 'source view revision definition is immutable' USING ERRCODE='23514';
        END IF;
        IF NEW.updated_at < OLD.updated_at THEN
            RAISE EXCEPTION 'source view updated_at cannot move backwards' USING ERRCODE='23514';
        END IF;
        IF OLD.is_current AND (NOT NEW.is_current OR NEW.status NOT IN ('ACTIVE','PAUSED')) AND EXISTS (
            SELECT 1 FROM source_bindings child
             WHERE child.tenant_id=OLD.tenant_id AND child.source_id=OLD.source_id
               AND child.view_id=OLD.view_id AND child.view_version=OLD.version
               AND child.is_current
        ) THEN
            RAISE EXCEPTION 'current source view has current bindings' USING ERRCODE='23514';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM source_views existing
         WHERE existing.view_id=NEW.view_id
           AND existing.revision_id<>NEW.revision_id
           AND (existing.tenant_id<>NEW.tenant_id OR existing.source_id<>NEW.source_id OR existing.connection_id<>NEW.connection_id)
    ) THEN
        RAISE EXCEPTION 'source view identity cannot change tenant, source or connection' USING ERRCODE='23514';
    END IF;
    SELECT is_current,adapter_kind INTO parent_current,parent_kind
      FROM source_connections
     WHERE connection_id=NEW.connection_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.connection_version;
    IF NEW.is_current AND NOT COALESCE(parent_current,false) THEN
        RAISE EXCEPTION 'current source view requires its current connection revision' USING ERRCODE='23514';
    END IF;
    IF parent_kind='REFERENCE' THEN
        RAISE EXCEPTION 'reference connections cannot own executable views' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.stable_keys) AS keys(key_name)
         WHERE NOT EXISTS (
             SELECT 1
               FROM jsonb_array_elements(NEW.native_schema) AS fields(field)
              WHERE fields.field->>'name'=keys.key_name
         )
    ) THEN
        RAISE EXCEPTION 'source view stable keys must exist in the native schema' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER source_view_revision_guard_trigger BEFORE INSERT OR UPDATE ON source_views FOR EACH ROW EXECUTE FUNCTION source_view_revision_guard();

CREATE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    parent_current boolean;
    parent_stable_keys jsonb;
    parent_native_schema jsonb;
BEGIN
    IF TG_OP='UPDATE' THEN
        IF ROW(OLD.revision_id,OLD.binding_id,OLD.tenant_id,OLD.source_id,OLD.view_id,OLD.view_version,OLD.code,OLD.name,OLD.purpose,OLD.operations,OLD.selected_fields,OLD.key_fields,OLD.page_rows,OLD.response_bytes,OLD.lookup_values,OLD.timeout_ms,OLD.mapping,OLD.parameter_schema,OLD.output_schema,OLD.required_freshness_minutes,OLD.completeness,OLD.sensitivity_handling,OLD.version,OLD.created_by,OLD.created_at)
           IS DISTINCT FROM
           ROW(NEW.revision_id,NEW.binding_id,NEW.tenant_id,NEW.source_id,NEW.view_id,NEW.view_version,NEW.code,NEW.name,NEW.purpose,NEW.operations,NEW.selected_fields,NEW.key_fields,NEW.page_rows,NEW.response_bytes,NEW.lookup_values,NEW.timeout_ms,NEW.mapping,NEW.parameter_schema,NEW.output_schema,NEW.required_freshness_minutes,NEW.completeness,NEW.sensitivity_handling,NEW.version,NEW.created_by,NEW.created_at) THEN
            RAISE EXCEPTION 'source binding revision definition is immutable' USING ERRCODE='23514';
        END IF;
        IF NEW.updated_at < OLD.updated_at THEN
            RAISE EXCEPTION 'source binding updated_at cannot move backwards' USING ERRCODE='23514';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM source_bindings existing
         WHERE existing.binding_id=NEW.binding_id
           AND existing.revision_id<>NEW.revision_id
           AND (existing.tenant_id<>NEW.tenant_id OR existing.source_id<>NEW.source_id OR existing.view_id<>NEW.view_id)
    ) THEN
        RAISE EXCEPTION 'source binding identity cannot change tenant, source or view' USING ERRCODE='23514';
    END IF;
    SELECT is_current,stable_keys,native_schema
      INTO parent_current,parent_stable_keys,parent_native_schema
      FROM source_views
     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version;
    IF NEW.is_current AND NOT COALESCE(parent_current,false) THEN
        RAISE EXCEPTION 'current source binding requires its current view revision' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.selected_fields) AS selected(field_name)
         WHERE NOT EXISTS (
             SELECT 1
               FROM jsonb_array_elements(parent_native_schema) AS fields(field)
              WHERE fields.field->>'name'=selected.field_name
         )
    ) THEN
        RAISE EXCEPTION 'source binding selected fields must exist in the view schema' USING ERRCODE='23514';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM jsonb_array_elements_text(NEW.key_fields) AS keys(key_name)
         WHERE NOT (NEW.selected_fields ? keys.key_name)
            OR NOT (parent_stable_keys ? keys.key_name)
    ) THEN
        RAISE EXCEPTION 'source binding key fields must be selected stable view keys' USING ERRCODE='23514';
    END IF;
    IF (NEW.operations ? 'PAGE' OR NEW.operations ? 'LOOKUP') AND jsonb_array_length(NEW.key_fields)=0 THEN
        RAISE EXCEPTION 'page and lookup bindings require a stable key' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER source_binding_revision_guard_trigger BEFORE INSERT OR UPDATE ON source_bindings FOR EACH ROW EXECUTE FUNCTION source_binding_revision_guard();

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM evidence_sources
         WHERE btrim(endpoint)<>''
           AND (
               octet_length(jsonb_build_object('endpoint',btrim(endpoint))::text)>32768
               OR endpoint ~ '[[:cntrl:]]'
           )
    ) THEN
        RAISE EXCEPTION 'legacy evidence source endpoint is invalid for source connection migration';
    END IF;
END
$$;

INSERT INTO source_connections(
    tenant_id,source_id,code,name,adapter_kind,adapter_version,secret_ref,definition,
    declared_capabilities,verified_capabilities,owner_principal_id,status,is_current,
    effective_from,version,created_by,created_at,updated_at
)
SELECT tenant_id,id,'PRIMARY_REFERENCE','Primary reference','REFERENCE','reference-v1','',
       jsonb_build_object('endpoint',btrim(endpoint)),'[]'::jsonb,'[]'::jsonb,
       owner_principal_id,'ACTIVE',true,created_at,1,owner_principal_id,created_at,updated_at
  FROM evidence_sources
 WHERE btrim(endpoint)<>'';

ALTER TABLE evidence_sources DROP COLUMN endpoint;

COMMIT;

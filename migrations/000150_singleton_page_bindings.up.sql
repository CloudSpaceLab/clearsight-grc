BEGIN;
CREATE OR REPLACE FUNCTION source_binding_revision_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    parent_current boolean;
    parent_stable_keys jsonb;
    parent_native_schema jsonb;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('source_binding:' || NEW.binding_id::text,0));
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
     WHERE view_id=NEW.view_id AND tenant_id=NEW.tenant_id AND source_id=NEW.source_id AND version=NEW.view_version
     FOR SHARE;
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
    -- Only a REST root-object singleton may perform a keyless PAGE read.
    IF (NEW.operations ? 'PAGE' OR NEW.operations ? 'LOOKUP') AND jsonb_array_length(NEW.key_fields)=0
       AND NOT (NEW.operations = '["PAGE"]'::jsonb AND EXISTS (
           SELECT 1 FROM source_views sv JOIN source_connections sc
             ON sc.connection_id=sv.connection_id AND sc.tenant_id=sv.tenant_id
            AND sc.source_id=sv.source_id AND sc.version=sv.connection_version
            WHERE sv.view_id=NEW.view_id AND sv.tenant_id=NEW.tenant_id
              AND sv.source_id=NEW.source_id AND sv.version=NEW.view_version
              AND sc.adapter_kind='REST_JSON' AND sv.definition->'singleton'='true'::jsonb
              AND COALESCE(sv.definition->>'records_pointer','')=''
              AND COALESCE(sv.definition->'pagination'->>'mode','NONE')='NONE'
              AND COALESCE(sv.definition->'pagination'->>'page_size_query_param','')=''
              AND (NOT sv.definition ? 'lookup' OR sv.definition->'lookup'='null'::jsonb)
       )) THEN
        RAISE EXCEPTION 'page and lookup bindings require a stable key' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
COMMIT;

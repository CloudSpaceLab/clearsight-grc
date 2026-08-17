BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM source_bindings LIMIT 1) OR EXISTS (SELECT 1 FROM source_views LIMIT 1) THEN
        RAISE EXCEPTION 'source views or bindings exist; migrate them before rolling back the source catalog';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM source_connections
         WHERE code<>'PRIMARY_REFERENCE'
            OR name<>'Primary reference'
            OR adapter_kind<>'REFERENCE'
            OR adapter_version<>'reference-v1'
            OR secret_ref<>''
            OR status<>'ACTIVE'
            OR NOT is_current
            OR version<>1
            OR effective_from IS NULL
            OR effective_until IS NOT NULL
            OR jsonb_typeof(definition)<>'object'
            OR NULLIF(btrim(definition->>'endpoint'),'') IS NULL
            OR definition <> jsonb_build_object('endpoint',definition->>'endpoint')
    ) THEN
        RAISE EXCEPTION 'non-legacy source connection data exists; migrate it before rolling back the source catalog';
    END IF;
END
$$;

ALTER TABLE evidence_sources ADD COLUMN endpoint text NOT NULL DEFAULT '';

UPDATE evidence_sources es
   SET endpoint=sc.definition->>'endpoint'
  FROM source_connections sc
 WHERE sc.tenant_id=es.tenant_id
   AND sc.source_id=es.id
   AND sc.code='PRIMARY_REFERENCE'
   AND sc.adapter_kind='REFERENCE'
   AND sc.is_current;

DROP TABLE source_bindings;
DROP TABLE source_views;
DROP TABLE source_connections;
DROP FUNCTION source_binding_revision_guard();
DROP FUNCTION source_view_revision_guard();
DROP FUNCTION source_connection_revision_guard();
DROP FUNCTION source_catalog_native_schema_valid(jsonb);
DROP FUNCTION source_catalog_text_array_valid(jsonb,integer,integer,text[]);

COMMIT;

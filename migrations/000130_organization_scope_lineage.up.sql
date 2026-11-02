BEGIN;

CREATE TABLE organization_scope_lineage_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    scope_version bigint NOT NULL CHECK (scope_version>0),
    parent_scope_id uuid,
    department_path text[] NOT NULL,
    status text NOT NULL CHECK (status IN ('ACTIVE','RETIRED')),
    event_kind text NOT NULL CHECK (event_kind IN ('BACKFILL','CREATE','MOVE','RETIRE','STRUCTURE_CHANGE')),
    effective_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (tenant_id,legal_entity_id,scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    FOREIGN KEY (tenant_id,legal_entity_id,parent_scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    CHECK (cardinality(department_path) BETWEEN 1 AND 12),
    CHECK (array_position(department_path,'') IS NULL)
);

CREATE INDEX organization_scope_lineage_lookup_idx
    ON organization_scope_lineage_events(
        tenant_id,legal_entity_id,scope_id,effective_at DESC,id DESC
    );

CREATE INDEX organization_scope_lineage_effective_idx
    ON organization_scope_lineage_events(
        tenant_id,legal_entity_id,effective_at DESC,scope_id
    );

INSERT INTO organization_scope_lineage_events(
    tenant_id,legal_entity_id,scope_id,scope_version,parent_scope_id,
    department_path,status,event_kind,effective_at
)
SELECT scope.tenant_id,scope.legal_entity_id,scope.id,scope.version,scope.parent_scope_id,
       scope.department_path,scope.status,'BACKFILL',clock_timestamp()
FROM organization_scopes scope
WHERE scope.status='ACTIVE'
  AND scope.valid_from<=clock_timestamp()
  AND (scope.valid_until IS NULL OR clock_timestamp()<scope.valid_until);

CREATE FUNCTION capture_organization_scope_lineage() RETURNS trigger
LANGUAGE plpgsql
AS $lineage$
DECLARE
    event_kind_value text;
BEGIN
    IF TG_OP='INSERT' THEN
        event_kind_value := 'CREATE';
    ELSIF NEW.parent_scope_id IS NOT DISTINCT FROM OLD.parent_scope_id
       AND NEW.department_path IS NOT DISTINCT FROM OLD.department_path
       AND NEW.status IS NOT DISTINCT FROM OLD.status
       AND NEW.valid_until IS NOT DISTINCT FROM OLD.valid_until THEN
        RETURN NEW;
    ELSIF NEW.status='RETIRED' AND OLD.status IS DISTINCT FROM NEW.status THEN
        event_kind_value := 'RETIRE';
    ELSIF NEW.parent_scope_id IS DISTINCT FROM OLD.parent_scope_id
       OR NEW.department_path IS DISTINCT FROM OLD.department_path THEN
        event_kind_value := 'MOVE';
    ELSE
        event_kind_value := 'STRUCTURE_CHANGE';
    END IF;

    INSERT INTO organization_scope_lineage_events(
        tenant_id,legal_entity_id,scope_id,scope_version,parent_scope_id,
        department_path,status,event_kind,effective_at
    )
    VALUES(
        NEW.tenant_id,NEW.legal_entity_id,NEW.id,NEW.version,NEW.parent_scope_id,
        NEW.department_path,NEW.status,event_kind_value,clock_timestamp()
    );
    RETURN NEW;
END;
$lineage$;

CREATE TRIGGER organization_scopes_capture_lineage
    AFTER INSERT OR UPDATE OF parent_scope_id,department_path,status,valid_until
    ON organization_scopes
    FOR EACH ROW EXECUTE FUNCTION capture_organization_scope_lineage();

CREATE FUNCTION prevent_organization_scope_lineage_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $lineage_immutable$
BEGIN
    RAISE EXCEPTION 'Organization scope lineage history is immutable';
END;
$lineage_immutable$;

CREATE TRIGGER organization_scope_lineage_immutable
    BEFORE UPDATE OR DELETE ON organization_scope_lineage_events
    FOR EACH ROW EXECUTE FUNCTION prevent_organization_scope_lineage_mutation();

COMMIT;

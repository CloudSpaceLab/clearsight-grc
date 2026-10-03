BEGIN;

DO $
BEGIN
    IF EXISTS (SELECT 1 FROM organization_scope_revisions) THEN
        RAISE EXCEPTION 'organization scope revisions exist; refusing to erase organization governance history';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM governance_decisions
        WHERE object_type IN ('ORGANIZATION_SCOPE','ORGANIZATION_SCOPE_REVISION')
    ) THEN
        RAISE EXCEPTION 'organization scope governance decisions exist; refusing to erase organization governance history';
    END IF;
END;
$;

DROP INDEX IF EXISTS organization_scope_revisions_queue_idx;
DROP INDEX IF EXISTS organization_scope_revisions_pending_idx;
DROP TABLE IF EXISTS organization_scope_revisions;

ALTER TABLE governance_decisions
    DROP CONSTRAINT IF EXISTS governance_decisions_object_type_check;
ALTER TABLE governance_decisions
    ADD CONSTRAINT governance_decisions_object_type_check
    CHECK (object_type IN (
        'ROUTING_POLICY',
        'DELEGATION',
        'SEGREGATION_RULE',
        'SCIM_SOURCE',
        'DIRECTORY_GROUP_ROLE_BINDING'
    ));

CREATE OR REPLACE FUNCTION bind_legacy_organization_scope() RETURNS trigger
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

COMMIT;

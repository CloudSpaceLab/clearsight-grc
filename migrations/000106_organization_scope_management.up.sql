BEGIN;

CREATE TABLE organization_scope_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    operation text NOT NULL CHECK (operation IN ('CREATE','UPDATE','MOVE','RETIRE')),
    base_version bigint NOT NULL DEFAULT 0 CHECK (base_version>=0),
    proposed_parent_scope_id uuid,
    proposed_code text NOT NULL DEFAULT '',
    proposed_name text NOT NULL DEFAULT '',
    proposed_kind text NOT NULL DEFAULT '',
    maker_id uuid NOT NULL,
    checker_id uuid,
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPLIED','REJECTED')),
    rationale text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    decided_at timestamptz,
    applied_at timestamptz,
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (tenant_id, maker_id) REFERENCES principals(tenant_id, id),
    FOREIGN KEY (tenant_id, checker_id) REFERENCES principals(tenant_id, id),
    FOREIGN KEY (tenant_id, legal_entity_id, proposed_parent_scope_id)
        REFERENCES organization_scopes(tenant_id, legal_entity_id, id),
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (char_length(proposed_code)<=128),
    CHECK (char_length(proposed_name)<=240),
    CHECK (proposed_kind='' OR proposed_kind IN ('BRANCH','DEPARTMENT','FUNCTION','BUSINESS_UNIT','CRITICAL_SERVICE'))
);

CREATE UNIQUE INDEX organization_scope_revisions_pending_idx
    ON organization_scope_revisions(tenant_id, legal_entity_id, scope_id)
    WHERE status='PENDING';
CREATE INDEX organization_scope_revisions_queue_idx
    ON organization_scope_revisions(tenant_id, legal_entity_id, status, created_at, id);

CREATE OR REPLACE FUNCTION bind_legacy_organization_scope() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    bound_path text[];
BEGIN
    IF NEW.legal_entity_id IS NULL THEN
        NEW.organization_scope_id := NULL;
        RETURN NEW;
    END IF;

    IF NEW.organization_scope_id IS NOT NULL THEN
        SELECT scope.department_path INTO bound_path
        FROM organization_scopes scope
        WHERE scope.tenant_id=NEW.tenant_id
          AND scope.legal_entity_id=NEW.legal_entity_id
          AND scope.id=NEW.organization_scope_id
          AND scope.status='ACTIVE'
          AND scope.valid_from<=clock_timestamp()
          AND (scope.valid_until IS NULL OR clock_timestamp()<scope.valid_until);
        IF bound_path IS NULL THEN
            RAISE EXCEPTION 'organization scope is unavailable';
        END IF;
        NEW.department_path := bound_path;
        RETURN NEW;
    END IF;

    IF cardinality(NEW.department_path)=0 THEN
        RETURN NEW;
    END IF;

    NEW.organization_scope_id := organization_scope_for_department_path(
        NEW.tenant_id,
        NEW.legal_entity_id,
        NEW.department_path,
        NEW.valid_from
    );
    SELECT scope.department_path INTO NEW.department_path
    FROM organization_scopes scope
    WHERE scope.id=NEW.organization_scope_id;
    RETURN NEW;
END;
$$;

COMMIT;

BEGIN;

CREATE TABLE organization_position_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    position_id uuid NOT NULL,
    operation text NOT NULL CHECK (operation IN ('CREATE','UPDATE','RETIRE')),
    base_version bigint NOT NULL DEFAULT 0 CHECK (base_version>=0),
    previous_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    proposed_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    effective_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    maker_id uuid NOT NULL,
    checker_id uuid,
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','APPLIED','REJECTED')),
    rationale text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    decided_at timestamptz,
    applied_at timestamptz,
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (tenant_id, maker_id) REFERENCES principals(tenant_id, id),
    FOREIGN KEY (tenant_id, checker_id) REFERENCES principals(tenant_id, id),
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (jsonb_typeof(previous_state)='object'),
    CHECK (jsonb_typeof(proposed_state)='object'),
    CHECK (
        (operation='CREATE' AND base_version=0 AND previous_state='{}'::jsonb)
        OR
        (operation IN ('UPDATE','RETIRE') AND base_version>0)
    )
);

CREATE UNIQUE INDEX organization_position_revisions_pending_idx
    ON organization_position_revisions(tenant_id, legal_entity_id, position_id)
    WHERE status IN ('PENDING','APPROVED');

CREATE INDEX organization_position_revisions_queue_idx
    ON organization_position_revisions(tenant_id, legal_entity_id, status, effective_at, created_at, id);

CREATE FUNCTION protect_organization_position_revision() RETURNS trigger
LANGUAGE plpgsql
AS $position_revision$
BEGIN
    IF TG_OP='DELETE' THEN
        RAISE EXCEPTION 'Organization position revision history is retained';
    END IF;
    IF NEW.tenant_id IS DISTINCT FROM OLD.tenant_id
       OR NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id
       OR NEW.position_id IS DISTINCT FROM OLD.position_id
       OR NEW.operation IS DISTINCT FROM OLD.operation
       OR NEW.base_version IS DISTINCT FROM OLD.base_version
       OR NEW.previous_state IS DISTINCT FROM OLD.previous_state
       OR NEW.proposed_state IS DISTINCT FROM OLD.proposed_state
       OR NEW.effective_at IS DISTINCT FROM OLD.effective_at
       OR NEW.maker_id IS DISTINCT FROM OLD.maker_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'Organization position revision proposal is immutable';
    END IF;
    RETURN NEW;
END;
$position_revision$;

CREATE TRIGGER organization_position_revisions_history_guard
    BEFORE UPDATE OR DELETE ON organization_position_revisions
    FOR EACH ROW EXECUTE FUNCTION protect_organization_position_revision();

ALTER TABLE governance_decisions
    DROP CONSTRAINT IF EXISTS governance_decisions_object_type_check;
ALTER TABLE governance_decisions
    ADD CONSTRAINT governance_decisions_object_type_check
    CHECK (object_type IN (
        'ROUTING_POLICY',
        'DELEGATION',
        'SEGREGATION_RULE',
        'SCIM_SOURCE',
        'DIRECTORY_GROUP_ROLE_BINDING',
        'ORGANIZATION_SCOPE',
        'ORGANIZATION_SCOPE_REVISION',
        'ORGANIZATION_POSITION_REVISION'
    ));

COMMIT;

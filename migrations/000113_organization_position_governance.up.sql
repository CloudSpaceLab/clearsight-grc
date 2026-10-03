BEGIN;

CREATE TABLE organization_position_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    position_id uuid NOT NULL,
    operation text NOT NULL CHECK (operation IN ('CREATE','UPDATE','RETIRE')),
    base_version bigint NOT NULL DEFAULT 0 CHECK (base_version>=0),

    base_code text NOT NULL DEFAULT '',
    base_title text NOT NULL DEFAULT '',
    base_function_name text NOT NULL DEFAULT '',
    base_organization_scope_id uuid,
    base_parent_position_id uuid,
    base_occupant_principal_id uuid,

    proposed_code text NOT NULL DEFAULT '',
    proposed_title text NOT NULL DEFAULT '',
    proposed_function_name text NOT NULL DEFAULT '',
    proposed_organization_scope_id uuid,
    proposed_parent_position_id uuid,
    proposed_occupant_principal_id uuid,

    maker_id uuid NOT NULL,
    checker_id uuid,
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPLIED','REJECTED')),
    rationale text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    decided_at timestamptz,
    applied_at timestamptz,

    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (tenant_id,legal_entity_id,base_organization_scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    FOREIGN KEY (tenant_id,legal_entity_id,proposed_organization_scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    FOREIGN KEY (base_parent_position_id) REFERENCES org_positions(id),
    FOREIGN KEY (proposed_parent_position_id) REFERENCES org_positions(id),
    FOREIGN KEY (base_occupant_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (proposed_occupant_principal_id,tenant_id) REFERENCES principals(id,tenant_id),

    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (char_length(base_code)<=128 AND char_length(proposed_code)<=128),
    CHECK (char_length(base_title)<=240 AND char_length(proposed_title)<=240),
    CHECK (char_length(base_function_name)<=240 AND char_length(proposed_function_name)<=240)
);

CREATE UNIQUE INDEX organization_position_revisions_pending_idx
    ON organization_position_revisions(tenant_id,legal_entity_id,position_id)
    WHERE status='PENDING';
CREATE INDEX organization_position_revisions_queue_idx
    ON organization_position_revisions(tenant_id,legal_entity_id,status,created_at,id);

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
        'ORGANIZATION_POSITION',
        'ORGANIZATION_POSITION_REVISION'
    ));

CREATE FUNCTION validate_organization_position_write() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    valid_parent boolean;
    valid_occupant boolean;
    creates_cycle boolean;
BEGIN
    IF NEW.legal_entity_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.parent_position_id IS NOT NULL THEN
        IF NEW.parent_position_id=NEW.id THEN
            RAISE EXCEPTION 'position cannot report to itself';
        END IF;

        SELECT EXISTS(
            SELECT 1
            FROM org_positions parent
            WHERE parent.id=NEW.parent_position_id
              AND parent.tenant_id=NEW.tenant_id
              AND parent.legal_entity_id=NEW.legal_entity_id
              AND parent.valid_from<=clock_timestamp()
              AND (parent.valid_until IS NULL OR clock_timestamp()<parent.valid_until)
        ) INTO valid_parent;
        IF NOT valid_parent THEN
            RAISE EXCEPTION 'parent position is outside the active legal entity';
        END IF;

        WITH RECURSIVE chain(id,parent_position_id,depth,path) AS (
            SELECT parent.id,parent.parent_position_id,1,ARRAY[parent.id]
            FROM org_positions parent
            WHERE parent.id=NEW.parent_position_id
              AND parent.tenant_id=NEW.tenant_id
              AND parent.legal_entity_id=NEW.legal_entity_id
            UNION ALL
            SELECT parent.id,parent.parent_position_id,chain.depth+1,chain.path || parent.id
            FROM chain
            JOIN org_positions parent
              ON parent.id=chain.parent_position_id
             AND parent.tenant_id=NEW.tenant_id
             AND parent.legal_entity_id=NEW.legal_entity_id
            WHERE chain.depth<64
              AND NOT parent.id=ANY(chain.path)
        )
        SELECT EXISTS(SELECT 1 FROM chain WHERE id=NEW.id) INTO creates_cycle;
        IF creates_cycle THEN
            RAISE EXCEPTION 'position reporting cycle is not allowed';
        END IF;
    END IF;

    IF NEW.occupant_principal_id IS NOT NULL THEN
        SELECT EXISTS(
            SELECT 1
            FROM principals principal
            WHERE principal.id=NEW.occupant_principal_id
              AND principal.tenant_id=NEW.tenant_id
              AND principal.kind='PERSON'
              AND principal.status='ACTIVE'
              AND principal.valid_from<=clock_timestamp()
              AND (principal.valid_until IS NULL OR clock_timestamp()<principal.valid_until)
        ) INTO valid_occupant;
        IF NOT valid_occupant THEN
            RAISE EXCEPTION 'position occupant is not an active person in this tenant';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER organization_position_write_guard
    BEFORE INSERT OR UPDATE OF legal_entity_id,parent_position_id,occupant_principal_id
    ON org_positions
    FOR EACH ROW EXECUTE FUNCTION validate_organization_position_write();

COMMIT;

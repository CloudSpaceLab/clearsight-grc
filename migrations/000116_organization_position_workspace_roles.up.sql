BEGIN;

CREATE TABLE organization_position_role_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    position_id uuid NOT NULL REFERENCES org_positions(id),
    role_template_id uuid NOT NULL REFERENCES role_templates(id),
    operation text NOT NULL CHECK (operation IN ('ADD','REMOVE')),
    base_position_version bigint NOT NULL CHECK (base_position_version>0),
    base_role_version bigint NOT NULL CHECK (base_role_version>0),
    base_binding_id uuid REFERENCES position_role_bindings(id),
    role_code text NOT NULL,
    role_name text NOT NULL,
    capabilities text[] NOT NULL DEFAULT '{}',
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
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (
        (operation='ADD' AND base_binding_id IS NULL)
        OR
        (operation='REMOVE' AND base_binding_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX organization_position_role_revisions_pending_idx
    ON organization_position_role_revisions(tenant_id,legal_entity_id,position_id,role_template_id)
    WHERE status='PENDING';

CREATE INDEX organization_position_role_revisions_queue_idx
    ON organization_position_role_revisions(tenant_id,legal_entity_id,status,created_at,id);

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
        'ORGANIZATION_POSITION_REVISION',
        'LEGAL_ENTITY_DATA_BOUNDARY',
        'LEGAL_ENTITY_DATA_BOUNDARY_REVISION',
        'ORGANIZATION_POSITION_ROLE_BINDING',
        'ORGANIZATION_POSITION_ROLE_REVISION'
    ));

COMMIT;

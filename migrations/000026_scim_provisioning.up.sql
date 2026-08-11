BEGIN;

ALTER TABLE role_templates
    ADD CONSTRAINT role_templates_tenant_identity_unique UNIQUE (tenant_id, id);

CREATE TABLE scim_sources (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code text NOT NULL,
    token_hash bytea NOT NULL,
    identity_issuer text,
    subject_attribute text NOT NULL DEFAULT 'externalId' CHECK (subject_attribute IN ('externalId','userName')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','REVOKED')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, code),
    UNIQUE (token_hash),
    CHECK (length(code) BETWEEN 1 AND 80),
    CHECK (octet_length(token_hash) = 32),
    CHECK (identity_issuer IS NULL OR length(identity_issuer) BETWEEN 1 AND 2048)
);

CREATE TABLE scim_users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    source_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    external_id text,
    user_name text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    deleted_at timestamptz,
    FOREIGN KEY (tenant_id, source_id) REFERENCES scim_sources(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, principal_id) REFERENCES principals(tenant_id, id),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, principal_id),
    CHECK (external_id IS NULL OR length(external_id) BETWEEN 1 AND 2048),
    CHECK (length(user_name) BETWEEN 1 AND 320)
);
CREATE UNIQUE INDEX scim_users_source_external_active_idx
    ON scim_users(source_id, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX scim_users_source_username_active_idx
    ON scim_users(source_id, lower(user_name))
    WHERE deleted_at IS NULL;
CREATE INDEX scim_users_source_list_idx
    ON scim_users(source_id, updated_at DESC, id)
    WHERE deleted_at IS NULL;

CREATE TABLE directory_groups (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    source_id uuid NOT NULL,
    external_id text,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    deleted_at timestamptz,
    FOREIGN KEY (tenant_id, source_id) REFERENCES scim_sources(tenant_id, id) ON DELETE CASCADE,
    UNIQUE (tenant_id, id),
    CHECK (external_id IS NULL OR length(external_id) BETWEEN 1 AND 2048),
    CHECK (length(display_name) BETWEEN 1 AND 200)
);
CREATE UNIQUE INDEX directory_groups_source_external_active_idx
    ON directory_groups(source_id, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX directory_groups_source_name_idx
    ON directory_groups(source_id, lower(display_name), id)
    WHERE deleted_at IS NULL;

CREATE TABLE directory_group_members (
    tenant_id uuid NOT NULL,
    group_id uuid NOT NULL,
    scim_user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (group_id, scim_user_id),
    FOREIGN KEY (tenant_id, group_id) REFERENCES directory_groups(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, scim_user_id) REFERENCES scim_users(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX directory_group_members_user_idx ON directory_group_members(tenant_id, scim_user_id, group_id);

CREATE TABLE directory_group_role_bindings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    group_id uuid NOT NULL,
    role_template_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    department_path text[] NOT NULL DEFAULT ARRAY[]::text[],
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (tenant_id, group_id) REFERENCES directory_groups(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, role_template_id) REFERENCES role_templates(tenant_id, id),
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    CHECK (valid_until IS NULL OR valid_from < valid_until),
    CHECK (cardinality(department_path) <= 12),
    CHECK (array_position(department_path, '') IS NULL)
);
CREATE INDEX directory_group_role_bindings_effective_idx
    ON directory_group_role_bindings(tenant_id, legal_entity_id, group_id, valid_from, valid_until);

COMMIT;

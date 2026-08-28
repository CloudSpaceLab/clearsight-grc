BEGIN;

CREATE TABLE form_brand_assets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    artifact_key text NOT NULL CHECK (artifact_key=btrim(artifact_key) AND char_length(artifact_key) BETWEEN 1 AND 1024),
    digest_hex text NOT NULL CHECK (digest_hex ~ '^[0-9a-fA-F]{64}$'),
    media_type text NOT NULL CHECK (media_type='image/png'),
    width integer NOT NULL CHECK (width BETWEEN 1 AND 4096),
    height integer NOT NULL CHECK (height BETWEEN 1 AND 4096),
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 524288),
    alt_text text NOT NULL CHECK (alt_text=btrim(alt_text) AND char_length(alt_text) BETWEEN 1 AND 160),
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,legal_entity_id,digest_hex),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX form_brand_assets_scope_idx
    ON form_brand_assets(tenant_id,legal_entity_id,created_at DESC,id DESC);

CREATE TABLE form_communication_profiles (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    default_locale text NOT NULL CHECK (default_locale=btrim(default_locale) AND char_length(default_locale) BETWEEN 2 AND 32),
    bank_name text NOT NULL CHECK (bank_name=btrim(bank_name) AND char_length(bank_name) BETWEEN 1 AND 200),
    support_contact text NOT NULL DEFAULT '' CHECK (support_contact=btrim(support_contact) AND char_length(support_contact) <= 320),
    brand_asset_id uuid,
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','RETIRED')),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    maker_id uuid NOT NULL,
    checker_id uuid,
    rollback_origin_version bigint,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,legal_entity_id,version),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (brand_asset_id,tenant_id,legal_entity_id) REFERENCES form_brand_assets(id,tenant_id,legal_entity_id),
    FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (tenant_id,legal_entity_id,rollback_origin_version)
        REFERENCES form_communication_profiles(tenant_id,legal_entity_id,version),
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (effective_until IS NULL OR effective_until>effective_from),
    CHECK (updated_at>=created_at)
);
CREATE INDEX form_communication_profiles_scope_idx
    ON form_communication_profiles(tenant_id,legal_entity_id,status,effective_from DESC,version DESC);

CREATE TABLE form_communication_templates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('INVITATION','REMINDER','DUE_SOON','EXPIRED','CHANGE_REQUESTED','AMENDMENT','COMPLETION')),
    locale text NOT NULL CHECK (locale=btrim(locale) AND char_length(locale) BETWEEN 2 AND 32),
    version bigint NOT NULL CHECK (version > 0),
    subject_template text NOT NULL CHECK (subject_template=btrim(subject_template) AND char_length(subject_template) BETWEEN 1 AND 200 AND position(E'\n' in subject_template)=0 AND position(E'\r' in subject_template)=0),
    document jsonb NOT NULL CHECK (jsonb_typeof(document)='array' AND jsonb_array_length(document) BETWEEN 1 AND 100 AND octet_length(document::text)<=65536),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','RETIRED')),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    maker_id uuid NOT NULL,
    checker_id uuid,
    rollback_origin_version bigint,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,legal_entity_id,action,locale,version),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (tenant_id,legal_entity_id,action,locale,rollback_origin_version)
        REFERENCES form_communication_templates(tenant_id,legal_entity_id,action,locale,version),
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (effective_until IS NULL OR effective_until>effective_from),
    CHECK (updated_at>=created_at)
);
CREATE INDEX form_communication_templates_lookup_idx
    ON form_communication_templates(tenant_id,legal_entity_id,action,locale,status,effective_from DESC,version DESC);

CREATE TABLE form_delivery_attempts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    recipient_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('INVITATION','REMINDER','DUE_SOON','EXPIRED','CHANGE_REQUESTED','AMENDMENT','COMPLETION')),
    template_id uuid NOT NULL,
    template_version bigint NOT NULL CHECK (template_version>0),
    outbox_event_id uuid NOT NULL REFERENCES outbox_events(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('PENDING','DELIVERED','FAILED','SKIPPED')),
    provider_message_id text NOT NULL DEFAULT '' CHECK (char_length(provider_message_id)<=512),
    recipient_hint text NOT NULL DEFAULT '' CHECK (char_length(recipient_hint)<=320),
    failure_code text NOT NULL DEFAULT '' CHECK (char_length(failure_code)<=128),
    attempted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id,outbox_event_id,recipient_id,action),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id)
        REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id,tenant_id,legal_entity_id,distribution_id)
        REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id,distribution_id) ON DELETE CASCADE,
    FOREIGN KEY (template_id,tenant_id,legal_entity_id)
        REFERENCES form_communication_templates(id,tenant_id,legal_entity_id)
);
CREATE INDEX form_delivery_attempts_distribution_idx
    ON form_delivery_attempts(tenant_id,legal_entity_id,distribution_id,attempted_at DESC,id DESC);
CREATE INDEX form_delivery_attempts_outbox_idx
    ON form_delivery_attempts(tenant_id,outbox_event_id,status,attempted_at DESC);

COMMIT;

BEGIN;

-- Operator-only demo curation metadata. No bank lifecycle state is changed.
ALTER TABLE third_party_relationships
    ADD CONSTRAINT third_party_relationships_archive_scope_unique UNIQUE (id,tenant_id,legal_entity_id);
ALTER TABLE matters
    ADD CONSTRAINT matters_archive_scope_unique UNIQUE (id,tenant_id,legal_entity_id);

CREATE TABLE demo_record_archives (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    record_type text NOT NULL CHECK (record_type IN ('VENDOR_RELATIONSHIP','MATTER')),
    record_id uuid NOT NULL,
    vendor_relationship_id uuid GENERATED ALWAYS AS (CASE WHEN record_type='VENDOR_RELATIONSHIP' THEN record_id END) STORED,
    matter_id uuid GENERATED ALWAYS AS (CASE WHEN record_type='MATTER' THEN record_id END) STORED,
    reason text NOT NULL CHECK (char_length(btrim(reason)) BETWEEN 1 AND 2000),
    source_manifest text NOT NULL CHECK (char_length(btrim(source_manifest)) BETWEEN 1 AND 512),
    archived_by uuid NOT NULL,
    archived_at timestamptz NOT NULL,
    restored_by uuid,
    restored_at timestamptz,
    restoration_reason text NOT NULL DEFAULT '',
    FOREIGN KEY (vendor_relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id,legal_entity_id) REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY (archived_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (restored_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((restored_at IS NULL AND restored_by IS NULL AND restoration_reason='') OR
           (restored_at IS NOT NULL AND restored_at>=archived_at AND restored_by IS NOT NULL AND char_length(btrim(restoration_reason)) BETWEEN 1 AND 2000))
);
CREATE UNIQUE INDEX demo_record_archives_current_idx
    ON demo_record_archives(tenant_id,legal_entity_id,record_type,record_id)
    WHERE restored_at IS NULL;

CREATE FUNCTION protect_demo_record_archive_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP='DELETE' THEN
        RAISE EXCEPTION 'Demo archive history is retained';
    END IF;
    -- Stored generated IDs are recalculated after BEFORE triggers. Their
    -- immutable source fields (record_type and record_id) remain compared.
    IF OLD.restored_at IS NOT NULL OR NEW.restored_at IS NULL OR
       (to_jsonb(NEW)-'restored_at'-'restored_by'-'restoration_reason'-'vendor_relationship_id'-'matter_id') IS DISTINCT FROM
       (to_jsonb(OLD)-'restored_at'-'restored_by'-'restoration_reason'-'vendor_relationship_id'-'matter_id') THEN
        RAISE EXCEPTION 'Only one attributed restoration may update demo archive history';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER demo_record_archive_history_immutable
    BEFORE UPDATE OR DELETE ON demo_record_archives
    FOR EACH ROW EXECUTE FUNCTION protect_demo_record_archive_history();

COMMIT;

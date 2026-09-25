BEGIN;

-- Operator curation only. Archiving never changes a request, response or assessment.
CREATE TABLE demo_form_distribution_archives (
    distribution_id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    reason text NOT NULL CHECK (char_length(btrim(reason)) BETWEEN 1 AND 2000),
    source_manifest text NOT NULL CHECK (char_length(btrim(source_manifest)) BETWEEN 1 AND 512),
    archived_by uuid NOT NULL,
    archived_at timestamptz NOT NULL DEFAULT now(),
    restored_by uuid,
    restored_at timestamptz,
    restoration_reason text NOT NULL DEFAULT '',
    FOREIGN KEY (archived_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (restored_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id),
    CHECK ((restored_at IS NULL AND restored_by IS NULL AND restoration_reason='') OR
           (restored_at IS NOT NULL AND restored_at>=archived_at AND restored_by IS NOT NULL AND char_length(btrim(restoration_reason)) BETWEEN 1 AND 2000))
);

CREATE FUNCTION protect_demo_form_distribution_archive() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP='DELETE' THEN
        RAISE EXCEPTION 'Demo form archive history is retained';
    END IF;
    IF OLD.restored_at IS NOT NULL OR NEW.restored_at IS NULL OR
       (to_jsonb(NEW)-'restored_at'-'restored_by'-'restoration_reason') IS DISTINCT FROM
       (to_jsonb(OLD)-'restored_at'-'restored_by'-'restoration_reason') THEN
        RAISE EXCEPTION 'Only one attributed restoration may update demo form archive history';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER demo_form_distribution_archive_immutable
    BEFORE UPDATE OR DELETE ON demo_form_distribution_archives
    FOR EACH ROW EXECUTE FUNCTION protect_demo_form_distribution_archive();

COMMIT;

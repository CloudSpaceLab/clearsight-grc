BEGIN;

CREATE UNIQUE INDEX matters_form_origin_scope_uq
    ON matters(id,tenant_id,legal_entity_id);

ALTER TABLE monitoring_form_templates
    ADD COLUMN origin_type text,
    ADD COLUMN origin_id uuid,
    ADD CONSTRAINT monitoring_form_templates_origin_pair_ck CHECK (
        (origin_type IS NULL AND origin_id IS NULL)
        OR (origin_type='MATTER' AND origin_id IS NOT NULL)
    ),
    ADD CONSTRAINT monitoring_form_templates_origin_matter_fk
        FOREIGN KEY (origin_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id);

CREATE INDEX monitoring_form_templates_origin_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,origin_type,origin_id,updated_at DESC,id,version DESC)
    WHERE origin_id IS NOT NULL;

CREATE FUNCTION enforce_monitoring_form_origin_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM monitoring_form_templates existing
        WHERE existing.tenant_id=NEW.tenant_id
          AND existing.id=NEW.id
          AND (
              existing.origin_type IS DISTINCT FROM NEW.origin_type
              OR existing.origin_id IS DISTINCT FROM NEW.origin_id
          )
    ) THEN
        RAISE EXCEPTION 'Form origin cannot change across revisions'
            USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER monitoring_form_origin_immutable_trigger
BEFORE INSERT OR UPDATE OF origin_type,origin_id ON monitoring_form_templates
FOR EACH ROW EXECUTE FUNCTION enforce_monitoring_form_origin_immutable();

COMMIT;

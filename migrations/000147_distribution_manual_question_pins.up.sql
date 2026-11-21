BEGIN;
CREATE TABLE capture_distribution_manual_question_pins (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL,
 legal_entity_id uuid NOT NULL,
 distribution_id uuid NOT NULL,
 field_id text NOT NULL CHECK (length(field_id) BETWEEN 1 AND 80),
 approval_id uuid NOT NULL,
 created_at timestamptz NOT NULL,
 FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id),
 FOREIGN KEY (approval_id,tenant_id,legal_entity_id) REFERENCES program_adoption_question_approvals(id,tenant_id,legal_entity_id),
 UNIQUE (tenant_id,legal_entity_id,distribution_id,field_id)
);
CREATE FUNCTION prevent_distribution_manual_pin_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Distribution manual question pins are immutable'; END $$;
CREATE TRIGGER distribution_manual_pin_immutable BEFORE UPDATE OR DELETE ON capture_distribution_manual_question_pins FOR EACH ROW EXECUTE FUNCTION prevent_distribution_manual_pin_mutation();
COMMIT;

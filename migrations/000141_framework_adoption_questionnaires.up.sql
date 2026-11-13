BEGIN;
CREATE TABLE framework_adoption_questionnaires (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 tenant_id uuid NOT NULL,
 legal_entity_id uuid NOT NULL,
 program_id uuid NOT NULL,
 adoption_id uuid NOT NULL,
 edition_id uuid NOT NULL,
 form_template_id uuid NOT NULL,
 form_template_version bigint NOT NULL CHECK (form_template_version > 0),
 linked_by uuid NOT NULL,
 linked_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 ended_at timestamptz,
 ended_by uuid,
 end_reason text,
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 UNIQUE (id,tenant_id,legal_entity_id),
 FOREIGN KEY (adoption_id,tenant_id,legal_entity_id,program_id,edition_id) REFERENCES program_framework_adoptions(id,tenant_id,legal_entity_id,program_id,edition_id),
 FOREIGN KEY (tenant_id,legal_entity_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version),
 FOREIGN KEY (linked_by,tenant_id) REFERENCES principals(id,tenant_id),
 FOREIGN KEY (ended_by,tenant_id) REFERENCES principals(id,tenant_id),
 CHECK ((ended_at IS NULL AND ended_by IS NULL AND end_reason IS NULL) OR (ended_at IS NOT NULL AND ended_at >= linked_at AND ended_by IS NOT NULL AND end_reason IS NOT NULL AND length(btrim(end_reason)) BETWEEN 1 AND 1000))
);
CREATE UNIQUE INDEX framework_adoption_questionnaires_current_uq ON framework_adoption_questionnaires(tenant_id,legal_entity_id,program_id,adoption_id) WHERE ended_at IS NULL;
CREATE TABLE framework_adoption_questionnaire_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 link_id uuid NOT NULL,
 tenant_id uuid NOT NULL,
 legal_entity_id uuid NOT NULL,
 event_type text NOT NULL CHECK (event_type IN ('LINKED','UNLINKED')),
 actor_id uuid NOT NULL,
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload)='object'),
 FOREIGN KEY (link_id,tenant_id,legal_entity_id) REFERENCES framework_adoption_questionnaires(id,tenant_id,legal_entity_id),
 FOREIGN KEY (tenant_id,legal_entity_id) REFERENCES legal_entities(tenant_id,id),
 FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX framework_adoption_questionnaire_events_link_idx ON framework_adoption_questionnaire_events(link_id,occurred_at,id);
COMMIT;


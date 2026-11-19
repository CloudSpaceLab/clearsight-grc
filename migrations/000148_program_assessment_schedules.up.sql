BEGIN;
CREATE TABLE program_assessment_schedules (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 tenant_id uuid NOT NULL,
 legal_entity_id uuid NOT NULL,
 program_id uuid NOT NULL REFERENCES programs(id),
 owner_id uuid NOT NULL,
 approver_id uuid,
 form_selection jsonb NOT NULL,
 first_period_start date NOT NULL,
 time_zone text NOT NULL,
 interval_months integer NOT NULL CHECK(interval_months IN (1,3,6,12)),
 response_days integer NOT NULL CHECK(response_days BETWEEN 1 AND 365),
 cycle_limit integer NOT NULL CHECK(cycle_limit BETWEEN 1 AND 12),
 cycle_index integer NOT NULL DEFAULT 0 CHECK(cycle_index >= 0),
 version bigint NOT NULL DEFAULT 1,
 state text NOT NULL DEFAULT 'PENDING' CHECK(state IN ('PENDING','ACTIVE','STOPPED','COMPLETE','BLOCKED')),
 next_attempt_at timestamptz NOT NULL,
 attempts integer NOT NULL DEFAULT 0,
 last_outcome text NOT NULL DEFAULT '',
 last_request_id uuid REFERENCES capture_requests(id),
 rationale text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(approver_id IS NULL OR approver_id <> owner_id)
);
CREATE UNIQUE INDEX program_assessment_schedule_current ON program_assessment_schedules
 (program_id,(form_selection->>'adoption_id'),(form_selection->>'form_template_id'),(form_selection->>'form_template_version'),(form_selection->>'respondent_principal_id'))
 WHERE state IN ('PENDING','ACTIVE','BLOCKED');
CREATE INDEX program_assessment_schedule_due ON program_assessment_schedules (next_attempt_at,id) WHERE state='ACTIVE';
COMMIT;

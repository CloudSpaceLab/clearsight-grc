-- Operator-only, nonproduction Clear Bank demo curation. NOT a migration.
-- Run only after a verified backup and a successful source seed receipt reporting
-- 40 Matters. Confirm CLEARSIGHT_DEMO_MODE=true and
-- CLEARSIGHT_DEMO_SEED_MODE=manual on host.
-- Invocation: psql -X -v ON_ERROR_STOP=1 -v confirmed_nonproduction_demo=on
--             -d clearsight -f source-register-replacement-20260910.sql
-- Only reversible archive metadata is inserted. No lifecycle, assessment,
-- request, response, action, outcome, event or original source record is changed.
-- No private source narratives or workbook content belong in this public script.
\set ON_ERROR_STOP on
\if :{?confirmed_nonproduction_demo}
\else
  \echo 'Confirm the nonproduction demo deployment before curation.'
  \quit 3
\endif
\if :confirmed_nonproduction_demo
\else
  \echo 'Nonproduction demo confirmation is required.'
  \quit 3
\endif

BEGIN;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_database()<>'clearsight' OR NOT EXISTS (
  SELECT 1 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id
  WHERE t.id='00000000-0000-4000-8000-000000000001' AND t.slug='clearsight-demo'
    AND le.id='00000000-0000-4000-8000-000000000002'
 ) THEN RAISE EXCEPTION 'Exact hosted demo scope required'; END IF;
 -- Shares the source installer's lock: an incomplete/running seed must not pass.
 IF NOT pg_try_advisory_xact_lock(842019260910) THEN
  RAISE EXCEPTION 'Source installer is running; wait for its successful receipt';
 END IF;
END $$;
SELECT pg_advisory_xact_lock(hashtext('clearsight-source-curation-20260910'));

CREATE TEMP TABLE replacement_targets(id uuid PRIMARY KEY,trigger_key text NOT NULL) ON COMMIT DROP;
INSERT INTO replacement_targets VALUES
 ('01a08695-fd91-7eb8-8181-e219b1ed540c','bank_operating_demo_v1:access_remediation'),
 ('01a08695-fde7-7118-bf09-f02561ab8a9b','bank_operating_demo_v1:branch_control_followup'),
 ('01a08695-fe0b-7ad0-aad7-4dead87d9356','bank_operating_demo_v1:continuity_retest'),
 ('01a08695-fe22-73b3-aaf7-707c7ea15196','bank_operating_demo_v1:privacy_screening'),
 ('01a08695-fe30-7973-b73c-caebd7c6f2d6','bank_operating_demo_v1:source_verification'),
 ('01a08695-fe46-7a99-aca1-4cbf5d231ee5','bank_operating_demo_v1:filing_preparation'),
 ('01a08695-fe65-7027-a714-c8e5c60c4eb8','bank_operating_demo_v1:loss_recovery'),
 ('01a0810e-00c4-79a6-8628-e95e95c7db10','thirdparty-assessment:01a0810e-00ad-7519-af83-aee0cb52017b');

-- Lock only these eight records against concurrent identity/scope changes.
SELECT m.id FROM matters m JOIN replacement_targets x ON x.id=m.id FOR UPDATE OF m;
DO $$ BEGIN
 IF (SELECT count(*) FROM matters m
     WHERE m.tenant_id='00000000-0000-4000-8000-000000000001'
       AND m.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND m.scope->>'seed_package'='fidelity-source-records-v1'
       AND m.scope->>'sample'='true'
       AND m.trigger_type='SOURCE_REGISTER_IMPORT'
       AND m.trigger_key LIKE 'fidelity-source-records-v1:%'
       AND m.known_facts->>'source_sha256' ~ '^[0-9a-f]{64}$'
       AND length(m.known_facts->>'source_range')>0
       AND length(m.scope->>'source_group')>0
       AND NOT EXISTS (SELECT 1 FROM demo_record_archives a
           WHERE a.tenant_id=m.tenant_id AND a.legal_entity_id=m.legal_entity_id
             AND a.record_type='MATTER' AND a.record_id=m.id AND a.restored_at IS NULL))<>40
 THEN RAISE EXCEPTION 'Expected 40 active source-backed replacement Matters'; END IF;

 IF (SELECT count(*) FROM replacement_targets x JOIN matters m ON m.id=x.id AND m.trigger_key=x.trigger_key
     WHERE m.tenant_id='00000000-0000-4000-8000-000000000001'
       AND m.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND m.scope->>'seed_package'='bank_operating_demo_v1'
       AND m.scope->>'sample'='true' AND m.trigger_type='SAMPLE_REGISTER_ENTRY')<>7
 THEN RAISE EXCEPTION 'Seven generic operating fixture identities must match'; END IF;

 IF NOT EXISTS (SELECT 1 FROM replacement_targets x JOIN matters m ON m.id=x.id AND m.trigger_key=x.trigger_key
     JOIN third_party_assessments a ON a.id::text=m.source_id AND a.tenant_id=m.tenant_id AND a.legal_entity_id=m.legal_entity_id
     WHERE m.id='01a0810e-00c4-79a6-8628-e95e95c7db10'
       AND m.tenant_id='00000000-0000-4000-8000-000000000001'
       AND m.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND m.source_type='THIRD_PARTY_ASSESSMENT'
       AND m.source_id='01a0810e-00ad-7519-af83-aee0cb52017b'
       AND m.trigger_type='VENDOR_DUE_DILIGENCE_STARTED'
       AND m.scope->>'relationship_id'='01a0810d-9a4a-7a9e-a3c2-abf0727927eb'
       AND a.relationship_id='01a0810d-9a4a-7a9e-a3c2-abf0727927eb')
 THEN RAISE EXCEPTION 'Generic Cloudspace assessment Matter provenance changed'; END IF;

 IF NOT EXISTS (SELECT 1 FROM capture_form_distributions d
     WHERE d.id='01a089dd-5d2b-7966-9c57-cbc999831ebe'
       AND d.tenant_id='00000000-0000-4000-8000-000000000001'
       AND d.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND d.form_template_id='01a041c9-8e35-7a5d-9d03-17a524558370'
       AND d.subject_type='VENDOR_RELATIONSHIP'
       AND d.subject_id='01a0810d-9a4a-7a9e-a3c2-abf0727927eb' AND d.status='REVOKED')
 THEN RAISE EXCEPTION 'Source installer must revoke the exact superseded generic capture first'; END IF;
END $$;

INSERT INTO demo_record_archives(tenant_id,legal_entity_id,record_type,record_id,reason,source_manifest,archived_by,archived_at)
 SELECT '00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','MATTER',id,
 'Generic demo work replaced by 40 source-backed register Matters; original workflow states and history retained.',
 'deploy/curation/source-register-replacement-20260910.sql','00000000-0000-4000-8000-000000000105',now()
 FROM replacement_targets
 ON CONFLICT(tenant_id,legal_entity_id,record_type,record_id) WHERE restored_at IS NULL DO NOTHING;

DO $$ BEGIN
 IF (SELECT count(*) FROM replacement_targets x JOIN demo_record_archives a ON a.record_id=x.id
     WHERE a.tenant_id='00000000-0000-4000-8000-000000000001'
       AND a.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND a.record_type='MATTER' AND a.restored_at IS NULL)<>8
 THEN RAISE EXCEPTION 'Expected all eight superseded Matters to be archived'; END IF;
END $$;
SELECT count(*) AS archived_replacement_targets FROM replacement_targets x JOIN demo_record_archives a ON a.record_id=x.id
 WHERE a.tenant_id='00000000-0000-4000-8000-000000000001'
   AND a.legal_entity_id='00000000-0000-4000-8000-000000000002'
   AND a.record_type='MATTER' AND a.restored_at IS NULL;
COMMIT;

-- Restoration uses migration 000089's attributed restoration fields on the
-- exact active archive receipt; never delete history or rewrite Matter states.
-- Assessment cancellation is intentionally separate: POST
-- /api/v1/vendor-assessments/01a0810e-00ad-7519-af83-aee0cb52017b/cancel
-- with the freshly read expected_version and a reason, by an authorized owner.
-- Its service commits the cancellation/event/outbox and revokes request access.

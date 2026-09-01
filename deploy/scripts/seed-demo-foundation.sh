#!/usr/bin/env bash
set -Eeuo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

psql -X "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -v demo_staff_email="${CLEARSIGHT_DEMO_STAFF_EMAIL:-}" >/dev/null <<'SQL'
BEGIN;
INSERT INTO tenants(id, slug, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'clearsight-demo', 'Clear Bank')
ON CONFLICT (id) DO NOTHING;

UPDATE tenants
SET name = 'Clear Bank'
WHERE id = '00000000-0000-4000-8000-000000000001'
  AND name IN ('ClearSight Demonstration Bank', 'Demo Bank', 'Clear Bank');

INSERT INTO legal_entities(id, tenant_id, code, name, jurisdiction)
VALUES (
  '00000000-0000-4000-8000-000000000002',
  '00000000-0000-4000-8000-000000000001',
  'BANK-NG', 'Clear Bank Nigeria', 'Nigeria'
)
ON CONFLICT (id) DO NOTHING;

UPDATE legal_entities
SET name = 'Clear Bank Nigeria'
WHERE id = '00000000-0000-4000-8000-000000000002'
  AND name IN ('Demonstration Bank Nigeria', 'Demo Bank Nigeria', 'Clear Bank Nigeria');

INSERT INTO principals(id, tenant_id, kind, external_ref, display_name)
VALUES
  ('00000000-0000-4000-8000-000000000101', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-cro', 'Chief Risk Officer'),
  ('00000000-0000-4000-8000-000000000102', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-cco', 'Chief Compliance Officer'),
  ('00000000-0000-4000-8000-000000000103', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-ciso', 'Chief Information Security Officer'),
  ('00000000-0000-4000-8000-000000000104', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-grc-admin', 'GRC Administrator'),
  ('00000000-0000-4000-8000-000000000105', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-system-admin', 'System Administrator'),
  ('00000000-0000-4000-8000-000000000106', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-auditor', 'Internal Auditor'),
  ('00000000-0000-4000-8000-000000000107', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-program-owner', 'Program Owner'),
  ('00000000-0000-4000-8000-000000000108', '00000000-0000-4000-8000-000000000001', 'PERSON', 'demo-evidence-respondent', 'Evidence Respondent')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_templates(id, tenant_id, code, name, description, responsibilities, capabilities, valid_from)
VALUES
  ('00000000-0000-4000-8000-000000000401', '00000000-0000-4000-8000-000000000001', 'CRO', 'Chief Risk Officer', 'Executive risk authorization and escalation.', ARRAY['AUTHORIZER','ESCALATION_OWNER'], ARRAY['read:all','governance:authorize'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000402', '00000000-0000-4000-8000-000000000001', 'CCO', 'Chief Compliance Officer', 'Compliance authorization and external sign-off.', ARRAY['AUTHORIZER','SIGNATORY','TRANSMITTER','ACKNOWLEDGEMENT_RECORDER','PROPOSER'], ARRAY['read:all','governance:authorize'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000403', '00000000-0000-4000-8000-000000000001', 'CISO', 'Chief Information Security Officer', 'Information security executive oversight.', ARRAY['ACCOUNTABLE_OWNER'], ARRAY['read:all'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000404', '00000000-0000-4000-8000-000000000001', 'GRC_ADMIN', 'GRC Administrator', 'Governance configuration administration.', ARRAY['PROPOSER'], ARRAY['configure:governance'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000405', '00000000-0000-4000-8000-000000000001', 'SYSTEM_ADMIN', 'System Administrator', 'System configuration administration.', ARRAY[]::text[], ARRAY['configure:system'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000406', '00000000-0000-4000-8000-000000000001', 'INTERNAL_AUDITOR', 'Internal Auditor', 'Independent review and challenge.', ARRAY['REVIEWER','INDEPENDENT_CHALLENGER'], ARRAY['read:all','review:evidence'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000407', '00000000-0000-4000-8000-000000000001', 'PROGRAM_OWNER', 'Program Owner', 'Accountable ownership and assigned implementation work.', ARRAY['ACCOUNTABLE_OWNER','PERFORMER'], ARRAY['manage:program','manage:matter'], '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000408', '00000000-0000-4000-8000-000000000001', 'EVIDENCE_RESPONDENT', 'Evidence Respondent', 'Assigned evidence collection work.', ARRAY['PERFORMER'], ARRAY['respond:evidence'], '2020-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO org_positions(id, tenant_id, legal_entity_id, code, title, function_name, occupant_principal_id, valid_from, department_path)
VALUES
  ('00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'CRO', 'Chief Risk Officer', 'Risk', '00000000-0000-4000-8000-000000000101', '2020-01-01T00:00:00Z', ARRAY['Risk']),
  ('00000000-0000-4000-8000-000000000302', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'CCO', 'Chief Compliance Officer', 'Compliance', '00000000-0000-4000-8000-000000000102', '2020-01-01T00:00:00Z', ARRAY['Compliance']),
  ('00000000-0000-4000-8000-000000000303', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'CISO', 'Chief Information Security Officer', 'Information Security', '00000000-0000-4000-8000-000000000103', '2020-01-01T00:00:00Z', ARRAY['Information Security']),
  ('00000000-0000-4000-8000-000000000304', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'GRC-ADMIN', 'GRC Administrator', 'Risk', '00000000-0000-4000-8000-000000000104', '2020-01-01T00:00:00Z', ARRAY['Risk','GRC']),
  ('00000000-0000-4000-8000-000000000305', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'SYSTEM-ADMIN', 'System Administrator', 'Technology', '00000000-0000-4000-8000-000000000105', '2020-01-01T00:00:00Z', ARRAY['Technology']),
  ('00000000-0000-4000-8000-000000000306', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'INTERNAL-AUDITOR', 'Internal Auditor', 'Internal Audit', '00000000-0000-4000-8000-000000000106', '2020-01-01T00:00:00Z', ARRAY['Internal Audit']),
  ('00000000-0000-4000-8000-000000000307', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'PROGRAM-OWNER', 'Program Owner', 'Risk', '00000000-0000-4000-8000-000000000107', '2020-01-01T00:00:00Z', ARRAY['Risk','Programs']),
  ('00000000-0000-4000-8000-000000000308', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000002', 'EVIDENCE-RESPONDENT', 'Evidence Respondent', 'Operations', '00000000-0000-4000-8000-000000000108', '2020-01-01T00:00:00Z', ARRAY['Operations'])
ON CONFLICT (id) DO NOTHING;

INSERT INTO position_role_bindings(id, tenant_id, position_id, role_template_id, scope, priority, valid_from)
VALUES
  ('00000000-0000-4000-8000-000000000501', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000301', '00000000-0000-4000-8000-000000000401', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000502', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000302', '00000000-0000-4000-8000-000000000402', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000503', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000303', '00000000-0000-4000-8000-000000000403', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000504', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000304', '00000000-0000-4000-8000-000000000404', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000505', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000305', '00000000-0000-4000-8000-000000000405', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000506', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000306', '00000000-0000-4000-8000-000000000406', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000507', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000307', '00000000-0000-4000-8000-000000000407', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000508', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000308', '00000000-0000-4000-8000-000000000408', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 100, '2020-01-01T00:00:00Z'),
  ('00000000-0000-4000-8000-000000000509', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000307', '00000000-0000-4000-8000-000000000408', '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}', 90, '2020-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

DO $identity_fixture$
BEGIN
  IF (SELECT count(*) FROM org_positions WHERE id BETWEEN '00000000-0000-4000-8000-000000000301'::uuid AND '00000000-0000-4000-8000-000000000308'::uuid AND tenant_id = '00000000-0000-4000-8000-000000000001' AND legal_entity_id = '00000000-0000-4000-8000-000000000002' AND valid_until IS NULL) <> 8 THEN
    RAISE EXCEPTION 'demo organizational positions differ from the managed fixture';
  END IF;
  IF (SELECT count(*) FROM role_templates WHERE id BETWEEN '00000000-0000-4000-8000-000000000401'::uuid AND '00000000-0000-4000-8000-000000000408'::uuid AND tenant_id = '00000000-0000-4000-8000-000000000001' AND valid_until IS NULL) <> 8 THEN
    RAISE EXCEPTION 'demo role templates differ from the managed fixture';
  END IF;
  IF (SELECT count(*) FROM position_role_bindings WHERE id BETWEEN '00000000-0000-4000-8000-000000000501'::uuid AND '00000000-0000-4000-8000-000000000509'::uuid AND tenant_id = '00000000-0000-4000-8000-000000000001' AND valid_until IS NULL) <> 9 THEN
    RAISE EXCEPTION 'demo position role bindings differ from the managed fixture';
  END IF;
  IF EXISTS (
    SELECT 1
    FROM (VALUES
      ('00000000-0000-4000-8000-000000000101'::uuid, 'demo-cro', 'Chief Risk Officer'),
      ('00000000-0000-4000-8000-000000000102'::uuid, 'demo-cco', 'Chief Compliance Officer'),
      ('00000000-0000-4000-8000-000000000103'::uuid, 'demo-ciso', 'Chief Information Security Officer'),
      ('00000000-0000-4000-8000-000000000104'::uuid, 'demo-grc-admin', 'GRC Administrator'),
      ('00000000-0000-4000-8000-000000000105'::uuid, 'demo-system-admin', 'System Administrator'),
      ('00000000-0000-4000-8000-000000000106'::uuid, 'demo-auditor', 'Internal Auditor'),
      ('00000000-0000-4000-8000-000000000107'::uuid, 'demo-program-owner', 'Program Owner'),
      ('00000000-0000-4000-8000-000000000108'::uuid, 'demo-evidence-respondent', 'Evidence Respondent')
    ) expected(id, external_ref, display_name)
    LEFT JOIN principals actual ON actual.id = expected.id
    WHERE actual.id IS NULL
       OR actual.tenant_id <> '00000000-0000-4000-8000-000000000001'::uuid
       OR actual.kind <> 'PERSON'
       OR actual.external_ref IS DISTINCT FROM expected.external_ref
       OR actual.display_name IS DISTINCT FROM expected.display_name
  ) THEN
    RAISE EXCEPTION 'demo principal mappings differ from the managed fixture';
  END IF;
  IF EXISTS (
    SELECT 1
    FROM (VALUES
      ('00000000-0000-4000-8000-000000000401'::uuid, 'CRO', ARRAY['AUTHORIZER','ESCALATION_OWNER']::text[], ARRAY['read:all','governance:authorize']::text[]),
      ('00000000-0000-4000-8000-000000000402'::uuid, 'CCO', ARRAY['AUTHORIZER','SIGNATORY','TRANSMITTER','ACKNOWLEDGEMENT_RECORDER','PROPOSER']::text[], ARRAY['read:all','governance:authorize']::text[]),
      ('00000000-0000-4000-8000-000000000403'::uuid, 'CISO', ARRAY['ACCOUNTABLE_OWNER']::text[], ARRAY['read:all']::text[]),
      ('00000000-0000-4000-8000-000000000404'::uuid, 'GRC_ADMIN', ARRAY['PROPOSER']::text[], ARRAY['configure:governance']::text[]),
      ('00000000-0000-4000-8000-000000000405'::uuid, 'SYSTEM_ADMIN', ARRAY[]::text[], ARRAY['configure:system']::text[]),
      ('00000000-0000-4000-8000-000000000406'::uuid, 'INTERNAL_AUDITOR', ARRAY['REVIEWER','INDEPENDENT_CHALLENGER']::text[], ARRAY['read:all','review:evidence']::text[]),
      ('00000000-0000-4000-8000-000000000407'::uuid, 'PROGRAM_OWNER', ARRAY['ACCOUNTABLE_OWNER','PERFORMER']::text[], ARRAY['manage:program','manage:matter']::text[]),
      ('00000000-0000-4000-8000-000000000408'::uuid, 'EVIDENCE_RESPONDENT', ARRAY['PERFORMER']::text[], ARRAY['respond:evidence']::text[])
    ) expected(id, code, responsibilities, capabilities)
    LEFT JOIN role_templates actual ON actual.id = expected.id
    WHERE actual.id IS NULL
       OR actual.tenant_id <> '00000000-0000-4000-8000-000000000001'::uuid
       OR actual.code <> expected.code
       OR actual.responsibilities IS DISTINCT FROM expected.responsibilities
       OR actual.capabilities IS DISTINCT FROM expected.capabilities
       OR actual.valid_until IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'demo role mappings differ from the managed fixture';
  END IF;
  IF EXISTS (
    SELECT 1
    FROM (VALUES
      ('00000000-0000-4000-8000-000000000301'::uuid, 'CRO', '00000000-0000-4000-8000-000000000101'::uuid),
      ('00000000-0000-4000-8000-000000000302'::uuid, 'CCO', '00000000-0000-4000-8000-000000000102'::uuid),
      ('00000000-0000-4000-8000-000000000303'::uuid, 'CISO', '00000000-0000-4000-8000-000000000103'::uuid),
      ('00000000-0000-4000-8000-000000000304'::uuid, 'GRC-ADMIN', '00000000-0000-4000-8000-000000000104'::uuid),
      ('00000000-0000-4000-8000-000000000305'::uuid, 'SYSTEM-ADMIN', '00000000-0000-4000-8000-000000000105'::uuid),
      ('00000000-0000-4000-8000-000000000306'::uuid, 'INTERNAL-AUDITOR', '00000000-0000-4000-8000-000000000106'::uuid),
      ('00000000-0000-4000-8000-000000000307'::uuid, 'PROGRAM-OWNER', '00000000-0000-4000-8000-000000000107'::uuid),
      ('00000000-0000-4000-8000-000000000308'::uuid, 'EVIDENCE-RESPONDENT', '00000000-0000-4000-8000-000000000108'::uuid)
    ) expected(id, code, occupant_principal_id)
    LEFT JOIN org_positions actual ON actual.id = expected.id
    WHERE actual.id IS NULL
       OR actual.tenant_id <> '00000000-0000-4000-8000-000000000001'::uuid
       OR actual.legal_entity_id <> '00000000-0000-4000-8000-000000000002'::uuid
       OR actual.code <> expected.code
       OR actual.occupant_principal_id IS DISTINCT FROM expected.occupant_principal_id
       OR actual.valid_until IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'demo position mappings differ from the managed fixture';
  END IF;
  IF EXISTS (
    SELECT 1
    FROM (VALUES
      ('00000000-0000-4000-8000-000000000501'::uuid, '00000000-0000-4000-8000-000000000301'::uuid, '00000000-0000-4000-8000-000000000401'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000502'::uuid, '00000000-0000-4000-8000-000000000302'::uuid, '00000000-0000-4000-8000-000000000402'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000503'::uuid, '00000000-0000-4000-8000-000000000303'::uuid, '00000000-0000-4000-8000-000000000403'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000504'::uuid, '00000000-0000-4000-8000-000000000304'::uuid, '00000000-0000-4000-8000-000000000404'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000505'::uuid, '00000000-0000-4000-8000-000000000305'::uuid, '00000000-0000-4000-8000-000000000405'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000506'::uuid, '00000000-0000-4000-8000-000000000306'::uuid, '00000000-0000-4000-8000-000000000406'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000507'::uuid, '00000000-0000-4000-8000-000000000307'::uuid, '00000000-0000-4000-8000-000000000407'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000508'::uuid, '00000000-0000-4000-8000-000000000308'::uuid, '00000000-0000-4000-8000-000000000408'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 100),
      ('00000000-0000-4000-8000-000000000509'::uuid, '00000000-0000-4000-8000-000000000307'::uuid, '00000000-0000-4000-8000-000000000408'::uuid, '{"legal_entity_id":"00000000-0000-4000-8000-000000000002"}'::jsonb, 90)
    ) expected(id, position_id, role_template_id, scope, priority)
    LEFT JOIN position_role_bindings actual ON actual.id = expected.id
    WHERE actual.id IS NULL
       OR actual.tenant_id <> '00000000-0000-4000-8000-000000000001'::uuid
       OR actual.position_id <> expected.position_id
       OR actual.role_template_id <> expected.role_template_id
       OR actual.scope IS DISTINCT FROM expected.scope
       OR actual.priority <> expected.priority
       OR actual.valid_until IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'demo position-role mappings differ from the managed fixture';
  END IF;
END
$identity_fixture$;

INSERT INTO scim_sources(id, tenant_id, code, token_hash, status)
SELECT '00000000-0000-4000-8000-000000000601', '00000000-0000-4000-8000-000000000001', 'CLEARSIGHT-DEMO-CONTACTS', decode('8f1a607f99c24bc89d4075b4c86d12dc8a00f52ded84bc076780c0d55e6a8f91', 'hex'), 'ACTIVE'
WHERE NULLIF(:'demo_staff_email', '') IS NOT NULL
ON CONFLICT (id) DO UPDATE SET status = 'ACTIVE', updated_at = clock_timestamp();

INSERT INTO scim_users(id, tenant_id, source_id, principal_id, external_id, user_name, active, deleted_at)
SELECT '00000000-0000-4000-8000-000000000602', '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000601', '00000000-0000-4000-8000-000000000107', 'demo-program-owner-contact', :'demo_staff_email', true, NULL
WHERE NULLIF(:'demo_staff_email', '') IS NOT NULL
ON CONFLICT (tenant_id, principal_id) DO UPDATE
SET source_id = EXCLUDED.source_id, external_id = EXCLUDED.external_id, user_name = EXCLUDED.user_name,
    active = true, deleted_at = NULL, updated_at = clock_timestamp();

DO $foundation$
DECLARE
  expected_definition jsonb := $definition${
    "rules": [
      {"id":"demo-program-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"PROGRAM","object_id":"*","responsibility":"AUTHORIZER","decision_type":"program.transition","min_materiality":0,"priority":200,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-accountable-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-reviewer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"REVIEWER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"AUTHORIZER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-signatory","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"SIGNATORY","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-transmitter","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"TRANSMITTER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-acknowledger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACKNOWLEDGEMENT_RECORDER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-performer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PERFORMER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-proposer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PROPOSER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-challenger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"INDEPENDENT_CHALLENGER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-escalation-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ESCALATION_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}}
    ]
  }$definition$::jsonb;
BEGIN
  IF EXISTS (
    SELECT 1 FROM routing_policies
    WHERE id = '00000000-0000-4000-8000-000000000201'
      AND (
        tenant_id <> '00000000-0000-4000-8000-000000000001'
        OR legal_entity_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000002'::uuid
        OR code <> 'CLEARSIGHT-DEMO-AUTHORITY'
      )
  ) THEN
    RAISE EXCEPTION 'demo authority policy id is already used by an incompatible policy';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policies
    WHERE tenant_id = '00000000-0000-4000-8000-000000000001'
      AND legal_entity_id = '00000000-0000-4000-8000-000000000002'
      AND code = 'CLEARSIGHT-DEMO-AUTHORITY'
      AND id <> '00000000-0000-4000-8000-000000000201'
  ) THEN
    RAISE EXCEPTION 'demo authority policy code is already used by an incompatible policy';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policy_versions
    WHERE id = '00000000-0000-4000-8000-000000000202'
      AND (
        policy_id <> '00000000-0000-4000-8000-000000000201'
        OR legal_entity_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000002'::uuid
        OR version <> 1
      )
  ) THEN
    RAISE EXCEPTION 'demo authority policy version id is already used by an incompatible version';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policy_versions
    WHERE policy_id = '00000000-0000-4000-8000-000000000201'
      AND version = 1
      AND id <> '00000000-0000-4000-8000-000000000202'
  ) THEN
    RAISE EXCEPTION 'demo authority policy version number is already used by an incompatible version';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policies
    WHERE id = '00000000-0000-4000-8000-000000000201'
      AND (
        tenant_id <> '00000000-0000-4000-8000-000000000001'
        OR legal_entity_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000002'::uuid
        OR code <> 'CLEARSIGHT-DEMO-AUTHORITY'
        OR name <> 'ClearSight demo authority routes'
        OR status <> 'ACTIVE'
        OR current_version NOT IN (1, 2)
        OR maker_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000104'::uuid
        OR checker_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000106'::uuid
        OR submitted_at IS NULL
        OR approved_at IS NULL
        OR retired_at IS NOT NULL
      )
  ) THEN
    RAISE EXCEPTION 'demo authority policy differs from the approved managed fixture';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policy_versions
    WHERE id = '00000000-0000-4000-8000-000000000202'
      AND (
        policy_id <> '00000000-0000-4000-8000-000000000201'
        OR legal_entity_id IS DISTINCT FROM '00000000-0000-4000-8000-000000000002'::uuid
        OR version <> 1
        OR definition IS DISTINCT FROM expected_definition
        OR checksum <> 'd315abab6729fac5611327a56aa0f3d4ed07aad2ba160106beb0ce7a3f99e91e'
        OR effective_from IS DISTINCT FROM '2020-01-01T00:00:00Z'::timestamptz
        OR effective_until IS NOT NULL
        OR created_by IS DISTINCT FROM '00000000-0000-4000-8000-000000000104'::uuid
        OR approved_by IS DISTINCT FROM '00000000-0000-4000-8000-000000000106'::uuid
        OR approved_at IS NULL
      )
  ) THEN
    RAISE EXCEPTION 'demo authority policy version differs from the approved managed fixture';
  END IF;
END
$foundation$;

INSERT INTO routing_policies(
  id, tenant_id, legal_entity_id, code, name, status, current_version,
  maker_id, checker_id, submitted_at, approved_at, version
)
VALUES (
  '00000000-0000-4000-8000-000000000201',
  '00000000-0000-4000-8000-000000000001',
  '00000000-0000-4000-8000-000000000002',
  'CLEARSIGHT-DEMO-AUTHORITY', 'ClearSight demo authority routes', 'ACTIVE', 1,
  '00000000-0000-4000-8000-000000000104',
  '00000000-0000-4000-8000-000000000106',
  clock_timestamp(), clock_timestamp(), 1
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO routing_policy_versions(
  id, policy_id, legal_entity_id, version, definition, checksum, effective_from,
  created_by, approved_by, approved_at
)
VALUES (
  '00000000-0000-4000-8000-000000000202',
  '00000000-0000-4000-8000-000000000201',
  '00000000-0000-4000-8000-000000000002',
  1,
  $definition${
    "rules": [
      {"id":"demo-program-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"PROGRAM","object_id":"*","responsibility":"AUTHORIZER","decision_type":"program.transition","min_materiality":0,"priority":200,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-accountable-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-reviewer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"REVIEWER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"AUTHORIZER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-signatory","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"SIGNATORY","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-transmitter","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"TRANSMITTER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-acknowledger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACKNOWLEDGEMENT_RECORDER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-performer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PERFORMER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-proposer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PROPOSER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-challenger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"INDEPENDENT_CHALLENGER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-escalation-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ESCALATION_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}}
    ]
  }$definition$::jsonb,
  'd315abab6729fac5611327a56aa0f3d4ed07aad2ba160106beb0ce7a3f99e91e',
  '2020-01-01T00:00:00Z'::timestamptz,
  '00000000-0000-4000-8000-000000000104',
  '00000000-0000-4000-8000-000000000106',
  clock_timestamp()
)
ON CONFLICT (policy_id, version) DO NOTHING;

INSERT INTO routing_policy_versions(
  id, policy_id, legal_entity_id, version, definition, checksum, effective_from,
  created_by, approved_by, approved_at
)
VALUES (
  '00000000-0000-4000-8000-000000000203',
  '00000000-0000-4000-8000-000000000201',
  '00000000-0000-4000-8000-000000000002',
  2,
  $definition${
    "rules": [
      {"id":"demo-program-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"PROGRAM","object_id":"*","responsibility":"AUTHORIZER","decision_type":"program.transition","min_materiality":0,"priority":200,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-accountable-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-reviewer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"REVIEWER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"AUTHORIZER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-signatory","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"SIGNATORY","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-transmitter","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"TRANSMITTER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-acknowledger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACKNOWLEDGEMENT_RECORDER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-performer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PERFORMER","min_materiality":0,"priority":100,"selector":{"kind":"ROLE","ref":"EVIDENCE_RESPONDENT"}},
      {"id":"demo-proposer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PROPOSER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-challenger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"INDEPENDENT_CHALLENGER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-escalation-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ESCALATION_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}}
    ]
  }$definition$::jsonb,
  '157b7a984f7930c08002715ebc320f7dd1b0f2eb986cc03c18c7ff346065ce9f',
  '2020-01-01T00:00:00Z'::timestamptz,
  '00000000-0000-4000-8000-000000000104',
  '00000000-0000-4000-8000-000000000106',
  clock_timestamp()
)
ON CONFLICT (policy_id, version) DO NOTHING;

UPDATE routing_policies
SET current_version = 2,
    version = CASE WHEN current_version = 1 THEN version + 1 ELSE version END,
    updated_at = CASE WHEN current_version = 1 THEN clock_timestamp() ELSE updated_at END
WHERE id = '00000000-0000-4000-8000-000000000201'
  AND current_version IN (1, 2);

SELECT refresh_effective_authority_routes('00000000-0000-4000-8000-000000000001');

DO $authority_v2$
DECLARE
  expected_definition jsonb := $definition${
    "rules": [
      {"id":"demo-program-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"PROGRAM","object_id":"*","responsibility":"AUTHORIZER","decision_type":"program.transition","min_materiality":0,"priority":200,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-accountable-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACCOUNTABLE_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000107"}},
      {"id":"demo-reviewer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"REVIEWER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-authorizer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"AUTHORIZER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}},
      {"id":"demo-signatory","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"SIGNATORY","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-transmitter","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"TRANSMITTER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-acknowledger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ACKNOWLEDGEMENT_RECORDER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-performer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PERFORMER","min_materiality":0,"priority":100,"selector":{"kind":"ROLE","ref":"EVIDENCE_RESPONDENT"}},
      {"id":"demo-proposer","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"PROPOSER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000102"}},
      {"id":"demo-challenger","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"INDEPENDENT_CHALLENGER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000106"}},
      {"id":"demo-escalation-owner","legal_entity_id":"00000000-0000-4000-8000-000000000002","object_type":"*","object_id":"*","responsibility":"ESCALATION_OWNER","min_materiality":0,"priority":100,"selector":{"kind":"PRINCIPAL","ref":"00000000-0000-4000-8000-000000000101"}}
    ]
  }$definition$::jsonb;
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM routing_policies
    WHERE id = '00000000-0000-4000-8000-000000000201'
      AND status = 'ACTIVE' AND current_version = 2
  ) OR NOT EXISTS (
    SELECT 1 FROM routing_policy_versions
    WHERE id = '00000000-0000-4000-8000-000000000203'
      AND policy_id = '00000000-0000-4000-8000-000000000201'
      AND version = 2
      AND definition IS NOT DISTINCT FROM expected_definition
      AND checksum = '157b7a984f7930c08002715ebc320f7dd1b0f2eb986cc03c18c7ff346065ce9f'
      AND approved_by = '00000000-0000-4000-8000-000000000106'
      AND approved_at IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'demo authority policy v2 differs from the approved managed fixture';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM effective_authority_routes
    WHERE tenant_id = '00000000-0000-4000-8000-000000000001'
      AND source_rule_id = 'demo-performer'
      AND policy_version = 'CLEARSIGHT-DEMO-AUTHORITY:v2'
      AND selector_kind = 'ROLE'
      AND selector_ref = 'EVIDENCE_RESPONDENT'
  ) THEN
    RAISE EXCEPTION 'demo performer route was not projected from authority policy v2';
  END IF;
  IF (
    SELECT count(DISTINCT op.occupant_principal_id)
    FROM role_templates rt
    JOIN position_role_bindings prb ON prb.role_template_id = rt.id AND prb.tenant_id = rt.tenant_id
    JOIN org_positions op ON op.id = prb.position_id AND op.tenant_id = prb.tenant_id
    WHERE rt.tenant_id = '00000000-0000-4000-8000-000000000001'
      AND rt.code = 'EVIDENCE_RESPONDENT'
      AND rt.valid_until IS NULL AND prb.valid_until IS NULL AND op.valid_until IS NULL
      AND op.occupant_principal_id IN (
        '00000000-0000-4000-8000-000000000107'::uuid,
        '00000000-0000-4000-8000-000000000108'::uuid
      )
  ) <> 2 THEN
    RAISE EXCEPTION 'demo performer route does not resolve both governed assignees';
  END IF;
END
$authority_v2$;
COMMIT;
SQL

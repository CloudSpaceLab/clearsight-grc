#!/usr/bin/env bash
set -Eeuo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

psql -X "$DATABASE_URL" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
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
      AND (tenant_id <> '00000000-0000-4000-8000-000000000001' OR code <> 'CLEARSIGHT-DEMO-AUTHORITY')
  ) THEN
    RAISE EXCEPTION 'demo authority policy id is already used by an incompatible policy';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policies
    WHERE tenant_id = '00000000-0000-4000-8000-000000000001'
      AND code = 'CLEARSIGHT-DEMO-AUTHORITY'
      AND id <> '00000000-0000-4000-8000-000000000201'
  ) THEN
    RAISE EXCEPTION 'demo authority policy code is already used by an incompatible policy';
  END IF;
  IF EXISTS (
    SELECT 1 FROM routing_policy_versions
    WHERE id = '00000000-0000-4000-8000-000000000202'
      AND (policy_id <> '00000000-0000-4000-8000-000000000201' OR version <> 1)
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
        OR code <> 'CLEARSIGHT-DEMO-AUTHORITY'
        OR name <> 'ClearSight demo authority routes'
        OR status <> 'ACTIVE'
        OR current_version <> 1
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
  id, tenant_id, code, name, status, current_version,
  maker_id, checker_id, submitted_at, approved_at, version
)
VALUES (
  '00000000-0000-4000-8000-000000000201',
  '00000000-0000-4000-8000-000000000001',
  'CLEARSIGHT-DEMO-AUTHORITY', 'ClearSight demo authority routes', 'ACTIVE', 1,
  '00000000-0000-4000-8000-000000000104',
  '00000000-0000-4000-8000-000000000106',
  clock_timestamp(), clock_timestamp(), 1
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO routing_policy_versions(
  id, policy_id, version, definition, checksum, effective_from,
  created_by, approved_by, approved_at
)
VALUES (
  '00000000-0000-4000-8000-000000000202',
  '00000000-0000-4000-8000-000000000201',
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

SELECT refresh_effective_authority_routes('00000000-0000-4000-8000-000000000001');
COMMIT;
SQL

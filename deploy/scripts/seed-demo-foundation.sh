#!/usr/bin/env bash
set -Eeuo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

psql -X "$DATABASE_URL" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
BEGIN;
INSERT INTO tenants(id, slug, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'clearsight-demo', 'ClearSight Demonstration Bank')
ON CONFLICT (id) DO NOTHING;

INSERT INTO legal_entities(id, tenant_id, code, name, jurisdiction)
VALUES (
  '00000000-0000-4000-8000-000000000002',
  '00000000-0000-4000-8000-000000000001',
  'BANK-NG', 'Demonstration Bank Nigeria', 'Nigeria'
)
ON CONFLICT (id) DO NOTHING;

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
COMMIT;
SQL

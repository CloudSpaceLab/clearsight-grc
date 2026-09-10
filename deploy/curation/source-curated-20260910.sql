-- Operator-only Clear Bank demo reconciliation. Run after migration 000089,
-- a verified backup, and CLEARSIGHT_DEMO_SEED_MODE=manual. No rows are deleted
-- and no bank lifecycle, assessment conclusion or response is changed.
\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('clearsight-source-curation-20260910'));
DO $$ BEGIN
 IF current_database()<>'clearsight' OR NOT EXISTS (
  SELECT 1 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id
  WHERE t.id='00000000-0000-4000-8000-000000000001' AND t.slug='clearsight-demo'
  AND le.id='00000000-0000-4000-8000-000000000002'
 ) THEN RAISE EXCEPTION 'Exact hosted demo scope required'; END IF;
END $$;

CREATE TEMP TABLE curation_vendor_targets(id uuid PRIMARY KEY, name text NOT NULL, source text NOT NULL) ON COMMIT DROP;
INSERT INTO curation_vendor_targets VALUES
 ('01a08695-fef3-73b3-954d-b861c3cfa333','ArchiveGuard Records Limited','reference_data'),
 ('01a060b8-ca2c-770e-86eb-19be96e3f1c4','ClearSight SMTP Acceptance Vendor 1788329119 Ltd','hosted-acceptance'),
 ('01a06eb6-a5ea-77d9-997f-9f5e44718e0e','Northstar Infrastructure Services Limited','reference_data'),
 ('01a08245-d8aa-7149-a092-ec6a3f98f8e1','Northstar Infrastructure Services Limited','fictional_document_samples_v1'),
 ('01a053e7-5819-705f-9c26-41d48b43a8fe','Northstar Payments Acceptance Test Ltd','hosted-email-acceptance'),
 ('01a08695-fee5-7ff2-a325-ea458659ea75','Paywave Transaction Services Limited','reference_data'),
 ('01a08695-fef8-7132-a155-027bc5002008','PeopleLink Payroll Services Limited','reference_data'),
 ('01a08695-fefd-76cb-bb45-069dc8e37046','Sentinel Collections Technology Limited','reference_data');
DO $$ BEGIN
 IF (SELECT count(*) FROM curation_vendor_targets x JOIN third_parties p ON p.id=x.id AND p.legal_name=x.name AND p.source_id=x.source WHERE p.tenant_id='00000000-0000-4000-8000-000000000001')<>8
 THEN RAISE EXCEPTION 'Vendor fixture identity changed; reconcile before curation'; END IF;
END $$;

CREATE TEMP TABLE curation_records(kind text NOT NULL,id uuid NOT NULL,reason text NOT NULL,PRIMARY KEY(kind,id)) ON COMMIT DROP;
INSERT INTO curation_records
 SELECT 'VENDOR_RELATIONSHIP',r.id,'Out-of-scope generic or acceptance-test vendor fixture; source-curated demo requested 10 September 2026.'
 FROM third_party_relationships r JOIN curation_vendor_targets x ON x.id=r.vendor_id
 WHERE r.tenant_id='00000000-0000-4000-8000-000000000001' AND r.legal_entity_id='00000000-0000-4000-8000-000000000002';
INSERT INTO curation_records
 SELECT 'VENDOR_RELATIONSHIP',r.id,'Earlier duplicate Cloudspace OEM sample episode; retain history. Current sample relationship: 01a0810d-9a4a-7a9e-a3c2-abf0727927eb.'
 FROM third_party_relationships r JOIN third_parties p ON p.id=r.vendor_id AND p.tenant_id=r.tenant_id
 WHERE r.id='01a0411f-9267-766d-8cb6-e0b7158feeed' AND r.tenant_id='00000000-0000-4000-8000-000000000001'
 AND r.legal_entity_id='00000000-0000-4000-8000-000000000002' AND p.legal_name='Cloudspace Technologies Ltd' AND r.service_name='OEM';

INSERT INTO curation_records
 SELECT 'MATTER',m.id,'Out-of-scope acceptance-test or superseded duplicate sample work; source-curated demo requested 10 September 2026.'
 FROM matters m WHERE m.tenant_id='00000000-0000-4000-8000-000000000001' AND m.legal_entity_id='00000000-0000-4000-8000-000000000002'
 AND m.id IN (
 '01a053e7-f81f-7abe-83da-85f12f73a6c0','01a060ba-4426-7a77-95d5-54ffd86bc654',
 '01a06164-22ee-77a9-8bbd-760d24298782','01a08245-dac3-7cba-ae38-2241740cead0',
 '01a05a63-f4da-7e7e-b3a9-36cb6e7db92c','01a072eb-fa22-7183-886f-90733c2f17cc',
 '01a04fd2-3f67-7157-ab5e-1f3fd3b5194f');
INSERT INTO curation_records
 SELECT 'MATTER',m.id,'Synthetic oversight history excluded from the source-curated demo; original closed outcome retained.'
 FROM matters m WHERE m.tenant_id='00000000-0000-4000-8000-000000000001' AND m.legal_entity_id='00000000-0000-4000-8000-000000000002'
 AND m.trigger_key IN (
 'reference:oversight:audit-access-review','reference:oversight:control-gap-logging','reference:oversight:vendor-access',
 'reference:oversight:vendor-address','reference:oversight:vendor-certification','reference:oversight:vendor-payment-controls','reference:oversight:vendor-resilience');

DO $$ BEGIN
 IF (SELECT count(*) FROM curation_records WHERE kind='VENDOR_RELATIONSHIP')<>9 OR (SELECT count(*) FROM curation_records WHERE kind='MATTER')<>14
 THEN RAISE EXCEPTION 'Curation population changed; expected 9 relationships and 14 sample Matters'; END IF;
 IF NOT EXISTS(SELECT 1 FROM third_party_relationships WHERE id='01a0810d-9a4a-7a9e-a3c2-abf0727927eb' AND tenant_id='00000000-0000-4000-8000-000000000001' AND legal_entity_id='00000000-0000-4000-8000-000000000002')
 THEN RAISE EXCEPTION 'Current Cloudspace sample relationship missing'; END IF;
END $$;

INSERT INTO demo_record_archives(tenant_id,legal_entity_id,record_type,record_id,reason,source_manifest,archived_by,archived_at)
 SELECT '00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',kind,id,reason,
 'deploy/curation/source-curated-20260910.sql','00000000-0000-4000-8000-000000000105',now() FROM curation_records
 ON CONFLICT(tenant_id,legal_entity_id,record_type,record_id) WHERE restored_at IS NULL DO NOTHING;
SELECT record_type,count(*) AS archived_records FROM demo_record_archives
 WHERE tenant_id='00000000-0000-4000-8000-000000000001' AND source_manifest='deploy/curation/source-curated-20260910.sql' AND restored_at IS NULL GROUP BY record_type;
COMMIT;

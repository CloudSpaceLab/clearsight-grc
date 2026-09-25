-- Exclude six synthetic scoring acceptance fixtures from active demo lists.
-- Preserve all original forms, responses and assessments for audit retrieval.
\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('clearsight-demo-scoring-curation-20260925'));
DO $$ BEGIN
 IF current_database()<>'clearsight' OR NOT EXISTS (
  SELECT 1 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id
  WHERE t.id='00000000-0000-4000-8000-000000000001' AND t.slug='clearsight-demo'
  AND le.id='00000000-0000-4000-8000-000000000002'
 ) THEN RAISE EXCEPTION 'Exact hosted demo scope required'; END IF;
END $$;

CREATE TEMP TABLE scoring_targets(id uuid PRIMARY KEY, title text NOT NULL) ON COMMIT DROP;
INSERT INTO scoring_targets VALUES
 ('01a072eb-f930-7885-a0cf-ef28a05d31d4','Scoring acceptance — borderline'),
 ('01a072eb-f8e9-752a-9c53-0f8eacf20fbb','Scoring acceptance — good'),
 ('01a072eb-f94f-75aa-aa91-5fec84c517ef','Scoring acceptance — poor'),
 ('01a072eb-f9dc-71a0-a3a0-65db5369ad48','Scoring acceptance — post-policy-good'),
 ('01a072eb-f9f4-7ec5-b5de-fb59aa7414ae','Scoring acceptance — post-policy-poor'),
 ('01a072f6-32d6-7c12-abf5-e66dfcb17e37','Scoring acceptance — post-policy-poor-same-episode-reconcile');

DO $$ BEGIN
 IF (SELECT count(*) FROM scoring_targets x JOIN capture_form_distributions d ON d.id=x.id AND d.title=x.title
     WHERE d.tenant_id='00000000-0000-4000-8000-000000000001'
       AND d.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND d.form_template_id='01a072eb-f8d2-75cb-af9d-c4fb2d896f77'
       AND d.subject_type='PROGRAM' AND d.subject_id='019ff790-a36f-7f33-8849-9e57ac62e62a')<>6
 THEN RAISE EXCEPTION 'Scoring fixture identity changed; reconcile before curation'; END IF;
 IF (SELECT count(*) FROM capture_form_distributions d
     WHERE d.tenant_id='00000000-0000-4000-8000-000000000001'
       AND d.legal_entity_id='00000000-0000-4000-8000-000000000002'
       AND d.title LIKE 'Scoring acceptance%')<>6
 THEN RAISE EXCEPTION 'Additional scoring fixtures found; reconcile before curation'; END IF;
END $$;

INSERT INTO demo_form_distribution_archives(distribution_id,tenant_id,legal_entity_id,reason,source_manifest,archived_by)
 SELECT x.id,d.tenant_id,d.legal_entity_id,
 'Synthetic scoring acceptance fixture is outside the supplied bank sample data.',
 'deploy/curation/scoring-acceptance-20260925.sql',
 '00000000-0000-4000-8000-000000000105'
 FROM scoring_targets x JOIN capture_form_distributions d ON d.id=x.id
 ON CONFLICT (distribution_id) DO NOTHING;

DO $$ BEGIN
 IF (SELECT count(*) FROM demo_form_distribution_archives archive
     JOIN scoring_targets x ON x.id=archive.distribution_id
     WHERE archive.restored_at IS NULL
       AND archive.tenant_id='00000000-0000-4000-8000-000000000001'
       AND archive.legal_entity_id='00000000-0000-4000-8000-000000000002')<>6
 THEN RAISE EXCEPTION 'Six active scoring archive receipts required'; END IF;
END $$;

SELECT count(*) AS archived_scoring_fixtures FROM demo_form_distribution_archives archive
 JOIN scoring_targets x ON x.id=archive.distribution_id WHERE archive.restored_at IS NULL;
COMMIT;

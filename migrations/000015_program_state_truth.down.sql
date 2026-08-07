BEGIN;

DROP TRIGGER IF EXISTS programs_preserve_period_on_resume_trg ON programs;
DROP FUNCTION IF EXISTS preserve_program_period_on_resume();
DROP TRIGGER IF EXISTS evidence_assessments_freshness_trg ON evidence_assessments;
DROP FUNCTION IF EXISTS enforce_evidence_assessment_freshness();
ALTER TABLE evidence_assessments
  DROP CONSTRAINT IF EXISTS evidence_assessments_validity_order_ck;

COMMIT;

BEGIN;

ALTER TABLE evidence_assessments
  ADD CONSTRAINT evidence_assessments_validity_order_ck
  CHECK (valid_until IS NULL OR assessed_at < valid_until);

CREATE OR REPLACE FUNCTION enforce_evidence_assessment_freshness()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  freshness integer;
  maximum_valid_until timestamptz;
BEGIN
  SELECT ec.freshness_minutes
  INTO freshness
  FROM evidence_contracts ec
  WHERE ec.tenant_id=NEW.tenant_id
    AND ec.program_id=NEW.program_id
    AND ec.id=NEW.contract_id;

  IF freshness IS NULL THEN
    RAISE EXCEPTION 'evidence contract does not exist for assessment';
  END IF;

  maximum_valid_until := NEW.assessed_at + make_interval(mins => freshness);
  IF NEW.valid_until IS NULL OR NEW.valid_until > maximum_valid_until THEN
    NEW.valid_until := maximum_valid_until;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER evidence_assessments_freshness_trg
BEFORE INSERT OR UPDATE OF contract_id, assessed_at, valid_until ON evidence_assessments
FOR EACH ROW EXECUTE FUNCTION enforce_evidence_assessment_freshness();

CREATE OR REPLACE FUNCTION preserve_program_period_on_resume()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF OLD.status='PAUSED' AND NEW.status='ACTIVE'
     AND OLD.effective_until IS NOT NULL AND NEW.effective_until IS NULL THEN
    NEW.effective_until := OLD.effective_until;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER programs_preserve_period_on_resume_trg
BEFORE UPDATE OF status,effective_until ON programs
FOR EACH ROW EXECUTE FUNCTION preserve_program_period_on_resume();

COMMIT;

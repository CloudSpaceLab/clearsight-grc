BEGIN;

-- Reuse the existing governed clause/Requirement relation for proposals.
-- Pending/rejected records are audit facts, never approved crosswalks or scoring units.
ALTER TABLE framework_clause_requirement_links
  ALTER COLUMN reviewed_by DROP NOT NULL,
  ALTER COLUMN reviewed_at DROP NOT NULL,
  ALTER COLUMN effective_from DROP NOT NULL,
  ADD COLUMN status text NOT NULL DEFAULT 'APPROVED'
    CHECK (status IN ('PENDING','APPROVED','REJECTED')),
  ADD COLUMN proposed_at timestamptz,
  ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0);

-- Existing reviewed links retain their original approval timestamps.
UPDATE framework_clause_requirement_links
SET proposed_at=LEAST(created_at,reviewed_at);
ALTER TABLE framework_clause_requirement_links
  ALTER COLUMN proposed_at SET NOT NULL,
  ADD CONSTRAINT framework_clause_mapping_decision_check CHECK (
    (status='PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL
      AND effective_from IS NULL AND retired_at IS NULL)
    OR (status='APPROVED' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
      AND effective_from IS NOT NULL AND effective_from >= reviewed_at)
    OR (status='REJECTED' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
      AND effective_from IS NULL AND retired_at IS NULL)
  ),
  ADD CONSTRAINT framework_clause_mapping_review_dates_check CHECK (
    reviewed_at IS NULL OR reviewed_at >= proposed_at
  );

-- A rejected proposal does not prevent a newly reviewed replacement.
DROP INDEX framework_clause_requirement_links_current_uq;
CREATE UNIQUE INDEX framework_clause_requirement_links_current_uq
ON framework_clause_requirement_links (
  tenant_id,legal_entity_id,adoption_id,clause_id,requirement_id
)
WHERE retired_at IS NULL AND status IN ('PENDING','APPROVED');

CREATE INDEX framework_clause_mapping_review_queue_idx
ON framework_clause_requirement_links (
  tenant_id,legal_entity_id,program_id,adoption_id,status,proposed_at DESC,id
)
WHERE status='PENDING';

-- A direct SQL update cannot silently bypass the Program-event decision path.
-- SQL validates the same scope/effective facts even if a privileged writer errs.
CREATE OR REPLACE FUNCTION guard_framework_clause_requirement_link()
RETURNS trigger LANGUAGE plpgsql AS $framework_guard$
DECLARE
  at_time timestamptz;
BEGIN
  IF TG_OP='UPDATE' THEN
    IF OLD.status <> 'PENDING' OR NEW.status NOT IN ('APPROVED','REJECTED')
       OR OLD.id <> NEW.id OR OLD.tenant_id <> NEW.tenant_id
       OR OLD.legal_entity_id <> NEW.legal_entity_id
       OR OLD.program_id <> NEW.program_id OR OLD.adoption_id <> NEW.adoption_id
       OR OLD.edition_id <> NEW.edition_id OR OLD.clause_id <> NEW.clause_id
       OR OLD.requirement_id <> NEW.requirement_id OR OLD.mapped_by <> NEW.mapped_by
       OR OLD.proposed_at <> NEW.proposed_at OR OLD.mapping_kind <> NEW.mapping_kind
       OR OLD.rationale <> NEW.rationale OR NEW.version <> OLD.version+1 THEN
      RAISE EXCEPTION 'framework mapping decisions require a pending unchanged proposal';
    END IF;
  ELSIF NEW.status <> 'PENDING' THEN
    RAISE EXCEPTION 'framework mapping must be proposed before approval';
  END IF;

  -- Rejection must remain possible after the source, adoption or
  -- Requirement becomes inactive. Rejecting cannot create effective coverage.
  IF NEW.status='REJECTED' THEN
    RETURN NEW;
  END IF;
  at_time := CASE WHEN NEW.status='APPROVED' THEN NEW.effective_from
                  ELSE NEW.proposed_at END;
  IF NOT EXISTS (
    SELECT 1
    FROM program_framework_adoptions a
    JOIN program_requirements req
      ON req.id=NEW.requirement_id
     AND req.tenant_id=a.tenant_id
     AND req.program_id=a.program_id
    WHERE a.id=NEW.adoption_id
      AND a.tenant_id=NEW.tenant_id
      AND a.legal_entity_id=NEW.legal_entity_id
      AND a.program_id=NEW.program_id
      AND a.edition_id=NEW.edition_id
      AND a.status='ACTIVE'
      AND a.effective_from <= at_time
      AND req.status='APPROVED'
      AND req.effective_from <= at_time
      AND (req.effective_until IS NULL OR at_time < req.effective_until)
  ) THEN
    RAISE EXCEPTION 'framework mapping requires active adoption and approved effective Requirement';
  END IF;
  RETURN NEW;
END;
$framework_guard$;

DROP TRIGGER framework_clause_requirement_link_guard
ON framework_clause_requirement_links;
CREATE TRIGGER framework_clause_requirement_link_guard
BEFORE INSERT OR UPDATE OF status,reviewed_by,reviewed_at,effective_from,version
ON framework_clause_requirement_links
FOR EACH ROW EXECUTE FUNCTION guard_framework_clause_requirement_link();

COMMIT;

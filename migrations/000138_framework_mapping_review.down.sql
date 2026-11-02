BEGIN;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM framework_clause_requirement_links LIMIT 1) THEN
    RAISE EXCEPTION 'cannot downgrade existing framework clause mapping review history';
  END IF;
END;
$$;

DROP TRIGGER framework_clause_requirement_link_guard ON framework_clause_requirement_links;
DROP INDEX framework_clause_mapping_review_queue_idx;
DROP INDEX framework_clause_requirement_links_current_uq;
ALTER TABLE framework_clause_requirement_links
  DROP CONSTRAINT framework_clause_mapping_decision_check,
  DROP CONSTRAINT framework_clause_mapping_review_dates_check,
  DROP COLUMN status,
  DROP COLUMN proposed_at,
  DROP COLUMN version,
  ALTER COLUMN reviewed_by SET NOT NULL,
  ALTER COLUMN reviewed_at SET NOT NULL,
  ALTER COLUMN effective_from SET NOT NULL;
CREATE UNIQUE INDEX framework_clause_requirement_links_current_uq
ON framework_clause_requirement_links (
  tenant_id,legal_entity_id,adoption_id,clause_id,requirement_id
) WHERE retired_at IS NULL;

CREATE OR REPLACE FUNCTION guard_framework_clause_requirement_link()
RETURNS trigger LANGUAGE plpgsql AS $framework_guard$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM program_framework_adoptions a
    JOIN program_requirements req
      ON req.id=NEW.requirement_id AND req.tenant_id=a.tenant_id
     AND req.program_id=a.program_id
    WHERE a.id=NEW.adoption_id AND a.tenant_id=NEW.tenant_id
      AND a.legal_entity_id=NEW.legal_entity_id
      AND a.program_id=NEW.program_id AND a.edition_id=NEW.edition_id
      AND a.status='ACTIVE' AND a.effective_from<=NEW.effective_from
      AND a.reviewed_at<=NEW.reviewed_at AND req.status='APPROVED'
      AND req.effective_from<=NEW.effective_from
      AND (req.effective_until IS NULL OR NEW.effective_from<req.effective_until)
  ) THEN
    RAISE EXCEPTION 'framework clause mapping requires an active approved Program adoption and Requirement';
  END IF;
  RETURN NEW;
END;
$framework_guard$;
CREATE TRIGGER framework_clause_requirement_link_guard
BEFORE INSERT ON framework_clause_requirement_links
FOR EACH ROW EXECUTE FUNCTION guard_framework_clause_requirement_link();

COMMIT;

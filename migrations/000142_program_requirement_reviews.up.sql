BEGIN;

-- A review is scoped to one existing draft Requirement. Existing Requirements
-- retain their status; this migration never marks historical data as reviewed.
CREATE TABLE program_requirement_reviews (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  tenant_id uuid NOT NULL,
  program_id uuid NOT NULL,
  requirement_id uuid NOT NULL,
  proposed_by uuid NOT NULL,
  proposal_rationale text NOT NULL CHECK (length(btrim(proposal_rationale)) BETWEEN 1 AND 2000),
  proposed_at timestamptz NOT NULL,
  reviewed_by uuid,
  review_rationale text,
  reviewed_at timestamptz,
  status text NOT NULL CHECK (status IN ('PENDING','APPROVED','REJECTED')),
  version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
  FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
  FOREIGN KEY (proposed_by,tenant_id) REFERENCES principals(id,tenant_id),
  FOREIGN KEY (reviewed_by,tenant_id) REFERENCES principals(id,tenant_id),
  CONSTRAINT program_requirement_review_independent CHECK (reviewed_by IS NULL OR reviewed_by <> proposed_by),
  CONSTRAINT program_requirement_review_lifecycle CHECK (
    (status='PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL AND review_rationale IS NULL)
    OR (status IN ('APPROVED','REJECTED') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
        AND reviewed_at >= proposed_at AND review_rationale IS NOT NULL AND length(btrim(review_rationale)) BETWEEN 1 AND 2000)
  )
);
CREATE UNIQUE INDEX program_requirement_review_one_pending
  ON program_requirement_reviews (tenant_id,program_id,requirement_id) WHERE status='PENDING';
CREATE INDEX program_requirement_review_queue
  ON program_requirement_reviews (tenant_id,program_id,status,proposed_at DESC,id);

CREATE OR REPLACE FUNCTION guard_program_requirement_review() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP='DELETE' THEN
    RAISE EXCEPTION 'requirement review history cannot be deleted';
  END IF;
  IF TG_OP='INSERT' THEN
    IF NEW.status <> 'PENDING' OR NEW.version <> 1 THEN
      RAISE EXCEPTION 'requirement review must start pending';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM program_requirements r
      WHERE r.id=NEW.requirement_id AND r.tenant_id=NEW.tenant_id
        AND r.program_id=NEW.program_id AND r.status='DRAFT') THEN
      RAISE EXCEPTION 'requirement review requires a draft';
    END IF;
  ELSE
    IF OLD.status <> 'PENDING' OR NEW.status NOT IN ('APPROVED','REJECTED')
       OR NEW.id<>OLD.id OR NEW.tenant_id<>OLD.tenant_id OR NEW.program_id<>OLD.program_id
       OR NEW.requirement_id<>OLD.requirement_id OR NEW.proposed_by<>OLD.proposed_by
       OR NEW.proposed_at<>OLD.proposed_at OR NEW.proposal_rationale<>OLD.proposal_rationale
       OR NEW.version<>OLD.version+1 THEN
      RAISE EXCEPTION 'requirement review decision requires an unchanged pending proposal';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM program_requirements r
      WHERE r.id=NEW.requirement_id AND r.tenant_id=NEW.tenant_id
        AND r.program_id=NEW.program_id AND r.status='DRAFT') THEN
      RAISE EXCEPTION 'requirement review decision requires a draft';
    END IF;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER program_requirement_review_guard BEFORE INSERT OR UPDATE OR DELETE
ON program_requirement_reviews FOR EACH ROW EXECUTE FUNCTION guard_program_requirement_review();
COMMIT;




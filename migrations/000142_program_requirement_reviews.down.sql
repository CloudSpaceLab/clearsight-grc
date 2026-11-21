BEGIN;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM program_requirement_reviews LIMIT 1) THEN
    RAISE EXCEPTION 'Cannot remove requirement reviews while review records exist';
  END IF;
END $$;
DROP TABLE program_requirement_reviews;
DROP FUNCTION guard_program_requirement_review();
COMMIT;


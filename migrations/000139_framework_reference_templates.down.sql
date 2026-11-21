BEGIN;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM framework_reference_templates LIMIT 1) THEN
    RAISE EXCEPTION 'Cannot remove non-empty framework reference templates';
  END IF;
END $$;
DROP TABLE framework_reference_guidance_areas;
DROP TABLE framework_reference_guidance;
DROP TABLE framework_reference_templates;
COMMIT;

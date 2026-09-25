BEGIN;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM demo_form_distribution_archives) THEN
    RAISE EXCEPTION 'Demo form archive history must be retained';
  END IF;
END $$;
DROP TRIGGER demo_form_distribution_archive_immutable ON demo_form_distribution_archives;
DROP FUNCTION protect_demo_form_distribution_archive();
DROP TABLE demo_form_distribution_archives;
COMMIT;

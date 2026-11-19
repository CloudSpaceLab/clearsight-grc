BEGIN;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM framework_starter_requirement_links l WHERE NOT EXISTS(SELECT 1 FROM framework_starter_requirements r WHERE r.template_code=l.template_code AND r.requirement_code=l.requirement_code)) THEN RAISE EXCEPTION 'Bank-defined starter requirements exist; retain maintenance support'; END IF; END $$;
ALTER TABLE framework_starter_requirement_links DROP CONSTRAINT framework_starter_template_fk;
ALTER TABLE framework_starter_requirement_links ADD FOREIGN KEY(template_code,requirement_code) REFERENCES framework_starter_requirements(template_code,requirement_code);
COMMIT;

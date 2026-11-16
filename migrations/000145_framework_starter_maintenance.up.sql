BEGIN;
-- Bank-defined starter requirements share the scoped library, not the global seed.
DO $$ DECLARE constraint_name text; BEGIN
 SELECT conname INTO constraint_name FROM pg_constraint WHERE conrelid='framework_starter_requirement_links'::regclass AND confrelid='framework_starter_requirements'::regclass AND contype='f';
 IF constraint_name IS NOT NULL THEN EXECUTE format('ALTER TABLE framework_starter_requirement_links DROP CONSTRAINT %I',constraint_name); END IF;
END $$;
ALTER TABLE framework_starter_requirement_links ADD CONSTRAINT framework_starter_template_fk FOREIGN KEY(template_code) REFERENCES framework_reference_templates(template_code);
-- Source identifiers verified against NIST CSWP 29. Starter prose is not NIST text.
UPDATE framework_starter_requirements SET reference_label='NIST CSF 2.0 - Function '||CASE requirement_code WHEN 'GOVERN' THEN 'GV' WHEN 'IDENTIFY' THEN 'ID' WHEN 'PROTECT' THEN 'PR' WHEN 'DETECT' THEN 'DE' WHEN 'RESPOND' THEN 'RS' WHEN 'RECOVER' THEN 'RC' END WHERE template_code='NIST_CSF_2';
UPDATE framework_starter_requirement_links l SET reference_label=r.reference_label FROM framework_starter_requirements r WHERE l.template_code=r.template_code AND l.requirement_code=r.requirement_code AND l.template_code='NIST_CSF_2' AND l.reference_label='Starter assessment area; clause mapping requires review';
-- Correct an encoding defect in non-authoritative starter metadata only.
-- Immutable Forms revisions and saved adoption snapshots remain untouched.
UPDATE framework_starter_requirements SET reference_label='ISO/IEC 27001:2022 - access governance; verify detailed control mapping against licensed text' WHERE template_code='ISO_27001_2022' AND requirement_code='PRIVILEGED_ACCESS';
UPDATE framework_starter_requirement_links SET reference_label='ISO/IEC 27001:2022 - access governance; verify detailed control mapping against licensed text' WHERE template_code='ISO_27001_2022' AND requirement_code='PRIVILEGED_ACCESS';
COMMIT;

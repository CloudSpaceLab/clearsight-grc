BEGIN;
-- Additive starter upgrade: the original eight template identities remain unchanged.
-- Original bank prompts only; no standards text or approval decisions are installed.
CREATE TABLE framework_starter_requirements (
 template_code text NOT NULL REFERENCES framework_reference_templates(template_code),
 requirement_code text NOT NULL, title text NOT NULL, statement text NOT NULL,
 reference_label text NOT NULL, group_id text NOT NULL, ordinal integer NOT NULL,
 PRIMARY KEY(template_code,requirement_code), UNIQUE(template_code,group_id)
);
INSERT INTO framework_starter_requirements
SELECT template_code, area_code, title,
 'Maintain and review ' || lower(title) || ' for the Program scope. Record accountable owners, review the supporting evidence and track unresolved exceptions.',
 'Starter assessment area; clause mapping requires review', lower(area_code), ordinal
FROM framework_reference_guidance_areas
ON CONFLICT DO NOTHING;
INSERT INTO framework_starter_requirements VALUES
('ISO_27001_2022','PRIVILEGED_ACCESS','Review privileged digital banking access',
 'Review privileged access to digital banking services at the frequency agreed by the bank. Remove unnecessary access, resolve exceptions and retain service-owner sign-off.',
 'ISO/IEC 27001:2022 â€” access governance; verify detailed control mapping against licensed text',
 'privileged_access',50) ON CONFLICT DO NOTHING;


UPDATE framework_starter_requirements r SET statement=v.statement FROM (VALUES ('ISO_27001_2022',1,'Document in-scope banking services, information, teams and suppliers. Assign security objectives and accountable owners and review changes with management.'),('ISO_27001_2022',2,'Assess risks to customer information and critical services using approved bank criteria. Assign treatment owners, record accepted exceptions and review overdue actions.'),('ISO_27001_2022',3,'Maintain a Statement of Applicability with selected controls, owners, implementation status and reasons for exclusions. Obtain approval for changes.'),('ISO_27001_2022',4,'Review monitoring results and internal audit findings with management. Assign corrective actions, verify effectiveness and retain closure evidence.'),('NIST_CSF_2',1,'Maintain cybersecurity risk strategy, decision rights, policy exceptions and supplier oversight. Record management decisions for the Program scope.'),('NIST_CSF_2',2,'Maintain inventories of banking services, data, assets and suppliers. Review cybersecurity risk when dependencies change and assign improvement owners.'),('NIST_CSF_2',3,'Review privileged access, data protection, training and platform safeguards for in-scope services. Track exceptions with owners and remediation dates.'),('NIST_CSF_2',4,'Maintain detection coverage for critical services. Review alerts, retain investigation records and escalate monitoring gaps.'),('NIST_CSF_2',5,'Maintain and exercise incident response roles and playbooks. Record containment, communications decisions and unresolved actions.'),('NIST_CSF_2',6,'Test recovery of critical services against bank recovery objectives. Retain restoration results and verify corrective actions.'),('PCI_DSS_4_0_1',1,'Maintain payment data flows, system inventories and third-party connections. Record scope changes and obtain assessor input.'),('PCI_DSS_4_0_1',2,'Review payment data encryption and access roles. Remove unnecessary privileges and track protection exceptions.'),('PCI_DSS_4_0_1',3,'Review payment security logs, vulnerability scans and penetration tests. Assign remediation owners and retain retest evidence.'),('PCI_DSS_4_0_1',4,'Agree the applicable validation route with payment stakeholders and the assessor. Maintain responsibilities, policies and an evidence register.'),('GDPR_2016_679',1,'Record processing purposes, personal data, recipients, retention periods and lawful bases. Review records when banking services change.'),('GDPR_2016_679',2,'Log individual rights requests, verify identity and track applicable deadlines. Record restriction and refusal decisions.'),('GDPR_2016_679',3,'Assess risks to individuals and DPIA requirements. Document selected security safeguards and unresolved privacy risks.'),('GDPR_2016_679',4,'Review processor contracts and international transfers. Assess personal-data incidents and record notification decisions.'),('NDPA_2023',1,'Maintain customer and staff processing records, responsible teams, retention decisions and lawful bases. Review privacy notices when processing changes.'),('NDPA_2023',2,'Log rights requests, verify identity and assign a case owner. Retain responses and escalate unresolved cases to the privacy team.'),('NDPA_2023',3,'Assess privacy risks, access safeguards and incident procedures. Retain incident decisions and notification assessments.'),('NDPA_2023',4,'Assess registration and DPO obligations. Maintain regulator correspondence, filing evidence and accountable owners.'),('UK_DPA_2018',1,'Identify UK processing activities, processing roles and applicable legal regime. Retain legal assessments alongside UK GDPR obligations.'),('UK_DPA_2018',2,'Maintain UK privacy notices and rights request records. Review identity checks, response times and exemptions with the privacy team.'),('UK_DPA_2018',3,'Review risks to individuals and document DPIA decisions, retention controls, access restrictions and security evidence.'),('UK_DPA_2018',4,'Review UK incidents, supplier contracts and transfers. Record notification decisions and unresolved follow-up actions.'),('DORA_2022_2554',1,'Maintain the ICT risk framework, critical functions, assets and accountable owners. Record management review and risk decisions.'),('DORA_2022_2554',2,'Maintain ICT incident classification, escalation and reporting procedures. Retain incident records and reporting assessments.'),('DORA_2022_2554',3,'Plan applicable resilience tests. Record scope, results and findings and verify remediation before closure.'),('DORA_2022_2554',4,'Maintain ICT provider registers, risk assessments, contracts and exit arrangements. Review material supplier changes and exceptions.'),('SOC_2',1,'Agree service boundaries, reporting period and trust services categories with the auditor. Identify subservice organizations and customer responsibilities.'),('SOC_2',2,'Maintain the service risk assessment and control matrix. Assign owners and record the review of governance changes and exceptions.'),('SOC_2',3,'Retain access review, authorized change, monitoring and incident evidence for the report period. Assign remediation owners for deviations.'),('SOC_2',4,'Maintain a dated examination evidence index. Track auditor requests and exceptions and retain supporting resolution evidence.')) v(template_code,ordinal,statement) WHERE r.template_code=v.template_code AND r.ordinal=v.ordinal;

-- Question groups and fields use the native Forms starter catalogue.
INSERT INTO form_starter_templates(code,catalog_version,published_on,reference_label,name,purpose,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields)
SELECT 'FRAMEWORK_'||t.template_code,1,DATE '2026-11-14',
 'ClearSight starter material. Not complete framework coverage; review before use. '||g.official_source,
 t.title||' starter assessment','Collect evidence for the selected Program scope using '||t.title||' starter requirements.',
 ARRAY['PROGRAM_ASSESSMENT'],ARRAY['framework',t.template_code,'starter'],'CONFIDENTIAL','NONE',
 '{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
 (SELECT jsonb_agg(jsonb_build_object('id',r.group_id,'title',CASE WHEN r.requirement_code='PRIVILEGED_ACCESS' THEN 'Privileged access review' ELSE r.title END,'help',r.reference_label) ORDER BY r.ordinal) FROM framework_starter_requirements r WHERE r.template_code=t.template_code),
 (SELECT jsonb_agg(q.field ORDER BY r.ordinal,q.n)
  FROM framework_starter_requirements r CROSS JOIN LATERAL (VALUES
  (1,jsonb_build_object('id',r.group_id||'_review_date','section_id',r.group_id,'label',CASE WHEN r.requirement_code='PRIVILEGED_ACCESS' THEN 'When was the last privileged access review completed?' ELSE 'When was the latest '||lower(r.title)||' review completed?' END,'type','date','required',true)),
  (2,jsonb_build_object('id',r.group_id||'_method','section_id',r.group_id,'label',CASE WHEN r.requirement_code='PRIVILEGED_ACCESS' THEN 'How were unnecessary administrator grants and access exceptions resolved?' ELSE 'How does the Program address '||lower(r.title)||' and resolve exceptions?' END,'type','long_text','required',true)),
  (3,jsonb_build_object('id',r.group_id||'_record','section_id',r.group_id,'label',CASE WHEN r.requirement_code='PRIVILEGED_ACCESS' THEN 'Upload the signed privileged access review record.' ELSE 'Upload the approved '||lower(r.title)||' review record.' END,'type','file','required',true,'accepted_formats',jsonb_build_array('application/pdf','image/png','image/jpeg'),'browser_cache_policy','NO_BROWSER_CACHE'))
  ) q(n,field) WHERE r.template_code=t.template_code)
FROM framework_reference_templates t JOIN framework_reference_guidance g USING(template_code)
ON CONFLICT (code,catalog_version) DO NOTHING;

CREATE UNIQUE INDEX monitoring_form_templates_entity_revision_uq ON monitoring_form_templates(tenant_id,legal_entity_id,id,version);
CREATE TABLE framework_starter_requirement_links (
 tenant_id uuid NOT NULL, legal_entity_id uuid NOT NULL,
 template_code text NOT NULL, requirement_code text NOT NULL,
 title text NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
 statement text NOT NULL CHECK (char_length(btrim(statement)) BETWEEN 1 AND 2000),
 reference_label text NOT NULL,
 form_template_id uuid NOT NULL, form_template_version bigint NOT NULL,
 group_id text NOT NULL, version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 PRIMARY KEY(tenant_id,legal_entity_id,template_code,requirement_code),
 FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
 FOREIGN KEY(template_code,requirement_code) REFERENCES framework_starter_requirements(template_code,requirement_code),
 FOREIGN KEY(tenant_id,legal_entity_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version)
);
CREATE TABLE framework_starter_requirement_events (
 id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL, legal_entity_id uuid NOT NULL,
 template_code text NOT NULL, requirement_code text NOT NULL, version bigint NOT NULL,
 actor_id uuid NOT NULL, payload jsonb NOT NULL, rationale text NOT NULL, occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id),
 UNIQUE(tenant_id,legal_entity_id,template_code,requirement_code,version)
);
-- Safe explicit backfill for existing and newly created legal entities. Staff-edited
-- forms and links are never updated. New questionnaires remain DRAFT.
CREATE FUNCTION backfill_framework_starter_forms(target_entity uuid DEFAULT NULL) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,status,is_current,version,starter_catalog_code,starter_catalog_version)
 SELECT md5(e.id::text||s.code)::uuid,e.tenant_id,e.id,s.code,s.name,s.purpose,'Governance, Risk and Compliance',s.approved_uses,s.tags,s.sensitivity,s.scoring_mode,s.presentation,s.sections,s.fields,'DRAFT',false,1,s.code,s.catalog_version
 FROM legal_entities e CROSS JOIN form_starter_templates s
 WHERE s.code LIKE 'FRAMEWORK_%' AND s.catalog_version=1 AND (target_entity IS NULL OR e.id=target_entity)
 AND NOT EXISTS(SELECT 1 FROM monitoring_form_templates f WHERE f.tenant_id=e.tenant_id AND f.legal_entity_id=e.id AND f.starter_catalog_code=s.code)
 ON CONFLICT(tenant_id,id,version) DO NOTHING;
 INSERT INTO framework_starter_requirement_links(tenant_id,legal_entity_id,template_code,requirement_code,title,statement,reference_label,form_template_id,form_template_version,group_id)
 SELECT f.tenant_id,f.legal_entity_id,r.template_code,r.requirement_code,r.title,r.statement,r.reference_label,f.id,f.version,r.group_id
 FROM framework_starter_requirements r JOIN monitoring_form_templates f ON f.starter_catalog_code='FRAMEWORK_'||r.template_code AND f.version=1
 WHERE (target_entity IS NULL OR f.legal_entity_id=target_entity)
 ON CONFLICT DO NOTHING;
END $$;
SELECT backfill_framework_starter_forms();
ALTER TABLE program_framework_adoptions ALTER COLUMN edition_id DROP NOT NULL;
ALTER TABLE program_framework_adoptions ADD COLUMN template_code text REFERENCES framework_reference_templates(template_code),
 ADD COLUMN configuration jsonb,
 ADD CONSTRAINT program_framework_adoption_source_ck CHECK ((edition_id IS NOT NULL AND template_code IS NULL) OR (edition_id IS NULL AND template_code IS NOT NULL AND configuration IS NOT NULL));

CREATE UNIQUE INDEX program_framework_adoptions_scope_uq ON program_framework_adoptions(id,tenant_id,legal_entity_id,program_id);
CREATE TABLE program_framework_requirement_questions (
 adoption_id uuid NOT NULL,
 requirement_id uuid NOT NULL, tenant_id uuid NOT NULL, legal_entity_id uuid NOT NULL, program_id uuid NOT NULL,
 form_template_id uuid NOT NULL, form_template_version bigint NOT NULL, group_id text NOT NULL,
 PRIMARY KEY(tenant_id,program_id,requirement_id),
 FOREIGN KEY(adoption_id,tenant_id,legal_entity_id,program_id) REFERENCES program_framework_adoptions(id,tenant_id,legal_entity_id,program_id),
 FOREIGN KEY(requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
 FOREIGN KEY(tenant_id,legal_entity_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version),
 FOREIGN KEY(program_id,tenant_id,legal_entity_id) REFERENCES programs(id,tenant_id,legal_entity_id)
);

CREATE FUNCTION validate_framework_question_group() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM monitoring_form_templates f WHERE f.tenant_id=NEW.tenant_id AND f.legal_entity_id=NEW.legal_entity_id AND f.id=NEW.form_template_id AND f.version=NEW.form_template_version
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(f.sections) x WHERE x->>'id'=NEW.group_id)
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(f.fields) x WHERE x->>'section_id'=NEW.group_id)) THEN RAISE EXCEPTION 'Questionnaire group does not belong to the selected form revision' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER framework_starter_group_check BEFORE INSERT OR UPDATE ON framework_starter_requirement_links FOR EACH ROW EXECUTE FUNCTION validate_framework_question_group();
CREATE TRIGGER program_framework_group_check BEFORE INSERT OR UPDATE ON program_framework_requirement_questions FOR EACH ROW EXECUTE FUNCTION validate_framework_question_group();
CREATE FUNCTION install_new_entity_framework_starters() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM backfill_framework_starter_forms(NEW.id); RETURN NEW; END $$;
CREATE TRIGGER legal_entity_framework_starters AFTER INSERT ON legal_entities FOR EACH ROW EXECUTE FUNCTION install_new_entity_framework_starters();
COMMIT;

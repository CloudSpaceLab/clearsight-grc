BEGIN;
-- Reference templates only. These rows are not published framework editions,
-- approved requirements, Program adoptions, or compliance conclusions.
-- Safe to rerun: existing records and the published catalog are not changed.
CREATE TABLE IF NOT EXISTS framework_reference_templates (
    template_code text PRIMARY KEY,
    title text NOT NULL,
    description text NOT NULL,
    publisher text NOT NULL,
    edition text NOT NULL DEFAULT '',
    jurisdiction text NOT NULL,
    topic text NOT NULL,
    content_status text NOT NULL DEFAULT 'REFERENCE_ONLY'
        CHECK (content_status = 'REFERENCE_ONLY')
);
INSERT INTO framework_reference_templates
    (template_code, title, description, publisher, edition, jurisdiction, topic)
VALUES
    ('ISO_27001_2022', 'ISO/IEC 27001:2022', 'Information security management systems', 'ISO/IEC', '2022', 'GLOBAL', 'Information security'),
    ('NIST_CSF_2', 'NIST CSF 2.0', 'Cybersecurity Framework', 'NIST', '2.0 · 2024', 'US', 'Cybersecurity'),
    ('PCI_DSS_4_0_1', 'PCI DSS v4.0.1', 'Payment Card Industry Data Security Standard', 'PCI SSC', '4.0.1 · 2024', 'GLOBAL', 'Payments'),
    ('GDPR_2016_679', 'General Data Protection Regulation', 'Regulation (EU) 2016/679 · GDPR', 'European Union', '2016/679', 'EU', 'Privacy'),
    ('NDPA_2023', 'Nigeria Data Protection Act', 'Data protection obligations · NDPA 2023', 'Federal Republic of Nigeria', '2023', 'NG', 'Privacy'),
    ('UK_DPA_2018', 'UK Data Protection Act', 'Data Protection Act 2018', 'UK Parliament', '2018', 'UK', 'Privacy'),
    ('DORA_2022_2554', 'Digital Operational Resilience Act', 'Regulation (EU) 2022/2554 · DORA', 'European Union', '2022/2554', 'EU', 'Operational resilience'),
    ('SOC_2', 'SOC 2', 'Service organization controls reporting', 'AICPA', '', 'GLOBAL', 'Assurance')
ON CONFLICT (template_code) DO NOTHING;
-- Starter guidance for locally installed reference templates. These are
-- original prompts, not official clause text, approved mappings or scores.
-- Reruns preserve existing guidance; review/version changes separately.
CREATE TABLE IF NOT EXISTS framework_reference_guidance (
  template_code text PRIMARY KEY REFERENCES framework_reference_templates(template_code),
  purpose text NOT NULL,
  applicability_prompt text NOT NULL,
  official_source text NOT NULL CHECK (official_source LIKE 'https://%'),
  source_note text NOT NULL,
  guidance_version text NOT NULL DEFAULT 'starter-1',
  review_status text NOT NULL DEFAULT 'REVIEW_REQUIRED' CHECK (review_status = 'REVIEW_REQUIRED')
);
CREATE TABLE IF NOT EXISTS framework_reference_guidance_areas (
  template_code text NOT NULL REFERENCES framework_reference_guidance(template_code),
  area_code text NOT NULL,
  title text NOT NULL,
  assessment_prompt text NOT NULL,
  evidence_examples text NOT NULL,
  ordinal integer NOT NULL CHECK (ordinal > 0),
  PRIMARY KEY (template_code, area_code),
  UNIQUE (template_code, ordinal)
);
INSERT INTO framework_reference_guidance(template_code,purpose,applicability_prompt,official_source,source_note) VALUES
('ISO_27001_2022','Support a risk-based information security management system.','Identify the entity, information, services and boundaries of the proposed management system.','https://www.iso.org/standard/27001','ISO/IEC 27001:2022; check the 2024 amendment and obtain licensed text for clause-level work.'),
('SOC_2','Support preparation for a service organization controls examination.','Identify the service, system boundaries, report period and trust services categories with the intended auditor.','https://www.aicpa-cima.com/resources/download/2017-trust-services-criteria-with-revised-points-of-focus-2022','2017 Trust Services Criteria with revised 2022 points of focus; confirm applicable criteria and report scope.'),
('NIST_CSF_2','Organize cybersecurity risk management outcomes across the enterprise.','Identify the organization, services, assets, target profile and risk context.','https://www.nist.gov/publications/nist-cybersecurity-framework-csf-20','NIST Cybersecurity Framework 2.0; starter prompts are not the complete core.'),
('PCI_DSS_4_0_1','Support management of security risks to payment account data.','Determine the payment environment, roles, dependencies and validation approach with the appropriate assessor.','https://www.pcisecuritystandards.org/document_library/','PCI DSS v4.0.1; verify current validation requirements and licensed source materials.'),
('GDPR_2016_679','Support governance of personal-data processing and data subject rights.','Determine relevant entities, processing activities, roles and territorial scope with legal counsel.','https://eur-lex.europa.eu/eli/reg/2016/679','Regulation (EU) 2016/679; review current consolidated text and local implementing law.'),
('NDPA_2023','Support governance of personal-data processing in relation to Nigerian data protection law.','Determine processing roles, Nigerian nexus and any additional regulatory obligations with local counsel.','https://forms.ndpc.gov.ng/ndp-act-2023-gaid-2025-implementation-workshop/','NDPC overview references the Act and GAID; obtain and review authoritative texts and current guidance.'),
('UK_DPA_2018','Support governance of personal-data processing under the UK data protection regime.','Determine processing roles, territorial scope and how the Act interacts with UK GDPR.','https://www.legislation.gov.uk/ukpga/2018/12/contents','Consult the current revised Act, UK GDPR and relevant guidance; amendments may apply.'),
('DORA_2022_2554','Support management of digital operational resilience in financial services.','Determine entity type, services and applicable obligations with legal and compliance teams.','https://eur-lex.europa.eu/eli/reg/2022/2554/oj','Regulation (EU) 2022/2554; also review applicable technical standards and implementing measures.')
ON CONFLICT (template_code) DO NOTHING;
INSERT INTO framework_reference_guidance_areas(template_code,area_code,title,assessment_prompt,evidence_examples,ordinal) VALUES
('ISO_27001_2022','GOV','Governance and scope','Has management defined the ISMS scope, responsibilities and security objectives?','Scope statement, policy, roles and management review records',1),
('ISO_27001_2022','RISK','Risk assessment and treatment','Are information-security risks assessed, treated and reviewed against agreed criteria?','Risk methodology, risk register, treatment decisions and approvals',2),
('ISO_27001_2022','CONTROL','Controls and applicability','Are controls selected, owned and justified against identified risks?','Statement of applicability, control register and control-owner records',3),
('ISO_27001_2022','ASSURE','Monitoring and improvement','Are performance, internal audits and corrective actions reviewed and followed through?','Monitoring results, audit reports, corrective-action records',4),
('SOC_2','SCOPE','Service and report scope','Are the service, system boundaries, period and relevant categories agreed?','System description, scope inventory and auditor correspondence',1),
('SOC_2','GOV','Control governance','Are responsibilities, risks and control activities documented for the scoped service?','Risk assessment, control matrix, owner assignments',2),
('SOC_2','SECURITY','Security operations','Are access, change, incident and monitoring activities operated and evidenced?','Access reviews, change approvals, incidents and monitoring records',3),
('SOC_2','ASSURANCE','Evidence and independent examination','Can the team provide dated evidence and resolve exceptions for the examination period?','Evidence index, exceptions log and auditor requests',4),
('NIST_CSF_2','GOVERN','Govern','Are cybersecurity risks governed in line with organizational objectives?','Risk strategy, policies, responsibilities and oversight minutes',1),
('NIST_CSF_2','IDENTIFY','Identify','Are assets, dependencies and cybersecurity risks understood?','Asset inventory, risk assessment and supplier inventory',2),
('NIST_CSF_2','PROTECT','Protect','Are safeguards selected and operated for important assets and services?','Access reviews, configuration baselines and awareness records',3),
('NIST_CSF_2','DETECT','Detect','Are adverse events monitored and analyzed?','Detection rules, alert records and investigation procedures',4),
('NIST_CSF_2','RESPOND','Respond','Are incident response activities coordinated, recorded and improved?','Response plans, incident records and exercise reports',5),
('NIST_CSF_2','RECOVER','Recover','Are recovery capabilities maintained and tested?','Recovery plans, backup tests and restoration exercise records',6),
('PCI_DSS_4_0_1','SCOPE','Payment environment scope','Is the account-data environment and its supporting connections inventoried?','Data flow diagrams, asset inventory and scope review',1),
('PCI_DSS_4_0_1','PROTECT','Data and access protection','Are payment data protections and access restrictions reviewed for the scoped environment?','Access reviews, configuration standards and encryption records',2),
('PCI_DSS_4_0_1','MONITOR','Monitoring and testing','Are security monitoring and testing activities performed for the scoped environment?','Logs, scan results, test reports and remediation records',3),
('PCI_DSS_4_0_1','VALIDATE','Governance and validation','Are responsibilities, evidence and the required validation route agreed?','Responsibility matrix, policies, assessor correspondence and validation records',4),
('GDPR_2016_679','INVENTORY','Processing and lawful basis','Are personal-data processing activities and their legal bases documented and reviewed?','Processing inventory, legal-basis analysis and privacy notices',1),
('GDPR_2016_679','RIGHTS','Individual rights','Is there a process to receive, verify and respond to individual rights requests?','Request procedures, case logs and response records',2),
('GDPR_2016_679','RISK','Security and privacy risk','Are processing risks assessed and appropriate safeguards reviewed?','Risk assessments, DPIAs where applicable and security controls',3),
('GDPR_2016_679','SHARING','Processors, transfers and incidents','Are processors, transfers and personal-data incidents governed and reviewed?','Contracts, transfer assessments and breach response records',4),
('NDPA_2023','INVENTORY','Processing governance','Are processing activities, roles and legal bases documented and reviewed?','Processing register, privacy notices and role assignments',1),
('NDPA_2023','RIGHTS','Individual rights','Are requests handled through an accountable and monitored process?','Rights-request procedures, cases and response records',2),
('NDPA_2023','PROTECTION','Safeguards and incidents','Are privacy risks, safeguards and incident response practices reviewed?','Risk assessments, security measures and incident records',3),
('NDPA_2023','OVERSIGHT','Regulatory oversight','Have applicable registration, reporting and accountability obligations been assessed?','Legal assessment, regulator correspondence and filing records where applicable',4),
('UK_DPA_2018','SCOPE','Processing scope and roles','Have the relevant UK processing activities, roles and applicable legal regime been identified?','Processing inventory, legal review and responsibility records',1),
('UK_DPA_2018','RIGHTS','Rights and transparency','Are notices and individual-rights handling processes documented and monitored?','Privacy notices, request logs and response procedures',2),
('UK_DPA_2018','SAFEGUARDS','Risk and safeguards','Are processing risks and appropriate protections reviewed?','Risk assessments, DPIAs where applicable and security records',3),
('UK_DPA_2018','INCIDENT','Incident and supplier oversight','Are incidents, suppliers and transfers subject to defined review?','Incident records, contracts and transfer assessments',4),
('DORA_2022_2554','RISK','ICT risk management','Are ICT risks, critical services and accountable roles documented and reviewed?','ICT risk framework, asset inventory and oversight minutes',1),
('DORA_2022_2554','INCIDENT','ICT incident management','Are ICT incidents classified, escalated and managed under documented processes?','Incident procedures, classification records and notifications where applicable',2),
('DORA_2022_2554','TEST','Resilience testing','Are resilience tests planned, executed and followed up according to applicability?','Test plan, results, findings and remediation records',3),
('DORA_2022_2554','THIRD_PARTY','ICT third-party risk','Are relevant ICT service providers and dependencies governed throughout their lifecycle?','Supplier inventory, contracts, risk assessments and exit plans',4)
ON CONFLICT (template_code,area_code) DO NOTHING;
COMMIT;

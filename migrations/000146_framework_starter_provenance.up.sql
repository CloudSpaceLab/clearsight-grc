BEGIN;
-- Source metadata corrections only. Never rewrite a saved Program snapshot,
-- native questionnaire revision, staff wording or bank-selected question link.
UPDATE framework_reference_templates SET edition='2017 TSC - revised points of focus 2022' WHERE template_code='SOC_2' AND edition='';
UPDATE framework_reference_guidance SET official_source='https://ndpc.gov.ng/wp-content/uploads/2024/03/Nigeria_Data_Protection_Act_2023.pdf',source_note='Nigeria Data Protection Act 2023. Original bank starter prompts; review the Act and current implementation guidance before approval.' WHERE template_code='NDPA_2023' AND official_source='https://forms.ndpc.gov.ng/ndp-act-2023-gaid-2025-implementation-workshop/';
UPDATE framework_starter_requirements r SET reference_label=v.reference FROM (VALUES
('GDPR_2016_679','INVENTORY','GDPR - Articles 5, 6 and 30'),
('GDPR_2016_679','RIGHTS','GDPR - Articles 12-22'),
('GDPR_2016_679','RISK','GDPR - Articles 32 and 35'),
('GDPR_2016_679','SHARING','GDPR - Articles 28, 33-34 and 44-49'),
('DORA_2022_2554','RISK','DORA - Chapter II'),
('DORA_2022_2554','INCIDENT','DORA - Chapter III'),
('DORA_2022_2554','TEST','DORA - Chapter IV'),
('DORA_2022_2554','THIRD_PARTY','DORA - Chapter V')
) v(template_code,requirement_code,reference) WHERE r.template_code=v.template_code AND r.requirement_code=v.requirement_code AND r.reference_label='Starter assessment area; clause mapping requires review';
UPDATE framework_starter_requirement_links l SET reference_label=r.reference_label FROM framework_starter_requirements r WHERE l.template_code=r.template_code AND l.requirement_code=r.requirement_code AND l.reference_label='Starter assessment area; clause mapping requires review' AND r.template_code IN ('GDPR_2016_679','DORA_2022_2554');
COMMIT;

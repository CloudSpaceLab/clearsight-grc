BEGIN;

CREATE TABLE form_starter_templates (
    code text NOT NULL,
    catalog_version bigint NOT NULL CHECK (catalog_version > 0),
    published_on date NOT NULL,
    reference_label text NOT NULL CHECK (reference_label=btrim(reference_label) AND char_length(reference_label) BETWEEN 1 AND 500),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 200),
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 1000),
    approved_uses text[] NOT NULL DEFAULT '{}',
    tags text[] NOT NULL DEFAULT '{}',
    sensitivity text NOT NULL CHECK (sensitivity IN ('INTERNAL','CONFIDENTIAL','RESTRICTED')),
    scoring_mode text NOT NULL CHECK (scoring_mode IN ('NONE','RISK','COMPLIANCE')),
    presentation jsonb NOT NULL CHECK (jsonb_typeof(presentation)='object'),
    sections jsonb NOT NULL CHECK (jsonb_typeof(sections)='array'),
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields)='array'),
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    PRIMARY KEY (code,catalog_version),
    CHECK (code=btrim(code) AND code=upper(code) AND char_length(code) BETWEEN 1 AND 128)
);

CREATE UNIQUE INDEX form_starter_templates_enabled_code_uq ON form_starter_templates(code) WHERE enabled;

INSERT INTO form_starter_templates(
    code,catalog_version,published_on,reference_label,name,purpose,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields
) VALUES (
    'VENDOR_DUE_DILIGENCE',1,DATE '2026-08-27',
    'ClearSight reference data. Review this draft against the bank''s policy before approval.',
    'Vendor due diligence review',
    'Confirm the vendor identity held by the bank and collect current operating evidence for review.',
    ARRAY['VENDOR_DUE_DILIGENCE','VENDOR_REFRESH'],ARRAY['vendor','due-diligence','reference'],'CONFIDENTIAL','COMPLIANCE',
    '{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
    '[{"id":"identity","title":"Vendor identity"},{"id":"operations","title":"Operating safeguards","weight":100}]'::jsonb,
    '[
      {"id":"legal_name","section_id":"identity","label":"Legal name","type":"short_text","required":true,"collection_intent":"CONFIRM_OR_CORRECT","record_target":{"key":"VENDOR.IDENTITY.LEGAL_NAME","required_subject_type":"VENDOR_RELATIONSHIP"}},
      {"id":"registered_address","section_id":"identity","label":"Registered office address","type":"long_text","required":true,"collection_intent":"CONFIRM_OR_CORRECT","record_target":{"key":"VENDOR.IDENTITY.REGISTERED_ADDRESS","required_subject_type":"VENDOR_RELATIONSHIP"}},
      {"id":"website_domain","section_id":"identity","label":"Website","type":"url","required":true,"collection_intent":"CONFIRM_OR_CORRECT","record_target":{"key":"VENDOR.IDENTITY.WEBSITE_DOMAIN","required_subject_type":"VENDOR_RELATIONSHIP"}},
      {"id":"certificate","section_id":"identity","label":"Certificate of operation","type":"vendor_document","required":true,"collection_intent":"REPLACE_HELD_DOCUMENT","browser_cache_policy":"NO_BROWSER_CACHE","record_target":{"key":"VENDOR.DOCUMENT.CERTIFICATE_OF_OPERATION","required_subject_type":"VENDOR_RELATIONSHIP"},"accepted_formats":["application/pdf","image/png","image/jpeg"]},
      {"id":"security_policy","section_id":"operations","label":"An approved information security policy is current","type":"yes_no","required":true,"scoring":{"weight":50,"answer_scores":{"Yes":100,"No":0},"critical_answers":["No"]}},
      {"id":"incident_process","section_id":"operations","label":"The incident response process has been tested in the last 12 months","type":"yes_no","required":true,"scoring":{"weight":50,"answer_scores":{"Yes":100,"No":0},"critical_answers":["No"]}},
      {"id":"attestation","section_id":"operations","label":"Response confirmation","type":"attestation","required":true,"attestation":"I confirm that this response is accurate to the best of my knowledge."},
      {"id":"signature","section_id":"operations","label":"Authorized sign-off","type":"signature","required":true,"browser_cache_policy":"NO_BROWSER_CACHE","accepted_formats":["image/png"]}
    ]'::jsonb
);

COMMIT;

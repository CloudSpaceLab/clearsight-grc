BEGIN;

CREATE TABLE user_presentation_preferences (
    tenant_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    home_focus text NOT NULL DEFAULT 'AUTO' CHECK (home_focus IN ('AUTO','POSTURE','MY_WORK')),
    portfolio_lens text NOT NULL DEFAULT 'AUTO' CHECK (portfolio_lens IN ('AUTO','PROGRAMS','RISKS','LOSSES','VENDORS','PROCESSING_ACTIVITIES','FORMS')),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1 CHECK (version>0),
    PRIMARY KEY (tenant_id,principal_id),
    CONSTRAINT user_presentation_preferences_principal_fk
        FOREIGN KEY (principal_id,tenant_id)
        REFERENCES principals(id,tenant_id)
        ON DELETE CASCADE
);

COMMIT;

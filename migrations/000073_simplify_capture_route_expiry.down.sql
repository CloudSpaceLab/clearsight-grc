ALTER TABLE capture_access_routes
    ADD COLUMN max_redemptions integer,
    ADD COLUMN redemptions integer;

UPDATE capture_access_routes
SET max_redemptions = CASE WHEN access_policy = 'SHARED_LINK_EMAIL_OTP' THEN 20 ELSE 1 END,
    redemptions = 0;

ALTER TABLE capture_access_routes
    ALTER COLUMN max_redemptions SET DEFAULT 1,
    ALTER COLUMN max_redemptions SET NOT NULL,
    ALTER COLUMN redemptions SET DEFAULT 0,
    ALTER COLUMN redemptions SET NOT NULL,
    ADD CONSTRAINT capture_access_routes_max_redemptions_check CHECK (max_redemptions BETWEEN 1 AND 20),
    ADD CONSTRAINT capture_access_routes_redemptions_check CHECK (redemptions >= 0 AND redemptions <= max_redemptions);

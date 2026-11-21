BEGIN;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM framework_catalog_publication_reviews LIMIT 1) THEN RAISE EXCEPTION 'Cannot remove publication review history'; END IF;
END $$;
DROP TRIGGER framework_catalog_publication_review_guard ON framework_catalog_publication_reviews;
DROP FUNCTION guard_framework_catalog_publication_review();
DROP TABLE framework_catalog_publication_reviews;
COMMIT;

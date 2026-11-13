BEGIN;
-- A release proposal is separate from the immutable published catalog.
-- Tenant identifies the accountable operator's organization, not ownership of the global edition.
CREATE TABLE framework_catalog_publication_reviews (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 maker_id uuid NOT NULL,
 release_sha256 text NOT NULL CHECK (release_sha256 ~ '^[0-9a-f]{64}$'),
 source_sha256 text NOT NULL CHECK (source_sha256 ~ '^[0-9a-f]{64}$'),
 source_bytes bigint NOT NULL CHECK (source_bytes BETWEEN 1 AND 67108864),
 manifest jsonb NOT NULL CHECK (jsonb_typeof(manifest)='object'),
 status text NOT NULL CHECK (status IN ('DRAFT','PENDING','APPROVED','REJECTED','PUBLISHED')),
 version bigint NOT NULL CHECK (version > 0),
 prepared_at timestamptz NOT NULL,
 reviewer_id uuid,
 review_rationale text,
 reviewed_at timestamptz,
 published_edition_id uuid REFERENCES framework_catalog_editions(id),
 is_demo boolean NOT NULL DEFAULT false,
 FOREIGN KEY (maker_id, tenant_id) REFERENCES principals(id, tenant_id),
 FOREIGN KEY (reviewer_id, tenant_id) REFERENCES principals(id, tenant_id),
 CHECK (reviewer_id IS NULL OR reviewer_id <> maker_id),
 CHECK ((status IN ('DRAFT','PENDING') AND reviewer_id IS NULL AND review_rationale IS NULL AND reviewed_at IS NULL AND published_edition_id IS NULL)
  OR (status IN ('APPROVED','REJECTED') AND reviewer_id IS NOT NULL AND review_rationale IS NOT NULL AND length(btrim(review_rationale)) BETWEEN 1 AND 2000 AND reviewed_at IS NOT NULL AND reviewed_at >= prepared_at AND published_edition_id IS NULL)
  OR (status='PUBLISHED' AND reviewer_id IS NOT NULL AND review_rationale IS NOT NULL AND length(btrim(review_rationale)) BETWEEN 1 AND 2000 AND reviewed_at IS NOT NULL AND reviewed_at >= prepared_at AND published_edition_id IS NOT NULL))
);
CREATE INDEX framework_catalog_publication_reviews_queue ON framework_catalog_publication_reviews(tenant_id,status,prepared_at DESC,id);
CREATE FUNCTION guard_framework_catalog_publication_review() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'publication review history cannot be deleted'; END IF;
 IF TG_OP='INSERT' THEN
   IF NEW.status <> 'DRAFT' OR NEW.version <> 1 THEN RAISE EXCEPTION 'publication reviews start as draft'; END IF;
 ELSE
   IF NEW.id<>OLD.id OR NEW.tenant_id<>OLD.tenant_id OR NEW.maker_id<>OLD.maker_id OR NEW.manifest<>OLD.manifest OR NEW.release_sha256<>OLD.release_sha256 OR NEW.source_sha256<>OLD.source_sha256 OR NEW.source_bytes<>OLD.source_bytes OR NEW.prepared_at<>OLD.prepared_at OR NEW.is_demo<>OLD.is_demo OR NEW.version<>OLD.version+1 THEN RAISE EXCEPTION 'publication source and proposal are immutable'; END IF;
   IF NOT ((OLD.status='DRAFT' AND NEW.status='PENDING') OR (OLD.status='PENDING' AND NEW.status IN ('APPROVED','REJECTED')) OR (OLD.status='APPROVED' AND NEW.status='PUBLISHED')) THEN RAISE EXCEPTION 'invalid publication review transition'; END IF;
   IF OLD.status='APPROVED' AND (NEW.reviewer_id IS DISTINCT FROM OLD.reviewer_id OR NEW.review_rationale IS DISTINCT FROM OLD.review_rationale OR NEW.reviewed_at IS DISTINCT FROM OLD.reviewed_at) THEN RAISE EXCEPTION 'publication decision is immutable'; END IF;
   IF NEW.status='PUBLISHED' AND NOT EXISTS (SELECT 1 FROM framework_catalog_editions e WHERE e.id=NEW.published_edition_id AND e.release_sha256=NEW.release_sha256 AND e.source_sha256=NEW.source_sha256) THEN RAISE EXCEPTION 'published edition must match reviewed release'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER framework_catalog_publication_review_guard BEFORE INSERT OR UPDATE OR DELETE ON framework_catalog_publication_reviews FOR EACH ROW EXECUTE FUNCTION guard_framework_catalog_publication_review();
COMMIT;

BEGIN;
DROP INDEX IF EXISTS third_party_assessment_matter_relationship_link_idx;
ALTER TABLE third_party_assessment_matter_links
    DROP CONSTRAINT IF EXISTS third_party_assessment_matter_relationship_link_fk,
    DROP COLUMN relationship_link_id;
COMMIT;

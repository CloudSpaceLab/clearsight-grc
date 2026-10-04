BEGIN;

DROP TABLE IF EXISTS legal_entity_data_boundary_revisions;
DROP TABLE IF EXISTS legal_entity_data_boundaries;
DROP FUNCTION IF EXISTS valid_legal_entity_data_regions(text[]);

COMMIT;

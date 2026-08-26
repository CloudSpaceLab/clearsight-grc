BEGIN;
DELETE FROM outbox_events WHERE aggregate_type='VENDOR_RELATIONSHIP_LINK';
DROP TABLE IF EXISTS third_party_relationship_link_events;
DROP TABLE IF EXISTS third_party_relationship_matter_links;
DROP TABLE IF EXISTS third_party_relationship_program_links;
COMMIT;

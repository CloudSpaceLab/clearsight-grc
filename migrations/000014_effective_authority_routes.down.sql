BEGIN;

DROP TRIGGER IF EXISTS routing_policy_versions_effective_routes_trg ON routing_policy_versions;
DROP TRIGGER IF EXISTS routing_policies_effective_routes_trg ON routing_policies;
DROP FUNCTION IF EXISTS refresh_effective_authority_routes_trigger();
DROP FUNCTION IF EXISTS refresh_effective_authority_routes(uuid);
DROP TABLE IF EXISTS effective_authority_routes;

COMMIT;

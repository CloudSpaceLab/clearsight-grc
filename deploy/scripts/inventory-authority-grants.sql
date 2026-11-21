-- Read-only compatibility inventory. Includes expired and future grants.
BEGIN TRANSACTION READ ONLY;

WITH classified AS (
  SELECT g.id, g.tenant_id, g.legal_entity_id, g.principal_id,
         g.position_id, g.role_template_id, g.decision_type,
         g.valid_from, g.valid_until, g.policy_version,
         jsonb_typeof(g.limits) = 'object' AS is_object,
         ARRAY(SELECT k FROM jsonb_object_keys(
           CASE WHEN jsonb_typeof(g.limits) = 'object' THEN g.limits ELSE '{}'::jsonb END
         ) AS k WHERE k NOT IN ('min_materiality', 'max_materiality') ORDER BY k) AS unsupported_keys,
         CASE WHEN jsonb_typeof(g.limits) = 'object' AND g.limits ? 'min_materiality'
           THEN COALESCE(jsonb_typeof(g.limits->'min_materiality') = 'number'
                 AND g.limits->>'min_materiality' ~ '^[0-5]$', false)
           ELSE true END AS min_valid,
         CASE WHEN jsonb_typeof(g.limits) = 'object' AND g.limits ? 'max_materiality'
           THEN COALESCE(jsonb_typeof(g.limits->'max_materiality') = 'number'
                 AND g.limits->>'max_materiality' ~ '^[0-5]$', false)
           ELSE true END AS max_valid,
         g.limits
    FROM authority_grants g
), findings AS (
  SELECT *, ARRAY_REMOVE(ARRAY[
    CASE WHEN NOT is_object THEN 'limits is not an object' END,
    CASE WHEN cardinality(unsupported_keys) > 0 THEN 'unsupported keys' END,
    CASE WHEN NOT min_valid THEN 'invalid min_materiality' END,
    CASE WHEN NOT max_valid THEN 'invalid max_materiality' END,
    CASE WHEN min_valid AND max_valid AND limits ? 'min_materiality' AND limits ? 'max_materiality'
      THEN CASE WHEN (limits->>'min_materiality')::integer > (limits->>'max_materiality')::integer
        THEN 'min_materiality exceeds max_materiality' END END
  ], NULL) AS issues
  FROM classified
)
SELECT id, tenant_id, legal_entity_id, principal_id, position_id, role_template_id,
       decision_type, valid_from, valid_until, policy_version, unsupported_keys, issues
  FROM findings
 WHERE cardinality(issues) > 0
 ORDER BY tenant_id, legal_entity_id NULLS FIRST, id;

COMMIT;

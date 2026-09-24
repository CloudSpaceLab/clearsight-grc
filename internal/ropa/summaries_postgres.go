package ropa

// ListActivitiesSQL returns one bounded page from the current processing-activity
// table. The caller supplies, in order: tenant ID, legal-entity ID, status,
// lawful basis, owner principal ID, search text, include-retired flag,
// has-cursor flag, cursor status rank, cursor review date, cursor activity ID,
// and limit+1.
//
// All scope and filter predicates are inside the materialized page CTE, before
// its limit. The row-comparison predicate and ORDER BY deliberately use the
// same status-rank, review-date, and ID expressions as the migration's
// ropa_register_keyset_idx and ropa_register_history_keyset_idx indexes.
func ListActivitiesSQL() string {
	return `
WITH page AS MATERIALIZED (
  SELECT id::text,
         tenant_id::text,
         legal_entity_id::text,
         code,
         name,
         description,
         status,
         purpose,
         lawful_basis,
         controller,
         processor,
         automated_decision_making,
         data_subject_categories,
         personal_data_categories,
         security_measures,
         retention_period,
         start_date,
         end_date,
         next_review_date,
         COALESCE(owner_principal_id::text, ''),
         COALESCE(required_authority_principal_id::text, ''),
         COALESCE(program_id::text, ''),
         version,
         created_at,
         updated_at
  FROM ropa_processing_activities
  WHERE tenant_id = $1::uuid
    AND legal_entity_id = $2::uuid
    AND ($3 = '' OR status = $3)
    AND ($4 = '' OR lawful_basis = $4)
    AND ($5 = '' OR owner_principal_id = $5::uuid)
    AND (
      $6 = '' OR
      name ILIKE '%' || $6 || '%' OR
      code ILIKE '%' || $6 || '%' OR
      description ILIKE '%' || $6 || '%' OR
      purpose ILIKE '%' || $6 || '%' OR
      controller ILIKE '%' || $6 || '%' OR
      processor ILIKE '%' || $6 || '%' OR
      data_subject_categories ILIKE '%' || $6 || '%' OR
      personal_data_categories ILIKE '%' || $6 || '%' OR
      security_measures ILIKE '%' || $6 || '%' OR
      retention_period ILIKE '%' || $6 || '%' OR
      id::text ILIKE '%' || $6 || '%' OR
      owner_principal_id::text ILIKE '%' || $6 || '%' OR
      required_authority_principal_id::text ILIKE '%' || $6 || '%' OR
      program_id::text ILIKE '%' || $6 || '%'
    )
    AND ($7 OR end_date IS NULL)
    AND (NOT $8 OR (CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
         COALESCE(next_review_date, '0001-01-01'::date),
         id) >
         ($9::integer, $10::date, $11::uuid))
  ORDER BY CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
         COALESCE(next_review_date, '0001-01-01'::date),
         id
  LIMIT $12
)
SELECT *
FROM page
ORDER BY CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END,
         COALESCE(next_review_date, '0001-01-01'::date),
         id
`
}

// RegisterSummarySQL builds one database-aggregated dashboard snapshot for an
// exact tenant and legal entity. The caller supplies tenant ID, legal-entity ID,
// generated time, projection version, and the as-of time used for overdue
// reviews. The scope seed keeps an empty register representable while GROUP BY
// performs all population reduction in PostgreSQL.
func RegisterSummarySQL() string {
	return `
WITH requested_scope AS (
  SELECT $1::uuid AS tenant_id, $2::uuid AS legal_entity_id
), grouped AS (
  SELECT requested_scope.tenant_id,
         requested_scope.legal_entity_id,
         COUNT(activity.id)::integer AS population,
         COALESCE(MAX(activity.updated_at), $3::timestamptz) AS source_high_water,
         COUNT(activity.id) FILTER (WHERE activity.status = 'NEW') AS new_count,
         COUNT(activity.id) FILTER (WHERE activity.status = 'OPEN') AS open_count,
         COUNT(activity.id) FILTER (WHERE activity.status = 'CLOSED') AS closed_count,
         COUNT(activity.id) FILTER (WHERE activity.next_review_date IS NOT NULL AND activity.next_review_date < $5::timestamptz) AS review_overdue_count,
         COUNT(activity.id) FILTER (WHERE btrim(activity.lawful_basis) = '') AS missing_basis_count,
         COUNT(activity.id) FILTER (WHERE activity.owner_principal_id IS NULL) AS missing_owner_count,
         COUNT(activity.id) FILTER (WHERE btrim(activity.data_subject_categories) = '') AS no_data_subjects_count,
         COUNT(activity.id) FILTER (WHERE activity.end_date IS NOT NULL) AS retired_count
  FROM requested_scope
  LEFT JOIN ropa_processing_activities activity
    ON activity.tenant_id = requested_scope.tenant_id
   AND activity.legal_entity_id = requested_scope.legal_entity_id
  GROUP BY requested_scope.tenant_id, requested_scope.legal_entity_id
)
INSERT INTO ropa_register_summary (
  tenant_id,
  legal_entity_id,
  generated_at,
  projection_version,
  source_high_water,
  population,
  excluded,
  unknown,
  counts
)
SELECT grouped.tenant_id,
       grouped.legal_entity_id,
       $3::timestamptz,
       $4,
       grouped.source_high_water,
       grouped.population,
       0,
       0,
       jsonb_build_object(
         'total', grouped.population,
         'new', grouped.new_count,
         'open', grouped.open_count,
         'closed', grouped.closed_count,
         'review_overdue', grouped.review_overdue_count,
         'missing_lawful_basis', grouped.missing_basis_count,
         'missing_owner', grouped.missing_owner_count,
         'no_data_subjects', grouped.no_data_subjects_count,
         'retired', grouped.retired_count
       )
FROM grouped
ON CONFLICT (tenant_id, legal_entity_id) DO UPDATE SET
  generated_at = EXCLUDED.generated_at,
  projection_version = EXCLUDED.projection_version,
  source_high_water = EXCLUDED.source_high_water,
  population = EXCLUDED.population,
  excluded = EXCLUDED.excluded,
  unknown = EXCLUDED.unknown,
  counts = EXCLUDED.counts
WHERE EXCLUDED.generated_at >= ropa_register_summary.generated_at
`
}

func latestSummarySQL() string {
	return `
SELECT generated_at,
       projection_version,
       source_high_water,
       population,
       excluded,
       unknown,
       counts
FROM ropa_register_summary
WHERE tenant_id = $1::uuid
  AND legal_entity_id = $2::uuid
`
}

func replaceSummarySQL() string {
	return `
INSERT INTO ropa_register_summary (
  tenant_id,
  legal_entity_id,
  generated_at,
  projection_version,
  source_high_water,
  population,
  excluded,
  unknown,
  counts
)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::jsonb)
ON CONFLICT (tenant_id, legal_entity_id) DO UPDATE SET
  generated_at = EXCLUDED.generated_at,
  projection_version = EXCLUDED.projection_version,
  source_high_water = EXCLUDED.source_high_water,
  population = EXCLUDED.population,
  excluded = EXCLUDED.excluded,
  unknown = EXCLUDED.unknown,
  counts = EXCLUDED.counts
WHERE EXCLUDED.generated_at >= ropa_register_summary.generated_at
`
}

func buildInsertActivitySQL() string {
	return `
INSERT INTO ropa_processing_activities (
  id,
  tenant_id,
  legal_entity_id,
  code,
  name,
  description,
  status,
  purpose,
  lawful_basis,
  controller,
  processor,
  automated_decision_making,
  data_subject_categories,
  personal_data_categories,
  security_measures,
  retention_period,
  start_date,
  end_date,
  next_review_date,
  owner_principal_id,
  required_authority_principal_id,
  program_id,
  version,
  created_at,
  updated_at
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13,
  $14,
  $15,
  $16,
  $17::date,
  $18::date,
  $19::date,
  $20::uuid,
  $21::uuid,
  $22::uuid,
  $23,
  $24,
  $25
)
ON CONFLICT (tenant_id, legal_entity_id, code) DO NOTHING
`
}

func buildAppendEventSQL() string {
	return `
INSERT INTO ropa_events (
  id,
  tenant_id,
  legal_entity_id,
  aggregate_type,
  aggregate_id,
  aggregate_version,
  type,
  payload,
  actor_type,
  actor_id,
  occurred_at
)
VALUES (
  $1::uuid,
  $2::uuid,
  $3::uuid,
  $4,
  $5::uuid,
  $6,
  $7,
  $8::jsonb,
  $9,
  $10::uuid,
  $11
)
`
}

func buildRevisionSQL() string {
	return `
INSERT INTO ropa_processing_activity_revisions (
  tenant_id,
  legal_entity_id,
  activity_id,
  version,
  snapshot,
  recorded_at
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::jsonb, $6)
`
}

func buildOutboxSQL() string {
	return `
INSERT INTO outbox_events (
  id,
  tenant_id,
  aggregate_type,
  aggregate_id,
  event_type,
  payload,
  occurred_at,
  available_at,
  next_attempt_at
)
VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5, $6::jsonb, $7, $7, $7)
`
}

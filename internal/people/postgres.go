package people

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository reads current assignments and append-only continuity
// events. It does not infer completion or activity from a demo fixture.
type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Profile(ctx context.Context, scope Scope) (Profile, error) {
	if r == nil || r.pool == nil {
		return Profile{}, ErrNotFound
	}
	var value Profile
	err := r.pool.QueryRow(ctx, `
SELECT p.id::text,p.display_name,p.status,COALESCE(op.title,''),COALESCE(op.function_name,'')
FROM principals p
JOIN tenants t ON t.id=p.tenant_id
LEFT JOIN LATERAL (
  SELECT title,function_name FROM org_positions
  WHERE tenant_id=p.tenant_id AND occupant_principal_id=p.id AND legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=p.tenant_id AND (id::text=$3 OR code=$3) AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until) LIMIT 1)
    AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)
  ORDER BY valid_from DESC,id DESC LIMIT 1
) op ON true
WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2 AND p.kind='PERSON'
  AND EXISTS (
    SELECT 1 FROM org_positions person_scope
    WHERE person_scope.tenant_id=p.tenant_id AND person_scope.occupant_principal_id=p.id
      AND person_scope.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=p.tenant_id AND (id::text=$3 OR code=$3) AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until) LIMIT 1)
      AND person_scope.valid_from<=clock_timestamp() AND (person_scope.valid_until IS NULL OR clock_timestamp()<person_scope.valid_until)
  )
  AND p.valid_from<=clock_timestamp() AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)`, scope.Viewer.TenantID, scope.PersonID, scope.Viewer.LegalEntityID).
		Scan(&value.Person.ID, &value.Person.DisplayName, &value.Person.Status, &value.Person.Position, &value.Person.Function)
	if err != nil {
		return Profile{}, notFound(err)
	}
	active, overdue, blocked, awaiting, err := r.metrics(ctx, scope)
	if err != nil {
		return Profile{}, err
	}
	value.Metrics = Metrics{Active: countMetric(active), Overdue: countMetric(overdue), Blocked: countMetric(blocked), AwaitingOutcome: countMetric(awaiting)}
	value.AsOf = time.Now().UTC()
	return value, nil
}

func (r *PostgresRepository) metrics(ctx context.Context, scope Scope) (int, int, int, int, error) {
	var active, overdue, blocked, awaiting int
	err := r.pool.QueryRow(ctx, `
SELECT
  count(*) FILTER (WHERE a.status NOT IN ('IMPLEMENTED','CANCELLED')),
  count(*) FILTER (WHERE a.status NOT IN ('IMPLEMENTED','CANCELLED') AND a.due_at IS NOT NULL AND a.due_at<clock_timestamp()),
  count(*) FILTER (WHERE a.status='BLOCKED'),
  count(*) FILTER (WHERE a.status NOT IN ('IMPLEMENTED','CANCELLED') AND EXISTS (SELECT 1 FROM verification_contracts vc WHERE vc.tenant_id=a.tenant_id AND vc.matter_id=a.matter_id AND vc.action_id=a.id AND vc.status='ACTIVE'))
FROM matter_actions a JOIN matters m ON m.tenant_id=a.tenant_id AND m.id=a.matter_id JOIN tenants t ON t.id=m.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND a.owner_principal_id::text=$2 AND `+matterVisibleSQL(`$3`, true)+``, scope.Viewer.TenantID, scope.PersonID, scope.Viewer.PrincipalID).Scan(&active, &overdue, &blocked, &awaiting)
	return active, overdue, blocked, awaiting, err
}

func (r *PostgresRepository) Work(ctx context.Context, query PageQuery) (WorkPage, error) {
	rows, err := r.pool.Query(ctx, `
SELECT a.id::text,a.matter_id::text,'ACTION',COALESCE(a.required_responsibility,''),a.title,a.status,a.due_at,a.updated_at
FROM matter_actions a JOIN matters m ON m.tenant_id=a.tenant_id AND m.id=a.matter_id JOIN tenants t ON t.id=m.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND a.owner_principal_id::text=$2 AND `+matterVisibleSQL(`$3`, true)+`
  AND ($4='' OR ($4='ACTIVE' AND a.status NOT IN ('IMPLEMENTED','CANCELLED')) OR ($4='COMPLETED' AND a.status IN ('IMPLEMENTED','CANCELLED')))
  AND ($5='' OR (a.updated_at,a.id)<(SELECT cursor.updated_at,cursor.id FROM matter_actions cursor WHERE cursor.tenant_id=a.tenant_id AND cursor.id::text=$5))
ORDER BY a.updated_at DESC,a.id DESC LIMIT $6`, query.Scope.Viewer.TenantID, query.Scope.PersonID, query.Scope.Viewer.PrincipalID, query.State, query.Cursor, query.Limit+1)
	if err != nil {
		return WorkPage{}, err
	}
	defer rows.Close()
	page := WorkPage{Items: []WorkItem{}, AsOf: time.Now().UTC()}
	for rows.Next() {
		var item WorkItem
		if err := rows.Scan(&item.ID, &item.RecordID, &item.RecordType, &item.Responsibility, &item.Title, &item.Status, &item.DueAt, &item.UpdatedAt); err != nil {
			return WorkPage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return WorkPage{}, err
	}
	if len(page.Items) > query.Limit {
		page.NextCursor = page.Items[query.Limit-1].ID
		page.Items = page.Items[:query.Limit]
	}
	return page, nil
}

func (r *PostgresRepository) Assignments(ctx context.Context, query PageQuery) (AssignmentPage, error) {
	rows, err := r.pool.Query(ctx, `
SELECT a.id::text,a.created_at,'ACTION',a.matter_id::text,a.title,COALESCE(a.required_responsibility,''),'','', 'Assigned action'
FROM matter_actions a JOIN matters m ON m.tenant_id=a.tenant_id AND m.id=a.matter_id JOIN tenants t ON t.id=m.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND a.owner_principal_id::text=$2 AND `+matterVisibleSQL(`$3`, true)+` AND ($4='' OR a.id::text<$4)
ORDER BY a.created_at DESC,a.id DESC LIMIT $5`, query.Scope.Viewer.TenantID, query.Scope.PersonID, query.Scope.Viewer.PrincipalID, query.Cursor, query.Limit+1)
	if err != nil {
		return AssignmentPage{}, err
	}
	defer rows.Close()
	page := AssignmentPage{Items: []AssignmentItem{}, AsOf: time.Now().UTC()}
	for rows.Next() {
		var item AssignmentItem
		if err := rows.Scan(&item.ID, &item.OccurredAt, &item.RecordType, &item.RecordID, &item.Title, &item.Responsibility, &item.ActorID, &item.ActorName, &item.Action); err != nil {
			return AssignmentPage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return AssignmentPage{}, err
	}
	if len(page.Items) > query.Limit {
		page.NextCursor = page.Items[query.Limit-1].ID
		page.Items = page.Items[:query.Limit]
	}
	return page, nil
}

func (r *PostgresRepository) Activity(ctx context.Context, query PageQuery) (ActivityPage, error) {
	rows, err := r.pool.Query(ctx, `
SELECT ce.id::text,ce.occurred_at,ce.event_type,'MATTER',ce.aggregate_id::text,m.title,'Continuity record'
FROM continuity_events ce JOIN matters m ON m.tenant_id=ce.tenant_id AND m.id=ce.aggregate_id JOIN tenants t ON t.id=ce.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='MATTER' AND ce.actor_id::text=$2 AND `+matterVisibleSQL(`$3`, false)+`
  AND ($4='' OR ce.id::text<$4) AND ($5::timestamptz IS NULL OR ce.occurred_at >= $5) AND ($6::timestamptz IS NULL OR ce.occurred_at <= $6)
ORDER BY ce.occurred_at DESC,ce.id DESC LIMIT $7`, query.Scope.Viewer.TenantID, query.Scope.PersonID, query.Scope.Viewer.PrincipalID, query.Cursor, query.From, query.To, query.Limit+1)
	if err != nil {
		return ActivityPage{}, err
	}
	defer rows.Close()
	page := ActivityPage{Items: []ActivityItem{}, AsOf: time.Now().UTC()}
	for rows.Next() {
		var item ActivityItem
		if err := rows.Scan(&item.ID, &item.OccurredAt, &item.Action, &item.RecordType, &item.RecordID, &item.Title, &item.Source); err != nil {
			return ActivityPage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ActivityPage{}, err
	}
	if len(page.Items) > query.Limit {
		page.NextCursor = page.Items[query.Limit-1].ID
		page.Items = page.Items[:query.Limit]
	}
	return page, nil
}

func countMetric(value int) Metric { return Metric{Value: &value} }
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func matterVisibleSQL(viewer string, actionJoined bool) string {
	assigned := `EXISTS (SELECT 1 FROM matter_actions visible_action WHERE visible_action.tenant_id=m.tenant_id AND visible_action.matter_id=m.id AND visible_action.owner_principal_id::text=` + viewer + `)`
	if actionJoined {
		assigned = `COALESCE(a.owner_principal_id::text,'')=` + viewer
	}
	return `(CASE WHEN NOT (m.scope ? 'access') THEN true WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN CASE WHEN jsonb_typeof(m.scope->'allowed_principal_ids')<>'array' THEN false ELSE EXISTS (SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') allowed(value) WHERE btrim(allowed.value)=` + viewer + `) END ELSE false END OR COALESCE(m.owner_principal_id::text,'')=` + viewer + ` OR ` + assigned + `)`
}

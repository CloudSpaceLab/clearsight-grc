//go:build postgres

package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// listAssignedDistributions is a private, deliberately narrow read primitive.
// The public routes remain gated: creators, owners and reviewers require their
// own proved visibility policy and exact detail/revision reads are not covered.
func (s *PostgresDistributionStore) listAssignedDistributions(ctx context.Context, query DistributionListQuery) ([]FormDistribution, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now().UTC()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.TenantID != query.TenantID || actor.LegalEntityID != query.LegalEntityID || actor.LegalEntityID == "*" || s == nil || s.repo == nil || s.repo.pool == nil {
		return nil, ErrNotFound
	}
	if !normalizeDistributionListQuery(&query, time.Now().UTC()) {
		return nil, ErrDistributionInvalid
	}
	cursor, err := decodeDistributionCursor(query.Cursor)
	if err != nil {
		return nil, err
	}
	limit := query.Limit
	if query.probeNext {
		limit++
	}
	statement := `SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
 d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
 d.reminder_policy,d.created_by::text,d.version,d.created_at,d.updated_at
 FROM capture_form_distributions d
 JOIN legal_entities scope ON scope.id=d.legal_entity_id AND scope.tenant_id=d.tenant_id AND scope.valid_from<=statement_timestamp() AND (scope.valid_until IS NULL OR statement_timestamp()<scope.valid_until)
 JOIN principals viewer ON viewer.tenant_id=d.tenant_id AND viewer.id::text=$3 AND viewer.kind='PERSON' AND viewer.status='ACTIVE'
 WHERE d.tenant_id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM tenants WHERE id::text=$1 OR slug=$1)
 AND d.legal_entity_id::text=$2 AND viewer.valid_from<=statement_timestamp() AND (viewer.valid_until IS NULL OR statement_timestamp()<viewer.valid_until)
 AND $4::timestamptz>statement_timestamp() AND $4::timestamptz>clock_timestamp()
 AND ($5::timestamptz IS NULL OR $5::timestamptz<=statement_timestamp()+interval '1 minute')
 AND d.status IN ('OPEN','LOCKED','COMPLETED','EXPIRED')
 AND NOT EXISTS(SELECT 1 FROM capture_requests protected WHERE protected.distribution_id=d.id AND protected.tenant_id=d.tenant_id AND protected.legal_entity_id=d.legal_entity_id AND protected.origin_type IS NOT NULL)
 AND NOT EXISTS(SELECT 1 FROM demo_form_distribution_archives archive WHERE archive.distribution_id=d.id AND archive.tenant_id=d.tenant_id AND archive.legal_entity_id=d.legal_entity_id AND archive.restored_at IS NULL)
 AND CASE d.subject_type
 WHEN 'PROGRAM' THEN EXISTS(SELECT 1 FROM programs visible WHERE visible.id=d.subject_id AND visible.tenant_id=d.tenant_id AND visible.legal_entity_id=d.legal_entity_id AND ` + recipientSubjectVisibilityPredicate("visible", "$3") + `)
 WHEN 'MATTER' THEN EXISTS(SELECT 1 FROM matters visible WHERE visible.id=d.subject_id AND visible.tenant_id=d.tenant_id AND visible.legal_entity_id=d.legal_entity_id AND ` + recipientSubjectVisibilityPredicate("visible", "$3") + `)
 ELSE false END
 AND EXISTS(SELECT 1 FROM capture_distribution_recipients recipient
 JOIN capture_requests req ON req.id=recipient.request_id AND req.tenant_id=recipient.tenant_id AND req.legal_entity_id=recipient.legal_entity_id AND req.distribution_id=recipient.distribution_id
 WHERE recipient.distribution_id=d.id AND recipient.tenant_id=d.tenant_id AND recipient.legal_entity_id=d.legal_entity_id
 AND recipient.role='TO' AND recipient.recipient_type='INTERNAL_PRINCIPAL' AND recipient.principal_id::text=$3 AND recipient.state<>'REVOKED'
 AND req.recipient_type='INTERNAL_PRINCIPAL' AND req.recipient_principal_id::text=$3 AND req.subject_type=d.subject_type AND req.subject_id=d.subject_id::text
 AND req.form_template_id=d.form_template_id AND req.form_template_version=d.form_template_version
 AND req.status NOT IN ('CANCELLED','EXPIRED') AND req.origin_type IS NULL)
 AND ($6='' OR d.status=$6) AND ($7::timestamptz IS NULL OR (d.updated_at,d.id)<($7::timestamptz,$8::uuid))
 AND ($9='' OR upper(d.subject_type)=$9) AND ($10='' OR d.subject_id::text=$10) AND ($11='' OR d.created_by::text=$11)
 AND ($12='' OR ($12='OPEN' AND d.status NOT IN ('COMPLETED','EXPIRED','REVOKED','SUPERSEDED') AND d.deadline>$13) OR ($12='OVERDUE' AND d.status NOT IN ('COMPLETED','EXPIRED','REVOKED','SUPERSEDED') AND d.deadline<=$13) OR ($12='CLOSED' AND d.status IN ('COMPLETED','EXPIRED','REVOKED','SUPERSEDED')))
 ORDER BY d.updated_at DESC,d.id DESC LIMIT $14`
	var issued *time.Time
	if !actor.IssuedAt.IsZero() {
		issued = &actor.IssuedAt
	}
	rows, err := s.repo.pool.Query(ctx, statement, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, actor.ExpiresAt, issued, string(query.Status), cursor.UpdatedAt, nullableUUID(cursor.ID), query.SubjectType, query.SubjectID, query.OwnerPrincipalID, string(query.DueState), query.Now, limit)
	if err != nil {
		return nil, fmt.Errorf("list assigned form requests: %w", err)
	}
	defer rows.Close()
	values := make([]FormDistribution, 0, limit)
	for rows.Next() {
		value, e := scanDistribution(rows)
		if e != nil {
			return nil, e
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

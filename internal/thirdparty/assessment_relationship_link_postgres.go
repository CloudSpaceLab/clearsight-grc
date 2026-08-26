//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

func ensureAssessmentMatterRelationshipLink(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment, matterID string, kind AssessmentMatterLinkKind, actorID string, at time.Time) (RelationshipLink, error) {
	load := func() (RelationshipLink, error) {
		value, err := scanRelationshipLink(tx.QueryRow(ctx, `SELECT l.id::text,t.slug,l.legal_entity_id,l.relationship_id,'MATTER'::text,l.matter_id,l.purpose_code,l.purpose_label,l.state,
			l.created_by_principal_id::text,COALESCE(l.ended_by_principal_id::text,''),l.end_reason,l.version,l.created_at,l.updated_at,l.ended_at
			FROM third_party_relationship_matter_links l JOIN tenants t ON t.id=l.tenant_id
			WHERE l.tenant_id=$1::uuid AND l.legal_entity_id::text=$2 AND l.relationship_id=$3::uuid AND l.matter_id=$4::uuid AND l.state='ACTIVE'
			FOR UPDATE OF l`, tenantID, assessment.LegalEntityID, assessment.RelationshipID, matterID))
		return value, err
	}
	if existing, err := load(); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return RelationshipLink{}, fmt.Errorf("load assessment vendor relationship link: %w", err)
	}
	linkID, err := id.NewUUIDv7()
	if err != nil {
		return RelationshipLink{}, err
	}
	purposeCode, purposeLabel := assessmentMatterRelationshipPurpose(kind)
	value := RelationshipLink{ID: linkID, TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, RelationshipID: assessment.RelationshipID, TargetType: LinkTargetMatter, TargetID: matterID, PurposeCode: purposeCode, PurposeLabel: purposeLabel, State: RelationshipLinkActive, CreatedBy: actorID, Version: 1, CreatedAt: at.UTC(), UpdatedAt: at.UTC()}
	tag, err := tx.Exec(ctx, `INSERT INTO third_party_relationship_matter_links(
		id,tenant_id,legal_entity_id,relationship_id,matter_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at
	) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'ACTIVE',$8::uuid,1,$9,$9)
	ON CONFLICT (tenant_id,legal_entity_id,relationship_id,matter_id) WHERE state='ACTIVE' DO NOTHING`,
		value.ID, tenantID, value.LegalEntityID, value.RelationshipID, value.TargetID, value.PurposeCode, value.PurposeLabel, value.CreatedBy, value.CreatedAt)
	if err != nil {
		return RelationshipLink{}, fmt.Errorf("store assessment vendor relationship link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		existing, loadErr := load()
		if loadErr != nil {
			return RelationshipLink{}, fmt.Errorf("load concurrent assessment vendor relationship link: %w", loadErr)
		}
		return existing, nil
	}
	if err := appendRelationshipLinkEvent(ctx, tx, tenantID, value, "VendorRelationshipLinked"); err != nil {
		return RelationshipLink{}, err
	}
	return value, nil
}

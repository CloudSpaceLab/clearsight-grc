//go:build postgres

package continuity

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// Program mapping materializations share the Program event transaction and
// outbox; neither proposal nor approval mutates a Program Requirement or score.
func applyFrameworkMappingProjection(ctx context.Context, tx pgx.Tx, event Event) error {
	if event.Type == EventFrameworkMappingProposed {
		var mapping FrameworkClauseMapping
		if err := json.Unmarshal(event.Payload, &mapping); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `
			INSERT INTO framework_clause_requirement_links(
				id,tenant_id,legal_entity_id,program_id,adoption_id,edition_id,
				clause_id,requirement_id,mapping_kind,rationale,mapped_by,
				proposed_at,status,version)
			SELECT $1::uuid,t.id,p.legal_entity_id,p.id,$4::uuid,$5::uuid,
				$6::uuid,$7::uuid,$8,$9,$10::uuid,$11,$12,$13
			FROM programs p JOIN tenants t ON t.id=p.tenant_id
			WHERE p.id=$2::uuid AND (t.id::text=$3 OR t.slug=$3)
				AND p.legal_entity_id=$14::uuid`,
			mapping.ID, mapping.ProgramID, mapping.TenantID, mapping.AdoptionID,
			mapping.EditionID, mapping.ClauseID, mapping.RequirementID,
			mapping.MappingKind, mapping.Rationale, mapping.MappedBy,
			mapping.ProposedAt, mapping.Status, mapping.Version, mapping.LegalEntityID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrNotFound
		}
		return nil
	}
	var decision frameworkMappingDecision
	if err := json.Unmarshal(event.Payload, &decision); err != nil {
		return err
	}
	m := decision.Mapping
	result, err := tx.Exec(ctx, `
		UPDATE framework_clause_requirement_links l
		SET status=$7, reviewed_by=$8::uuid, reviewed_at=$9,
			effective_from=$10,version=$11
		FROM tenants t
		WHERE l.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1)
		  AND l.program_id=$2::uuid AND l.legal_entity_id=$3::uuid
		  AND l.id=$4::uuid AND l.adoption_id=$5::uuid AND l.edition_id=$6::uuid
		  AND l.status='PENDING' AND l.version=$12`,
		m.TenantID, m.ProgramID, m.LegalEntityID,
		m.ID, m.AdoptionID, m.EditionID, m.Status,
		m.ReviewedBy, m.ReviewedAt, m.EffectiveFrom, m.Version, m.Version-1)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	return nil
}

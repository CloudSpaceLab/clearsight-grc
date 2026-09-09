//go:build postgres

package thirdparty

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func WriteImportedLinkTx(ctx context.Context, tx pgx.Tx, link RelationshipLink) error {
	tenant, err := resolveTenant(ctx, tx, link.TenantID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_relationship_matter_links(id,tenant_id,legal_entity_id,relationship_id,matter_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'ACTIVE',$8::uuid,1,$9,$9)`, link.ID, tenant, link.LegalEntityID, link.RelationshipID, link.TargetID, link.PurposeCode, link.PurposeLabel, link.CreatedBy, link.CreatedAt)
	if err != nil {
		return err
	}
	_, err = appendRelationshipLinkEvent(ctx, tx, tenant, link, "VendorRelationshipLinked")
	return err
}

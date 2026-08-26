//go:build postgres

package thirdparty

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type thirdPartyCommitProof struct {
	kind, eventID, tenantID, legalEntityID, aggregateType, aggregateID, eventType string
	version                                                                       int64
}

func (r *PostgresRepository) commitThirdPartyEvents(ctx context.Context, tx pgx.Tx, proofs ...thirdPartyCommitProof) error {
	if err := tx.Commit(ctx); err != nil {
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.thirdPartyEventsRecorded(probeCtx, proofs...)
		return exactCommitResult(err, confirmed, probeErr)
	}
	return nil
}

func (r *PostgresRepository) thirdPartyEventsRecorded(ctx context.Context, proofs ...thirdPartyCommitProof) (bool, error) {
	for _, proof := range proofs {
		var confirmed bool
		var err error
		switch proof.kind {
		case "work":
			err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_work_events e JOIN tenants t ON t.id=e.tenant_id WHERE e.id::text=$1 AND (t.id::text=$2 OR t.slug=$2) AND e.legal_entity_id::text=$3 AND e.work_request_id::text=$4 AND e.work_version=$5 AND e.event_type=$6)`, proof.eventID, proof.tenantID, proof.legalEntityID, proof.aggregateID, proof.version, proof.eventType).Scan(&confirmed)
		case "link":
			err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_relationship_link_events e JOIN tenants t ON t.id=e.tenant_id WHERE e.id::text=$1 AND (t.id::text=$2 OR t.slug=$2) AND e.legal_entity_id::text=$3 AND e.link_id::text=$4 AND e.link_version=$5 AND e.event_type=$6)`, proof.eventID, proof.tenantID, proof.legalEntityID, proof.aggregateID, proof.version, proof.eventType).Scan(&confirmed)
		default:
			err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM third_party_events e JOIN tenants t ON t.id=e.tenant_id WHERE e.id::text=$1 AND (t.id::text=$2 OR t.slug=$2) AND e.aggregate_type=$3 AND e.aggregate_id::text=$4 AND e.aggregate_version=$5 AND e.event_type=$6)`, proof.eventID, proof.tenantID, proof.aggregateType, proof.aggregateID, proof.version, proof.eventType).Scan(&confirmed)
		}
		if err != nil || !confirmed {
			return false, err
		}
	}
	return len(proofs) > 0, nil
}

func relationshipCommitProof(eventID string, value Relationship, eventType string) thirdPartyCommitProof {
	return thirdPartyCommitProof{eventID: eventID, tenantID: value.TenantID, aggregateType: "VENDOR_RELATIONSHIP", aggregateID: value.ID, version: value.Version, eventType: eventType}
}

func assessmentCommitProof(eventID string, value Assessment, eventType string) thirdPartyCommitProof {
	return thirdPartyCommitProof{eventID: eventID, tenantID: value.TenantID, aggregateType: "THIRD_PARTY_ASSESSMENT", aggregateID: value.ID, version: value.Version, eventType: eventType}
}

func vendorWorkCommitProof(eventID string, value VendorWorkRequest, eventType string) thirdPartyCommitProof {
	return thirdPartyCommitProof{kind: "work", eventID: eventID, tenantID: value.TenantID, legalEntityID: value.LegalEntityID, aggregateID: value.ID, version: value.Version, eventType: eventType}
}

func relationshipLinkCommitProof(eventID string, value RelationshipLink, eventType string) thirdPartyCommitProof {
	return thirdPartyCommitProof{kind: "link", eventID: eventID, tenantID: value.TenantID, legalEntityID: value.LegalEntityID, aggregateID: value.ID, version: value.Version, eventType: eventType}
}

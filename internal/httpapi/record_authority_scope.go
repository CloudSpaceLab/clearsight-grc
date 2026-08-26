package httpapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// exactRecordActor narrows a verified tenant-wide identity to the exact legal
// entity already proven by an authorized record read. The narrowed copy is
// used only for that record request; it never broadens a fixed entity scope.
func (a *API) exactRecordActor(ctx context.Context, actor identity.Actor, requestedTenant, recordTenant, recordEntity string) (identity.Actor, error) {
	requestedTenant = strings.TrimSpace(requestedTenant)
	recordTenant = strings.TrimSpace(recordTenant)
	recordEntity = strings.TrimSpace(recordEntity)
	if strings.TrimSpace(actor.TenantID) == "" || requestedTenant == "" || actor.TenantID != requestedTenant || recordTenant != requestedTenant {
		return identity.Actor{}, fmt.Errorf("%w: loaded record is outside the verified tenant", commandauth.ErrTenantMismatch)
	}
	actorEntity := strings.TrimSpace(actor.LegalEntityID)
	if actorEntity == "" || recordEntity == "" || recordEntity == "*" {
		return identity.Actor{}, fmt.Errorf("%w: loaded record does not have an exact legal entity", commandauth.ErrLegalEntityMismatch)
	}
	if actorEntity != "*" && actorEntity != recordEntity {
		if a == nil || a.deps.Continuity == nil {
			return identity.Actor{}, fmt.Errorf("%w: loaded record is outside the verified legal entity", commandauth.ErrLegalEntityMismatch)
		}
		canonical, err := a.deps.Continuity.ResolveLegalEntity(ctx, requestedTenant, actorEntity)
		if err != nil || strings.TrimSpace(canonical) != recordEntity {
			return identity.Actor{}, fmt.Errorf("%w: loaded record is outside the verified legal entity", commandauth.ErrLegalEntityMismatch)
		}
	}
	actor.LegalEntityID = recordEntity
	return actor, nil
}

func validateRequestedRecordEntity(actor identity.Actor, requested, recordEntity string) error {
	requested = strings.TrimSpace(requested)
	recordEntity = strings.TrimSpace(recordEntity)
	if requested == "" || requested == recordEntity {
		return nil
	}
	// A fixed actor scope may be represented by an active legal-entity code
	// while the loaded record carries its canonical ID. The successful exact
	// record read has already proven that equivalence.
	if strings.TrimSpace(actor.LegalEntityID) != "*" && requested == strings.TrimSpace(actor.LegalEntityID) {
		return nil
	}
	return fmt.Errorf("%w: requested legal entity conflicts with the loaded record", commandauth.ErrLegalEntityMismatch)
}

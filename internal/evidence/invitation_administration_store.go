package evidence

import (
	"context"
	"time"
)

type invitationAdministrationStore interface {
	ListInvitationMetadata(context.Context, string, string, int) ([]InvitationMetadata, error)
	RevokeInvitationForRequester(context.Context, RevokeInvitationAsRequesterInput, time.Time) error
	RevokeSessionForRequester(context.Context, RevokeSessionAsRequesterInput, time.Time) error
	ReplaceInvitation(context.Context, ReplaceInvitationInput, Invitation, time.Time) error
}

func invitationAdministrationPersistence(repo Repository) (invitationAdministrationStore, error) {
	store, ok := repo.(invitationAdministrationStore)
	if !ok {
		return nil, ErrRecipientInvalid
	}
	return store, nil
}
